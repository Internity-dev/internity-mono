package vacancy

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

var vacancySortColumns = map[string]bool{"name": true, "created_at": true, "updated_at": true}
var applianceSortColumns = map[string]bool{"created_at": true, "updated_at": true, "status": true}

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

// --- Vacancies ---

type createVacancyRequest struct {
	CompanyID   int64   `json:"company_id" binding:"required"`
	Name        string  `json:"name" binding:"required,min=2,max=255"`
	Category    *string `json:"category" binding:"omitempty,max=255"`
	Description *string `json:"description"`
	Skills      *string `json:"skills"`
	Slots       int     `json:"slots" binding:"omitempty,min=1"`
}

func (h *Handler) ListVacancies(c *gin.Context) {
	params := httpx.ParseListParams(c, "created_at", vacancySortColumns)
	rows, total, err := h.svc.ListVacancies(c.Request.Context(), middleware.CurrentUser(c),
		queryInt64(c, "company_id"), queryInt64(c, "department_id"), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) GetVacancy(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	v, err := h.svc.GetVacancy(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, v, "OK", nil)
}

func (h *Handler) CreateVacancy(c *gin.Context) {
	var req createVacancyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	v := &Vacancy{
		CompanyID: req.CompanyID, Name: req.Name, Category: req.Category,
		Description: req.Description, Skills: req.Skills, Slots: req.Slots,
	}
	if err := h.svc.CreateVacancy(c.Request.Context(), middleware.CurrentUser(c), v); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, v, "Vacancy created", nil)
}

func (h *Handler) UpdateVacancy(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var patch VacancyPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	v, err := h.svc.UpdateVacancy(c.Request.Context(), middleware.CurrentUser(c), id, patch)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, v, "Vacancy updated", nil)
}

func (h *Handler) DeleteVacancy(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteVacancy(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Saved vacancies ---

func (h *Handler) SaveVacancy(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.SaveVacancy(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, nil, "Vacancy saved", nil)
}

func (h *Handler) UnsaveVacancy(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.UnsaveVacancy(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListSavedVacancies(c *gin.Context) {
	params := httpx.ParseListParams(c, "created_at", vacancySortColumns)
	rows, total, err := h.svc.ListSavedVacancies(c.Request.Context(), middleware.CurrentUser(c), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

// --- Appliances ---

type applyRequest struct {
	VacancyID int64   `json:"vacancy_id" binding:"required"`
	Message   *string `json:"message" binding:"omitempty,max=2000"`
}

func (h *Handler) Apply(c *gin.Context) {
	var req applyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	a, err := h.svc.Apply(c.Request.Context(), middleware.CurrentUser(c), req.VacancyID, req.Message)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, a, "Application submitted", nil)
}

func (h *Handler) ListMyAppliances(c *gin.Context) {
	params := httpx.ParseListParams(c, "created_at", applianceSortColumns)
	rows, total, err := h.svc.ListMyAppliances(c.Request.Context(), middleware.CurrentUser(c), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) ListVacancyAppliances(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	params := httpx.ParseListParams(c, "created_at", applianceSortColumns)
	rows, total, err := h.svc.ListVacancyAppliances(c.Request.Context(), middleware.CurrentUser(c), id, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) CancelAppliance(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	a, err := h.svc.Cancel(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, a, "Application canceled", nil)
}

func (h *Handler) ProcessAppliance(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	a, err := h.svc.Process(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, a, "Application marked as under review", nil)
}

func (h *Handler) AcceptAppliance(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	a, err := h.svc.Accept(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, a, "Application accepted", nil)
}

func (h *Handler) RejectAppliance(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	a, err := h.svc.Reject(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, a, "Application rejected", nil)
}

func (h *Handler) ApplianceStatusCounts(c *gin.Context) {
	counts, err := h.svc.ApplianceStatusCounts(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, counts, "OK", nil)
}

func (h *Handler) VacancyStatusCounts(c *gin.Context) {
	counts, err := h.svc.VacancyStatusCounts(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, counts, "OK", nil)
}
