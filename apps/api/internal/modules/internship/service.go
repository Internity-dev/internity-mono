package internship

import (
	"context"
	"errors"
	"fmt"
	"time"

	"internity/internal/httpx"
	"internity/internal/modules/identity"
	"internity/internal/platform/cachex"
	"internity/internal/platform/postgres"
	"internity/internal/platform/storage"

	"github.com/redis/go-redis/v9"
)

var errForbidden = httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")
var errNotFoundAPI = httpx.NewError(httpx.ErrNotFound, "Not found")

// statusCountsCacheTTL — see vacancy.Service's identical constant; same
// dashboard-staleness tradeoff.
const statusCountsCacheTTL = 60 * time.Second

// CompanyScopeResolver — same narrow shape as vacancy.Service depends on;
// both are satisfied by the one companyScopeAdapter wired in cmd/api/main.go.
type CompanyScopeResolver interface {
	ResolveCompanyScope(ctx context.Context, companyID int64) (schoolID, departmentID int64, err error)
}

type Service struct {
	repo      *Repository
	companies CompanyScopeResolver
	storage   *storage.Client
	rdb       *redis.Client
}

func NewService(repo *Repository, companies CompanyScopeResolver, storageClient *storage.Client, rdb *redis.Client) *Service {
	return &Service{repo: repo, companies: companies, storage: storageClient, rdb: rdb}
}

// ScheduleForAcceptedAppliance is the cross-module entrypoint vacancy.Service
// calls on Accept.
func (s *Service) ScheduleForAcceptedAppliance(ctx context.Context, applianceID int64, userID string, companyID int64) error {
	err := s.repo.Create(ctx, &InternDate{
		UserID: userID, CompanyID: companyID, ApplianceID: applianceID,
		Status: StatusScheduled, Version: 1,
	})
	if err == nil {
		return nil
	}
	// `intern_dates` has UNIQUE (user_id, company_id) — a student can only
	// hold one placement per company. The generic vacancy.Service.Accept
	// caller has no context to phrase this well, so it's translated here,
	// against the error it's actually caused by, before returning.
	if apiErr := postgres.TranslateError(err); apiErr != nil {
		if apiErr.Code == httpx.ErrConflict {
			return httpx.NewError(httpx.ErrConflict, "This student already has a placement at this company")
		}
		return apiErr
	}
	return err
}

// --- InternDate ---

func (s *Service) ListMyInternDates(ctx context.Context, actor *identity.User) ([]InternDate, error) {
	if actor.Role != identity.RoleStudent {
		return nil, errForbidden
	}
	return s.repo.ListForUser(ctx, actor.ID)
}

func (s *Service) GetInternDate(ctx context.Context, actor *identity.User, id int64) (*InternDate, error) {
	row, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanAccessPlacement(ctx, actor, row); err != nil {
		return nil, err
	}
	return row, nil
}

type SetDatesInput struct {
	StartDate       time.Time
	EndDate         time.Time
	ExtendedUntil   *time.Time
	ExpectedVersion int
}

func (s *Service) SetDates(ctx context.Context, actor *identity.User, id int64, in SetDatesInput) (*InternDate, error) {
	row, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManagePlacement(ctx, actor, row); err != nil {
		return nil, err
	}
	if row.Status == StatusCompleted {
		return nil, httpx.NewError(httpx.ErrConflict, "This placement has already been marked completed")
	}
	if !in.StartDate.Before(in.EndDate) {
		return nil, httpx.NewError(httpx.ErrValidation, "Start date must be before end date",
			httpx.ErrorDetail{Field: "end_date", Issue: "must be after start_date"})
	}

	row.StartDate = &in.StartDate
	row.EndDate = &in.EndDate
	row.ExtendedUntil = in.ExtendedUntil

	if err := s.repo.UpdateWithVersion(ctx, row, in.ExpectedVersion); err != nil {
		if errors.Is(err, ErrVersionConflict) {
			return nil, httpx.NewError(httpx.ErrConflict, "This record was changed by someone else — please reload and try again")
		}
		if apiErr := postgres.TranslateError(err); apiErr != nil {
			if apiErr.Code == httpx.ErrConflict {
				return nil, httpx.NewError(httpx.ErrConflict, "These dates overlap with another one of this student's placements")
			}
			return nil, apiErr
		}
		return nil, err
	}
	return row, nil
}

