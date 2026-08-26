package content

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

var newsSortColumns = map[string]bool{"title": true, "created_at": true, "published_at": true}

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Fail(c, httpx.BadPathParam("id", "must be a positive integer"))
		return 0, false
	}
	return id, true
}

// --- Public (unauthenticated) ---

func (h *Handler) ListPublicNews(c *gin.Context) {
	params := httpx.ParseListParams(c, "published_at", newsSortColumns)
	rows, total, err := h.svc.ListPublicNews(c.Request.Context(), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) GetPublicNewsBySlug(c *gin.Context) {
	row, err := h.svc.GetNewsBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "OK", nil)
}

func (h *Handler) ListPublicFAQs(c *gin.Context) {
	rows, err := h.svc.ListFAQs(c.Request.Context())
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", nil)
}

// --- Authenticated ---

func (h *Handler) ListManagedNews(c *gin.Context) {
	params := httpx.ParseListParams(c, "created_at", newsSortColumns)
	var scopeType *NewsScopeType
	if raw := c.Query("scope_type"); raw == string(NewsScopeSchool) || raw == string(NewsScopeDepartment) {
		st := NewsScopeType(raw)
		scopeType = &st
	}
	var scopeID *int64
	if raw := c.Query("scope_id"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			scopeID = &v
		}
	}
	rows, total, err := h.svc.ListNews(c.Request.Context(), middleware.CurrentUser(c), scopeType, scopeID, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

type createNewsRequest struct {
	ScopeType string  `json:"scope_type" binding:"required,oneof=school department"`
	ScopeID   int64   `json:"scope_id" binding:"required"`
	Title     string  `json:"title" binding:"required,min=2,max=255"`
	Content   string  `json:"content" binding:"required,min=1"`
	ImageKey  *string `json:"image_key"`
	Publish   bool    `json:"publish"`
}

func (h *Handler) CreateNews(c *gin.Context) {
	var req createNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.CreateNews(c.Request.Context(), middleware.CurrentUser(c), CreateNewsInput{
		ScopeType: NewsScopeType(req.ScopeType), ScopeID: req.ScopeID,
		Title: req.Title, Content: req.Content, ImageKey: req.ImageKey, Publish: req.Publish,
	})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "News created", nil)
}

type updateNewsRequest struct {
	Title    *string `json:"title" binding:"omitempty,min=2,max=255"`
	Content  *string `json:"content"`
	ImageKey *string `json:"image_key"`
	Publish  *bool   `json:"publish"`
}

func (h *Handler) UpdateNews(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req updateNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.UpdateNews(c.Request.Context(), middleware.CurrentUser(c), id, NewsPatch{
		Title: req.Title, Content: req.Content, ImageKey: req.ImageKey, Publish: req.Publish,
	})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "News updated", nil)
}

func (h *Handler) DeleteNews(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteNews(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type createFAQRequest struct {
	Question  string `json:"question" binding:"required,min=1"`
	Answer    string `json:"answer" binding:"required,min=1"`
	SortOrder int    `json:"sort_order"`
}

func (h *Handler) CreateFAQ(c *gin.Context) {
	var req createFAQRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row := &FAQ{Question: req.Question, Answer: req.Answer, SortOrder: req.SortOrder}
	if err := h.svc.CreateFAQ(c.Request.Context(), middleware.CurrentUser(c), row); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "FAQ created", nil)
}

type updateFAQRequest struct {
	Question  *string `json:"question"`
	Answer    *string `json:"answer"`
	SortOrder *int    `json:"sort_order"`
}

func (h *Handler) UpdateFAQ(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req updateFAQRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.UpdateFAQ(c.Request.Context(), middleware.CurrentUser(c), id, req.Question, req.Answer, req.SortOrder)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "FAQ updated", nil)
}

func (h *Handler) DeleteFAQ(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteFAQ(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
