package identity

import (
	"context"
	"errors"
	"time"

	"internity/internal/httpx"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("identity: record not found")

// Repository is the narrow port this module's service depends on — no
// generic Repository[T], just the exact queries Login/Refresh/Register/etc need.
type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByID(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, u *User) error
	UpdateUser(ctx context.Context, u *User) error
	EmailTaken(ctx context.Context, email string) (bool, error)

	CreateSession(ctx context.Context, s *Session) error
	FindActiveSessionByHash(ctx context.Context, hash string) (*Session, error)
	FindSessionByHashAnyState(ctx context.Context, hash string) (*Session, error)
	RevokeSession(ctx context.Context, id string) error
	RevokeSessionFamily(ctx context.Context, familyID string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error

	CreatePasswordResetToken(ctx context.Context, t *PasswordResetToken) error
	FindActivePasswordResetToken(ctx context.Context, hash string) (*PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, id string) error

	FindActiveInviteCode(ctx context.Context, code string) (*InviteCode, error)
	CreateInviteCode(ctx context.Context, c *InviteCode) error
	ResolveCourseScope(ctx context.Context, courseID int64) (*CourseScope, error)

	// ListUserIDsBySchool/ByDepartment back the content module's news
	// publish-notification fan-out — "everyone with a stake in this school /
	// department" (see content.AudienceResolver).
	ListUserIDsBySchool(ctx context.Context, schoolID int64) ([]string, error)
	ListUserIDsByDepartment(ctx context.Context, departmentID int64) ([]string, error)

	// ListStudentsForExport backs the reporting module's student-roster Excel
	// export — a direct join against `courses` (see ResolveCourseScope for
	// the same pattern) rather than a round-trip through the orgs module.
	ListStudentsForExport(ctx context.Context, departmentID int64) ([]StudentExportRow, error)

	// ListUsers backs the admin/coordinator user directory (GET /users) —
	// scope enforcement (which role/school/department/company an actor is
	// even allowed to filter by) happens in service.go, not here.
	ListUsers(ctx context.Context, filter UserFilter, params httpx.ListParams) ([]User, int64, error)
}

// UserFilter is every optional, AND'd narrowing ListUsers accepts. The
// service layer is what turns these into an actual RBAC boundary (e.g.
// forcing SchoolID to the actor's own school for a coordinator) — this type
// itself makes no scope decisions.
type UserFilter struct {
	Role         *Role
	SchoolID     *int64
	DepartmentID *int64
	CompanyID    *int64
}

type StudentExportRow struct {
	Name       string
	NIS        *string
	CourseName string
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *gormRepository) FindUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *gormRepository) CreateUser(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *gormRepository) UpdateUser(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

func (r *gormRepository) EmailTaken(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *gormRepository) CreateSession(ctx context.Context, s *Session) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *gormRepository) FindActiveSessionByHash(ctx context.Context, hash string) (*Session, error) {
	var s Session
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash, time.Now()).
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *gormRepository) FindSessionByHashAnyState(ctx context.Context, hash string) (*Session, error) {
	var s Session
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *gormRepository) RevokeSession(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&Session{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now()).Error
}

func (r *gormRepository) RevokeSessionFamily(ctx context.Context, familyID string) error {
	return r.db.WithContext(ctx).Model(&Session{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", time.Now()).Error
}

func (r *gormRepository) RevokeAllUserSessions(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

func (r *gormRepository) CreatePasswordResetToken(ctx context.Context, t *PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *gormRepository) FindActivePasswordResetToken(ctx context.Context, hash string) (*PasswordResetToken, error) {
	var t PasswordResetToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", hash, time.Now()).
		First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *gormRepository) MarkPasswordResetTokenUsed(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&PasswordResetToken{}).
		Where("id = ?", id).
		Update("used_at", time.Now()).Error
}

func (r *gormRepository) FindActiveInviteCode(ctx context.Context, code string) (*InviteCode, error) {
	var c InviteCode
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *gormRepository) CreateInviteCode(ctx context.Context, c *InviteCode) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *gormRepository) ListUserIDsBySchool(ctx context.Context, schoolID int64) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&User{}).Where("school_id = ?", schoolID).Pluck("id", &ids).Error
	return ids, err
}

func (r *gormRepository) ListUserIDsByDepartment(ctx context.Context, departmentID int64) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&User{}).Where("department_id = ?", departmentID).Pluck("id", &ids).Error
	return ids, err
}

func (r *gormRepository) ListStudentsForExport(ctx context.Context, departmentID int64) ([]StudentExportRow, error) {
	var rows []StudentExportRow
	err := r.db.WithContext(ctx).
		Table("users").
		Select("users.name AS name, users.nis AS nis, courses.name AS course_name").
		Joins("LEFT JOIN courses ON courses.id = users.course_id").
		Where("users.role = 'student' AND users.department_id = ?", departmentID).
		Order("course_name, users.name").
		Find(&rows).Error
	return rows, err
}

// applyUserFilter is shared between the count and the page query below so
// the two can never drift out of sync (same pattern as orgs.applyList).
func applyUserFilter(q *gorm.DB, filter UserFilter) *gorm.DB {
	if filter.Role != nil {
		q = q.Where("role = ?", *filter.Role)
	}
	if filter.SchoolID != nil {
		q = q.Where("school_id = ?", *filter.SchoolID)
	}
	if filter.DepartmentID != nil {
		q = q.Where("department_id = ?", *filter.DepartmentID)
	}
	if filter.CompanyID != nil {
		q = q.Where("company_id = ?", *filter.CompanyID)
	}
	return q
}

func (r *gormRepository) ListUsers(ctx context.Context, filter UserFilter, params httpx.ListParams) ([]User, int64, error) {
	countQ := applyUserFilter(r.db.WithContext(ctx).Model(&User{}), filter)
	if params.Search != "" {
		countQ = countQ.Where("name ILIKE ? OR email ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := applyUserFilter(r.db.WithContext(ctx).Model(&User{}), filter)
	if params.Search != "" {
		q = q.Where("name ILIKE ? OR email ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	var rows []User
	err := q.Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *gormRepository) ResolveCourseScope(ctx context.Context, courseID int64) (*CourseScope, error) {
	var scope CourseScope
	err := r.db.WithContext(ctx).
		Table("courses").
		Select("courses.id AS course_id, courses.department_id AS department_id, departments.school_id AS school_id").
		Joins("JOIN departments ON departments.id = courses.department_id").
		Where("courses.id = ?", courseID).
		Take(&scope).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &scope, nil
}
