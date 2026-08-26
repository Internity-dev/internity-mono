// Package content owns News (school/department-scoped announcements) and
// FAQs. Both have a public read surface — News' published items and all
// FAQs are servable without a session (the landing page's /faq page and any
// logged-out visitor need this), while writes stay authenticated+scoped.
package content

import "time"

type NewsScopeType string

const (
	NewsScopeSchool     NewsScopeType = "school"
	NewsScopeDepartment NewsScopeType = "department"
)

type NewsStatus string

const (
	NewsDraft     NewsStatus = "draft"
	NewsPublished NewsStatus = "published"
)

type News struct {
	ID          int64         `gorm:"column:id;primaryKey" json:"id"`
	AuthorID    string        `gorm:"column:author_id" json:"author_id"`
	ScopeType   NewsScopeType `gorm:"column:scope_type" json:"scope_type"`
	ScopeID     int64         `gorm:"column:scope_id" json:"scope_id"`
	Title       string        `gorm:"column:title" json:"title"`
	Slug        string        `gorm:"column:slug" json:"slug"`
	Content     string        `gorm:"column:content" json:"content"`
	ImageKey    *string       `gorm:"column:image_key" json:"image_key"`
	Status      NewsStatus    `gorm:"column:status" json:"status"`
	PublishedAt *time.Time    `gorm:"column:published_at" json:"published_at"`
	CreatedAt   time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"column:updated_at" json:"updated_at"`
}

func (News) TableName() string { return "news" }

type FAQ struct {
	ID        int64     `gorm:"column:id;primaryKey" json:"id"`
	Question  string    `gorm:"column:question" json:"question"`
	Answer    string    `gorm:"column:answer" json:"answer"`
	SortOrder int       `gorm:"column:sort_order" json:"sort_order"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (FAQ) TableName() string { return "faqs" }
