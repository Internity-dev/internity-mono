package identity

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"internity/internal/httpx"

	"github.com/gin-gonic/gin"
)

const (
	CookieSession = "internity_session"
	CookieRefresh = "internity_refresh"
	CookieCSRF    = "internity_csrf"

	RefreshPath = "/api/v1/auth/refresh"

	// ContextUserKey is where middleware.RequireAuth stashes the authenticated
	// *User. Owned here (not by internal/middleware) so middleware can import
	// identity for the key+type without identity ever importing middleware back.
	ContextUserKey = "auth_user"
)

type Handler struct {
	svc          *Service
	cookieSecure bool
}

func NewHandler(svc *Service, cookieSecure bool) *Handler {
	return &Handler{svc: svc, cookieSecure: cookieSecure}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type registerRequest struct {
	Name                 string `json:"name" binding:"required,min=2,max=255"`
	Email                string `json:"email" binding:"required,email"`
	Password             string `json:"password" binding:"required,min=8,max=72"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required"`
	InviteCode           string `json:"invite_code" binding:"required"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token                string `json:"token" binding:"required"`
	Password             string `json:"password" binding:"required,min=8,max=72"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}

	h.setAuthCookies(c, result)
	httpx.OK(c, http.StatusOK, gin.H{"user": result.User.ToResponse()}, "Login successful", nil)
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}

	result, err := h.svc.Register(c.Request.Context(), RegisterInput{
		Name: req.Name, Email: req.Email, Password: req.Password,
		PasswordConfirmation: req.PasswordConfirmation, InviteCode: req.InviteCode,
		UserAgent: c.Request.UserAgent(), IP: c.ClientIP(),
	})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}

	h.setAuthCookies(c, result)
	httpx.OK(c, http.StatusCreated, gin.H{"user": result.User.ToResponse()}, "Registration successful", nil)
}

func (h *Handler) Refresh(c *gin.Context) {
	rawRefresh, err := c.Cookie(CookieRefresh)
	if err != nil || rawRefresh == "" {
		httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
		return
	}

	// Read whatever CSRF cookie the caller currently has (ignore the error —
	// an empty string here just falls back to minting a fresh one, same as
	// before). See Service.Refresh's comment for why this must be carried
	// forward instead of always rotating.
	existingCSRF, _ := c.Cookie(CookieCSRF)

	result, err := h.svc.Refresh(c.Request.Context(), rawRefresh, existingCSRF, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		h.clearAuthCookies(c)
		httpx.FailFromErr(c, err)
		return
	}

	h.setAuthCookies(c, result)
	httpx.OK(c, http.StatusOK, gin.H{"user": result.User.ToResponse()}, "Session refreshed", nil)
}

func (h *Handler) Logout(c *gin.Context) {
	rawAccess, _ := c.Cookie(CookieSession)
	rawRefresh, _ := c.Cookie(CookieRefresh)
	_ = h.svc.Logout(c.Request.Context(), rawAccess, rawRefresh)
	h.clearAuthCookies(c)
	httpx.OK(c, http.StatusOK, nil, "Logged out", nil)
}

func (h *Handler) Me(c *gin.Context) {
	user := currentUserFromContext(c)
	if user == nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
		return
	}
	httpx.OK(c, http.StatusOK, gin.H{"user": user.ToResponse()}, "OK", nil)
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, nil, "If that email is registered, a reset link has been sent", nil)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	if req.Password != req.PasswordConfirmation {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Password confirmation does not match",
			httpx.ErrorDetail{Field: "password_confirmation", Issue: "must match password"}))
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, nil, "Password reset. Please log in again.", nil)
}

var userSortColumns = map[string]bool{"name": true, "email": true, "created_at": true}

func queryInt64(c *gin.Context, key string) *int64 {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	return &v
}

func (h *Handler) ListUsers(c *gin.Context) {
	actor := currentUserFromContext(c)
	if actor == nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
		return
	}

	params := httpx.ParseListParams(c, "created_at", userSortColumns)
	filter := UserFilter{
		SchoolID:     queryInt64(c, "school_id"),
		DepartmentID: queryInt64(c, "department_id"),
		CompanyID:    queryInt64(c, "company_id"),
	}
	if raw := c.Query("role"); raw != "" {
		r := Role(raw)
		filter.Role = &r
	}

	rows, total, err := h.svc.ListUsers(c.Request.Context(), actor, filter, params)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}

	responses := make([]UserResponse, len(rows))
	for i, u := range rows {
		responses[i] = u.ToResponse()
	}
	httpx.OK(c, http.StatusOK, responses, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
}

type createInviteCodeRequest struct {
	CourseID  int64      `json:"course_id" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at" binding:"omitempty"`
}

func (h *Handler) CreateInviteCode(c *gin.Context) {
	var req createInviteCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	actor := currentUserFromContext(c)
	if actor == nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
		return
	}
	code, err := h.svc.CreateInviteCode(c.Request.Context(), actor, InviteCodeInput{CourseID: req.CourseID, ExpiresAt: req.ExpiresAt})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, code, "Invite code created", nil)
}

