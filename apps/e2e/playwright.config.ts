import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright config for Internity's Playwright critical-path E2E suite.
 *
 * This suite is designed to run against the already-composed docker-compose
 * stack (`make dev`, i.e. `docker compose --env-file .env -f
 * deploy/docker-compose.yml up --build`, plus `make seed` for demo data) —
 * see the root Makefile's `test-e2e` target. It deliberately does NOT
 * configure Playwright's `webServer` option: this package never boots its
 * own dashboard/API/Postgres/Redis/MinIO, it only drives a browser against
 * whatever is already listening at `baseURL`.
 *
 * `E2E_BASE_URL` mirrors the dashboard's docker-compose port. Per
 * `.env.example` / `deploy/docker-compose.yml`, `DASHBOARD_PORT` defaults to
 * 5173, so that's the default here too.
 */
export default defineConfig({
  testDir: './tests',
  // The critical-path spec is one stateful, sequential flow (login -> apply
  // -> accept -> schedule -> attend/journal -> review -> score ->
  // certificate) — parallelizing across files/workers isn't meaningful for
  // it, and running two copies against the same seeded accounts at once
  // would race on shared state (the same student, the same vacancy).
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  timeout: 60_000,
  // Playwright's own default (5s) is tight for mutations against this
  // stack's actual dev database, which is remote (not localhost) and, in
  // this session's own testing, has shown occasional multi-second
  // latency spikes under concurrent load — confirmed live via a toast
  // assertion timing out while its dialog's submit button was still
  // showing "Saving…", not an app bug. 10s gives real slow responses room
  // without meaningfully slowing down a suite where most assertions
  // resolve in milliseconds anyway.
  expect: { timeout: 10_000 },
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  // Two projects rather than one so `critical-path` always finishes before
  // the rest start, regardless of file discovery order (which is otherwise
  // alphabetical — admin-crud, business-flows, THEN critical-path — meaning
  // a bare `npx playwright test` would always run critical-path.spec.ts
  // LAST). business-flows.spec.ts's excuse/performance-review scenarios
  // need an *active* placement that only critical-path.spec.ts's own full
  // apply -> accept -> schedule flow normally creates; confirmed live, not
  // theoretical — this broke on the very first run against a genuinely
  // untouched company, where no leftover placement from an earlier session
  // happened to paper over the ordering gap. (business-flows.spec.ts's own
  // ensureActivePlacement helper still self-heals if run without this
  // ordering — e.g. business-flows.spec.ts alone, in isolation — but that
  // path costs several extra logins against the auth rate limiter, so
  // letting critical-path go first here is what keeps a full suite run
  // cheap rather than just "eventually correct".)
  projects: [
    {
      name: 'critical-path',
      testMatch: 'critical-path.spec.ts',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'chromium',
      testIgnore: 'critical-path.spec.ts',
      dependencies: ['critical-path'],
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
