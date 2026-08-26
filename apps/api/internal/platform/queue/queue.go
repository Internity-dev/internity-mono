// Package queue defines the background job contract shared between the API
// process (enqueues) and the worker process (handles) — a job's type string
// and payload shape live here once so neither side can drift from the other.
package queue

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

const (
	TypeNotificationSend   = "notification:send"
	TypeEmailPasswordReset = "email:password_reset"
)

type NotificationSendPayload struct {
	UserID string `json:"user_id"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type EmailPasswordResetPayload struct {
	ToEmail  string `json:"to_email"`
	RawToken string `json:"raw_token"`
}

// Enqueuer wraps an asynq.Client with typed methods, so a caller never
// hand-builds a task type string or marshals a payload itself.
type Enqueuer struct {
	client *asynq.Client
}

func NewEnqueuer(redisOpt asynq.RedisConnOpt) *Enqueuer {
	return &Enqueuer{client: asynq.NewClient(redisOpt)}
}

func (e *Enqueuer) Close() error {
	return e.client.Close()
}

func (e *Enqueuer) EnqueueNotification(ctx context.Context, p NotificationSendPayload) error {
	return e.enqueue(ctx, TypeNotificationSend, p)
}

func (e *Enqueuer) EnqueueEmailPasswordReset(ctx context.Context, p EmailPasswordResetPayload) error {
	return e.enqueue(ctx, TypeEmailPasswordReset, p)
}

func (e *Enqueuer) enqueue(ctx context.Context, taskType string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	// Retry a handful of times with asynq's default backoff before a job
	// lands in the dead-letter set — a transient DB hiccup shouldn't drop a
	// notification, but a permanently broken payload shouldn't retry forever.
	_, err = e.client.EnqueueContext(ctx, asynq.NewTask(taskType, raw), asynq.MaxRetry(5))
	return err
}
