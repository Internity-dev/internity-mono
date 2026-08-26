# Deployment & Local Dev

Monorepo: `apps/api` (Go/Gin/GORM), `apps/dashboard` (Vue 3 + Vite), `apps/landing` (Nuxt, static
preset), `apps/e2e` (Playwright), `packages/design-tokens` (shared Tailwind v4 theme, consumed by
both frontends via `@internity/design-tokens/theme.css`).

Three compose files, all in `deploy/`, all defining the same nine services (postgres, redis,
minio, minio-init, migrate, api, worker, dashboard, landing):

| File | Used by | Purpose |
|---|---|---|
| `deploy/docker-compose.yml` | `make dev` | Local dev — live-reloading dev servers, source bind-mounts, host ports published |
| `deploy/docker-compose.prod.yml` | `make prod` | Generic hardened prod — static builds behind nginx, host ports published |
| `deploy/docker-compose.dokploy.yml` | Dokploy platform | Same app as `.prod.yml`, but joins `dokploy-network` and uses `expose` instead of `ports:` (Traefik routes in) |

## Local dev

```
cp .env.example .env
make dev          # = docker compose --project-directory . --env-file .env -f deploy/docker-compose.yml up --build
```

Makefile: `d:\Project\Mumtaz\teesatas\Makefile:1-8`. `make down` stops containers but keeps named
volumes (`pgdata`, `miniodata`).

**Topology** (`deploy/docker-compose.yml`):
- `postgres` (postgres:16-alpine) → host port `${POSTGRES_PORT:-5432}`
- `redis` (redis:7-alpine) → host port `${REDIS_PORT:-6379}`
- `minio` (minio/minio:latest) → host ports `${MINIO_PORT:-9000}` (API) and `${MINIO_CONSOLE_PORT:-9001}` (console)
- `minio-init` — one-shot `mc` container, creates 4 buckets (`internity-avatars`, `internity-attachments`, `internity-documents`, `internity-logos`), sets `avatars`/`logos` to public-download, then exits
- `migrate` — one-shot `migrate/migrate`, runs `apps/api/migrations` against postgres, then exits
- `api` — built from `apps/api/Dockerfile`, host port `${API_PORT:-8080}`, waits on `migrate` (`service_completed_successfully`), `redis` and `minio-init` (`service_healthy`/`service_completed_successfully`)
- `worker` — same image as `api` (`internity-api:local`), overridden `entrypoint: ["/app/worker"]`, no healthcheck (no HTTP listener) and no published port
- `dashboard` — `apps/dashboard/Dockerfile` (dev-only, runs `pnpm dev -- --host 0.0.0.0`), host port `${DASHBOARD_PORT:-5173}`, bind-mounts `apps/dashboard/src` and `packages/design-tokens` from host
- `landing` — `apps/landing/Dockerfile` (dev-only, `pnpm dev -- --host 0.0.0.0`), host port `${LANDING_PORT:-3000}`, bind-mounts `apps/landing/app` and `packages/design-tokens` from host

Full file: `d:\Project\Mumtaz\teesatas\deploy\docker-compose.yml`.

