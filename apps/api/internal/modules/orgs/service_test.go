package orgs

import (
	"testing"

	"internity/internal/modules/identity"

	"github.com/stretchr/testify/assert"
)

func int64Ptr(v int64) *int64 { return &v }

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
