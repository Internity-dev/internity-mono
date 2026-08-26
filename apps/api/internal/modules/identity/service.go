package identity

import (
	"context"
	"errors"
	"time"

	"internity/internal/httpx"
	"internity/internal/platform/storage"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Mailer is the outbound-email port. NoopMailer (below) is the only
// implementation until a real SMTP/API provider is wired up — see README's
// "Future Improvements" / plan section 10 for why that's deliberately deferred.
type Mailer interface {
	SendPasswordReset(ctx context.Context, toEmail, rawToken string) error
}

type NoopMailer struct {
	Log func(event string, fields map[string]any)
}

func (m NoopMailer) SendPasswordReset(_ context.Context, toEmail, rawToken string) error {
	if m.Log != nil {
		m.Log("password_reset_issued", map[string]any{"email": toEmail, "token": rawToken})
	}
	return nil
}

type Config struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

func DefaultConfig() Config {
	return Config{AccessTTL: 15 * time.Minute, RefreshTTL: 7 * 24 * time.Hour}
}

// CompanyScopeResolver lets a coordinator's mentor lookups (ListUsers) and a
// coordinator's mentor-account creation (CreateStaffAccount) verify a
// company belongs to their own school, without identity importing orgs'
// repository directly — same shape and same cross-module pattern already
// used by vacancy.Service (see the companyScopeAdapter wired in
// cmd/api/main.go, reused here rather than duplicated).
type CompanyScopeResolver interface {
	ResolveCompanyScope(ctx context.Context, companyID int64) (schoolID, departmentID int64, err error)
}

type Service struct {
	repo      Repository
	mailer    Mailer
	cfg       Config
	storage   *storage.Client
	companies CompanyScopeResolver
}

func NewService(repo Repository, mailer Mailer, cfg Config, storageClient *storage.Client, companies CompanyScopeResolver) *Service {
	return &Service{repo: repo, mailer: mailer, cfg: cfg, storage: storageClient, companies: companies}
}

// LoginResult carries everything the HTTP handler needs to set the three
// cookies (access session, refresh session, CSRF) — cookie mechanics
// themselves belong to handler.go, not here.
type LoginResult struct {
	User             *User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	CSRFToken        string
}

type RegisterInput struct {
	Name                 string
	Email                string
	Password             string
	PasswordConfirmation string
	InviteCode           string
	UserAgent            string
	IP                   string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (*LoginResult, error) {
	if in.Password != in.PasswordConfirmation {
		return nil, httpx.NewError(httpx.ErrValidation, "Password confirmation does not match",
			httpx.ErrorDetail{Field: "password_confirmation", Issue: "must match password"})
	}

	code, err := s.repo.FindActiveInviteCode(ctx, in.InviteCode)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NewError(httpx.ErrValidation, "Invalid invite code",
			httpx.ErrorDetail{Field: "invite_code", Issue: "not found"})
	}
	if err != nil {
		return nil, err
	}
	if code.Expired(time.Now()) {
		return nil, httpx.NewError(httpx.ErrValidation, "This invite code has expired",
			httpx.ErrorDetail{Field: "invite_code", Issue: "expired"})
	}

	taken, err := s.repo.EmailTaken(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, httpx.NewError(httpx.ErrConflict, "An account with this email already exists",
			httpx.ErrorDetail{Field: "email", Issue: "already registered"})
	}

	scope, err := s.repo.ResolveCourseScope(ctx, code.CourseID)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.NewString(),
		Role:         RoleStudent,
		SchoolID:     &scope.SchoolID,
		DepartmentID: &scope.DepartmentID,
		CourseID:     &scope.CourseID,
		Name:         in.Name,
		Email:        in.Email,
		PasswordHash: string(hash),
		IsActive:     true,
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return s.issueSession(ctx, user, uuid.NewString(), in.UserAgent, in.IP)
}

func (s *Service) Login(ctx context.Context, email, password, userAgent, ip string) (*LoginResult, error) {
	invalidCreds := httpx.NewError(httpx.ErrUnauthenticated, "Invalid email or password")

	user, err := s.repo.FindUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		// Still run bcrypt against a dummy hash so a missing-vs-wrong-password
		// response doesn't leak via timing (a cheap, standard mitigation).
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$C6GBY/FSuiCn6nWbhCSRhOaOQip1HRHdA/rKa55vG1TNL//DdpmOu"), []byte(password))
		return nil, invalidCreds
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, invalidCreds
	}
	if !user.IsActive {
		return nil, httpx.NewError(httpx.ErrForbidden, "This account has been deactivated. Contact your administrator.")
	}

	now := time.Now()
	user.LastLoginAt = &now
	user.LastLoginIP = &ip
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return s.issueSession(ctx, user, uuid.NewString(), userAgent, ip)
}

