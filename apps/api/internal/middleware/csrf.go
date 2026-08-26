package middleware

import (
	"crypto/subtle"
	"net/http"

	"internity/internal/httpx"
	"internity/internal/modules/identity"

	"github.com/gin-gonic/gin"
)

const CSRFHeader = "X-CSRF-Token"

var mutatingMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// RequireCSRF implements double-submit cookie verification: the CSRF cookie
// (set alongside the session cookie on login, readable by JS on purpose —
// see identity.Handler.setAuthCookies) must be echoed back as X-CSRF-Token
// on every mutating request. A cross-site form/script can trigger a cookie
// to be *sent*, but it cannot *read* another origin's cookie to put its
// value in a header — that's what makes this work. Chosen over the
// synchronizer-token pattern because there is no server-rendered form to
// embed a per-page token into (see plan section 2.3 / ADR).
func RequireCSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !mutatingMethods[c.Request.Method] {
			c.Next()
			return
		}

		cookieVal, err := c.Cookie(identity.CookieCSRF)
		headerVal := c.GetHeader(CSRFHeader)
		if err != nil || cookieVal == "" || headerVal == "" ||
			subtle.ConstantTimeCompare([]byte(cookieVal), []byte(headerVal)) != 1 {
			httpx.Fail(c, httpx.NewError(httpx.ErrForbidden, "CSRF token missing or invalid"))
			return
		}
		c.Next()
	}
}
