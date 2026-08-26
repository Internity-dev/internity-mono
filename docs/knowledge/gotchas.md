# Gotchas — read this before you touch this repo

Quick-reference bug journal. Each entry is something that actually broke, live,
this repo's real history — not hypothetical. Full writeup + citations live in
the linked doc; this file exists so you don't have to read all of them to
avoid stepping on the same rake twice.

1. **`useListQuery`'s `enabled` option: never read from that same call's own
   destructured return value**, even transitively through a `computed`. It's a
   circular-initialization crash (blank page) at runtime that TypeScript does
   **not** catch. Read `useRoute().query` directly instead. →
   `frontend-architecture.md`

2. **New mentor-facing endpoint keyed by a student's user ID (not a company
   ID directly)?** Checking `actor.CompanyID == companyID` is not enough — you
   need a placement check (does the student actually have an `InternDate` at
   that company), or you reintroduce the exact cross-tenant review leak fixed
   in commit `3027939`. Copy `review.assertMentorMentorsStudent` /
   `PlacementChecker`. → `domain-model.md` §4, `auth-and-security.md`

3. **"Most recent row by date" is not "today's row"** wherever a date field
   has no max-date constraint (e.g. `FileExcuse` allows future dates). Filter
   explicitly by the calendar date you mean, don't trust sort+limit=1. Real
   bug, fixed in commit `54353dc`. → `domain-model.md` §3

4. **CSRF 403 for no obvious reason** — check two things first: (a) did
   `/auth/refresh` just rotate the CSRF token out from under a concurrent
   request (fixed — the token now carries forward across a refresh instead of
   rotating every time); (b) is `COOKIE_DOMAIN` set correctly if the dashboard
   and API are served from different subdomains (the CSRF cookie is
   unreadable by the dashboard's JS otherwise, even though it's visible in
   devtools). → `auth-and-security.md`

5. **`middleware.RequireRole` is dead code** — defined, never wired into any
   route. There is no declarative role gate anywhere. All RBAC (role checks
   *and* scope checks) is written by hand inline in each `Service` method.
   Adding a new endpoint means copying an existing `assertCan*` pattern from a
   sibling method, not looking for a middleware to attach. →
   `backend-architecture.md`

6. **Fresh clone or Docker build failing on a Tailwind/design-token import?**
   `packages/design-tokens/dist/` is gitignored. Run `pnpm tokens:build`
   before the app build. Already wired into `Dockerfile.prod` for both
   dashboard and landing — if you're touching those Dockerfiles, don't drop
   that step. → `deployment.md`, `design-system.md`

7. **`redis.ParseURL`**: the literal string `"redis"` collides with a
   recognized URL scheme name, so passing just a host:port without a scheme
   silently resolves to `localhost`. Already fixed (checks for `"://"` first)
   in `apps/api/internal/platform/redisx/redis.go` — don't revert that check.

8. **Rate limiter** (10 requests / 5 min, keyed by IP+email, on every
   `/auth/*` endpoint) makes rapid live-testing of login flows flaky/slow
   under concurrent load (multiple agents/sessions hitting the same dev DB).
   A hang or a slow response during testing is often this, not a real bug —
   retry once patiently before concluding something's broken. →
   `testing.md`, `auth-and-security.md`

9. **This dev database is shared and accumulates E2E test-fixture rows**
   (names containing "E2E", "Bugfix", "throwaway", "Cancel Flow", etc.) from
   every Playwright run against it. When picking "real" data for a demo,
   screenshot, or manual walkthrough, don't assume the first row you see is
   representative — search/sort to find genuinely clean data, or explicitly
   note the pollution rather than treating it as a product bug. →
   `testing.md`

10. **Dark mode contrast was previously too extreme** (~17:1 background-to-
    foreground) and was deliberately softened this session. Don't regress it
    back toward the extreme when touching `tokens.json`'s dark section — check
    the current, already-tuned values first. → `design-system.md`

11. **`Textarea.vue`'s `field-sizing-content`** needs `break-words`/`max-w-
    full` or an unbroken long string (1000+ chars, no spaces — reachable via
    the excuse-filing dialog or the uncapped journal description field) blows
    the textarea's intrinsic width past its container, pushing dialog buttons
    off-screen. Already fixed — don't drop the wrap classes.

12. **Score predicate bands and presence-status kinds are not validated for
    completeness.** Gaps between score bands silently resolve to `"-"` on a
    certificate; overlapping bands silently resolve to whichever is listed
    first. This is existing, accepted behavior, not something recent — know
    it before you "fix" a certificate that shows `"-"` for a score that looks
    like it should have a grade. → `domain-model.md` §5

13. **`FAQ` has no `school_id` column at all** — it's genuinely platform-wide
    by schema, not by a missing scope check. Any coordinator can edit any
    school's FAQ. This needs a migration to change, not a service-layer
    patch — don't "fix" it as if it were an oversight in `review.go`/
    `content/service.go`. → `domain-model.md` §6

14. **Screenshots used for documentation/manuals must be live-verified
    error-free (no 4xx/5xx, no error toasts) and show genuinely loaded data**
    (no loading skeletons, no empty states) before being treated as final —
    this is a standing project rule (`docs/RULES.md` #19-20), not just
    session guidance. Build a verification script that waits for the actual
    data-loaded selector, not a fixed timeout.

15. **A subagent's own "fixed and verified" report is not proof.** This
    session had a subagent confidently report a CSS fix for a Tabs layout bug
    that was actually backwards (it inverted the default/vertical-variant
    class order, which would have broken every other Tabs consumer in the
    app) — caught only by re-deriving the CSS logic by hand and checking a
    second real consumer of the component, not by trusting the report. Always
    re-verify a subagent's fix against the actual diff, especially for shared
    components with more than one consumer.
