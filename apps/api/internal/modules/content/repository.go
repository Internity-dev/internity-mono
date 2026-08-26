package content

import (
	"context"
	"errors"

	"internity/internal/httpx"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("content: record not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- News ---

type NewsFilter struct {
	ScopeType     *NewsScopeType
	ScopeID       *int64
	PublishedOnly bool
}

func (r *Repository) ListNews(ctx context.Context, filter NewsFilter, params httpx.ListParams) ([]News, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if filter.ScopeType != nil {
			q = q.Where("scope_type = ?", *filter.ScopeType)
		}
		if filter.ScopeID != nil {
			q = q.Where("scope_id = ?", *filter.ScopeID)
		}
		if filter.PublishedOnly {
			q = q.Where("status = ?", NewsPublished)
		}
		return q
	}

	countQ := scope(r.db.WithContext(ctx).Model(&News{}))
	if params.Search != "" {
		countQ = countQ.Where("title ILIKE ?", "%"+params.Search+"%")
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := scope(r.db.WithContext(ctx).Model(&News{}))
	if params.Search != "" {
		q = q.Where("title ILIKE ?", "%"+params.Search+"%")
	}
	var rows []News
	err := q.Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetNews(ctx context.Context, id int64) (*News, error) {
	var row News
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) GetNewsBySlug(ctx context.Context, slug string) (*News, error) {
	var row News
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&row).Error
	if err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) CreateNews(ctx context.Context, row *News) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateNews(ctx context.Context, row *News) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) DeleteNews(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&News{}, id).Error
}

// --- FAQs ---

func (r *Repository) ListFAQs(ctx context.Context) ([]FAQ, error) {
	var rows []FAQ
	err := r.db.WithContext(ctx).Order("sort_order, id").Find(&rows).Error
	return rows, err
}

func (r *Repository) GetFAQ(ctx context.Context, id int64) (*FAQ, error) {
	var row FAQ
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) CreateFAQ(ctx context.Context, row *FAQ) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateFAQ(ctx context.Context, row *FAQ) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) DeleteFAQ(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&FAQ{}, id).Error
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