// Refresh rotates the refresh token: the presented token is revoked and a new
// access+refresh pair is issued in the same family. Presenting an
// already-revoked token (replay of a stolen/duplicated refresh cookie) is
// treated as theft evidence and revokes every session in that family,
// forcing a full re-login everywhere — see plan section 2.3.
func (s *Service) Refresh(ctx context.Context, rawRefreshToken, userAgent, ip string) (*LoginResult, error) {
	unauthenticated := httpx.NewError(httpx.ErrUnauthenticated, "Session expired, please log in again")

	hash := hashToken(rawRefreshToken)
	session, err := s.repo.FindSessionByHashAnyState(ctx, hash)
	if errors.Is(err, ErrNotFound) {
		return nil, unauthenticated
	}
	if err != nil {
		return nil, err
	}
	if session.Kind != SessionRefresh {
		return nil, unauthenticated
	}
	if session.RevokedAt != nil {
		_ = s.repo.RevokeSessionFamily(ctx, session.FamilyID)
		return nil, unauthenticated
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, unauthenticated
	}

	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, httpx.NewError(httpx.ErrForbidden, "This account has been deactivated. Contact your administrator.")
	}

	if err := s.repo.RevokeSession(ctx, session.ID); err != nil {
		return nil, err
	}
	return s.issueSession(ctx, user, session.FamilyID, userAgent, ip)
}

func (s *Service) Logout(ctx context.Context, rawAccessToken, rawRefreshToken string) error {
	if rawAccessToken != "" {
		if sess, err := s.repo.FindSessionByHashAnyState(ctx, hashToken(rawAccessToken)); err == nil {
			_ = s.repo.RevokeSession(ctx, sess.ID)
		}
	}
	if rawRefreshToken != "" {
		if sess, err := s.repo.FindSessionByHashAnyState(ctx, hashToken(rawRefreshToken)); err == nil {
			_ = s.repo.RevokeSession(ctx, sess.ID)
		}
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (*User, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated")
	}
	return user, err
}

// ForgotPassword always returns nil (success) whether or not the email
// exists — the alternative leaks which emails are registered.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.repo.FindUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	raw, err := newOpaqueToken()
	if err != nil {
		return err
	}
	token := &PasswordResetToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: hashToken(raw),
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := s.repo.CreatePasswordResetToken(ctx, token); err != nil {
		return err
	}
	return s.mailer.SendPasswordReset(ctx, user.Email, raw)
}

func (s *Service) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	invalid := httpx.NewError(httpx.ErrValidation, "This reset link is invalid or has expired",
		httpx.ErrorDetail{Field: "token", Issue: "invalid or expired"})

	rec, err := s.repo.FindActivePasswordResetToken(ctx, hashToken(rawToken))
	if errors.Is(err, ErrNotFound) {
		return invalid
	}
	if err != nil {
		return err
	}

	user, err := s.repo.FindUserByID(ctx, rec.UserID)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return err
	}
	if err := s.repo.MarkPasswordResetTokenUsed(ctx, rec.ID); err != nil {
		return err
	}
	// Force re-login on every device — a leaked password means every
	// existing session is suspect, not just the one used to reset it.
	return s.repo.RevokeAllUserSessions(ctx, user.ID)
}

