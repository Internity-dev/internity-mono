// Package scoring owns scores, per-school score_predicates (letter-grade
// bands), and certificate generation.
package scoring

import "time"

type ScoreType string

const (
	ScoreTeknis    ScoreType = "teknis"
	ScoreNonTeknis ScoreType = "non-teknis"
)

type Score struct {
	ID        int64     `gorm:"column:id;primaryKey" json:"id"`
	UserID    string    `gorm:"column:user_id" json:"user_id"`
	CompanyID int64     `gorm:"column:company_id" json:"company_id"`
	Name      string    `gorm:"column:name" json:"name"`
	Score     int       `gorm:"column:score" json:"score"`
	Type      ScoreType `gorm:"column:type" json:"type"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Score) TableName() string { return "scores" }

type ScorePredicate struct {
	ID          int64     `gorm:"column:id;primaryKey" json:"id"`
	SchoolID    int64     `gorm:"column:school_id" json:"school_id"`
	Name        string    `gorm:"column:name" json:"name"`
	Description *string   `gorm:"column:description" json:"description"`
	Color       *string   `gorm:"column:color" json:"color"`
	Min         float64   `gorm:"column:min" json:"min"`
	Max         float64   `gorm:"column:max" json:"max"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (ScorePredicate) TableName() string { return "score_predicates" }

// ResolvePredicate is a pure function — given a score and a school's
// configured bands, find the matching letter grade. Returns "" if no band
// covers the score (a configuration gap for that school, not an error the
// caller should crash on).
func ResolvePredicate(score float64, predicates []ScorePredicate) string {
	for _, p := range predicates {
		if score >= p.Min && score <= p.Max {
			return p.Name
		}
	}
	return ""
}

// Average is a pure function over a set of scores — used both for the
// certificate's headline number and for resolving its predicate.
func Average(scores []Score) float64 {
	if len(scores) == 0 {
		return 0
	}
	var sum int
	for _, s := range scores {
		sum += s.Score
	}
	return float64(sum) / float64(len(scores))
}

type Certificate struct {
	ID                int64     `gorm:"column:id;primaryKey" json:"id"`
	UserID            string    `gorm:"column:user_id" json:"user_id"`
	DepartmentID      int64     `gorm:"column:department_id" json:"department_id"`
	CompanyID         int64     `gorm:"column:company_id" json:"company_id"`
	CertificateNumber string    `gorm:"column:certificate_number" json:"certificate_number"`
	FileKey           *string   `gorm:"column:file_key" json:"file_key"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"created_at"`
}

func (Certificate) TableName() string { return "certificates" }
