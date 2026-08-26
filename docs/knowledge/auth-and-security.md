# Auth & Security — Backend Reference

Backend-focused. Frontend interceptor mechanics (axios wiring, `useAuthStore`, route
guards) belong in `frontend-architecture.md` — this file goes deep on session storage,
CSRF correctness, rate limiting, and RBAC scoping in `apps/api`.

All primary source: `apps/api/internal/modules/identity/{handler,service,repository,domain,tokens}.go`,
`apps/api/internal/middleware/{auth,csrf,rbac,ratelimit}.go`, `apps/api/internal/httpx/pagination.go`,
`apps/api/internal/modules/review/service.go`, `apps/api/cmd/api/main.go`.

## Session model

Three cookies, all set together in `identity.Handler.setAuthCookies` (handler.go:272-282):

| Cookie | Purpose | HttpOnly | Path | Domain | TTL |
|---|---|---|---|---|---|
| `internity_session` | access token | yes | `/` | host-only (`""`) | `AccessTTL` = 15 min (`DefaultConfig`, service.go:38-40) |
| `internity_refresh` | refresh token | yes | `/api/v1/auth/refresh` only | host-only | `RefreshTTL` = 7 days |
| `internity_csrf` | double-submit token | **no** (must be JS-readable) | `/` | `cfg.CookieDomain` (empty by default) | same as refresh |

All three: `SameSite=Lax` (`c.SetSameSite(http.SameSiteLaxMode)`, called once per
`setAuthCookies`/`clearAuthCookies`), `Secure=h.cookieSecure` (wired from `COOKIE_SECURE`
env var, `true` in the Dokploy compose file). `clearAuthCookies` (handler.go:284-289) sets
all three to `""` with maxAge `-1`, same path/domain as they were set with — a mismatched
path/domain on the clear call would silently fail to delete the cookie, so if you ever
change a cookie's Path/Domain, update both `setAuthCookies` and `clearAuthCookies` together.

Tokens are opaque (32 random bytes, base64 URL-encoded — `newOpaqueToken`, tokens.go:12-18),
never JWTs. Only `sha256(raw)` hex is persisted (`hashToken`, tokens.go:20-23); the DB
(`sessions` table) never sees the raw value, so a DB dump alone can't be replayed. Access
and refresh tokens are both rows in the same `Session` table (`Kind` = `access`/`refresh`,
domain.go:63-87) — `Authenticate` (service.go:553-581) explicitly rejects a refresh-kind
token presented as an access cookie and vice versa in `Refresh` (service.go:205-207), so
the two can't be swapped.

**Session family / rotation** (service.go:173-227): every session issued together (login,
or each successive refresh from it) shares a `FamilyID` (a UUID, `= session.FamilyID` on
refresh, `= uuid.NewString()` on a fresh login/register). `Refresh` revokes the presented
refresh token and issues a brand-new access+refresh pair *in the same family*. If a
refresh token that's already `RevokedAt != nil` is presented again — i.e. someone replayed
an old refresh cookie — that's treated as theft evidence and `RevokeSessionFamily` nukes
**every** session in that family (service.go:208-211), forcing a full re-login on every
device that shared it. This is the standard refresh-token-rotation-with-reuse-detection
pattern; don't "fix" a replayed-refresh 401 by making it just re-issue silently.

`ResetPassword` and `ChangePassword` both call `RevokeAllUserSessions` on success
(service.go:309, 662) — a password change/reset invalidates every session everywhere, not
just the one that triggered it.

## CSRF — double-submit pattern

`middleware.RequireCSRF()` (csrf.go:30-46) runs only on mutating methods (POST/PUT/PATCH/
DELETE, `mutatingMethods` map) and only on the `authed` route group in `server.go:74-75`
(public `/auth/*` routes — login/register/refresh/forgot/reset — are exempt, there's no
session yet to protect). It compares the `internity_csrf` cookie value against the
`X-CSRF-Token` request header with `subtle.ConstantTimeCompare`; mismatch or either-missing
→ 403 `FORBIDDEN`. The security property: a cross-site request can make the browser *send*
the cookie automatically, but cross-origin JS cannot *read* it to also set the header — the
CSRF cookie is deliberately **not** `HttpOnly` for exactly this reason (see the comment
block at handler.go:276-281).

