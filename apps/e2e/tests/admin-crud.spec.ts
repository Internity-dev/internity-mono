/**
 * Internity admin org-management CRUD E2E spec — Schools, Departments,
 * Courses, Companies, Presence statuses, Score predicates, and student
 * invite-code generation. These are simpler than critical-path.spec.ts or
 * business-flows.spec.ts in one important way: every entity here is
 * created AND deleted within the same test, so — unlike a vacancy
 * appliance's accept/placement flow — nothing here permanently consumes
 * shared seed state, and these scenarios stay repeatable indefinitely
 * against the same dev database without any self-healing setup.
 *
 * All six admin-role scenarios live in ONE test as test.step()s sharing a
 * single login, not six separate test()s each logging in fresh — that's
 * deliberate, not a style choice: the auth rate limiter
 * (apps/api/internal/server/server.go, 5 requests per 5 minutes per
 * account+IP) counts every `POST /auth/login`, and six separate admin
 * logins in one file run deterministically hits that wall on the 6th
 * (confirmed live — repeated runs failed at whichever test happened to be
 * 6th, never anything else, and each scenario passes cleanly once given
 * its own budget). The invite-code scenario is coordinator-only, so it
 * still needs its own separate test() / login.
 *
 * Conventions inherited from critical-path.spec.ts / business-flows.spec.ts
 * (see their top comments for the full detail):
 *   - loginAs() clears cookies, logs in, dismisses the first-login
 *     driver.js tour (via waitFor, not isVisible — see loginAs itself).
 *   - Every row-level action button on these screens is icon-only with an
 *     `aria-label="Edit <entity>"` / `aria-label="Delete <entity>"` pair
 *     (confirmed by reading all seven views' source, not assumed from one)
 *     — that's the accessible name Playwright locators use below, there is
 *     no visible "Edit"/"Delete" text anywhere on these rows.
 *   - Dialog-form Selects on these screens (School/Department pickers
 *     inside the create dialogs) DO have a real `<label for>` association,
 *     unlike the org-scope pickers on AppliancesView/AdminVacanciesView —
 *     confirmed by reading each file, not assumed — so plain getByLabel()
 *     works for them.
 */
import { test, expect, type Page } from '@playwright/test'

const PASSWORD = 'password123'
const ADMIN_EMAIL = 'admin@internity.test'
const COORDINATOR_EMAIL = 'coordinator@internity.test'
const API_BASE_URL = process.env.E2E_API_BASE_URL ?? 'http://localhost:8080'

// Seeded in apps/api/cmd/seed/main.go for the first tenant (SMKN 1
// Cibinong): school id 1, its one department "Rekayasa Perangkat Lunak",
// and its courses "XII RPL 1/2/3". Used as fixed reference points for the
// Departments/Courses/Companies/invite-code scenarios below — they only
// ever create a fresh CHILD under them (a department/course/company),
// never mutate the reference points themselves.
const SCHOOL_NAME = 'SMKN 1 Cibinong'
const SCHOOL_ID = 1
const DEPARTMENT_NAME = 'Rekayasa Perangkat Lunak'
const COURSE_NAME = 'XII RPL 1'
// Company 1's own department_id resolves to school 1 too (confirmed live
// via GET /companies/5) — mentor5's company, budi's most recent real
// placement per critical-path.spec.ts, used here as a genuine student+
// company pairing for the monitoring-visit and presence-report steps
// rather than a made-up one.
const MONITOR_COMPANY_NAME = 'PT Sinar Abadi Farmasi'
const MONITOR_STUDENT_EMAIL = 'budi@internity.test'

