package vacancy

import (
	"context"
	"testing"

	"internity/internal/httpx"
	"internity/internal/modules/identity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

// fakeCompanyScope resolves every company/department to the same
// school/department pair — enough for the tests below, which only ever
// check one at a time. Mirrors identity's fakeCompanyScope.
type fakeCompanyScope struct {
	schoolID, departmentID int64
	err                    error
}

func (f fakeCompanyScope) ResolveCompanyScope(_ context.Context, _ int64) (int64, int64, error) {
	return f.schoolID, f.departmentID, f.err
}

func (f fakeCompanyScope) ResolveDepartmentSchool(_ context.Context, _ int64) (int64, error) {
	return f.schoolID, f.err
}

func newTestService(companies CompanyScopeResolver) *Service {
	// repo is deliberately nil: every test below only exercises a role gate
	// or a companies-only helper that returns before s.repo is ever touched.
	// A test relying on repo behavior would panic here, which is exactly
	// the point — see identity/service_test.go's fakeListUsersRepo comment
	// for the same "fail loudly, don't silently no-op" reasoning.
	return NewService(nil, companies, nil, nil, nil)
}

func requireAPIErr(t *testing.T, err error, code httpx.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, code, apiErr.Code)
}

func TestSaveVacancy_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	for _, role := range []identity.Role{identity.RoleAdmin, identity.RoleCoordinator, identity.RoleMentor} {
		err := svc.SaveVacancy(context.Background(), &identity.User{Role: role}, 1)
		requireAPIErr(t, err, httpx.ErrForbidden)
	}
}

func TestUnsaveVacancy_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	err := svc.UnsaveVacancy(context.Background(), &identity.User{Role: identity.RoleMentor}, 1)
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestListSavedVacancies_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	_, _, err := svc.ListSavedVacancies(context.Background(), &identity.User{Role: identity.RoleCoordinator}, httpx.ListParams{})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestApply_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	admin := &identity.User{Role: identity.RoleAdmin}
	_, err := svc.Apply(context.Background(), admin, 1, nil)
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestApply_StudentWithoutDepartmentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	student := &identity.User{Role: identity.RoleStudent, DepartmentID: nil}
	_, err := svc.Apply(context.Background(), student, 1, nil)
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestAssertCanManageCompany(t *testing.T) {
	t.Run("admin manages any company", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{})
		admin := &identity.User{Role: identity.RoleAdmin}
		assert.NoError(t, svc.assertCanManageCompany(context.Background(), admin, 999))
	})

	t.Run("mentor manages only their own company", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{})
		mentor := &identity.User{Role: identity.RoleMentor, CompanyID: int64Ptr(5)}
		assert.NoError(t, svc.assertCanManageCompany(context.Background(), mentor, 5))
		requireAPIErr(t, svc.assertCanManageCompany(context.Background(), mentor, 6), httpx.ErrForbidden)
	})

	t.Run("coordinator manages a company only within their own school", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{schoolID: 7})
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(7)}
		assert.NoError(t, svc.assertCanManageCompany(context.Background(), coordinator, 1))

		svcOtherSchool := newTestService(fakeCompanyScope{schoolID: 999})
		requireAPIErr(t, svcOtherSchool.assertCanManageCompany(context.Background(), coordinator, 1), httpx.ErrForbidden)
	})

	t.Run("student can never manage a company", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{})
		student := &identity.User{Role: identity.RoleStudent, DepartmentID: int64Ptr(1)}
		requireAPIErr(t, svc.assertCanManageCompany(context.Background(), student, 1), httpx.ErrForbidden)
	})
}

func TestAssertCanViewCompany(t *testing.T) {
	t.Run("student can view a company only within their own department", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{departmentID: 3})
		student := &identity.User{Role: identity.RoleStudent, DepartmentID: int64Ptr(3)}
		assert.NoError(t, svc.assertCanViewCompany(context.Background(), student, 1))

		otherStudent := &identity.User{Role: identity.RoleStudent, DepartmentID: int64Ptr(4)}
		requireAPIErr(t, svc.assertCanViewCompany(context.Background(), otherStudent, 1), httpx.ErrForbidden)
	})

	t.Run("mentor can view only their own company", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{})
		mentor := &identity.User{Role: identity.RoleMentor, CompanyID: int64Ptr(5)}
		assert.NoError(t, svc.assertCanViewCompany(context.Background(), mentor, 5))
		requireAPIErr(t, svc.assertCanViewCompany(context.Background(), mentor, 6), httpx.ErrForbidden)
	})
}

func TestAssertCoordinatorOwnsFilter(t *testing.T) {
	t.Run("filter by company_id delegates to assertCanManageCompany", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{schoolID: 7})
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(7)}
		assert.NoError(t, svc.assertCoordinatorOwnsFilter(context.Background(), coordinator, VacancyFilter{CompanyID: int64Ptr(1)}))
	})

	t.Run("filter by department_id must resolve to the coordinator's own school", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{schoolID: 7})
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(7)}
		assert.NoError(t, svc.assertCoordinatorOwnsFilter(context.Background(), coordinator, VacancyFilter{DepartmentID: int64Ptr(2)}))

		svcOtherSchool := newTestService(fakeCompanyScope{schoolID: 999})
		requireAPIErr(t, svcOtherSchool.assertCoordinatorOwnsFilter(context.Background(), coordinator, VacancyFilter{DepartmentID: int64Ptr(2)}), httpx.ErrForbidden)
	})

	t.Run("coordinator without a school and no filter is forbidden", func(t *testing.T) {
		svc := newTestService(fakeCompanyScope{})
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: nil}
		requireAPIErr(t, svc.assertCoordinatorOwnsFilter(context.Background(), coordinator, VacancyFilter{}), httpx.ErrForbidden)
	})
}
