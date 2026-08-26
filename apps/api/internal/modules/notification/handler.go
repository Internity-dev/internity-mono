package notification

import (
	"net/http"

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

var sortColumns = map[string]bool{"created_at": true}

func (h *Handler) List(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	params := httpx.ParseListParams(c, "created_at", sortColumns)

	rows, total, err := h.svc.ListForUser(c.Request.Context(), actor.ID, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	unread, err := h.svc.UnreadCount(c.Request.Context(), actor.ID)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}

	httpx.OK(c, http.StatusOK, gin.H{"notifications": rows, "unread_count": unread}, "OK",
		&httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	actor := middleware.CurrentUser(c)
	if err := h.svc.MarkAllRead(c.Request.Context(), actor.ID); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, nil, "All notifications marked as read", nil)
}