func (s *Service) MarkCompleted(ctx context.Context, actor *identity.User, id int64) (*InternDate, error) {
	row, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManagePlacement(ctx, actor, row); err != nil {
		return nil, err
	}
	row.Status = StatusCompleted
	if err := s.repo.UpdateWithVersion(ctx, row, row.Version); err != nil {
		if errors.Is(err, ErrVersionConflict) {
			return nil, httpx.NewError(httpx.ErrConflict, "This record was changed by someone else — please reload and try again")
		}
		return nil, err
	}
	return row, nil
}

// --- Presence statuses ---

func (s *Service) ListPresenceStatuses(ctx context.Context, actor *identity.User, schoolID int64, params httpx.ListParams) ([]PresenceStatus, int64, error) {
	if err := s.assertCanViewSchool(actor, schoolID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListPresenceStatuses(ctx, schoolID, params)
}

func (s *Service) CreatePresenceStatus(ctx context.Context, actor *identity.User, row *PresenceStatus) error {
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return err
	}
	row.IsActive = true
	return translateWriteErr(s.repo.CreatePresenceStatus(ctx, row))
}

func (s *Service) UpdatePresenceStatus(ctx context.Context, actor *identity.User, id int64, patch PresenceStatusPatch) (*PresenceStatus, error) {
	row, err := s.repo.GetPresenceStatus(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return nil, err
	}
	patch.applyTo(row)
	if err := s.repo.UpdatePresenceStatus(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) DeletePresenceStatus(ctx context.Context, actor *identity.User, id int64) error {
	row, err := s.repo.GetPresenceStatus(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	if err := s.assertCanManageSchool(actor, row.SchoolID); err != nil {
		return err
	}
	return translateWriteErr(s.repo.DeletePresenceStatus(ctx, id))
}

// --- Presence: check-in / check-out / excuse ---

type CheckInInput struct {
	CompanyID int64
	Photo     []byte
	Filename  string
	Lat, Lng  *float64
}

// CheckIn always attempts to CREATE today's presence row — relying on the
// (user_id, company_id, date) unique constraint to reject a second check-in
// for the same day as a 409, rather than a separate check-then-insert query
// (which would race under concurrent requests).
func (s *Service) CheckIn(ctx context.Context, actor *identity.User, in CheckInInput) (*Presence, error) {
	if actor.Role != identity.RoleStudent {
		return nil, errForbidden
	}
	placement, err := s.repo.GetByUserCompany(ctx, actor.ID, in.CompanyID)
	if err != nil {
		return nil, translateGetErr(err)
	}
	today := truncateToDate(time.Now())
	if !placement.IsActiveOn(today) {
		return nil, httpx.NewError(httpx.ErrConflict, "Your internship is not currently active at this company")
	}

	schoolID, _, err := s.companies.ResolveCompanyScope(ctx, in.CompanyID)
	if err != nil {
		return nil, err
	}
	present, err := s.repo.FindPresenceStatusByKind(ctx, schoolID, KindPresent)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NewError(httpx.ErrConflict, "This school has not configured a 'present' attendance status yet")
	}
	if err != nil {
		return nil, err
	}

	var attachmentKey *string
	if len(in.Photo) > 0 {
		result, err := s.storage.Upload(ctx, storage.UploadInput{
			Bucket: storage.BucketAttachments, KeyPrefix: presenceKeyPrefix(today),
			OriginalFilename: in.Filename, Data: in.Photo,
			AllowedKinds: []string{"image"}, MaxBytes: storage.MaxImageBytes,
		})
		if err != nil {
			return nil, httpx.NewError(httpx.ErrValidation, err.Error(), httpx.ErrorDetail{Field: "photo", Issue: err.Error()})
		}
		attachmentKey = &result.Key
	}

	now := time.Now()
	row := &Presence{
		UserID: actor.ID, CompanyID: in.CompanyID, PresenceStatusID: present.ID, Date: today,
		CheckInAt: &now, CheckInLat: in.Lat, CheckInLng: in.Lng, AttachmentKey: attachmentKey,
	}
	if err := s.repo.CreatePresence(ctx, row); err != nil {
		if apiErr := postgres.TranslateError(err); apiErr != nil {
			if apiErr.Code == httpx.ErrConflict {
				return nil, httpx.NewError(httpx.ErrConflict, "You've already reported attendance for today")
			}
			return nil, apiErr
		}
		return nil, err
	}
	return row, nil
}

func (s *Service) CheckOut(ctx context.Context, actor *identity.User, companyID int64) (*Presence, error) {
	if actor.Role != identity.RoleStudent {
		return nil, errForbidden
	}
	today := truncateToDate(time.Now())
	row, err := s.repo.FindPresence(ctx, actor.ID, companyID, today)
	if errors.Is(err, ErrNotFound) || (err == nil && row.CheckInAt == nil) {
		return nil, httpx.NewError(httpx.ErrConflict, "Check in before checking out")
	}
	if err != nil {
		return nil, err
	}
	if row.CheckOutAt != nil {
		return nil, httpx.NewError(httpx.ErrConflict, "You've already checked out today")
	}
	now := time.Now()
	row.CheckOutAt = &now
	if err := s.repo.UpdatePresence(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

type ExcuseInput struct {
	CompanyID   int64
	Date        time.Time
	Kind        PresenceStatusKind // must be permitted or sick
	Description string
	Attachment  []byte
	Filename    string
}

func (s *Service) FileExcuse(ctx context.Context, actor *identity.User, in ExcuseInput) (*Presence, error) {
	if actor.Role != identity.RoleStudent {
		return nil, errForbidden
	}
	if in.Kind != KindPermitted && in.Kind != KindSick {
		return nil, httpx.NewError(httpx.ErrValidation, "Excuse kind must be 'permitted' or 'sick'",
			httpx.ErrorDetail{Field: "kind", Issue: "must be permitted or sick"})
	}
	placement, err := s.repo.GetByUserCompany(ctx, actor.ID, in.CompanyID)
	if err != nil {
		return nil, translateGetErr(err)
	}
	date := truncateToDate(in.Date)
	if !placement.IsActiveOn(date) {
		return nil, httpx.NewError(httpx.ErrConflict, "That date is outside your internship period")
	}

	schoolID, _, err := s.companies.ResolveCompanyScope(ctx, in.CompanyID)
	if err != nil {
		return nil, err
	}
	status, err := s.repo.FindPresenceStatusByKind(ctx, schoolID, in.Kind)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NewError(httpx.ErrConflict, fmt.Sprintf("This school has not configured a %q attendance status yet", in.Kind))
	}
	if err != nil {
		return nil, err
	}

	var attachmentKey *string
	if len(in.Attachment) > 0 {
		result, err := s.storage.Upload(ctx, storage.UploadInput{
			Bucket: storage.BucketAttachments, KeyPrefix: presenceKeyPrefix(date),
			OriginalFilename: in.Filename, Data: in.Attachment,
			AllowedKinds: []string{"image", "pdf"}, MaxBytes: storage.MaxImageBytes,
		})
		if err != nil {
			return nil, httpx.NewError(httpx.ErrValidation, err.Error(), httpx.ErrorDetail{Field: "attachment", Issue: err.Error()})
		}
		attachmentKey = &result.Key
	}

	row := &Presence{
		UserID: actor.ID, CompanyID: in.CompanyID, PresenceStatusID: status.ID, Date: date,
		Description: &in.Description, AttachmentKey: attachmentKey,
	}
	if err := s.repo.CreatePresence(ctx, row); err != nil {
		if apiErr := postgres.TranslateError(err); apiErr != nil {
			if apiErr.Code == httpx.ErrConflict {
				return nil, httpx.NewError(httpx.ErrConflict, "You already have an attendance record for that date")
			}
			return nil, apiErr
		}
		return nil, err
	}
	return row, nil
}

func (s *Service) ListMyPresences(ctx context.Context, actor *identity.User, companyID int64, params httpx.ListParams) ([]Presence, int64, error) {
	if actor.Role != identity.RoleStudent {
		return nil, 0, errForbidden
	}
	return s.repo.ListPresencesForUser(ctx, actor.ID, companyID, params)
}

func (s *Service) ApprovePresence(ctx context.Context, actor *identity.User, id int64) (*Presence, error) {
	row, err := s.repo.GetPresence(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManageCompany(ctx, actor, row.CompanyID); err != nil {
		return nil, err
	}
	row.IsApproved = true
	if err := s.repo.UpdatePresence(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) BulkApprovePresences(ctx context.Context, actor *identity.User, companyID int64, ids []int64) (int64, error) {
	if err := s.assertCanManageCompany(ctx, actor, companyID); err != nil {
		return 0, err
	}
	return s.repo.BulkApprovePresences(ctx, companyID, ids)
}

func (s *Service) ListPresencesForApproval(ctx context.Context, actor *identity.User, companyID int64, params httpx.ListParams) ([]Presence, int64, error) {
	if err := s.assertCanManageCompany(ctx, actor, companyID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListPresencesForApproval(ctx, companyID, params)
}

// --- Attendance summary read model ---

func (s *Service) AttendanceSummary(ctx context.Context, actor *identity.User, internDateID int64, month time.Time) ([]AttendanceDay, error) {
	placement, err := s.repo.Get(ctx, internDateID)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanAccessPlacement(ctx, actor, placement); err != nil {
		return nil, err
	}

	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	if placement.StartDate == nil || placement.EffectiveEndDate() == nil {
		return nil, nil // not yet scheduled — nothing to summarize
	}
	rangeStart := *placement.StartDate
	rangeEnd := *placement.EffectiveEndDate()

	presences, err := s.repo.ListPresencesInRange(ctx, placement.UserID, placement.CompanyID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}
	byDate := make(map[string]Presence, len(presences))
	for _, p := range presences {
		byDate[p.Date.Format("2006-01-02")] = p
	}

	today := truncateToDate(time.Now())
	var days []AttendanceDay
	for d := monthStart; !d.After(monthEnd); d = d.AddDate(0, 0, 1) {
		if d.Before(rangeStart) || d.After(rangeEnd) {
			days = append(days, AttendanceDay{Date: d, Status: DayOutOfRange})
			continue
		}
		if p, ok := byDate[d.Format("2006-01-02")]; ok {
			pCopy := p
			days = append(days, AttendanceDay{Date: d, Status: DayReported, Presence: &pCopy})
			continue
		}
		if d.After(today) {
			days = append(days, AttendanceDay{Date: d, Status: DayUpcoming})
		} else {
			days = append(days, AttendanceDay{Date: d, Status: DayMissing})
		}
	}
	return days, nil
}

// --- Journal ---

type JournalInput struct {
	CompanyID   int64
	Date        time.Time
	WorkType    string
	Description string
}

func (s *Service) UpsertJournal(ctx context.Context, actor *identity.User, in JournalInput) (*Journal, error) {
	if actor.Role != identity.RoleStudent {
		return nil, errForbidden
	}
	date := truncateToDate(in.Date)

	presence, err := s.repo.FindPresence(ctx, actor.ID, in.CompanyID, date)
	if errors.Is(err, ErrNotFound) || (err == nil && presence.CheckInAt == nil) {
		return nil, httpx.NewError(httpx.ErrConflict, "You can only journal a day you checked in")
	}
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.FindJournal(ctx, actor.ID, in.CompanyID, date)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		if existing.IsApproved {
			return nil, httpx.NewError(httpx.ErrConflict, "This journal entry has already been approved and can no longer be edited")
		}
		existing.WorkType = &in.WorkType
		existing.Description = &in.Description
		if err := s.repo.UpdateJournal(ctx, existing); err != nil {
			return nil, translateWriteErr(err)
		}
		return existing, nil
	}

	row := &Journal{UserID: actor.ID, CompanyID: in.CompanyID, Date: date, WorkType: &in.WorkType, Description: &in.Description}
	if err := s.repo.CreateJournal(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) ListMyJournals(ctx context.Context, actor *identity.User, companyID int64, params httpx.ListParams) ([]Journal, int64, error) {
	if actor.Role != identity.RoleStudent {
		return nil, 0, errForbidden
	}
	return s.repo.ListJournalsForUser(ctx, actor.ID, companyID, params)
}

func (s *Service) ListJournalsForApproval(ctx context.Context, actor *identity.User, companyID int64, params httpx.ListParams) ([]Journal, int64, error) {
	if err := s.assertCanManageCompany(ctx, actor, companyID); err != nil {
		return nil, 0, err
	}
	return s.repo.ListJournalsForApproval(ctx, companyID, params)
}

func (s *Service) ApproveJournal(ctx context.Context, actor *identity.User, id int64) (*Journal, error) {
	row, err := s.repo.GetJournal(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if err := s.assertCanManageCompany(ctx, actor, row.CompanyID); err != nil {
		return nil, err
	}
	if !row.Filled() {
		return nil, httpx.NewError(httpx.ErrConflict, "This journal entry isn't filled in yet")
	}
	row.IsApproved = true
	if err := s.repo.UpdateJournal(ctx, row); err != nil {
		return nil, translateWriteErr(err)
	}
	return row, nil
}

func (s *Service) BulkApproveJournals(ctx context.Context, actor *identity.User, companyID int64, ids []int64) (int64, error) {
	if err := s.assertCanManageCompany(ctx, actor, companyID); err != nil {
		return 0, err
	}
	return s.repo.BulkApproveJournals(ctx, companyID, ids)
}

// --- scope helpers ---

func (s *Service) assertCanAccessPlacement(ctx context.Context, actor *identity.User, placement *InternDate) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.Role == identity.RoleStudent {
		if placement.UserID != actor.ID {
			return errForbidden
		}
		return nil
	}
	return s.assertCanManageCompany(ctx, actor, placement.CompanyID)
}

func (s *Service) assertCanManagePlacement(ctx context.Context, actor *identity.User, placement *InternDate) error {
	if actor.Role == identity.RoleStudent {
		if placement.UserID != actor.ID {
			return errForbidden
		}
		return nil
	}
	return s.assertCanManageCompany(ctx, actor, placement.CompanyID)
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

func (s *Service) assertCanViewSchool(actor *identity.User, schoolID int64) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.SchoolID == nil || *actor.SchoolID != schoolID {
		return errForbidden
	}
	return nil
}

func (s *Service) assertCanManageSchool(actor *identity.User, schoolID int64) error {
	if actor.Role == identity.RoleAdmin {
		return nil
	}
	if actor.Role == identity.RoleCoordinator && actor.SchoolID != nil && *actor.SchoolID == schoolID {
		return nil
	}
	return errForbidden
}

func presenceKeyPrefix(date time.Time) string {
	return fmt.Sprintf("presence/%04d/%02d/%02d", date.Year(), date.Month(), date.Day())
}

// PresenceStatusCounts backs the admin overview dashboard's attendance
// breakdown chart, same admin-only scoping rationale as the vacancy
// module's ApplianceStatusCounts. Defaults to the current calendar month —
// presences is the fastest-growing table in the schema, so an unbounded
// all-time query here would only get slower every school-year; a caller may
// still pass an explicit from/to to look at a different period.
func (s *Service) PresenceStatusCounts(ctx context.Context, actor *identity.User, from, to *time.Time) (map[PresenceStatusKind]int64, error) {
	if actor.Role != identity.RoleAdmin {
		return nil, errForbidden
	}
	start, end := resolvePresenceCountsRange(from, to)
	key := fmt.Sprintf("cache:presence-status-counts:%s:%s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	return cachex.GetOrSet(ctx, s.rdb, key, statusCountsCacheTTL, func() (map[PresenceStatusKind]int64, error) {
		return s.repo.CountPresencesByKind(ctx, start, end)
	})
}

func resolvePresenceCountsRange(from, to *time.Time) (time.Time, time.Time) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := now
	if from != nil {
		start = *from
	}
	if to != nil {
		end = *to
	}
	return start, end
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
