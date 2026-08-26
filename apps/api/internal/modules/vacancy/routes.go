package vacancy

import "github.com/gin-gonic/gin"

// RegisterRoutes must be attached under a group that already carries
// RequireAuth. Every handler re-derives the actor and every write is
// scope-checked in service.go — see plan section 2.2.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	vacancies := rg.Group("/vacancies")
	vacancies.GET("", h.ListVacancies)
	vacancies.GET("/status-counts", h.VacancyStatusCounts)
	vacancies.GET("/:id", h.GetVacancy)
	vacancies.POST("", h.CreateVacancy)
	vacancies.PUT("/:id", h.UpdateVacancy)
	vacancies.DELETE("/:id", h.DeleteVacancy)
	vacancies.POST("/:id/save", h.SaveVacancy)
	vacancies.DELETE("/:id/save", h.UnsaveVacancy)
	vacancies.GET("/:id/appliances", h.ListVacancyAppliances)

	rg.GET("/saved-vacancies", h.ListSavedVacancies)

	appliances := rg.Group("/appliances")
	appliances.GET("", h.ListMyAppliances)
	appliances.GET("/status-counts", h.ApplianceStatusCounts)
	appliances.POST("", h.Apply)
	appliances.PUT("/:id/cancel", h.CancelAppliance)
	appliances.PUT("/:id/process", h.ProcessAppliance)
	appliances.PUT("/:id/accept", h.AcceptAppliance)
	appliances.PUT("/:id/reject", h.RejectAppliance)
}
