package notification

import (
	"context"

	"internity/internal/httpx"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Send/SendMany are the cross-module entrypoint other modules' services call
// directly (vacancy, and later internship/scoring) — a plain synchronous
// insert, not queued. Deliberate for now: this is one cheap row write, not a
// slow operation (email/PDF) that would justify Asynq — that queue gets
// introduced in the next phase alongside the first genuinely slow job
// (PDF generation), rather than wired up early for work that doesn't need it.
func (s *Service) Send(ctx context.Context, userID, notifType, title, body string) error {
	return s.repo.Create(ctx, &Notification{UserID: userID, Type: notifType, Title: title, Body: body})
}

func (s *Service) SendMany(ctx context.Context, userIDs []string, notifType, title, body string) error {
	rows := make([]Notification, 0, len(userIDs))
	for _, id := range userIDs {
		rows = append(rows, Notification{UserID: id, Type: notifType, Title: title, Body: body})
	}
	return s.repo.CreateMany(ctx, rows)
}

func (s *Service) ListForUser(ctx context.Context, userID string, params httpx.ListParams) ([]Notification, int64, error) {
	return s.repo.ListForUser(ctx, userID, params)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
