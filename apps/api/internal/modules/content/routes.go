package content

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes: no session required — the landing page and any
// logged-out visitor read published news + all FAQs from here.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/news", h.ListPublicNews)
	rg.GET("/news/:slug", h.GetPublicNewsBySlug)
	rg.GET("/faqs", h.ListPublicFAQs)
}

// RegisterAuthenticatedRoutes: writes only — every handler re-derives the
// actor and every mutation is scope-checked in service.go.
func (h *Handler) RegisterAuthenticatedRoutes(rg *gin.RouterGroup) {
	news := rg.Group("/news")
	news.GET("/manage", h.ListManagedNews)
	news.POST("", h.CreateNews)
	news.PUT("/:id", h.UpdateNews)
	news.DELETE("/:id", h.DeleteNews)

	faqs := rg.Group("/faqs")
	faqs.POST("", h.CreateFAQ)
	faqs.PUT("/:id", h.UpdateFAQ)
	faqs.DELETE("/:id", h.DeleteFAQ)
}
