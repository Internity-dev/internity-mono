package internship

type PresenceStatusPatch struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description *string `json:"description"`
	Color       *string `json:"color" binding:"omitempty,max=20"`
	Icon        *string `json:"icon" binding:"omitempty,max=50"`
	IsActive    *bool   `json:"is_active"`
}

func (p PresenceStatusPatch) applyTo(row *PresenceStatus) {
	if p.Name != nil {
		row.Name = *p.Name
	}
	if p.Description != nil {
		row.Description = p.Description
	}
	if p.Color != nil {
		row.Color = p.Color
	}
	if p.Icon != nil {
		row.Icon = p.Icon
	}
	if p.IsActive != nil {
		row.IsActive = *p.IsActive
	}
}
