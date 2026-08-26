# Testing Internity — Practical Reference

For an agent that needs to verify a change live. Read this before running any suite or
touching the shared dev DB — several things here look reasonable but silently do the
wrong thing.

## 1. Running each suite

Commands below are copied verbatim from the root `Makefile` and `.github/workflows/*`
(the CI files are the ground truth for exact flags — read them again if these drift).

### apps/api (Go)

```
cd apps/api && go test ./... -race -cover        # unit tests, Makefile target: test-api
cd apps/api && go test -tags=integration ./...   # Makefile target: test-integration
```

- Only **one** file in the whole tree uses testcontainers:
  `apps/api/internal/modules/internship/integration_test.go`, gated behind the
  `integration` build tag. The other 18 `*_test.go` files run with plain `go test ./...`
  and need no Docker/DB — safe to run anywhere, anytime.
- `-tags=integration` needs a real Docker daemon. `ci-api.yml`'s own comment says this
  outright: "GitHub-hosted runners have a real Docker daemon, unlike the sandboxed dev
  environment this project was otherwise built in" — i.e. this tag is expected to only
  *compile* (not run) in a sandboxed agent environment without Docker.
- CI also runs `go build ./...`, `go vet ./...`, and `gofmt -l . | tee /tmp/fmt.out && test ! -s /tmp/fmt.out`
  (`.github/workflows/ci-api.yml:20-23`) before tests — run those too if you want the full
  CI-equivalent check, not just `go test`.
