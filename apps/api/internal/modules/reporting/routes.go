package reporting

import "github.com/gin-gonic/gin"

// RegisterRoutes must be attached under a group that already carries RequireAuth.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	exports := rg.Group("/exports")
	exports.GET("/students", h.ExportStudentRoster)
	exports.GET("/presence", h.ExportPresence)
}
