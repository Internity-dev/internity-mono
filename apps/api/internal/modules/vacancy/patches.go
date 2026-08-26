package vacancy

type VacancyPatch struct {
	Name        *string        `json:"name" binding:"omitempty,min=2,max=255"`
	Category    *string        `json:"category" binding:"omitempty,max=255"`
	Description *string        `json:"description"`
	Skills      *string        `json:"skills"`
	Slots       *int           `json:"slots" binding:"omitempty,min=1"`
	Status      *VacancyStatus `json:"status" binding:"omitempty,oneof=open closed"`
}

func (p VacancyPatch) applyTo(v *Vacancy) {
	if p.Name != nil {
		v.Name = *p.Name
	}
	if p.Category != nil {
		v.Category = p.Category
	}
	if p.Description != nil {
		v.Description = p.Description
	}
	if p.Skills != nil {
		v.Skills = p.Skills
	}
	if p.Slots != nil {
		v.Slots = *p.Slots
	}
	if p.Status != nil {
		v.Status = *p.Status
	}
}
