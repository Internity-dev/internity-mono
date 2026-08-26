package scoring

import "github.com/gin-gonic/gin"

// RegisterRoutes must be attached under a group that already carries
// RequireAuth. Every handler re-derives the actor and every write is
// scope-checked in service.go.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	scores := rg.Group("/scores")
	scores.GET("", h.ListScores)
	scores.POST("", h.CreateScore)
	scores.PUT("/:id", h.UpdateScore)
	scores.DELETE("/:id", h.DeleteScore)

	predicates := rg.Group("/score-predicates")
	predicates.GET("", h.ListScorePredicates)
	predicates.POST("", h.CreateScorePredicate)
	predicates.PUT("/:id", h.UpdateScorePredicate)
	predicates.DELETE("/:id", h.DeleteScorePredicate)

	rg.GET("/certificate", h.DownloadCertificate)
}
