package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const headerRequestID = "X-Request-ID"

// RequestID propagates an inbound X-Request-ID (e.g. from a reverse proxy) or
// generates one, so every log line and error envelope can be grepped to one request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(headerRequestID)
		if id == "" {
			id = generateID()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set(headerRequestID, id)
		c.Next()
	}
}

func generateID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
