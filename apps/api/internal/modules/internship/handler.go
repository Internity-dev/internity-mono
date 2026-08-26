package internship

import (
	"io"
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

var presenceSortColumns = map[string]bool{"date": true, "created_at": true}
var journalSortColumns = map[string]bool{"date": true, "created_at": true}
var presenceStatusSortColumns = map[string]bool{"name": true, "created_at": true}

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

// queryDate returns nil (not an error) when the param is absent or
// unparseable — callers that need it fall back to their own default range.
func queryDate(c *gin.Context, key string) *time.Time {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil
	}
	return &t
}

// --- InternDate ---

func (h *Handler) ListMyInternDates(c *gin.Context) {
	rows, err := h.svc.ListMyInternDates(c.Request.Context(), middleware.CurrentUser(c))
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", nil)
}

func (h *Handler) GetInternDate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	row, err := h.svc.GetInternDate(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "OK", nil)
}

type setDatesRequest struct {
	StartDate       string  `json:"start_date" binding:"required,datetime=2006-01-02"`
	EndDate         string  `json:"end_date" binding:"required,datetime=2006-01-02"`
	ExtendedUntil   *string `json:"extended_until" binding:"omitempty,datetime=2006-01-02"`
	ExpectedVersion int     `json:"expected_version" binding:"required"`
}

func (h *Handler) SetDates(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var req setDatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	start, _ := time.Parse("2006-01-02", req.StartDate)
	end, _ := time.Parse("2006-01-02", req.EndDate)
	var extended *time.Time
	if req.ExtendedUntil != nil {
		t, _ := time.Parse("2006-01-02", *req.ExtendedUntil)
		extended = &t
	}

	row, err := h.svc.SetDates(c.Request.Context(), middleware.CurrentUser(c), id, SetDatesInput{
		StartDate: start, EndDate: end, ExtendedUntil: extended, ExpectedVersion: req.ExpectedVersion,
	})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Dates updated", nil)
}

func (h *Handler) MarkCompleted(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	row, err := h.svc.MarkCompleted(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Placement marked completed", nil)
}

func (h *Handler) AttendanceSummary(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	monthStr := c.DefaultQuery("month", time.Now().Format("2006-01"))
	month, err := time.Parse("2006-01", monthStr)
	if err != nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Invalid month", httpx.ErrorDetail{Field: "month", Issue: "expected YYYY-MM"}))
		return
	}
	days, svcErr := h.svc.AttendanceSummary(c.Request.Context(), middleware.CurrentUser(c), id, month)
	if svcErr != nil {
		httpx.FailFromErr(c, svcErr)
		return
	}
	httpx.OK(c, http.StatusOK, days, "OK", nil)
}

// --- Presence statuses ---

type createPresenceStatusRequest struct {
	SchoolID    int64   `json:"school_id" binding:"required"`
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	Kind        string  `json:"kind" binding:"required,oneof=present permitted sick absent holiday"`
	Description *string `json:"description"`
	Color       *string `json:"color" binding:"omitempty,max=20"`
	Icon        *string `json:"icon" binding:"omitempty,max=50"`
}

func (h *Handler) ListPresenceStatuses(c *gin.Context) {
	schoolID, ok := queryInt64Required(c, "school_id")
	if !ok {
		return
	}
	params := httpx.ParseListParams(c, "name", presenceStatusSortColumns)
	rows, total, err := h.svc.ListPresenceStatuses(c.Request.Context(), middleware.CurrentUser(c), schoolID, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) CreatePresenceStatus(c *gin.Context) {
	var req createPresenceStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row := &PresenceStatus{
		SchoolID: req.SchoolID, Name: req.Name, Kind: PresenceStatusKind(req.Kind),
		Description: req.Description, Color: req.Color, Icon: req.Icon,
	}
	if err := h.svc.CreatePresenceStatus(c.Request.Context(), middleware.CurrentUser(c), row); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "Presence status created", nil)
}

func (h *Handler) UpdatePresenceStatus(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var patch PresenceStatusPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.UpdatePresenceStatus(c.Request.Context(), middleware.CurrentUser(c), id, patch)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Presence status updated", nil)
}

func (h *Handler) DeletePresenceStatus(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeletePresenceStatus(c.Request.Context(), middleware.CurrentUser(c), id); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Presence actions ---

const maxUploadMemory = 10 << 20 // 10MB, matches storage.MaxDocBytes ceiling

func (h *Handler) CheckIn(c *gin.Context) {
	companyID, ok := formInt64(c, "company_id")
	if !ok {
		return
	}
	lat := formFloatPtr(c, "lat")
	lng := formFloatPtr(c, "lng")

	photo, filename, err := readOptionalFile(c, "photo")
	if err != nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Could not read photo upload", httpx.ErrorDetail{Field: "photo", Issue: err.Error()}))
		return
	}

	row, svcErr := h.svc.CheckIn(c.Request.Context(), middleware.CurrentUser(c), CheckInInput{
		CompanyID: companyID, Photo: photo, Filename: filename, Lat: lat, Lng: lng,
	})
	if svcErr != nil {
		httpx.FailFromErr(c, svcErr)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "Checked in", nil)
}

type checkOutRequest struct {
	CompanyID int64 `json:"company_id" binding:"required"`
}

func (h *Handler) CheckOut(c *gin.Context) {
	var req checkOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	row, err := h.svc.CheckOut(c.Request.Context(), middleware.CurrentUser(c), req.CompanyID)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Checked out", nil)
}

