# Deploying to Dokploy

This uses `deploy/docker-compose.dokploy.yml`, a Dokploy-specific variant of
`deploy/docker-compose.prod.yml`. The app itself is unchanged between the two;
the differences are Dokploy platform conventions:

- Every service joins `dokploy-network`, the shared network Dokploy's own
  Traefik instance listens on, instead of getting its own default network.
  That's what lets Traefik reach `api`/`dashboard`/`landing` directly, and
  lets those containers still resolve `postgres`/`redis`/`minio` by service
  name.
- `api`/`dashboard`/`landing` use `expose` instead of `ports:` — nothing is
  published to the host. Dokploy's Domains tab routes a public hostname
  straight to a container port over `dokploy-network`, so a host port mapping
  would be redundant, and would fight with other projects on the same
  Dokploy server for that port.
- No host ports anywhere, including `postgres`/`redis`/`minio`, since nothing
  outside this compose project needs to reach them directly.

This hasn't been deployed against a live Dokploy instance in this
environment (no Dokploy access here) — it's written to match Dokploy's
documented Docker Compose and Domains behavior as closely as possible. Say so
plainly rather than claiming a verification that didn't happen; flag anything
that doesn't come up clean on first deploy.

## 1. Create the project

In Dokploy: **Create Project → Compose**, point it at this Git repository,
and set:

- **Compose Path:** `deploy/docker-compose.dokploy.yml`
- **Build Path / Context:** repository root (the compose file's own relative
  `context: ..` / `context: ../apps/api` paths assume Dokploy checks out the
  whole repo, the same way `make dev`/`make prod` do locally)

## 2. Paste the environment

Copy `.env.dokploy.example`, fill in real secrets and your real domains, then
paste the whole block into Dokploy's **Environment** tab for this project.
Dokploy writes it to a `.env` file next to the compose file, which Compose's
own `${VAR}` interpolation already reads, exactly like running
`docker compose --env-file .env.dokploy up` locally. Every var is required
(no weak fallback), so a missing one fails the deploy with a clear error
naming which one, instead of silently starting with a wrong default.

## 3. Deploy

Trigger the deploy. Dokploy builds all four images (`api`'s shared
api/worker image, plus `dashboard`'s and `landing`'s `Dockerfile.prod`) and
brings the stack up. `migrate` and `minio-init` are one-shot: they run,
finish, and exit — `api`/`worker` wait on `service_completed_successfully`
for `migrate` and `service_healthy` for `redis`, so they don't start against
a not-yet-migrated database.

## 4. Add domains

Dokploy's Domains tab injects the Traefik labels itself — nothing to add to
the compose file. For each of these three services, add a domain and set
**Container Port** to `8080` (all three, including `api`, listen on 8080
internally — `api` via its own `EXPOSE 8080`, `dashboard`/`landing` via the
`nginx-unprivileged` image both `Dockerfile.prod`s serve from):

| Service | Suggested domain | Container port |
|---|---|---|
| `api` | `api.yourdomain.com` | 8080 |
| `dashboard` | `app.yourdomain.com` | 8080 |
| `landing` | `yourdomain.com` | 8080 |

Match these to whatever you put in `.env.dokploy.example`'s
`CORS_ALLOWED_ORIGINS`, `DASHBOARD_API_BASE_URL`, `DASHBOARD_MINIO_URL`,
`LANDING_API_BASE_URL`, and `LANDING_DASHBOARD_URL` before the first build —
the dashboard and landing images bake those URLs in at build time (Vite env
vars and Nuxt's static-prerendered public runtime config are both
compile-time, not runtime), so changing a domain after the fact means
rebuilding those two images, not just editing an env var and restarting.

## 5. MinIO's own public URL

If uploaded files (avatars, attachments) need to be reachable directly from
a browser rather than proxied through the API, also add a domain for `minio`
pointing at container port `9000`, and use that domain as
`DASHBOARD_MINIO_URL`. If everything goes through the API instead, this step
isn't needed.

## Troubleshooting

- **A service can't resolve another by name.** Confirm both are actually on
  `dokploy-network` — `docker network inspect dokploy-network` on the Dokploy
  host should list every container in this stack.
- **`dokploy-network` doesn't exist.** It's created automatically when
  Dokploy itself is installed; if it's somehow missing,
  `docker network create dokploy-network` on the host recreates it.
- **Compose refuses to start citing a missing variable.** That's the
  `${VAR:?message}` required-var check in the compose file doing its job —
  add the named variable to the Environment tab and redeploy.
