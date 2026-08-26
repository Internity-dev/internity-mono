package internship

import "github.com/gin-gonic/gin"

// RegisterRoutes must be attached under a group that already carries
// RequireAuth. Every handler re-derives the actor and every write is
// scope-checked in service.go.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	internships := rg.Group("/internships")
	internships.GET("/mine", h.ListMyInternDates)
	internships.GET("/:id", h.GetInternDate)
	internships.PUT("/:id/dates", h.SetDates)
	internships.PUT("/:id/complete", h.MarkCompleted)
	internships.GET("/:id/attendance-summary", h.AttendanceSummary)

	statuses := rg.Group("/presence-statuses")
	statuses.GET("", h.ListPresenceStatuses)
	statuses.POST("", h.CreatePresenceStatus)
	statuses.PUT("/:id", h.UpdatePresenceStatus)
	statuses.DELETE("/:id", h.DeletePresenceStatus)

	presences := rg.Group("/presences")
	presences.GET("", h.ListMyPresences)
	presences.GET("/status-counts", h.PresenceStatusCounts)
	presences.GET("/pending-approval", h.ListPresencesForApproval)
	presences.POST("/check-in", h.CheckIn)
	presences.POST("/check-out", h.CheckOut)
	presences.POST("/excuse", h.FileExcuse)
	presences.PUT("/:id/approve", h.ApprovePresence)
	presences.PUT("/bulk-approve", h.BulkApprovePresences)

	journals := rg.Group("/journals")
	journals.GET("", h.ListMyJournals)
	journals.GET("/pending-approval", h.ListJournalsForApproval)
	journals.POST("", h.UpsertJournal)
	journals.PUT("/:id/approve", h.ApproveJournal)
	journals.PUT("/bulk-approve", h.BulkApproveJournals)
}
