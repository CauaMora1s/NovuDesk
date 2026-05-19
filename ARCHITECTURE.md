# Architecture

This document describes the technical decisions behind NovuDesk and the reasoning for each choice.

---

## Overview

NovuDesk is a monorepo containing a Go backend and a SvelteKit frontend, packaged with Docker Compose for local development and production self-hosting.

```
┌─────────────────────────────────────────────────────────────┐
│  Browser (SvelteKit SPA)                                    │
│  - Permission-based rendering (no CSS hide)                 │
│  - SSE connection for realtime updates                      │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP / SSE
┌──────────────────────▼──────────────────────────────────────┐
│  Go API (chi router)                                        │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐ │
│  │  Middleware │  │   Handlers   │  │    SSE Manager     │ │
│  │  auth/perm  │  │  (per domain)│  │  (Redis pub/sub)   │ │
│  └─────────────┘  └──────┬───────┘  └────────────────────┘ │
│                          │                                  │
│  ┌───────────────────────▼──────────────────────────────┐  │
│  │  Application layer (use cases)                        │  │
│  │  auth / organization / user / ticket / comment / sla  │  │
│  └───────────────────────┬──────────────────────────────┘  │
│                          │                                  │
│  ┌───────────────────────▼──────────────────────────────┐  │
│  │  Infrastructure                                       │  │
│  │  postgres repos │ redis client │ smtp │ s3 storage    │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
         │ SQL              │ Redis        │ SMTP / S3
    ┌────▼────┐       ┌─────▼──────┐  ┌───▼────────────┐
    │Postgres │       │   Redis    │  │ SMTP / Storage │
    └─────────┘       └────────────┘  └────────────────┘
```

---

## Backend — Clean Architecture

The backend follows a strict dependency rule: inner layers never import outer layers.

```
domain/       ← no external imports, pure Go structs and interfaces
application/  ← imports domain only, orchestrates use cases
infrastructure/ ← implements domain interfaces (postgres, redis, smtp, s3)
interfaces/   ← HTTP handlers, middleware, SSE — imports application only
cmd/server/   ← dependency injection wiring, no business logic
```

**Why Clean Architecture?**
Contributors can understand and extend any layer in isolation. Swapping the database, email provider, or storage backend requires only a new implementation of the relevant interface — no changes to business logic.

---

## Multi-tenancy

Row-level isolation: every tenant-scoped table has an `org_id` column. The `org_id` is extracted from the JWT on every request and injected into the repository context. It is never trusted from the request body.

**Future path:** because repositories are behind interfaces, migrating high-volume tenants to schema-per-tenant or dedicated databases requires only a new repository implementation and a connection pool router.

---

## Authentication

- **Access token:** JWT RS256, 15-minute TTL. Contains `user_id`, `org_id`, `role`, and `permissions[]`.
- **Refresh token:** 256-bit random token, stored as SHA-256 hash in the database, delivered as an HTTP-only cookie.
- Token rotation on every refresh. Revocation maintained in the `refresh_tokens` table.
- **Future SSO:** an `AuthProvider` interface wraps local login. OAuth/OIDC/SAML are new implementations of the same interface.

---

## Permission System

Permissions are loaded from the database at login and embedded in the JWT. The frontend reads them from the decoded JWT and makes all rendering decisions at runtime — no element is hidden with CSS, it is conditionally rendered.

The permission key format is `resource:action`, for example `tickets:create`, `comments:create_internal`.

Custom roles can be created per organization with any subset of the defined permission keys.

---

## Realtime (SSE)

```
Ticket updated
  → application layer emits Event
  → EventBus.Publish → Redis PUBLISH sse:org:{org_id}
  → SSE Manager (all API instances) receives message
  → Each connected client for that org receives the event
```

SSE (Server-Sent Events) was chosen over WebSockets because:
- Works over HTTP/1.1, no special proxy configuration
- Unidirectional push is sufficient for notifications and live ticket updates
- Simpler horizontal scaling via Redis pub/sub

---

## Async Queues (Redis Streams)

Workers for email, webhooks, and automations consume from dedicated Redis streams:

| Stream | Consumer | Purpose |
|---|---|---|
| `queue:emails` | EmailWorker | Async SMTP delivery |
| `queue:webhooks` | WebhookWorker | HTTP delivery with retry |
| `queue:automations` | AutomationWorker | Rule evaluation and actions |

Workers run as goroutines inside the same binary for MVP. They can be extracted to separate processes without changing the queue interface.

---

## Storage

A `storage.Provider` interface abstracts file storage:

- `LocalProvider` — writes to disk, serves via `/files/` endpoint. Used in development.
- `S3Provider` — AWS S3, Cloudflare R2, MinIO, DigitalOcean Spaces. Selected via `STORAGE_DRIVER=s3`.

MinIO is included in `docker-compose.dev.yml` so the dev environment is S3-compatible.

---

## Frontend Architecture

SvelteKit in SPA mode (`adapter-static`, SSR disabled). All pages are client-rendered after initial load.

- **Stores** — Svelte writable stores for auth state, permissions, and UI state
- **API client** — typed fetch wrapper with automatic 401 redirect
- **Permissions** — `can(permission)` helper reads from `authStore`, used in templates for conditional rendering
- **i18n** — `svelte-i18n` with JSON files per locale. Adding a new language requires only a new JSON file.
- **Design** — DaisyUI customized via Tailwind config tokens. Light and dark themes via `data-theme` attribute.

---

## Database

PostgreSQL 16 with goose for migrations.

Key conventions:
- Every tenant-scoped table: `id UUID` + `org_id UUID` as first two columns
- Composite index `(org_id, id)` on all tenant-scoped tables
- `JSONB` for custom fields with GIN index for query support
- `text[]` for tags with GIN index
- Full-text search index on tickets using `tsvector`
- Audit log partitioned by `created_at` (monthly) for long-term performance

---

## CI/CD

GitHub Actions runs on every pull request to `main`:

1. Frontend: type check → lint → build
2. Backend: lint (golangci-lint) → tests → build
3. Docker: build validation for both images
4. Migrations: up → down → up (idempotency check)

Releases are tagged on `main` with semantic versioning (`v1.0.0`). Docker images are published to GitHub Container Registry.
