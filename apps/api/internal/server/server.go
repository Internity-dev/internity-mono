// Package server wires the Gin engine: global middleware plus route
// registration. Domain modules own their own routes.go — this file only
// owns cross-cutting concerns, health checks, and the auth/CSRF gate that
// separates the public and authenticated route groups.
package server

import (
	"context"
	"net/http"
	"time"

	"internity/internal/config"
	"internity/internal/middleware"
	"internity/internal/modules/content"
	"internity/internal/modules/identity"
	"internity/internal/modules/internship"
	"internity/internal/modules/notification"
	"internity/internal/modules/orgs"
	"internity/internal/modules/reporting"
	"internity/internal/modules/review"
	"internity/internal/modules/scoring"
	"internity/internal/modules/vacancy"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Dependencies struct {
	DB            *gorm.DB
	Redis         *redis.Client
	Identity      *identity.Handler
	Orgs          *orgs.Handler
	Vacancy       *vacancy.Handler
	Notification  *notification.Handler
	Internship    *internship.Handler
	Scoring       *scoring.Handler
	Content       *content.Handler
	Review        *review.Handler
	Reporting     *reporting.Handler
	Authenticator middleware.Authenticator
}

func New(cfg *config.Config, deps Dependencies) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(), gin.Logger(), middleware.CORS(cfg.CORSAllowedOrigins), middleware.SecurityHeaders(cfg.CookieSecure))

	registerHealthRoutes(r, deps.DB, deps.Redis)

	api := r.Group("/api/v1")

	// 10, not 5: this key is shared across login/register/forgot-password/
	// reset-password (see AuthRateLimitKey), and a realistic multi-step
	// session for one account — e.g. a student re-authenticating a few
	// times while switching between the dashboard's own role-scoped views —
	// can legitimately need more than 5 within one 5-minute window.
	// Confirmed live, not theoretical: the E2E suite's own business-flow
	// tests (all real, single-account UI flows, not a bot) tripped this at
	// 5 purely from their own necessary logins. 10 is still tight enough to
	// meaningfully bound brute-force/credential-stuffing (worst case ~2,880
	// guesses/day against a bcrypt hash) while giving legitimate multi-step
	// usage headroom.
	authRateLimit := middleware.RateLimit(deps.Redis, 10, 5*time.Minute, middleware.AuthRateLimitKey)
	deps.Identity.RegisterPublicRoutes(api, authRateLimit)
	deps.Content.RegisterPublicRoutes(api)

	authed := api.Group("")
	authed.Use(middleware.RequireAuth(deps.Authenticator), middleware.RequireCSRF())
	deps.Identity.RegisterAuthenticatedRoutes(authed)
	deps.Orgs.RegisterRoutes(authed)
	deps.Vacancy.RegisterRoutes(authed)
	deps.Notification.RegisterRoutes(authed)
	deps.Internship.RegisterRoutes(authed)
	deps.Scoring.RegisterRoutes(authed)
	deps.Content.RegisterAuthenticatedRoutes(authed)
	deps.Review.RegisterRoutes(authed)
	deps.Reporting.RegisterRoutes(authed)

	return r
}

func registerHealthRoutes(r *gin.Engine, db *gorm.DB, rdb *redis.Client) {
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		checks := gin.H{}
		ready := true

		if sqlDB, err := db.DB(); err != nil || sqlDB.PingContext(ctx) != nil {
			checks["postgres"] = "down"
			ready = false
		} else {
			checks["postgres"] = "ok"
		}

		if err := rdb.Ping(ctx).Err(); err != nil {
			checks["redis"] = "down"
			ready = false
		} else {
			checks["redis"] = "ok"
		}

		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"status": map[bool]string{true: "ok", false: "degraded"}[ready], "checks": checks})
	})
}
