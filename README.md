# Internity: SMK PKL/Internship Management Platform

> Take-home submission for PT Mumtaz Teknologi Indonesia (Fullstack Engineer).

## Overview

Internity digitizes *PKL* (Praktik Kerja Lapangan), the mandatory internship program every
Indonesian vocational high school (SMK) runs. Schools publish internship vacancies through
partner companies, students apply and get placed, track daily attendance and work journals, get
scored by their on-site mentor, and receive a certificate at the end.

It's a rebuild of a real system I shipped previously for a live school (legacy stack: Laravel,
React, Nuxt, at `D:\Project\Internity`, kept read-only here as the domain reference). This
rewrite uses the take-home's required stack instead: Go, Gin, GORM, PostgreSQL, and Redis for the
backend; Vue 3, shadcn-vue, and Tailwind for the dashboard; Nuxt for the landing page. Same
domain, but the data modeling and security gaps the legacy system had accumulated are fixed this
time (see [Technical Decisions & Trade-offs](#technical-decisions--trade-offs)).

## Problem & Target Users

Manually tracking a few hundred students across dozens of partner companies with spreadsheets and
WhatsApp groups is slow and error-prone for the school, and opaque for the student. Four actors,
one system:

- **Admin.** Platform owner, manages schools onboarded to the platform.
- **Coordinator.** School/department staff (teacher, department head, workshop head in the legacy
  system, merged into one role here) who manage students, review applications, and approve
  attendance/journals.
- **Mentor.** The company-side supervisor who scores students and approves their
  attendance/journal at that company.
- **Student.** The intern: applies to vacancies, checks in/out daily, writes journals, downloads
  their certificate.

## Features

**Auth & account.** Cookie-session login/register (with school invite codes), forgot/reset
password, profile edit and avatar upload, change password (revokes all other sessions).

**Org management (admin/coordinator).** Schools, departments, courses, and companies CRUD; user
management; presence-status and score-predicate configuration per school.

**Vacancies & applications.** Staff post vacancies per company; students browse, search, save, and
apply. The appliance state machine (`pending → processed → accepted/rejected`,
`pending → canceled`) fans out a notification on every transition, and staff set a student's
intern dates once accepted.

**Attendance & journals.** Daily check-in/out with photo and geolocation, excuse requests, and
daily work journals, all created on the fly rather than pre-materialized. A derived
attendance-summary view (`reported`/`missing`/`upcoming`/`outside_range`) shows each student's
status per day, and staff can bulk-approve presence/journal entries.

**Scoring & certificate.** Mentors and staff enter per-category scores; the letter predicate is
derived from the school's configurable min/max bands. Certificate generation is idempotent and
renders a real PDF in pure Go, no external binary.

**Monitoring & reviews.** Coordinators log monitoring visits to partner companies. Mentors rate a
student's performance and students rate the company they interned at, each a 1-to-5 star rating
with an optional title and comment.

**Content.** News (school/company/global scoped) and FAQ, both admin-managed and publicly
readable. The landing page's FAQ fetches live from the API.

**Reporting.** Excel/PDF export of rosters, presence, and journals.

**Onboarding tours.** Each role gets a first-login guided tour ([driver.js](https://driverjs.com/)),
written as role-specific user stories (for example, *"As a student, I want to apply to a vacancy
so I can start my internship"*), dismissible and replayable from the user menu.

**Landing page.** A statically generated Nuxt marketing site with hero, features, CTA, a
live-fetched FAQ, and a privacy policy. I looked at [joi.software](https://joi.software/) for the
restraint I wanted: generous whitespace, large confident type, no illustration clutter, while
keeping Internity's own blue/teal palette (the same design tokens the dashboard uses).

## Tech stack

| Layer | Choice |
|---|---|
| Backend | Go, Gin, GORM (query-builder only), PostgreSQL, Redis, MinIO (object storage) |
| Dashboard | Vue 3, [shadcn-vue](https://www.shadcn-vue.com/), Tailwind CSS v4, Pinia, TanStack Query, vee-validate + zod, driver.js |
| Landing | Nuxt 3 (SSG) |
| Shared | `packages/design-tokens`, single source of truth for both frontends' theme |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) (raw SQL, up/down pairs) |
| Testing | Go `testing` (unit) · Vitest + Vue Test Utils (dashboard) · `nuxi typecheck` (landing) |

## Architecture

Monorepo, pnpm workspaces plus a single Go module:

```
apps/
  api/        Go, package-by-feature modules (internal/modules/*)
  dashboard/  Vue 3 + shadcn-vue, one RBAC-gated app for all 4 roles
  landing/    Nuxt 3, marketing site (SSG)
packages/
  design-tokens/  shared color/type/spacing tokens -> Tailwind v4 theme for both frontends
deploy/
  docker-compose.yml  postgres, redis, minio(+init), migrate, api, worker, dashboard, landing
```

**Backend modules** (`apps/api/internal/modules/`): `identity` (auth/sessions/users/invite
codes), `orgs` (schools/departments/courses/companies), `vacancy` (vacancies/saved
vacancies/appliances), `internship` (intern dates/presence/journal/presence statuses), `scoring`
(scores/predicates/certificates), `review` (monitors/questions/reviews), `content` (news/faq),
`notification`, `reporting` (exports). Each module owns
`domain.go`/`repository.go`/`service.go`/`handler.go`/`routes.go`. Cross-module calls go through
narrow interfaces and adapters wired in `cmd/api/main.go`; no module reaches into another
module's database tables directly.

**Auth** is an opaque, cookie-based session, not a JWT: a short-lived access cookie plus a
long-lived refresh cookie rotated on every use. Reusing an already-rotated refresh token revokes
the whole session family. A non-`httpOnly` CSRF cookie gets echoed back as `X-CSRF-Token` on
mutating requests (the double-submit pattern), enforced by `RequireCSRF` middleware on every
authenticated route.

**RBAC** is a flat `role` enum (`admin`/`coordinator`/`mentor`/`student`) plus nullable scope
columns (`school_id`/`department_id`/`company_id`/`course_id`), tied together by a DB `CHECK`
constraint so a coordinator can't exist without a school, or a mentor without a company. Service
methods also re-check scope on their own, not just at the route middleware.

**Response envelope.** Every endpoint returns
`{success, data, message, meta:{request_id, pagination?}}` on success, or
`{success:false, data:null, message, error:{code, details}, meta}` on failure, through a fixed
error-code taxonomy (`VALIDATION_ERROR`→422, `UNAUTHENTICATED`→401, `FORBIDDEN`→403,
`NOT_FOUND`→404, `CONFLICT`→409, `RATE_LIMITED`→429, `INTERNAL_ERROR`→500).

**Redis** backs the `/readyz` health check and a fixed-window rate limiter
(`internal/middleware/ratelimit.go`) on `/auth/*` (login/register/refresh/forgot-password/
reset-password), keyed by IP and email, 10 requests per 5 minutes. That's the brute-force and
credential-stuffing protection the take-home spec calls out as a bonus. The limit started at 5 and
moved to 10 after live E2E runs showed a normal multi-step session on one account (switching
between the dashboard's role-scoped views a few times) could hit 5 on its own, well before any
actual abuse. If Redis is unreachable the limiter degrades to "allow" rather than taking auth down
with it; it's defense-in-depth, not the primary boundary.

**MinIO** stores avatars, attachments, documents, and logos in separate buckets. Uploaded files
are always renamed to `{uuid}.{sniffed-extension}`, and the extension comes from sniffing the
file's actual bytes, never the client-supplied filename or `Content-Type` header.

## Database design

25 migrations live in `apps/api/migrations/` as golang-migrate raw SQL. I picked that over GORM's
`AutoMigrate` because the take-home asks for real, reviewable down-migrations, and `AutoMigrate`
can't produce those. A few modeling decisions worth calling out:

- `users` carries `role` plus nullable `school_id`/`department_id`/`company_id`/`course_id`,
  guarded by a `CHECK` constraint. This replaces the legacy system's `school_user`/
  `department_user`/`company_user` pivot tables, which were often queried with a bare `->first()`
  that silently picked an arbitrary row whenever a user had more than one.
- `intern_dates` uses a Postgres `EXCLUDE` constraint (`btree_gist`,
  `EXCLUDE USING gist (user_id WITH =, daterange(start_date, end_date, '[]') WITH &&)`) so a
  student's placements can never overlap in time. The database enforces it, not an ad hoc
  application check, and a `version` column handles optimistic locking on concurrent edits.
- `presences`/`journals` have no bulk pre-materialized rows. A row exists only once the student
  actually acts, and a read-time `attendance-summary` query (date range times `generate_series`
  left join) classifies each day without needing placeholder rows.
- `notifications` always materializes one row per recipient at write time. That fixes a latent
  legacy bug where a nullable-`user_id` "broadcast" notification couldn't track per-user read
  state.
- `appliances` has a partial unique index enforcing one active (non-terminal) application per
  `(user_id, vacancy_id)`.

Every domain struct now carries explicit `json:"snake_case"` tags matching the request-binding
DTOs. An early version of this codebase didn't, so nearly every list/detail response came back
PascalCase while every request body was snake_case. Fixed at the source instead of papering over
it in the frontend.

## Getting started

### Prerequisites
- Docker + Docker Compose
- Go 1.23+, Node 20+, pnpm 10+ (only needed for local development outside Docker)
- [golang-migrate CLI](https://github.com/golang-migrate/migrate#installation) (only needed to
  run `make migrate-*` outside Docker)

### Environment
```
cp .env.example .env
```

### Run everything
```
make dev
```
Boots Postgres, Redis, MinIO (plus bucket init), runs migrations, then starts the API, worker,
dashboard (http://localhost:5173) and landing (http://localhost:3000) containers. Caveat: the
`docker compose up` path itself hasn't been run in this environment, which never had Docker
installed. Everything it wires together has been run for real, though, just outside a container:
the API and dashboard against a live Postgres and Redis, driven end to end by the Playwright suite
below. Flag anything that doesn't come up clean on first run.

### Migrations
```
make migrate-up                        # apply all pending migrations
make migrate-down                      # roll back the most recent one
make migrate-create name=add_foo_table
```

### Seeding demo data
```
make seed
```
Idempotent, safe to re-run. Seeds one school (SMKN 1 Cibinong), one department, two courses, two
companies, one user per role plus two students, presence statuses, score predicates, two
vacancies, and a student self-registration invite code (`RPL1DEMO`). All seeded accounts share the
password `password123`:

| Email | Role |
|---|---|
| admin@internity.test | admin |
| coordinator@internity.test | coordinator |
| mentor1@internity.test | mentor (PT Mumtaz Teknologi Indonesia) |
| mentor2@internity.test | mentor (PT Teknologi Nusantara) |
| budi@internity.test | student |
| siti@internity.test | student |

## Testing

```
make test-api          # Go unit tests
make test-dashboard     # Vitest component/composable tests
make test-e2e           # Playwright critical-path E2E (needs `make dev` + `make seed` running first)
```

**Unit tests.** 104 Go tests cover the state-machine transition guards (appliance, intern-date),
scope checks, Postgres error translation, and DTO validation as pure-function tests. 164 Vitest
tests cover the dashboard's shared UI primitives (`components/shared/*`), composables
(`useListQuery`, `useTour`), the auth store, and the `lib/` helpers (status mapping, nav
filtering, the axios instance's CSRF and single-flight-refresh interceptors) via
`@vue/test-utils` and mocked dependencies. The dashboard's real type-check command is
`vue-tsc --build` (via `pnpm run type-check`); a plain `vue-tsc --noEmit` silently no-ops in this
project's TS project-references setup, so use `--build` when verifying.

**Integration tests were designed but not run against `testcontainers-go`.** Repository/
service-layer tests against a disposable, containerized Postgres were planned, but this
environment has no Docker to run them in. The E2E suite below fills a similar gap in practice: it
drives real HTTP requests through the real service and repository layers against a live database,
just not through `testcontainers-go` specifically.

**E2E runs against a live stack, and it's what actually found most of the bugs listed below.**
Three Playwright spec files cover the full app, not just one happy path:
`critical-path.spec.ts` (login, browse/apply to a vacancy, staff accepts, student self-schedules
their internship dates, checks in, writes a journal, staff approves both and scores it, student
downloads a certificate), `business-flows.spec.ts` (rejected and self-canceled applications,
filing and approving an attendance excuse, the mentor/student review exchange, auth edge cases
like a wrong password or an expired reset link), and `admin-crud.spec.ts` (every admin/coordinator
management screen, plus a coordinator generating a student invite code). All three ran repeatedly
against the live API and dashboard, talking to the same remote Postgres and Redis this project
already uses for development, not a mocked or seeded-once snapshot. That surfaced and fixed real
problems no code read would have caught on its own:

- The news-publish endpoint fanned out a notification to every user in scope synchronously before
  responding, so publishing to a school with over a hundred students could take 30-40 seconds.
  Fixed by moving the fan-out to its own goroutine off the request path.
- A leaked idle-in-transaction Postgres connection (from an interrupted `cmd/seed` run) was
  silently holding locks that made unrelated writes hang. Killing it, and separately confirming
  the app itself never opens a transaction it doesn't close, ruled out the app as the cause.
- `postgres.TranslateError` recognized unique and foreign-key violations but not exclusion
  violations, so the intern-dates overlap check surfaced as a raw 500 instead of a clean 409.
  Covered by a new table-driven test in `errors_test.go`.
- A guest's first visit to `/register`, `/forgot-password`, or `/reset-password` was bouncing
  straight back to `/login`. The dashboard's axios interceptor treated the router guard's own
  expected 401 from `/auth/me` the same as a real session expiry.
- A Driver.js onboarding tour mounts asynchronously after login; checking `isVisible()`
  immediately after redirect could lose that race and leave an invisible overlay blocking clicks
  on whatever page loaded next. All three spec files now wait for the tour to actually appear
  before deciding whether to dismiss it.
- The suite's own login volume (three role switches in the critical path alone, times three spec
  files run back to back) could exhaust the auth rate limit purely from legitimate use, which is
  what led to raising it from 5 to 10 above.

The one thing E2E surfaced that isn't a bug: the original plan assumed a staff-facing "set
internship dates" screen. The only such screen (`MyInternshipView.vue`) is student-only.
Accepting an application creates the `intern_dates` row with no dates set, and the student fills
them in themselves. The spec follows what's actually implemented, not the original plan.

Run `make test-e2e` (needs `make dev` + `make seed` up first) or the on-demand `ci-e2e` GitHub
Actions workflow.

## API documentation

`docs/openapi.yaml` is an OpenAPI 3.1 spec covering the API's endpoints (auth, orgs, vacancies,
internship, scoring, review, content, notifications, reporting). View it in any Swagger/Redoc UI,
or at https://editor.swagger.io/ by pasting the file contents.

## Technical decisions & trade-offs

- **Cookie sessions over JWT.** Logout, forced logout, and password change all need instant
  server-side revocation. A bare JWT would need a denylist to do that anyway, so a session table
  is simpler for a cookie-authenticated SPA talking to a same-organization API.
- **Double-submit CSRF over synchronizer tokens.** There's no server-rendered form to embed a
  token into, since this is a JSON SPA, so double-submit is the standard fit. It works alongside
  `SameSite=Lax` rather than replacing it.
- **On-the-fly presence/journal rows over pre-materialization.** The legacy system bulk-inserted a
  blank row per day across a student's whole placement range up front. That's write
  amplification, it has awkward edge cases when a placement gets shortened, and it conflates "no
  data yet" with a fake "Pending" status row. Creating rows only when the student acts, plus a
  derived summary view, avoids all three, at the cost of one read-time aggregation query.
- **`EXCLUDE` constraint over application-level overlap checks.** The legacy system's
  multi-company overlap rule was one ad hoc query in one controller method. Moving it into a
  database constraint means no future code path (a worker job, a bulk-edit endpoint, a script) can
  bypass it. The rule holds structurally instead of by discipline.
- **golang-migrate over GORM `AutoMigrate`.** The take-home asks for real migrations with working
  down-migrations. `AutoMigrate` only moves a schema forward and can't express a drop or rename
  safely.
- **4-role RBAC over the legacy system's 6 overlapping ones.** Manager, kaprog (head of study
  program), and kepala bengkel (workshop head) all resolved to "school staff with
  department-level scope" in every flow that mattered. Collapsing them into `coordinator` removes
  three near-duplicate code paths without losing any real capability; the only thing lost is a
  distinction the legacy UI drew that no business rule ever used.
- **Fixed-window Redis rate limiting over a token bucket or sliding window.** One `INCR`+`EXPIRE`
  round trip is enough for what this guards against (login, register, forgot-password). A
  sliding window or token bucket would need a Lua script for precision this doesn't need at this
  scale.
- **SSG landing page.** It has no per-request personalization, so static generation means zero
  Node runtime cost in production, at the cost of a rebuild (not just a redeploy) whenever the
  copy changes. FAQ content, the one thing that does change without a deploy, is fetched
  client-side instead of baked in.
- **Asynq queue covers notifications and email, not exports yet.** `cmd/worker` runs a real
  `asynq.Server`; the API enqueues two job types (`notification:send`, `email:password_reset`)
  instead of doing that work inline, so a request returns before a fan-out to many recipients
  finishes writing rows. Excel/PDF export generation still runs synchronously in the request path,
  which is the remaining gap against the original plan (see Future improvements). Email delivery
  itself is still a logged no-op (`identity.NoopMailer`) since no SMTP/provider is configured. What
  moved is *where* that call happens, the worker instead of the request path, not what it does.
  Wiring a real provider is a one-line swap in `cmd/worker/main.go`.

## Known limitations

Noted deliberately, not hidden, so expectations are clear going into a demo or review:

- There's no admin screen to create a new coordinator or mentor account. Those are still created
  through `make seed` or a direct database insert.
- A coordinator can't see the mentors in their own school from the Users page. Mentor accounts are
  scoped by `company_id`, and that list query never joins back to `school_id` to filter by it.
  Confirmed live during E2E testing: a coordinator session calling `GET /users?role=mentor` gets
  an empty result even for a mentor at a company genuinely in that coordinator's school, because
  `ListUsers` forces a `school_id` filter onto every coordinator request and mentors don't have
  one.
- A company can only belong to one school/department. There's no many-to-many relationship for a
  company that genuinely partners with more than one school.

## Future improvements

- Route PDF/Excel export generation and a cron-style internship-end-date reminder through the
  Asynq queue too, the way notification fan-out and password-reset email already are.
- Integration tests against a real, containerized Postgres via `testcontainers-go`. Designed and
  ready, just blocked here by the lack of Docker in this environment.
- OpenTelemetry tracing across the API and background jobs.
- Upload malware/AV scanning on MinIO objects before they're served back.
- A CD pipeline and live hosting. Out of scope by design for this submission (the deployment
  decision made at project kickoff was local `docker compose` only).
