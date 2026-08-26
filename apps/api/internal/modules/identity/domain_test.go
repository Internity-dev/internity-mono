package identity

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRoleValid(t *testing.T) {
	valid := []Role{RoleAdmin, RoleCoordinator, RoleMentor, RoleStudent}
	for _, r := range valid {
		assert.True(t, r.Valid(), "expected %q to be valid", r)
	}

	invalid := []Role{"", "superadmin", "teacher", "Admin"}
	for _, r := range invalid {
		assert.False(t, r.Valid(), "expected %q to be invalid", r)
	}
}

func TestInviteCodeExpired(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("no expiry never expires", func(t *testing.T) {
		c := InviteCode{ExpiresAt: nil}
		assert.False(t, c.Expired(now))
	})

	t.Run("future expiry is not expired", func(t *testing.T) {
		future := now.Add(24 * time.Hour)
		c := InviteCode{ExpiresAt: &future}
		assert.False(t, c.Expired(now))
	})

	t.Run("past expiry is expired", func(t *testing.T) {
		past := now.Add(-24 * time.Hour)
		c := InviteCode{ExpiresAt: &past}
		assert.True(t, c.Expired(now))
	})
}

func TestSessionActive(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	t.Run("unrevoked and unexpired is active", func(t *testing.T) {
		s := Session{ExpiresAt: future, RevokedAt: nil}
		assert.True(t, s.Active(now))
	})

	t.Run("expired is not active", func(t *testing.T) {
		s := Session{ExpiresAt: past, RevokedAt: nil}
		assert.False(t, s.Active(now))
	})

	t.Run("revoked is not active even if unexpired", func(t *testing.T) {
		revokedAt := now.Add(-time.Minute)
		s := Session{ExpiresAt: future, RevokedAt: &revokedAt}
		assert.False(t, s.Active(now))
	})
}

// TestUserResponseNeverExposesPasswordHash guards against a future edit that
// adds PasswordHash (or anything else internal) to UserResponse and forgets
// it'll be json-encoded straight into every /me, /login, /register response.
func TestUserResponseNeverExposesPasswordHash(t *testing.T) {
	forbidden := []string{"PasswordHash", "password_hash"}

	respType := reflect.TypeOf(UserResponse{})
	for i := 0; i < respType.NumField(); i++ {
		field := respType.Field(i)
		assert.NotContains(t, forbidden, field.Name)
		if tag, ok := field.Tag.Lookup("json"); ok {
			assert.NotContains(t, forbidden, tag)
		}
	}

	u := User{ID: "u1", Role: RoleStudent, Name: "Budi", Email: "budi@example.com", PasswordHash: "$2a$10$secret"}
	resp := u.ToResponse()
	assert.Equal(t, "u1", resp.ID)
	assert.Equal(t, "budi@example.com", resp.Email)
}
