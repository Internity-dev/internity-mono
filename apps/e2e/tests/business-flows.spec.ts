/**
 * Internity business-flow E2E spec — scenarios beyond the one happy path
 * covered by critical-path.spec.ts: auth edge cases, an application getting
 * rejected instead of accepted, a student canceling their own pending
 * application, and the attendance excuse flow (permitted/sick, as opposed
 * to a normal check-in).
 *
 * Unlike critical-path.spec.ts, none of these scenarios ever call
 * `PUT /appliances/:id/accept` — so none of them create an `intern_dates`
 * row, and none of them hit the per-company "one placement per student,
 * ever" wall that spec's top comment documents. A *complete* run of the
 * reject/cancel flows ends with the appliance rejected/canceled, which
 * isn't "active" (see the DB's partial unique index on active statuses),
 * so re-applying to the same vacancy on a re-run works with no cleanup
 * needed. An *interrupted* run (a rate-limited login, a flaky redirect)
 * can still leave a "pending" appliance behind, which IS active and blocks
 * the next run's apply — confirmed live, not theoretical — so both flows
 * self-heal that via `cancelLeftoverAppliance` below before applying fresh.
 *
 * Conventions and known gotchas are inherited from critical-path.spec.ts —
 * see that file's top comment for the full detail on each of these:
 *   - the shared `Select` component's trigger has no accessible name
 *     (`combobox: Select department`, not `combobox "Select department"` in
 *     an ariaSnapshot) — targeted via the adjacent `<label>` instead of
 *     `getByRole('combobox', { name })`.
 *   - `DataTable` rows have no data-testid — targeted structurally.
 *   - mutating `page.request` calls need the `internity_csrf` cookie value
 *     forwarded as an `X-CSRF-Token` header by hand.
 *
 * Verified against a live seeded stack, not written speculatively — every
 * locator below was exercised for real via `npx playwright test` and its
 * failures fixed against the actual rendered DOM, the same bar
 * critical-path.spec.ts's own header now meets.
 */
import { test, expect, type Page } from '@playwright/test'

const PASSWORD = 'password123'
const STUDENT_EMAIL = 'budi@internity.test'
const COORDINATOR_EMAIL = 'coordinator@internity.test'
const ADMIN_EMAIL = 'admin@internity.test'
const DEPARTMENT_ID = 1

const API_BASE_URL = process.env.E2E_API_BASE_URL ?? 'http://localhost:8080'

// Reject/cancel only ever need a vacancy to apply to, never an accepted
// placement, so — unlike critical-path.spec.ts — these are reused
// indefinitely across companies without ever running out: rejecting or
// canceling leaves the appliance in an inactive status, which the DB's
// partial unique index (active statuses only) doesn't block re-applying
// against. Company 1 is used here purely because it's the simplest fixed
// reference point (no rotation needed, unlike the accept flow).
const REJECT_DEPARTMENT_NAME = 'Rekayasa Perangkat Lunak'
const REJECT_COMPANY_NAME = 'PT Mumtaz Teknologi Indonesia'
const REJECT_COMPANY_ID = 1
const REJECT_VACANCY_NAME = 'E2E Reject Flow Intern'
const CANCEL_VACANCY_NAME = 'E2E Cancel Flow Intern'

