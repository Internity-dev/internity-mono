# Internity domain model — PKL (vocational internship) management

Read this before touching business logic. It documents the *business domain*, not
the code architecture. All claims are verified against the actual Go source —
file:line citations let you jump straight to the real definition instead of
re-grepping.

The take-home spec (`Take Home Test Fullstack - PT Mumtaz Teknologi Indonesia.pdf`,
repo root) leaves the application domain free — it only mandates the
non-functional shape (Go/Gin/GORM/Postgres backend, Vue/React+Tailwind frontend,
cookie auth, RBAC, versioned REST, etc.), explicitly requiring "problem yang
jelas, target user yang jelas, business flow, business rules" rather than plain
CRUD. This repo's chosen domain is **PKL** — Praktik Kerja Lapangan, the
mandatory Indonesian vocational-high-school (SMK) internship program. Target
users: school coordinators placing students, company mentors supervising them,
and students doing the internship. This doc is about *that* domain; the
take-home spec itself has no PKL-specific requirements to cross-check against.

## 1. Org hierarchy and cardinality

```
School 1─┬─* Department ─┬─* Course
         │                └─* Company
         └─(transitively, via Department)─* Company, Course
```

Defined in `apps/api/internal/modules/orgs/domain.go`:
- `School` (7-67): standalone, no parent.
- `Department.SchoolID` (22-34): belongs to exactly one School.
- `Course.DepartmentID` (36-46): belongs to exactly one Department.
- `Company.DepartmentID` (48-67): belongs to exactly one Department — **not**
  directly to a School. Verified both in the struct field and the migration:
  `apps/api/migrations/000005_create_companies.up.sql:3` —
  `department_id BIGINT NOT NULL REFERENCES departments (id) ON DELETE RESTRICT`.
  There is no `school_id` column on `companies` at all.

**Why this matters for scoping:** a coordinator's identity row only carries
`SchoolID` (see §2). To authorize anything company-scoped (vacancies,
placements, presence, journals, scores, monitors), the backend has to resolve
`company -> department -> school` on every check. This join is centralized in
one place — `orgs.Repository.ResolveCompanyScope`
(`apps/api/internal/modules/orgs/repository.go:250-263`):

```go
Table("companies").
    Select("companies.id AS company_id, companies.department_id AS department_id, departments.school_id AS school_id").
    Joins("JOIN departments ON departments.id = companies.department_id").
    Where("companies.id = ?", companyID)
```

Every module that needs to scope-check a company against a coordinator's
school (vacancy, internship, scoring, content, review) depends on a narrow
`CompanyScopeResolver` interface satisfied by this one method via an adapter
wired in `apps/api/cmd/api/main.go` — never imports `orgs.Repository` directly
(hexagonal-ish port pattern used throughout). A coordinator is "school-scoped"
in the DB, but everywhere in code that translates to "every department in that
school, and every company/course/vacancy/placement under those departments" —
never a direct FK filter.

## 2. Roles and `identity.User` scope columns

Defined in `apps/api/internal/modules/identity/domain.go:8-59`. One `users`
table, one `Role` enum, four *nullable* scope FKs (`SchoolID`, `DepartmentID`,
`CompanyID`, `CourseID`) — which ones are non-nil depends entirely on role.
Verified against both the seed script (`apps/api/cmd/seed/main.go`, calls to
`upsertUser`) and the real self-registration path
(`identity.Service.Register`, `apps/api/internal/modules/identity/service.go:86-135`):

| Role | SchoolID | DepartmentID | CompanyID | CourseID | Set by |
|---|---|---|---|---|---|
| `admin` | nil | nil | nil | nil | seed only (`main.go:733`) — platform-wide, no scope at all |
| `coordinator` | **set** | nil | nil | nil | seed (`main.go:736`) |
| `mentor` | nil | nil | **set** | nil | seed (`main.go:746`) |
| `student` | **set** | **set** | nil | **set** | seed (`main.go:808`) *and* real registration (`service.go:124-134`, via invite code -> `CourseScope`) |

Gotcha: **a student's `User.CompanyID` is never set, ever** — not at
registration, not when their appliance is accepted, not for the duration of
their placement. Grep confirms no write path touches it. A student's current
company affiliation lives *only* in the `intern_dates` table (§4) and must be
looked up via `internship.Repository.GetByUserCompany` /
`ListForUser` — never assume `actor.CompanyID` is populated for a student, it
always reads nil.

