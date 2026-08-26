COMPOSE = docker compose --project-directory . --env-file .env -f deploy/docker-compose.yml
DATABASE_URL ?= postgres://internity:internity@localhost:5432/internity?sslmode=disable

.PHONY: dev down migrate-up migrate-down migrate-create seed test-api test-dashboard test-e2e lint

dev: ## Build + start the full stack (postgres, redis, minio, api, worker, dashboard, landing)
	$(COMPOSE) up --build

down: ## Stop and remove all containers (keeps named volumes)
	$(COMPOSE) down

migrate-up: ## Apply all pending migrations against DATABASE_URL
	migrate -database "$(DATABASE_URL)" -path apps/api/migrations up

migrate-down: ## Roll back the most recent migration
	migrate -database "$(DATABASE_URL)" -path apps/api/migrations down 1

migrate-create: ## Create a new migration pair: make migrate-create name=add_foo_table
	migrate create -ext sql -dir apps/api/migrations -seq $(name)

seed: ## Load minimal demo data (1 school -> 1 department -> 1 course -> 2 companies -> one user per role)
	cd apps/api && go run ./cmd/seed

test-api: ## Go unit + integration tests (race detector, coverage)
	cd apps/api && go test ./... -race -cover

test-dashboard: ## Vitest component/composable tests
	pnpm --filter @internity/dashboard test:unit

test-e2e: ## Playwright critical-path E2E against the composed stack
	pnpm --filter @internity/e2e test

lint: ## Lint every workspace (Go + JS)
	cd apps/api && go vet ./... && gofmt -l .
	pnpm -r --if-present lint
