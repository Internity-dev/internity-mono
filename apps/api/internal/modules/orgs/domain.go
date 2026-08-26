// Package orgs owns the School -> Department -> Course/Company hierarchy —
// the org structure a Coordinator/Mentor/Student's scope columns point into.
package orgs

import "time"

type School struct {
	ID        int64     `gorm:"column:id;primaryKey" json:"id"`
	Name      string    `gorm:"column:name" json:"name"`
	Email     *string   `gorm:"column:email" json:"email"`
	Phone     *string   `gorm:"column:phone" json:"phone"`
	Address   *string   `gorm:"column:address" json:"address"`
	Website   *string   `gorm:"column:website" json:"website"`
	LogoKey   *string   `gorm:"column:logo_key" json:"logo_key"`
	IsActive  bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (School) TableName() string { return "schools" }

type Department struct {
	ID           int64     `gorm:"column:id;primaryKey" json:"id"`
	SchoolID     int64     `gorm:"column:school_id" json:"school_id"`
	Name         string    `gorm:"column:name" json:"name"`
	Description  *string   `gorm:"column:description" json:"description"`
	StudyProgram *string   `gorm:"column:study_program" json:"study_program"`
	LogoKey      *string   `gorm:"column:logo_key" json:"logo_key"`
	IsActive     bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Department) TableName() string { return "departments" }

type Course struct {
	ID           int64     `gorm:"column:id;primaryKey" json:"id"`
	DepartmentID int64     `gorm:"column:department_id" json:"department_id"`
	Name         string    `gorm:"column:name" json:"name"`
	Description  *string   `gorm:"column:description" json:"description"`
	IsActive     bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Course) TableName() string { return "courses" }

type Company struct {
	ID            int64     `gorm:"column:id;primaryKey" json:"id"`
	DepartmentID  int64     `gorm:"column:department_id" json:"department_id"`
	Name          string    `gorm:"column:name" json:"name"`
	Category      *string   `gorm:"column:category" json:"category"`
	City          *string   `gorm:"column:city" json:"city"`
	State         *string   `gorm:"column:state" json:"state"`
	Country       *string   `gorm:"column:country" json:"country"`
	Address       *string   `gorm:"column:address" json:"address"`
	Email         *string   `gorm:"column:email" json:"email"`
	Phone         *string   `gorm:"column:phone" json:"phone"`
	Website       *string   `gorm:"column:website" json:"website"`
	LogoKey       *string   `gorm:"column:logo_key" json:"logo_key"`
	ContactPerson *string   `gorm:"column:contact_person" json:"contact_person"`
	IsActive      bool      `gorm:"column:is_active" json:"is_active"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Company) TableName() string { return "companies" }
