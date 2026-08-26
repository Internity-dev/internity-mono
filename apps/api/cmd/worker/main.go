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
	"internity/internal/platform/otel"
	"internity/internal/platform/postgres"
	"internity/internal/platform/queue"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
	gootel "go.opentelemetry.io/otel"
)

// tracer names spans for the two job handlers below. There's no maintained
// asynq-otel-contrib middleware to reach for, so each handler starts its own
// span manually rather than building a generic middleware layer for two
// handlers.
var tracer = gootel.Tracer("internity-worker")

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger.Init(cfg.Env)

	shutdownTracing, err := otel.Init(context.Background(), "internity-worker", cfg.OTelExporterEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("worker: failed to init opentelemetry")
	}
	defer func() { _ = shutdownTracing(context.Background()) }()

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
		ctx, span := tracer.Start(ctx, queue.TypeNotificationSend)
		defer span.End()
		var p queue.NotificationSendPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		return notificationSvc.Send(ctx, p.UserID, p.Type, p.Title, p.Body)
	})
	mux.HandleFunc(queue.TypeEmailPasswordReset, func(ctx context.Context, t *asynq.Task) error {
		ctx, span := tracer.Start(ctx, queue.TypeEmailPasswordReset)
		defer span.End()
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
