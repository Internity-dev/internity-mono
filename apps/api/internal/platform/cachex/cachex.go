// Package cachex is a small get-or-compute JSON cache over Redis, built for
// dashboard aggregate queries (status-breakdown counts) that would otherwise
// re-scan a fast-growing table on every admin page load. A short TTL is a
// much simpler correctness story here than wiring cache-busting into every
// appliance/presence mutation call site, and a dashboard chart doesn't need
// to be accurate to the second.
package cachex

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// GetOrSet returns the cached value at key if present and parses cleanly,
// otherwise calls compute, caches the result (best-effort — a Redis write
// failure doesn't fail the request), and returns it.
func GetOrSet[T any](ctx context.Context, rdb *redis.Client, key string, ttl time.Duration, compute func() (T, error)) (T, error) {
	var out T
	if raw, err := rdb.Get(ctx, key).Bytes(); err == nil {
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	out, err := compute()
	if err != nil {
		return out, err
	}
	if raw, err := json.Marshal(out); err == nil {
		rdb.Set(ctx, key, raw, ttl)
	}
	return out, nil
}
