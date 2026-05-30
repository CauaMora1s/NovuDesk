# Getting Started

This guide walks you from a fresh clone to a fully running local development environment.

---

## Prerequisites

| Tool | Minimum version | Purpose |
|---|---|---|
| Git | 2.x | Clone the repository |
| Docker Desktop | 4.x | Run all services (Postgres, Redis, MailHog, MinIO, API, Web) |
| Make | 3.x | Run project commands (macOS/Linux built-in; Windows: install via Chocolatey or WSL) |
| Node.js | 20+ | Only needed if running the frontend outside Docker |
| Go | 1.23+ | Only needed if running the API outside Docker |

> On Windows, all `make` commands work inside WSL2 or Git Bash. Alternatively, open `Makefile` and run the Docker Compose commands directly.

---

## Clone and First-time Setup

```bash
git clone https://github.com/CauaMora1s/NovuDesk.git
cd NovuDesk

# Copies .env.example → .env and generates RSA key pair for JWT signing
make setup
```

`make setup` does two things:
1. Copies `.env.example` to `.env` (safe defaults for local development)
2. Generates `apps/api/config/keys/private.pem` and `public.pem` (RSA-2048 key pair used for JWT RS256 signing)

> The RSA keys are gitignored. If you delete them, run `make setup` again or use `make keys`.

---

## Start All Services

```bash
make dev
```

This runs `docker compose -f docker-compose.dev.yml up` which starts six containers:

| Container | What it runs | Notes |
|---|---|---|
| `postgres` | PostgreSQL 16 | Data persisted in a named Docker volume |
| `redis` | Redis 7 | Used for SSE pub/sub and async job queues |
| `mailhog` | MailHog SMTP capture | Catches all outgoing emails — no real emails sent |
| `minio` | MinIO S3-compatible object storage | Used for file attachments in dev |
| `api` | Go API with hot-reload (Air) | Rebuilds automatically on `.go` file changes |
| `web` | SvelteKit Vite dev server | HMR active — instant browser updates |

---

## Service URLs

| Service | URL | Credentials |
|---|---|---|
| Frontend | http://localhost:5173 | — |
| API (health check) | http://localhost:8080/api/v1/health | — |
| MailHog (email UI) | http://localhost:8025 | — |
| MinIO console | http://localhost:9001 | `minioadmin` / `minioadmin` |
| PostgreSQL | localhost:5432 | user: `novudesk`, pass: `novudesk`, db: `novudesk` |

---

## Default Development Credentials

After running migrations and seeds (see below), you can log in with:

| Field | Value |
|---|---|
| Organization slug | `acme` |
| Owner email | `admin@acme.com` |
| Owner password | `password123` |
| Agent email | `agent@acme.com` |
| Agent password | `password123` |

---

## Run Migrations

```bash
make migrate
```

Runs all pending SQL migrations in order using Goose. Migration files are at `apps/api/migrations/`. Safe to run multiple times — Goose skips already-applied migrations.

```bash
make migrate-down    # Roll back the last migration
make migrate-status  # Show which migrations are applied
```

---

## Seed Development Data

```bash
make seed
```

Inserts sample data: the `acme` organization, default roles (owner, admin, agent, viewer), and the two dev users. Also inserts sample ticket categories and SLA policies.

---

## Useful Make Commands