- Migration reproducibility is checked separately (`ci-api.yml`'s `migrate` job): apply all
  up, roll back latest, reapply — proves the newest migration's down is correct, not just
  its up.

### apps/dashboard (Vue 3)

```
pnpm --filter @internity/dashboard run type-check          # vue-tsc --build
pnpm --filter @internity/dashboard run test:unit -- --run   # vitest, non-watch mode
```

- `package.json` scripts (`apps/dashboard/package.json`): `"type-check": "vue-tsc --build"`,
  `"test:unit": "vitest"`. Bare `pnpm run test:unit` launches vitest in **watch mode** and
  never exits — always append `-- --run` (this is exactly what `ci-dashboard.yml:21` does).
- RULES.md rule 14 is explicit: **`vue-tsc --noEmit` is a silent no-op in this project**
  (TS project-references setup) — don't substitute it for `--build`, you'll get a false
  "no errors" with zero output.
- 17 existing spec files live under `apps/dashboard/src/**/__tests__/*.spec.ts` (e.g.
  `src/components/shared/__tests__/PasswordInput.spec.ts`) — follow that colocation
  pattern for new tests, not a top-level `tests/` dir.
- `make test-dashboard` (root Makefile) wraps just the vitest command, not type-check —
  run `type-check` separately.

### apps/e2e (Playwright)

```
pnpm --filter @internity/e2e run test           # full suite, Makefile target: test-e2e
pnpm --filter @internity/e2e exec playwright test --list   # list without running
```

This suite drives a browser against an **already-running** stack — it does not boot its
own Postgres/API/dashboard (`playwright.config.ts` deliberately has no `webServer`
option). Before running it you need `make dev` (or the stack already up) plus
`make seed`. Default `baseURL` is `http://localhost:5173` (`E2E_BASE_URL` env var
overrides it; the dashboard's docker-compose port per `.env.example`'s `DASHBOARD_PORT`).

**The project-dependency gotcha (hit this session):** `playwright.config.ts` defines two
projects:

```ts
projects: [
  { name: 'critical-path', testMatch: 'critical-path.spec.ts', use: {...} },
  { name: 'chromium', testIgnore: 'critical-path.spec.ts', dependencies: ['critical-path'], use: {...} },
]
```

`chromium` depends on `critical-path` — Playwright will **always** run
`critical-path.spec.ts` first, even if you only asked for another file, because
`business-flows.spec.ts` needs an *active* placement that only `critical-path.spec.ts`'s
apply→accept→schedule flow normally creates. So:

- `npx playwright test business-flows.spec.ts` → silently also runs the entire
  critical-path spec first (expensive, and burns a company slot — see §3).
- To run **only** one non-critical-path file: `npx playwright test business-flows.spec.ts --project=chromium --no-deps`.
- To run only critical-path: `npx playwright test --project=critical-path` (or just target
  the file, since `chromium` ignores it anyway).

`fullyParallel: false`, `workers: 1` — the whole suite is intentionally sequential
(shared stateful accounts/vacancies), don't try to parallelize it.
`expect: { timeout: 10_000 }` (not Playwright's 5s default) because this dev DB is remote
and shows real multi-second latency spikes under load — see §4.

## 2. Seeded accounts (source: `apps/api/cmd/seed/main.go`, 2346 lines, idempotent —
safe to re-run, fills in whatever's missing)

Shared password for **every** seeded account: `password123` (`main.go:31`,
`bcrypt.DefaultCost` hash, 8-char minimum).

Seed creates **3 schools** (genuinely separate tenants) to demonstrate multi-tenant
isolation. School 1 (SMKN 1 Cibinong) is the big one — 25 companies, ~150 students.

| Email | Role | Scope |
|---|---|---|
| `admin@internity.test` | admin | cross-school |
| `coordinator@internity.test` | coordinator | SMKN 1 Cibinong |
| `mentor1@internity.test` .. `mentor25@internity.test` | mentor | 1 per company, school 1 |
| `budi@internity.test`, `siti@internity.test`, ... | student | school 1, first-name@internity.test pattern |
| `coordinator.bogor@internity.test` | coordinator | SMKN 2 Bogor (TKJ dept) |
| `mentor.bogor1@internity.test`... | mentor | school 2 |
| firstname.lastname@internity.test (e.g. `rafi.ardiansyah@internity.test`) | student | school 2 |
| `coordinator.depok@internity.test` | coordinator | SMKN 1 Depok (Multimedia dept) |
| `mentor.depok1@internity.test`... | mentor | school 3 |
| firstname.lastname@internity.test | student | school 3 |

Invite codes (self-registration, one per course):
- School 1 (RPL): `RPL1DEMO`, `RPL2DEMO`, `RPL3DEMO`
- School 2 (TKJ): `TKJ1DEMO`, `TKJ2DEMO`, `TKJ3DEMO`
- School 3 (MM): `MM1DEMO`, `MM2DEMO`, `MM3DEMO`

**Richest demo data: `mentor1@internity.test` / company `PT Mumtaz Teknologi Indonesia`**
(`companySpecs[0]`, `main.go:187`) — confirmed still accurate. It has:
- 2 vacancies (`Frontend Developer Intern`, `UI/UX Design Intern`, both open) —
  `main.go:293-294`.
- A completed placement with a 5-star company review ("Pengalaman PKL di PT Mumtaz
  Teknologi Indonesia...", `main.go:557`) plus an accepted-and-starting-soon placement and
  a canceled appliance (`main.go:355,368,383`) — i.e. real variety across appliance
  statuses and intern-placement stages, not just one thin row.
- In a **freshly seeded** DB it's typically `company_id=1` (first company upserted,
  insert-if-missing by name). Don't assume this on a DB that's been reseeded on top of
  older rows — verify by name, not by hardcoding the ID.

⚠️ **This same company is also an E2E fixture target.** `business-flows.spec.ts` runs its
reject/cancel flows against `PT Mumtaz Teknologi Indonesia` by name
(`REJECT_COMPANY_NAME`/`CANCEL_VACANCY_NAME`, `business-flows.spec.ts:52-56`), leaving
behind vacancies named `E2E Reject Flow Intern` / `E2E Cancel Flow Intern`. On a dev DB
that's had E2E runs against it, the "clean" richest-data company is also the one most
likely to be polluted — check for these before using it in a screenshot (see §4).

## 3. Login form selectors (Playwright)

`apps/dashboard/src/views/auth/LoginView.vue` — **Indonesian**, even though the rest of
the dashboard (admin views, vacancy/appliance/score forms, etc.) is in English:

```ts
await page.getByLabel('Email').fill(email)
await page.getByLabel('Kata sandi', { exact: true }).fill(password)  // exact:true required!
await page.getByRole('button', { name: 'Masuk' }).click()
```

- Email label: exactly `"Email"` (`LoginView.vue:69`).
- Password label: `"Kata sandi"` (`LoginView.vue:89`) — **must** pass `exact: true`.
  `PasswordInput.vue`'s own show/hide toggle button has `aria-label="Tampilkan kata sandi"`
  (or `"Sembunyikan kata sandi"` once revealed) — `"Kata sandi"` is a substring of both,
  and Playwright's `getByLabel` matches substrings by default, so without `exact: true`
  it's ambiguous/wrong-matches.
- Submit button: `"Masuk"` while idle; becomes `"Sedang masuk…"` (with a spinner) only
  while `isSubmitting` is true (`LoginView.vue:109`) — target `"Masuk"` before clicking.
- This exact `loginAs` helper already exists at `apps/e2e/tests/critical-path.spec.ts:172-201`
  — reuse/copy it rather than re-deriving selectors.

**Post-login gotcha:** a Driver.js onboarding tour overlay fires on first login per
user/session and its SVG intercepts pointer events, blocking clicks on the next page if
left open (`critical-path.spec.ts:187-201`). The existing `loginAs` helper handles this:

```ts
const tourClose = page.locator('.driver-popover-close-btn')
const tourShown = await tourClose.waitFor({ state: 'visible', timeout: 2000 }).then(() => true).catch(() => false)
if (tourShown) await tourClose.click()
```

Use `waitFor` (async-safe), not a synchronous `isVisible()` check — the tour mounts
asynchronously in `DefaultLayout.vue`, and a sync check races it (confirmed live, not a
cold-Vite-compile red herring per the comment at `critical-path.spec.ts:189-193`).

## 4. Operational hazards on the shared dev DB (hit repeatedly this session)

This is **not** a local/disposable database — it's shared across sessions/agents and
accumulates state permanently. Three distinct issues, don't conflate them:

**a) Fixture-row pollution.** Every Playwright run leaves rows behind with recognizable
names: `E2E Automation Intern (Co. N)`, `E2E Reject Flow Intern`, `E2E Cancel Flow Intern`
(`business-flows.spec.ts`), `E2E CRUD Test School <timestamp>`, `E2E CRUD Test
Department/Course/Company (edited)`, `E2E Status`, `E2E Predicate`, `E2E Test News Post
<timestamp>`, `E2E Test FAQ question <timestamp>`, `E2E Created Mentor`
(`admin-crud.spec.ts`) — plus whatever manual ad-hoc rows past agents created while
chasing bugs (arbitrary names like "Bugfix", "throwaway", "test", etc. — nothing
guarantees these follow a pattern). **When live-verifying something with real seeded
data — e.g. for a screenshot or manual walkthrough — do not assume the first row/result
you see is representative.** Use search/sort/filter to find genuinely clean seed data, or
explicitly note the pollution in your findings rather than treating it as a product bug.

**b) Permanent, unfixable per-student-per-company exhaustion.** `intern_dates` has
`UNIQUE (user_id, company_id)` with `ON DELETE RESTRICT`
(`apps/api/migrations/000013_create_intern_dates.up.sql`) — a student can only ever hold
ONE placement at a given company, ever, with no delete path. Running
`critical-path.spec.ts` against the same dev DB repeatedly **permanently** burns budi's
placement slot at whichever company it targets that run. The spec's own comment
(`critical-path.spec.ts:94-152`) tracks a running tally of which company id is "next"
(company 1 permanently blocked because budi's *seed data itself* already has an accepted
appliance there; companies 2-12 exhausted as of that writing; 13 is next) — **the number
in that comment is stale the moment anyone runs the spec again**, don't trust it as
current without re-reading the file. Also: `intern_dates` EXCLUDEs overlapping date
ranges per `user_id` (not per company) — budi can only have one placement whose window
includes "today" at any given time, so switching which company "owns today" requires
moving another placement's dates out of the way first. None of this is a bug to report —
it's expected on a reused dev DB; only a fresh `make seed` against an empty DB avoids it.