Frontend side reads it via `document.cookie` regex (`apps/dashboard/src/lib/http.ts:15-16,28-29`,
`readCookie('internity_csrf')`) and attaches it as `X-CSRF-Token` on every request through
the shared axios instance.

### Two real bugs fixed here — check these first if CSRF errors show up again

**1. Refresh used to mint a new CSRF token every time → spurious 403s under concurrency.**
`issueSession` (service.go:504-547) takes an `existingCSRFToken` param and reuses it as-is
(`rawCSRF := existingCSRFToken`, only generating a new one if that's `""`). `Login`/`Register`
pass `""` (nothing to carry forward on a brand-new session); `Refresh` passes whatever the
caller's current `internity_csrf` cookie value was, read by the handler
(`existingCSRF, _ := c.Cookie(CookieCSRF)`, handler.go:109) *before* calling the service.
The full reasoning is in the doc comment on `Service.Refresh` (service.go:178-193): the
frontend reads `document.cookie` synchronously to build the header, but the browser attaches
the actual `Cookie` header at network-dispatch time — a distinct, later moment. If some
*other* concurrent request's `/auth/refresh` response rotated the CSRF cookie in that gap, a
request already in flight would ship a stale header against a fresh cookie → a clean single
403 with no visible retry, on a URL that has nothing to do with the refresh that caused it.
**Do not rotate the CSRF cookie value on refresh.** Only its expiry is extended (same value,
new `RefreshExpiresAt`-based maxAge in `setAuthCookies`).

**2. `CookieDomain` defaulting empty broke CSRF whenever dashboard and API are on different
subdomains.** With `COOKIE_DOMAIN` unset, `internity_csrf` is host-only, scoped to the API's
own origin (e.g. `api.example.com`). The Dokploy deploy (`docs/dokploy.md:65-66`) puts the
API at `api.*` and the dashboard at `app.*` — the dashboard's JS running on `app.*` can
**never** read a cookie scoped to `api.*` via `document.cookie`, even though the cookie is
visibly set and sent by the browser and shows up fine in devtools' Application tab. Result:
`X-CSRF-Token` silently never gets attached, every mutating request 403s, and it looks like
a token problem rather than a scoping problem. Fixed via the `COOKIE_DOMAIN` env var
(`config.go:17-26`, e.g. `.example.com`), wired only into the CSRF cookie's `Domain` param
in `setAuthCookies`/`clearAuthCookies` (`h.cookieDomain` — the session/refresh cookies
deliberately stay host-only, they're HttpOnly and only ever need to reach the API's own
host, see the field comment). In `docker-compose.dokploy.yml:132`: `COOKIE_DOMAIN: ${COOKIE_DOMAIN:-}`.

**If `X-CSRF-Token` isn't being sent, or CSRF errors appear (esp. after a deploy or under
load):** check (1) whether refresh is racing — look for 403s clustered right after 401s on
unrelated endpoints; check (2) whether `COOKIE_DOMAIN` is set correctly for the environment
whenever dashboard and API are on different hosts/subdomains — an empty/wrong value is
invisible in devtools (cookie *looks* present) and only shows up as "JS can't read this
cookie from this origin."

## Rate limiting

`middleware.RateLimit` (ratelimit.go:24-56) is a Redis fixed-window counter: `INCR` a key,
`EXPIRE` it on first hit, 429 with a `Retry-After` header once the count exceeds `limit`
within `window`. Deliberately fails **open**, not closed — if Redis errors, the request is
allowed through (`c.Next()` on `Incr` error, ratelimit.go:31-38) since rate limiting is
defense-in-depth, not the auth boundary itself.

**Actual limit** (server.go:70): `RateLimit(deps.Redis, 10, 5*time.Minute, AuthRateLimitKey)`
— **10 requests per 5-minute window**, keyed by `IP + ":" + email` (`AuthRateLimitKey`,
ratelimit.go:69-84, reads/re-buffers the JSON body's `email` field; falls back to IP alone if
no email present/parseable). Applied as one shared middleware across the entire `/auth`
public group (`identity/routes.go:12-19`): `login`, `register`, `refresh`,
`forgot-password`, `reset-password` **all count against the same 10/5min bucket per
IP+email** — they are not independently limited. The 10 (not a tighter 5) was deliberately
chosen because the E2E suite's own legitimate business-flow tests tripped a 5-request limit
purely from necessary re-logins in one account during normal multi-step flows (see the
comment at server.go:59-69).

