# Database Guide

NovuDesk uses **PostgreSQL 16** with **Goose** for migrations.

**Migration files:** `apps/api/migrations/`
**Repository implementations:** `apps/api/internal/infrastructure/postgres/`

---

## Conventions

Every table in the project follows these rules:

| Convention | Why |
|---|---|
| `id UUID PRIMARY KEY DEFAULT gen_random_uuid()` | UUIDs avoid sequential ID enumeration attacks |
| `org_id UUID NOT NULL` on every tenant table | Row-level multi-tenancy isolation |
| `created_at TIMESTAMPTZ DEFAULT NOW()` | All records track creation time |
| `updated_at TIMESTAMPTZ DEFAULT NOW()` | Updated by triggers or application code |
| Composite index `(org_id, id)` on all tenant tables | Efficient per-org lookups by ID |
| Foreign keys with `ON DELETE CASCADE` | Referential integrity, automatic cleanup |
| `JSONB` for flexible metadata | Custom fields, audit diffs, settings |
| `TEXT[]` arrays for tags | Simple array storage, GIN-indexed |

---

## Migration Files

| # | File | What it creates |
|---|---|---|
| 001 | `001_create_organizations.sql` | `organizations` table |
| 002 | `002_create_users.sql` | `users` table |
| 003 | `003_create_roles_permissions.sql` | `roles`, `role_permissions`, `permissions` tables |
| 004 | `004_create_org_members_teams.sql` | `organization_members`, `teams`, `team_members` tables |
| 005 | `005_create_sla_policies.sql` | `sla_policies` table |
| 006 | `006_create_tickets.sql` | `tickets` table with indexes |
| 007 | `007_create_comments_attachments.sql` | `comments`, `attachments` tables |
| 008 | `008_create_audit_logs.sql` | `audit_logs` table (partitioned by month) |
| 009 | `009_create_auth_tables.sql` | `refresh_tokens` table |
| 010 | `010_create_custom_fields.sql` | `custom_field_definitions` table |
| 011 | `011_create_api_keys_feature_flags.sql` | `api_keys`, `feature_flags` tables |
| 012 | `012_create_automation_webhooks.sql` | `automation_rules`, `webhooks`, `webhook_deliveries` tables |
| 013 | `013_create_categories.sql` | `categories` table |
| 014 | `014_create_member_permission_overrides.sql` | `member_permission_overrides` table |
| 015 | `015_sla_category_link.sql` | `category_id` column added to `sla_policies` |
| 016 | `016_create_org_billing.sql` | `payment_sessions`, billing columns on `organizations` |

All migration files use the Goose format with explicit `Up` and `Down` sections.

---

## Tables

### `organizations`

Stores tenant information. One row per organization.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `name` | TEXT | no | Display name (e.g., "Acme Corp") |
| `slug` | TEXT | no | URL-safe identifier (e.g., "acme") — unique globally |
| `logo_url` | TEXT | yes | URL to logo image |
| `plan_tier` | TEXT | no | `free`, `pro`, `enterprise` |
| `plan_renews_at` | TIMESTAMPTZ | yes | Next billing date |
| `billing_status` | TEXT | yes | `active`, `past_due`, `canceled` |
| `billing_provider` | TEXT | yes | `stripe`, `manual`, etc. |
| `billing_customer_ref` | TEXT | yes | External provider customer ID |
| `payment_method_brand` | TEXT | yes | `visa`, `mastercard`, etc. |
| `payment_method_last4` | TEXT | yes | Last 4 digits of card |
| `settings` | JSONB | yes | Tenant-specific settings (future use) |
| `created_at` | TIMESTAMPTZ | no | — |
| `updated_at` | TIMESTAMPTZ | no | — |

**Indexes:** `UNIQUE(slug)`
**Repository:** `postgres/organization_repo.go`

---

### `users`

Global user accounts. Not org-scoped. A user can belong to multiple organizations via `organization_members`.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `email` | TEXT | no | Unique email address |
| `password_hash` | TEXT | no | bcrypt hash |
| `full_name` | TEXT | no | Display name |
| `avatar_url` | TEXT | yes | Profile picture URL |
| `locale` | TEXT | no | `en` or `pt` (default: `pt`) |
| `is_active` | BOOLEAN | no | Global active flag (default: true) |
| `created_at` | TIMESTAMPTZ | no | — |
| `updated_at` | TIMESTAMPTZ | no | — |

**Indexes:** `UNIQUE(email)`
**Repository:** `postgres/user_repo.go`

---

### `roles`

