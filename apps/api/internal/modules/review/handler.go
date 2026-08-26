package review

import (
	"net/http"
	"strconv"
	"time"

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

var monitorSortColumns = map[string]bool{"date": true, "created_at": true}
var questionSortColumns = map[string]bool{"sort_order": true, "question": true, "created_at": true}

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

// --- Monitors ---

func (h *Handler) ListMonitors(c *gin.Context) {
	var studentID *string
	if v := c.Query("student_id"); v != "" {
		studentID = &v
	}
	params := httpx.ParseListParams(c, "date", monitorSortColumns)
	rows, total, err := h.svc.ListMonitors(c.Request.Context(), middleware.CurrentUser(c), studentID, queryInt64(c, "company_id"), params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

type createMonitorRequest struct {
	StudentID     string  `json:"student_id" binding:"required,uuid"`
	CompanyID     int64   `json:"company_id" binding:"required"`
	Date          string  `json:"date" binding:"required,datetime=2006-01-02"`
	AttachmentKey *string `json:"attachment_key"`
	Notes         *string `json:"notes"`
	Suggest       *string `json:"suggest"`
	MatchRating   int     `json:"match_rating" binding:"min=1,max=4"`
}

func (h *Handler) CreateMonitor(c *gin.Context) {
	var req createMonitorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	date, _ := time.Parse("2006-01-02", req.Date)
	row := &Monitor{
		StudentID: req.StudentID, CompanyID: req.CompanyID, Date: date,
		AttachmentKey: req.AttachmentKey, Notes: req.Notes, Suggest: req.Suggest, MatchRating: req.MatchRating,
	}
	if err := h.svc.CreateMonitor(c.Request.Context(), middleware.CurrentUser(c), row); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "Monitoring visit logged", nil)
}

func (h *Handler) DeleteMonitor(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteMonitor(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Questions ---

func (h *Handler) ListQuestions(c *gin.Context) {
	schoolID, ok := requireQueryInt64(c, "school_id")
	if !ok {
		return
	}
	params := httpx.ParseListParams(c, "sort_order", questionSortColumns)
	rows, total, err := h.svc.ListQuestions(c.Request.Context(), middleware.CurrentUser(c), schoolID, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

type createQuestionRequest struct {
	SchoolID  int64  `json:"school_id" binding:"required"`
	Question  string `json:"question" binding:"required,min=1"`
	SortOrder int    `json:"sort_order"`
}

func (h *Handler) CreateQuestion(c *gin.Context) {
	var req createQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row := &Question{SchoolID: req.SchoolID, Question: req.Question, SortOrder: req.SortOrder}
	if err := h.svc.CreateQuestion(c.Request.Context(), middleware.CurrentUser(c), row); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "Question created", nil)
}

type updateQuestionRequest struct {
	Question  *string `json:"question"`
	SortOrder *int    `json:"sort_order"`
}

func (h *Handler) UpdateQuestion(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req updateQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.UpdateQuestion(c.Request.Context(), middleware.CurrentUser(c), id, req.Question, req.SortOrder)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Question updated", nil)
}

func (h *Handler) DeleteQuestion(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteQuestion(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Reviews ---

type createReviewRequest struct {
	RevieweeType      string  `json:"reviewee_type" binding:"required,oneof=user company"`
	RevieweeUserID    *string `json:"reviewee_user_id" binding:"omitempty,uuid"`
	RevieweeCompanyID *int64  `json:"reviewee_company_id"`
	QuestionID        *int64  `json:"question_id"`
	Title             *string `json:"title" binding:"omitempty,max=255"`
	Body              *string `json:"body"`
	Rating            int     `json:"rating" binding:"min=1,max=5"`
}

func (h *Handler) CreateReview(c *gin.Context) {
	var req createReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.CreateReview(c.Request.Context(), middleware.CurrentUser(c), CreateReviewInput{
		RevieweeType: RevieweeType(req.RevieweeType), RevieweeUserID: req.RevieweeUserID,
		RevieweeCompanyID: req.RevieweeCompanyID, QuestionID: req.QuestionID,
		Title: req.Title, Body: req.Body, Rating: req.Rating,
	})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "Review submitted", nil)
}

func (h *Handler) ListReviewsForUser(c *gin.Context) {
	userID := c.Param("userID")
	rows, err := h.svc.ListReviewsForUser(c.Request.Context(), middleware.CurrentUser(c), userID)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", nil)
}

func (h *Handler) ListReviewsForCompany(c *gin.Context) {
	companyID, ok := idParam(c)
	if !ok {
		return
	}
	rows, err := h.svc.ListReviewsForCompany(c.Request.Context(), companyID)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, gin.H{"reviews": rows, "average_rating": AverageRating(rows)}, "OK", nil)
}

func requireQueryInt64(c *gin.Context, key string) (int64, bool) {
	v, err := strconv.ParseInt(c.Query(key), 10, 64)
	if err != nil || v < 1 {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Missing or invalid "+key, httpx.ErrorDetail{Field: key, Issue: "required, positive integer"}))
		return 0, false
	}
	return v, true
}
