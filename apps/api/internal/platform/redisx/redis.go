// Package redisx wires the shared Redis client used for the session lookup
// cache, rate limiting, and (from Phase 2) the Asynq queue.
package redisx

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Open accepts either a bare "host:port" (docker-compose's internal service
// hostnames, no auth) or a full "redis://[:password@]host:port[/db]" URL —
// the latter is what a password-protected remote Redis (e.g. a managed
// instance on Dokploy) requires.
func Open(connStr string) (*redis.Client, error) {
	opts, err := redis.ParseURL(connStr)
	if err != nil {
		opts = &redis.Options{Addr: connStr}
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}
