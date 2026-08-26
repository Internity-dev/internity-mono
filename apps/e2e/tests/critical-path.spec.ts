/**
 * Internity critical-path E2E spec.
 *
 * Covers, in order: login -> student browses vacancies and applies ->
 * coordinator accepts the application -> student self-schedules their
 * internship dates -> student checks in for attendance and writes a
 * journal entry -> mentor approves the attendance and journal entries ->
 * mentor enters a score -> student downloads their certificate.
 *
 * This is written against the demo dataset seeded by
 * `apps/api/cmd/seed/main.go` (`make seed`), which is idempotent and safe
 * to re-run:
 *   - admin@internity.test / coordinator@internity.test /
 *     mentor1@internity.test / budi@internity.test, all password
 *     "password123"
 *   - a "Frontend Developer Intern" vacancy at "PT Mumtaz Teknologi
 *     Indonesia" — mentor1's own company — used here so the same mentor
 *     account can review/score without needing the department/company
 *     pickers that a non-mentor staff role would use
 *   - budi already has an NIS on file, which the certificate step requires
 *
 * Every locator below was checked against the actual Vue component source
 * under apps/dashboard/src/views/** (button labels, dialog titles, field
 * labels/placeholders, and route paths are copied verbatim) rather than
 * guessed. A couple of structural notes worth flagging for reviewers:
 *
 *   - The shared `Select` (department/company/vacancy pickers) is a Reka UI
 *     `role="combobox"` button with no `aria-label`/`aria-labelledby`. This
 *     was assumed (from reading the compiled source, before ever running
 *     against a browser) to give it an accessible name from its visible
 *     placeholder text, but `combobox` is not a name-from-content role per
 *     the ARIA accname spec — confirmed live via `page.locator('body')
 *     .ariaSnapshot()`, which prints these as `combobox: Select department`
 *     (colon = content, not name) rather than `combobox "Select
 *     department"`. `getByRole('combobox', { name: ... })` therefore never
 *     matches, and hangs for the full test timeout instead of failing fast.
 *     This is a real accessibility gap in the shared Select component (every
 *     screen using it is effectively unlabeled for screen readers), worth
 *     fixing there separately. Here, the triggers are targeted via the
 *     adjacent `<label>` instead (`label + [data-slot="select-trigger"]`).
 *   - The shared `DataTable` renders a real <table>/<tbody>/<tr> with no
 *     data-testid or per-row key beyond array index, so rows are targeted
 *     structurally (`table tbody tr`) rather than by content. On the
 *     presence/journal review screens exactly one pending row is expected
 *     (this spec only ever creates one), so `.first()` is safe there. The
 *     coordinator's Appliances table is different: every real seeded
 *     vacancy at mentor1's company already has other applicants (and is, in
 *     fact, already fully booked — see the vacancy note below), and a
 *     re-run of this spec leaves its own earlier canceled attempts behind
 *     too, so that row is targeted by this student's UUID (via the
 *     applicant cell's `title` attribute) minus anything already
 *     "Canceled", not by position.
 *   - There is no staff-facing "set internship dates" screen anywhere in
 *     this codebase — MyInternshipView.vue (the only place dates are set)
 *     is restricted to the `student` role in router/index.ts. Accepting an
 *     application creates an intern_dates row with null start/end, and the
 *     *student* fills them in themselves the next time they open "My
 *     Internship" (the form only renders when start_date is null). So the
 *     "staff sets intern dates" step in the plan is implemented here as
 *     the student scheduling their own placement — see the report for
 *     this call-out; it isn't a bug in this spec, it's how the app
 *     actually works.
 *   - ScoresView.vue has no student-search endpoint (its own source
 *     comment says as much) — staff must paste the student's raw user
 *     UUID. This spec captures that UUID from the `title` attribute the
 *     Appliances table already puts on the (truncated) applicant cell
 *     (see AppliancesView.vue's `shortId()` helper) rather than
 *     hard-coding a UUID that would drift from a real seed run.
 *
 * NOT YET EXECUTED END-TO-END: running this for real needs the composed
 * docker-compose stack (Postgres/Redis/MinIO/API/dashboard) plus `make
 * seed`, neither of which is available in the environment this was
 * authored in. It has been verified with `tsc --noEmit` and
 * `playwright test --list` (parses/enumerates the spec without a browser
 * or server) but has not been run against a live app — the same honesty
 * bar the project README already applies to its integration-test gap.
 */
