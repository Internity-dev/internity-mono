// The Asynq worker binary — processes jobs enqueued by the API process (see
// internal/platform/queue). Kept as a separate entrypoint sharing this
// module's internal packages so domain logic is never duplicated between
// the HTTP API and the worker.
package main

import (
	"context"
	"encoding/json"

	"internity/internal/config"
	"internity/internal/modules/identity"
	"internity/internal/modules/notification"
	"internity/internal/platform/logger"
	"internity/internal/platform/postgres"
	"internity/internal/platform/queue"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger.Init(cfg.Env)

	db, err := postgres.Open(cfg.DatabaseURL, cfg.Env == "development")
	if err != nil {
		log.Fatal().Err(err).Msg("worker: failed to connect to postgres")
	}
	redisConnOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("worker: failed to parse redis url")
	}

	notificationSvc := notification.NewService(notification.NewRepository(db))
	// Still a logged no-op — no SMTP/provider is configured. What changed
	// with the queue is where this runs (off the request path), not what it
	// does; wiring a real provider later is a one-line swap here.
	mailer := identity.NoopMailer{Log: func(event string, fields map[string]any) {
		log.Info().Str("event", event).Fields(fields).Msg("mailer (noop)")
	}}

	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeNotificationSend, func(ctx context.Context, t *asynq.Task) error {
		var p queue.NotificationSendPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		return notificationSvc.Send(ctx, p.UserID, p.Type, p.Title, p.Body)
	})
	mux.HandleFunc(queue.TypeEmailPasswordReset, func(ctx context.Context, t *asynq.Task) error {
		var p queue.EmailPasswordResetPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		return mailer.SendPasswordReset(ctx, p.ToEmail, p.RawToken)
	})

	srv := asynq.NewServer(redisConnOpt, asynq.Config{Concurrency: 10})
	log.Info().Msg("worker: listening for jobs")
	if err := srv.Run(mux); err != nil {
		log.Fatal().Err(err).Msg("worker: server stopped")
	}
}
