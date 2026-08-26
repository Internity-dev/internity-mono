package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS allows cross-origin requests from an explicit allowlist only — never
// "*", since the dashboard/landing send cookies (withCredentials: true) and
// the CORS spec forbids combining a wildcard origin with
// Access-Control-Allow-Credentials. The matched origin is echoed back
// per-request (with Vary: Origin so caches don't mix responses across
// origins).
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
