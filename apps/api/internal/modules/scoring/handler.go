package scoring

import (
	"fmt"
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

func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		httpx.Fail(c, httpx.BadPathParam("id", "must be a positive integer"))
		return 0, false
	}
	return id, true
}

func queryInt64Required(c *gin.Context, key string) (int64, bool) {
	v, err := strconv.ParseInt(c.Query(key), 10, 64)
	if err != nil || v < 1 {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Missing or invalid "+key, httpx.ErrorDetail{Field: key, Issue: "required, positive integer"}))
		return 0, false
	}
	return v, true
}

// --- Scores ---

func (h *Handler) ListScores(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = middleware.CurrentUser(c).ID
	}
	companyID, ok := queryInt64Required(c, "company_id")
	if !ok {
		return
	}
	rows, err := h.svc.ListScores(c.Request.Context(), middleware.CurrentUser(c), userID, companyID)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", nil)
}

type createScoreRequest struct {
	UserID    string `json:"user_id" binding:"required,uuid"`
	CompanyID int64  `json:"company_id" binding:"required"`
	Name      string `json:"name" binding:"required,min=1,max=255"`
	Score     int    `json:"score" binding:"min=0,max=100"`
	Type      string `json:"type" binding:"required,oneof=teknis non-teknis"`
}

func (h *Handler) CreateScore(c *gin.Context) {
	var req createScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.CreateScore(c.Request.Context(), middleware.CurrentUser(c), CreateScoreInput{
		UserID: req.UserID, CompanyID: req.CompanyID, Name: req.Name, Score: req.Score, Type: ScoreType(req.Type),
	})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "Score recorded", nil)
}

func (h *Handler) UpdateScore(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var patch ScorePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.UpdateScore(c.Request.Context(), middleware.CurrentUser(c), id, patch)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Score updated", nil)
}

func (h *Handler) DeleteScore(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteScore(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Score predicates ---

type createScorePredicateRequest struct {
	SchoolID    int64   `json:"school_id" binding:"required"`
	Name        string  `json:"name" binding:"required,min=1,max=50"`
	Description *string `json:"description"`
	Color       *string `json:"color" binding:"omitempty,max=20"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
}

func (h *Handler) ListScorePredicates(c *gin.Context) {
	schoolID, ok := queryInt64Required(c, "school_id")
	if !ok {
		return
	}
	rows, err := h.svc.ListScorePredicates(c.Request.Context(), middleware.CurrentUser(c), schoolID)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", nil)
}

func (h *Handler) CreateScorePredicate(c *gin.Context) {
	var req createScorePredicateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row := &ScorePredicate{
		SchoolID: req.SchoolID, Name: req.Name, Description: req.Description,
		Color: req.Color, Min: req.Min, Max: req.Max,
	}
	if err := h.svc.CreateScorePredicate(c.Request.Context(), middleware.CurrentUser(c), row); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "Score predicate created", nil)
}

func (h *Handler) UpdateScorePredicate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var patch ScorePredicatePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.UpdateScorePredicate(c.Request.Context(), middleware.CurrentUser(c), id, patch)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Score predicate updated", nil)
}

func (h *Handler) DeleteScorePredicate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteScorePredicate(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Certificate ---

func (h *Handler) DownloadCertificate(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = middleware.CurrentUser(c).ID
	}
	companyID, ok := queryInt64Required(c, "company_id")
	if !ok {
		return
	}
	result, err := h.svc.GenerateCertificate(c.Request.Context(), middleware.CurrentUser(c), userID, companyID)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	filename := fmt.Sprintf("%s.pdf", result.Certificate.CertificateNumber)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", result.PDF)
}