**c) Infra flakiness under concurrent load.** This dev DB is remote (not localhost) and
has shown real contention under concurrent agent load — logins taking 3-20+ seconds,
occasional timeouts/500s with "context canceled; transaction already committed". This is
why `playwright.config.ts` overrides the default 5s `expect.timeout` up to 10s
(`playwright.config.ts` comment, confirmed live via a toast assertion timing out while its
dialog still showed "Saving…", not an app bug). **Retry once, patiently, before
concluding something is actually broken.** Avoid hammering `/auth/login` in tight loops —
it's also rate-limited (RULES/testing-flow §3.4: >5 failed attempts → 429
`RATE_LIMITED`), so a retry loop can trip that too.

## 5. `docs/testing-flow.pdf` — manual QA script (read directly if you need the full text)

5-page Indonesian-language manual test script, explicitly built for this take-home
submission's interview discussion (references spec points 3,4,6,7,8,9,10,12,16,17). It's
the canonical "what does correct behavior look like" reference for anything not covered
by an automated spec. Structure:

1. **Seed accounts table** (§1) — same accounts as §2 above, slightly abbreviated.
2. **Critical E2E flow** (§2, 19 numbered steps) — the full business cycle: register with
   invite code → login (checks HttpOnly/SameSite cookie attributes, single in-flight
   `/auth/refresh` under parallel 401s) → vacancy apply → coordinator processes appliance
   through its state machine (pending→processed→accepted, illegal transitions must 409/422
   not 500) → schedule internship dates (anti-overlap constraint) → attendance check-in
   (camera-optional graceful fallback) → excuse → journal → coordinator review/approve →
   scores (0-100 range, auto letter-grade) → certificate download (idempotent numbering,
   blocked state before scores are complete).
