package notification

import "github.com/gin-gonic/gin"

// RegisterRoutes must be attached under a group that already carries RequireAuth.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	n := rg.Group("/notifications")
	n.GET("", h.List)
	n.PUT("/mark-as-read", h.MarkAllRead)
}