| Command | What it does |
|---|---|
| `make dev` | Start all containers (foreground, with logs) |
| `make dev-build` | Force-recreate containers (use after adding a dependency) |
| `make stop` | Stop all containers |
| `make logs` | Tail logs from all containers |
| `make logs-api` | Tail API container logs only |
| `make migrate` | Apply pending migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-status` | Show migration status |
| `make seed` | Seed development data |
| `make test` | Run all tests (API + web) |
| `make test-api` | Run Go unit tests |
| `make test-api-coverage` | Run Go tests with coverage report |
| `make test-web` | Run frontend Vitest tests |
| `make lint` | Lint all code (Go + TypeScript) |
| `make lint-api` | Run golangci-lint on the API |
| `make lint-web` | Run ESLint + Prettier check on the frontend |
| `make fmt` | Auto-format all code |
| `make build` | Build production Docker images |
| `make build-api` | Build production API image only |
| `make build-web` | Build production frontend image only |
| `make setup` | First-time setup (copy .env, generate keys) |
| `make keys` | Re-generate RSA key pair |

---

## Environment Variables Reference

All variables live in `.env` (copied from `.env.example`). The Go config loader is at `apps/api/config/config.go`.

### Application

| Variable | Type | Default | Description |
|---|---|---|---|
| `APP_ENV` | string | `development` | Set to `production` in prod — changes log format and CORS behavior |
| `APP_PORT` | int | `8080` | Port the Go API listens on |
| `APP_SECRET_KEY` | string | — | General secret key (currently reserved for future HMAC use) |

### Database

| Variable | Type | Default | Description |
|---|---|---|---|
| `DB_HOST` | string | `postgres` | PostgreSQL host (Docker service name in dev) |
| `DB_PORT` | int | `5432` | PostgreSQL port |
| `DB_NAME` | string | `novudesk` | Database name |
| `DB_USER` | string | `novudesk` | Database user |
| `DB_PASSWORD` | string | `novudesk` | Database password |
| `DB_SSLMODE` | string | `disable` | Set to `require` in production |
| `DB_MAX_OPEN_CONNS` | int | `25` | Max open connections in pool |
| `DB_MAX_IDLE_CONNS` | int | `10` | Max idle connections in pool |

### Redis

| Variable | Type | Default | Description |
|---|---|---|---|
| `REDIS_ADDR` | string | `redis:6379` | Redis address |
| `REDIS_PASSWORD` | string | (empty) | Redis password (set in production) |
| `REDIS_DB` | int | `0` | Redis database index |

### JWT

| Variable | Type | Default | Description |
|---|---|---|---|
| `JWT_PRIVATE_KEY_PATH` | string | `./config/keys/private.pem` | Path to RSA private key (for signing access tokens) |
| `JWT_PUBLIC_KEY_PATH` | string | `./config/keys/public.pem` | Path to RSA public key (for validating access tokens) |
| `JWT_ACCESS_TOKEN_TTL` | duration | `15m` | Access token time-to-live |
| `JWT_REFRESH_TOKEN_TTL` | duration | `720h` | Refresh token time-to-live (30 days) |

### Email (SMTP)

| Variable | Type | Default | Description |
|---|---|---|---|
| `SMTP_HOST` | string | `mailhog` | SMTP server host |
| `SMTP_PORT` | int | `1025` | SMTP port (MailHog dev port) |
| `SMTP_USER` | string | (empty) | SMTP username (not needed for MailHog) |
| `SMTP_PASSWORD` | string | (empty) | SMTP password |
| `SMTP_FROM` | string | `noreply@novudesk.io` | From address for all emails |

### Storage

| Variable | Type | Default | Description |
|---|---|---|---|
| `STORAGE_DRIVER` | string | `local` | `local` for dev, `s3` for production |
| `S3_ENDPOINT` | string | `http://minio:9000` | S3-compatible endpoint URL |
| `S3_BUCKET` | string | `novudesk` | Bucket name |
| `S3_ACCESS_KEY_ID` | string | `minioadmin` | Access key |
| `S3_SECRET_ACCESS_KEY` | string | `minioadmin` | Secret key |
| `S3_REGION` | string | `us-east-1` | Region (can be any value for MinIO/R2) |

### CORS

| Variable | Type | Default | Description |
|---|---|---|---|
| `CORS_ALLOWED_ORIGINS` | string | `http://localhost:5173` | Comma-separated list of allowed origins |

### Frontend

| Variable | Type | Default | Description |
|---|---|---|---|
| `VITE_API_URL` | string | (empty = use proxy) | API base URL — leave empty in dev (Vite proxies `/api` to port 8080) |

---

## Adding a New Language

1. Copy `apps/web/src/lib/i18n/en.json` to `apps/web/src/lib/i18n/<lang>.json` (e.g., `es.json`)
2. Translate every value in the JSON (keep keys unchanged)
3. Open `apps/web/src/lib/i18n/index.ts` and register the new locale:
   ```typescript
   register('es', () => import('./es.json'));
   ```
4. Open a Pull Request — new languages are welcome community contributions

---

## Production Deployment

For self-hosted production deployment, use `docker-compose.prod.yml`:

```bash
# Set environment variables (DO NOT use .env from dev)
export DB_PASSWORD=<strong_password>
export REDIS_PASSWORD=<strong_password>
# ... set all required production vars

docker compose -f docker-compose.prod.yml up -d
```

Production differences from dev:
- No MailHog (connect a real SMTP provider)
- No MinIO (use Cloudflare R2, AWS S3, or DigitalOcean Spaces)
- Redis has password authentication
- Nginx serves the built frontend and proxies `/api/` to the Go container
- No hot-reload — pre-built Docker images from GHCR

See [ARCHITECTURE.md](../../ARCHITECTURE.md) for full infrastructure decisions.