**Practical consequence for live/manual testing:** repeatedly logging in and out against the
shared dev DB (e.g. iterating on a login-flow fix, or testing session revocation by hand)
burns through this same 10-per-5-minutes bucket per email+IP. Hitting it produces a 429 with
`Retry-After`, not a hang — but if you're scripting rapid repeated login/refresh/logout
cycles against dev, budget for this or expect flakiness that looks like a bug but isn't.
Waiting out `Retry-After` (or switching the test email) clears it; there's no manual-reset
endpoint.

## RBAC pattern

Two layers, deliberately never merged into one generic helper (see the comment on
`middleware.RequireRole`, rbac.go:10-15):

1. **Coarse gate — `middleware.RequireRole(roles...)`** (rbac.go:16-34): 403s unless
   `actor.Role` is in an allow-list. Runs after `RequireAuth`. Answers "is this role even
   allowed to hit this route at all" — nothing about *which* row.
2. **Fine-grained ownership — hand-written per service method.** "Does this specific
   company/school/student belong to this specific actor" differs per entity/join-path, so
   it lives next to each module's business logic, not in middleware. The standard
   cross-module lookup shape is a narrow interface + adapter wired in `cmd/api/main.go`:
   e.g. `identity.CompanyScopeResolver` / `review.CompanyScopeResolver` (both
   `ResolveCompanyScope(ctx, companyID) (schoolID, departmentID int64, err error)`) are
   satisfied by `companyScopeAdapter{repo: orgsRepo}` (main.go:76-84) so a module never
   imports another module's concrete repository directly.

  A correctly-scoped example: `identity.Service.ListUsers` (service.go:375-407) — admin
  unrestricted; coordinator force-pinned to `filter.SchoolID = actor.SchoolID` **except**
  when filtering `role=mentor`, where school_id is always NULL on a mentor row so the pin
  would silently return zero rows — that branch instead requires `company_id` and resolves
  it to a school via `CompanyScopeResolver` before comparing to the actor's own school
  (service.go:385-398, read the comment — it explains exactly why the naive pin breaks for
  mentors specifically).

### Worked example of a fix: `review.Service` had zero ownership check for mentors

`review/service.go:195-234`. Before the fix, `ListReviewsForUser` and `CreateReview` scoped
students (`actor.Role == RoleStudent && actor.ID != userID → forbidden`) but had **no
check at all** for the mentor branch — a mentor role, given *any* student UUID (guessed,
enumerated, or pasted from another URL), could read or write that student's review record
regardless of which company that student was actually placed at. Every sibling
mentor-facing endpoint (scoring, presences, journals) already scoped to the mentor's own
company; this one didn't.

Fix: a new narrow interface mirroring the `CompanyScopeResolver` adapter pattern —

```go
// PlacementChecker answers "is this student placed at this company"
type PlacementChecker interface {
    HasPlacementAtCompany(ctx context.Context, userID string, companyID int64) (bool, error)
}
```

satisfied by `placementCheckerAdapter{repo: internshipRepo}` (main.go:120-134), which calls
`internshipRepo.GetByUserCompany(userID, companyID)` and treats `ErrNotFound` as
`(false, nil)` rather than an error. `review.Service.assertMentorMentorsStudent`
(service.go:222-234) is now called from both `ListReviewsForUser` (service.go:199-203) and
`CreateReview`'s `RevieweeUser` branch (service.go:168-170): forbidden if
`actor.CompanyID == nil`, forbidden if the student has no real `InternDate` row at that
company. Regression tests: `review/service_test.go`
`TestListReviewsForUser_MentorScopedToOwnCompany`,
`TestAssertMentorMentorsStudent_AllowsRealPlacement`,
`TestCreateReview_MentorScopedToOwnCompany`.

