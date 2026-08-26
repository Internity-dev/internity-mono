# apps/api backend architecture

Go 1.x, Gin, GORM-as-query-builder-only (no AutoMigrate), Postgres, Redis, MinIO, asynq queue.
Module path is `internity` (imports are `internity/internal/...`, not a GitHub path).

All claims below are verified against the actual files at the cited paths — read the cited
file:line instead of re-grepping when you need the full picture.

## Module layout (domain.go / repository.go / service.go / handler.go / routes.go)

Every module under `apps/api/internal/modules/<name>/` follows the same 5-file split. Verified
against `orgs`, `vacancy`, `review`, `scoring`, `content` — all identical shape.

- **domain.go** — GORM structs (`TableName()` method, `gorm:"column:..."` tags matching the
  migration exactly), enums/status types, and pure domain logic that needs no DB or service
  (e.g. `vacancy.Appliance.CanTransitionTo` in `vacancy/domain.go:86`, `review.AverageRating` in
  `review/service.go:38` — note that one pure helper lives in service.go, not domain.go, when
  it's conceptually service-level).
- **repository.go** — a concrete `type Repository struct { db *gorm.DB }` (NOT an interface —
  see the one exception, identity, below) with `NewRepository(db *gorm.DB) *Repository`, one
  method per query. Owns `ErrNotFound` (a plain `errors.New`, module-local) and a
  `translateNotFound`/`translateNotFound` helper that maps `gorm.ErrRecordNotFound` to it. No
  business logic, no actor/role awareness — just query shape.
- **service.go** — `type Service struct { repo *Repository; <cross-module interfaces...> }`,
  holds ALL authorization logic (role checks, scope checks), input validation (range checks,
  required-relationship checks), and orchestration across the repo + cross-module interfaces.
  Every exported method takes `actor *identity.User` as its second param (after ctx) and is
  where a request gets allowed or `errForbidden`'d. Owns `translateGetErr` (ErrNotFound ->
  `httpx.ErrNotFound` APIError) and `translateWriteErr` (routes through
  `postgres.TranslateError`, see below).
- **handler.go** — `type Handler struct { svc *Service }`, one method per route. Parses
  path/query params (`idParam`, `queryInt64` — copy-pasted verbatim per module, not shared —
  see `orgs/handler.go:23`, `vacancy/handler.go:24`), binds request DTOs (`createXRequest` /
  `XPatch` structs with `binding:"..."` validator tags), calls `middleware.CurrentUser(c)` to
  get the actor, calls the service, and writes the response via `httpx.OK`/`httpx.FailFromErr`.
  Zero business logic — a handler that branches on `actor.Role` is a bug, that belongs in
  service.go.
- **routes.go** — `func (h *Handler) RegisterRoutes(rg *gin.RouterGroup)`, just route
  registration. Every routes.go has a comment stating it must be mounted under a group that
  already has `RequireAuth` (see `orgs/routes.go:5-8`, `vacancy/routes.go:5-7`) — auth is never
  re-checked per-module, it's applied once in `server/server.go`.
- **patches.go** (optional, e.g. `orgs/patches.go`, `vacancy/patches.go`, `scoring/patches.go`)
  — `XPatch` DTOs with every field as a pointer (`*string`, `*bool`, ...) so a PUT only touches
  fields the caller actually sent, plus an `applyTo(existing *X)` method that nil-checks each
  field before assigning. This is the update pattern everywhere: `Get -> patch.applyTo(existing)
  -> repo.Update(existing)` (full `Save`, not a partial `Updates` map).

### Module boundary discipline

- `domain.go` never imports `service.go` concepts; `repository.go` never imports `identity` for
  role checks (repositories are role-blind); `service.go` is the only layer that imports
  `identity.User` and branches on `.Role`.
- Every service method that mutates re-fetches the row via `repo.GetX` first (even
  Update/Delete) specifically so the scope check has real data to check against — there is no
  "trust the client's payload" path anywhere.