import { test, expect, type Page } from '@playwright/test'

const PASSWORD = 'password123'
const STUDENT_EMAIL = 'budi@internity.test'
const COORDINATOR_EMAIL = 'coordinator@internity.test'
const MENTOR_EMAIL = 'mentor13@internity.test'

// Mirrors E2E_BASE_URL below — direct API calls (bypassing the dashboard UI)
// are used to read/reset state that the UI has no screen for.
const API_BASE_URL = process.env.E2E_API_BASE_URL ?? 'http://localhost:8080'

// Whichever company this spec targets, the SAME mentor account there can
// review/score without the department/company pickers a non-mentor staff
// role would need — that's why a vacancy is always created at the mentor's
// own company rather than picked freely.
//
// IMPORTANT — this constant needs bumping to an untouched company after
// every full run against the SAME dev database (company 1 is permanently
// unusable, see below; companies 2 through 12 are spent as of this writing;
// company 13 is next): `intern_dates` has `UNIQUE (user_id, company_id)`
// (apps/api/migrations/000013_create_intern_dates.up.sql) — a student can
// only ever hold one placement per company, ever, with no delete path (the
// FK is `ON DELETE RESTRICT`, and until a backend fix landed alongside this
// spec, `DELETE /vacancies/:id` itself 500'd once an appliance referenced
// it instead of a clean 409 — see postgres.TranslateError's
// codeRestrictViolation case). So a full run of this spec *permanently*
// consumes budi's placement slot at whichever company it used, on every
// environment it runs against. That's expected, not a design flaw here: a
// real CI run (`make dev` + `make seed` + `test-e2e`, per the root
// Makefile) gets a fresh database every time, so this only bites when
// re-running locally against a dev database this spec (or a person) has
// already run against before.
//
// There's a SECOND, independent constraint worth knowing about before
// touching the dates below: `intern_dates` also EXCLUDEs any two of the
// same student's placements from having overlapping date ranges, scoped by
// user_id only — not per company (a student can't intern at two places on
// the same days). Concretely, budi can only ever have ONE placement whose
// window includes "today" at a time, company aside. The dates chosen below
// (today .. today+30) are what make attendance check-in valid (it checks
// the placement is currently active), so whichever company most recently
// ran this spec "owns" today's date until its window is manually moved
// aside — confirmed live by hitting this exact conflict switching from
// company 2 to company 4, then resolving it with a direct
// `PUT /internships/:id/dates` call moving company 2's window into the
// past. A fresh seed avoids all of this the same way it avoids the
// per-company exhaustion above.
//
// Company 1 ("PT Mumtaz Teknologi Indonesia", mentor1's own company) was
// tried first and doesn't work at all, even before repeat-run exhaustion:
// budi's seed data already gives them a permanently-accepted appliance
// there (vacancy 1, "Frontend Developer Intern", id 1), so the very first
// attempt already collides with the per-company unique constraint above.
// Confirmed live — the request succeeds in marking the appliance accepted,
// then fails inside vacancy.Service.Accept's call to
// internship.Service.ScheduleForAcceptedAppliance. That failure is now a
// clean 409 ("This student already has a placement at this company") —
// apps/api/internal/modules/internship/service.go now translates it — but
// originally surfaced as a raw, generic "Something went wrong" 500, which
// is how this was first found.
//
// The real seeded vacancies aren't reused either way: they're all either
// already fully booked or (at company 1) permanently blocked for budi the
// same way. This spec creates and reuses its own dedicated vacancy instead
// (see the first test.step below), which doubles as real coverage for the
// "coordinator creates a vacancy" flow (AdminVacanciesView.vue) rather than
// only ever reading seed data.
const DEPARTMENT_NAME = 'Rekayasa Perangkat Lunak'
const COMPANY_NAME = 'PT Konstruksi Bangun Persada'
const COMPANY_ID = 13
// The student-facing /vacancies browse page isn't scoped to one company, so
// a fixed name would collide with this same dedicated vacancy left behind
// at every OTHER company this spec has already targeted (companies 1
// through 5, as of this writing) — the company id keeps it globally unique.
const VACANCY_NAME = `E2E Automation Intern (Co. ${COMPANY_ID})`