**Checklist for scoping any new mentor/coordinator-facing endpoint** (apply this whenever
adding a handler that isn't admin-only): for every non-admin, non-coordinator-with-a-clean-
school-pin role that can reach the endpoint, ask — is there an explicit
ownership/relationship check tying the actor to *this specific resource*, or does the code
only check "is authenticated" / "has the right role name"? `RequireRole` only proves the
second. A role check alone is not a scope check — `identity.Role` membership says nothing
about *whose* data is being touched. If the resource has an owner/company/school column,
there must be a corresponding actor-vs-resource comparison in the service layer before the
query runs, following either the `CompanyScopeResolver` (school/department ownership) or
`PlacementChecker` (mentor-student relationship) shape above rather than inventing a new one.

## Password reset flow

`ForgotPassword` (service.go:254-277) always returns success regardless of whether the
email exists (`errors.Is(err, ErrNotFound) → return nil`) — prevents user enumeration via
this endpoint. Token: `newOpaqueToken()` (32 random bytes), only `hashToken(raw)` stored in
`password_reset_tokens.token_hash`, 30-minute expiry, delivered via
`Mailer.SendPasswordReset` (queued through `queueMailerAdapter` → worker; currently a
logged no-op, `identity.NoopMailer`, no real SMTP wired — see service.go:15-31).
`ResetPassword` (service.go:279-310) validates + consumes the token
(`FindActivePasswordResetToken` filters `used_at IS NULL AND expires_at > now`,
`MarkPasswordResetTokenUsed` after success) and — like `ChangePassword` — revokes every
existing session on that account.

**guestOnly-guard interaction (frontend, `apps/dashboard/src/router/index.ts:88-112`):**
`/reset-password` carries `?token=...` for a *specific* account and is `meta: { guestOnly: true }`
like `/login`/`/register`. The router's global guard bounces any `guestOnly` route straight
to `/dashboard` when `auth.isAuthenticated` — correct for `/login`, but for
`/reset-password` this used to silently discard the token with zero explanation: someone
already logged in on that browser (a different account, or a stale session) who clicks a
real reset-password email link just gets redirected to their own dashboard with no
indication the link "did" anything. Fixed by special-casing `to.name === 'reset-password'`
in the guard to `toast.error('You need to log out before using a password reset link for a
different account.')` before the redirect (router/index.ts:97-104). If you touch this guard,
keep that special case — the silent-discard behavior is correct for every *other*
`guestOnly` route but actively misleading for this one.

## Search input sanitization — `httpx.ParseListParams` is the one choke point

`apps/api/internal/httpx/pagination.go:26-49`. Every list endpoint across every module
(`content`, `identity`, `internship`, `notification`, `orgs`, `review`, `scoring`,
`vacancy` — confirmed via grep, all call `httpx.ParseListParams`) parses
`page`/`limit`/`search`/`sort`/`order` through this single function; there is no other path
a `search` query param reaches an ILIKE clause through.

`sanitizeSearch` (pagination.go:58-70) strips NUL and other C0/C1 control characters (but
**not** `\t`/`\n`/`\r` — see `isUnwantedControl`, pagination.go:76-82, a pasted multi-line
search term is legitimate) before the value is used. This exists because **a bare NUL byte
in a `search` param used to crash list-search endpoints with a 500**: Postgres rejects NUL
bytes in `text` values outright, and an unfiltered NUL reaching an `ILIKE ?` bind panicked
(caught by `middleware.Recovery`, surfaced as an opaque `INTERNAL_ERROR`) instead of
returning a clean result. If you add a new list endpoint, route its search param through
`ParseListParams` rather than reading `c.Query("search")` directly — that's what keeps this
fixed instead of reintroduced per-endpoint. Regression tests:
`apps/api/internal/httpx/pagination_test.go` (`TestParseListParams_StripsNULByteFromSearch`,
`_StripsOtherControlCharsFromSearch`, `_KeepsCommonWhitespaceInSearch`).

## Error envelope reference

`httpx.FailFromErr` (envelope.go:102-114): a known `*httpx.APIError` (built via
`httpx.NewError(code, message, ...details)`) passes its message straight to the client;
anything else (raw GORM/driver/network error) is logged server-side with request/trace ID
and answered with a generic `"Something went wrong..."` — internals never leak. Status
codes: `UNAUTHENTICATED`→401, `FORBIDDEN`→403, `VALIDATION_ERROR`→422, `NOT_FOUND`→404,
`CONFLICT`→409, `RATE_LIMITED`→429, `BAD_REQUEST`→400, `INTERNAL_ERROR`→500
(`statusByCode`, envelope.go:31-40). When adding a new auth/RBAC check, reuse
`httpx.NewError(httpx.ErrForbidden, "You do not have permission to do that")` /
`httpx.NewError(httpx.ErrUnauthenticated, "Not authenticated")` rather than inventing new
message text — both `identity` and `review` already define these as shared package-level
vars (`errForbidden`/`errNotFoundAPI` in review/service.go:12-13) precisely so every 403 in
a module reads identically.
