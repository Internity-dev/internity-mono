# syntax=docker/dockerfile:1
# Dev image only (local docker-compose demo — see plan's deployment decision).
# Build context is the monorepo root so pnpm workspaces (design-tokens) resolve.
FROM node:22-alpine
WORKDIR /repo
RUN corepack enable
COPY pnpm-workspace.yaml package.json pnpm-lock.yaml* ./
COPY packages/design-tokens ./packages/design-tokens
COPY apps/landing ./apps/landing
RUN pnpm install --frozen-lockfile=false
WORKDIR /repo/apps/landing
EXPOSE 3000
CMD ["pnpm", "dev", "--", "--host", "0.0.0.0"]
