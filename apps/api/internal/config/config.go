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
	Env                  string // "development" | "production" | "test"
	Port                 string
	CookieSecure         bool
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