type InviteCodeInput struct {
	CourseID  int64
	ExpiresAt *time.Time
}

// CreateInviteCode is how a student registration code (Register/InviteCode
// above) comes to exist — issued by an admin (any course) or a coordinator
// (only for a course within their own school).
func (s *Service) CreateInviteCode(ctx context.Context, actor *User, in InviteCodeInput) (*InviteCode, error) {
	forbidden := httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")
	if actor.Role != RoleAdmin && actor.Role != RoleCoordinator {
		return nil, forbidden
	}

	scope, err := s.repo.ResolveCourseScope(ctx, in.CourseID)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NewError(httpx.ErrValidation, "Course not found",
			httpx.ErrorDetail{Field: "course_id", Issue: "not found"})
	}
	if err != nil {
		return nil, err
	}
	if actor.Role == RoleCoordinator && (actor.SchoolID == nil || *actor.SchoolID != scope.SchoolID) {
		return nil, forbidden
	}

	raw, err := newInviteCodeString()
	if err != nil {
		return nil, err
	}
	code := &InviteCode{Code: raw, CourseID: in.CourseID, ExpiresAt: in.ExpiresAt}
	if err := s.repo.CreateInviteCode(ctx, code); err != nil {
		// Collision against the unique index is astronomically unlikely at this
		// alphabet/length, but cheap to self-heal once rather than 500 the caller.
		retryRaw, genErr := newInviteCodeString()
		if genErr != nil {
			return nil, err
		}
		code.Code = retryRaw
		if err := s.repo.CreateInviteCode(ctx, code); err != nil {
			return nil, err
		}
	}
	return code, nil
}

// ListUsers backs the admin/coordinator user directory. Admin sees every
// user under whatever filter they asked for. Coordinator is force-pinned to
// their own school_id regardless of what was requested — per the DB's
// chk_users_scope_matches_role constraint, school_id is only ever set on
// coordinator and student rows (admin and mentor rows always have it NULL),
// so filtering on it alone already limits the result to "coordinators and
// students at my school" with no extra role check needed.
//
// Mentors are the one exception: they're scoped by company_id, not
// school_id, so the blanket school_id pin above would AND against a column
// that's always NULL on a mentor row and silently return nothing. A
// coordinator asking for role=mentor must supply a company_id, which gets
// resolved to a school via CompanyScopeResolver (companies -> departments ->
// school_id) and checked against the coordinator's own school instead of
// pinning school_id directly. This only fixes the single-company lookup —
// it deliberately does not support "every mentor in my school" in one call,
// since UserFilter has no company-list-by-school capability to back that.
func (s *Service) ListUsers(ctx context.Context, actor *User, filter UserFilter, params httpx.ListParams) ([]User, int64, error) {
	forbidden := httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")

	switch actor.Role {
	case RoleAdmin:
		// no restriction — filter passed through as requested
	case RoleCoordinator:
		if actor.SchoolID == nil {
			return nil, 0, forbidden
		}
		if filter.Role != nil && *filter.Role == RoleMentor {
			if filter.CompanyID == nil {
				return nil, 0, httpx.NewError(httpx.ErrValidation, "company_id is required when filtering mentors",
					httpx.ErrorDetail{Field: "company_id", Issue: "required"})
			}
			schoolID, _, err := s.companies.ResolveCompanyScope(ctx, *filter.CompanyID)
			if err != nil {
				return nil, 0, err
			}
			if schoolID != *actor.SchoolID {
				return nil, 0, forbidden
			}
			// Leave filter.SchoolID nil here — mentor rows never have one,
			// and filter.CompanyID above already does the real narrowing.
		} else {
			filter.SchoolID = actor.SchoolID
		}
	default:
		return nil, 0, forbidden
	}

	return s.repo.ListUsers(ctx, filter, params)
}

type CreateStaffAccountInput struct {
	Name                 string
	Email                string
	Password             string
	PasswordConfirmation string
	Role                 Role
	SchoolID             *int64
	CompanyID            *int64
}