`InviteCode.CourseID` (`identity/domain.go:106-118`) is how self-registration
resolves a student's School/Department/Course in one shot —
`identity.Repository.ResolveCourseScope(courseID)` returns a `CourseScope{CourseID,
DepartmentID, SchoolID}` by walking the FK chain, so a coordinator only has to
generate a code per-course (`CreateInviteCode`, `service.go:320-353`) and
students never pick their own school/department.

Role-based access control is **not** enforced by router middleware in this
codebase, even though `middleware.RequireRole` exists
(`apps/api/internal/middleware/rbac.go:16`) — grepping the whole `apps/api`
tree shows it is defined but never called from any route registration. Every
module's `routes.go` just says "must be attached under a group that already
carries `RequireAuth`" (e.g. `review/routes.go:5-7`) and all real
authorization — role checks *and* scope checks — happens inline in each
`Service` method by switching on `actor.Role` and comparing scope columns
(see the `assertCanManageCompany` / `assertCanManageSchool` helpers repeated
with small variations across `vacancy/service.go`, `internship/service.go`,
`scoring/service.go`, `review/service.go`). **When adding a new endpoint, the
authorization has to be written by hand in the service method — there is no
declarative role gate to lean on, and copying an existing sibling method's
`assertCan*` call is the expected pattern.**

## 3. Student lifecycle, end to end

Real sequence, module/table at each step:

1. **Browse** — `vacancy.Vacancy` (`vacancy/domain.go:18-31`), scoped to the
   student's own `DepartmentID` and (by default) `status = open`
   (`vacancy.Service.ListVacancies`, `vacancy/service.go:75-84`).
2. **Apply** — `vacancy.Service.Apply` (`service.go:170-205`) creates an
   `Appliance` row (`vacancy/domain.go:62-78`) with `Status: StatusPending`.
   Guarded: actor must be a student, vacancy must be open, and the vacancy's
   company must resolve to the student's own department
   (`vacancyDept != *actor.DepartmentID` -> 403). One "active" application per
   (user, vacancy) is enforced by a partial unique index at the DB level, not
   pre-checked in app code — a race-safe 409 on conflict instead of
   check-then-insert.
3. **State machine** — `Appliance.CanTransitionTo`
   (`vacancy/domain.go:57-93`): `pending -> {processed, accepted, rejected,
   canceled}`, `processed -> {accepted, rejected, canceled}`; accepted /
   rejected / canceled are terminal (no map entry = no outgoing edges).
   `Process` (mentor/coordinator marks "under review") is **optional** —
   pending can jump straight to `accepted` without ever passing through
   `processed`. `Reject`/`Process`/`Accept` all gate through
   `assertCanManageCompany` (mentor: own company only; coordinator: own
   school, resolved via `ResolveCompanyScope`; admin: unrestricted).
4. **Accept** (`vacancy.Service.Accept`, `service.go:243-274`) — the one
   transition with side effects: checks `CountAcceptedForVacancy < v.Slots`
   (409 if full), flips the appliance to `accepted`, then calls
   `InternshipScheduler.ScheduleForAcceptedAppliance` — a narrow
   cross-module port into `internship.Service.ScheduleForAcceptedAppliance`
   (`internship/service.go:44-63`), which creates one **`InternDate`** row
   (`internship/domain.go:22-36`) with `Status: StatusScheduled`,
   `StartDate`/`EndDate` both nil ("accepted, unscheduled" — placement exists
   but has no dates until someone calls `SetDates`). `intern_dates` has
   `UNIQUE (user_id, company_id)` (migration `000013`) — a student can only
   ever hold one placement per company, ever, even across different
   appliances; violating this surfaces as "This student already has a
   placement at this company" (409). A GiST exclusion constraint
   (`excl_intern_dates_no_overlap_per_user`, same migration, lines 23-27) also
   forbids two of a student's placements (at *different* companies) from
   having overlapping date ranges, once both dates are set.
   - Gotcha: the DB CHECK on `intern_dates.status` allows `'scheduled'`,
     `'active'`, `'completed'` (migration `000013:9-10`), but the Go
     `InternDateStatus` enum (`internship/domain.go:9-14`) only defines
     `StatusScheduled`/`StatusCompleted` — `'active'` is a DB-legal value the
     application layer never writes. "Is this placement currently active" is
     a *computed* property, `InternDate.IsActiveOn(day)`
     (`domain.go:49-55`, checks `day` against `[StartDate,
     EffectiveEndDate()]`), not a stored state — there's no scheduler job
     flipping status at start/end.
