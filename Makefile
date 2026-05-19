.PHONY: dev dev-api dev-web stop build migrate migrate-down seed test lint fmt help keys

# ─── Colors ───────────────────────────────────────────────────
CYAN  := \033[0;36m
RESET := \033[0m

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

# ─── Development ──────────────────────────────────────────────
dev: ## Start all services (database, redis, api, web, mailhog, minio)
	docker compose -f docker-compose.dev.yml up --build

dev-build: ## Build and start all services
	docker compose -f docker-compose.dev.yml up --build --force-recreate

stop: ## Stop all services
	docker compose -f docker-compose.dev.yml down

logs: ## Tail logs from all services
	docker compose -f docker-compose.dev.yml logs -f

logs-api: ## Tail API logs
	docker compose -f docker-compose.dev.yml logs -f api

# ─── Database ─────────────────────────────────────────────────
migrate: ## Run pending migrations
	docker compose -f docker-compose.dev.yml exec api sh -c 'goose -dir ./migrations postgres "host=$$DB_HOST port=$$DB_PORT dbname=$$DB_NAME user=$$DB_USER password=$$DB_PASSWORD sslmode=$$DB_SSL_MODE" up'

migrate-down: ## Roll back last migration
	docker compose -f docker-compose.dev.yml exec api sh -c 'goose -dir ./migrations postgres "host=$$DB_HOST port=$$DB_PORT dbname=$$DB_NAME user=$$DB_USER password=$$DB_PASSWORD sslmode=$$DB_SSL_MODE" down'

migrate-status: ## Show migration status
	docker compose -f docker-compose.dev.yml exec api sh -c 'goose -dir ./migrations postgres "host=$$DB_HOST port=$$DB_PORT dbname=$$DB_NAME user=$$DB_USER password=$$DB_PASSWORD sslmode=$$DB_SSL_MODE" status'

seed: ## Seed development data
	docker compose -f docker-compose.dev.yml exec api go run ./seeds/...

# ─── Testing ──────────────────────────────────────────────────
test: test-api test-web ## Run all tests

test-api: ## Run backend tests
	cd apps/api && go test ./... -race -timeout 120s

test-api-coverage: ## Run backend tests with coverage
	cd apps/api && go test ./... -race -coverprofile=coverage.out && go tool cover -html=coverage.out

test-web: ## Run frontend tests
	cd apps/web && pnpm test

# ─── Linting ──────────────────────────────────────────────────
lint: lint-api lint-web ## Lint all code

lint-api: ## Lint backend
	cd apps/api && golangci-lint run ./...

lint-web: ## Lint frontend
	cd apps/web && pnpm lint

fmt: ## Format all code
	cd apps/api && gofmt -w .
	cd apps/web && pnpm format

# ─── Build ────────────────────────────────────────────────────
build: ## Build production Docker images
	docker compose -f docker-compose.prod.yml build

build-api: ## Build API Docker image only
	docker build -f infra/docker/api.Dockerfile -t novudesk-api:latest .

build-web: ## Build web Docker image only
	docker build -f infra/docker/web.Dockerfile -t novudesk-web:latest .

# ─── Keys ─────────────────────────────────────────────────────
keys: ## Generate RSA key pair for JWT signing
	@mkdir -p apps/api/config/keys
	@openssl genrsa -out apps/api/config/keys/private.pem 2048
	@openssl rsa -in apps/api/config/keys/private.pem -pubout -out apps/api/config/keys/public.pem
	@echo "RSA key pair generated at apps/api/config/keys/"

# ─── Setup ────────────────────────────────────────────────────
setup: ## First-time project setup (copy .env, generate keys)
	@cp -n .env.example .env || true
	@$(MAKE) keys
	@echo "Setup complete. Edit .env and run 'make dev' to start."
