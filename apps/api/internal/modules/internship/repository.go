package internship

import (
	"context"
	"errors"
	"time"

	"internity/internal/httpx"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("internship: record not found")
var ErrVersionConflict = errors.New("internship: version conflict")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- InternDate ---

func (r *Repository) Create(ctx context.Context, id *InternDate) error {
	return r.db.WithContext(ctx).Create(id).Error
}

func (r *Repository) GetByUserCompany(ctx context.Context, userID string, companyID int64) (*InternDate, error) {
	var row InternDate
	err := r.db.WithContext(ctx).Where("user_id = ? AND company_id = ?", userID, companyID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (*InternDate, error) {
	var row InternDate
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) ListForUser(ctx context.Context, userID string) ([]InternDate, error) {
	var rows []InternDate
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&rows).Error
	return rows, err
}

// UpdateWithVersion applies an optimistic-locking update: the row is only
// written if `version` still matches what the caller last read, guarding
// against two staff members editing the same student's dates concurrently
// (a realistic scenario in a school office — see plan section 2.8).
func (r *Repository) UpdateWithVersion(ctx context.Context, row *InternDate, expectedVersion int) error {
	row.Version = expectedVersion + 1
	result := r.db.WithContext(ctx).Model(&InternDate{}).
		Where("id = ? AND version = ?", row.ID, expectedVersion).
		Updates(map[string]any{
			"start_date": row.StartDate, "end_date": row.EndDate, "extended_until": row.ExtendedUntil,
			"status": row.Status, "version": row.Version,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVersionConflict
	}
	return nil
}

// --- Presence statuses ---

func (r *Repository) ListPresenceStatuses(ctx context.Context, schoolID int64, params httpx.ListParams) ([]PresenceStatus, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Where("school_id = ?", schoolID)
		if params.Search != "" {
			like := "%" + params.Search + "%"
			q = q.Where("name ILIKE ? OR description ILIKE ?", like, like)
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&PresenceStatus{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []PresenceStatus
	err := scope(r.db.WithContext(ctx).Model(&PresenceStatus{})).
		Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetPresenceStatus(ctx context.Context, id int64) (*PresenceStatus, error) {
	var row PresenceStatus
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) FindPresenceStatusByKind(ctx context.Context, schoolID int64, kind PresenceStatusKind) (*PresenceStatus, error) {
	var row PresenceStatus
	err := r.db.WithContext(ctx).Where("school_id = ? AND kind = ?", schoolID, kind).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) CreatePresenceStatus(ctx context.Context, row *PresenceStatus) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdatePresenceStatus(ctx context.Context, row *PresenceStatus) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) DeletePresenceStatus(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&PresenceStatus{}, id).Error
}

// --- Presence ---

func (r *Repository) FindPresence(ctx context.Context, userID string, companyID int64, date time.Time) (*Presence, error) {
	var row Presence
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND company_id = ? AND date = ?", userID, companyID, date).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) GetPresence(ctx context.Context, id int64) (*Presence, error) {
	var row Presence
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) CreatePresence(ctx context.Context, row *Presence) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdatePresence(ctx context.Context, row *Presence) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) ListPresencesInRange(ctx context.Context, userID string, companyID int64, from, to time.Time) ([]Presence, error) {
	var rows []Presence
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND company_id = ? AND date BETWEEN ? AND ?", userID, companyID, from, to).
		Order("date").Find(&rows).Error
	return rows, err
}

func (r *Repository) ListPresencesForUser(ctx context.Context, userID string, companyID int64, params httpx.ListParams) ([]Presence, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Where("user_id = ? AND company_id = ?", userID, companyID)
		if params.Search != "" {
			q = q.Where("description ILIKE ?", "%"+params.Search+"%")
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Presence{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Presence
	err := scope(r.db.WithContext(ctx).Model(&Presence{})).
		Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// PresenceExportRow backs the reporting module's presence Excel export — a
// join against presence_statuses for its human-readable name.
type PresenceExportRow struct {
	Date        time.Time
	CheckInAt   *time.Time
	CheckOutAt  *time.Time
	StatusName  string
	IsApproved  bool
	Description *string
}

func (r *Repository) ListPresencesForExport(ctx context.Context, userID string, companyID int64) ([]PresenceExportRow, error) {
	var rows []PresenceExportRow
	err := r.db.WithContext(ctx).
		Table("presences").
		Select("presences.date AS date, presences.check_in_at AS check_in_at, presences.check_out_at AS check_out_at, "+
			"presence_statuses.name AS status_name, presences.is_approved AS is_approved, presences.description AS description").
		Joins("JOIN presence_statuses ON presence_statuses.id = presences.presence_status_id").
		Where("presences.user_id = ? AND presences.company_id = ?", userID, companyID).
		Order("presences.date").
		Find(&rows).Error
	return rows, err
}

func (r *Repository) ListPresencesForApproval(ctx context.Context, companyID int64, params httpx.ListParams) ([]Presence, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Joins("JOIN users ON users.id = presences.user_id").Where("presences.company_id = ?", companyID)
		if params.Search != "" {
			like := "%" + params.Search + "%"
			q = q.Where("users.name ILIKE ? OR users.nis ILIKE ? OR presences.description ILIKE ?", like, like, like)
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Presence{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Presence
	err := scope(r.db.WithContext(ctx).Model(&Presence{})).
		Select("presences.*, users.name AS user_name, users.nis AS user_nis").
		Order("presences." + params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) BulkApprovePresences(ctx context.Context, companyID int64, ids []int64) (int64, error) {
	result := r.db.WithContext(ctx).Model(&Presence{}).
		Where("id IN ? AND company_id = ? AND check_in_at IS NOT NULL AND check_out_at IS NOT NULL", ids, companyID).
		Update("is_approved", true)
	return result.RowsAffected, result.Error
}

// --- Journal ---

func (r *Repository) FindJournal(ctx context.Context, userID string, companyID int64, date time.Time) (*Journal, error) {
	var row Journal
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND company_id = ? AND date = ?", userID, companyID, date).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) GetJournal(ctx context.Context, id int64) (*Journal, error) {
	var row Journal
	if err := r.db.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &row, nil
}

func (r *Repository) CreateJournal(ctx context.Context, row *Journal) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *Repository) UpdateJournal(ctx context.Context, row *Journal) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *Repository) ListJournalsForUser(ctx context.Context, userID string, companyID int64, params httpx.ListParams) ([]Journal, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Where("user_id = ? AND company_id = ?", userID, companyID)
		if params.Search != "" {
			like := "%" + params.Search + "%"
			q = q.Where("work_type ILIKE ? OR description ILIKE ?", like, like)
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Journal{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Journal
	err := scope(r.db.WithContext(ctx).Model(&Journal{})).
		Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) ListJournalsForApproval(ctx context.Context, companyID int64, params httpx.ListParams) ([]Journal, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Joins("JOIN users ON users.id = journals.user_id").Where("journals.company_id = ?", companyID)
		if params.Search != "" {
			like := "%" + params.Search + "%"
			q = q.Where("users.name ILIKE ? OR users.nis ILIKE ? OR journals.work_type ILIKE ? OR journals.description ILIKE ?", like, like, like, like)
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Journal{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Journal
	err := scope(r.db.WithContext(ctx).Model(&Journal{})).
		Select("journals.*, users.name AS user_name, users.nis AS user_nis").
		Order("journals." + params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) BulkApproveJournals(ctx context.Context, companyID int64, ids []int64) (int64, error) {
	result := r.db.WithContext(ctx).Model(&Journal{}).
		Where("id IN ? AND company_id = ? AND work_type IS NOT NULL AND description IS NOT NULL", ids, companyID).
		Update("is_approved", true)
	return result.RowsAffected, result.Error
}

// PresenceStatusCountsScope narrows CountPresencesByKind to one school
// (coordinator) or one company (mentor) instead of the whole platform. Zero
// value (both nil) means unscoped — admin only.
type PresenceStatusCountsScope struct {
	SchoolID  *int64
	CompanyID *int64
}

// CountPresencesByKind is a single GROUP BY query (joined through
// presence_statuses for its `kind`, since presences only stores a
// school-specific presence_status_id) backing the overview dashboard's
// attendance-breakdown chart. Every known kind is present in the result
// even at zero. Bounded to [from, to] (inclusive) — presences is the
// fastest-growing table in the schema, so an all-time GROUP BY here would
// only get slower with every school-year that passes.
func (r *Repository) CountPresencesByKind(ctx context.Context, from, to time.Time, scope PresenceStatusCountsScope) (map[PresenceStatusKind]int64, error) {
	var rows []struct {
		Kind  PresenceStatusKind
		Count int64
	}
	q := r.db.WithContext(ctx).
		Table("presences").
		Select("presence_statuses.kind AS kind, count(*) as count").
		Joins("JOIN presence_statuses ON presence_statuses.id = presences.presence_status_id").
		Where("presences.date BETWEEN ? AND ?", from, to)
	switch {
	case scope.CompanyID != nil:
		q = q.Where("presences.company_id = ?", *scope.CompanyID)
	case scope.SchoolID != nil:
		q = q.Joins("JOIN companies ON companies.id = presences.company_id").
			Joins("JOIN departments ON departments.id = companies.department_id").
			Where("departments.school_id = ?", *scope.SchoolID)
	}
	if err := q.Group("presence_statuses.kind").Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[PresenceStatusKind]int64{
		KindPresent: 0, KindPermitted: 0, KindSick: 0, KindAbsent: 0, KindHoliday: 0,
	}
	for _, row := range rows {
		counts[row.Kind] = row.Count
	}
	return counts, nil
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
