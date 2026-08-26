package orgs

// Patch DTOs: every field is optional (pointer) so a PUT only touches the
// fields the caller actually sent, rather than requiring the full resource
// on every update.

type SchoolPatch struct {
	Name    *string `json:"name" binding:"omitempty,min=2,max=255"`
	Email   *string `json:"email" binding:"omitempty,email"`
	Phone   *string `json:"phone" binding:"omitempty,max=50"`
	Address *string `json:"address"`
	Website *string `json:"website" binding:"omitempty,url"`
}

func (p SchoolPatch) applyTo(s *School) {
	if p.Name != nil {
		s.Name = *p.Name
	}
	if p.Email != nil {
		s.Email = p.Email
	}
	if p.Phone != nil {
		s.Phone = p.Phone
	}
	if p.Address != nil {
		s.Address = p.Address
	}
	if p.Website != nil {
		s.Website = p.Website
	}
}

type DepartmentPatch struct {
	Name         *string `json:"name" binding:"omitempty,min=2,max=255"`
	Description  *string `json:"description"`
	StudyProgram *string `json:"study_program" binding:"omitempty,max=255"`
	IsActive     *bool   `json:"is_active"`
}

func (p DepartmentPatch) applyTo(d *Department) {
	if p.Name != nil {
		d.Name = *p.Name
	}
	if p.Description != nil {
		d.Description = p.Description
	}
	if p.StudyProgram != nil {
		d.StudyProgram = p.StudyProgram
	}
	if p.IsActive != nil {
		d.IsActive = *p.IsActive
	}
}

type CoursePatch struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

func (p CoursePatch) applyTo(c *Course) {
	if p.Name != nil {
		c.Name = *p.Name
	}
	if p.Description != nil {
		c.Description = p.Description
	}
	if p.IsActive != nil {
		c.IsActive = *p.IsActive
	}
}

type CompanyPatch struct {
	Name          *string `json:"name" binding:"omitempty,min=2,max=255"`
	Category      *string `json:"category" binding:"omitempty,max=255"`
	City          *string `json:"city" binding:"omitempty,max=255"`
	State         *string `json:"state" binding:"omitempty,max=255"`
	Country       *string `json:"country" binding:"omitempty,max=255"`
	Address       *string `json:"address"`
	Email         *string `json:"email" binding:"omitempty,email"`
	Phone         *string `json:"phone" binding:"omitempty,max=50"`
	Website       *string `json:"website" binding:"omitempty,url"`
	ContactPerson *string `json:"contact_person" binding:"omitempty,max=255"`
	IsActive      *bool   `json:"is_active"`
}

func (p CompanyPatch) applyTo(c *Company) {
	if p.Name != nil {
		c.Name = *p.Name
	}
	if p.Category != nil {
		c.Category = p.Category
	}
	if p.City != nil {
		c.City = p.City
	}
	if p.State != nil {
		c.State = p.State
	}
	if p.Country != nil {
		c.Country = p.Country
	}
	if p.Address != nil {
		c.Address = p.Address
	}
	if p.Email != nil {
		c.Email = p.Email
	}
	if p.Phone != nil {
		c.Phone = p.Phone
	}
	if p.Website != nil {
		c.Website = p.Website
	}
	if p.ContactPerson != nil {
		c.ContactPerson = p.ContactPerson
	}
	if p.IsActive != nil {
		c.IsActive = *p.IsActive
	}
}
