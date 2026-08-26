package middleware

import (
	"context"

	"internity/internal/httpx"
	"internity/internal/modules/identity"

	"github.com/gin-gonic/gin"
)

// Authenticator is the narrow slice of identity.Service this middleware
// depends on — kept as an interface so it stays mockable in tests without a DB.
type Authenticator interface {
	Authenticate(ctx context.Context, rawAccessToken string) (*identity.User, error)
}

// RequireAuth validates the session cookie and attaches the *identity.User
// to the context under identity.ContextUserKey. This is the ONLY place that
// happens — every downstream handler/service re-derives the user from the
// context, never from a raw cookie again.
func RequireAuth(auth Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, _ := c.Cookie(identity.CookieSession)
		user, err := auth.Authenticate(c.Request.Context(), raw)
		if err != nil {
			httpx.FailFromErr(c, err)
			return
		}
		c.Set(identity.ContextUserKey, user)
		c.Next()
	}
}

func CurrentUser(c *gin.Context) *identity.User {
	v, ok := c.Get(identity.ContextUserKey)
	if !ok {
		return nil
	}
	u, _ := v.(*identity.User)
	return u
}