Role definitions. System roles (`owner`, `admin`, `agent`, `viewer`) are seeded once. Custom roles are created per organization.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | yes | NULL for system roles (global); UUID for custom org roles |
| `name` | TEXT | no | Role name |
| `is_system_role` | BOOLEAN | no | True for owner/admin/agent/viewer |
| `created_at` | TIMESTAMPTZ | no | — |

**Indexes:** `(org_id, name)` unique

---

### `permissions`

Catalog of all available permission strings in the system.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `key` | TEXT | no | Permission string (e.g., `tickets:view`) |
| `description` | TEXT | yes | Human-readable description |
| `created_at` | TIMESTAMPTZ | no | — |

**Indexes:** `UNIQUE(key)`

---

### `role_permissions`

Many-to-many: which permissions are assigned to which role.

| Column | Type | Description |
|---|---|---|
| `role_id` | UUID | → `roles.id` |
| `permission_id` | UUID | → `permissions.id` |

**Repository:** `postgres/role_repo.go`

---

### `organization_members`

Links users to organizations and assigns a role within that org. A user can be a member of multiple orgs with different roles.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `user_id` | UUID | no | → `users.id` |
| `role_id` | UUID | no | → `roles.id` (their role in this org) |
| `is_active` | BOOLEAN | no | False = deactivated (cannot log in to this org) |
| `joined_at` | TIMESTAMPTZ | no | When they joined this org |

**Indexes:** `UNIQUE(org_id, user_id)`, `(org_id, role_id)`
**Repository:** `postgres/user_repo.go`

---

### `member_permission_overrides`

Per-member permission exceptions that override the role's default permissions.

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `org_id` | UUID | → `organizations.id` |
| `member_id` | UUID | → `organization_members.id` |
| `permission_id` | UUID | → `permissions.id` |
| `granted` | BOOLEAN | `true` = explicitly grant, `false` = explicitly deny |
| `created_at` | TIMESTAMPTZ | — |

**Repository:** `postgres/user_repo.go` → `GetMemberPermissionOverrides()`, `SetMemberPermissionOverrides()`

---

### `teams`

Agent groupings within an organization. Teams can be linked to ticket categories.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `name` | TEXT | no | Team name |
| `description` | TEXT | yes | Team description |
| `created_at` | TIMESTAMPTZ | no | — |
| `updated_at` | TIMESTAMPTZ | no | — |

**Repository:** `postgres/team_repo.go`

---

### `team_members`

Many-to-many: which users are in which team.

| Column | Type | Description |
|---|---|---|
| `team_id` | UUID | → `teams.id` |
| `user_id` | UUID | → `users.id` |
| `joined_at` | TIMESTAMPTZ | — |

**Indexes:** `UNIQUE(team_id, user_id)`

---

### `categories`

Ticket classification labels, scoped to an organization.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `name` | TEXT | no | Category name (e.g., "Bug", "Feature Request") |
| `description` | TEXT | yes | Description |
| `created_at` | TIMESTAMPTZ | no | — |
| `updated_at` | TIMESTAMPTZ | no | — |

**Repository:** `postgres/category_repo.go`

---

### `sla_policies`

SLA (Service Level Agreement) policies. One policy per category per organization.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `name` | TEXT | no | Policy name (default: "SLA") |
| `category_id` | UUID | yes | → `categories.id` |
| `response_hours` | INT | yes | First-response SLA in hours |
| `resolution_value` | INT | no | Numeric resolution time |
| `resolution_unit` | TEXT | no | `hours`, `days`, `weeks` |
| `resolution_hours` | INT | yes | Computed: resolution_value × unit factor |
| `conditions` | JSONB | yes | Future: conditional SLA rules |
| `is_active` | BOOLEAN | no | Toggle (default: true) |
| `created_at` | TIMESTAMPTZ | no | — |
| `updated_at` | TIMESTAMPTZ | no | — |

**Indexes:** `(org_id, category_id)` unique
**Repository:** `postgres/sla_repo.go`

**Domain methods** (in `domain/sla/sla.go`):
- `ResolutionHoursComputed()` — converts `resolution_value` + `resolution_unit` to hours
- `CalculateDueDates(openedAt)` — returns `response_due_at` and `resolution_due_at`

---

### `tickets`

