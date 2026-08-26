package review

import "github.com/gin-gonic/gin"

// RegisterRoutes must be attached under a group that already carries
// RequireAuth. Every handler re-derives the actor and every write is
// scope-checked in service.go.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	monitors := rg.Group("/monitors")
	monitors.GET("", h.ListMonitors)
	monitors.POST("", h.CreateMonitor)
	monitors.DELETE("/:id", h.DeleteMonitor)

	questions := rg.Group("/questions")
	questions.GET("", h.ListQuestions)
	questions.POST("", h.CreateQuestion)
	questions.PUT("/:id", h.UpdateQuestion)
	questions.DELETE("/:id", h.DeleteQuestion)

	reviews := rg.Group("/reviews")
	reviews.POST("", h.CreateReview)
	reviews.GET("/users/:userID", h.ListReviewsForUser)
	reviews.GET("/companies/:id", h.ListReviewsForCompany)
}
