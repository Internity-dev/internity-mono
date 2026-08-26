package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders sets the small set of response headers that matter for a
// JSON API (not an HTML-rendering one, so no CSP script/style directives to
// tune — default-src 'none' is correct since nothing here is ever meant to
// render as a page). hstsEnabled should track whatever already gates the
// Secure cookie attribute (cfg.CookieSecure) — HSTS is meaningless, and
// actively wrong to send, over a plain-HTTP local dev connection.
func SecurityHeaders(hstsEnabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'none'")
		if hstsEnabled {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Next()
	}
}
