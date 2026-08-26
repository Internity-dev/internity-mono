package reporting

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

const xlsxContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func (h *Handler) ExportStudentRoster(c *gin.Context) {
	departmentID, err := strconv.ParseInt(c.Query("department_id"), 10, 64)
	if err != nil || departmentID < 1 {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "department_id is required", httpx.ErrorDetail{Field: "department_id", Issue: "required, positive integer"}))
		return
	}
	data, svcErr := h.svc.ExportStudentRoster(c.Request.Context(), middleware.CurrentUser(c), departmentID)
	if svcErr != nil {
		httpx.FailFromErr(c, svcErr)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="students.xlsx"`)
	c.Data(http.StatusOK, xlsxContentType, data)
}

func (h *Handler) ExportPresence(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = middleware.CurrentUser(c).ID
	}
	companyID, err := strconv.ParseInt(c.Query("company_id"), 10, 64)
	if err != nil || companyID < 1 {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "company_id is required", httpx.ErrorDetail{Field: "company_id", Issue: "required, positive integer"}))
		return
	}
	data, svcErr := h.svc.ExportPresence(c.Request.Context(), middleware.CurrentUser(c), userID, companyID)
	if svcErr != nil {
		httpx.FailFromErr(c, svcErr)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="presence-%d.xlsx"`, companyID))
	c.Data(http.StatusOK, xlsxContentType, data)
}
