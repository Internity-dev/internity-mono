// Package vacancy owns vacancies, saved (bookmarked) vacancies, and the
// Appliance state machine — the core "apply -> review -> accept/reject"
// workflow this whole system is built around.
package vacancy

import (
	"fmt"
	"time"
)

type VacancyStatus string

const (
	VacancyOpen   VacancyStatus = "open"
	VacancyClosed VacancyStatus = "closed"
)

type Vacancy struct {
	ID          int64         `gorm:"column:id;primaryKey" json:"id"`
	CompanyID   int64         `gorm:"column:company_id" json:"company_id"`
	Name        string        `gorm:"column:name" json:"name"`
	Category    *string       `gorm:"column:category" json:"category"`
	Description *string       `gorm:"column:description" json:"description"`
	Skills      *string       `gorm:"column:skills" json:"skills"`
	Slots       int           `gorm:"column:slots" json:"slots"`
	Status      VacancyStatus `gorm:"column:status" json:"status"`
	CreatedAt   time.Time     `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time     `gorm:"column:updated_at" json:"updated_at"`
}

func (Vacancy) TableName() string { return "vacancies" }

type SavedVacancy struct {
	ID        int64     `gorm:"column:id;primaryKey" json:"id"`
	UserID    string    `gorm:"column:user_id" json:"user_id"`
	VacancyID int64     `gorm:"column:vacancy_id" json:"vacancy_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (SavedVacancy) TableName() string { return "saved_vacancies" }

type ApplianceStatus string

const (
	StatusPending   ApplianceStatus = "pending"
	StatusProcessed ApplianceStatus = "processed"
	StatusAccepted  ApplianceStatus = "accepted"
	StatusRejected  ApplianceStatus = "rejected"
	StatusCanceled  ApplianceStatus = "canceled"
)

// allowedTransitions is the whole state machine in one place: pending can
// move anywhere, processed (staff "under review") can only resolve to
// accepted/rejected, and accepted/rejected/canceled are terminal — no
// outgoing edges, by omission from this map (see plan section 2.4 for why
// "accepted" being terminal is what keeps Reject from needing any cleanup logic).
var allowedTransitions = map[ApplianceStatus][]ApplianceStatus{
	StatusPending:   {StatusProcessed, StatusAccepted, StatusRejected, StatusCanceled},
	StatusProcessed: {StatusAccepted, StatusRejected, StatusCanceled},
}

type Appliance struct {
	ID        int64           `gorm:"column:id;primaryKey" json:"id"`
	UserID    string          `gorm:"column:user_id" json:"user_id"`
	VacancyID int64           `gorm:"column:vacancy_id" json:"vacancy_id"`
	Status    ApplianceStatus `gorm:"column:status" json:"status"`
	Message   *string         `gorm:"column:message" json:"message"`
	CreatedAt time.Time       `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at" json:"updated_at"`
}

func (Appliance) TableName() string { return "appliances" }

type ErrInvalidTransition struct{ From, To ApplianceStatus }

func (e ErrInvalidTransition) Error() string {
	return fmt.Sprintf("cannot transition appliance from %q to %q", e.From, e.To)
}

func (a Appliance) CanTransitionTo(target ApplianceStatus) error {
	for _, allowed := range allowedTransitions[a.Status] {
		if allowed == target {
			return nil
		}
	}
	return ErrInvalidTransition{From: a.Status, To: target}
}

func (a Appliance) IsTerminal() bool {
	_, hasOutgoing := allowedTransitions[a.Status]
	return !hasOutgoing
}