## httpx package (`apps/api/internal/httpx`)

Three files: `pagination.go`, `envelope.go`, `binding.go`.

### ListParams / ParseListParams (`pagination.go`)

```go
type ListParams struct { Page, Limit int; Search, Sort, Order string }
func (p ListParams) Offset() int { return (p.Page - 1) * p.Limit }
func ParseListParams(c *gin.Context, defaultSort string, allowedSort map[string]bool) ListParams
```

- `page` defaults to 1 if missing/invalid/<1. `limit` defaults to 20, clamped to max 100.
- `sort` falls back to `defaultSort` if not in the module's `allowedSort` map (e.g.
  `orgSortColumns` in `orgs/handler.go:21`, `vacancySortColumns`/`applianceSortColumns` in
  `vacancy/handler.go:21-22`) — **this is the SQL-injection guard for the ORDER BY clause**,
  since `Sort`/`Order` get string-concatenated straight into `.Order(...)` in every repository
  (see below). Never add a sortable column without adding it to the allow-list map first.
- `search` is passed through `sanitizeSearch` which strips NUL/control bytes (not other
  unicode) — a raw NUL in an ILIKE param panics in Postgres and would otherwise become an
  unhandled 500 via the recovery middleware (`pagination.go:51-56`).

### The repository ListX pattern — two fresh `Model()` builds

Every `ListX` repository method (verified in `orgs.Repository.ListSchools`
(`orgs/repository.go:33`), `orgs.Repository.ListDepartments` (`:80`),
`vacancy.Repository.ListVacancies` (`vacancy/repository.go:28`),
`scoring.Repository.ListScoresPaged` (`scoring/repository.go:36`)) follows this exact shape:

```go
scope := func(q *gorm.DB) *gorm.DB { /* apply filters (search, foreign-key filters) */ return q }

countQ := scope(r.db.WithContext(ctx).Model(&X{}))
var total int64
if err := countQ.Count(&total).Error; err != nil { return nil, 0, err }

var rows []X
err := scope(r.db.WithContext(ctx).Model(&X{})).
    Order(params.Sort + " " + params.Order).Limit(params.Limit).Offset(params.Offset()).Find(&rows).Error
```

**Why two separate `Model()` calls, not one query reused**: GORM chains are stateful — reusing
one `*gorm.DB` for both `.Count()` and then `.Find()` carries `.Count()`'s side effects
(notably its own implicit ordering/select changes) into the second query and produces wrong
results. The `scope` closure exists so the *filter* logic (WHERE clauses only) is written once
and applied fresh to two independent query builders — count gets it without
Limit/Offset/Order, find gets it with all three. `orgs.applyList` (`orgs/repository.go:24`) is
a shared helper for the find-side chain across all 4 org entities, but even it is always called
against a *fresh* `Model()` build, never the same builder used for `.Count()`.

Simpler ListX methods without extra filters (e.g. `orgs.Repository.ListSchools`) inline the
same shape without a named `scope` closure — same two-fresh-builds rule, just no filter to
factor out.

**Gotcha**: `vacancy.Repository.ListVacancies` and `CountAppliancesByStatus` qualify sortable/
groupable columns with the table name (`"vacancies." + params.Sort`, `appliances.status`)
because their `scope` closure can `JOIN` in other tables that have colliding column names
(`vacancies.status` vs `appliances.status`) — an unqualified `Order(params.Sort ...)` would be
ambiguous SQL once a join is in the WHERE-scope. Modules without joins in their scope (orgs,
scoring) don't need this.

### Pagination / OK envelope (`envelope.go`)

```go
httpx.OK(c, http.StatusOK, rows, "OK", &httpx.Pagination{Page: params.Page, Limit: params.Limit, Total: total})
```