5. **Daily presence** — student calls `CheckIn`/`CheckOut`/`FileExcuse`
   (`internship/service.go:201-344`), all gated on
   `placement.IsActiveOn(date)`. `CheckIn` always targets *today*
   (`truncateToDate(time.Now())`); `FileExcuse` takes a caller-supplied
   `Date` with **no min/max validation** — it only has to fall inside the
   placement's active range, so a student can file a `permitted`/`sick`
   excuse for a date in the future relative to today. Each `Presence` row
   (`domain.go:90-113`) maps to one `PresenceStatus` (§5) via
   `PresenceStatusID`; the `(user_id, company_id, date)` uniqueness is
   enforced by relying on insert-conflict (409) rather than check-then-insert,
   same race-safety pattern as Apply.
   - **Worked example of a lifecycle edge case, fixed this session**
     (commit `54353dc`): the dashboard's "today's attendance" card
     (`apps/dashboard/src/views/attendance/AttendanceView.vue`) originally
     fetched `sort=date desc, limit=1` and trusted row #1 to be today's row.
     Because `FileExcuse` has no date ceiling, a future-dated excuse (e.g.
     filed in advance for a planned absence next week) sorts ahead of today's
     real check-in under `date desc`, so the card wrongly reported "not
     checked in today" and hid the Check-out button — even with a real
     check-in on record — for as long as that future row remained the
     max-dated one. Fix (`AttendanceView.vue:70-90`): fetch a small window
     (`limit: 5`) and find the row whose `date` field actually equals
     today client-side, rather than trusting sort order to mean "today".
     **General lesson for this codebase: never assume "most recent by date"
     equals "today's row" wherever a date field lacks a max-date
     constraint — filter explicitly by the calendar date you mean.**
