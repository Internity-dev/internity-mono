package orgs

import (
	"context"
	"errors"

	"internity/internal/httpx"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("orgs: record not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// applyList applies search (ILIKE name) + sort + pagination to a base query,
// shared by all four entities below since they all list the same shape.
func applyList(q *gorm.DB, params httpx.ListParams) *gorm.DB {
	if params.Search != "" {
		q = q.Where("name ILIKE ?", "%"+params.Search+"%")
	}
	return q.Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset())
}

// --- Schools ---

func (r *Repository) ListSchools(ctx context.Context, params httpx.ListParams) ([]School, int64, error) {
	var rows []School
	var total int64
	base := r.db.WithContext(ctx).Model(&School{})
	if params.Search != "" {
		base = base.Where("name ILIKE ?", "%"+params.Search+"%")
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := applyList(r.db.WithContext(ctx).Model(&School{}), params).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetSchool(ctx context.Context, id int64) (*School, error) {
	var s School
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &s, nil
}

func (r *Repository) CreateSchool(ctx context.Context, s *School) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *Repository) UpdateSchool(ctx context.Context, s *School) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *Repository) DeleteSchool(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&School{}, id).Error
}

// CountDepartmentsBySchool counts departments still pointing at this school —
// used by the service to pre-check a school delete and name the blocker
// instead of letting the departments.school_id FK RESTRICT fail generically.
func (r *Repository) CountDepartmentsBySchool(ctx context.Context, schoolID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Department{}).Where("school_id = ?", schoolID).Count(&count).Error
	return count, err
}

// --- Departments ---

func (r *Repository) ListDepartments(ctx context.Context, schoolID *int64, params httpx.ListParams) ([]Department, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if schoolID != nil {
			q = q.Where("school_id = ?", *schoolID)
		}
		return q
	}

	countQ := scope(r.db.WithContext(ctx).Model(&Department{}))
	if params.Search != "" {
		countQ = countQ.Where("name ILIKE ?", "%"+params.Search+"%")
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []Department
	if err := applyList(scope(r.db.WithContext(ctx).Model(&Department{})), params).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetDepartment(ctx context.Context, id int64) (*Department, error) {
	var d Department
	if err := r.db.WithContext(ctx).First(&d, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &d, nil
}

func (r *Repository) CreateDepartment(ctx context.Context, d *Department) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *Repository) UpdateDepartment(ctx context.Context, d *Department) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *Repository) DeleteDepartment(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Department{}, id).Error
}

// CountCoursesByDepartment and CountCompaniesByDepartment count a
// department's two direct child tables — used by the service to pre-check a
// department delete and name whichever is blocking it.
func (r *Repository) CountCoursesByDepartment(ctx context.Context, departmentID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Course{}).Where("department_id = ?", departmentID).Count(&count).Error
	return count, err
}

func (r *Repository) CountCompaniesByDepartment(ctx context.Context, departmentID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Company{}).Where("department_id = ?", departmentID).Count(&count).Error
	return count, err
}

// --- Courses ---

func (r *Repository) ListCourses(ctx context.Context, departmentID *int64, params httpx.ListParams) ([]Course, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if departmentID != nil {
			q = q.Where("department_id = ?", *departmentID)
		}
		return q
	}

	countQ := scope(r.db.WithContext(ctx).Model(&Course{}))
	if params.Search != "" {
		countQ = countQ.Where("name ILIKE ?", "%"+params.Search+"%")
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []Course
	if err := applyList(scope(r.db.WithContext(ctx).Model(&Course{})), params).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetCourse(ctx context.Context, id int64) (*Course, error) {
	var c Course
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &c, nil
}

func (r *Repository) CreateCourse(ctx context.Context, c *Course) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *Repository) UpdateCourse(ctx context.Context, c *Course) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *Repository) DeleteCourse(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Course{}, id).Error
}

// --- Companies ---

func (r *Repository) ListCompanies(ctx context.Context, departmentID *int64, params httpx.ListParams) ([]Company, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if departmentID != nil {
			q = q.Where("department_id = ?", *departmentID)
		}
		return q
	}

	countQ := scope(r.db.WithContext(ctx).Model(&Company{}))
	if params.Search != "" {
		countQ = countQ.Where("name ILIKE ?", "%"+params.Search+"%")
	}
	var total int64
	if err := countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []Company
	if err := applyList(scope(r.db.WithContext(ctx).Model(&Company{})), params).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) GetCompany(ctx context.Context, id int64) (*Company, error) {
	var c Company
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, translateNotFound(err)
	}
	return &c, nil
}

func (r *Repository) CreateCompany(ctx context.Context, c *Company) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *Repository) UpdateCompany(ctx context.Context, c *Company) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *Repository) DeleteCompany(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&Company{}, id).Error
}

// CountVacanciesByCompany counts a company's direct child table. Vacancies
// are owned by the vacancy module, not this one, so this queries the table
// name directly — same cross-table approach as ResolveCompanyScope below —
// rather than importing that module's repository just for a count.
func (r *Repository) CountVacanciesByCompany(ctx context.Context, companyID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("vacancies").Where("company_id = ?", companyID).Count(&count).Error
	return count, err
}

// CompanyScope is the resolved department/school for a company — used by
// other modules (vacancy) to scope-check without importing this module's
// repository directly; see the small adapter wired in cmd/api/main.go.
type CompanyScope struct {
	CompanyID    int64
	DepartmentID int64
	SchoolID     int64
}

func (r *Repository) ResolveCompanyScope(ctx context.Context, companyID int64) (*CompanyScope, error) {
	var scope CompanyScope
	err := r.db.WithContext(ctx).
		Table("companies").
		Select("companies.id AS company_id, companies.department_id AS department_id, departments.school_id AS school_id").
		Joins("JOIN departments ON departments.id = companies.department_id").
		Where("companies.id = ?", companyID).
		Take(&scope).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

func translateNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}
