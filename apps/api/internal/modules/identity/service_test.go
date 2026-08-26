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

func newTestService(repo Repository, companies CompanyScopeResolver) *Service {
	return NewService(repo, NoopMailer{}, DefaultConfig(), nil, companies)
}

func int64Ptr(v int64) *int64 { return &v }

// fakeCompanyScope resolves every company ID to the same school/department —
// enough for the mentor-lookup branch tests below, which only ever check
// one company at a time.
type fakeCompanyScope struct {
	schoolID, departmentID int64
	err                    error
}

func (f fakeCompanyScope) ResolveCompanyScope(_ context.Context, _ int64) (int64, int64, error) {
	return f.schoolID, f.departmentID, f.err
}

func TestServiceListUsers_AdminSeesAll(t *testing.T) {
	repo := &fakeListUsersRepo{rows: []User{{ID: "u1"}, {ID: "u2"}}, total: 2}
	svc := newTestService(repo, nil)
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
	svc := newTestService(repo, nil)
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
	svc := newTestService(repo, nil)
	coordinator := &User{Role: RoleCoordinator, SchoolID: nil}

	_, _, err := svc.ListUsers(context.Background(), coordinator, UserFilter{}, httpx.ListParams{Page: 1, Limit: 20})

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}

func TestServiceListUsers_MentorForbidden(t *testing.T) {
	repo := &fakeListUsersRepo{}
	svc := newTestService(repo, nil)
	mentor := &User{Role: RoleMentor, CompanyID: int64Ptr(3)}

	_, _, err := svc.ListUsers(context.Background(), mentor, UserFilter{}, httpx.ListParams{Page: 1, Limit: 20})

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}

func TestServiceListUsers_StudentForbidden(t *testing.T) {
	repo := &fakeListUsersRepo{}
	svc := newTestService(repo, nil)
	student := &User{Role: RoleStudent, SchoolID: int64Ptr(5), DepartmentID: int64Ptr(1)}

	_, _, err := svc.ListUsers(context.Background(), student, UserFilter{}, httpx.ListParams{Page: 1, Limit: 20})

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}

func rolePtr(r Role) *Role { return &r }

func TestServiceListUsers_CoordinatorMentorLookupInOwnSchool(t *testing.T) {
	repo := &fakeListUsersRepo{rows: []User{{ID: "m1", Role: RoleMentor}}, total: 1}
	companies := fakeCompanyScope{schoolID: 5}
	svc := newTestService(repo, companies)
	coordinator := &User{Role: RoleCoordinator, SchoolID: int64Ptr(5)}

	requested := UserFilter{Role: rolePtr(RoleMentor), CompanyID: int64Ptr(42)}
	rows, total, err := svc.ListUsers(context.Background(), coordinator, requested, httpx.ListParams{Page: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, rows, 1)
	// company_id does the real narrowing here — school_id must NOT get
	// force-pinned the way it does for every other coordinator query, since
	// mentor rows never have one and that would zero out the result.
	assert.Nil(t, repo.gotFilter.SchoolID)
	require.NotNil(t, repo.gotFilter.CompanyID)
	assert.Equal(t, int64(42), *repo.gotFilter.CompanyID)
}

func TestServiceListUsers_CoordinatorMentorLookupOutsideOwnSchoolIsForbidden(t *testing.T) {
	repo := &fakeListUsersRepo{}
	companies := fakeCompanyScope{schoolID: 999} // a different school than the coordinator's
	svc := newTestService(repo, companies)
	coordinator := &User{Role: RoleCoordinator, SchoolID: int64Ptr(5)}

	requested := UserFilter{Role: rolePtr(RoleMentor), CompanyID: int64Ptr(42)}
	_, _, err := svc.ListUsers(context.Background(), coordinator, requested, httpx.ListParams{Page: 1, Limit: 20})

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}

func TestServiceListUsers_CoordinatorMentorLookupWithoutCompanyIDIsRejected(t *testing.T) {
	repo := &fakeListUsersRepo{}
	svc := newTestService(repo, fakeCompanyScope{})
	coordinator := &User{Role: RoleCoordinator, SchoolID: int64Ptr(5)}

	requested := UserFilter{Role: rolePtr(RoleMentor)}
	_, _, err := svc.ListUsers(context.Background(), coordinator, requested, httpx.ListParams{Page: 1, Limit: 20})

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrValidation, apiErr.Code)
}

// fakeCreateStaffRepo embeds a nil Repository for the same reason
// fakeListUsersRepo does — see that type's comment.
type fakeCreateStaffRepo struct {
	Repository
	emailTaken bool
	created    *User
}

func (f *fakeCreateStaffRepo) EmailTaken(_ context.Context, _ string) (bool, error) {
	return f.emailTaken, nil
}

func (f *fakeCreateStaffRepo) CreateUser(_ context.Context, u *User) error {
	f.created = u
	return nil
}

func validStaffInput(role Role) CreateStaffAccountInput {
	return CreateStaffAccountInput{
		Name: "New Staff", Email: "new-staff@internity.test",
		Password: "password123", PasswordConfirmation: "password123",
		Role: role,
	}
}

