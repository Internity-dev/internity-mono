package vacancy

import (
	"context"
	"errors"

	"internity/internal/httpx"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("vacancy: record not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type VacancyFilter struct {
	CompanyID    *int64
	DepartmentID *int64
	Status       *VacancyStatus
}

func (r *Repository) ListVacancies(ctx context.Context, filter VacancyFilter, params httpx.ListParams) ([]Vacancy, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if filter.DepartmentID != nil {
			q = q.Joins("JOIN companies ON companies.id = vacancies.company_id").
				Where("companies.department_id = ?", *filter.DepartmentID)
		}
		if filter.CompanyID != nil {
			q = q.Where("vacancies.company_id = ?", *filter.CompanyID)
		}
		if filter.Status != nil {
			q = q.Where("vacancies.status = ?", *filter.Status)
		}
		return q
	}

	countQ := scope(r.db.WithContext(ctx).Model(&Vacancy{}))
	if params.Search != "" {
		countQ = countQ.Where("vacancies.name ILIKE ? OR vacancies.skills ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q := scope(r.db.WithContext(ctx).Model(&Vacancy{}))
	if params.Search != "" {
		q = q.Where("vacancies.name ILIKE ? OR vacancies.skills ILIKE ?", "%"+params.Search+"%", "%"+params.Search+"%")
	}
	var rows []Vacancy
	err := q.Order("vacancies." + params.Sort + " " + params.Order).
		Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetVacancy(ctx context.Context, id int64) (*Vacancy, error) {
	var v Vacancy
	if err := r.db.WithContext(ctx).First(&v, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &v, nil
}

func (r *Repository) CreateVacancy(ctx context.Context, v *Vacancy) error {
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *Repository) UpdateVacancy(ctx context.Context, v *Vacancy) error {
	return r.db.WithContext(ctx).Save(v).Error
}

func (r *Repository) DeleteVacancy(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Vacancy{}, id).Error
}

// --- Saved vacancies ---

func (r *Repository) SaveVacancy(ctx context.Context, userID string, vacancyID int64) error {
	return r.db.WithContext(ctx).Create(&SavedVacancy{UserID: userID, VacancyID: vacancyID}).Error
}

func (r *Repository) UnsaveVacancy(ctx context.Context, userID string, vacancyID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND vacancy_id = ?", userID, vacancyID).
		Delete(&SavedVacancy{}).Error
}

func (r *Repository) IsSaved(ctx context.Context, userID string, vacancyID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&SavedVacancy{}).
		Where("user_id = ? AND vacancy_id = ?", userID, vacancyID).Count(&count).Error
	return count > 0, err
}

func (r *Repository) ListSavedVacancies(ctx context.Context, userID string, params httpx.ListParams) ([]Vacancy, int64, error) {
	base := r.db.WithContext(ctx).Model(&Vacancy{}).
		Joins("JOIN saved_vacancies ON saved_vacancies.vacancy_id = vacancies.id").
		Where("saved_vacancies.user_id = ?", userID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []Vacancy
	err := r.db.WithContext(ctx).Model(&Vacancy{}).
		Joins("JOIN saved_vacancies ON saved_vacancies.vacancy_id = vacancies.id").
		Where("saved_vacancies.user_id = ?", userID).
		Order("vacancies." + params.Sort + " " + params.Order).
		Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// --- Appliances ---

func (r *Repository) CreateAppliance(ctx context.Context, a *Appliance) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *Repository) GetAppliance(ctx context.Context, id int64) (*Appliance, error) {
	var a Appliance
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &a, nil
}

func (r *Repository) UpdateAppliance(ctx context.Context, a *Appliance) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *Repository) ListAppliancesForUser(ctx context.Context, userID string, params httpx.ListParams) ([]Appliance, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Where("user_id = ?", userID)
		if params.Search != "" {
			q = q.Where("message ILIKE ?", "%"+params.Search+"%")
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Appliance{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Appliance
	err := scope(r.db.WithContext(ctx).Model(&Appliance{})).
		Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) ListAppliancesForVacancy(ctx context.Context, vacancyID int64, params httpx.ListParams) ([]Appliance, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Joins("JOIN users ON users.id = appliances.user_id").Where("appliances.vacancy_id = ?", vacancyID)
		if params.Search != "" {
			like := "%" + params.Search + "%"
			q = q.Where("users.name ILIKE ? OR users.nis ILIKE ? OR appliances.message ILIKE ?", like, like, like)
		}
		return q
	}
	var total int64
	if err := scope(r.db.WithContext(ctx).Model(&Appliance{})).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Appliance
	err := scope(r.db.WithContext(ctx).Model(&Appliance{})).
		Select("appliances.*, users.name AS user_name, users.nis AS user_nis").
		Order("appliances." + params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) CountAcceptedForVacancy(ctx context.Context, vacancyID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Appliance{}).
		Where("vacancy_id = ? AND status = ?", vacancyID, StatusAccepted).Count(&count).Error
	return count, err
}

// StatusCountsScope narrows the two dashboard status-breakdown queries below
// to one school (coordinator) or one company (mentor) instead of the whole
// platform. Zero value (both nil) means unscoped — admin only.
type StatusCountsScope struct {
	SchoolID  *int64
	CompanyID *int64
}

// CountAppliancesByStatus is a single GROUP BY query backing the overview
// dashboard's status-breakdown chart — every known ApplianceStatus is
// present in the result even at zero, so the caller never has to guess
// which keys exist. `status`/`Group` are qualified with the table name
// because scope's joins bring in other tables that could otherwise collide
// (vacancies has its own `status` column).
func (r *Repository) CountAppliancesByStatus(ctx context.Context, scope StatusCountsScope) (map[ApplianceStatus]int64, error) {
	var rows []struct {
		Status ApplianceStatus
		Count  int64
	}
	q := r.db.WithContext(ctx).Model(&Appliance{}).Select("appliances.status, count(*) as count")
	switch {
	case scope.CompanyID != nil:
		q = q.Joins("JOIN vacancies ON vacancies.id = appliances.vacancy_id").
			Where("vacancies.company_id = ?", *scope.CompanyID)
	case scope.SchoolID != nil:
		q = q.Joins("JOIN vacancies ON vacancies.id = appliances.vacancy_id").
			Joins("JOIN companies ON companies.id = vacancies.company_id").
			Joins("JOIN departments ON departments.id = companies.department_id").
			Where("departments.school_id = ?", *scope.SchoolID)
	}
	if err := q.Group("appliances.status").Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[ApplianceStatus]int64{
		StatusPending: 0, StatusProcessed: 0, StatusAccepted: 0, StatusRejected: 0, StatusCanceled: 0,
	}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}

// CountVacanciesByStatus mirrors CountAppliancesByStatus for the vacancy
// status-breakdown chart on the overview dashboard.
func (r *Repository) CountVacanciesByStatus(ctx context.Context, scope StatusCountsScope) (map[VacancyStatus]int64, error) {
	var rows []struct {
		Status VacancyStatus
		Count  int64
	}
	q := r.db.WithContext(ctx).Model(&Vacancy{}).Select("vacancies.status, count(*) as count")
	switch {
	case scope.CompanyID != nil:
		q = q.Where("vacancies.company_id = ?", *scope.CompanyID)
	case scope.SchoolID != nil:
		q = q.Joins("JOIN companies ON companies.id = vacancies.company_id").
			Joins("JOIN departments ON departments.id = companies.department_id").
			Where("departments.school_id = ?", *scope.SchoolID)
	}
	if err := q.Group("vacancies.status").Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[VacancyStatus]int64{VacancyOpen: 0, VacancyClosed: 0}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
