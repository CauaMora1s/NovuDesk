# NovuDesk — Documentation Index

> **English documentation.** Portuguese documentation is in the project root [`README.md`](../../README.md).

---

## What is NovuDesk?

NovuDesk is an open-source, multi-tenant helpdesk platform built to be both a production-ready commercial product and a high-quality portfolio project. Organizations (tenants) are fully isolated from each other. Each organization has its own users, teams, ticket categories, roles, SLA policies, and billing plan. The platform ships as a self-hostable Docker stack and can also run as a managed SaaS.

Key capabilities: ticket lifecycle management, granular role-based permissions, SLA tracking, public and internal comments, file attachments, real-time updates via SSE, email notifications, audit logs, and a bilingual UI (English / Portuguese).

---

## Tech Stack at a Glance

| Layer | Technology | Version |
|---|---|---|
| Backend API | Go | 1.23+ |
| Database | PostgreSQL | 16 |
| Cache / Pub-Sub | Redis | 7 |
| Frontend | SvelteKit + Svelte 5 | 2.7 / 5.x |
| UI | TailwindCSS + DaisyUI | 3.4 / 4.x |
| Migrations | Goose | latest |
| Container | Docker Compose | v2 |
| HTTP Router | chi | v5 |
| Auth | JWT RS256 + refresh tokens | — |

---

## Repository Layout

```
NovuDesk/
├── apps/
│   ├── api/                   # Go backend
│   │   ├── cmd/server/        # Entry point (main.go — wires everything together)
│   │   ├── config/            # Env-var config loader + RSA key files
│   │   ├── internal/
│   │   │   ├── domain/        # Layer 1 — pure entities + repository interfaces
│   │   │   ├── application/   # Layer 2 — use cases / business logic
│   │   │   ├── infrastructure/# Layer 3 — DB, Redis, storage, email
│   │   │   └── interfaces/    # Layer 4 — HTTP handlers, middleware, SSE
│   │   ├── migrations/        # SQL migration files (goose format)
│   │   ├── seeds/             # Development seed data
│   │   └── pkg/               # Shared utilities (errors, logger, pagination, validator)
│   │
│   └── web/                   # SvelteKit frontend
│       └── src/
│           ├── routes/        # File-based pages
│           │   ├── (auth)/    # Unauthenticated pages (login)
│           │   └── (app)/     # Protected pages (sidebar layout)
│           └── lib/
│               ├── api/       # Typed HTTP client modules
│               ├── components/# Reusable Svelte components
│               ├── i18n/      # Translation JSON files
│               ├── permissions/# can() / canAny() / canAll() helpers
│               └── stores/    # Svelte writable stores (auth, polling)
│
├── infra/
│   └── docker/                # Dockerfiles for dev and production
│
├── docs/
│   └── en/                    # ← You are here
│
├── docker-compose.dev.yml     # Local development stack
├── docker-compose.prod.yml    # Production self-hosting stack
├── Makefile                   # All developer commands
├── README.md                  # Portuguese overview
├── README.en.md               # English overview
├── ARCHITECTURE.md            # System design decisions
├── CONTRIBUTING.md            # Contribution workflow and conventions
└── ROADMAP.md                 # Feature roadmap (v1.0 → v2.0)
```

---

## Recommended Study Path

Read the codebase in this order. Each step builds on the previous one.

### Step 1 — Understand the product
**File:** [`README.md`](../../README.md)
Why: Learn what the system does, what features exist, and how to log in locally. You need this context before reading any code.

### Step 2 — Understand the architecture
**File:** [`ARCHITECTURE.md`](../../ARCHITECTURE.md)
Why: The entire codebase follows Clean Architecture with strict dependency rules. Without this mental model, the folder structure will seem confusing.

### Step 3 — Understand the roadmap
**File:** [`ROADMAP.md`](../../ROADMAP.md)
Why: Know what is already implemented (v1.0) versus what is planned (v1.1+). This prevents wasting time looking for features that do not exist yet.

