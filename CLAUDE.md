# Internity

PKL (Praktik Kerja Lapangan — Indonesian vocational-school internship) management
platform. Monorepo: `apps/api` (Go/Gin/GORM/Postgres), `apps/dashboard` (Vue 3 +
TanStack Query + shadcn-vue, the main authenticated app), `apps/landing` (Nuxt,
public marketing site), `apps/e2e` (Playwright), `packages/design-tokens`
(shared theme, `tokens.json` -> `pnpm tokens:build` -> `dist/theme.css`, gitignored
dist so a fresh build must run tokens:build first).

## Read before working here

`docs/knowledge/gotchas.md` first, always — a short list of real bugs already
hit in this repo and how to avoid repeating them. Then whichever of these
actually applies to the task, read the whole file rather than re-deriving it
by grepping around (each is verified against real source, with file:line
citations):

- `docs/knowledge/backend-architecture.md` — Go module layout, the
  domain/repository/service/handler/routes split, httpx conventions, RBAC
  patterns, cross-module adapter wiring, migrations, caching, testing convention.
- `docs/knowledge/frontend-architecture.md` — Vue app structure, `useListQuery`
  (server-side search/filter/pagination, URL-synced) and its circular-init
  `enabled` gotcha, auth/CSRF interceptor flow, onboarding tour system, design
  tokens pipeline.
- `docs/knowledge/domain-model.md` — the actual PKL business domain: org
  hierarchy, roles and their scope columns, the full student lifecycle
  (vacancy -> apply -> accept -> placement -> presence/journal -> score ->
  certificate), review relationships, multi-tenancy quick-reference table.
- `docs/knowledge/auth-and-security.md` — sessions, CSRF, rate limiting, RBAC
  enforcement, the two CSRF bugs and the review-scoping bug fixed this
  project's history.
- `docs/knowledge/testing.md` — how to actually run each test suite, seeded
  accounts, Playwright gotchas, shared-dev-DB hazards (fixture pollution, rate
  limiting under concurrent load).
- `docs/knowledge/deployment.md` — local dev, Dokploy production deploy,
  required env vars, three real deploy bugs already fixed.
- `docs/knowledge/design-system.md` — brand tokens, typography, dark mode,
  component library conventions, real available brand assets.

`docs/RULES.md` is binding, not just reference — read it too.

## Quick orientation

Seeded accounts (password `password123` for all): `admin@internity.test`,
`coordinator@internity.test`, `mentor1@internity.test` (has rich real seed
data), `budi@internity.test` (student). Full list and how seeding works:
`apps/api/cmd/seed/main.go`.

Local dev: `make dev` (docker-compose) or run `apps/api` and `apps/dashboard`
directly — see `deployment.md` for exact commands. Backend tests: `go test
./...` in `apps/api`. Frontend: `vue-tsc --build` + `vitest run` in
`apps/dashboard`. Both must be clean before any change is considered done —
see `docs/RULES.md` #3, #14.