3. **Security** (§3) — no tokens in localStorage/sessionStorage; logout clears cookies;
   `requiresAuth` redirect preserves `?redirect=`; CSRF header required on mutations;
   RBAC (student can't reach `/admin/*` even via direct API call — must get 403 not 200;
   coordinator/mentor scoped to own school/company); rate limiting on login (429 after
   >5 failures).
4. **Input validation matrix** (§4) — required fields, string length bounds, email format,
   enum rejection, numeric range (scores 0-100, non-negative slots), date ordering,
   UUID/ID path-param format, file type/size limits on uploads.
5. **Response envelope + status code matrix** (§5) — the `{success,data,message,meta}` /
   `{success:false,data:null,message,error:{code,details},meta}` shape, with one concrete
   real-endpoint example per code 200/201/204/400/401/403/404/409/422/429/500.
6. **UI/UX states + search/filter/sort/pagination** (§6-7) — loading skeletons vs blank
   screen, empty states, error states, disabled-during-submit buttons, confirmation
   dialogs before destructive actions, toasts on every mutation, responsive breakpoints
   (375/768/1280px), keyboard nav, and the standard
   `?page=&limit=&search=&sort=&order=&status=` query contract across listings.

## 6. Documentation screenshot rules (RULES.md rules 19-20, added this session)

Applies to **any** future documentation/manual/report work, not just testing:

- **Rule 19:** a screenshot used for documentation must be live-verified error-free first
  — check the network tab and console for 4xx/5xx responses or error toasts *before*
  treating a screenshot as final. Never assume correctness just from how the page looks.
- **Rule 20:** the data shown must be genuinely loaded — no loading skeletons, no empty
  ("No data yet") states. If the account/scope you're using has no real data, switch
  accounts or seed data first rather than shipping a skeleton/empty screenshot. (This is
  exactly why §2's note on `mentor1@internity.test` / PT Mumtaz Teknologi Indonesia
  matters — it's the account most likely to actually have something to show, pollution
  caveat in §4a aside.)
