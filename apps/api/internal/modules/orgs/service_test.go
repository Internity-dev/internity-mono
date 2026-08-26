package orgs

import (
	"testing"

	"internity/internal/httpx"
	"internity/internal/modules/identity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

// requireAPIErr mirrors content/service_test.go's helper of the same name:
// asserts err is a *httpx.APIError carrying the given code.
func requireAPIErr(t *testing.T, err error, code httpx.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	var apiErr *httpx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, code, apiErr.Code)
}

func TestActorInSchool(t *testing.T) {
	t.Run("nil school_id never matches", func(t *testing.T) {
		actor := &identity.User{Role: identity.RoleCoordinator, SchoolID: nil}
		assert.False(t, actorInSchool(actor, 1))
	})

	t.Run("matching school_id matches", func(t *testing.T) {
		actor := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}
		assert.True(t, actorInSchool(actor, 5))
	})

	t.Run("mismatched school_id does not match", func(t *testing.T) {
		actor := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}
		assert.False(t, actorInSchool(actor, 6))
	})
}

func TestCanManageSchool(t *testing.T) {
	t.Run("admin manages any school", func(t *testing.T) {
		admin := &identity.User{Role: identity.RoleAdmin}
		assert.True(t, canManageSchool(admin, 1))
		assert.True(t, canManageSchool(admin, 999))
	})

	t.Run("coordinator manages only their own school", func(t *testing.T) {
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}
		assert.True(t, canManageSchool(coordinator, 5))
		assert.False(t, canManageSchool(coordinator, 6))
	})

	t.Run("mentor never manages org structure", func(t *testing.T) {
		mentor := &identity.User{Role: identity.RoleMentor, CompanyID: int64Ptr(1)}
		assert.False(t, canManageSchool(mentor, 1))
	})

	t.Run("student never manages org structure", func(t *testing.T) {
		student := &identity.User{Role: identity.RoleStudent, SchoolID: int64Ptr(5)}
		assert.False(t, canManageSchool(student, 5))
	})
}

// TestConflictStillReferenced_DeleteBlockerMessages covers the message
// helpers behind the pre-delete child-count check (DeleteSchool,
// DeleteDepartment, DeleteCompany): the specific "still has N x using it"
// conflict message built ahead of attempting the delete, in place of the
// generic FK-conflict message postgres.TranslateError would otherwise give.
func TestConflictStillReferenced_DeleteBlockerMessages(t *testing.T) {
	t.Run("pluralize singular vs plural", func(t *testing.T) {
		assert.Equal(t, "1 department", pluralize(1, "department", "departments"))
		assert.Equal(t, "3 departments", pluralize(3, "department", "departments"))
		assert.Equal(t, "0 departments", pluralize(0, "department", "departments"))
		assert.Equal(t, "1 company", pluralize(1, "company", "companies"))
		assert.Equal(t, "2 companies", pluralize(2, "company", "companies"))
	})

	t.Run("pluralizeIfAny is empty for a zero count", func(t *testing.T) {
		assert.Equal(t, "", pluralizeIfAny(0, "course", "courses"))
		assert.Equal(t, "1 course", pluralizeIfAny(1, "course", "courses"))
		assert.Equal(t, "4 courses", pluralizeIfAny(4, "course", "courses"))
	})

	t.Run("joinBlockers skips empty clauses and joins the rest with and", func(t *testing.T) {
		assert.Equal(t, "", joinBlockers("", ""))
		assert.Equal(t, "2 courses", joinBlockers("2 courses", ""))
		assert.Equal(t, "3 companies", joinBlockers("", "3 companies"))
		assert.Equal(t, "2 courses and 3 companies", joinBlockers("2 courses", "3 companies"))
	})

	t.Run("conflictStillReferenced names the entity and the blocker as a 409", func(t *testing.T) {
		err := conflictStillReferenced("school", "3 departments")
		requireAPIErr(t, err, httpx.ErrConflict)
		assert.Equal(t, "This school still has 3 departments using it — remove or reassign them first", err.Error())
	})
}

func TestScopedSchoolFilter(t *testing.T) {
	t.Run("admin can request any school or none", func(t *testing.T) {
		admin := &identity.User{Role: identity.RoleAdmin}

		got, err := scopedSchoolFilter(admin, nil)
		assert.NoError(t, err)
		assert.Nil(t, got)

		requested := int64Ptr(7)
		got, err = scopedSchoolFilter(admin, requested)
		assert.NoError(t, err)
		assert.Equal(t, requested, got)
	})

	t.Run("non-admin is pinned to their own school regardless of what they asked for", func(t *testing.T) {
		coordinator := &identity.User{Role: identity.RoleCoordinator, SchoolID: int64Ptr(5)}

		got, err := scopedSchoolFilter(coordinator, int64Ptr(999))
		assert.NoError(t, err)
		assert.Equal(t, int64(5), *got)
	})

	t.Run("non-admin with no school_id is forbidden", func(t *testing.T) {
		mentor := &identity.User{Role: identity.RoleMentor, SchoolID: nil}
		_, err := scopedSchoolFilter(mentor, nil)
		assert.Error(t, err)
	})
}
