package scoring

import (
	"context"
	"errors"
	"time"

	"internity/internal/httpx"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("scoring: record not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- Scores ---

func (r *Repository) ListScores(ctx context.Context, userID string, companyID int64) ([]Score, error) {
	var rows []Score
	err := r.db.WithContext(ctx).Where("user_id = ? AND company_id = ?", userID, companyID).
		Order("created_at").Find(&rows).Error
	return rows, err
}

// ListScoresPaged backs the handler-facing, paginated listing. ListScores
// above stays untouched and unpaginated on purpose: GenerateCertificate
// needs every score for a placement to compute the certificate average, and
// must never silently see only one page of them.
func (r *Repository) ListScoresPaged(ctx context.Context, userID string, companyID int64, params httpx.ListParams) ([]Score, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Where("user_id = ? AND company_id = ?", userID, companyID)
		if params.Search != "" {
			q = q.Where("name ILIKE ?", "%"+params.Search+"%")
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Score{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Score
	err := scope(r.db.WithContext(ctx).Model(&Score{})).
		Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetScore(ctx context.Context, id int64) (*Score, error) {
	var row Score
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) CreateScore(ctx context.Context, row *Score) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateScore(ctx context.Context, row *Score) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) DeleteScore(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Score{}, id).Error
}

// --- Score predicates ---

// ListScorePredicates returns every predicate for a school, unpaginated —
// GenerateCertificate needs the complete band set to resolve a grade
// correctly and must never see only one page of them.
func (r *Repository) ListScorePredicates(ctx context.Context, schoolID int64) ([]ScorePredicate, error) {
	var rows []ScorePredicate
	err := r.db.WithContext(ctx).Where("school_id = ?", schoolID).Order("min").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListScorePredicatesPaged(ctx context.Context, schoolID int64, params httpx.ListParams) ([]ScorePredicate, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Where("school_id = ?", schoolID)
		if params.Search != "" {
			like := "%" + params.Search + "%"
			q = q.Where("name ILIKE ? OR description ILIKE ?", like, like)
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&ScorePredicate{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ScorePredicate
	err := scope(r.db.WithContext(ctx).Model(&ScorePredicate{})).
		Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetScorePredicate(ctx context.Context, id int64) (*ScorePredicate, error) {
	var row ScorePredicate
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) CreateScorePredicate(ctx context.Context, row *ScorePredicate) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateScorePredicate(ctx context.Context, row *ScorePredicate) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) DeleteScorePredicate(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&ScorePredicate{}, id).Error
}

// --- Certificates ---

func (r *Repository) FindCertificate(ctx context.Context, userID string, companyID int64) (*Certificate, error) {
	var row Certificate
	err := r.db.WithContext(ctx).Where("user_id = ? AND company_id = ?", userID, companyID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CreateCertificate(ctx context.Context, row *Certificate) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateCertificateFileKey(ctx context.Context, id int64, fileKey string) error {
	return r.db.WithContext(ctx).Model(&Certificate{}).Where("id = ?", id).Update("file_key", fileKey).Error
}

// NextCertificateSequence counts certificates already issued for this
// department this year, +1 — used to format CERT-{department}-{year}-{seq}.
// A small collision window exists under concurrent generation for the same
// department in the same second; acceptable at this system's scale (a
// school issuing certificates), and the UNIQUE(user_id, company_id) +
// UNIQUE(certificate_number) constraints still make a true duplicate
// impossible — a collision here would surface as a 409 to retry, not silent
// corruption.
func (r *Repository) NextCertificateSequence(ctx context.Context, departmentID int64, year int) (int, error) {
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	var count int64
	err := r.db.WithContext(ctx).Model(&Certificate{}).
		Where("department_id = ? AND created_at >= ? AND created_at < ?", departmentID, start, end).
		Count(&count).Error
	return int(count) + 1, err
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