6. **Journal** — `internship.Service.UpsertJournal`
   (`service.go:441-476`) requires a same-day `Presence` row with
   `CheckInAt != nil` to already exist ("You can only journal a day you
   checked in") — journal entries can't precede attendance. One `Journal` row
   per (user, company, date); editable until `IsApproved`, then frozen.
7. **Mentor scores the student** — `scoring.Score`
   (`scoring/domain.go:14-25`): `(UserID, CompanyID, Name, Score 0-100, Type:
   teknis|non-teknis)`. Multiple named score line-items per placement (e.g.
   "Kedisiplinan", "Kualitas Kerja"), not one aggregate number. Gated by
   `assertCanManageCompany` — same mentor/coordinator/admin pattern as
   everywhere else.
8. **Certificate** — `scoring.Service.GenerateCertificate`
   (`scoring/service.go:180-276`). Idempotent: a `CertificateNumber`, once
   issued for a (user, company) pair, is never reissued — a re-request
   re-renders the PDF from the *current* score snapshot but keeps the same
   number (`repo.FindCertificate` before falling back to
   `NextCertificateSequence`). Preconditions actually checked: student's
   `NIS` must be non-empty, and at least one `Score` row must exist for that
   (user, company) — **there is no check that the placement itself is
   `StatusCompleted`, or that scores of both `teknis`/`non-teknis` types
   exist** — a certificate can be generated the moment a single score line
   is entered on an ongoing placement. The average
   (`scoring.Average`, `scoring/domain.go:56-65`) resolves to a letter grade
   via `ResolvePredicate` against the school's `ScorePredicate` bands (§5);
   an unmatched average silently renders as `"-"` rather than erroring.

## 4. Review: two distinct relationships that look similar but aren't

`review.Review` (`review/domain.go:45-58`) has two mutually-exclusive nullable
FKs, `RevieweeUserID` and `RevieweeCompanyID`, with a DB CHECK
(`chk_reviews_exactly_one_reviewee`, migration `000025:16-18`) enforcing
exactly one is set — a leftover-nullable-FK pattern the code comment says
replaces a legacy polymorphic column.

- **`RevieweeCompany`** — a *student* reviewing the company they interned at
  (public-facing reputation signal; `review.Service.ListReviewsForCompany` is
  deliberately open to any authenticated role, `service.go:207-211`, "company
  reputation is useful context for every actor browsing vacancies").
- **`RevieweeUser`** — a *mentor* rating a student they mentor
  (confidential, staff-only). These are not symmetric: only students write
  company reviews, only mentors write user reviews (`CreateReview`,
  `service.go:159-182`).

**Ownership rule for the mentor-rates-student case, and a real fixed bug**
(commit `3027939`): a mentor may only review/read a student who has an actual
`InternDate` placement at the mentor's own company — "mentored" is defined as
"has a placement row", not "I happen to know their UUID". Before the fix,
`ListReviewsForUser`/`CreateReview` only checked that a *student* wasn't
reading someone else's reviews; every staff role (mentor included) had **zero**
company-ownership check, unlike every sibling mentor-facing endpoint
(scoring, presences, journals all gate through `assertCanManageCompany`). A
mentor who could see a student's UUID anywhere in their own applicant queue
(exposed via a title-tooltip) could pull up or write reviews for *any*
student at *any other company* — confirmed live against the running dev
stack per the commit message.

Fix, and the reference pattern to copy for any new mentor-facing endpoint:
`review.Service.assertMentorMentorsStudent`
(`review/service.go:222-234`) checks `actor.CompanyID != nil` then calls a
narrow `PlacementChecker` port —

```go
type PlacementChecker interface {
    HasPlacementAtCompany(ctx context.Context, userID string, companyID int64) (bool, error)
}
```

— backed by `internship.Repository.GetByUserCompany` via
`placementCheckerAdapter` in `apps/api/cmd/api/main.go` (checks `ErrNotFound`
-> `false, nil`, any other error passes through). Called from both
`CreateReview` (case `RevieweeUser`, `service.go:168-170`) and
`ListReviewsForUser` (`service.go:199-203`, only for `actor.Role ==
RoleMentor` — coordinator/admin stay unrestricted, matching their broader
scope elsewhere). **When wiring a new mentor-facing read/write, don't assume
`assertCanManageCompany`'s company-ID check is enough if the target resource
is keyed by a student's user ID rather than a company ID directly — you need
the placement check, not just a company match, or you've reintroduced this
exact bug.**

## 5. Monitor, Question, PresenceStatus, ScorePredicate

- **`Monitor`** (`review/domain.go:8-22`) — a coordinator's site-visit log to
  a company: `(CoordinatorID, StudentID, CompanyID, Date, MatchRating 1-4,
  Notes, Suggest, AttachmentKey)`. Create requires coordinator/admin +
  `assertCoordinatorOwnsCompany`; delete requires admin or the *original*
  coordinator who logged it (`row.CoordinatorID != actor.ID` -> 403,
  `review/service.go:83-92`) — not just "any coordinator at that school".
- **`Question`** (`review/domain.go:24-33`) — the review questionnaire
  template, **school-scoped** (`SchoolID` field). One school's coordinator
  can't read or edit another school's questions (`ListQuestions`/
  `CreateQuestion`/etc. all gate via `assertCanManageSchool` /
  `SchoolID == *actor.SchoolID`).
- **`PresenceStatus`** (`internship/domain.go:73-86`) — per-school
  configurable attendance categories: `(SchoolID, Name, Kind, Color, Icon)`.
  `Kind` enum (`KindPresent|KindPermitted|KindSick|KindAbsent|KindHoliday`,
  domain.go:63-71) is the fixed vocabulary the backend logic switches on
  (`CheckIn` always resolves `KindPresent`; `FileExcuse` only accepts
  `KindPermitted`/`KindSick`); `Name` is the free-text school-chosen label.
  Seed's real mapping (`cmd/seed/main.go:818-820`): Hadir=present,
  Izin=permitted, Sakit=sick, Alpa=absent, Libur=holiday. A school that
  hasn't configured a given `Kind` yet gets a 409 ("This school has not
  configured a 'present' attendance status yet") rather than a silent
  fallback — `FindPresenceStatusByKind` errors as `ErrNotFound` and the
  service translates it explicitly (`internship/service.go:218-221`,
  `309-312`).
- **`ScorePredicate`** (`scoring/domain.go:27-52`) — per-school configurable
  letter-grade bands: `(SchoolID, Name, Min, Max)`. `ResolvePredicate`
  (`domain.go:45-52`) does a linear scan for the first band where `score >=
  Min && score <= Max` — bands are **not required to be exhaustive or
  non-overlapping** by any DB or app-level validation beyond `CreateScorePredicate`'s
  own `Min > Max` check (`scoring/service.go:132-135`); a gap between bands
  silently resolves to `""` (rendered as `"-"` on the certificate, see §3
  step 8), and overlapping bands silently take whichever is listed first.
  Seed's real bands (`cmd/seed/main.go:835-839`): D 0–59.99, C 60–74.99,
  B 75–89.99, A 90–100.

## 6. Content: News and FAQ

- **`News`** (`content/domain.go:23-38`) — scoped to either a whole `School`
  or a single `Department` via `(ScopeType, ScopeID)`
  (`NewsScopeSchool|NewsScopeDepartment`). Two read paths:
  `ListPublicNews`/`GetNewsBySlug` (`content/service.go:70-83`, no auth,
  `PublishedOnly: true` forced, drafts 404 even by slug — "don't reveal
  drafts exist via slug guessing") for the landing page/logged-out visitors,
  versus the authenticated `ListNews` (`service.go:56-66`) which non-admins
  can only use pinned to their *own* school's posts (department-scoped posts
  are still reachable individually by ID through `GetNews`/`UpdateNews`,
  which re-check `ResolveDepartmentSchool` against the actor's school).
  Publishing fans out a notification to everyone in scope
  (`notifyAudience`, `service.go:179-196`) on a detached goroutine —
  best-effort, errors swallowed, doesn't block or fail the publish response.
- **`FAQ`** (`content/domain.go:40-49`) — **no scope column at all** (see the
  code comment at `content/service.go:198`, and seed's own comment at
  `cmd/seed/main.go:514-518`: "faqs has no school_id column ... the schema
  has no way to scope a FAQ to one school"). One shared list, platform-wide,
  public read (`ListFAQs`, no auth check) / admin-or-coordinator write —
  note coordinator write access here is *unscoped*, since there's no school
  column to scope against; any coordinator can edit any FAQ.

## 7. Multi-tenancy quick reference

| Entity | Scope | How it's enforced |
|---|---|---|
| `School` | platform-wide (admin only manages) | `assertCanManageSchool`-style checks, admin-only writes |
| `Department`, `Course` | school-scoped | resolved via `SchoolID`/`ResolveDepartmentSchool` |
| `Company` | school-scoped (via Department, no direct FK) | `ResolveCompanyScope` join; mentor further pinned to exactly one company via `actor.CompanyID` |
| `Vacancy`, `Appliance` | company-scoped (-> school-scoped) | `assertCanManageCompany` |
| `InternDate` | company + user scoped | `assertCanManagePlacement`/`assertCanAccessPlacement` |
| `Presence`, `Journal` | company-scoped | `assertCanManageCompany`; student sees only own rows |
| `Score`, `Certificate` | company-scoped | `assertCanManageCompany` |
| `PresenceStatus`, `ScorePredicate`, `Question` | **school**-scoped (not company) | `assertCanManageSchool`/`assertCanViewSchool` |
| `Monitor` | company-scoped for read/create; delete additionally pinned to the logging coordinator | `assertCoordinatorOwnsCompany` + `row.CoordinatorID == actor.ID` on delete |
| `Review` (`RevieweeCompany`) | company-scoped, but **read is platform-wide** | write: student only; read: any authenticated role |
| `Review` (`RevieweeUser`) | placement-scoped (mentor <-> student via `InternDate`, not company match alone) | `assertMentorMentorsStudent` — see §4 |
| `News` | school- or department-scoped; published rows also public | `assertCanManageScope`; `ListPublicNews` bypasses auth entirely |
| `FAQ` | **platform-wide, unscoped** | no scope column exists in schema |
| `User` (admin) | platform-wide | no scope columns set |
| `User` (coordinator) | school-scoped | `SchoolID` set; everything else derived transitively |
| `User` (mentor) | company-scoped | `CompanyID` set only |
| `User` (student) | school+department+course-scoped | `SchoolID`/`DepartmentID`/`CourseID` set; `CompanyID` **always nil** — active placement lives in `InternDate`, not on the user row |
