package identity

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes: no session cookie exists yet, so no auth/CSRF
// middleware applies here (see plan section 2.3 on why login/register are
// exempt from CSRF checks). rateLimit is applied to every route in this
// group — brute-force/enumeration protection on exactly the endpoints an
// attacker would hammer (login, register, forgot-password); refresh is
// included too since a stolen/duplicated refresh cookie retried in a loop
// is the same class of abuse.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup, rateLimit gin.HandlerFunc) {
	auth := rg.Group("/auth")
	auth.Use(rateLimit)
	auth.POST("/login", h.Login)
	auth.POST("/register", h.Register)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/forgot-password", h.ForgotPassword)
	auth.POST("/reset-password", h.ResetPassword)
}

// RegisterAuthenticatedRoutes must be attached under a group that already
// carries RequireAuth (+ RequireCSRF for the mutating one) — see server.go.
func (h *Handler) RegisterAuthenticatedRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/logout", h.Logout)
	auth.GET("/me", h.Me)

	rg.PUT("/change-profile", h.ChangeProfile)
	rg.PUT("/change-password", h.ChangePassword)
	rg.POST("/avatars", h.UploadAvatar)
	rg.POST("/invite-codes", h.CreateInviteCode)

	users := rg.Group("/users")
	users.GET("", h.ListUsers)
}
