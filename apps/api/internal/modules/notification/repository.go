package notification

import (
	"context"

	"internity/internal/httpx"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *Repository) CreateMany(ctx context.Context, rows []Notification) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *Repository) ListForUser(ctx context.Context, userID string, params httpx.ListParams) ([]Notification, int64, error) {
	base := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ?", userID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []Notification
	err := r.db.WithContext(ctx).Model(&Notification{}).Where("user_id = ?", userID).
		Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) UnreadCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).Count(&count).Error
	return count, err
}

// MarkAllRead marks every unread notification belonging to userID as read —
// scoped by user_id so one user can never mark another's notifications read.
func (r *Repository) MarkAllRead(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", gorm.Expr("now()")).Error
}