/** Escapes regex metacharacters — VACANCY_NAME has literal parens in it. */
function toNameRegExp(text: string): RegExp {
  return new RegExp(text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
}

/** YYYY-MM-DD, matching the native <input type="date"> fields used throughout. */
function isoDate(date: Date): string {
  return date.toISOString().slice(0, 10)
}

/**
 * Logs in as `email` on the given page. Auth is cookie-based (see
 * apps/dashboard/src/lib/http.ts — axios withCredentials + a CSRF cookie),
 * and the router redirects an already-authenticated session away from
 * `/login` (see the `guestOnly` guard in router/index.ts) — so switching
 * actors on a shared page requires clearing cookies first, not just
 * visiting `/login` again.
 */
async function loginAs(page: Page, email: string, password: string) {
  await page.context().clearCookies()
  await page.goto('/login')
  // LoginView.vue is localized to Indonesian ("Kata sandi" / "Masuk") while
  // most of the rest of the dashboard stays in English — see docs/RULES.md
  // and the auth-flow localization pass. Copy the exact current strings
  // instead of the English ones the original plan assumed.
  await page.getByLabel('Email').fill(email)
  // exact: true — the password field's own show/hide toggle button has an
  // aria-label of "Tampilkan kata sandi", which contains "Kata sandi" as a
  // substring, and Playwright's getByLabel matches substrings by default.
  await page.getByLabel('Kata sandi', { exact: true }).fill(password)
  await page.getByRole('button', { name: 'Masuk' }).click()
  await expect(page).toHaveURL(/\/dashboard$/)

  // Driver.js fires a first-login onboarding tour overlay per user/session;
  // its SVG overlay intercepts pointer events and blocks clicks on whatever
  // page is visited next if left open. waitFor (not isVisible, which checks
  // once, synchronously) because the tour mounts async in DefaultLayout.vue
  // — see admin-crud.spec.ts's loginAs for the full story of chasing this
  // down as a real race, not the "cold Vite compile" red herring it first
  // looked like earlier in this suite's own history.
  const tourClose = page.locator('.driver-popover-close-btn')
  const tourShown = await tourClose
    .waitFor({ state: 'visible', timeout: 2000 })
    .then(() => true)
    .catch(() => false)
  if (tourShown) {
    await tourClose.click()
  }
}

test('critical path: apply -> accept -> schedule -> attend/journal -> review -> score -> certificate', async ({
  page,
}) => {
  test.setTimeout(120_000)

  // Captured right after student login via /auth/me, then reused to target
  // this student's own row on the coordinator's Appliances table (which is
  // shared with every other seeded applicant to this vacancy — see the
  // "apply" step below) and again on the mentor's Scores page.
  let studentUserId = ''
  // Resolved by the setup step below — see the top comment for why this
  // can't just be a hardcoded seeded vacancy id.
  let vacancyId = 0

  await test.step('coordinator ensures a dedicated vacancy with open slots exists', async () => {
    await loginAs(page, COORDINATOR_EMAIL, PASSWORD)

    const list = await page.request.get(`${API_BASE_URL}/api/v1/vacancies`, {
      params: { company_id: COMPANY_ID, limit: 100 },
    })
    const existing = (await list.json()).data.find((v: { name: string }) => v.name === VACANCY_NAME)
    if (existing) {
      vacancyId = existing.id
      return
    }

    await page.goto('/admin/vacancies')
    await page.locator('label:text-is("Department") + [data-slot="select-trigger"]').click()
    await page.getByRole('option', { name: DEPARTMENT_NAME }).click()
    await page.locator('label:text-is("Company") + [data-slot="select-trigger"]').click()
    await page.getByRole('option', { name: COMPANY_NAME }).click()

    await page.getByRole('button', { name: 'Add vacancy' }).click()
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Name').fill(VACANCY_NAME)
    await dialog.getByLabel('Category').fill('teknis')
    // High on purpose so this vacancy never runs out of room across repeat
    // runs of this spec the way the real seeded vacancies already have.
    await dialog.getByLabel('Slots').fill('999')
    await dialog.getByLabel('Skills').fill('Vue, TypeScript, Tailwind')
    await dialog.getByRole('button', { name: 'Save' }).click()
    await expect(dialog).toBeHidden()

    const created = await page.request.get(`${API_BASE_URL}/api/v1/vacancies`, {
      params: { company_id: COMPANY_ID, limit: 100 },
    })
    const createdVacancy = (await created.json()).data.find((v: { name: string }) => v.name === VACANCY_NAME)
    if (!createdVacancy) throw new Error(`vacancy "${VACANCY_NAME}" not found via the API right after creating it`)
    vacancyId = createdVacancy.id
  })

  await test.step('student logs in', async () => {
    await loginAs(page, STUDENT_EMAIL, PASSWORD)
    const me = await page.request.get(`${API_BASE_URL}/api/v1/auth/me`)
    studentUserId = (await me.json()).data.user.id
  })

  await test.step('student finds the seeded vacancy and applies', async () => {
    // Self-healing setup: a prior run of this spec that failed partway
    // through the flow (or was re-run without a DB reset) can leave a
    // pending/processed appliance behind, which the backend's active-status
    // unique index would then reject as a duplicate. Cancel it first so the
    // apply step below always starts clean. An already-accepted leftover
    // means a *complete* prior run already consumed this vacancy for this
    // student — a real user can't reapply either, so that's a hard stop,
    // not something the spec should try to route around.
    const mine = await page.request.get(`${API_BASE_URL}/api/v1/appliances`)
    const existing = (await mine.json()).data.find((a: { vacancy_id: number }) => a.vacancy_id === vacancyId)
    if (existing?.status === 'pending' || existing?.status === 'processed') {
      // Mutating requests need the CSRF double-submit header (see
      // apps/dashboard/src/lib/http.ts) — page.request shares cookies with
      // the browser context but doesn't read/attach this header on its own.
      const csrf = (await page.context().cookies()).find((c) => c.name === 'internity_csrf')?.value ?? ''
      const cancel = await page.request.put(`${API_BASE_URL}/api/v1/appliances/${existing.id}/cancel`, {
        headers: { 'X-CSRF-Token': csrf },
      })
      if (!cancel.ok()) {
        throw new Error(`failed to cancel leftover appliance ${existing.id} before reapplying: ${cancel.status()} ${await cancel.text()}`)
      }
    } else if (existing?.status === 'accepted') {
      throw new Error(
        `budi is already accepted for vacancy ${vacancyId} (${VACANCY_NAME}) from a previous full run — reset the seed DB before re-running this spec.`,
      )
    }

    await page.goto('/vacancies')
    // ListToolbar debounces input by 300ms before refetching (see
    // ListToolbar.vue) — Playwright's auto-waiting on the link locator
    // below covers that without an explicit sleep.
    await page.getByPlaceholder('Search vacancies…').fill(VACANCY_NAME)
    await page.getByRole('link', { name: toNameRegExp(VACANCY_NAME) }).click()
    await expect(page).toHaveURL(/\/vacancies\/\d+$/)

    await page
      .getByPlaceholder('Add a short note to your application…')
      .fill('Excited to apply — this matches my Vue/TypeScript coursework.')
    await page.getByRole('button', { name: 'Apply now' }).click()

    await expect(page).toHaveURL(/\/my-applications$/)
    // A re-run of this spec accumulates canceled rows for the same vacancy
    // (see the self-healing cancel above — cancel changes status, it doesn't
    // delete the row), so the row is narrowed to the freshly-created
    // "Pending" one rather than assumed unique by vacancy name alone.
    const row = page.getByRole('row', { name: toNameRegExp(VACANCY_NAME) }).filter({ hasText: 'Pending' })
    await expect(row).toContainText('Pending')
  })

  await test.step('coordinator accepts the application', async () => {
    await loginAs(page, COORDINATOR_EMAIL, PASSWORD)
    await page.goto('/admin/appliances')

    // Org-scope pickers: coordinator/admin see Department -> Company ->
    // Vacancy selects (a mentor is auto-scoped to their own company and
    // skips these — see the `!isMentor` guard in AppliancesView.vue).
    // Targeted via the adjacent <label> rather than getByRole(name:) — see
    // the top comment on the Select component's accessible-name gap.
    await page.locator('label:text-is("Department") + [data-slot="select-trigger"]').click()
    await page.getByRole('option', { name: DEPARTMENT_NAME }).click()
    await page.locator('label:text-is("Company") + [data-slot="select-trigger"]').click()
    await page.getByRole('option', { name: COMPANY_NAME }).click()
    await page.locator('label:text-is("Vacancy") + [data-slot="select-trigger"]').click()
    await page.getByRole('option', { name: VACANCY_NAME }).click()

    // This vacancy already has other seeded applicants (confirmed live via
    // `GET /vacancies/3/appliances` — 6 rows, not 1), so the row is targeted
    // by this student's UUID (exposed via the applicant cell's `title`
    // attribute — see AppliancesView.vue's `shortId()` helper) instead of
    // assuming position. A re-run also leaves this student's own earlier
    // canceled row(s) behind (see the self-healing cancel above), so those
    // are excluded by status text rather than the row narrowed to "Pending"
    // — this locator is re-evaluated live and is also asserted against
    // *after* accepting, by which point the status text has become
    // "Accepted", not "Pending".
    const row = page
      .locator('table tbody tr')
      .filter({ has: page.locator(`[title="${studentUserId}"]`) })
      .filter({ hasNotText: 'Canceled' })
    await expect(row).toBeVisible()

    await row.getByRole('button', { name: 'Accept' }).click()
    // The row-level "Accept" button and the ConfirmDialog's "Accept" button
    // both have the same accessible name, so the second click is scoped to
    // the dialog to disambiguate.
    await page.getByRole('dialog').getByRole('button', { name: 'Accept' }).click()
    await expect(row).toContainText('Accepted')
  })

  await test.step("student schedules their internship dates", async () => {
    // There is no staff-facing "set intern dates" screen in this codebase —
    // see the top comment. Accepting the application above creates an
    // intern_dates row with null start/end; MyInternshipView shows the
    // dates form automatically until the student (not staff) fills it in.
    await loginAs(page, STUDENT_EMAIL, PASSWORD)
    await page.goto('/my-internship')

    // Self-healing setup, same spirit as the appliance-cancel above: budi can
    // only ever have ONE placement whose date window includes "today" (see
    // the EXCLUDE-constraint note in the top comment) — so a placement left
    // scheduled-and-active by an earlier run at a *different* company (the
    // company this run targets is always untouched-until-now, see COMPANY_ID
    // above) would collide the moment this run tries to claim today for
    // itself. Move any such leftover out of the way first by shrinking it to
    // end yesterday, rather than requiring a human to do this by hand before
    // every re-run against a shared, never-reset dev database. Postgres
    // reports the un-healed collision as an `exclusion_violation` (SQLSTATE
    // 23P01), which `postgres.TranslateError`
    // (apps/api/internal/platform/postgres/errors.go) doesn't recognize — it
    // only maps `unique_violation` (23505) — so it would otherwise fail as a
    // raw, untranslated 500 ("Something went wrong") rather than SetDates's
    // own clean "These dates overlap with another one of this student's
    // placements" message.
    {
      const csrf = (await page.context().cookies()).find((c) => c.name === 'internity_csrf')?.value ?? ''
      const mine: { id: number; company_id: number; start_date: string | null; end_date: string | null; status: string; version: number }[] = (
        await (await page.request.get(`${API_BASE_URL}/api/v1/internships/mine`)).json()
      ).data
      const today = new Date()
      const newEnd = new Date(today)
      newEnd.setDate(newEnd.getDate() + 30)
      const todayStr = isoDate(today)
      const newEndStr = isoDate(newEnd)

      // The window this run is about to claim below is today..today+30 — any
      // OTHER placement whose own range actually overlaps that window has to
      // move first (a same-range end_date far in the future from an
      // unrelated, already-displaced placement must NOT be treated as a
      // blocker here — only a true overlap counts). Pushing a blocker
      // *forward* (rather than shrinking its end into the past) works
      // unconditionally regardless of its own start_date — including the
      // common case where that start_date IS today, which shrinking to "ends
      // yesterday" can't satisfy (start would land after end).
      const blockers = mine.filter(
        (p) => p.company_id !== COMPANY_ID && p.status !== 'completed' && p.start_date && p.end_date && isoDate(new Date(p.start_date)) <= newEndStr && isoDate(new Date(p.end_date)) >= todayStr,
      )
      // Each blocker gets its own 30-day slot, stacked back-to-back starting
      // after the LATEST end_date among *all* of budi's placements (blockers
      // included) — not just today+31 — so a moved blocker can't land on top
      // of some other, unrelated placement that already sits further out.
      const latestEnd = mine.reduce((max, p) => (p.end_date && p.end_date > max ? p.end_date : max), newEndStr)
      let slotStart = new Date(latestEnd)
      slotStart.setDate(slotStart.getDate() + 1)
      for (const p of blockers) {
        const slotEnd = new Date(slotStart)
        slotEnd.setDate(slotEnd.getDate() + 30)
        const res = await page.request.put(`${API_BASE_URL}/api/v1/internships/${p.id}/dates`, {
          headers: { 'X-CSRF-Token': csrf },
          data: { start_date: isoDate(slotStart), end_date: isoDate(slotEnd), expected_version: p.version },
        })
        if (!res.ok()) {
          throw new Error(`failed to move leftover placement ${p.id} (company ${p.company_id}) out of the way: ${res.status()} ${await res.text()}`)
        }
        slotStart = new Date(slotEnd)
        slotStart.setDate(slotStart.getDate() + 1)
      }
    }

    const start = new Date()
    const end = new Date(start)
    end.setDate(end.getDate() + 30)

    await page.getByLabel('Start date').fill(isoDate(start))
    await page.getByLabel('End date').fill(isoDate(end))
    await page.getByRole('button', { name: 'Save dates' }).click()

    // Once dates are saved the form is replaced by a read-only view with an
    // "Edit dates" button (see MyInternshipView.vue's `showDatesForm`).
    await expect(page.getByRole('button', { name: 'Edit dates' })).toBeVisible()
  })

  await test.step('student checks in for attendance', async () => {
    await page.goto('/attendance')
    await page.getByRole('button', { name: 'Check in' }).click()
    await expect(page.getByRole('dialog', { name: 'Check in' })).toBeVisible()

    // Skip the webcam capture: there's no real camera in a CI/headless
    // environment, and the component explicitly supports checking in
    // without a photo via this button (AttendanceView.vue's ghost
    // "Skip photo" button, wired to the same check-in mutation).
    await page.getByRole('dialog').getByRole('button', { name: 'Skip photo' }).click()
    await expect(page.getByRole('dialog')).toBeHidden()
  })

  await test.step('student writes a journal entry', async () => {
    await page.goto('/journals')
    await page.getByRole('button', { name: 'Add journal entry' }).click()
    await expect(page.getByRole('dialog', { name: 'New journal entry' })).toBeVisible()

    // Date defaults to today (todayISODate()) and is left as-is.
    await page.getByLabel('Work type').fill('Frontend development')
    await page
      .getByLabel('Description')
      .fill('Built the vacancy listing page and wired up the apply form end to end.')
    await page.getByRole('button', { name: 'Save entry' }).click()
    await expect(page.getByRole('dialog')).toBeHidden()
  })

  await test.step("mentor approves the attendance record", async () => {
    await loginAs(page, MENTOR_EMAIL, PASSWORD)
    await page.goto('/admin/presence')

    // mentor2 is auto-scoped to their own company (isMentor hides the
    // department/company pickers — see PresenceReviewView.vue). Despite its
    // URL, `/presences/pending-approval` returns this company's whole
    // presence history (approved rows included, confirmed live via a
    // screenshot after approving), not only pending ones — company 2 has
    // plenty of other seeded students' presence rows in both states, so the
    // row is targeted by this student's UUID rather than position, and the
    // empty-title "Nothing pending" (which only fires when the table has
    // zero rows at all) never applies here.
    const row = page.locator('table tbody tr').filter({ has: page.locator(`[title="${studentUserId}"]`) })
    await expect(row).toBeVisible()
    await row.getByRole('button', { name: 'Approve' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Approve' }).click()
    await expect(row).toContainText('Approved')
  })

  await test.step('mentor approves the journal entry', async () => {
    await page.goto('/admin/journals')

    // Same reasoning as the attendance row above — JournalReviewView.vue has
    // the identical "whole history, not just pending" shape.
    const row = page.locator('table tbody tr').filter({ has: page.locator(`[title="${studentUserId}"]`) })
    await expect(row).toBeVisible()
    await row.getByRole('button', { name: 'Approve' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Approve' }).click()
    await expect(row).toContainText('Approved')
  })

  await test.step('mentor enters a score', async () => {
    await page.goto('/admin/scores')

    // No student-search endpoint exists (see the top comment) — the raw
    // UUID captured from the Appliances table is pasted directly, exactly
    // as a real staff user would have to.
    await page.getByPlaceholder('e.g. 3fa85f64-5717-4562-b3fc-2c963f66afa6').fill(studentUserId)

    await page.getByRole('button', { name: 'Add score' }).click()
    await expect(page.getByRole('dialog', { name: 'Add score' })).toBeVisible()

    await page.getByLabel('Name').fill('Frontend development skills')
    // Type defaults to "Teknis" (ScoresView.vue's openCreate() resets the
    // form with type: 'teknis'), so it's left unchanged here.
    await page.getByLabel('Score (0–100)').fill('88')
    await page.getByRole('button', { name: 'Save' }).click()

    await expect(page.getByRole('dialog')).toBeHidden()
    await expect(page.getByText(/Average score:/)).toBeVisible()
  })

  await test.step('student downloads their certificate', async () => {
    await loginAs(page, STUDENT_EMAIL, PASSWORD)
    await page.goto('/certificate')

    // CertificateView.vue builds a Blob URL and programmatically clicks a
    // hidden <a download> — Chromium still fires a real "download" event
    // for that, so this doesn't need any DOM trick beyond waiting for it.
    const downloadPromise = page.waitForEvent('download')
    await page.getByRole('button', { name: 'Download certificate' }).click()
    const download = await downloadPromise

    expect(download.suggestedFilename()).toMatch(/certificate/i)
  })
})
