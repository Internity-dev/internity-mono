// Package notification owns the in-app notification inbox. Every recipient
// gets their own row (see migration 000014) — there is no nullable-user_id
// "broadcast" row the way legacy had, so per-user read state is always correct.
package notification

import "time"

type Notification struct {
	ID        int64      `gorm:"column:id;primaryKey" json:"id"`
	UserID    string     `gorm:"column:user_id" json:"user_id"`
	Type      string     `gorm:"column:type" json:"type"`
	Title     string     `gorm:"column:title" json:"title"`
	Body      string     `gorm:"column:body" json:"body"`
	ReadAt    *time.Time `gorm:"column:read_at" json:"read_at"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (Notification) TableName() string { return "notifications" }