**Gotcha — design tokens must be built once on the host before/alongside `make dev`.**
`packages/design-tokens/dist/` is gitignored (`.gitignore:14`, bare `dist/`) and only produced by
`pnpm run tokens:build` (root `package.json:13`, runs `pnpm --filter @internity/design-tokens
build` → `packages/design-tokens/scripts/build.mjs`). Both `apps/dashboard/src/assets/main.css:5`
and `apps/landing/app/assets/css/main.css:2` do `@import "@internity/design-tokens/theme.css"`,
which resolves to `packages/design-tokens/dist/theme.css`. The dev `dashboard`/`landing` containers
**bind-mount `../packages/design-tokens` straight from the host** (`docker-compose.yml:135`,
`:150`) — the dev Dockerfiles (`apps/dashboard/Dockerfile`, `apps/landing/Dockerfile`) never run
`tokens:build` themselves, so on a fresh clone with no host-side `dist/` the mounted directory has
no `theme.css` and Vite's dev server fails resolving that import. Run `pnpm install && pnpm run
tokens:build` from the repo root before `make dev` (or any time `packages/design-tokens/tokens.json`
changes). Not mentioned in the README's "Getting started" section — easy to miss.

**Alternative to Docker for iterating on API/dashboard code:** root `package.json` has a `dev`
script (`concurrently -n api,dashboard,landing ...`) that runs `go run ./cmd/api`,
`pnpm --filter @internity/dashboard dev`, `pnpm --filter @internity/landing dev` directly on the
host — still needs postgres/redis/minio reachable (e.g. via `docker compose -f
deploy/docker-compose.yml up postgres redis minio minio-init migrate` for just the infra, or
already-running `make dev` containers) and still needs `tokens:build` run first.

**README caveat worth repeating:** the README states the `docker compose up` path for `make dev`
"hasn't been run in this environment, which never had Docker installed" — the pieces were verified
individually (API+dashboard against live Postgres/Redis, Playwright end to end) but not the full
compose boot. Treat a clean `make dev` as unverified until you've actually run it; flag anything
that doesn't come up clean on first run (README.md:186-191).

### Migrations & seeding

```
make migrate-up                        # apps/api/migrations, via golang-migrate CLI against DATABASE_URL
make migrate-down
make migrate-create name=add_foo_table
make seed                              # cd apps/api && go run ./cmd/seed
```

`apps/api/cmd/seed/main.go:1-12` — every insert is idempotent (lookup-by-natural-key-or-insert for
low-volume tables, load-existing-then-filter for bulk ones), so re-running `make seed` against an
already-seeded DB just fills in gaps, never duplicates or errors. Seeds one school (3 courses), 25
companies, ~150 students, vacancies, appliances across all 5 statuses, intern placements with
matching presence/journal/score/certificate history, news/FAQs/monitoring/reviews/notifications.
All seeded accounts share password `password123` (`demoPassword` const, seed/main.go:31); see
README.md:238-245 for the specific login table (admin@internity.test, coordinator@internity.test,
mentor1@internity.test, mentor2@internity.test, budi@internity.test, siti@internity.test).

## CI / local verification loop

Read before pushing — from `Makefile:31-46`:

```
make lint             # cd apps/api && go vet ./... && gofmt -l .   +   pnpm -r --if-present lint
make test-api          # cd apps/api && go test ./... -race -cover
make test-integration   # cd apps/api && go test -tags=integration ./...   (needs Docker — testcontainers spins up real Postgres)
make test-dashboard     # pnpm --filter @internity/dashboard test:unit   (Vitest)
make test-e2e            # pnpm --filter @internity/e2e test   (Playwright; needs `make dev` + `make seed` already running — playwright.config.ts:41 defaults baseURL to http://localhost:5173, override with E2E_BASE_URL)
```

`go vet` + `gofmt -l .` will not fail the make invocation on a gofmt diff by itself (`gofmt -l`
only lists files, doesn't exit non-zero) — actually check its output, an empty line means clean;
any listed path means unformatted files remain.

## Production (generic — `deploy/docker-compose.prod.yml`)

```
cp .env.prod.example .env.prod   # fill in real secrets + public URLs
make prod                         # docker compose --project-directory . --env-file .env.prod -f deploy/docker-compose.prod.yml up --build -d
make prod-down
```

Same nine services as dev, hardened:
- `dashboard`/`landing` build from `Dockerfile.prod` (multi-stage: `pnpm run build` → static output
  served by `nginxinc/nginx-unprivileged:alpine`, port 8080 internally) instead of the dev
  Dockerfiles' live-reload dev servers — no source bind-mounts.
- `apps/api/Dockerfile` runs as non-root (`adduser -D -u 10001 appuser`, `Dockerfile:18,21`).
- `APP_ENV=production`, `COOKIE_SECURE=true` baked into the compose file itself
  (`docker-compose.prod.yml:101,103`).
- Every secret is `${VAR:?VAR is required}` — compose refuses to start and names the missing var,
  no silent weak fallback (contrast with dev compose's `${VAR:-default}` pattern).
- postgres/redis/minio have **no published host ports**; only api/dashboard/landing do.
- `restart: unless-stopped` + `deploy.resources.limits` (cpu/memory) on every long-running service.
- `minio` and `migrate` pinned to specific tags (`RELEASE.2025-09-07T16-13-09Z-cpuv1`, `v4.19.1`)
  instead of `latest`.
- **`docker-compose.prod.yml` does NOT set `COOKIE_DOMAIN`** on the `api` service — this variant
  assumes dashboard and API can share a host/scheme setup where host-only CSRF cookies still work,
  or that you'll add it yourself. Only `docker-compose.dokploy.yml` sets it, because Dokploy's
  standard setup puts api/dashboard/landing on different subdomains. If you deploy `.prod.yml`
  across subdomains, add `COOKIE_DOMAIN` to the `api` service's environment yourself — see the CSRF
  section below.

Full file: `d:\Project\Mumtaz\teesatas\deploy\docker-compose.prod.yml`.

## Production on Dokploy (`deploy/docker-compose.dokploy.yml`)

Full walkthrough: `d:\Project\Mumtaz\teesatas\docs\dokploy.md`. Not build-tested against a live
Dokploy instance (no Dokploy access when written) — reviewed against Dokploy's documented Compose
behavior. Verify on first real deploy rather than assuming.

Setup summary:
1. **Create Project → Compose** in Dokploy, pointed at this repo, **Compose Path:**
   `deploy/docker-compose.dokploy.yml`.
2. Copy `.env.dokploy.example`, fill in real values, paste the whole block into Dokploy's
   **Environment** tab. Dokploy writes it to a `.env` next to the compose file; Compose's own
   `${VAR}` interpolation reads it — same mechanism as `--env-file` locally.
3. Trigger deploy. Builds `api`'s shared api/worker image + `dashboard`'s and `landing`'s
   `Dockerfile.prod`. `migrate`/`minio-init` run once and exit; `api`/`worker` wait on
   `service_completed_successfully` (migrate) and `service_healthy` (redis) first.
4. **Domains tab**, one domain per service, **Container Port `8080` for all three** — `api` via its
   own `EXPOSE 8080`, `dashboard`/`landing` via `nginx-unprivileged` (dokploy.md:56-67):

   | Service | Suggested domain |
   |---|---|
   | `api` | `api.yourdomain.com` |
   | `dashboard` | `app.yourdomain.com` |
   | `landing` | `yourdomain.com` |

5. If uploaded files need direct browser access instead of proxying through the API, add a domain
   for `minio` on container port `9000` and point `DASHBOARD_MINIO_URL` at it.

Dokploy-specific compose differences from `.prod.yml` (all platform convention, zero app changes,
per the file's own header comment, `docker-compose.dokploy.yml:3-19`): every service joins the
external `dokploy-network` (Dokploy's own Traefik listens there) instead of getting a default
bridge network; `api`/`dashboard`/`landing` use `expose:` not `ports:` (Dokploy's Domains tab
routes hostname → container port over `dokploy-network` directly); no host ports anywhere.

Troubleshooting (docs/dokploy.md:85-96): a service that can't resolve another by name → confirm
both are actually on `dokploy-network` (`docker network inspect dokploy-network` on the host);
`dokploy-network` missing → `docker network create dokploy-network` recreates it (normally
auto-created by the Dokploy installer); compose refusing to start citing a missing var → that's the
`${VAR:?message}` guard doing its job, add the var to the Environment tab and redeploy.

## Three bugs fixed this session (context for touching this setup again)

### 1. `redis.ParseURL` silently defaulting to `localhost` for bare `host:port`

`apps/api/internal/platform/redisx/redis.go:13-28`. Every compose file sets `REDIS_URL=redis:6379`
(bare host:port, no scheme, no auth — internal Docker network, no password). The bug: calling
`redis.ParseURL("redis:6379")` directly does **not** error on this input — `"redis"` happens to be
one of `ParseURL`'s own recognized URL schemes, so it parses `"redis:6379"` as
scheme=`redis`, and silently defaults `Addr` to `localhost:6379`, discarding the `6379` port
component entirely (it gets consumed as something else, not as the port). The client then connects
to `localhost:6379` inside the container instead of the `redis` service — which may or may not even
have anything listening, so this can fail in a confusing way far from the actual cause, or in the
worst case silently connect to an unrelated local Redis.

Fix: `resolveOptions` (redisx/redis.go:23-28) checks for the literal substring `"://"` **before**
ever calling `redis.ParseURL`. Only strings containing `"://"` (i.e. an actual
`redis://[:password@]host:port[/db]` URL, needed for password-protected managed Redis) go through
`ParseURL`; anything else is treated as `&redis.Options{Addr: connStr}` directly. Any future change
to this function must preserve that ordering — do not "simplify" it back to calling `ParseURL`
first and falling back on error, because `ParseURL` does not error here.

