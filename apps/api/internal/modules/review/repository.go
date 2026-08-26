package review

import (
	"context"
	"errors"

	"internity/internal/httpx"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("review: record not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- Monitors ---

func (r *Repository) ListMonitors(ctx context.Context, studentID *string, companyID *int64, params httpx.ListParams) ([]Monitor, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Joins("JOIN users ON users.id = monitors.student_id")
		if studentID != nil {
			q = q.Where("monitors.student_id = ?", *studentID)
		}
		if companyID != nil {
			q = q.Where("monitors.company_id = ?", *companyID)
		}
		if params.Search != "" {
			like := "%" + params.Search + "%"
			q = q.Where("users.name ILIKE ? OR users.nis ILIKE ? OR monitors.notes ILIKE ? OR monitors.suggest ILIKE ?", like, like, like, like)
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Monitor{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Monitor
	err := scope(r.db.WithContext(ctx).Model(&Monitor{})).
		Order("monitors." + params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) CreateMonitor(ctx context.Context, row *Monitor) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) GetMonitor(ctx context.Context, id int64) (*Monitor, error) {
	var row Monitor
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) UpdateMonitor(ctx context.Context, row *Monitor) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) DeleteMonitor(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Monitor{}, id).Error
}

// --- Questions ---

func (r *Repository) ListQuestions(ctx context.Context, schoolID int64, params httpx.ListParams) ([]Question, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Where("school_id = ?", schoolID)
		if params.Search != "" {
			q = q.Where("question ILIKE ?", "%"+params.Search+"%")
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Question{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Question
	err := scope(r.db.WithContext(ctx).Model(&Question{})).
		Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetQuestion(ctx context.Context, id int64) (*Question, error) {
	var row Question
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) CreateQuestion(ctx context.Context, row *Question) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateQuestion(ctx context.Context, row *Question) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) DeleteQuestion(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Question{}, id).Error
}

// --- Reviews ---

func (r *Repository) ListReviewsForUser(ctx context.Context, userID string) ([]Review, error) {
	var rows []Review
	err := r.db.WithContext(ctx).Where("reviewee_user_id = ?", userID).Order("created_at desc").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListReviewsForCompany(ctx context.Context, companyID int64) ([]Review, error) {
	var rows []Review
	err := r.db.WithContext(ctx).Where("reviewee_company_id = ?", companyID).Order("created_at desc").Find(&rows).Error
	return rows, err
}

func (r *Repository) CreateReview(ctx context.Context, row *Review) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) GetReview(ctx context.Context, id int64) (*Review, error) {
	var row Review
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) UpdateReview(ctx context.Context, row *Review) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