func TestCreateStaffAccount_AdminCreatesCoordinatorAnySchool(t *testing.T) {
	repo := &fakeCreateStaffRepo{}
	svc := newTestService(repo, fakeCompanyScope{})
	admin := &User{Role: RoleAdmin}

	in := validStaffInput(RoleCoordinator)
	in.SchoolID = int64Ptr(999)
	user, err := svc.CreateStaffAccount(context.Background(), admin, in)

	require.NoError(t, err)
	require.NotNil(t, repo.created)
	assert.Equal(t, RoleCoordinator, user.Role)
	assert.Equal(t, int64(999), *user.SchoolID)
	assert.NotEmpty(t, user.ID)
	assert.NotEqual(t, "password123", user.PasswordHash) // hashed, not stored raw
}

func TestCreateStaffAccount_AdminCreatesMentorAnyCompany(t *testing.T) {
	repo := &fakeCreateStaffRepo{}
	svc := newTestService(repo, fakeCompanyScope{schoolID: 1})
	admin := &User{Role: RoleAdmin}

	in := validStaffInput(RoleMentor)
	in.CompanyID = int64Ptr(42)
	user, err := svc.CreateStaffAccount(context.Background(), admin, in)

	require.NoError(t, err)
	assert.Equal(t, RoleMentor, user.Role)
	assert.Equal(t, int64(42), *user.CompanyID)
}

func TestCreateStaffAccount_CoordinatorCreatesMentorInOwnSchool(t *testing.T) {
	repo := &fakeCreateStaffRepo{}
	svc := newTestService(repo, fakeCompanyScope{schoolID: 5})
	coordinator := &User{Role: RoleCoordinator, SchoolID: int64Ptr(5)}

	in := validStaffInput(RoleMentor)
	in.CompanyID = int64Ptr(42)
	user, err := svc.CreateStaffAccount(context.Background(), coordinator, in)

	require.NoError(t, err)
	assert.Equal(t, RoleMentor, user.Role)
}

func TestCreateStaffAccount_CoordinatorCreatesMentorOutsideOwnSchoolIsForbidden(t *testing.T) {
	repo := &fakeCreateStaffRepo{}
	svc := newTestService(repo, fakeCompanyScope{schoolID: 999})
	coordinator := &User{Role: RoleCoordinator, SchoolID: int64Ptr(5)}

	in := validStaffInput(RoleMentor)
	in.CompanyID = int64Ptr(42)
	_, err := svc.CreateStaffAccount(context.Background(), coordinator, in)

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
	assert.Nil(t, repo.created)
}

func TestCreateStaffAccount_CoordinatorCannotCreateAnotherCoordinator(t *testing.T) {
	repo := &fakeCreateStaffRepo{}
	svc := newTestService(repo, fakeCompanyScope{})
	coordinator := &User{Role: RoleCoordinator, SchoolID: int64Ptr(5)}

	in := validStaffInput(RoleCoordinator)
	in.SchoolID = int64Ptr(5)
	_, err := svc.CreateStaffAccount(context.Background(), coordinator, in)

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}

func TestCreateStaffAccount_NonAdminNonCoordinatorForbidden(t *testing.T) {
	repo := &fakeCreateStaffRepo{}
	svc := newTestService(repo, fakeCompanyScope{})
	mentor := &User{Role: RoleMentor, CompanyID: int64Ptr(1)}

	_, err := svc.CreateStaffAccount(context.Background(), mentor, validStaffInput(RoleMentor))

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrForbidden, apiErr.Code)
}

func TestCreateStaffAccount_InvalidRoleRejected(t *testing.T) {
	repo := &fakeCreateStaffRepo{}
	svc := newTestService(repo, fakeCompanyScope{})
	admin := &User{Role: RoleAdmin}

	_, err := svc.CreateStaffAccount(context.Background(), admin, validStaffInput(RoleStudent))

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrValidation, apiErr.Code)
}

func TestCreateStaffAccount_PasswordMismatchRejected(t *testing.T) {
	repo := &fakeCreateStaffRepo{}
	svc := newTestService(repo, fakeCompanyScope{})
	admin := &User{Role: RoleAdmin}

	in := validStaffInput(RoleCoordinator)
	in.SchoolID = int64Ptr(1)
	in.PasswordConfirmation = "different"
	_, err := svc.CreateStaffAccount(context.Background(), admin, in)

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrValidation, apiErr.Code)
}

func TestCreateStaffAccount_DuplicateEmailIsConflict(t *testing.T) {
	repo := &fakeCreateStaffRepo{emailTaken: true}
	svc := newTestService(repo, fakeCompanyScope{})
	admin := &User{Role: RoleAdmin}

	in := validStaffInput(RoleCoordinator)
	in.SchoolID = int64Ptr(1)
	_, err := svc.CreateStaffAccount(context.Background(), admin, in)

	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, httpx.ErrConflict, apiErr.Code)
}
