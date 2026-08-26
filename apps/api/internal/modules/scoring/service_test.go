package scoring

import (
	"context"
	"testing"

	"internity/internal/httpx"
	"internity/internal/modules/identity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

// fakeCompanyScope mirrors vacancy's/internship's: resolves every company to
// the same school/department, enough since these tests check one at a time.
type fakeCompanyScope struct {
	schoolID, departmentID int64
	err                    error
}

func (f fakeCompanyScope) ResolveCompanyScope(_ context.Context, _ int64) (int64, int64, error) {
	return f.schoolID, f.departmentID, f.err
}

func newTestService(companies CompanyScopeResolver) *Service {
	// repo nil on purpose: only gate-then-validate paths are exercised here.
	// UpdateScore/DeleteScore/UpdateScorePredicate/DeleteScorePredicate all
	// touch s.repo before their gate check, so they're not testable this way
	// and are deliberately not covered here.
	return NewService(nil, companies, nil, nil, nil)
}

func requireAPIErr(t *testing.T, err error, code httpx.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, code, apiErr.Code)
}

func TestListScores_OtherStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	student := &identity.User{Role: identity.RoleStudent, ID: "student-1"}
	_, _, err := svc.ListScores(context.Background(), student, "student-2", 1, httpx.ListParams{})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestListScores_NonStudentMustManageCompany(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	mentor := &identity.User{Role: identity.RoleMentor, CompanyID: int64Ptr(5)}
	_, _, err := svc.ListScores(context.Background(), mentor, "student-1", 6, httpx.ListParams{})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestCreateScore_PermissionGate(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	student := &identity.User{Role: identity.RoleStudent}
	_, err := svc.CreateScore(context.Background(), student, CreateScoreInput{CompanyID: 1, Score: 80})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestCreateScore_RangeValidation(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	admin := &identity.User{Role: identity.RoleAdmin}

	_, err := svc.CreateScore(context.Background(), admin, CreateScoreInput{CompanyID: 1, Score: -1})
	requireAPIErr(t, err, httpx.ErrValidation)

	_, err = svc.CreateScore(context.Background(), admin, CreateScoreInput{CompanyID: 1, Score: 101})
	requireAPIErr(t, err, httpx.ErrValidation)
}

func TestListScorePredicates_ViewSchoolGate(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}
	_, _, err := svc.ListScorePredicates(context.Background(), coordinator, 6, httpx.ListParams{})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestCreateScorePredicate_ManageSchoolGate(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}
	err := svc.CreateScorePredicate(context.Background(), coordinator, &ScorePredicate{SchoolID: 6, Min: 0, Max: 100})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestCreateScorePredicate_MinMaxValidation(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	admin := &identity.User{Role: identity.RoleAdmin}
	err := svc.CreateScorePredicate(context.Background(), admin, &ScorePredicate{SchoolID: 1, Min: 90, Max: 80})
	requireAPIErr(t, err, httpx.ErrValidation)
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
		student := &identity.User{Role: identity.RoleStudent}
		requireAPIErr(t, svc.assertCanManageCompany(context.Background(), student, 1), httpx.ErrForbidden)
	})
}

func TestAssertCanViewSchool(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})

	t.Run("admin can view any school", func(t *testing.T) {
		admin := &identity.User{Role: identity.RoleAdmin}
		assert.NoError(t, svc.assertCanViewSchool(admin, 999))
	})

	t.Run("coordinator can view only their own school", func(t *testing.T) {
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}
		assert.NoError(t, svc.assertCanViewSchool(coordinator, 5))
		requireAPIErr(t, svc.assertCanViewSchool(coordinator, 6), httpx.ErrForbidden)
	})
}

func TestAssertCanManageSchool(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})

	t.Run("admin can manage any school", func(t *testing.T) {
		admin := &identity.User{Role: identity.RoleAdmin}
		assert.NoError(t, svc.assertCanManageSchool(admin, 999))
	})

	t.Run("coordinator can manage only their own school", func(t *testing.T) {
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}
		assert.NoError(t, svc.assertCanManageSchool(coordinator, 5))
		requireAPIErr(t, svc.assertCanManageSchool(coordinator, 6), httpx.ErrForbidden)
	})

	t.Run("mentor can never manage a school", func(t *testing.T) {
		mentor := &identity.User{Role: identity.RoleMentor, CompanyID: int64Ptr(1)}
		requireAPIErr(t, svc.assertCanManageSchool(mentor, 5), httpx.ErrForbidden)
	})
}