The core entity. Contains status, priority, SLA tracking fields, tags, and custom fields.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `number` | INT | no | Sequential per-org ticket number (e.g., #42) |
| `title` | TEXT | no | Ticket title |
| `description` | TEXT | yes | Full description |
| `status` | TEXT | no | `open`, `pending`, `on_hold`, `resolved`, `closed` |
| `priority` | TEXT | no | `low`, `normal`, `high`, `urgent` |
| `assignee_id` | UUID | yes | → `users.id` |
| `requester_id` | UUID | no | → `users.id` (creator) |
| `team_id` | UUID | yes | → `teams.id` |
| `category_id` | UUID | yes | → `categories.id` |
| `sla_policy_id` | UUID | yes | → `sla_policies.id` |
| `sla_response_due_at` | TIMESTAMPTZ | yes | First-response deadline |
| `sla_resolution_due_at` | TIMESTAMPTZ | yes | Resolution deadline |
| `sla_breached` | BOOLEAN | no | True if past resolution deadline (default: false) |
| `tags` | TEXT[] | yes | Array of tag strings |
| `custom_fields` | JSONB | yes | Key-value map for custom field data |
| `resolved_at` | TIMESTAMPTZ | yes | Set when status → resolved |
| `closed_at` | TIMESTAMPTZ | yes | Set when status → closed |
| `created_at` | TIMESTAMPTZ | no | — |
| `updated_at` | TIMESTAMPTZ | no | — |
| `search_vector` | TSVECTOR | yes | Full-text search index (title + description) |

**Indexes:**
- `(org_id, id)`
- `(org_id, number)` unique
- `(org_id, status)`
- `(org_id, assignee_id)`
- `(org_id, team_id)`
- `(org_id, category_id)`
- `(org_id, sla_resolution_due_at)` — for SLA due date sorting
- GIN index on `tags` array — for tag filtering
- GIN index on `custom_fields` JSONB
- GIN index on `search_vector` — for full-text search

**Repository:** `postgres/ticket_repo.go`

---

### `comments`

Stores both public (customer-visible) and internal (agent-only) comments on tickets. The unified timeline mixes comments with audit events.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `ticket_id` | UUID | no | → `tickets.id` |
| `author_id` | UUID | no | → `users.id` |
| `body` | TEXT | no | Comment content |
| `is_internal` | BOOLEAN | no | True = agent-only (default: false) |
| `created_at` | TIMESTAMPTZ | no | — |
| `updated_at` | TIMESTAMPTZ | no | — |

**Indexes:** `(org_id, ticket_id)`, `(ticket_id, created_at)`
**Repository:** `postgres/comment_repo.go`

---

### `attachments`

File attachment metadata. The actual file is stored in S3/MinIO/local disk.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `ticket_id` | UUID | no | → `tickets.id` |
| `comment_id` | UUID | yes | → `comments.id` (optional link) |
| `uploader_id` | UUID | no | → `users.id` |
| `name` | TEXT | no | Original filename |
| `mime_type` | TEXT | no | MIME type (e.g., `image/png`) |
| `size_bytes` | BIGINT | no | File size in bytes |
| `storage_key` | TEXT | no | Key/path in the storage backend |
| `created_at` | TIMESTAMPTZ | no | — |

**Repository:** `postgres/attachment_repo.go`

---

### `audit_logs`

Full change history for the system. Partitioned by month for performance.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `actor_id` | UUID | no | → `users.id` (who made the change) |
| `resource_type` | TEXT | no | `ticket`, `comment`, `member`, `role`, etc. |
| `resource_id` | UUID | no | ID of the changed resource |
| `action` | TEXT | no | `created`, `updated`, `deleted`, `status_changed`, etc. |
| `before` | JSONB | yes | State before the change |
| `after` | JSONB | yes | State after the change |
| `created_at` | TIMESTAMPTZ | no | Timestamp of the action |

**Partitioning:** Monthly range partitions on `created_at`. New partitions must be created before the month begins (or use automatic partition creation if configured).

**Indexes:** `(org_id, resource_id)`, `(org_id, actor_id)`, `(created_at)` (partition key)
**Repository:** `postgres/audit_repo.go`

---

### `refresh_tokens`

Stored refresh tokens (as SHA-256 hashes). Enables token rotation and revocation.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `user_id` | UUID | no | → `users.id` |
| `org_id` | UUID | no | → `organizations.id` |
| `token_hash` | TEXT | no | SHA-256 hash of the raw token |
| `expires_at` | TIMESTAMPTZ | no | 30 days from creation |
| `created_at` | TIMESTAMPTZ | no | — |

**Indexes:** `UNIQUE(token_hash)`, `(user_id, org_id)`

---

### `custom_field_definitions`

Tenant-defined metadata fields that appear on tickets.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `name` | TEXT | no | Field label |
| `field_key` | TEXT | no | Key used in `tickets.custom_fields` JSONB |
| `field_type` | TEXT | no | `text`, `number`, `boolean`, `select`, `multi_select`, `date` |
| `config` | JSONB | yes | Options list for select types |
| `is_required` | BOOLEAN | no | — |
| `created_at` | TIMESTAMPTZ | no | — |

---

### `api_keys`

API keys for the public API. Scoped to specific permissions (v1.2 roadmap).

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | no | → `organizations.id` |
| `name` | TEXT | no | Key description |
| `key_hash` | TEXT | no | SHA-256 hash of the raw key |
| `scopes` | TEXT[] | yes | Allowed permission scopes |
| `last_used_at` | TIMESTAMPTZ | yes | — |
| `expires_at` | TIMESTAMPTZ | yes | Optional expiry |
| `created_at` | TIMESTAMPTZ | no | — |

---

### `feature_flags`

Per-tenant feature toggles for controlled rollouts.

| Column | Type | Nullable | Description |
|---|---|---|---|
| `id` | UUID | no | Primary key |
| `org_id` | UUID | yes | NULL = global flag; UUID = org-specific |
| `flag_key` | TEXT | no | Feature identifier |
| `is_enabled` | BOOLEAN | no | Toggle |
| `updated_at` | TIMESTAMPTZ | no | — |

---

### `automation_rules`

Event-driven rules that trigger actions when conditions are met (v1.1 roadmap).

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `org_id` | UUID | → `organizations.id` |
| `name` | TEXT | Rule name |
| `event_trigger` | TEXT | `ticket.created`, `ticket.updated`, `comment.created`, etc. |
| `conditions` | JSONB | Condition tree (field, operator, value) |
| `actions` | JSONB | Action list (assign, change_status, notify, webhook, etc.) |
| `is_active` | BOOLEAN | Toggle |
| `created_at` | TIMESTAMPTZ | — |

---

### `webhooks`

Outbound webhook subscriptions for external integrations (v1.1 roadmap).

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `org_id` | UUID | → `organizations.id` |
| `url` | TEXT | Target URL |
| `secret` | TEXT | HMAC signing secret |
| `events` | TEXT[] | Subscribed event types |
| `is_active` | BOOLEAN | Toggle |
| `created_at` | TIMESTAMPTZ | — |

---

### `webhook_deliveries`

Delivery log for each webhook attempt. Tracks retries and failures.

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `webhook_id` | UUID | → `webhooks.id` |
| `event_type` | TEXT | Event that triggered delivery |
| `payload` | JSONB | Request body sent |
| `response_status` | INT | HTTP response status code |
| `response_body` | TEXT | Response body (truncated) |
| `attempt_count` | INT | Number of delivery attempts |
| `succeeded` | BOOLEAN | Whether a 2xx was received |
| `created_at` | TIMESTAMPTZ | — |

---

### `payment_sessions`

Tracks pending and completed billing plan change sessions.

| Column | Type | Description |
|---|---|---|
| `id` | UUID | Primary key |
| `org_id` | UUID | → `organizations.id` |
| `requested_by` | UUID | → `users.id` |
| `from_plan` | TEXT | Current plan tier |
| `to_plan` | TEXT | Requested plan tier |
| `status` | TEXT | `pending`, `confirmed`, `cancelled` |
| `provider_session_id` | TEXT | External billing provider session ID |
| `created_at` | TIMESTAMPTZ | — |
| `updated_at` | TIMESTAMPTZ | — |

---

### `organization_usage`

Snapshot of resource usage per organization for quota enforcement.

| Column | Type | Description |
|---|---|---|
| `org_id` | UUID | → `organizations.id` (primary key) |
| `members` | INT | Current member count |
| `tickets` | INT | Current open ticket count |
| `storage_bytes` | BIGINT | Total storage used |
| `teams` | INT | Current team count |
| `categories` | INT | Current category count |
| `api_keys` | INT | Current API key count |
| `updated_at` | TIMESTAMPTZ | Last snapshot time |

---

## Adding a Migration

Migrations use the **Goose** format with explicit `Up` and `Down` sections.

### Filename convention

```
{number}_{description}.sql
```

Examples:
- `017_add_ticket_due_date.sql`
- `018_create_notification_preferences.sql`

The number must be greater than the highest existing migration number. Check with `make migrate-status`.

### File format

```sql
-- +goose Up

ALTER TABLE tickets ADD COLUMN due_date TIMESTAMPTZ;
CREATE INDEX idx_tickets_org_due_date ON tickets (org_id, due_date);

-- +goose Down

DROP INDEX IF EXISTS idx_tickets_org_due_date;
ALTER TABLE tickets DROP COLUMN IF EXISTS due_date;
```

**Rules:**
- Every `Up` must have a working `Down` — tested rollbacks are required by CONTRIBUTING.md
- `Down` must completely reverse the `Up` (no orphaned columns or indexes)
- Use `IF EXISTS` / `IF NOT EXISTS` for safe rollbacks
- Never modify existing migration files — only add new ones

### Running migrations

```bash
make migrate          # Apply all pending
make migrate-down     # Roll back last migration
make migrate-status   # Show applied/pending
```
