package orgs

import (
	"context"
	"errors"

	"internity/internal/httpx"
	"internity/internal/modules/identity"
	"internity/internal/platform/postgres"
)

var errForbidden = httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")
var errNotFoundAPI = httpx.NewError(httpx.ErrNotFound, "Not found")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// --- Schools: admin-only. Onboarding a school onto the platform is a
// platform-operator action, not something school staff do to themselves. ---

func (s *Service) ListSchools(ctx context.Context, actor *identity.User, params httpx.ListParams) ([]School, int64, error) {
	if actor.Role != identity.RoleAdmin {
		return nil, 0, errForbidden
	}
	return s.repo.ListSchools(ctx, params)
}

func (s *Service) GetSchool(ctx context.Context, actor *identity.User, id int64) (*School, error) {
	school, err := s.repo.GetSchool(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if actor.Role != identity.RoleAdmin && !actorInSchool(actor, id) {
		return nil, errForbidden
	}
	return school, nil
}

func (s *Service) CreateSchool(ctx context.Context, actor *identity.User, in *School) error {
	if actor.Role != identity.RoleAdmin {
		return errForbidden
	}
	in.IsActive = true
	return translateWriteErr(s.repo.CreateSchool(ctx, in))
}

func (s *Service) UpdateSchool(ctx context.Context, actor *identity.User, id int64, patch SchoolPatch) (*School, error) {
	if actor.Role != identity.RoleAdmin {
		return nil, errForbidden
	}
	existing, err := s.repo.GetSchool(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	patch.applyTo(existing)
	if err := s.repo.UpdateSchool(ctx, existing); err != nil {
		return nil, translateWriteErr(err)
	}
	return existing, nil
}

func (s *Service) DeleteSchool(ctx context.Context, actor *identity.User, id int64) error {
	if actor.Role != identity.RoleAdmin {
		return errForbidden
	}
	return translateWriteErr(s.repo.DeleteSchool(ctx, id))
}

// --- Departments: admin (any school) or coordinator (own school only, read+write).
// Mentor/student get read-only access scoped to their own school. ---

func (s *Service) ListDepartments(ctx context.Context, actor *identity.User, schoolID *int64, params httpx.ListParams) ([]Department, int64, error) {
	effective, err := scopedSchoolFilter(actor, schoolID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListDepartments(ctx, effective, params)
}

func (s *Service) GetDepartment(ctx context.Context, actor *identity.User, id int64) (*Department, error) {
	dept, err := s.repo.GetDepartment(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if actor.Role != identity.RoleAdmin && !actorInSchool(actor, dept.SchoolID) {
		return nil, errForbidden
	}
	return dept, nil
}

func (s *Service) CreateDepartment(ctx context.Context, actor *identity.User, in *Department) error {
	if !canManageSchool(actor, in.SchoolID) {
		return errForbidden
	}
	in.IsActive = true
	return translateWriteErr(s.repo.CreateDepartment(ctx, in))
}

func (s *Service) UpdateDepartment(ctx context.Context, actor *identity.User, id int64, patch DepartmentPatch) (*Department, error) {
	existing, err := s.repo.GetDepartment(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if !canManageSchool(actor, existing.SchoolID) {
		return nil, errForbidden
	}
	patch.applyTo(existing)
	if err := s.repo.UpdateDepartment(ctx, existing); err != nil {
		return nil, translateWriteErr(err)
	}
	return existing, nil
}

func (s *Service) DeleteDepartment(ctx context.Context, actor *identity.User, id int64) error {
	existing, err := s.repo.GetDepartment(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	if !canManageSchool(actor, existing.SchoolID) {
		return errForbidden
	}
	return translateWriteErr(s.repo.DeleteDepartment(ctx, id))
}

// --- Courses: same scope rule as departments, but resolved via the
// department's school (a course has no school_id column of its own). ---

func (s *Service) ListCourses(ctx context.Context, actor *identity.User, departmentID *int64, params httpx.ListParams) ([]Course, int64, error) {
	if departmentID != nil {
		dept, err := s.repo.GetDepartment(ctx, *departmentID)
		if err != nil {
			return nil, 0, translateGetErr(err)
		}
		if actor.Role != identity.RoleAdmin && !actorInSchool(actor, dept.SchoolID) {
			return nil, 0, errForbidden
		}
	} else if actor.Role != identity.RoleAdmin {
		return nil, 0, errForbidden // must scope to a department you belong to
	}
	return s.repo.ListCourses(ctx, departmentID, params)
}

func (s *Service) GetCourse(ctx context.Context, actor *identity.User, id int64) (*Course, error) {
	course, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	dept, err := s.repo.GetDepartment(ctx, course.DepartmentID)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if actor.Role != identity.RoleAdmin && !actorInSchool(actor, dept.SchoolID) {
		return nil, errForbidden
	}
	return course, nil
}

func (s *Service) CreateCourse(ctx context.Context, actor *identity.User, in *Course) error {
	dept, err := s.repo.GetDepartment(ctx, in.DepartmentID)
	if err != nil {
		return translateGetErr(err)
	}
	if !canManageSchool(actor, dept.SchoolID) {
		return errForbidden
	}
	in.IsActive = true
	return translateWriteErr(s.repo.CreateCourse(ctx, in))
}

func (s *Service) UpdateCourse(ctx context.Context, actor *identity.User, id int64, patch CoursePatch) (*Course, error) {
	existing, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	dept, err := s.repo.GetDepartment(ctx, existing.DepartmentID)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if !canManageSchool(actor, dept.SchoolID) {
		return nil, errForbidden
	}
	patch.applyTo(existing)
	if err := s.repo.UpdateCourse(ctx, existing); err != nil {
		return nil, translateWriteErr(err)
	}
	return existing, nil
}

func (s *Service) DeleteCourse(ctx context.Context, actor *identity.User, id int64) error {
	existing, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	dept, err := s.repo.GetDepartment(ctx, existing.DepartmentID)
	if err != nil {
		return translateGetErr(err)
	}
	if !canManageSchool(actor, dept.SchoolID) {
		return errForbidden
	}
	return translateWriteErr(s.repo.DeleteCourse(ctx, id))
}

// --- Companies: admin/coordinator manage within their school (via
// department); mentor gets read-only access to their own company only. ---

func (s *Service) ListCompanies(ctx context.Context, actor *identity.User, departmentID *int64, params httpx.ListParams) ([]Company, int64, error) {
	if departmentID != nil {
		dept, err := s.repo.GetDepartment(ctx, *departmentID)
		if err != nil {
			return nil, 0, translateGetErr(err)
		}
		if actor.Role != identity.RoleAdmin && actor.Role != identity.RoleMentor && !actorInSchool(actor, dept.SchoolID) {
			return nil, 0, errForbidden
		}
	} else if actor.Role != identity.RoleAdmin {
		return nil, 0, errForbidden
	}
	return s.repo.ListCompanies(ctx, departmentID, params)
}

func (s *Service) GetCompany(ctx context.Context, actor *identity.User, id int64) (*Company, error) {
	company, err := s.repo.GetCompany(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if actor.Role == identity.RoleMentor {
		if actor.CompanyID == nil || *actor.CompanyID != id {
			return nil, errForbidden
		}
		return company, nil
	}
	if actor.Role != identity.RoleAdmin {
		dept, err := s.repo.GetDepartment(ctx, company.DepartmentID)
		if err != nil {
			return nil, translateGetErr(err)
		}
		if !actorInSchool(actor, dept.SchoolID) {
			return nil, errForbidden
		}
	}
	return company, nil
}

func (s *Service) CreateCompany(ctx context.Context, actor *identity.User, in *Company) error {
	dept, err := s.repo.GetDepartment(ctx, in.DepartmentID)
	if err != nil {
		return translateGetErr(err)
	}
	if !canManageSchool(actor, dept.SchoolID) {
		return errForbidden
	}
	in.IsActive = true
	return translateWriteErr(s.repo.CreateCompany(ctx, in))
}

func (s *Service) UpdateCompany(ctx context.Context, actor *identity.User, id int64, patch CompanyPatch) (*Company, error) {
	existing, err := s.repo.GetCompany(ctx, id)
	if err != nil {
		return nil, translateGetErr(err)
	}
	dept, err := s.repo.GetDepartment(ctx, existing.DepartmentID)
	if err != nil {
		return nil, translateGetErr(err)
	}
	if !canManageSchool(actor, dept.SchoolID) {
		return nil, errForbidden
	}
	patch.applyTo(existing)
	if err := s.repo.UpdateCompany(ctx, existing); err != nil {
		return nil, translateWriteErr(err)
	}
	return existing, nil
}

func (s *Service) DeleteCompany(ctx context.Context, actor *identity.User, id int64) error {
	existing, err := s.repo.GetCompany(ctx, id)
	if err != nil {
		return translateGetErr(err)
	}
	dept, err := s.repo.GetDepartment(ctx, existing.DepartmentID)
	if err != nil {
		return translateGetErr(err)
	}
	if !canManageSchool(actor, dept.SchoolID) {
		return errForbidden
	}
	return translateWriteErr(s.repo.DeleteCompany(ctx, id))
}

// --- scope helpers ---

func actorInSchool(actor *identity.User, schoolID int64) bool {
	return actor.SchoolID != nil && *actor.SchoolID == schoolID
}

// canManageSchool: admin manages any school; coordinator manages only their
// own. Mentor/student never create/update/delete org structure.
func canManageSchool(actor *identity.User, schoolID int64) bool {
	if actor.Role == identity.RoleAdmin {
		return true
	}
	return actor.Role == identity.RoleCoordinator && actorInSchool(actor, schoolID)
}

// scopedSchoolFilter reconciles the caller-requested schoolID query filter
// with what the actor is actually allowed to see: admin can filter by any
// (or no) school; everyone else is pinned to their own school_id regardless
// of what they passed in.
func scopedSchoolFilter(actor *identity.User, requested *int64) (*int64, error) {
	if actor.Role == identity.RoleAdmin {
		return requested, nil
	}
	if actor.SchoolID == nil {
		return nil, errForbidden
	}
	return actor.SchoolID, nil
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
