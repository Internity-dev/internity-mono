package scoring

type ScorePatch struct {
	Name  *string    `json:"name" binding:"omitempty,min=1,max=255"`
	Score *int       `json:"score" binding:"omitempty,min=0,max=100"`
	Type  *ScoreType `json:"type" binding:"omitempty,oneof=teknis non-teknis"`
}

func (p ScorePatch) applyTo(row *Score) {
	if p.Name != nil {
		row.Name = *p.Name
	}
	if p.Score != nil {
		row.Score = *p.Score
	}
	if p.Type != nil {
		row.Type = *p.Type
	}
}

type ScorePredicatePatch struct {
	Name        *string  `json:"name" binding:"omitempty,min=1,max=50"`
	Description *string  `json:"description"`
	Color       *string  `json:"color" binding:"omitempty,max=20"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
}

func (p ScorePredicatePatch) applyTo(row *ScorePredicate) {
	if p.Name != nil {
		row.Name = *p.Name
	}
	if p.Description != nil {
		row.Description = p.Description
	}
	if p.Color != nil {
		row.Color = p.Color
	}
	if p.Min != nil {
		row.Min = *p.Min
	}
	if p.Max != nil {
		row.Max = *p.Max
	}
}