func (h *Handler) FileExcuse(c *gin.Context) {
	companyID, ok := formInt64(c, "company_id")
	if !ok {
		return
	}
	dateStr := c.PostForm("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Invalid date", httpx.ErrorDetail{Field: "date", Issue: "expected YYYY-MM-DD"}))
		return
	}
	kind := c.PostForm("kind")
	description := c.PostForm("description")
	if description == "" {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Description is required", httpx.ErrorDetail{Field: "description", Issue: "required"}))
		return
	}

	attachment, filename, err := readOptionalFile(c, "attachment")
	if err != nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Could not read attachment upload", httpx.ErrorDetail{Field: "attachment", Issue: err.Error()}))
		return
	}

	row, svcErr := h.svc.FileExcuse(c.Request.Context(), middleware.CurrentUser(c), ExcuseInput{
		CompanyID: companyID, Date: date, Kind: PresenceStatusKind(kind), Description: description,
		Attachment: attachment, Filename: filename,
	})
	if svcErr != nil {
		httpx.FailFromErr(c, svcErr)
		return
	}
	httpx.OK(c, http.StatusCreated, row, "Excuse submitted", nil)
}

func (h *Handler) ListMyPresences(c *gin.Context) {
	companyID, ok := queryInt64Required(c, "company_id")
	if !ok {
		return
	}
	params := httpx.ParseListParams(c, "date", presenceSortColumns)
	rows, total, err := h.svc.ListMyPresences(c.Request.Context(), middleware.CurrentUser(c), companyID, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) PresenceStatusCounts(c *gin.Context) {
	from := queryDate(c, "from")
	to := queryDate(c, "to")
	counts, err := h.svc.PresenceStatusCounts(c.Request.Context(), middleware.CurrentUser(c), from, to)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, counts, "OK", nil)
}

func (h *Handler) ListPresencesForApproval(c *gin.Context) {
	companyID, ok := queryInt64Required(c, "company_id")
	if !ok {
		return
	}
	params := httpx.ParseListParams(c, "date", presenceSortColumns)
	rows, total, err := h.svc.ListPresencesForApproval(c.Request.Context(), middleware.CurrentUser(c), companyID, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) ApprovePresence(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	row, err := h.svc.ApprovePresence(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Presence approved", nil)
}

type bulkApproveRequest struct {
	CompanyID int64   `json:"company_id" binding:"required"`
	IDs       []int64 `json:"ids" binding:"required,min=1"`
}

func (h *Handler) BulkApprovePresences(c *gin.Context) {
	var req bulkApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	count, err := h.svc.BulkApprovePresences(c.Request.Context(), middleware.CurrentUser(c), req.CompanyID, req.IDs)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, gin.H{"approved_count": count}, "Presences approved", nil)
}

// --- Journal ---

type journalRequest struct {
	CompanyID   int64  `json:"company_id" binding:"required"`
	Date        string `json:"date" binding:"required,datetime=2006-01-02"`
	WorkType    string `json:"work_type" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"required,min=1"`
}

func (h *Handler) UpsertJournal(c *gin.Context) {
	var req journalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	date, _ := time.Parse("2006-01-02", req.Date)
	row, err := h.svc.UpsertJournal(c.Request.Context(), middleware.CurrentUser(c), JournalInput{
		CompanyID: req.CompanyID, Date: date, WorkType: req.WorkType, Description: req.Description,
	})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Journal saved", nil)
}

func (h *Handler) ListMyJournals(c *gin.Context) {
	companyID, ok := queryInt64Required(c, "company_id")
	if !ok {
		return
	}
	params := httpx.ParseListParams(c, "date", journalSortColumns)
	rows, total, err := h.svc.ListMyJournals(c.Request.Context(), middleware.CurrentUser(c), companyID, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) ListJournalsForApproval(c *gin.Context) {
	companyID, ok := queryInt64Required(c, "company_id")
	if !ok {
		return
	}
	params := httpx.ParseListParams(c, "date", journalSortColumns)
	rows, total, err := h.svc.ListJournalsForApproval(c.Request.Context(), middleware.CurrentUser(c), companyID, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) ApproveJournal(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	row, err := h.svc.ApproveJournal(c.Request.Context(), middleware.CurrentUser(c), id)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, row, "Journal approved", nil)
}

func (h *Handler) BulkApproveJournals(c *gin.Context) {
	var req bulkApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	count, err := h.svc.BulkApproveJournals(c.Request.Context(), middleware.CurrentUser(c), req.CompanyID, req.IDs)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, gin.H{"approved_count": count}, "Journals approved", nil)
}

// --- multipart helpers ---

func formInt64(c *gin.Context, key string) (int64, bool) {
	v, err := strconv.ParseInt(c.PostForm(key), 10, 64)
	if err != nil || v < 1 {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Missing or invalid "+key, httpx.ErrorDetail{Field: key, Issue: "required, positive integer"}))
		return 0, false
	}
	return v, true
}

func formFloatPtr(c *gin.Context, key string) *float64 {
	raw := c.PostForm(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil
	}
	return &v
}

func readOptionalFile(c *gin.Context, field string) ([]byte, string, error) {
	fileHeader, err := c.FormFile(field)
	if err != nil {
		return nil, "", nil // no file provided is fine — caller decides if it's required
	}
	f, err := fileHeader.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxUploadMemory))
	if err != nil {
		return nil, "", err
	}
	return data, fileHeader.Filename, nil
}