### 2. `packages/design-tokens/dist` gitignored — Docker prod builds need an explicit `tokens:build` step

Already fixed in both `apps/dashboard/Dockerfile.prod:9-15` and
`apps/landing/Dockerfile.prod:11-17` — each copies `packages/design-tokens` into the build stage,
runs `pnpm install --frozen-lockfile`, **then** `pnpm run tokens:build` (root script, invokes
`packages/design-tokens/scripts/build.mjs`), all *before* `pnpm run build` for the app itself.
Without that step the app's own build fails resolving `@internity/design-tokens/theme.css` (both
apps' `main.css` import it directly, see the Local Dev gotcha above). If either Dockerfile.prod is
ever regenerated or refactored, keep the `tokens:build` RUN line — it is easy to drop when it looks
like "just another install step" since nothing else in the Dockerfile visibly depends on it.

The dev Dockerfiles (`apps/dashboard/Dockerfile`, `apps/landing/Dockerfile`) do **not** have this
step — they rely on the host doing it once and the container's bind-mount picking it up (see Local
Dev gotcha above). Don't assume symmetry between dev and prod Dockerfiles here.

### 3. `COOKIE_DOMAIN` required whenever dashboard and API are on different subdomains

`apps/api/internal/config/config.go:17-26` (field comment) and
`apps/api/internal/middleware/csrf.go:22-29` (mechanism). CSRF protection is double-submit-cookie:
login sets a non-HttpOnly `internity_csrf` cookie (`identity.Handler.setAuthCookies`,
`apps/api/internal/modules/identity/handler.go:272-282`) that the dashboard's JS must read via
`document.cookie` and echo back as the `X-CSRF-Token` header on every mutating request
(`RequireCSRF`, `middleware/csrf.go:30-46`); session/refresh cookies stay HttpOnly and host-only on
purpose, only the CSRF cookie is domain-scoped.

