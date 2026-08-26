package content

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"internity/internal/httpx"
	"internity/internal/modules/identity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

// fakeDepartmentScope resolves every department to the same school, enough
// since these tests check one at a time. Mirrors vacancy's fakeCompanyScope.
type fakeDepartmentScope struct {
	schoolID int64
	err      error
}

func (f fakeDepartmentScope) ResolveDepartmentSchool(_ context.Context, _ int64) (int64, error) {
	return f.schoolID, f.err
}

func newTestService(departments DepartmentScopeResolver) *Service {
	// repo nil on purpose: every test below returns on a role/scope gate
	// before s.repo is ever touched.
	return NewService(nil, departments, nil, nil)
}

func requireAPIErr(t *testing.T, err error, code httpx.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, code, apiErr.Code)
}

func TestListNews_NonAdminWithoutSchoolForbidden(t *testing.T) {
	svc := newTestService(fakeDepartmentScope{})
	coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: nil}
	_, _, err := svc.ListNews(context.Background(), coordinator, nil, nil, httpx.ListParams{})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestCreateFAQ_NonAdminNonCoordinatorForbidden(t *testing.T) {
	svc := newTestService(fakeDepartmentScope{})
	mentor := &identity.User{Role: identity.RoleMentor}
	err := svc.CreateFAQ(context.Background(), mentor, &FAQ{})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestUpdateFAQ_NonAdminNonCoordinatorForbidden(t *testing.T) {
	svc := newTestService(fakeDepartmentScope{})
	student := &identity.User{Role: identity.RoleStudent}
	_, err := svc.UpdateFAQ(context.Background(), student, 1, nil, nil, nil)
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestDeleteFAQ_NonAdminNonCoordinatorForbidden(t *testing.T) {
	svc := newTestService(fakeDepartmentScope{})
	mentor := &identity.User{Role: identity.RoleMentor}
	err := svc.DeleteFAQ(context.Background(), mentor, 1)
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestAssertCanManageScope(t *testing.T) {
	t.Run("admin manages any scope", func(t *testing.T) {
		svc := newTestService(fakeDepartmentScope{})
		admin := &identity.User{Role: identity.RoleAdmin}
		assert.NoError(t, svc.assertCanManageScope(context.Background(), admin, NewsScopeSchool, 999))
	})

	t.Run("coordinator manages their own school scope only", func(t *testing.T) {
		svc := newTestService(fakeDepartmentScope{})
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}
		assert.NoError(t, svc.assertCanManageScope(context.Background(), coordinator, NewsScopeSchool, 5))
		requireAPIErr(t, svc.assertCanManageScope(context.Background(), coordinator, NewsScopeSchool, 6), httpx.ErrForbidden)
	})

	t.Run("coordinator manages a department scope only when it resolves to their school", func(t *testing.T) {
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(7)}

		svc := newTestService(fakeDepartmentScope{schoolID: 7})
		assert.NoError(t, svc.assertCanManageScope(context.Background(), coordinator, NewsScopeDepartment, 1))

		svcOtherSchool := newTestService(fakeDepartmentScope{schoolID: 999})
		requireAPIErr(t, svcOtherSchool.assertCanManageScope(context.Background(), coordinator, NewsScopeDepartment, 1), httpx.ErrForbidden)
	})

	t.Run("mentor can never manage a scope", func(t *testing.T) {
		svc := newTestService(fakeDepartmentScope{})
		mentor := &identity.User{Role: identity.RoleMentor}
		requireAPIErr(t, svc.assertCanManageScope(context.Background(), mentor, NewsScopeSchool, 1), httpx.ErrForbidden)
	})
}

func TestSlugify(t *testing.T) {
	shape := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*-\d+$`)

	t.Run("lowercases and dashes non-alphanumerics", func(t *testing.T) {
		got := slugify("Hello, World! PKL 2026")
		assert.Regexp(t, shape, got)
		assert.Contains(t, got, "hello-world-pkl-2026-")
	})

	t.Run("trims leading and trailing separators", func(t *testing.T) {
		got := slugify("  --Weird Title--  ")
		assert.Regexp(t, shape, got)
		assert.True(t, strings.HasPrefix(got, "weird-title-"))
	})

	t.Run("two calls produce different suffixes", func(t *testing.T) {
		a := slugify("Same Title")
		b := slugify("Same Title")
		// Not guaranteed distinct (UnixNano can tie under %100000 on a fast
		// clock), but both must still match the expected shape.
		assert.Regexp(t, shape, a)
		assert.Regexp(t, shape, b)
	})
}
