// Package config loads process configuration from the environment, failing
// fast at startup rather than panicking deep inside a handler on a missing var.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env          string // "development" | "production" | "test"
	Port         string
	CookieSecure bool
	// CookieDomain scopes the CSRF cookie only (session/refresh stay
	// host-only — they're HttpOnly and only ever need to reach this API's
	// own host). Needed whenever the dashboard is served from a different
	// subdomain than the API (e.g. app.example.com vs api.example.com):
	// without it, the CSRF cookie defaults to a host-only scope on the API's
	// domain, which the dashboard's JS can never read via document.cookie,
	// so it can never echo it back as the X-CSRF-Token header. Set to the
	// shared parent domain, e.g. ".example.com". Empty (default) preserves
	// today's host-only behavior for same-host local dev.
	CookieDomain         string
	DatabaseURL          string
	RedisURL             string
	MinioEndpoint        string
	MinioAccessKey       string
	MinioSecretKey       string
	MinioUseSSL          bool
	CORSAllowedOrigins   []string
	OTelExporterEndpoint string // OTLP-HTTP collector host:port; empty disables tracing
}

func Load() (*Config, error) {
	// Best-effort: loads a .env file next to the binary's working directory
	// into the process env if one exists (local/no-Docker dev convenience).
	// Silently no-ops when absent — Docker and CI set real env vars directly.
	_ = godotenv.Load()

	cfg := &Config{
		Env:                  getEnv("APP_ENV", "development"),
		Port:                 getEnv("API_PORT", "8080"),
		CookieSecure:         getEnv("COOKIE_SECURE", "false") == "true",
		CookieDomain:         os.Getenv("COOKIE_DOMAIN"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		RedisURL:             os.Getenv("REDIS_URL"),
		MinioEndpoint:        os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:       os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:       os.Getenv("MINIO_SECRET_KEY"),
		MinioUseSSL:          getEnv("MINIO_USE_SSL", "false") == "true",
		CORSAllowedOrigins:   splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		OTelExporterEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	if cfg.Env == "production" {
		var missing []string
		for name, val := range map[string]string{
			"DATABASE_URL": cfg.DatabaseURL,
			"REDIS_URL":    cfg.RedisURL,
		} {
			if val == "" {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("missing required env vars: %v", missing)
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