### Step 4 — Understand contributor conventions
**File:** [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
Why: Learn branch naming, commit format, test format, and coding rules before writing a single line.

### Step 5 — Understand configuration
**File:** [`apps/api/config/config.go`](../../apps/api/config/config.go)
Why: Every environment variable the system uses is defined here. Read this once to know what can be configured and where.

### Step 6 — Read the domain models
**Folder:** [`apps/api/internal/domain/`](../../apps/api/internal/domain/)
Read in this order:
- `organization/organization.go` — tenant model
- `user/user.go` — user + member model + permission overrides
- `role/role.go` — role + permission system
- `ticket/ticket.go` — ticket entity, statuses, filters
- `sla/sla.go` — SLA policy, due-date calculation

Why: Domain is the core. It has zero external dependencies. Everything else depends on these interfaces.

### Step 7 — Read the application services
**Folder:** [`apps/api/internal/application/`](../../apps/api/internal/application/)
Read in this order:
- `auth/service.go` — login, JWT generation, refresh tokens
- `ticket/service.go` — ticket CRUD, SLA auto-apply, audit events
- `role/service.go` — role + permission management
- `sla/service.go` — SLA upsert per category

Why: This layer orchestrates the domain and infrastructure. All business rules live here.

### Step 8 — Read the infrastructure implementations
**Folder:** [`apps/api/internal/infrastructure/postgres/`](../../apps/api/internal/infrastructure/postgres/)
Start with:
- `ticket_repo.go` — complex SQL with dynamic WHERE clauses, full-text search, SLA filters
- `user_repo.go` — membership queries, permission overrides
- `role_repo.go` — permission matrix queries

Why: This is where SQL lives. Reading the queries alongside the domain interfaces shows how data is actually stored.

### Step 9 — Read the router (route map)
**File:** [`apps/api/internal/interfaces/http/router.go`](../../apps/api/internal/interfaces/http/router.go)
Why: One file shows every endpoint in the system, which middleware protects it, and which permission is required. Use this as a map.

### Step 10 — Read the HTTP handlers
**Folder:** [`apps/api/internal/interfaces/http/handlers/`](../../apps/api/internal/interfaces/http/handlers/)
Read in this order: `auth_handler.go` → `ticket_handler.go` → `comment_handler.go` → `member_handler.go` → the rest.
Why: Handlers show exactly what request shape is accepted and what response shape is returned.

### Step 11 — Understand the frontend auth model
**Files:**
- [`apps/web/src/lib/stores/auth.ts`](../../apps/web/src/lib/stores/auth.ts)
- [`apps/web/src/lib/permissions/index.ts`](../../apps/web/src/lib/permissions/index.ts)

Why: All permission-based UI rendering flows from these two files. The `can()` function used everywhere in Svelte templates is defined here.

### Step 12 — Read the frontend pages
**Folder:** [`apps/web/src/routes/(app)/`](../../apps/web/src/routes/(app)/)
Read in this order: `dashboard/` → `tickets/` → `teams/` → `settings/`.
Why: Pages are self-contained. Each one shows the complete data-fetch → render → interaction loop for a feature.

---

## Key Concepts

### Multi-tenancy (row-level isolation)
Every table that stores tenant data has an `org_id` column. All repository queries filter by `org_id`. The `org_id` is always read from the validated JWT claims — never from the request body or query string. This is enforced in every handler.

### JWT RS256 authentication
Access tokens are signed with an RSA private key (15-minute TTL). Refresh tokens are 256-bit random values stored in the database as a SHA-256 hash (30-day TTL). On every refresh, the old token is rotated (deleted and replaced). Tokens are validated in [`middleware/auth.go`](../../apps/api/internal/interfaces/http/middleware/auth.go).

### Permission strings (`resource:action`)
Permissions are strings like `tickets:view`, `users:create`, `organization:manage_settings`. They are loaded from the database at login, embedded in the JWT, and checked both in backend middleware and in the Svelte frontend via `can()`. System roles (owner, admin, agent, viewer) each have a fixed set of permissions. Custom roles can be created per organization.

### Server-Sent Events (SSE) for real-time updates
When a ticket changes, the backend publishes an event to Redis pub/sub. The SSE manager subscribes to that channel and pushes the event to all connected browser clients over `GET /api/v1/events`. The frontend uses polling as a fallback for browsers that reconnect.

### Clean Architecture dependency rule
```
domain  ←  application  ←  infrastructure  ←  interfaces  ←  cmd/server
```
Each layer may only import layers to its left. `domain/` has zero external imports. `application/` imports only `domain/`. Breaking this rule causes architectural rot.

---

## Documentation Pages

| File | Contents |
|---|---|
| [getting-started.md](getting-started.md) | Local setup, credentials, Make commands, environment variables |
| [api-reference.md](api-reference.md) | Every API endpoint — method, path, params, response, permissions |
| [frontend-guide.md](frontend-guide.md) | Every page, component, store, permission helper, i18n |
| [database-guide.md](database-guide.md) | Every table, column, index, migration file |
| [how-to-change.md](how-to-change.md) | Step-by-step recipes for common changes |