Response shape: `{ success: true, data, message, meta: { request_id, pagination? } }`.
`Pagination` is `{page, limit, total}` — the handler always builds it straight from `params` +
the repo's returned `total`, never recomputed. Non-list `OK` calls pass `nil` for pagination.

### Errors (`envelope.go`, `binding.go`)

- `APIError{Code, Details, Message}` — `Code` is a typed `ErrorCode` string, mapped to an HTTP
  status via `statusByCode` (`envelope.go:31-40`): `BAD_REQUEST`=400, `VALIDATION_ERROR`=422,
  `UNAUTHENTICATED`=401, `FORBIDDEN`=403, `NOT_FOUND`=404, `CONFLICT`=409, `RATE_LIMITED`=429,
  `INTERNAL_ERROR`=500.
- **BAD_REQUEST vs VALIDATION_ERROR distinction is deliberate** (`envelope.go:17-20`):
  BAD_REQUEST = the request couldn't even be parsed (bad JSON, wrong type, non-numeric path
  param); VALIDATION_ERROR = it parsed fine but a value fails a business/format rule
  (`binding:"required,min=2"` tag failure, or a hand-written range check in a service like
  scoring's "Score must be between 0 and 100").
- `httpx.Fail(c, err)` writes the error envelope directly — use when you already have an
  `*APIError` (e.g. from `httpx.BindingError` or `httpx.BadPathParam`).
- `httpx.FailFromErr(c, err)` — the handler-level catch-all: if `err` unwraps to `*APIError` via
  `errors.As`, passes through as-is; otherwise logs the raw error server-side (with
  request_id/trace_id) and responds with a generic `INTERNAL_ERROR` message — **raw
  driver/GORM errors never reach the client**. Every handler ends its error branch with
  `httpx.FailFromErr(c, err); return`.
- `httpx.BindingError(err)` classifies a `c.ShouldBindJSON` failure: `validator.ValidationErrors`
  -> 422 with one `ErrorDetail{Field: snake_case(field), Issue: tag}` per failed field;
  JSON syntax/type errors -> 400 "Request body is not valid JSON".

## Cross-module communication (interfaces wired in `cmd/api/main.go`)

**Rule, confirmed by reading every module's service.go + main.go**: a module's `Service` never
imports another module's `Repository` type directly. Instead it declares a small
narrow interface for exactly what it needs, and `main.go` wires a concrete adapter (or, for
identity, the module's own already-interface `Repository`) at construction time.

Examples (all read from `cmd/api/main.go`):

1. **`vacancy.CompanyScopeResolver`** (`vacancy/service.go:29-32`) — needs
   `ResolveCompanyScope(companyID) (schoolID, departmentID, error)` and
   `ResolveDepartmentSchool(departmentID) (schoolID, error)`. Wired via
   `companyScopeAdapter{repo: orgsRepo}` (`main.go:76-92`), which adapts `orgs.Repository`'s
   concrete `*CompanyScope` struct return into the narrow tuple shape. The *same*
   `companyScopeAdapter` type is reused to satisfy `scoring.CompanyScopeResolver`,
   `review.CompanyScopeResolver`, and `internship.Service`'s equivalent need — one adapter
   struct, multiple consumer interfaces, because they all happen to want the identical method
   set.
2. **`review.PlacementChecker`** (`review/service.go:23-25`) — needs
   `HasPlacementAtCompany(userID, companyID) (bool, error)`. Wired via
   `placementCheckerAdapter{repo: internshipRepo}` (`main.go:123-134`), which translates
   `internship.Repository.GetByUserCompany`'s not-found error into a clean `false, nil` (no
   placement) instead of leaking `internship.ErrNotFound` across the module boundary.
3. **`scoring.OrgLookup` / `scoring.StudentLookup`** (`scoring/service.go:25-38`) — read-only
   display-name lookups. Wired via `orgLookupAdapter{repo: orgsRepo}` and
   `studentLookupAdapter{repo: identityRepo}` (`main.go:96-149`) — note these adapters also
   translate the target module's not-found sentinel into an `httpx.NewError(httpx.ErrNotFound,
   ...)` *at the adapter*, so the consuming service never needs to know the producing module's
   internal error type.
4. **`content.DepartmentScopeResolver`** — same shape as vacancy's, wired via
   `departmentScopeAdapter{repo: orgsRepo}` (`main.go:168-179`), a *separate* adapter type from
   `companyScopeAdapter` even though both wrap `orgsRepo`, because this one only needs
   `ResolveDepartmentSchool` and returns an `httpx.NewError` on not-found rather than a raw error.
5. **`reporting`** depends on 5 narrow interfaces simultaneously (`main.go:272-273`): student
   lookup, presence export, company scope, org lookup — each wired with its own
   adapter/reused adapter.

**Exception to "no direct repo import"**: `identity.Repository` (`identity/repository.go:17`)
is itself declared as an **interface**, not a concrete struct (unlike every other module's
repository). Because of that, other modules can depend on it directly with zero adapter glue as
long as they only need a subset of its methods — Go's structural typing means
`identity.Repository` (the interface) already satisfies e.g. `content.AudienceResolver`
(`ListUserIDsBySchool`/`ListUserIDsByDepartment`) with no wrapper struct at all: `main.go:265`
passes `identityRepo` straight in. When the *shape* needed differs from identity's own return
type (e.g. `scoring.StudentInfo` vs `reporting.StudentInfo` are structurally identical but
distinct named types), an adapter is still used purely to reshape the struct
(`studentLookupAdapter`, `reportingStudentLookupAdapter`, `main.go:138-164`) — not to hide
`identity.Repository`, since it's already interface-shaped.

**Where to add a new one**: define the interface in the *consuming* module's `service.go` next
to its `Service` struct, scoped to exactly the methods needed (never reuse a whole existing
interface if you only need one method of it). Add the adapter struct in `cmd/api/main.go`
(pattern: `type xAdapter struct{ repo *producerModule.Repository }`, method bodies translate
the producer's sentinel errors to either a clean zero-value or an `httpx.NewError`). Wire it
into the consuming module's `NewService(...)` call in `main()`.

## RBAC / scoping conventions

There is **no generic authorization middleware for per-resource ownership** —
`middleware.RequireRole` exists (`middleware/rbac.go:16`) but grep confirms **it is never
actually called from any routes.go** in the repo; it's dead code (or a hook for future use).
Actual RBAC is 100% hand-written per-service-method, using `actor.Role` plus the resource's
scope columns (`SchoolID`/`DepartmentID`/`CompanyID`) on `identity.User`. The pattern repeats
near-verbatim across modules; the canonical helper names to grep for are
`assertCanManageCompany` / `assertCanViewCompany` / `assertCanManageSchool` /
`assertCanViewSchool` / `scopedSchoolFilter`.

### Roles and their default reach

`identity.Role`: `admin`, `coordinator`, `mentor`, `student` (`identity/domain.go:8-15`).

- **admin** — unscoped/broad access everywhere. Every `assertCanManage*`/`assertCanView*`
  helper's first branch is `if actor.Role == identity.RoleAdmin { return nil }` (see
  `orgs/service.go:339-344`, `vacancy/service.go:351-372`, `scoring/service.go:280-321`,
  `review/service.go:236-255`). No exceptions found.
- **coordinator** — broad *within their own school*, resolved via `actor.SchoolID`. Two
  sub-patterns:
  - Direct compare when the resource has (or resolves cheaply to) a `school_id`:
    `actor.SchoolID != nil && *actor.SchoolID == schoolID` (`orgs.canManageSchool`,
    `orgs/service.go:339-344`; `scoring.assertCanManageSchool`, `scoring/service.go:313-321`).
  - Resolved compare when the resource only has a `company_id`/`department_id`: call the
    `CompanyScopeResolver`/`DepartmentScopeResolver` interface to get the owning school, then
    compare (`vacancy.assertCanManageCompany`, `vacancy/service.go:351-372`;
    `content.assertCanManageScope`, `content/service.go:243-264`).
  - `orgs.scopedSchoolFilter` (`orgs/service.go:346-358`) is the read/list-side version: admin's
    requested `school_id` query filter passes through untouched; every other role is **pinned**
    to `actor.SchoolID` regardless of what filter value they passed in the query string — a
    coordinator/mentor/student can never widen a list by passing a different `school_id`.
- **mentor** — narrow, tied to exactly `actor.CompanyID` (a single int64, not a list). Every
  mentor check is `actor.CompanyID == nil || *actor.CompanyID != companyID -> forbidden`
  (`vacancy/service.go:355-358`, `scoring/service.go:284-287`,
  `orgs.GetCompany`/`orgs/service.go:260-264`). **A mentor is never granted access via role
  alone** — every mentor-facing check compares the specific `company_id` on the resource
  against `actor.CompanyID`.
  - `review.assertMentorMentorsStudent` (`review/service.go:222-234`) is the sharpest example
    and is explicitly documented in-code as **the fix for a real, live-confirmed cross-tenant
    leak**: `ListReviewsForUser`/`CreateReview` used to let *any* mentor read/write *any*
    student's reviews just by knowing/guessing their UUID. The fix calls
    `review.PlacementChecker.HasPlacementAtCompany(studentID, actor.CompanyID)` — i.e. "does
    this student actually have a placement (an `intern_dates`/appliance-derived relationship)
    at my company" — not just "is this user a mentor". This is the model for any future
    student-scoped-to-mentor check: **verify an actual placement/company_id match, never
    'any mentor may see any student'.**
- **student** — narrowest; mostly self-scoped (`actor.ID == userID`) or department-scoped
  (`actor.DepartmentID`) for browsing. E.g. `vacancy.ListVacancies`'s student branch
  (`vacancy/service.go:75-84`) forces `filter.DepartmentID = actor.DepartmentID` and, absent an
  explicit company filter, forces `Status = VacancyOpen` (students never browse closed
  vacancies by default). `scoring.ListScores`/`GenerateCertificate` let a student see only
  `actor.ID == userID`'s own rows (`scoring/service.go:54-64`, `180-188`).

### The per-role branching pattern (list endpoints)

`vacancy.Service.ListVacancies` (`vacancy/service.go:59-87`) is the clearest example of the
"one method, `switch actor.Role`" idiom used for every list endpoint that has different default
visibility per role:

```go
switch actor.Role {
case identity.RoleAdmin:
    // no restriction
case identity.RoleMentor:
    filter.CompanyID = actor.CompanyID // force to own company, ignore any caller-passed value
case identity.RoleCoordinator:
    if filter.CompanyID == nil && filter.DepartmentID == nil {
        return nil, 0, httpx.NewError(httpx.ErrValidation, "Provide a company_id or department_id filter")
    }
    if err := s.assertCoordinatorOwnsFilter(ctx, actor, filter); err != nil { return nil, 0, err }
case identity.RoleStudent:
    ... forces filter.DepartmentID, defaults Status=open ...
}
```

Same shape in `review.ListMonitors` (`review/service.go:51-64`) and `content.ListNews`
(`content/service.go:56-66`, though content forces the filter unconditionally for non-admin
rather than validating a caller-passed one).

**Gotcha caught and fixed in this codebase** (`vacancy/service.go:374-380`,
`assertCoordinatorOwnsFilter`): a coordinator-supplied `department_id` filter must be resolved
and compared against the coordinator's own school explicitly — nothing upstream guarantees the
`department_id` came from an already-school-scoped lookup. An earlier/naive version that only
checked "does this coordinator have *a* school" (without resolving where the filter's
`department_id` actually points) would leak cross-school data. Read this method before writing
any new coordinator-scoped list filter.

## Response/error envelope + Postgres error translation

Every write in every service ends its error path through `translateWriteErr`, defined
identically in each module (`orgs/service.go:409-417`, `vacancy/service.go:471-479`,
`scoring/service.go:330-338`, `review/service.go:264-272`):

```go
func translateWriteErr(err error) error {
    if err == nil { return nil }
    if apiErr := postgres.TranslateError(err); apiErr != nil { return apiErr }
    return err
}
```

`postgres.TranslateError` (`apps/api/internal/platform/postgres/errors.go:31-49`) inspects a
`*pgconn.PgError` (via `errors.As`) and maps Postgres SQLSTATE codes:

| Code | Meaning | Mapped to |
|---|---|---|
| `23505` unique_violation | duplicate | 409 CONFLICT "This record already exists." |
| `23503` foreign_key_violation | dangling FK on insert/update | 409 CONFLICT "still referenced..." |
| `23001` restrict_violation | `ON DELETE RESTRICT` blocked a delete | 409 CONFLICT "still referenced..." (same message as FK) |
| `23514` check_violation | CHECK constraint failed | 422 VALIDATION_ERROR "Invalid input." |
| `23P01` exclusion_violation | GiST/GIN EXCLUDE (e.g. overlapping `intern_dates`) | 409 CONFLICT "conflicts with another..." |

Returns `nil` (not a translated error) for anything else, so the caller falls through to
`httpx.FailFromErr`'s generic-500-plus-log path. Comment at `errors.go:16-19` notes the
restrict-vs-FK code split was **confirmed live against this schema**, not assumed from docs —
if you see a raw `23503` where you expected `23001` (or vice versa) on a delete, that's expected
per-constraint-type Postgres behavior, not a bug.

Several services additionally **pre-check** child-row counts before attempting a delete, purely
to produce a specific human-readable conflict message instead of the generic FK one — e.g.
`orgs.Service.DeleteSchool`/`DeleteDepartment`/`DeleteCompany` (`orgs/service.go:69-84,
131-156, 309-329`) call `CountXByY` repo methods first and build a message like *"This school
still has 3 departments using it — remove or reassign them first"* via
`conflictStillReferenced`/`pluralize`/`joinBlockers` (`orgs/service.go:360-400`). This is a
UX nicety layered *on top of* the FK constraint, not a replacement for it — the DB constraint
still fires (and `translateWriteErr` still catches it) if the pre-check race loses.

## Migrations (`apps/api/migrations`, golang-migrate)

- Confirmed via `postgres.go:1-5` doc comment and by reading migration files directly: **schema
  is owned entirely by golang-migrate SQL files. `AutoMigrate` is never called** — `grep
  AutoMigrate` across `apps/api` returns only that doc-comment mention, no actual call.
  `postgres.Open` sets `DisableForeignKeyConstraintWhenMigrating: true` specifically because
  GORM never creates/alters the schema, only queries it.
- Sequential numeric prefix, up/down pairs: `000001_enable_extensions.{up,down}.sql` through
  `000026_add_presences_date_index.{up,down}.sql` (26 migrations as of writing). Each pair is
  its own two files, `-seq`-numbered per `make migrate-create name=...`
  (`Makefile:20-21`, uses `migrate create -ext sql -dir apps/api/migrations -seq $(name)`).
- Down migrations are simple/destructive by design — e.g.
  `000012_create_appliances.down.sql` is literally `DROP TABLE IF EXISTS appliances;`
  (verified). Don't expect down migrations to preserve data; they're for local dev rollback,
  not production data recovery.
- Up migrations carry real constraints GORM never would infer on its own: partial unique
  indexes (`uq_appliances_active_per_user_vacancy ... WHERE status IN ('pending','processed',
  'accepted')`, `000012_create_appliances.up.sql:18-20` — this is what makes
  `vacancy.Service.Apply`'s "one active application" rule a DB-level guarantee, not just an
  app-level check), `updated_at` triggers (`trg_..._set_updated_at`), and CHECK constraints
  (e.g. review's reviewee-exactly-one-of-user-or-company CHECK, referenced in
  `review/domain.go:42-44` as migration 000025).
- Run via `make migrate-up`/`make migrate-down` (Makefile), which shell out to the `migrate`
  CLI directly against `DATABASE_URL` — not invoked from Go code at app startup.

## Caching (`apps/api/internal/platform/cachex`)

`cachex.GetOrSet[T any](ctx, rdb *redis.Client, key string, ttl time.Duration, compute func()
(T, error)) (T, error)` (`cachex.go:20-35`) — generic get-or-compute over Redis with JSON
marshaling.

- **On a cache hit**: `rdb.Get` succeeds AND `json.Unmarshal` succeeds -> return cached value,
  `compute` is never called.
- **On a miss (including a Redis error or a corrupt cached value)**: calls `compute()`. If
  `compute` itself errors, **`GetOrSet` returns immediately without writing to Redis** —
  `cachex.go:27-30`, the `if err != nil { return out, err }` is before the `Marshal`/`Set` call.
  This is the single most important behavioral detail: **an error result is never cached**, so
  a transient DB failure doesn't get "stuck" serving errors for the TTL window; the next
  request just retries `compute`.
- On success, the Redis write is best-effort (`rdb.Set(...)`, error ignored) — a Redis write
  failure doesn't fail the request that just computed the value.
- **Only current usage**: `vacancy.Service.ApplianceStatusCounts` /
  `VacancyStatusCounts` (`vacancy/service.go:440-462`), backing the dashboard status-breakdown
  charts. TTL is `statusCountsCacheTTL = 60 * time.Second` (`vacancy/service.go:20`), chosen
  per the doc comment as "short enough to feel live, long enough that a busy admin dashboard
  doesn't re-scan the table on every page load." Cache key is scope-qualified
  (`statusCountsCacheKey`, `vacancy/service.go:427-436`) — `cache:appliance-status-counts:school:5`
  or `:company:12` or the bare base key for admin's unscoped view — specifically so one
  coordinator's or mentor's cached numbers can never leak into another's via a shared key.
  No other module currently uses `cachex`; if you add a second cached query, follow this same
  scope-qualify-the-key discipline before reusing the pattern.

## Testing convention (verified by reading test files across 8 modules, not inferred)

Line counts: `content` 125, `identity` (domain 85 + service 322 + tokens 42), `internship`
(domain 95 + integration 167 + service 214), `orgs` 127, `reporting` (domain 63, no service
test), `review` 83, `scoring` (domain 67 + service 159), `vacancy` (domain 72 + service 152).
Every module has at minimum a `domain_test.go`; most also have `service_test.go`.

**What gets a real unit test:**

1. **Pure domain functions/methods** — no DB, no mocks needed. E.g.
   `vacancy.Appliance.CanTransitionTo` exhaustively table-tested over every from/to pair
   (`vacancy/domain_test.go`, all 14 transition cases), `scoring`'s `Average`/`ResolvePredicate`
   pure functions (`scoring/domain_test.go`), `review.AverageRating`.
2. **Authorization-gate logic that sits behind an interface field**, by faking the interface —
   never a real `*gorm.DB`. The recurring pattern (near-identical across `identity`, `vacancy`,
   `orgs`, `scoring`, `review` service tests):
   - A tiny `fakeCompanyScope{schoolID, departmentID int64; err error}` struct implementing
     `CompanyScopeResolver`, returning fixed values (`vacancy/service_test.go:19-30`, same shape
     duplicated per-module rather than shared).
   - `newTestService(...)` passes `nil` for the concrete `*Repository` field whenever the test
     path never touches it — the code comment is explicit about *why* (`vacancy/service_test.go:32-38`):
     "repo is deliberately nil... A test relying on repo behavior would panic here, which is
     exactly the point... fail loudly, don't silently no-op." Same reasoning repeated in
     `identity/service_test.go:13-16`'s `fakeListUsersRepo` (embeds a nil `Repository`
     interface, overrides only the one method under test).
   - Tests then call the service method directly (including **unexported** helpers like
     `assertCanManageCompany`, `assertCanViewCompany`, `assertCoordinatorOwnsFilter`,
     `canManageSchool`, `actorInSchool` — these are tested in-package, not just through the
     exported surface) with different `actor.Role`/`actor.SchoolID`/`actor.CompanyID`
     combinations and assert on the returned `httpx.ErrorCode` via a shared `requireAPIErr`
     helper (redefined per test file, same body every time).

**What does NOT get a unit test:**

- Concrete `*Repository` methods backed by a real `*gorm.DB` are **not unit tested** anywhere in
  this codebase — there is no sqlmock, no in-memory sqlite, no repository-level mocking. The
  only place a real database is exercised is:
- `internship/integration_test.go` — the **sole** integration test in the repo, gated behind
  `//go:build integration` (line 1) so `go test ./...` (what CI / `make test-api` runs) never
  touches it. It spins up a real Postgres via `testcontainers-go` (`postgres.Run(ctx,
  "postgres:16-alpine", ...)`, `integration_test.go:38`), applies the **actual** migrations
  from `apps/api/migrations` via `golang-migrate` pointed at
  `filepath.Join(...,"..","..","..","migrations")` resolved from `runtime.Caller(0)` (not CWD —
  works regardless of where `go test` is invoked from, `integration_test.go:29-32`), and only
  then runs repository-level assertions against it. Run explicitly via `make test-integration`
  (needs Docker).
