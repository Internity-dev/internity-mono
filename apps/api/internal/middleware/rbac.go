package middleware

import (
	"internity/internal/httpx"
	"internity/internal/modules/identity"

	"github.com/gin-gonic/gin"
)

// RequireRole 403s unless the authenticated user's role is one of the
// allowed set. Must run after RequireAuth. This is a coarse gate only —
// per-resource scope (does this appliance belong to MY company?) is a
// separate, per-module check written alongside each module's service (see
// plan section 2.2: the join path to "is this mine" differs per entity, so
// there is deliberately no single generic "ownership" helper here).
func RequireRole(roles ...identity.Role) gin.HandlerFunc {
	allowed := make(map[identity.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *gin.Context) {
		user := CurrentUser(c)
		if user == nil {
			httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
			return
		}
		if !allowed[user.Role] {
			httpx.Fail(c, httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that"))
			return
		}
		c.Next()
	}
}
