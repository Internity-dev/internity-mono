package scoring

import (
	"context"
	"errors"
	"time"

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

func (r *Repository) ListScorePredicates(ctx context.Context, schoolID int64) ([]ScorePredicate, error) {
	var rows []ScorePredicate
	err := r.db.WithContext(ctx).Where("school_id = ?", schoolID).Order("min").Find(&rows).Error
	return rows, err
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
