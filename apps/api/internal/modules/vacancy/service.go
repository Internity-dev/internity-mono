package vacancy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"internity/internal/httpx"
	"internity/internal/modules/identity"
	"internity/internal/platform/cachex"
	"internity/internal/platform/postgres"

	"github.com/redis/go-redis/v9"
)

// statusCountsCacheTTL bounds how stale a dashboard status-breakdown chart
// can be — short enough to feel live, long enough that a busy admin
// dashboard doesn't re-scan the table on every page load.
const statusCountsCacheTTL = 60 * time.Second

var errForbidden = httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")
var errNotFoundAPI = httpx.NewError(httpx.ErrNotFound, "Not found")

// CompanyScopeResolver lets this module scope-check against the org
// hierarchy (which school/department a company belongs to) without ever
// importing the orgs module's repository directly — see the adapter wired
// in cmd/api/main.go around orgs.Repository.ResolveCompanyScope.
type CompanyScopeResolver interface {
	ResolveCompanyScope(ctx context.Context, companyID int64) (schoolID, departmentID int64, err error)
	ResolveDepartmentSchool(ctx context.Context, departmentID int64) (schoolID int64, err error)
}

// Notifier is the cross-module entrypoint into the notification module.
type Notifier interface {
	Send(ctx context.Context, userID, notifType, title, body string) error
}

// InternshipScheduler is the cross-module entrypoint into the internship
// module — called exactly once, on Accept.
type InternshipScheduler interface {
	ScheduleForAcceptedAppliance(ctx context.Context, applianceID int64, userID string, companyID int64) error
}

type Service struct {
	repo      *Repository
	companies CompanyScopeResolver
	notifier  Notifier
	scheduler InternshipScheduler
	rdb       *redis.Client
}

func NewService(repo *Repository, companies CompanyScopeResolver, notifier Notifier, scheduler InternshipScheduler, rdb *redis.Client) *Service {
	return &Service{repo: repo, companies: companies, notifier: notifier, scheduler: scheduler, rdb: rdb}
}

// --- Vacancies ---

func (s *Service) ListVacancies(ctx context.Context, actor *identity.User, companyID, departmentID *int64, params httpx.ListParams) ([]Vacancy, int64, error) {
	filter := VacancyFilter{CompanyID: companyID, DepartmentID: departmentID}

	switch actor.Role {
	case identity.RoleAdmin:
		// no restriction
	case identity.RoleMentor:
		filter.CompanyID = actor.CompanyID // mentors only ever see their own company's vacancies
		filter.DepartmentID = nil
	case identity.RoleCoordinator:
		if filter.CompanyID == nil && filter.DepartmentID == nil {
			return nil, 0, httpx.NewError(httpx.ErrValidation, "Provide a company_id or department_id filter")
		}
		if err := s.assertCoordinatorOwnsFilter(ctx, actor, filter); err != nil {
			return nil, 0, err
		}
	case identity.RoleStudent:
		if actor.DepartmentID == nil {
			return nil, 0, errForbidden
		}
		filter.DepartmentID = actor.DepartmentID
		if filter.CompanyID == nil {
			open := VacancyOpen
			filter.Status = &open // students browsing only ever see open vacancies by default
		}
	}

	return s.repo.ListVacancies(ctx, filter, params)
}