type createStaffAccountRequest struct {
	Name                 string `json:"name" binding:"required,min=2,max=255"`
	Email                string `json:"email" binding:"required,email"`
	Password             string `json:"password" binding:"required,min=8,max=72"`
	PasswordConfirmation string `json:"password_confirmation" binding:"required"`
	Role                 Role   `json:"role" binding:"required,oneof=coordinator mentor"`
	SchoolID             *int64 `json:"school_id" binding:"omitempty"`
	CompanyID            *int64 `json:"company_id" binding:"omitempty"`
}

func (h *Handler) CreateStaffAccount(c *gin.Context) {
	var req createStaffAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	actor := currentUserFromContext(c)
	if actor == nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
		return
	}
	user, err := h.svc.CreateStaffAccount(c.Request.Context(), actor, CreateStaffAccountInput{
		Name: req.Name, Email: req.Email, Password: req.Password,
		PasswordConfirmation: req.PasswordConfirmation, Role: req.Role,
		SchoolID: req.SchoolID, CompanyID: req.CompanyID,
	})
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, user.ToResponse(), "Account created", nil)
}

func (h *Handler) setAuthCookies(c *gin.Context, result *LoginResult) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieSession, result.AccessToken, secondsUntil(result.AccessExpiresAt), "/", "", h.cookieSecure, true)
	c.SetCookie(CookieRefresh, result.RefreshToken, secondsUntil(result.RefreshExpiresAt), RefreshPath, "", h.cookieSecure, true)
	// CSRF cookie is deliberately NOT httpOnly — the frontend must read it to
	// echo it back as X-CSRF-Token (double-submit pattern, see middleware/csrf.go).
	c.SetCookie(CookieCSRF, result.CSRFToken, secondsUntil(result.RefreshExpiresAt), "/", "", h.cookieSecure, false)
}

func (h *Handler) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CookieSession, "", -1, "/", "", h.cookieSecure, true)
	c.SetCookie(CookieRefresh, "", -1, RefreshPath, "", h.cookieSecure, true)
	c.SetCookie(CookieCSRF, "", -1, "/", "", h.cookieSecure, false)
}

func secondsUntil(t time.Time) int {
	d := time.Until(t)
	if d < 0 {
		return -1
	}
	return int(d.Seconds())
}

type changeProfileRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=255"`
	Bio         *string `json:"bio"`
	Address     *string `json:"address"`
	Phone       *string `json:"phone" binding:"omitempty,max=50"`
	Gender      *string `json:"gender" binding:"omitempty,oneof=male female"`
	DateOfBirth *string `json:"date_of_birth" binding:"omitempty,datetime=2006-01-02"`
	Skills      *string `json:"skills"`
}

func (h *Handler) ChangeProfile(c *gin.Context) {
	var req changeProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	actor := currentUserFromContext(c)
	if actor == nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
		return
	}

	in := ChangeProfileInput{Name: req.Name, Bio: req.Bio, Address: req.Address, Phone: req.Phone, Skills: req.Skills}
	if req.Gender != nil {
		g := Gender(*req.Gender)
		in.Gender = &g
	}
	if req.DateOfBirth != nil {
		t, _ := time.Parse("2006-01-02", *req.DateOfBirth)
		in.DateOfBirth = &t
	}

	user, err := h.svc.ChangeProfile(c.Request.Context(), actor.ID, in)
	if err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, gin.H{"user": user.ToResponse()}, "Profile updated", nil)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72"`
	Confirmation    string `json:"new_password_confirmation" binding:"required"`
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BindingError(err))
		return
	}
	if req.NewPassword != req.Confirmation {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "Password confirmation does not match",
			httpx.ErrorDetail{Field: "new_password_confirmation", Issue: "must match new_password"}))
		return
	}
	actor := currentUserFromContext(c)
	if actor == nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), actor.ID, req.CurrentPassword, req.NewPassword); err != nil {
		httpx.FailFromErr(c, err)
		return
	}
	h.clearAuthCookies(c)
	httpx.OK(c, http.StatusOK, nil, "Password changed. Please log in again.", nil)
}

const maxAvatarUploadBytes = 5 << 20 // matches storage.MaxImageBytes

func (h *Handler) UploadAvatar(c *gin.Context) {
	actor := currentUserFromContext(c)
	if actor == nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated"))
		return
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrValidation, "avatar file is required", httpx.ErrorDetail{Field: "avatar", Issue: "required"}))
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrBadRequest, "Could not read uploaded file"))
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxAvatarUploadBytes))
	if err != nil {
		httpx.Fail(c, httpx.NewError(httpx.ErrBadRequest, "Could not read uploaded file"))
		return
	}

	user, svcErr := h.svc.UploadAvatar(c.Request.Context(), actor.ID, data, fileHeader.Filename)
	if svcErr != nil {
		httpx.FailFromErr(c, svcErr)
		return
	}
	httpx.OK(c, http.StatusOK, gin.H{"user": user.ToResponse()}, "Avatar updated", nil)
}

// currentUserFromContext reads the *User middleware.RequireAuth already
// attached to the context under ContextUserKey.
func currentUserFromContext(c *gin.Context) *User {
	v, ok := c.Get(ContextUserKey)
	if !ok {
		return nil
	}
	u, ok := v.(*User)
	if !ok {
		return nil
	}
	return u
}
