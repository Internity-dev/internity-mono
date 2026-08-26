package internship

import (
	"context"
	"testing"
	"time"

	"internity/internal/httpx"
	"internity/internal/modules/identity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

// fakeCompanyScope resolves every company to the same school/department —
// enough for the tests below, which only ever check one at a time. Mirrors
// vacancy's and identity's fakeCompanyScope.
type fakeCompanyScope struct {
	schoolID, departmentID int64
	err                    error
}

func (f fakeCompanyScope) ResolveCompanyScope(_ context.Context, _ int64) (int64, int64, error) {
	return f.schoolID, f.departmentID, f.err
}

func newTestService(companies CompanyScopeResolver) *Service {
	// repo is deliberately nil — every test below either returns on a role
	// gate before s.repo is ever touched, or exercises a companies-only /
	// pure helper that never reaches s.repo at all. See vacancy's
	// service_test.go for the same reasoning.
	return NewService(nil, companies, nil, nil)
}

func requireAPIErr(t *testing.T, err error, code httpx.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, code, apiErr.Code)
}

func TestListMyInternDates_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	_, err := svc.ListMyInternDates(context.Background(), &identity.User{Role: identity.RoleCoordinator})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestCheckIn_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	_, err := svc.CheckIn(context.Background(), &identity.User{Role: identity.RoleMentor}, CheckInInput{CompanyID: 1})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestCheckOut_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	_, err := svc.CheckOut(context.Background(), &identity.User{Role: identity.RoleAdmin}, 1)
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestFileExcuse_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	_, err := svc.FileExcuse(context.Background(), &identity.User{Role: identity.RoleCoordinator}, ExcuseInput{CompanyID: 1, Kind: KindSick})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestFileExcuse_InvalidKindRejected(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	student := &identity.User{Role: identity.RoleStudent}
	// The kind check runs before any repo lookup, so this is reachable even
	// with a nil repo — KindAbsent/KindHoliday are valid PresenceStatusKind
	// values elsewhere in the domain, just not valid *excuse* kinds.
	_, err := svc.FileExcuse(context.Background(), student, ExcuseInput{CompanyID: 1, Kind: KindAbsent})
	requireAPIErr(t, err, httpx.ErrValidation)
}

func TestListMyPresences_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	_, _, err := svc.ListMyPresences(context.Background(), &identity.User{Role: identity.RoleMentor, CompanyID: int64Ptr(1)}, 1, httpx.ListParams{})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestUpsertJournal_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	_, err := svc.UpsertJournal(context.Background(), &identity.User{Role: identity.RoleAdmin}, JournalInput{CompanyID: 1})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestListMyJournals_NonStudentForbidden(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	_, _, err := svc.ListMyJournals(context.Background(), &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(1)}, 1, httpx.ListParams{})
	requireAPIErr(t, err, httpx.ErrForbidden)
}

func TestAssertCanAccessPlacement(t *testing.T) {
	svc := newTestService(fakeCompanyScope{})
	placement := &InternDate{UserID: "student-1", CompanyID: 5}

	t.Run("admin can access any placement", func(t *testing.T) {
		admin := &identity.User{Role: identity.RoleAdmin}
		assert.NoError(t, svc.assertCanAccessPlacement(context.Background(), admin, placement))
	})

	t.Run("owning student can access their own placement", func(t *testing.T) {
		owner := &identity.User{Role: identity.RoleStudent, ID: "student-1"}
		assert.NoError(t, svc.assertCanAccessPlacement(context.Background(), owner, placement))
	})

	t.Run("a different student cannot access someone else's placement", func(t *testing.T) {
		other := &identity.User{Role: identity.RoleStudent, ID: "student-2"}
		requireAPIErr(t, svc.assertCanAccessPlacement(context.Background(), other, placement), httpx.ErrForbidden)
	})

	t.Run("mentor can access a placement only at their own company", func(t *testing.T) {
		mentor := &identity.User{Role: identity.RoleMentor, CompanyID: int64Ptr(5)}
		assert.NoError(t, svc.assertCanAccessPlacement(context.Background(), mentor, placement))

		otherMentor := &identity.User{Role: identity.RoleMentor, CompanyID: int64Ptr(6)}
		requireAPIErr(t, svc.assertCanAccessPlacement(context.Background(), otherMentor, placement), httpx.ErrForbidden)
	})
}

func TestAssertCanManagePlacement(t *testing.T) {
	svc := newTestService(fakeCompanyScope{schoolID: 7})
	placement := &InternDate{UserID: "student-1", CompanyID: 5}

	t.Run("owning student can manage their own placement", func(t *testing.T) {
		owner := &identity.User{Role: identity.RoleStudent, ID: "student-1"}
		assert.NoError(t, svc.assertCanManagePlacement(context.Background(), owner, placement))
	})

	t.Run("a different student cannot manage someone else's placement", func(t *testing.T) {
		other := &identity.User{Role: identity.RoleStudent, ID: "student-2"}
		requireAPIErr(t, svc.assertCanManagePlacement(context.Background(), other, placement), httpx.ErrForbidden)
	})

	t.Run("coordinator can manage a placement only within their own school", func(t *testing.T) {
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(7)}
		assert.NoError(t, svc.assertCanManagePlacement(context.Background(), coordinator, placement))

		svcOtherSchool := newTestService(fakeCompanyScope{schoolID: 999})
		requireAPIErr(t, svcOtherSchool.assertCanManagePlacement(context.Background(), coordinator, placement), httpx.ErrForbidden)
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

func TestResolvePresenceCountsRange(t *testing.T) {
	t.Run("explicit from/to pass through unchanged", func(t *testing.T) {
		from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
		gotFrom, gotTo := resolvePresenceCountsRange(&from, &to)
		assert.True(t, from.Equal(gotFrom))
		assert.True(t, to.Equal(gotTo))
	})

	t.Run("defaults to the first of the current month through now", func(t *testing.T) {
		before := time.Now()
		gotFrom, gotTo := resolvePresenceCountsRange(nil, nil)
		after := time.Now()

		assert.Equal(t, 1, gotFrom.Day())
		assert.Equal(t, before.Month(), gotFrom.Month())
		assert.Equal(t, before.Year(), gotFrom.Year())
		assert.False(t, gotTo.Before(before))
		assert.False(t, gotTo.After(after))
	})

	t.Run("a supplied from with a nil to still defaults to now", func(t *testing.T) {
		from := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
		before := time.Now()
		gotFrom, gotTo := resolvePresenceCountsRange(&from, nil)
		after := time.Now()

		assert.True(t, from.Equal(gotFrom))
		assert.False(t, gotTo.Before(before))
		assert.False(t, gotTo.After(after))
	})
}
