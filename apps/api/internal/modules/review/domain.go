// Package review owns coordinator site-visit monitoring logs, per-school
// review questionnaire templates (questions), and reviews themselves
// (mentor-rates-student or aggregate company ratings).
package review

import "time"

type Monitor struct {
	ID            int64     `gorm:"column:id;primaryKey" json:"id"`
	CoordinatorID string    `gorm:"column:coordinator_id" json:"coordinator_id"`
	StudentID     string    `gorm:"column:student_id" json:"student_id"`
	CompanyID     int64     `gorm:"column:company_id" json:"company_id"`
	Date          time.Time `gorm:"column:date" json:"date"`
	AttachmentKey *string   `gorm:"column:attachment_key" json:"attachment_key"`
	Notes         *string   `gorm:"column:notes" json:"notes"`
	Suggest       *string   `gorm:"column:suggest" json:"suggest"`
	MatchRating   int       `gorm:"column:match_rating" json:"match_rating"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Monitor) TableName() string { return "monitors" }

type Question struct {
	ID        int64     `gorm:"column:id;primaryKey" json:"id"`
	SchoolID  int64     `gorm:"column:school_id" json:"school_id"`
	Question  string    `gorm:"column:question" json:"question"`
	SortOrder int       `gorm:"column:sort_order" json:"sort_order"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Question) TableName() string { return "questions" }

type RevieweeType string

const (
	RevieweeUser    RevieweeType = "user"
	RevieweeCompany RevieweeType = "company"
)

// Review targets exactly one of RevieweeUserID / RevieweeCompanyID — a DB
// CHECK constraint (migration 000025) enforces this alongside the app-level
// validation in service.go.
type Review struct {
	ID                int64     `gorm:"column:id;primaryKey" json:"id"`
	ReviewerID        string    `gorm:"column:reviewer_id" json:"reviewer_id"`
	QuestionID        *int64    `gorm:"column:question_id" json:"question_id"`
	RevieweeUserID    *string   `gorm:"column:reviewee_user_id" json:"reviewee_user_id"`
	RevieweeCompanyID *int64    `gorm:"column:reviewee_company_id" json:"reviewee_company_id"`
	Title             *string   `gorm:"column:title" json:"title"`
	Body              *string   `gorm:"column:body" json:"body"`
	Rating            int       `gorm:"column:rating" json:"rating"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Review) TableName() string { return "reviews" }
