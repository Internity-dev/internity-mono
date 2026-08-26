// Package redisx wires the shared Redis client used for the session lookup
// cache, rate limiting, and (from Phase 2) the Asynq queue.
package redisx

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// resolveOptions accepts either a bare "host:port" (docker-compose's internal
// service hostnames, no auth) or a full "redis://[:password@]host:port[/db]"
// URL — the latter is what a password-protected remote Redis (e.g. a managed
// instance on Dokploy) requires.
//
// The bare form can't be told apart from a URL by asking redis.ParseURL to
// error on it: "redis" is itself one of ParseURL's recognized schemes, so
// ParseURL("redis:6379") "succeeds" by treating "redis" as the scheme and
// silently defaulting Addr to "localhost:6379", discarding the port
// entirely. Checking for "://" first avoids that trap.
func resolveOptions(connStr string) (*redis.Options, error) {
	if strings.Contains(connStr, "://") {
		return redis.ParseURL(connStr)
	}
	return &redis.Options{Addr: connStr}, nil
}

func Open(connStr string) (*redis.Client, error) {
	opts, err := resolveOptions(connStr)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}