func (s *Service) GetVacancy(ctx context.Context, actor *identity.User, id int64) (*Vacancy, error) {
	v, err := s.repo.GetVacancy(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanViewCompany(ctx, actor, v.CompanyID); err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) CreateVacancy(ctx context.Context, actor *identity.User, in *Vacancy) error {
	if err := s.assertCanManageCompany(ctx, actor, in.CompanyID); err != nil {
		return err
	}
	if in.Slots < 1 {
		in.Slots = 1
	}
	in.Status = VacancyOpen
	return translateWriteErr(s.repo.CreateVacancy(ctx, in))
}

func (s *Service) UpdateVacancy(ctx context.Context, actor *identity.User, id int64, patch VacancyPatch) (*Vacancy, error) {
	existing, err := s.repo.GetVacancy(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManageCompany(ctx, actor, existing.CompanyID); err != nil {
		return nil, err
	}
	patch.applyTo(existing)
	if err := s.repo.UpdateVacancy(ctx, existing); err != nil {
		return nil, translateWriteErr(err)
	}
	return existing, nil
}

func (s *Service) DeleteVacancy(ctx context.Context, actor *identity.User, id int64) error {
	existing, err := s.repo.GetVacancy(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	if err := s.assertCanManageCompany(ctx, actor, existing.CompanyID); err != nil {
		return err
	}
	return translateWriteErr(s.repo.DeleteVacancy(ctx, id))
}

// --- Saved vacancies ---

func (s *Service) SaveVacancy(ctx context.Context, actor *identity.User, vacancyID int64) error {
	if actor.Role != identity.RoleStudent {
		return errForbidden
	}
	if _, err := s.repo.GetVacancy(ctx, vacancyID); err != nil {
		return translateGetErr(err)
	}
	return translateWriteErr(s.repo.SaveVacancy(ctx, actor.ID, vacancyID))
}

func (s *Service) UnsaveVacancy(ctx context.Context, actor *identity.User, vacancyID int64) error {
	if actor.Role != identity.RoleStudent {
		return errForbidden
	}
	return translateWriteErr(s.repo.UnsaveVacancy(ctx, actor.ID, vacancyID))
}

func (s *Service) ListSavedVacancies(ctx context.Context, actor *identity.User, params httpx.ListParams) ([]Vacancy, int64, error) {
	if actor.Role != identity.RoleStudent {
		return nil, 0, errForbidden
	}
	return s.repo.ListSavedVacancies(ctx, actor.ID, params)
}

// --- Appliances: the state machine ---

// Apply: student only, one department-matching vacancy at a time. The
// "one active application per vacancy" rule is enforced by the DB's partial
// unique index (uq_appliances_active_per_user_vacancy) — caught below as a
// 409 rather than pre-checked, which avoids a check-then-act race between
// two concurrent apply requests for the same vacancy.
func (s *Service) Apply(ctx context.Context, actor *identity.User, vacancyID int64, message *string) (*Appliance, error) {
	if actor.Role != identity.RoleStudent {
		return nil, errForbidden
	}
	if actor.DepartmentID == nil {
		return nil, errForbidden
	}

	v, err := s.repo.GetVacancy(ctx, vacancyID)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if v.Status != VacancyOpen {
		return nil, httpx.NewError(httpx.ErrConflict, "This vacancy is no longer open")
	}

	_, vacancyDept, err := s.companies.ResolveCompanyScope(ctx, v.CompanyID)
	if err != nil {
		return nil, err
	}
	if vacancyDept != *actor.DepartmentID {
		return nil, httpx.NewError(httpx.ErrForbidden, "This vacancy is not available for your department")
	}

	appliance := &Appliance{UserID: actor.ID, VacancyID: vacancyID, Status: StatusPending, Message: message}
	if err := s.repo.CreateAppliance(ctx, appliance); err != nil {
		if apiErr := postgres.TranslateError(err); apiErr != nil {
			if apiErr.Code == httpx.ErrConflict {
				return nil, httpx.NewError(httpx.ErrConflict, "You already have an active application for this vacancy")
			}
			return nil, apiErr
		}
		return nil, err
	}
	return appliance, nil
}

func (s *Service) Cancel(ctx context.Context, actor *identity.User, applianceID int64) (*Appliance, error) {
	a, err := s.repo.GetAppliance(ctx, applianceID)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if a.UserID != actor.ID {
		return nil, errForbidden
	}
	return s.transition(ctx, a, StatusCanceled, nil)
}

func (s *Service) Process(ctx context.Context, actor *identity.User, applianceID int64) (*Appliance, error) {
	a, v, err := s.getApplianceWithVacancy(ctx, applianceID)
	if err != nil {
		return nil, err
	}
	if err := s.assertCanManageCompany(ctx, actor, v.CompanyID); err != nil {
		return nil, err
	}
	return s.transition(ctx, a, StatusProcessed, v)
}

func (s *Service) Reject(ctx context.Context, actor *identity.User, applianceID int64) (*Appliance, error) {
	a, v, err := s.getApplianceWithVacancy(ctx, applianceID)
	if err != nil {
		return nil, err
	}
	if err := s.assertCanManageCompany(ctx, actor, v.CompanyID); err != nil {
		return nil, err
	}
	return s.transition(ctx, a, StatusRejected, v)
}

// Accept: the one transition with side effects — a slot-availability check
// (a real business rule, not just a state change) and handing off to the
// internship module to create the (still-unscheduled) placement record.
func (s *Service) Accept(ctx context.Context, actor *identity.User, applianceID int64) (*Appliance, error) {
	a, v, err := s.getApplianceWithVacancy(ctx, applianceID)
	if err != nil {
		return nil, err
	}
	if err := s.assertCanManageCompany(ctx, actor, v.CompanyID); err != nil {
		return nil, err
	}
	if err := a.CanTransitionTo(StatusAccepted); err != nil {
		return nil, httpx.NewError(httpx.ErrConflict, err.Error())
	}

	accepted, err := s.repo.CountAcceptedForVacancy(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	if accepted >= int64(v.Slots) {
		return nil, httpx.NewError(httpx.ErrConflict, "No slots remaining for this vacancy")
	}

	a.Status = StatusAccepted
	if err := s.repo.UpdateAppliance(ctx, a); err != nil {
		return nil, translateWriteErr(err)
	}

	if err := s.scheduler.ScheduleForAcceptedAppliance(ctx, a.ID, a.UserID, v.CompanyID); err != nil {
		return nil, err
	}
	_ = s.notifier.Send(ctx, a.UserID, "appliance_accepted", "Application accepted",
		"Your application has been accepted. Set your internship start/end date to continue.")
	return a, nil
}

func (s *Service) ListMyAppliances(ctx context.Context, actor *identity.User, params httpx.ListParams) ([]Appliance, int64, error) {
	return s.repo.ListAppliancesForUser(ctx, actor.ID, params)
}

func (s *Service) ListVacancyAppliances(ctx context.Context, actor *identity.User, vacancyID int64, params httpx.ListParams) ([]Appliance, int64, error) {
	v, err := s.repo.GetVacancy(ctx, vacancyID)
	if err != nil {
		return nil, 0, translateGetErr(err)
	}
	if err := s.assertCanManageCompany(ctx, actor, v.CompanyID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListAppliancesForVacancy(ctx, vacancyID, params)
}

// --- internal helpers ---

func (s *Service) transition(ctx context.Context, a *Appliance, target ApplianceStatus, v *Vacancy) (*Appliance, error) {
	if err := a.CanTransitionTo(target); err != nil {
		return nil, httpx.NewError(httpx.ErrConflict, err.Error())
	}
	a.Status = target
	if err := s.repo.UpdateAppliance(ctx, a); err != nil {
		return nil, translateWriteErr(err)
	}

	notifTitle := map[ApplianceStatus]string{
		StatusProcessed: "Application under review",
		StatusRejected:  "Application rejected",
		StatusCanceled:  "Application canceled",
	}[target]
	if notifTitle != "" {
		_ = s.notifier.Send(ctx, a.UserID, "appliance_"+string(target), notifTitle, notifTitle)
	}
	return a, nil
}

func (s *Service) getApplianceWithVacancy(ctx context.Context, applianceID int64) (*Appliance, *Vacancy, error) {
	a, err := s.repo.GetAppliance(ctx, applianceID)
	if err != nil {
		return nil, nil, translateGetErr(err)
	}
	v, err := s.repo.GetVacancy(ctx, a.VacancyID)
	if err != nil {
		return nil, nil, translateGetErr(err)
	}
	return a, v, nil
}

func (s *Service) assertCanViewCompany(ctx context.Context, actor *identity.User, companyID int64) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.Role == identity.RoleMentor {
		if actor.CompanyID == nil || *actor.CompanyID != companyID {
			return errForbidden
		}
		return nil
	}
	schoolID, departmentID, err := s.companies.ResolveCompanyScope(ctx, companyID)
	if err != nil {
		return err
	}
	if actor.Role == identity.RoleStudent {
		if actor.DepartmentID == nil || *actor.DepartmentID != departmentID {
			return errForbidden
		}
		return nil
	}
	if actor.SchoolID == nil || *actor.SchoolID != schoolID {
		return errForbidden
	}
	return nil
}

func (s *Service) assertCanManageCompany(ctx context.Context, actor *identity.User, companyID int64) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.Role == identity.RoleMentor {
		if actor.CompanyID == nil || *actor.CompanyID != companyID {
			return errForbidden
		}
		return nil
	}
	if actor.Role != identity.RoleCoordinator {
		return errForbidden
	}
	schoolID, _, err := s.companies.ResolveCompanyScope(ctx, companyID)
	if err != nil {
		return err
	}
	if actor.SchoolID == nil || *actor.SchoolID != schoolID {
		return errForbidden
	}
	return nil
}

// assertCoordinatorOwnsFilter checks a coordinator-supplied list filter
// against their own school. A company_id filter is checked precisely (via
// the company->department->school join). A department_id filter is resolved
// and compared directly too — nothing upstream of this call actually forces
// the department_id to have come from a school-scoped GetDepartment call
// first, so a bare "the coordinator has *a* school" check let a coordinator
// list any other school's vacancies just by passing its department_id.
func (s *Service) assertCoordinatorOwnsFilter(ctx context.Context, actor *identity.User, filter VacancyFilter) error {
	if filter.CompanyID != nil {
		return s.assertCanManageCompany(ctx, actor, *filter.CompanyID)
	}
	if actor.SchoolID == nil {
		return errForbidden
	}
	if filter.DepartmentID != nil {
		schoolID, err := s.companies.ResolveDepartmentSchool(ctx, *filter.DepartmentID)
		if err != nil {
			return err
		}
		if schoolID != *actor.SchoolID {
			return errForbidden
		}
	}
	return nil
}

// statusCountsScopeFor turns the actor's role into the narrowing the two
// queries below need: admin sees the whole platform, coordinator their own
// school, mentor their own company. Student has no meaningful scope here
// (their own appliance/vacancy footprint is a handful of rows, not a
// breakdown worth charting) and is rejected.
func statusCountsScopeFor(actor *identity.User) (StatusCountsScope, error) {
	switch actor.Role {
	case identity.RoleAdmin:
		return StatusCountsScope{}, nil
	case identity.RoleCoordinator:
		if actor.SchoolID == nil {
			return StatusCountsScope{}, errForbidden
		}
		return StatusCountsScope{SchoolID: actor.SchoolID}, nil
	case identity.RoleMentor:
		if actor.CompanyID == nil {
			return StatusCountsScope{}, errForbidden
		}
		return StatusCountsScope{CompanyID: actor.CompanyID}, nil
	default:
		return StatusCountsScope{}, errForbidden
	}
}

// statusCountsCacheKey keeps each scope's cached result separate — sharing
// one key across schools/companies would leak one coordinator's or mentor's
// numbers into another's dashboard.
func statusCountsCacheKey(base string, scope StatusCountsScope) string {
	switch {
	case scope.CompanyID != nil:
		return fmt.Sprintf("%s:company:%d", base, *scope.CompanyID)
	case scope.SchoolID != nil:
		return fmt.Sprintf("%s:school:%d", base, *scope.SchoolID)
	default:
		return base
	}
}

// ApplianceStatusCounts backs the overview dashboard's application
// status-breakdown chart, scoped per statusCountsScopeFor.
func (s *Service) ApplianceStatusCounts(ctx context.Context, actor *identity.User) (map[ApplianceStatus]int64, error) {
	scope, err := statusCountsScopeFor(actor)
	if err != nil {
		return nil, err
	}
	key := statusCountsCacheKey("cache:appliance-status-counts", scope)
	return cachex.GetOrSet(ctx, s.rdb, key, statusCountsCacheTTL, func() (map[ApplianceStatus]int64, error) {
		return s.repo.CountAppliancesByStatus(ctx, scope)
	})
}

// VacancyStatusCounts backs the same overview dashboard's vacancy
// status-breakdown chart, same scoping as ApplianceStatusCounts above.
func (s *Service) VacancyStatusCounts(ctx context.Context, actor *identity.User) (map[VacancyStatus]int64, error) {
	scope, err := statusCountsScopeFor(actor)
	if err != nil {
		return nil, err
	}
	key := statusCountsCacheKey("cache:vacancy-status-counts", scope)
	return cachex.GetOrSet(ctx, s.rdb, key, statusCountsCacheTTL, func() (map[VacancyStatus]int64, error) {
		return s.repo.CountVacanciesByStatus(ctx, scope)
	})
}

func translateGetErr(err error) error {
	if errors.Is(err, ErrNotFound) {
		return errNotFoundAPI
	}
	return err
}

func translateWriteErr(err error) error {
	if err == nil {
		return nil
	}
	if apiErr := postgres.TranslateError(err); apiErr != nil {
		return apiErr
	}
	return err
}