If `COOKIE_DOMAIN` is unset (empty string, the default), the CSRF cookie is host-only —
scoped to whatever host the API itself responds on (e.g. `api.example.com`). A dashboard served
from a *different* host (e.g. `app.example.com`) can never read that cookie via `document.cookie`
(browsers don't share host-only cookies across hosts), so it can never produce a matching
`X-CSRF-Token` header, and **every mutating request (POST/PUT/PATCH/DELETE) gets rejected with 403
"CSRF token missing or invalid"** — while GETs keep working fine, which makes this look like a
targeted bug in whatever mutation you happen to be testing rather than a global cookie-scope
misconfiguration.

Required value: the shared parent domain of dashboard and API, e.g. API on `api.example.com` +
dashboard on `app.example.com` → `COOKIE_DOMAIN=.example.com` (leading dot optional, `example.com`
and `.example.com` behave the same — `.env.dokploy.example:35`). Leave unset **only** when
dashboard and API share the exact same host. Wired: `cfg.CookieDomain` → `identity.NewHandler(...,
cfg.CookieDomain)` (`apps/api/cmd/api/main.go:245`) → `Handler.cookieDomain` → both `SetCookie`
calls for `CookieCSRF` in `setAuthCookies`/`clearAuthCookies`.

Set in `deploy/docker-compose.dokploy.yml:132` (`COOKIE_DOMAIN: ${COOKIE_DOMAIN:-}`, defaults to
empty — you must supply it via the Dokploy Environment tab, see `.env.dokploy.example:29-35`).
**Not set at all** in `deploy/docker-compose.prod.yml`'s `api` service — see the Production section
above.

## Required production env vars

Source of truth: `.env.dokploy.example` (Dokploy) and `.env.prod.example` (generic prod) — both
enforce every var as `${VAR:?required}` in their respective compose files, so an incomplete
`.env`/`.env.prod` fails the deploy immediately with the missing var's name rather than starting in
a half-configured state.

| Var | Load-bearing? | Notes |
|---|---|---|
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | Required | No fallback in prod/dokploy compose |
| `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD` | Required | Same |
| `CORS_ALLOWED_ORIGINS` | Required | Comma-separated exact origins (scheme+host), matched literally in `middleware.CORS` (`cors.go:15-19`) — must exactly equal what the browser sends as `Origin`, no wildcards (cookies + `Access-Control-Allow-Credentials: true` forbid `*` origin, see `cors.go:9-14`) |
| `COOKIE_DOMAIN` | **Required whenever dashboard and API are on different subdomains** (see bug #3 above); optional/omit only if same host | Dokploy compose only; add manually to `.prod.yml` if needed |
| `COOKIE_SECURE` | Hardcoded `true` in both prod compose files (not an env var to set — baked into `docker-compose.prod.yml:103` / `docker-compose.dokploy.yml:131`) | — |
| `DASHBOARD_API_BASE_URL`, `DASHBOARD_MINIO_URL` | Required, **build-time only** | Passed as Docker build `args` (`Dockerfile.prod ARG`/`ENV`), Vite bakes `VITE_*` vars into the JS bundle at `pnpm run build` time — **changing these after the image is built does nothing; you must rebuild the `dashboard` image**, not just edit the env var and restart the container. This is the single most common trap: editing `.env`/Dokploy env and restarting without rebuilding leaves the dashboard silently pointed at the old API/MinIO URL. |
| `LANDING_API_BASE_URL`, `LANDING_DASHBOARD_URL` | Required, **build-time only** | Same trap — Nuxt's `static` preset (`nitro.preset`) has no server process to read runtime config from; `runtimeConfig.public` is baked into the prerendered payload at `pnpm run build` time. Rebuild `landing` image to change. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Optional | Unset disables tracing entirely, no network calls attempted (`config.go:34` comment) |
| `MINIO_USE_SSL` | Optional, defaults `false` | Exists in `config.Config` (`config.go:32`) but **no compose file sets it** — internal MinIO traffic is plain HTTP inside the Docker network in every environment currently defined |

`apps/api/internal/config/config.go:58-71` — the *only* vars the Go process itself hard-fails on
missing when `APP_ENV=production` are `DATABASE_URL` and `REDIS_URL` (both always set by every
compose file via service-name DNS, so this rarely fires in practice); everything else in the table
above is enforced by the compose file's `${VAR:?}` guard instead, one layer up.

## The seed binary is baked into the production image

`apps/api/Dockerfile:1-22` builds three binaries in the same builder stage — `api`, `worker`, and
`seed` (`RUN ... go build -o /out/seed ./cmd/seed`, `Dockerfile:14`) — and copies all three into
the final `alpine:3.20` runtime image (`Dockerfile:20`). No compose service runs `seed` by default
(no `entrypoint`/`command` targets it). It's shipped specifically so you can seed/reseed demo data
against a **live** container using that container's already-configured `DATABASE_URL`, no separate
throwaway image or manually reconstructed connection string:

```
docker exec -it <api-or-worker-container-name> /app/seed
```

Works against either the `api` or `worker` container since both run from the same
`internity-api:{local,prod,dokploy}` image. Safe to re-run (idempotent, see Seeding section above).
