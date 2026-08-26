package orgs

import (
	"net/http"
	"strconv"

	"internity/internal/httpx"
	"internity/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

var orgSortColumns = map[string]bool{"name": true, "created_at": true, "updated_at": true}

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Fail(c, httpx.BadPathParam("id", "must be a positive integer"))
		return 0, false
	}
	return id, true
}

func queryInt64(c *gin.Context, key string) *int64 {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

// --- Schools ---

type createSchoolRequest struct {
	Name    string  `json:"name" binding:"required,min=2,max=255"`
	Email   *string `json:"email" binding:"omitempty,email"`
	Phone   *string `json:"phone" binding:"omitempty,max=50"`
	Address *string `json:"address"`
	Website *string `json:"website" binding:"omitempty,url"`
}

func (h *Handler) ListSchools(c *gin.Context) {
	params := httpx.ParseListParams(c, "name", orgSortColumns)
	rows, total, err := h.svc.ListSchools(c.Request.Context(), middleware.CurrentUser(c), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) GetSchool(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	school, err := h.svc.GetSchool(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, school, "OK", nil)
}

func (h *Handler) CreateSchool(c *gin.Context) {
	var req createSchoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	school := &School{Name: req.Name, Email: req.Email, Phone: req.Phone, Address: req.Address, Website: req.Website}
	if err := h.svc.CreateSchool(c.Request.Context(), middleware.CurrentUser(c), school); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, school, "School created", nil)
}

func (h *Handler) UpdateSchool(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var patch SchoolPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	school, err := h.svc.UpdateSchool(c.Request.Context(), middleware.CurrentUser(c), id, patch)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, school, "School updated", nil)
}

func (h *Handler) DeleteSchool(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteSchool(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Departments ---

type createDepartmentRequest struct {
	SchoolID     int64   `json:"school_id" binding:"required"`
	Name         string  `json:"name" binding:"required,min=2,max=255"`
	Description  *string `json:"description"`
	StudyProgram *string `json:"study_program" binding:"omitempty,max=255"`
}

func (h *Handler) ListDepartments(c *gin.Context) {
	params := httpx.ParseListParams(c, "name", orgSortColumns)
	rows, total, err := h.svc.ListDepartments(c.Request.Context(), middleware.CurrentUser(c), queryInt64(c, "school_id"), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) GetDepartment(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	dept, err := h.svc.GetDepartment(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, dept, "OK", nil)
}

func (h *Handler) CreateDepartment(c *gin.Context) {
	var req createDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	dept := &Department{SchoolID: req.SchoolID, Name: req.Name, Description: req.Description, StudyProgram: req.StudyProgram}
	if err := h.svc.CreateDepartment(c.Request.Context(), middleware.CurrentUser(c), dept); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, dept, "Department created", nil)
}

func (h *Handler) UpdateDepartment(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var patch DepartmentPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	dept, err := h.svc.UpdateDepartment(c.Request.Context(), middleware.CurrentUser(c), id, patch)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, dept, "Department updated", nil)
}

func (h *Handler) DeleteDepartment(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteDepartment(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Courses ---

type createCourseRequest struct {
	DepartmentID int64   `json:"department_id" binding:"required"`
	Name         string  `json:"name" binding:"required,min=1,max=255"`
	Description  *string `json:"description"`
}

func (h *Handler) ListCourses(c *gin.Context) {
	params := httpx.ParseListParams(c, "name", orgSortColumns)
	rows, total, err := h.svc.ListCourses(c.Request.Context(), middleware.CurrentUser(c), queryInt64(c, "department_id"), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) GetCourse(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	course, err := h.svc.GetCourse(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, course, "OK", nil)
}

func (h *Handler) CreateCourse(c *gin.Context) {
	var req createCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	course := &Course{DepartmentID: req.DepartmentID, Name: req.Name, Description: req.Description}
	if err := h.svc.CreateCourse(c.Request.Context(), middleware.CurrentUser(c), course); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, course, "Course created", nil)
}

func (h *Handler) UpdateCourse(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var patch CoursePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	course, err := h.svc.UpdateCourse(c.Request.Context(), middleware.CurrentUser(c), id, patch)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, course, "Course updated", nil)
}

func (h *Handler) DeleteCourse(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteCourse(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Companies ---

type createCompanyRequest struct {
	DepartmentID  int64   `json:"department_id" binding:"required"`
	Name          string  `json:"name" binding:"required,min=2,max=255"`
	Category      *string `json:"category" binding:"omitempty,max=255"`
	City          *string `json:"city" binding:"omitempty,max=255"`
	State         *string `json:"state" binding:"omitempty,max=255"`
	Country       *string `json:"country" binding:"omitempty,max=255"`
	Address       *string `json:"address"`
	Email         *string `json:"email" binding:"omitempty,email"`
	Phone         *string `json:"phone" binding:"omitempty,max=50"`
	Website       *string `json:"website" binding:"omitempty,url"`
	ContactPerson *string `json:"contact_person" binding:"omitempty,max=255"`
}

func (h *Handler) ListCompanies(c *gin.Context) {
	params := httpx.ParseListParams(c, "name", orgSortColumns)
	rows, total, err := h.svc.ListCompanies(c.Request.Context(), middleware.CurrentUser(c), queryInt64(c, "department_id"), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) GetCompany(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	company, err := h.svc.GetCompany(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, company, "OK", nil)
}

func (h *Handler) CreateCompany(c *gin.Context) {
	var req createCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	company := &Company{
		DepartmentID: req.DepartmentID, Name: req.Name, Category: req.Category, City: req.City,
		State: req.State, Country: req.Country, Address: req.Address, Email: req.Email,
		Phone: req.Phone, Website: req.Website, ContactPerson: req.ContactPerson,
	}
	if err := h.svc.CreateCompany(c.Request.Context(), middleware.CurrentUser(c), company); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, company, "Company created", nil)
}

func (h *Handler) UpdateCompany(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var patch CompanyPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	company, err := h.svc.UpdateCompany(c.Request.Context(), middleware.CurrentUser(c), id, patch)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, company, "Company updated", nil)
}

func (h *Handler) DeleteCompany(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteCompany(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
