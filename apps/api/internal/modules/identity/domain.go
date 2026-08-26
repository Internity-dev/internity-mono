// Package identity owns authentication (cookie sessions, CSRF-adjacent token
// issuance, password reset) and the User entity itself, including the
// role+scope model that replaces legacy's pivot-table-plus-`.first()` pattern.
package identity

import "time"

type Role string

const (
	RoleAdmin       Role = "admin"
	RoleCoordinator Role = "coordinator"
	RoleMentor      Role = "mentor"
	RoleStudent     Role = "student"
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleCoordinator, RoleMentor, RoleStudent:
		return true
	}
	return false
}

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

// User mirrors the `users` table exactly (migration 000006) — schema is owned
// by golang-migrate, this struct is only a GORM query-builder mapping.
type User struct {
	ID           string `gorm:"column:id;primaryKey"`
	Role         Role   `gorm:"column:role"`
	SchoolID     *int64 `gorm:"column:school_id"`
	DepartmentID *int64 `gorm:"column:department_id"`
	CompanyID    *int64 `gorm:"column:company_id"`
	CourseID     *int64 `gorm:"column:course_id"`

	Name         string     `gorm:"column:name"`
	Email        string     `gorm:"column:email"`
	PasswordHash string     `gorm:"column:password_hash"`
	NIS          *string    `gorm:"column:nis"`
	Gender       *Gender    `gorm:"column:gender"`
	Bio          *string    `gorm:"column:bio"`
	Address      *string    `gorm:"column:address"`
	Phone        *string    `gorm:"column:phone"`
	DateOfBirth  *time.Time `gorm:"column:date_of_birth"`
	AvatarKey    *string    `gorm:"column:avatar_key"`
	ResumeKey    *string    `gorm:"column:resume_key"`
	Skills       *string    `gorm:"column:skills"`
	IsActive     bool       `gorm:"column:is_active"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at"`
	LastLoginIP  *string    `gorm:"column:last_login_ip"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
}

func (User) TableName() string { return "users" }

type SessionKind string

const (
	SessionAccess  SessionKind = "access"
	SessionRefresh SessionKind = "refresh"
)

// Session backs both the access and refresh httpOnly cookies. Rows sharing a
// FamilyID form one rotation chain — reuse of an already-revoked refresh
// token is theft evidence and revokes the whole family (see service.go Refresh).
type Session struct {
	ID        string      `gorm:"column:id;primaryKey"`
	UserID    string      `gorm:"column:user_id"`
	TokenHash string      `gorm:"column:token_hash"`
	Kind      SessionKind `gorm:"column:kind"`
	FamilyID  string      `gorm:"column:family_id"`
	UserAgent string      `gorm:"column:user_agent"`
	IP        string      `gorm:"column:ip"`
	ExpiresAt time.Time   `gorm:"column:expires_at"`
	RevokedAt *time.Time  `gorm:"column:revoked_at"`
	CreatedAt time.Time   `gorm:"column:created_at"`
}

func (Session) TableName() string { return "sessions" }

func (s Session) Active(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}

type PasswordResetToken struct {
	ID        string     `gorm:"column:id;primaryKey"`
	UserID    string     `gorm:"column:user_id"`
	TokenHash string     `gorm:"column:token_hash"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

// InviteCode is what a student self-registers with — always resolves to a
// course, which already implies department + school via FK chain (see
// migration 000009 for why this isn't the legacy's polymorphic version).
type InviteCode struct {
	ID        int64      `gorm:"column:id;primaryKey" json:"id"`
	Code      string     `gorm:"column:code" json:"code"`
	CourseID  int64      `gorm:"column:course_id" json:"course_id"`
	ExpiresAt *time.Time `gorm:"column:expires_at" json:"expires_at"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (InviteCode) TableName() string { return "invite_codes" }

func (c InviteCode) Expired(now time.Time) bool {
	return c.ExpiresAt != nil && now.After(*c.ExpiresAt)
}

// CourseScope is the resolved department/school for a course — used to fill
// a newly self-registered student's scope columns.
type CourseScope struct {
	CourseID     int64
	DepartmentID int64
	SchoolID     int64
}

// UserResponse is the only shape a User is ever serialized as in an API
// response — PasswordHash (and anything else internal) never round-trips to
// the client. Handlers must call ToResponse(), never json-encode *User directly.
type UserResponse struct {
	ID           string     `json:"id"`
	Role         Role       `json:"role"`
	SchoolID     *int64     `json:"school_id,omitempty"`
	DepartmentID *int64     `json:"department_id,omitempty"`
	CompanyID    *int64     `json:"company_id,omitempty"`
	CourseID     *int64     `json:"course_id,omitempty"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	NIS          *string    `json:"nis,omitempty"`
	Gender       *Gender    `json:"gender,omitempty"`
	Bio          *string    `json:"bio,omitempty"`
	Address      *string    `json:"address,omitempty"`
	Phone        *string    `json:"phone,omitempty"`
	DateOfBirth  *time.Time `json:"date_of_birth,omitempty"`
	AvatarKey    *string    `json:"avatar_key,omitempty"`
	ResumeKey    *string    `json:"resume_key,omitempty"`
	Skills       *string    `json:"skills,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (u User) ToResponse() UserResponse {
	return UserResponse{
		ID: u.ID, Role: u.Role, SchoolID: u.SchoolID, DepartmentID: u.DepartmentID,
		CompanyID: u.CompanyID, CourseID: u.CourseID, Name: u.Name, Email: u.Email,
		NIS: u.NIS, Gender: u.Gender, Bio: u.Bio, Address: u.Address, Phone: u.Phone,
		DateOfBirth: u.DateOfBirth, AvatarKey: u.AvatarKey, ResumeKey: u.ResumeKey,
		Skills: u.Skills, IsActive: u.IsActive, CreatedAt: u.CreatedAt,
	}
}