// CreateStaffAccount is how a coordinator or mentor account comes to exist
// outside of `make seed` — issued by an admin (any school/company) or a
// coordinator (mentor only, and only for a company within their own
// school). Deliberately not a student- or admin-provisioning path: student
// self-registration already exists via Register, and creating another
// admin isn't a scenario this needs to cover.
//
// Unlike Register, no session is issued — the caller stays logged in as
// themselves, not as the account they just created. The caller sets the new
// account's initial password directly rather than one being generated and
// emailed: Mailer is a logged no-op with no real SMTP provider configured,
// so a generated password would be undeliverable and invisible to everyone,
// caller included. The new user changes it via the existing ChangePassword
// flow after their first login.
func (s *Service) CreateStaffAccount(ctx context.Context, actor *User, in CreateStaffAccountInput) (*User, error) {
	forbidden := httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")
	if actor.Role != RoleAdmin && actor.Role != RoleCoordinator {
		return nil, forbidden
	}
	if in.Role != RoleCoordinator && in.Role != RoleMentor {
		return nil, httpx.NewError(httpx.ErrValidation, "role must be coordinator or mentor",
			httpx.ErrorDetail{Field: "role", Issue: "must be coordinator or mentor"})
	}
	if actor.Role == RoleCoordinator && in.Role != RoleMentor {
		return nil, forbidden
	}
	if in.Password != in.PasswordConfirmation {
		return nil, httpx.NewError(httpx.ErrValidation, "Password confirmation does not match",
			httpx.ErrorDetail{Field: "password_confirmation", Issue: "must match password"})
	}

	user := &User{
		ID:       uuid.NewString(),
		Role:     in.Role,
		Name:     in.Name,
		Email:    in.Email,
		IsActive: true,
	}

	switch in.Role {
	case RoleCoordinator:
		if in.SchoolID == nil {
			return nil, httpx.NewError(httpx.ErrValidation, "school_id is required for a coordinator account",
				httpx.ErrorDetail{Field: "school_id", Issue: "required"})
		}
		user.SchoolID = in.SchoolID
	case RoleMentor:
		if in.CompanyID == nil {
			return nil, httpx.NewError(httpx.ErrValidation, "company_id is required for a mentor account",
				httpx.ErrorDetail{Field: "company_id", Issue: "required"})
		}
		schoolID, _, err := s.companies.ResolveCompanyScope(ctx, *in.CompanyID)
		if err != nil {
			return nil, err
		}
		if actor.Role == RoleCoordinator && (actor.SchoolID == nil || *actor.SchoolID != schoolID) {
			return nil, forbidden
		}
		user.CompanyID = in.CompanyID
	}

	taken, err := s.repo.EmailTaken(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, httpx.NewError(httpx.ErrConflict, "An account with this email already exists",
			httpx.ErrorDetail{Field: "email", Issue: "already registered"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = string(hash)

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) issueSession(ctx context.Context, user *User, familyID, userAgent, ip string) (*LoginResult, error) {
	now := time.Now()

	rawAccess, err := newOpaqueToken()
	if err != nil {
		return nil, err
	}
	accessExpiresAt := now.Add(s.cfg.AccessTTL)
	if err := s.repo.CreateSession(ctx, &Session{
		ID: uuid.NewString(), UserID: user.ID, TokenHash: hashToken(rawAccess),
		Kind: SessionAccess, FamilyID: familyID, UserAgent: userAgent, IP: ip, ExpiresAt: accessExpiresAt,
	}); err != nil {
		return nil, err
	}

	rawRefresh, err := newOpaqueToken()
	if err != nil {
		return nil, err
	}
	refreshExpiresAt := now.Add(s.cfg.RefreshTTL)
	if err := s.repo.CreateSession(ctx, &Session{
		ID: uuid.NewString(), UserID: user.ID, TokenHash: hashToken(rawRefresh),
		Kind: SessionRefresh, FamilyID: familyID, UserAgent: userAgent, IP: ip, ExpiresAt: refreshExpiresAt,
	}); err != nil {
		return nil, err
	}

	rawCSRF, err := newOpaqueToken()
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:             user,
		AccessToken:      rawAccess,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     rawRefresh,
		RefreshExpiresAt: refreshExpiresAt,
		CSRFToken:        rawCSRF,
	}, nil
}

// Authenticate validates a raw access-cookie value and returns its user —
// the RequireAuth middleware's core, exposed here (not in middleware) so it
// stays testable without spinning up Gin, and so the future SSE endpoint
// (Phase 2) can reuse it outside any HTTP-middleware chain.
func (s *Service) Authenticate(ctx context.Context, rawAccessToken string) (*User, error) {
	unauthenticated := httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated")
	if rawAccessToken == "" {
		return nil, unauthenticated
	}

	session, err := s.repo.FindActiveSessionByHash(ctx, hashToken(rawAccessToken))
	if errors.Is(err, ErrNotFound) {
		return nil, unauthenticated
	}
	if err != nil {
		return nil, err
	}
	if session.Kind != SessionAccess {
		return nil, unauthenticated
	}

	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if errors.Is(err, ErrNotFound) {
		return nil, unauthenticated
	}
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, httpx.NewError(httpx.ErrForbidden, "This account has been deactivated. Contact your administrator.")
	}
	return user, nil
}

// ChangeProfileInput covers the self-editable fields. Role, scope columns,
// email, and NIS are deliberately excluded — those are administrative, not
// self-service (changing your own school/company/role would be a privilege
// escalation vector; email changes would need a re-verification flow this
// build doesn't have yet — see README's Future Improvements).
type ChangeProfileInput struct {
	Name        *string
	Bio         *string
	Address     *string
	Phone       *string
	Gender      *Gender
	DateOfBirth *time.Time
	Skills      *string
}

func (s *Service) ChangeProfile(ctx context.Context, userID string, in ChangeProfileInput) (*User, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated")
	}
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		user.Name = *in.Name
	}
	if in.Bio != nil {
		user.Bio = in.Bio
	}
	if in.Address != nil {
		user.Address = in.Address
	}
	if in.Phone != nil {
		user.Phone = in.Phone
	}
	if in.Gender != nil {
		user.Gender = in.Gender
	}
	if in.DateOfBirth != nil {
		user.DateOfBirth = in.DateOfBirth
	}
	if in.Skills != nil {
		user.Skills = in.Skills
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// ChangePassword requires the current password (proof of continued
// authorization, not just an active cookie — matches the reset-password
// flow's security bar) and, like ResetPassword, revokes every existing
// session on success: a password change is exactly the moment to assume
// every other open session might be the thing you're trying to lock out.
func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.repo.FindUserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated")
	}
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return httpx.NewError(httpx.ErrValidation, "Current password is incorrect",
			httpx.ErrorDetail{Field: "current_password", Issue: "incorrect"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return err
	}
	return s.repo.RevokeAllUserSessions(ctx, user.ID)
}

// UploadAvatar validates + stores the image via the shared storage client
// (content-type sniffed from bytes, size-capped, never the client filename
// — see plan section 2.7) and updates the user's avatar_key.
func (s *Service) UploadAvatar(ctx context.Context, userID string, data []byte, filename string) (*User, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return nil, httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated")
	}
	if err != nil {
		return nil, err
	}

	result, err := s.storage.Upload(ctx, storage.UploadInput{
		Bucket: storage.BucketAvatars, KeyPrefix: "avatars/" + userID,
		OriginalFilename: filename, Data: data,
		AllowedKinds: []string{"image"}, MaxBytes: storage.MaxImageBytes,
	})
	if err != nil {
		return nil, httpx.NewError(httpx.ErrValidation, err.Error(), httpx.ErrorDetail{Field: "avatar", Issue: err.Error()})
	}

	user.AvatarKey = &result.Key
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
