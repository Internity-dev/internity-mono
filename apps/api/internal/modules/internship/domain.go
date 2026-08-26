// Package internship owns the placement lifecycle once an appliance is
// accepted: intern_dates, presence_statuses, presences, and journals.
// Presence/journal rows are created on-the-fly (only when the student
// actually acts), never bulk pre-materialized — see plan section 2.5.
package internship

import "time"

type InternDateStatus string

const (
	StatusScheduled InternDateStatus = "scheduled"
	StatusCompleted InternDateStatus = "completed"
)

// InternDate mirrors the `intern_dates` table (migration 000013). A freshly
// accepted appliance creates one of these with both dates nil — "accepted,
// unscheduled" — until the student/staff sets a start/end date. Whether a
// scheduled placement is "currently active" is a derived property (today
// within [start,end]), not a stored state — see IsActiveOn below; only the
// explicit scheduled->completed transition is persisted.
type InternDate struct {
	ID            int64            `gorm:"column:id;primaryKey" json:"id"`
	UserID        string           `gorm:"column:user_id" json:"user_id"`
	CompanyID     int64            `gorm:"column:company_id" json:"company_id"`
	ApplianceID   int64            `gorm:"column:appliance_id" json:"appliance_id"`
	StartDate     *time.Time       `gorm:"column:start_date" json:"start_date"`
	EndDate       *time.Time       `gorm:"column:end_date" json:"end_date"`
	ExtendedUntil *time.Time       `gorm:"column:extended_until" json:"extended_until"`
	Status        InternDateStatus `gorm:"column:status" json:"status"`
	Version       int              `gorm:"column:version" json:"version"`
	CreatedAt     time.Time        `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time        `gorm:"column:updated_at" json:"updated_at"`
}

func (InternDate) TableName() string { return "intern_dates" }

// EffectiveEndDate is end_date, pushed out by any extension.
func (i InternDate) EffectiveEndDate() *time.Time {
	if i.ExtendedUntil != nil {
		return i.ExtendedUntil
	}
	return i.EndDate
}

// IsActiveOn reports whether `day` falls within the placement's scheduled
// range — computed at read/write time rather than stored, so there is no
// scheduler job needed to flip a status when a range starts/ends.
func (i InternDate) IsActiveOn(day time.Time) bool {
	if i.StartDate == nil || i.EffectiveEndDate() == nil {
		return false
	}
	d := truncateToDate(day)
	return !d.Before(truncateToDate(*i.StartDate)) && !d.After(truncateToDate(*i.EffectiveEndDate()))
}

func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// --- Presence statuses ---

type PresenceStatusKind string

const (
	KindPresent   PresenceStatusKind = "present"
	KindPermitted PresenceStatusKind = "permitted"
	KindSick      PresenceStatusKind = "sick"
	KindAbsent    PresenceStatusKind = "absent"
	KindHoliday   PresenceStatusKind = "holiday"
)

type PresenceStatus struct {
	ID          int64              `gorm:"column:id;primaryKey" json:"id"`
	SchoolID    int64              `gorm:"column:school_id" json:"school_id"`
	Name        string             `gorm:"column:name" json:"name"`
	Kind        PresenceStatusKind `gorm:"column:kind" json:"kind"`
	Description *string            `gorm:"column:description" json:"description"`
	Color       *string            `gorm:"column:color" json:"color"`
	Icon        *string            `gorm:"column:icon" json:"icon"`
	IsActive    bool               `gorm:"column:is_active" json:"is_active"`
	CreatedAt   time.Time          `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time          `gorm:"column:updated_at" json:"updated_at"`
}

func (PresenceStatus) TableName() string { return "presence_statuses" }

// --- Presence ---

type Presence struct {
	ID     int64  `gorm:"column:id;primaryKey" json:"id"`
	UserID string `gorm:"column:user_id" json:"user_id"`
	// UserName/UserNIS are only ever populated by ListPresencesForApproval's
	// join against users (the `->` tag makes them read-only, so Save/Create
	// never try to write these non-existent columns back to `presences`) —
	// every other query path leaves them at their zero value.
	UserName         string     `gorm:"column:user_name;->" json:"user_name,omitempty"`
	UserNIS          *string    `gorm:"column:user_nis;->" json:"user_nis,omitempty"`
	CompanyID        int64      `gorm:"column:company_id" json:"company_id"`
	PresenceStatusID int64      `gorm:"column:presence_status_id" json:"presence_status_id"`
	Date             time.Time  `gorm:"column:date" json:"date"`
	CheckInAt        *time.Time `gorm:"column:check_in_at" json:"check_in_at"`
	CheckOutAt       *time.Time `gorm:"column:check_out_at" json:"check_out_at"`
	CheckInLat       *float64   `gorm:"column:check_in_lat" json:"check_in_lat"`
	CheckInLng       *float64   `gorm:"column:check_in_lng" json:"check_in_lng"`
	AttachmentKey    *string    `gorm:"column:attachment_key" json:"attachment_key"`
	IsApproved       bool       `gorm:"column:is_approved" json:"is_approved"`
	Description      *string    `gorm:"column:description" json:"description"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (Presence) TableName() string { return "presences" }

// --- Journal ---

type Journal struct {
	ID     int64  `gorm:"column:id;primaryKey" json:"id"`
	UserID string `gorm:"column:user_id" json:"user_id"`
	// UserName/UserNIS are only ever populated by ListJournalsForApproval's
	// join against users (the `->` tag makes them read-only, so Save/Create
	// never try to write these non-existent columns back to `journals`) —
	// every other query path leaves them at their zero value.
	UserName    string    `gorm:"column:user_name;->" json:"user_name,omitempty"`
	UserNIS     *string   `gorm:"column:user_nis;->" json:"user_nis,omitempty"`
	CompanyID   int64     `gorm:"column:company_id" json:"company_id"`
	Date        time.Time `gorm:"column:date" json:"date"`
	WorkType    *string   `gorm:"column:work_type" json:"work_type"`
	Description *string   `gorm:"column:description" json:"description"`
	IsApproved  bool      `gorm:"column:is_approved" json:"is_approved"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Journal) TableName() string { return "journals" }

func (j Journal) Filled() bool {
	return j.WorkType != nil && *j.WorkType != "" && j.Description != nil && *j.Description != ""
}

// --- Attendance summary read model ---

type DayStatus string

const (
	DayReported   DayStatus = "reported"
	DayMissing    DayStatus = "missing"
	DayUpcoming   DayStatus = "upcoming"
	DayOutOfRange DayStatus = "outside_range"
)

type AttendanceDay struct {
	Date     time.Time `json:"date"`
	Status   DayStatus `json:"status"`
	Presence *Presence `json:"presence,omitempty"`
}