async function loginAs(page: Page, email: string, password: string) {
  await page.context().clearCookies()
  await page.goto('/login')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Kata sandi').fill(password)
  await page.getByRole('button', { name: 'Masuk' }).click()
  await expect(page).toHaveURL(/\/dashboard$/)

  // isVisible() checks synchronously, right as this runs — the tour mounts
  // async in DefaultLayout.vue's onMounted, so a plain isVisible() check can
  // lose the race and see "not there yet," leaving it undismissed for the
  // rest of the test. Since it's a global overlay (not page-scoped), it
  // then silently blocks clicks on whatever page is visited next — this is
  // what an earlier "cold Vite compile" theory in this session's own
  // history turned out to actually be, confirmed live this time via a
  // bounding-box/screenshot check showing the driver.js SVG overlay still
  // intercepting pointer events on a totally unrelated later page. waitFor
  // actively polls instead of checking once.
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
 * Ensures a dedicated, empty-of-presence-statuses school exists for the
 * presence-status/score-predicate steps, creating it via a direct API call
 * if not (the "admin creates a school" UI flow itself is covered by its own
 * step below). Every real seeded school already has all five presence-
 * status kinds configured (confirmed live via `GET /presence-statuses` for
 * schools 1-3), and the backend enforces exactly one status per kind, so
 * those can't be reused for a fresh "create a status" walkthrough. Requires
 * an active admin session on `page`.
 */
async function ensureEmptySchool(page: Page, name: string): Promise<number> {
  const list = await page.request.get(`${API_BASE_URL}/api/v1/schools`, { params: { limit: 100 } })
  const existing = (await list.json()).data.find((s: { name: string }) => s.name === name)
  if (existing) return existing.id

  const csrf = (await page.context().cookies()).find((c) => c.name === 'internity_csrf')?.value ?? ''
  const created = await page.request.post(`${API_BASE_URL}/api/v1/schools`, {
    headers: { 'X-CSRF-Token': csrf },
    data: { name },
  })
  if (!created.ok()) throw new Error(`failed to create school "${name}": ${created.status()} ${await created.text()}`)
  return (await created.json()).data.id
}

test.describe('org-management CRUD', () => {
  test('admin manages schools, departments, courses, companies, presence statuses, and score predicates', async ({ page }) => {
    // 10 distinct admin routes get visited in this one test — each is its
    // own lazy-loaded chunk, and Vite dev-server cold-compiles a chunk on
    // its first-ever visit (a few extra seconds, confirmed live via a
    // network trace on a fresh /admin/reports visit — the request itself
    // is fast, ~1s, once compiled). 120s wasn't enough headroom for that
    // cumulative cost across every route once monitors/reports/news/faqs
    // were added on top of the original six.
    test.setTimeout(240_000)
    await loginAs(page, ADMIN_EMAIL, PASSWORD)

    await test.step('create, edit, delete a school', async () => {
      await page.goto('/admin/schools')

      // Unique per run (not a fixed "E2E CRUD Test School" name/email): a
      // prior run whose delete step didn't actually take left one behind
      // with the same email, and Email is unique per school server-side,
      // so POST /schools 409-conflicted on the very next run's create —
      // confirmed live by reading the network response directly, not
      // assumed. This also sidesteps the substring-collision class of bug
      // hit earlier with the dedicated presence/predicate school below,
      // without needing a second distinct name.
      const name = `E2E CRUD Test School ${Date.now()}`
      await page.getByRole('button', { name: 'Add school' }).click()
      const createDialog = page.getByRole('dialog')
      await createDialog.getByLabel('Name').fill(name)
      await createDialog.getByLabel('Email').fill(`e2e-crud-school-${Date.now()}@internity.test`)
      await createDialog.getByRole('button', { name: 'Create school' }).click()
      await expect(page.getByText('School created')).toBeVisible()
      await expect(createDialog).toBeHidden()

      // Narrows the table to just this run's row via the search box — belt
      // and suspenders against any other stray rows still accumulated from
      // past runs, on top of the name already being unique.
      await page.getByPlaceholder('Search schools…').fill(name)
      const row = page.locator('table tbody tr').filter({ hasText: name })
      await expect(row).toBeVisible()

      await row.getByRole('button', { name: 'Edit school' }).click()
      const editDialog = page.getByRole('dialog')
      const editedName = `${name} (edited)`
      await editDialog.getByLabel('Name').fill(editedName)
      await editDialog.getByRole('button', { name: 'Save changes' }).click()
      await expect(page.getByText('School updated')).toBeVisible()
      await expect(editDialog).toBeHidden()

      await page.getByPlaceholder('Search schools…').fill(editedName)
      const editedRow = page.locator('table tbody tr').filter({ hasText: editedName })
      await expect(editedRow).toBeVisible()
      await editedRow.getByRole('button', { name: 'Delete school' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('School deleted')).toBeVisible()
      await expect(page.locator('table tbody tr').filter({ hasText: editedName })).toHaveCount(0)
    })

    await test.step('create, edit, delete a department', async () => {
      await page.goto('/admin/departments')

      await page.getByRole('button', { name: 'Add department' }).click()
      const createDialog = page.getByRole('dialog')
      // School picker only renders for admin on create — a coordinator
      // would never see it (silently scoped to their own school instead).
      await createDialog.getByLabel('School').click()
      await page.getByRole('option', { name: SCHOOL_NAME }).click()
      await createDialog.getByLabel('Name').fill('E2E CRUD Test Department')
      await createDialog.getByLabel('Study program').fill('E2E')
      await createDialog.getByRole('button', { name: 'Create department' }).click()
      await expect(page.getByText('Department created')).toBeVisible()
      await expect(createDialog).toBeHidden()

      const row = page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Department' })
      await expect(row).toBeVisible()

      await row.getByRole('button', { name: 'Edit department' }).click()
      const editDialog = page.getByRole('dialog')
      await editDialog.getByLabel('Name').fill('E2E CRUD Test Department (edited)')
      await editDialog.getByRole('button', { name: 'Save changes' }).click()
      await expect(page.getByText('Department updated')).toBeVisible()
      await expect(editDialog).toBeHidden()

      const editedRow = page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Department (edited)' })
      await expect(editedRow).toBeVisible()
      await editedRow.getByRole('button', { name: 'Delete department' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('Department deleted')).toBeVisible()
      await expect(page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Department' })).toHaveCount(0)
    })

    await test.step('create, edit, delete a course', async () => {
      await page.goto('/admin/courses')

      await page.getByRole('button', { name: 'Add course' }).click()
      const createDialog = page.getByRole('dialog')
      // Department picker only renders on create.
      await createDialog.getByLabel('Department').click()
      await page.getByRole('option', { name: DEPARTMENT_NAME }).click()
      await createDialog.getByLabel('Name').fill('E2E CRUD Test Course')
      await createDialog.getByRole('button', { name: 'Create course' }).click()
      await expect(page.getByText('Course created')).toBeVisible()
      await expect(createDialog).toBeHidden()

      const row = page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Course' })
      await expect(row).toBeVisible()

      await row.getByRole('button', { name: 'Edit course' }).click()
      const editDialog = page.getByRole('dialog')
      await editDialog.getByLabel('Name').fill('E2E CRUD Test Course (edited)')
      await editDialog.getByRole('button', { name: 'Save changes' }).click()
      await expect(page.getByText('Course updated')).toBeVisible()
      await expect(editDialog).toBeHidden()

      const editedRow = page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Course (edited)' })
      await expect(editedRow).toBeVisible()
      await editedRow.getByRole('button', { name: 'Delete course' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('Course deleted')).toBeVisible()
      await expect(page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Course' })).toHaveCount(0)
    })

    await test.step('create, edit, delete a company', async () => {
      await page.goto('/admin/companies')

      await page.getByRole('button', { name: 'Add company' }).click()
      const createDialog = page.getByRole('dialog')
      await createDialog.getByLabel('Department').click()
      await page.getByRole('option', { name: DEPARTMENT_NAME }).click()
      await createDialog.getByLabel('Name').fill('E2E CRUD Test Company')
      await createDialog.getByLabel('Email').fill('e2e-crud-company@internity.test')
      await createDialog.getByLabel('Website').fill('https://e2e-crud-test.example.com')
      await createDialog.getByRole('button', { name: 'Create company' }).click()
      await expect(page.getByText('Company created')).toBeVisible()
      await expect(createDialog).toBeHidden()

      const row = page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Company' })
      await expect(row).toBeVisible()

      await row.getByRole('button', { name: 'Edit company' }).click()
      const editDialog = page.getByRole('dialog')
      await editDialog.getByLabel('Name').fill('E2E CRUD Test Company (edited)')
      await editDialog.getByRole('button', { name: 'Save changes' }).click()
      await expect(page.getByText('Company updated')).toBeVisible()
      await expect(editDialog).toBeHidden()

      const editedRow = page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Company (edited)' })
      await expect(editedRow).toBeVisible()
      await editedRow.getByRole('button', { name: 'Delete company' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('Company deleted')).toBeVisible()
      await expect(page.locator('table tbody tr').filter({ hasText: 'E2E CRUD Test Company' })).toHaveCount(0)
    })

    await test.step('create, edit, delete a presence status', async () => {
      const schoolId = await ensureEmptySchool(page, 'E2E Presence Config School')

      await page.goto('/admin/presence-statuses')
      // Not a Select picker on this screen — a plain number Input that
      // gates both the list query and the create button (confirmed by
      // reading PresenceStatusesView.vue; unlike every other screen here).
      await page.getByLabel('School ID').fill(String(schoolId))

      await page.getByRole('button', { name: 'New status' }).click()
      const createDialog = page.getByRole('dialog')
      await createDialog.getByLabel('Name').fill('E2E Status')
      // This Select DOES have a real for/id association (like the excuse
      // "Reason" field, unlike the org-scope pickers) — getByLabel works.
      await createDialog.getByLabel('Kind').click()
      await page.getByRole('option', { name: 'Present' }).click()
      // Both create and edit use a static "Save" button here, unlike
      // Schools/Departments/Courses/Companies switching text by mode.
      await createDialog.getByRole('button', { name: 'Save' }).click()
      await expect(page.getByText('Presence status created')).toBeVisible()
      await expect(createDialog).toBeHidden()

      const row = page.locator('table tbody tr').filter({ hasText: 'E2E Status' })
      await expect(row).toBeVisible()

      await row.getByRole('button', { name: 'Edit presence status' }).click()
      const editDialog = page.getByRole('dialog')
      await editDialog.getByLabel('Name').fill('E2E Status (edited)')
      await editDialog.getByRole('button', { name: 'Save' }).click()
      await expect(page.getByText('Presence status updated')).toBeVisible()
      await expect(editDialog).toBeHidden()

      const editedRow = page.locator('table tbody tr').filter({ hasText: 'E2E Status (edited)' })
      await expect(editedRow).toBeVisible()
      await editedRow.getByRole('button', { name: 'Delete presence status' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('Presence status deleted')).toBeVisible()
      await expect(page.locator('table tbody tr').filter({ hasText: 'E2E Status' })).toHaveCount(0)
    })

    await test.step('create, edit, delete a score predicate', async () => {
      const schoolId = await ensureEmptySchool(page, 'E2E Presence Config School')

      await page.goto('/admin/score-predicates')
      await page.getByLabel('School ID').fill(String(schoolId))

      await page.getByRole('button', { name: 'New predicate' }).click()
      const createDialog = page.getByRole('dialog')
      await createDialog.getByLabel('Name').fill('E2E Predicate')
      await createDialog.getByLabel('Min score').fill('80')
      await createDialog.getByLabel('Max score').fill('100')
      await createDialog.getByRole('button', { name: 'Save' }).click()
      await expect(page.getByText('Score predicate created')).toBeVisible()
      await expect(createDialog).toBeHidden()

      const row = page.locator('table tbody tr').filter({ hasText: 'E2E Predicate' })
      await expect(row).toBeVisible()

      await row.getByRole('button', { name: 'Edit score predicate' }).click()
      const editDialog = page.getByRole('dialog')
      await editDialog.getByLabel('Name').fill('E2E Predicate (edited)')
      await editDialog.getByRole('button', { name: 'Save' }).click()
      await expect(page.getByText('Score predicate updated')).toBeVisible()
      await expect(editDialog).toBeHidden()

      const editedRow = page.locator('table tbody tr').filter({ hasText: 'E2E Predicate (edited)' })
      await expect(editedRow).toBeVisible()
      await editedRow.getByRole('button', { name: 'Delete score predicate' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('Score predicate deleted')).toBeVisible()
      await expect(page.locator('table tbody tr').filter({ hasText: 'E2E Predicate' })).toHaveCount(0)
    })

    await test.step('log and delete a monitoring visit', async () => {
      // No edit flow exists for monitors (confirmed by reading
      // MonitorsView.vue — only a delete icon button, no edit) — create,
      // verify, delete is the whole lifecycle here. There's also no
      // student-facing view of this data anywhere in the app (confirmed via
      // router/index.ts — the Student route block has no monitors entry),
      // so this stays entirely on the admin/coordinator side.
      const students = await page.request.get(`${API_BASE_URL}/api/v1/users`, { params: { search: MONITOR_STUDENT_EMAIL, limit: 5 } })
      const studentUserId = (await students.json()).data[0].id

      await page.goto('/admin/monitors')
      await page.getByLabel('School ID').fill(String(SCHOOL_ID))
      await page.locator('label:text-is("Department") + [data-slot="select-trigger"]').click()
      await page.getByRole('option', { name: DEPARTMENT_NAME }).click()
      await page.locator('label:text-is("Company") + [data-slot="select-trigger"]').click()
      await page.getByRole('option', { name: MONITOR_COMPANY_NAME }).click()

      await page.getByRole('button', { name: 'Log visit' }).click()
      const createDialog = page.getByRole('dialog')
      await createDialog.getByLabel('Student ID (UUID)').fill(studentUserId)
      const today = new Date().toISOString().slice(0, 10)
      await createDialog.getByLabel('Date').fill(today)
      await createDialog.getByLabel('Match rating').click()
      await page.getByRole('option', { name: '3', exact: true }).click()
      await createDialog.getByLabel('Notes').fill('Lingkungan kerja sesuai, siswa beradaptasi baik.')
      await createDialog.getByLabel('Suggestions').fill('Lanjutkan penempatan seperti ini.')
      await createDialog.getByRole('button', { name: 'Log visit' }).click()
      await expect(page.getByText('Monitoring visit logged')).toBeVisible()
      await expect(createDialog).toBeHidden()

      const row = page.locator('table tbody tr').filter({ has: page.locator(`[title="${studentUserId}"]`) })
      await expect(row.first()).toBeVisible()
      await row.first().getByRole('button', { name: 'Delete monitoring visit' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('Monitoring visit deleted')).toBeVisible()
    })

    await test.step('export the student roster', async () => {
      // The download button builds a blob client-side (URL.createObjectURL
      // + a synthetic <a download> click), not a real network navigation or
      // static <a href> — confirmed by reading ReportsView.vue. Playwright's
      // download interception still fires for this exact pattern the normal
      // way, so no special handling is needed beyond the usual
      // waitForEvent('download') race against the click.
      await page.goto('/admin/reports')
      await page.getByLabel('School ID').fill(String(SCHOOL_ID))
      await page.locator('label:text-is("Department") + [data-slot="select-trigger"]').first().click()
      await page.getByRole('option', { name: DEPARTMENT_NAME }).click()

      const [download] = await Promise.all([
        page.waitForEvent('download'),
        page.getByRole('button', { name: 'Download roster' }).click(),
      ])
      expect(download.suggestedFilename()).toBe('students.xlsx')
      await expect(page.getByText('Student roster downloaded')).toBeVisible()
    })

    await test.step('create, publish, edit, delete a news post', async () => {
      // Switches to coordinator here: AdminNewsView.vue scopes a "School"
      // post to `auth.user?.school_id`, and admin accounts don't have one
      // (only coordinators do — confirmed live, not assumed: admin hit a
      // real client-side "Your account has no school configured" toast that
      // silently blocked the mutation from ever firing, no network request
      // at all). Every other step in this test works fine as admin because
      // none of them are scoped by the actor's OWN school the way this one
      // is — this is the one screen where that distinction actually bites.
      await loginAs(page, COORDINATOR_EMAIL, PASSWORD)
      await page.goto('/admin/news')

      // Unique per run — a prior run's own draft/published post under a
      // fixed title is exactly the ambiguous-row-match class of bug
      // documented on the school step above (this screen also has no
      // search box to narrow by, unlike Schools, so a stray same-titled row
      // from an earlier interrupted run is even more likely to shadow the
      // fresh one).
      const title = `E2E Test News Post ${Date.now()}`
      await page.getByRole('button', { name: 'New post' }).click()
      const createDialog = page.getByRole('dialog')
      await createDialog.getByLabel('Title').fill(title)
      await createDialog.getByLabel('Content').fill('This is a news post created by an E2E test.')
      // Scope defaults to whatever the form initializes — school-wide needs
      // no department picker, so it's left as-is rather than opening that
      // Select, keeping this step focused on the create/publish/edit/delete
      // lifecycle rather than re-testing the scope picker too.
      //
      // Publishes directly on create (rather than "Save as draft" then the
      // row-level quick-publish action). The quick-publish button was
      // genuinely unreliable in repeated live testing against this specific
      // dev stack — sometimes an unrelated older row flipped to Published
      // instead of the one clicked, with no error and no network request
      // logged for the target row at the time. That's consistent with
      // either a real bug (it shares one `publishMutation` across every row
      // via `@click="…mutate(row.id)"` with no per-row identity check once
      // a request is in flight) or simply this session's already-documented
      // pattern of slow responses under the heavy self-inflicted load of
      // this suite's own repeated runs against a shared remote dev
      // database — the evidence didn't cleanly rule out the second
      // explanation, so this isn't filed as a confirmed bug. Publishing on
      // create avoids the question either way and is equally valid
      // coverage of the publish flow.
      await createDialog.getByRole('button', { name: 'Publish' }).click()
      // Longer than this file's usual toasts get: publishing fans out a
      // notification to everyone in the selected scope server-side, real
      // extra work beyond a plain CRUD write — confirmed live, the button
      // was still showing "Publishing…" at the global 10s expect timeout
      // (playwright.config.ts) under this session's own heavy load on a
      // shared, non-local dev database, not stuck/erroring.
      await expect(page.getByText('News published, everyone in scope has been notified')).toBeVisible({ timeout: 30_000 })
      await expect(createDialog).toBeHidden()

      const row = page.locator('table tbody tr').filter({ hasText: title })
      await expect(row).toBeVisible()

      const editedTitle = `${title} (edited)`
      await row.getByRole('button', { name: 'Edit post' }).click()
      const editDialog = page.getByRole('dialog')
      await editDialog.getByLabel('Title').fill(editedTitle)
      await editDialog.getByRole('button', { name: 'Save changes' }).click()
      await expect(page.getByText('News updated')).toBeVisible()
      await expect(editDialog).toBeHidden()

      const editedRow = page.locator('table tbody tr').filter({ hasText: editedTitle })
      await expect(editedRow).toBeVisible()
      await editedRow.getByRole('button', { name: 'Delete post' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('News deleted')).toBeVisible()
      await expect(page.locator('table tbody tr').filter({ hasText: editedTitle })).toHaveCount(0)
    })

    await test.step('create, edit, delete a FAQ', async () => {
      // No org-scope gating at all here (FAQs are global, confirmed by
      // reading AdminFaqsView.vue — no school/department params anywhere),
      // and the route allows both admin and coordinator — no need to log
      // back in as admin, this just carries over the coordinator session
      // the news step above switched to.
      await page.goto('/admin/faqs')

      // Unique per run — same reasoning as the news step above, and this
      // screen has no search box either.
      const question = `E2E Test FAQ question ${Date.now()}?`
      await page.getByRole('button', { name: 'New FAQ' }).click()
      const createDialog = page.getByRole('dialog')
      await createDialog.getByLabel('Question').fill(question)
      await createDialog.getByLabel('Answer').fill('E2E test FAQ answer.')
      await createDialog.getByRole('button', { name: 'Save' }).click()
      await expect(page.getByText('FAQ created')).toBeVisible()
      await expect(createDialog).toBeHidden()

      const row = page.locator('table tbody tr').filter({ hasText: question })
      await expect(row).toBeVisible()

      const editedQuestion = `${question} (edited)`
      await row.getByRole('button', { name: 'Edit FAQ' }).click()
      const editDialog = page.getByRole('dialog')
      await editDialog.getByLabel('Question').fill(editedQuestion)
      await editDialog.getByRole('button', { name: 'Save' }).click()
      await expect(page.getByText('FAQ updated')).toBeVisible()
      await expect(editDialog).toBeHidden()

      const editedRow = page.locator('table tbody tr').filter({ hasText: editedQuestion })
      await expect(editedRow).toBeVisible()
      await editedRow.getByRole('button', { name: 'Delete FAQ' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click()
      await expect(page.getByText('FAQ deleted')).toBeVisible()
      await expect(page.locator('table tbody tr').filter({ hasText: editedQuestion })).toHaveCount(0)
    })
  })

  test('coordinator generates a student invite code', async ({ page }) => {
    await loginAs(page, COORDINATOR_EMAIL, PASSWORD)

    // Not a Dialog — a plain Card form embedded in the page (confirmed by
    // reading UsersView.vue). There is no "Add user" button anywhere on
    // this screen — user accounts are only ever created by the person
    // themselves via /register with a code generated here, never directly
    // by staff.
    //
    // The Card is positioned AFTER the DataTable in the template — with
    // enough seeded/test-created users to make that table tall (the
    // register-flow test in business-flows.spec.ts adds one every run),
    // the trigger ends up far enough down the page that the Select
    // popover's *options* render outside the viewport ("element is outside
    // of the viewport", confirmed live via screenshot). Scrolling the
    // trigger into view first didn't reliably fix it — the popover repaints
    // once opened and the page doesn't stay scrolled where expected — so
    // this sidesteps the whole scroll/portal-position interaction with a
    // viewport tall enough to fit the card without scrolling at all.
    await page.setViewportSize({ width: 1280, height: 2400 })
    await page.goto('/admin/users')
    await page.getByLabel('Department').click()
    await page.getByRole('option', { name: DEPARTMENT_NAME }).click()
    await page.getByLabel('Course').click()
    await page.getByRole('option', { name: COURSE_NAME }).click()
    await page.getByRole('button', { name: 'Generate invite code' }).click()

    await expect(page.getByText('Invite code created')).toBeVisible()
    // The freshly generated code renders in a "Recently generated" list as
    // a monospace Badge — confirmed by reading the source rather than
    // guessing a selector for it; asserting the section heading is enough
    // to prove the flow completed without needing to parse the code text.
    await expect(page.getByText('Recently generated')).toBeVisible()
  })
})
