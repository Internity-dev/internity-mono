package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"internity/internal/httpx"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimit is a fixed-window counter backed by Redis: at most `limit`
// requests per `window` for a given key (IP, or IP+email for auth
// endpoints — see keyFn), returning 429 with Retry-After once exceeded. A
// fixed window is a deliberate simplification over a sliding-window/token-
// bucket — it's one INCR+EXPIRE round trip instead of a Lua script, and the
// brute-force/abuse cases this guards (login, register, forgot-password)
// don't need sub-window precision to be effective.
func RateLimit(rdb *redis.Client, limit int, window time.Duration, keyFn func(c *gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ratelimit:" + keyFn(c)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// Redis being unavailable should degrade to "allow", not take the
			// whole API down — rate limiting is defense-in-depth, not the
			// primary auth boundary.
			c.Next()
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, window)
		}

		if count > int64(limit) {
			ttl, _ := rdb.TTL(ctx, key).Result()
			retryAfter := int(ttl.Seconds())
			if retryAfter < 1 {
				retryAfter = int(window.Seconds())
			}
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			httpx.Fail(c, httpx.NewError(httpx.ErrRateLimited, "Too many requests. Please try again later."))
			return
		}

		c.Next()
	}
}

// AuthRateLimitKey scopes the limit to IP+email together (from a JSON body
// field, best-effort) — an attacker spraying many emails from one IP, or
// hammering one email from many IPs, both still get bounded; an IP alone
// would over-block shared networks (school NAT), and an email alone
// wouldn't stop credential stuffing across many emails.
//
// Reads and re-buffers c.Request.Body by hand rather than using Gin's
// ShouldBindBodyWithJSON: that helper caches the bytes under a context key,
// but doesn't reset c.Request.Body itself, so the handler's own plain
// ShouldBindJSON further down the chain would read an already-drained
// body and fail with EOF.
func AuthRateLimitKey(c *gin.Context) string {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return c.ClientIP()
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))

	var body struct {
		Email string `json:"email"`
	}
	_ = json.Unmarshal(raw, &body)
	if body.Email != "" {
		return c.ClientIP() + ":" + body.Email
	}
	return c.ClientIP()
}

func IPRateLimitKey(c *gin.Context) string {
	return c.ClientIP()
}