- Handlers (`handler.go`) have **no tests** in any module — no `httptest`/`gin.Context`-based
  handler tests found anywhere under `apps/api/internal/modules`.

**If you're asked to add a test**: a new pure function or a new authorization-gate branch in a
service method gets a same-shape unit test (fake the relevant interface, assert on
`httpx.ErrorCode`). A new/changed repository query does **not** get a unit test in this
codebase's existing convention — it would only be covered if you extend
`internship/integration_test.go`'s testcontainer suite, and that's the only place doing so
would be idiomatic here. Don't invent a sqlmock-based repository test; it doesn't match this
codebase's actual practice.

## Other things worth knowing before touching this code

- `middleware.RequireAuth` (`middleware/auth.go:22-33`) is the **only** place a session cookie
  is read; it puts `*identity.User` on the Gin context once, and every downstream
  handler/service calls `middleware.CurrentUser(c)` — never re-reads the cookie.
  `authed := api.Group("")` in `server/server.go:74-84` is the single mount point carrying
  `RequireAuth` + `RequireCSRF`; every authenticated module's `RegisterRoutes` hangs off that
  one group.
- `middleware.RequireRole` exists but is **dead code** (see RBAC section above) — don't assume
  route-level role gating exists; check the service method.
- Auth rate limiting is shared across login/register/forgot-password/reset-password under one
  Redis-backed limiter keyed by `AuthRateLimitKey`, 10 requests / 5 minutes
  (`server/server.go:59-70`) — the comment there explains the 10-not-5 number was raised after
  the E2E suite's own legitimate multi-login flows tripped a tighter limit, so don't casually
  lower it without checking `apps/e2e` first.
- GORM struct tags always match the migration's column names exactly and every domain struct
  declares `TableName()` explicitly (GORM's pluralization-guessing is never relied upon).
- `notification`/`content` news fan-out and password-reset email both go through the `asynq`
  queue via `queue.Enqueuer`, wired through `queueNotifierAdapter`/`queueMailerAdapter` in
  `main.go:35-70` — notification/email sends are never inline in the request path.
