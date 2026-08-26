package identity

import (
	"context"
	"testing"

	"internity/internal/httpx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeListUsersRepo embeds a nil Repository so it only needs to implement
// the one method ListUsers actually exercises — any other method call would
// panic on the nil embed, which is exactly what we want: a test relying on
// unimplemented repo behavior should fail loudly, not silently no-op.
type fakeListUsersRepo struct {
	Repository
	gotFilter UserFilter
	rows      []User
	total     int64
}

func (f *fakeListUsersRepo) ListUsers(ctx context.Context, filter UserFilter, params httpx.ListParams) ([]User, int64, error) {
	f.gotFilter = filter
	return f.rows, f.total, nil
}

func newTestService(repo Repository) *Service {
	return NewService(repo, NoopMailer{}, DefaultConfig(), nil)
}

func int64Ptr(v int64) *int64 { return &v }

func TestServiceListUsers_AdminSeesAll(t *testing.T) {
	repo := &fakeListUsersRepo{rows: []User{{ID: "u1"}, {ID: "u2"}}, total: 2}
	svc := newTestService(repo)
	admin := &User{Role: RoleAdmin}

	requested := UserFilter{SchoolID: int64Ptr(7)}
	rows, total, err := svc.ListUsers(context.Background(), admin, requested, httpx.ListParams{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, rows, 2)
	// Admin's requested filter passes straight through — no school pinning.
	assert.Equal(t, requested, repo.gotFilter)
}

func TestServiceListUsers_CoordinatorScopedToOwnSchool(t *testing.T) {
	repo := &fakeListUsersRepo{rows: []User{{ID: "u1"}}, total: 1}
	svc := newTestService(repo)
	coordinator := &User{Role: RoleCoordinator, SchoolID: int64Ptr(5)}

	// Coordinator asks for a different school's users...
	requested := UserFilter{SchoolID: int64Ptr(999)}
	_, _, err := svc.ListUsers(context.Background(), coordinator, requested, httpx.ListParams{Page: 1, Limit: 20})

	require.NoError(t, err)
	// ...but the service force-overrides it to their own school regardless.
	require.NotNil(t, repo.gotFilter.SchoolID)
	assert.Equal(t, int64(5), *repo.gotFilter.SchoolID)
}

func TestServiceListUsers_CoordinatorWithoutSchoolIsForbidden(t *testing.T) {
	repo := &fakeListUsersRepo{}
	svc := newTestService(repo)
	coordinator := &User{Role: RoleCoordinator, SchoolID: nil}

	_, _, err := svc.ListUsers(context.Background(), coordinator, UserFilter{}, httpx.ListParams{Page: 1, Limit: 20})

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}

func TestServiceListUsers_MentorForbidden(t *testing.T) {
	repo := &fakeListUsersRepo{}
	svc := newTestService(repo)
	mentor := &User{Role: RoleMentor, CompanyID: int64Ptr(3)}

	_, _, err := svc.ListUsers(context.Background(), mentor, UserFilter{}, httpx.ListParams{Page: 1, Limit: 20})

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}

func TestServiceListUsers_StudentForbidden(t *testing.T) {
	repo := &fakeListUsersRepo{}
	svc := newTestService(repo)
	student := &User{Role: RoleStudent, SchoolID: int64Ptr(5), DepartmentID: int64Ptr(1)}

	_, _, err := svc.ListUsers(context.Background(), student, UserFilter{}, httpx.ListParams{Page: 1, Limit: 20})

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}