/** Escapes regex metacharacters in vacancy names used as locator patterns. */
function toNameRegExp(text: string): RegExp {
  return new RegExp(text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
}

/** YYYY-MM-DD, matching the native <input type="date"> fields used throughout. */
function isoDate(date: Date): string {
  return date.toISOString().slice(0, 10)
}

/** See critical-path.spec.ts's identical helper for the full rationale. */
async function loginAs(page: Page, email: string, password: string) {
  await page.context().clearCookies()
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  // exact: true — the password field's own show/hide toggle button has an
  // aria-label of "Tampilkan kata sandi", which contains "Kata sandi" as a
  // substring, and Playwright's getByLabel matches substrings by default.
  await page.getByLabel('Kata sandi', { exact: true }).fill(password)
  await page.getByRole('button', { name: 'Masuk' }).click()
  await expect(page).toHaveURL(/\/dashboard$/)

  // waitFor, not isVisible (checks once, synchronously) — see
  // admin-crud.spec.ts's loginAs for why this matters: the tour mounts
  // async, and a lost race here leaves it open and silently blocking
  // clicks on whatever page is visited next.
  const tourClose = page.locator('.driver-popover-close-btn')
  const tourShown = await tourClose
    .waitFor({ state: 'visible', timeout: 2000 })
    .then(() => true)
    .catch(() => false)
  if (tourShown) {
    await tourClose.click()
  }
}

/**
 * Ensures a vacancy with the given name exists at the given company,
 * creating it via a direct API call if not (the "coordinator creates a
 * vacancy" UI flow itself is already covered by critical-path.spec.ts, so
 * this skips straight to having something to apply to). Requires an active
 * coordinator/admin session on `page`.
 */
async function ensureVacancy(page: Page, name: string, companyId: number): Promise<number> {
  const list = await page.request.get(`${API_BASE_URL}/api/v1/vacancies`, { params: { company_id: companyId, limit: 100 } })
  const existing = (await list.json()).data.find((v: { name: string }) => v.name === name)
  if (existing) return existing.id

  const csrf = (await page.context().cookies()).find((c) => c.name === 'internity_csrf')?.value ?? ''
  const created = await page.request.post(`${API_BASE_URL}/api/v1/vacancies`, {
    headers: { 'X-CSRF-Token': csrf },
    data: { name, company_id: companyId, slots: 5, category: 'teknis' },
  })
  if (!created.ok()) throw new Error(`failed to create vacancy "${name}": ${created.status()} ${await created.text()}`)
  return (await created.json()).data.id
}

// Reject/cancel appliances aren't "active" (the DB's partial unique index
// only covers pending/processed), so a *complete* prior run of these flows
// really is naturally repeatable — but a prior run interrupted between
// applying and reject/cancel (a rate-limited login, a flaky redirect, this
// spec itself failing mid-step) leaves a "pending" appliance behind, which
// IS active and blocks the next run's own apply. Confirmed live, not
// theoretical: hit exactly this after an earlier interrupted run of the
// reject-flow test. Cancels any such leftover for the current user before
// applying fresh, the same self-healing spirit as critical-path.spec.ts's
// own appliance cleanup.
async function cancelLeftoverAppliance(page: Page, vacancyId: number): Promise<void> {
  const mine = await page.request.get(`${API_BASE_URL}/api/v1/appliances`)
  const existing = (await mine.json()).data.find(
    (a: { vacancy_id: number; status: string }) => a.vacancy_id === vacancyId && (a.status === 'pending' || a.status === 'processed'),
  )
  if (!existing) return
  const csrf = (await page.context().cookies()).find((c) => c.name === 'internity_csrf')?.value ?? ''
  const res = await page.request.put(`${API_BASE_URL}/api/v1/appliances/${existing.id}/cancel`, {
    headers: { 'X-CSRF-Token': csrf },
  })
  if (!res.ok()) throw new Error(`failed to cancel leftover appliance ${existing.id} before reapplying: ${res.status()} ${await res.text()}`)
}

type Placement = { id: number; company_id: number; start_date: string | null; end_date: string | null; status: string; version: number }

/** Reads the CSRF cookie for the double-submit header `page.request` needs on mutating calls. */
async function csrfToken(page: Page): Promise<string> {
  return (await page.context().cookies()).find((c) => c.name === 'internity_csrf')?.value ?? ''
}

// The excuse and performance-review scenarios below both need budi to
// already hold an *active* placement (a date window that includes today)
// somewhere — critical-path.spec.ts's own full apply -> accept -> schedule
// flow normally creates exactly that, but this file can't assume that file
// has already run: Playwright's default file discovery order is
// alphabetical, so a bare `npx playwright test` (this package's own "test"
// script) always runs critical-path.spec.ts LAST, after this file.
// Confirmed live, not theoretical — this broke on the very first run
// against a genuinely untouched company, where no earlier session's
// leftover placement happened to paper over the ordering gap. So this
// creates its own active placement via direct API calls when it can't find
// an existing one, fully independent of the other spec files' run order or
// leftover state — reusing critical-path.spec.ts's own approach to the
// `intern_dates` EXCLUDE constraint (move any other active placement out of
// the way first; see that file's top comment for the full detail).
async function ensureActivePlacement(page: Page): Promise<{ companyId: number; mentorEmail: string }> {
  await loginAs(page, STUDENT_EMAIL, PASSWORD)
  const todayStr = isoDate(new Date())
  const mine: Placement[] = (await (await page.request.get(`${API_BASE_URL}/api/v1/internships/mine`)).json()).data
  const active = mine.find(
    (p) => p.start_date && p.end_date && isoDate(new Date(p.start_date)) <= todayStr && isoDate(new Date(p.end_date)) >= todayStr,
  )

  let companyId: number
  if (active) {
    companyId = active.company_id
  } else {
    // Pick a company budi has never held ANY placement at (rather than
    // reasoning about which specific past ones are safe to reuse) — the
    // simplest way to guarantee `intern_dates`' UNIQUE(user_id, company_id)
    // never gets in the way of a fresh accept below.
    const touchedIds = new Set(mine.map((p) => p.company_id))
    await loginAs(page, COORDINATOR_EMAIL, PASSWORD)
    const companies: { id: number; name: string }[] = (
      await (await page.request.get(`${API_BASE_URL}/api/v1/companies`, { params: { department_id: DEPARTMENT_ID, limit: 100 } })).json()
    ).data
    const candidate = companies.find((c) => !touchedIds.has(c.id))
    if (!candidate) throw new Error('no untouched company left for budi to create a fresh active placement in')
    companyId = candidate.id
    const vacancyId = await ensureVacancy(page, `E2E Active Placement Intern (Co. ${companyId})`, companyId)

    await loginAs(page, STUDENT_EMAIL, PASSWORD)
    await cancelLeftoverAppliance(page, vacancyId)
    const applyRes = await page.request.post(`${API_BASE_URL}/api/v1/appliances`, {
      headers: { 'X-CSRF-Token': await csrfToken(page) },
      data: { vacancy_id: vacancyId },
    })
    if (!applyRes.ok()) throw new Error(`failed to apply to vacancy ${vacancyId}: ${applyRes.status()} ${await applyRes.text()}`)
    const applianceId = (await applyRes.json()).data.id

    await loginAs(page, COORDINATOR_EMAIL, PASSWORD)
    const acceptRes = await page.request.put(`${API_BASE_URL}/api/v1/appliances/${applianceId}/accept`, {
      headers: { 'X-CSRF-Token': await csrfToken(page) },
    })
    if (!acceptRes.ok()) throw new Error(`failed to accept appliance ${applianceId}: ${acceptRes.status()} ${await acceptRes.text()}`)

    await loginAs(page, STUDENT_EMAIL, PASSWORD)
    const mineAfterAccept: Placement[] = (await (await page.request.get(`${API_BASE_URL}/api/v1/internships/mine`)).json()).data
    const fresh = mineAfterAccept.find((p) => p.company_id === companyId)
    if (!fresh) throw new Error(`accepted appliance ${applianceId} didn't create an intern_dates row for company ${companyId}`)

    // Move any OTHER active placement out of the way first, same approach
    // as critical-path.spec.ts's own self-heal: push it forward into its
    // own 30-day slot stacked after the latest end_date among ALL of
    // budi's placements, which works regardless of that blocker's own
    // start_date and can't collide with anything else already out there.
    const newEnd = new Date()
    newEnd.setDate(newEnd.getDate() + 30)
    const newEndStr = isoDate(newEnd)
    const blockers = mineAfterAccept.filter(
      (p) =>
        p.company_id !== companyId &&
        p.status !== 'completed' &&
        p.start_date &&
        p.end_date &&
        isoDate(new Date(p.start_date)) <= newEndStr &&
        isoDate(new Date(p.end_date)) >= todayStr,
    )
    const latestEnd = mineAfterAccept.reduce((max, p) => (p.end_date && p.end_date > max ? p.end_date : max), newEndStr)
    let slotStart = new Date(latestEnd)
    slotStart.setDate(slotStart.getDate() + 1)
    const csrf = await csrfToken(page)
    for (const p of blockers) {
      const slotEnd = new Date(slotStart)
      slotEnd.setDate(slotEnd.getDate() + 30)
      const moveRes = await page.request.put(`${API_BASE_URL}/api/v1/internships/${p.id}/dates`, {
        headers: { 'X-CSRF-Token': csrf },
        data: { start_date: isoDate(slotStart), end_date: isoDate(slotEnd), expected_version: p.version },
      })
      if (!moveRes.ok()) throw new Error(`failed to move blocking placement ${p.id} out of the way: ${moveRes.status()} ${await moveRes.text()}`)
      slotStart = new Date(slotEnd)
      slotStart.setDate(slotStart.getDate() + 1)
    }

    const scheduleRes = await page.request.put(`${API_BASE_URL}/api/v1/internships/${fresh.id}/dates`, {
      headers: { 'X-CSRF-Token': csrf },
      data: { start_date: todayStr, end_date: newEndStr, expected_version: fresh.version },
    })
    if (!scheduleRes.ok()) throw new Error(`failed to schedule dates for placement ${fresh.id}: ${scheduleRes.status()} ${await scheduleRes.text()}`)
  }

  // ListUsers is admin/coordinator-only (identity.Service.ListUsers) — but
  // specifically ADMIN here, not coordinator: a coordinator's call forces
  // `filter.SchoolID = actor.SchoolID` on top of whatever filter was
  // requested, and mentor accounts have no school_id at all (only
  // company_id — they're not school-scoped), so that forced AND always
  // excludes every mentor and silently returns empty. Confirmed live, not
  // theoretical. Admin's call passes the filter through unrestricted.
  await loginAs(page, ADMIN_EMAIL, PASSWORD)
  const mentor = (await (await page.request.get(`${API_BASE_URL}/api/v1/users`, { params: { company_id: companyId, role: 'mentor', limit: 1 } })).json())
    .data[0]
  if (!mentor) throw new Error(`no mentor account found for company ${companyId}`)

  return { companyId, mentorEmail: mentor.email }
}

test.describe('auth edge cases', () => {
  test('invalid login shows an inline error, not just a toast', async ({ page }) => {
    await page.context().clearCookies()
    await page.goto('/login')
    // A nonexistent email rather than STUDENT_EMAIL with a wrong password —
    // functionally identical for what this test actually checks (any failed
    // login attempt), but doesn't spend one of budi's 5-per-5-minutes auth
    // rate-limit budget (apps/api/internal/middleware/ratelimit.go) on a
    // login that's never meant to succeed. Confirmed live, not theoretical:
    // this file's OWN legitimate login count (critical-path.spec.ts's own 3
    // + this file's own reject/cancel + excuse/reviews flows) already adds
    // up to more than 5 for budi specifically when this test also spent one.
    await page.getByLabel('Email').fill('nobody-e2e@internity.test')
    // exact: true — see loginAs's identical comment above.
    await page.getByLabel('Kata sandi', { exact: true }).fill('definitely-the-wrong-password')
    await page.getByRole('button', { name: 'Masuk' }).click()

    // LoginView.vue shows a persistent Alert (title always "Gagal masuk")
    // on any failed submit. The description prefers the API response's own
    // message when present — the backend's actual 401 body says "Invalid
    // email or password" (English), confirmed by reading
    // apps/api/internal/modules/identity/service.go, so that's what
    // actually renders here, not the Indonesian fallback string. The same
    // text is ALSO shown as a toast by http.ts's global 401 handler for
    // "auth endpoints without a session" (login is one) — confirmed live,
    // not assumed, so the assertion is scoped to the Alert specifically to
    // avoid a strict-mode violation against the toast's identical text.
    const alert = page.getByRole('alert')
    await expect(alert.getByText('Gagal masuk')).toBeVisible()
    await expect(alert.getByText('Invalid email or password')).toBeVisible()
    await expect(page).toHaveURL(/\/login$/)
  })

  test('registers a new account with a valid seeded invite code', async ({ page }) => {
    await page.context().clearCookies()
    await page.goto('/register')

    // RPL1DEMO is seeded by apps/api/cmd/seed/main.go for the first tenant
    // school's first course — confirmed in the seed source, not guessed.
    // The email must be unique per run since register hard-fails on a
    // duplicate; Date.now() keeps repeat runs collision-free.
    const email = `e2e-register-${Date.now()}@internity.test`
    await page.getByLabel('Nama lengkap').fill('E2E Test Student')
    await page.getByLabel('Email').fill(email)
    // getByLabel does substring matching by default, and "Konfirmasi kata
    // sandi" contains "kata sandi" — exact: true is required to hit only
    // the plain "Kata sandi" field.
    await page.getByLabel('Kata sandi', { exact: true }).fill(PASSWORD)
    await page.getByLabel('Konfirmasi kata sandi').fill(PASSWORD)
    await page.getByLabel('Kode undangan').fill('RPL1DEMO')
    await page.getByRole('button', { name: 'Buat akun' }).click()

    // Registration logs the new account straight in and redirects — no
    // separate confirmation step.
    await expect(page).toHaveURL(/\/dashboard$/)
  })

  test('forgot-password shows the same message whether or not the email exists', async ({ page }) => {
    await page.context().clearCookies()
    await page.goto('/forgot-password')
    await page.getByLabel('Email').fill('this-address-almost-certainly-does-not-exist@internity.test')
    await page.getByRole('button', { name: 'Kirim tautan reset' }).click()

    // Deliberately non-committal wording either way — confirmed by reading
    // ForgotPasswordView.vue, this exact text renders regardless of whether
    // the POST's backing email is real, so an attacker can't use this form
    // to enumerate registered accounts.
    await expect(page.getByText('Jika email tersebut terdaftar, tautan reset telah dikirim. Periksa kotak masuk Anda.')).toBeVisible()
  })
})

// Reject and cancel merged into ONE test rather than two, for the same
// rate-limit-budget reason documented on the excuse/reviews merge below:
// each separate test() pays for its own fresh logins, and this pair's
// naive total (2 coordinator + 2 student across the old "reject" test,
// plus 1 coordinator + 1 student in the old "cancel" test) adds up.
// Reordered so the "student sees the rejection" step also applies to (and
// cancels) the second vacancy while already in that same student session,
// cutting the total from 3 coordinator + 3 student logins down to 2 + 2.
test.describe('appliance review flows', () => {
  test('coordinator rejects one application; student cancels another', async ({ page }) => {
    await loginAs(page, COORDINATOR_EMAIL, PASSWORD)
    const rejectVacancyId = await ensureVacancy(page, REJECT_VACANCY_NAME, REJECT_COMPANY_ID)
    const cancelVacancyId = await ensureVacancy(page, CANCEL_VACANCY_NAME, REJECT_COMPANY_ID)

    let studentUserId = ''

    await test.step('student applies to the vacancy that will be rejected', async () => {
      await loginAs(page, STUDENT_EMAIL, PASSWORD)
      const me = await page.request.get(`${API_BASE_URL}/api/v1/auth/me`)
      studentUserId = (await me.json()).data.user.id

      await cancelLeftoverAppliance(page, rejectVacancyId)

      await page.goto('/vacancies')
      await page.getByPlaceholder('Search vacancies…').fill(REJECT_VACANCY_NAME)
      await page.getByRole('link', { name: toNameRegExp(REJECT_VACANCY_NAME) }).click()
      await expect(page).toHaveURL(new RegExp(`/vacancies/${rejectVacancyId}$`))
      await page.getByRole('button', { name: 'Apply now' }).click()
      await expect(page).toHaveURL(/\/my-applications$/)
    })

    await test.step('coordinator rejects it', async () => {
      await loginAs(page, COORDINATOR_EMAIL, PASSWORD)
      await page.goto('/admin/appliances')
      await page.locator('label:text-is("Department") + [data-slot="select-trigger"]').click()
      await page.getByRole('option', { name: REJECT_DEPARTMENT_NAME }).click()
      await page.locator('label:text-is("Company") + [data-slot="select-trigger"]').click()
      await page.getByRole('option', { name: REJECT_COMPANY_NAME }).click()
      await page.locator('label:text-is("Vacancy") + [data-slot="select-trigger"]').click()
      await page.getByRole('option', { name: REJECT_VACANCY_NAME }).click()

      // Every prior run's own appliance to this vacancy also ends up
      // "Rejected" (this is the only transition this test ever performs on
      // this vacancy) — unlike critical-path.spec.ts's "coordinator
      // accepts" step, there's no status to exclude that isn't ALSO the
      // destination status here, so the row can't be re-located by text
      // after the mutation. "Pending" is unique at click time (exactly one
      // fresh appliance per run), and the toast is a clean, state-
      // independent success signal for what happens after.
      const row = page.locator('table tbody tr').filter({ hasText: 'Pending' }).filter({ has: page.locator(`[title="${studentUserId}"]`) })
      await expect(row).toBeVisible()
      await row.getByRole('button', { name: 'Reject' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Reject' }).click()
      await expect(page.getByText('Application rejected')).toBeVisible()
    })

    await test.step('student sees the rejection, then applies to a second vacancy and cancels it', async () => {
      await loginAs(page, STUDENT_EMAIL, PASSWORD)
      await page.goto('/my-applications')
      // The vacancy name cell is resolved by a separate client-side batch
      // fetch (MyApplicationsView.vue's vacancyName()) after the appliances
      // list itself renders, falling back to "Vacancy #<id>" text until (or
      // if) that resolves — confirmed live, this can still be showing the
      // fallback text within the assertion's retry window even though the
      // underlying GET /vacancies/:id call itself succeeds fine when tried
      // directly. Matching either form keeps this from being flaky over
      // that harmless client-side race rather than papering over it with a
      // longer timeout.
      // \b doesn't work as a right-hand boundary here: Playwright's row
      // text is the cells concatenated with no separator (e.g.
      // "Vacancy #92RejectedAug 25, 2026"), and \b never fires between two
      // word characters — "2" and "R" are both \w, so `92\b` silently never
      // matched. A digit lookahead does what \b was meant to here.
      const rejectedRow = page
        .locator('table tbody tr')
        .filter({ hasText: new RegExp(`${toNameRegExp(REJECT_VACANCY_NAME).source}|Vacancy #${rejectVacancyId}(?!\\d)`) })
        .filter({ hasText: 'Rejected' })
      await expect(rejectedRow.first()).toBeVisible()

      await cancelLeftoverAppliance(page, cancelVacancyId)

      await page.goto('/vacancies')
      await page.getByPlaceholder('Search vacancies…').fill(CANCEL_VACANCY_NAME)
      await page.getByRole('link', { name: toNameRegExp(CANCEL_VACANCY_NAME) }).click()
      await expect(page).toHaveURL(new RegExp(`/vacancies/${cancelVacancyId}$`))
      await page.getByRole('button', { name: 'Apply now' }).click()
      await expect(page).toHaveURL(/\/my-applications$/)

      // Same reasoning as the reject flow above — every prior run's own
      // application to this vacancy also ends up "Canceled", the same
      // status this one is transitioning to, so there's nothing to exclude
      // that isn't also the destination. "Pending" is unique at click time;
      // the toast is the state-independent signal for what happens after.
      const cancelRow = page.getByRole('row', { name: toNameRegExp(CANCEL_VACANCY_NAME) }).filter({ hasText: 'Pending' })
      await expect(cancelRow).toBeVisible()
      await cancelRow.getByRole('button', { name: 'Cancel' }).click()

      // The confirm dialog's own dismiss button is also plain "Cancel" —
      // only the destructive confirm button is uniquely "Cancel application".
      //
      // Retried below rather than a single click: this specific mutation has
      // been seen, live and reproducibly (not once), to leave the dialog
      // stuck on "Please wait…" forever — the server-side log for that exact
      // request shows a genuine "context canceled" (the client end of the
      // connection closed mid-request), yet the browser never surfaces that
      // as a rejected request (no toast, button never re-enables). Doesn't
      // reproduce in an short, isolated repro of just this one interaction —
      // only inside a long multi-minute, multi-login suite run — which
      // matches this session's already-documented pattern of this specific
      // remote dev environment's connection reliability rather than a bug in
      // the mutation's own code (read: axios's response interceptor handles
      // a genuinely-rejected/network-failed request correctly and would
      // still fire onError/re-enable the button). A stuck client with a
      // server that actually processed the request is exactly the situation
      // to re-check server state for rather than blindly re-click.
      await page.getByRole('dialog').getByRole('button', { name: 'Cancel application' }).click()
      const canceledToast = page.getByText('Application canceled')
      const settled = await canceledToast
        .waitFor({ state: 'visible', timeout: 20_000 })
        .then(() => true)
        .catch(() => false)
      if (!settled) {
        const check = await page.request.get(`${API_BASE_URL}/api/v1/appliances`, { params: { vacancy_id: cancelVacancyId, limit: 5 } })
        const stillPending = (await check.json()).data.some((a: { status: string }) => a.status === 'pending')
        if (stillPending) {
          // The mutation genuinely never reached the server (or the server
          // never committed it) — reload and retry the whole interaction
          // once from a clean page state.
          await page.reload()
          const retryRow = page.getByRole('row', { name: toNameRegExp(CANCEL_VACANCY_NAME) }).filter({ hasText: 'Pending' })
          await expect(retryRow).toBeVisible()
          await retryRow.getByRole('button', { name: 'Cancel' }).click()
          await page.getByRole('dialog').getByRole('button', { name: 'Cancel application' }).click()
          await expect(canceledToast).toBeVisible()
        }
        // else: the server actually canceled it — the client just never
        // found out — nothing left to do.
      }
    })
  })
})

// Excuse flow and performance reviews merged into ONE test rather than two
// separate ones sharing a describe block each: both alternate between a
// student session and a mentor session, and Playwright gives every test()
// a fresh, unauthenticated page/context, so N separate tests each paying
// for their own student+mentor login pair adds up fast against the auth
// rate limiter (5 requests per 5 minutes per email, see
// apps/api/internal/middleware/ratelimit.go) — confirmed live, not
// theoretical: running this file's tests back-to-back tripped budi's own
// limit purely from its own necessary logins, well before any external
// noise. Grouping BOTH of budi's actions (file an excuse, rate the company)
// into one student session, and BOTH of the mentor's (approve the excuse,
// score the student) into one mentor session — reordered so neither
// depends on the other having happened first, which is true of all four —
// cuts this section from 4 logins down to 2, the same rate-limit-budget
// reasoning admin-crud.spec.ts's top comment documents for its own
// consolidation.
test.describe('attendance excuse flow and performance reviews', () => {
  test('student files an excuse and rates their company; mentor approves the excuse and scores the student', async ({ page }) => {
    let studentUserId = ''

    // Guarantees an active placement exists (self-sufficiently creating one
    // if needed — see ensureActivePlacement's own comment for why this
    // can't just assume critical-path.spec.ts already ran) before either of
    // the two sessions below need one. Leaves the page logged in as
    // coordinator.
    const { companyId, mentorEmail } = await ensureActivePlacement(page)

    await test.step('student files an excuse for a day other than today, and rates the company', async () => {
      await loginAs(page, STUDENT_EMAIL, PASSWORD)
      const me = await page.request.get(`${API_BASE_URL}/api/v1/auth/me`)
      studentUserId = (await me.json()).data.user.id

      // budi already has an approved check-in for *today* at this company
      // (either from critical-path.spec.ts's own run, or from
      // ensureActivePlacement's own fallback above) — presences has
      // UNIQUE(user_id, company_id, date), so filing an excuse for today at
      // that same company would collide. A day in the future avoids it
      // without needing a whole separate company/placement just for this
      // scenario.
      //
      // NOT yesterday: that placement itself only *starts* today either way,
      // and the backend correctly rejects an excuse dated before a
      // placement's start as "That date is outside your internship period"
      // — confirmed live via a direct API call, not assumed.
      //
      // The day is picked by checking which of the next several days
      // *don't* already have a presence row, rather than a fixed offset —
      // a fixed "+2" collided with this same test's own prior run on a
      // repeat execution (every run adds one more day to the history), so
      // this scans forward instead of needing a manual bump every time, the
      // same self-healing spirit as critical-path.spec.ts's appliance
      // cleanup.
      await page.goto('/attendance')
      const existing = await page.request.get(`${API_BASE_URL}/api/v1/presences`, { params: { company_id: companyId, limit: 100 } })
      const takenDates = new Set((await existing.json()).data.map((p: { date: string }) => p.date.slice(0, 10)))
      const targetDay = new Date()
      targetDay.setDate(targetDay.getDate() + 2)
      while (takenDates.has(isoDate(targetDay))) {
        targetDay.setDate(targetDay.getDate() + 1)
      }

      await page.getByRole('button', { name: 'File excuse' }).click()
      const excuseDialog = page.getByRole('dialog', { name: 'File an excuse' })
      await expect(excuseDialog).toBeVisible()

      await excuseDialog.getByLabel('Date').fill(isoDate(targetDay))
      // Unlike the org-scope pickers elsewhere, this Select's id="excuse-kind"
      // IS forwarded to the trigger and the <label for> really does
      // associate — getByLabel works here (confirmed against the rendered
      // DOM, not assumed from the other pickers' broken pattern).
      await excuseDialog.getByLabel('Reason').click()
      await page.getByRole('option', { name: 'Sick' }).click()
      await excuseDialog.getByLabel('Description').fill('Demam, istirahat di rumah sesuai saran dokter.')
      await excuseDialog.getByRole('button', { name: 'Submit excuse' }).click()
      await expect(excuseDialog).toBeHidden()

      await page.goto('/my-internship')
      // MyInternshipView.vue defaults to the first placement tab
      // (`selectedTab = ref('0')`) — the active placement resolved by
      // ensureActivePlacement above, same company the excuse just filed
      // above already depends on.
      await page.getByRole('button', { name: 'Rate this company' }).click()
      const reviewDialog = page.getByRole('dialog', { name: 'Rate this company' })
      await expect(reviewDialog).toBeVisible()

      await reviewDialog.getByRole('radio', { name: '5 out of 5' }).click()
      await reviewDialog.getByLabel('Title (optional)').fill('Pengalaman PKL')
      await reviewDialog.getByLabel('Comment (optional)').fill('Lingkungan kerja mendukung dan mentor membimbing dengan baik.')
      await reviewDialog.getByRole('button', { name: 'Submit review' }).click()
      await expect(page.getByText('Review submitted')).toBeVisible()
    })

    await test.step("mentor approves the excuse and scores the student's performance", async () => {
      await loginAs(page, mentorEmail, PASSWORD)
      await page.goto('/admin/presence')

      // PresenceReviewView.vue has no separate UI for excuse-type rows —
      // same table, same per-row "Approve" button, same ConfirmDialog copy
      // as a normal check-in. The only visible difference is blank
      // Check-in/Check-out cells ("—"), which this doesn't need to assert
      // on to prove the approval flow itself works.
      //
      // "Pending approval" is usually unique among this student's rows at
      // click time (every earlier row, from this test's own prior *complete*
      // runs, is already "Approved" — the same terminal status this one is
      // transitioning to) — but an *interrupted* prior run (this test's own
      // flakiness, a rate-limited login elsewhere in the suite) can leave
      // more than one behind, since each run files a fresh excuse before
      // this step. Confirmed live, not theoretical — a strict-mode
      // violation here caught exactly that. Approving all of them (rather
      // than assuming exactly one) is both robust to that and doubles as
      // cleanup so the next run starts from fewer leftovers, same spirit as
      // the reject/cancel self-healing above.
      const rows = page.locator('table tbody tr').filter({ has: page.locator(`[title="${studentUserId}"]`) }).filter({ hasText: 'Pending approval' })
      await expect(rows.first()).toBeVisible()
      const pendingCount = await rows.count()
      for (let i = 0; i < pendingCount; i++) {
        // Always .first() — approving one shrinks the live-filtered set out
        // from under a fixed index.
        await rows.first().getByRole('button', { name: 'Approve' }).click()
        await page.getByRole('dialog').getByRole('button', { name: 'Approve' }).click()
        // .last() — looping this fast can stack an earlier iteration's own
        // not-yet-dismissed toast with this one, which getByText() alone
        // would then see as two matches (strict-mode violation).
        await expect(page.getByText('Attendance approved').last()).toBeVisible()
      }

      await page.goto('/admin/scores')

      // Same gotcha as ScoresView's existing student-UUID field noted in
      // critical-path.spec.ts's top comment (no student-search endpoint) —
      // and this label has no `for`/`id` association either, confirmed by
      // reading ScoresView.vue, so it's targeted by its placeholder like
      // the rest of this codebase's UUID-paste fields.
      await page.getByPlaceholder('e.g. 3fa85f64-5717-4562-b3fc-2c963f66afa6').fill(studentUserId)

      await page.getByRole('button', { name: 'Add review' }).click()
      const scoreDialog = page.getByRole('dialog', { name: 'Add review' })
      await expect(scoreDialog).toBeVisible()

      await scoreDialog.getByRole('radio', { name: '4 out of 5' }).click()
      await scoreDialog.getByLabel('Title (optional)').fill('Evaluasi Kinerja Magang')
      await scoreDialog.getByLabel('Comment (optional)').fill('Kinerja baik, komunikatif, dan disiplin.')
      await scoreDialog.getByRole('button', { name: 'Submit review' }).click()
      await expect(page.getByText('Review submitted')).toBeVisible()
    })
  })
})
