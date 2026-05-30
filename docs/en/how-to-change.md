# How to Change Things

This guide gives step-by-step recipes for common development tasks. Each recipe follows the Clean Architecture dependency rule:

```
domain  ←  application  ←  infrastructure  ←  interfaces  ←  cmd/server
```

**Golden rule:** changes flow from left (domain) to right (interfaces). Never import a right-side layer in a left-side layer.

**Second rule:** All database queries must filter by `org_id`. Never retrieve or modify another tenant's data.

**Third rule:** Never trust `org_id` from the request body. Always read it from JWT claims:
```go
claims := middleware.ClaimsFromContext(r.Context())
orgID := claims.OrgID
```

---

## Common Pitfalls

| Mistake | Correct approach |
|---|---|
| Using `fmt.Errorf("not found")` for user-facing errors | Use `apperrors.NotFound(apperrors.CodeTicketNotFound)` from `pkg/errors` |
| Reading `org_id` from request body or query string | Always use `claims.OrgID` from middleware context |
| Importing `infrastructure/` inside `domain/` | Domain has zero external deps — add the method to the interface, implement in infrastructure |
| Hardcoded user-visible strings in Svelte | Always use `$t('key')` from svelte-i18n |
| Hiding UI elements with CSS based on permissions | Use `{#if can('permission')}` to remove from DOM |
| Modifying existing migration files | Add a new migration file — never edit applied migrations |

---

## Recipe 1: Add a New Field to Tickets

**Example:** adding a `due_date` (customer-requested deadline) to tickets.

### Step 1: Database migration

Create `apps/api/migrations/017_add_ticket_due_date.sql`:

```sql
-- +goose Up
ALTER TABLE tickets ADD COLUMN due_date TIMESTAMPTZ;
CREATE INDEX idx_tickets_org_due_date ON tickets (org_id, due_date);

-- +goose Down
DROP INDEX IF EXISTS idx_tickets_org_due_date;
ALTER TABLE tickets DROP COLUMN IF EXISTS due_date;
```

Run: `make migrate`

### Step 2: Domain entity

In `apps/api/internal/domain/ticket/ticket.go`:

```go
type Ticket struct {
    // ... existing fields
    DueDate *time.Time `json:"due_date"`
}

type CreateInput struct {
    // ... existing fields
    DueDate *time.Time `json:"due_date"`
}

type UpdateInput struct {
    // ... existing fields
    DueDate *time.Time `json:"due_date"`
}
```

### Step 3: Infrastructure — repository

In `apps/api/internal/infrastructure/postgres/ticket_repo.go`:

Add `due_date` to the `SELECT` in `FindByID()` and `List()`:
```sql
SELECT ..., t.due_date, ...
```

Add to the `INSERT` in `Create()`:
```sql
INSERT INTO tickets (..., due_date) VALUES (..., @due_date)
```

Add to the `UPDATE` in `Update()`:
```sql
due_date = COALESCE(@due_date, due_date)
```

### Step 4: HTTP handler

In `apps/api/internal/interfaces/http/handlers/ticket_handler.go`:

```go
type createTicketRequest struct {
    // ... existing fields
    DueDate *time.Time `json:"due_date"`
}

type updateTicketRequest struct {
    // ... existing fields
    DueDate *time.Time `json:"due_date"`
}
```

Include `DueDate` when building the `CreateInput` / `UpdateInput` in the handler methods.

### Step 5: Frontend type

In `apps/web/src/lib/api/tickets.ts`:

```typescript
export type Ticket = {
  // ... existing fields
  due_date: string | null; // ISO 8601 timestamp
};

export type CreateTicketInput = {
  // ... existing fields
  due_date?: string;
};
```

### Step 6: Frontend UI

In `apps/web/src/routes/(app)/tickets/new/+page.svelte`:
- Add a date input field for `due_date`
- Include in the `POST /tickets` body

In `apps/web/src/routes/(app)/tickets/[id]/+page.svelte`:
- Display the due date in the sidebar
- Add an inline date picker that calls `PATCH /tickets/{id}`

### Step 7: i18n

In `apps/web/src/lib/i18n/en.json`:
```json
{ "tickets": { "due_date": "Due Date" } }
```
In `pt.json`:
```json
{ "tickets": { "due_date": "Data de Vencimento" } }
```

---

## Recipe 2: Add a New API Endpoint

**Example:** `POST /api/v1/tickets/{id}/reopen` — re-opens a resolved/closed ticket.

### Step 1: Application service

In `apps/api/internal/application/ticket/service.go`:

```go
func (s *Service) Reopen(ctx context.Context, orgID, ticketID uuid.UUID, actorID uuid.UUID) error {
    ticket, err := s.repo.FindByID(ctx, orgID, ticketID)
    if err != nil {
        return err
    }
    if ticket.Status != "resolved" && ticket.Status != "closed" {
        return apperrors.BadRequest(apperrors.CodeValidationError, "only resolved or closed tickets can be reopened")
    }
    now := time.Now()
    return s.repo.Update(ctx, orgID, ticket.Domain.UpdateInput{
        Status:     pointer("open"),
        ResolvedAt: clearTime(), // set to null
    })
}
```

### Step 2: HTTP handler

In `apps/api/internal/interfaces/http/handlers/ticket_handler.go`:

```go
func (h *TicketHandler) Reopen(w http.ResponseWriter, r *http.Request) {
    claims := middleware.ClaimsFromContext(r.Context())
    ticketID := uuid.MustParse(chi.URLParam(r, "id"))

    if err := h.service.Reopen(r.Context(), claims.OrgID, ticketID, claims.UserID); err != nil {
        respond.Error(w, err)
        return
    }
    respond.NoContent(w)
}
```

### Step 3: Route registration

In `apps/api/internal/interfaces/http/router.go`:

```go
r.With(middleware.RequirePermission("tickets:edit")).
    Post("/tickets/{id}/reopen", ticketHandler.Reopen)
```

### Step 4: Frontend API client

In `apps/web/src/lib/api/tickets.ts`:

```typescript
export const ticketsApi = {
  // ... existing methods
  reopen: (id: string) =>
    api.post<void>(`/tickets/${id}/reopen`),
};
```

### Step 5: Frontend UI

In `apps/web/src/routes/(app)/tickets/[id]/+page.svelte`:

```svelte
{#if ticket.status === 'resolved' || ticket.status === 'closed'}
  {#if can('tickets:edit')}
    <button on:click={handleReopen}>
      {$t('tickets.reopen')}
    </button>
  {/if}
{/if}
```

```typescript
async function handleReopen() {
  await ticketsApi.reopen(ticket.id);
  ticket = await ticketsApi.get(ticket.id);
}
```

---

## Recipe 3: Add a New Permission

**Example:** `reports:export` — allows exporting ticket data.

### Step 1: Backend — register the permission key

In the seed file or a new migration, insert into the `permissions` table:

```sql
INSERT INTO permissions (id, key, description)
VALUES (gen_random_uuid(), 'reports:export', 'Export ticket reports');
```

Assign to the appropriate system roles in the seed:
```sql
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('owner', 'admin') AND p.key = 'reports:export';
```

### Step 2: Backend — use in middleware

In the handler, gate the endpoint:
```go
r.With(middleware.RequirePermission("reports:export")).
    Get("/reports/export", reportHandler.Export)
```

### Step 3: Frontend — add to the Permission type

In `apps/web/src/lib/permissions/index.ts`:

```typescript
export type Permission =
  // ... existing permissions
  | 'reports:export';
```

### Step 4: Frontend — use in templates

```svelte
{#if can('reports:export')}
  <button on:click={handleExport}>{$t('reports.export')}</button>
{/if}
```

The permission will also automatically appear in the `PermissionMatrix` component for role and member management, since it is populated from `GET /permissions`.

---

## Recipe 4: Add a New Svelte Page

**Example:** `/reports` — a basic reporting page.

### Step 1: Create the route file

Create `apps/web/src/routes/(app)/reports/+page.svelte`:

```svelte
<script lang="ts">
  import { t } from 'svelte-i18n';
  import { can } from '$lib/permissions';
  import { onMount } from 'svelte';

  let data = [];

  onMount(async () => {
    // fetch report data
  });
</script>

<div class="p-6">
  <h1 class="text-2xl font-bold">{$t('reports.title')}</h1>
  <!-- page content -->
</div>
```

### Step 2: Add a load function (if needed)

Create `apps/web/src/routes/(app)/reports/+page.ts`:

```typescript
import { redirect } from '@sveltejs/kit';
import { get } from 'svelte/store';
import { authStore } from '$lib/stores/auth';

export const load = async () => {
  const auth = get(authStore);
  if (!auth) throw redirect(302, '/login');
  // fetch server-side data if needed
};
```

### Step 3: Add to sidebar navigation

In `apps/web/src/lib/components/layout/Sidebar.svelte`:

```svelte
{#if can('reports:view')}
  <li>
    <a href="/reports" class={$page.url.pathname.startsWith('/reports') ? 'active' : ''}>
      <!-- icon SVG -->
      {$t('nav.reports')}
    </a>
  </li>
{/if}
```

### Step 4: Add i18n keys

`en.json`: `{ "nav": { "reports": "Reports" }, "reports": { "title": "Reports" } }`
`pt.json`: `{ "nav": { "reports": "Relatórios" }, "reports": { "title": "Relatórios" } }`

---

## Recipe 5: Add a New Database Migration

### Step 1: Find the next migration number

```bash
make migrate-status
# or: ls apps/api/migrations/ | tail -5
```

### Step 2: Create the file

`apps/api/migrations/017_add_notification_preferences.sql`

```sql
-- +goose Up

CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_on_assign BOOLEAN NOT NULL DEFAULT true,
    email_on_comment BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, user_id)
);

CREATE INDEX idx_notif_prefs_org ON notification_preferences (org_id);

-- +goose Down

DROP TABLE IF EXISTS notification_preferences;
```

### Step 3: Run it

```bash
make migrate
```

### Step 4: Test rollback

```bash
make migrate-down
make migrate        # re-apply to verify idempotency
```

---

## Recipe 6: Add a New Language

### Step 1: Copy the English translation file

```bash
cp apps/web/src/lib/i18n/en.json apps/web/src/lib/i18n/es.json
```

### Step 2: Translate all values

Open `es.json` and translate every JSON value (do not change any keys).

### Step 3: Register the locale

In `apps/web/src/lib/i18n/index.ts`:

```typescript
import { register, init, getLocaleFromNavigator } from 'svelte-i18n';

register('en', () => import('./en.json'));
register('pt', () => import('./pt.json'));
register('es', () => import('./es.json'));  // ← add this

init({
  fallbackLocale: 'pt',
  initialLocale: getLocaleFromNavigator(),
});
```

### Step 4: Open a Pull Request

New languages are welcome community contributions. See [CONTRIBUTING.md](../../CONTRIBUTING.md) for the PR process.

---

## Recipe 7: Change SLA Behavior

The SLA system spans four files:

| File | What it controls |
|---|---|
| `domain/sla/sla.go` | `ResolutionHoursComputed()`, `CalculateDueDates()` — the math |
| `application/sla/service.go` | Business rules: default values, upsert logic |
| `infrastructure/postgres/sla_repo.go` | Database queries |
| `application/ticket/service.go` | SLA auto-apply on ticket create/category change |

**To change the resolution unit options** (e.g., add `months`):
1. Update `ResolutionHoursComputed()` in `domain/sla/sla.go` to handle the new unit
2. Update the `resolution_unit` validation in `sla_handler.go`
3. Update the resolution unit select options in `routes/(app)/settings/+page.svelte`

**To change when SLA is applied to tickets:**
- Logic is in `application/ticket/service.go` → `Create()` and `Update()`
- The service looks up the SLA policy by `category_id` and calls `sla.CalculateDueDates()`

**To change SLA breach detection:**
- The `UpdateSLABreach()` method in `ticket_repo.go` marks tickets where `sla_resolution_due_at < NOW()` and `status NOT IN ('resolved', 'closed')`
- This is called by a worker or cron (see `infrastructure/worker/`)

---

## Recipe 8: Change Auth Behavior

The auth system spans these files:

| File | What it controls |
|---|---|
| `application/auth/service.go` | Login, JWT generation, refresh token creation |
| `config/config.go` | TTL values from environment variables |
| `interfaces/http/middleware/auth.go` | JWT validation and claims extraction |
| `apps/web/src/lib/stores/auth.ts` | JWT decode and session storage on the frontend |

**To add a new field to the JWT claims:**
1. Add the field to the claims struct in `application/auth/service.go`
2. Set the value during token generation in `Login()`
3. Add the field to the `Claims` struct in `middleware/auth.go`
4. Decode it in `decodeJwt()` in `stores/auth.ts`
5. Add to `authStore` shape and use in components

**To change token TTLs:**
- Set `JWT_ACCESS_TOKEN_TTL` and `JWT_REFRESH_TOKEN_TTL` in `.env`
- These are parsed in `config/config.go` → used in `auth/service.go`

---

## Recipe 9: Add a New Storage Provider

The storage system uses the Provider interface in `domain/storage/storage.go`.

### Step 1: Implement the interface

Create `apps/api/internal/infrastructure/storage/myprovider.go`:

```go
package storage

type MyProvider struct {
    // configuration fields
}

func (p *MyProvider) Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
    // implement upload logic
    // return public URL
}

func (p *MyProvider) Delete(ctx context.Context, key string) error {
    // implement deletion
}

func (p *MyProvider) URL(key string) string {
    // return the public URL for a key
}
```

### Step 2: Register in the DI container

In `apps/api/cmd/server/main.go`, find the storage initialization switch:

```go
var storageProvider storage.Provider
switch cfg.Storage.Driver {
case "s3":
    storageProvider = storage.NewS3Provider(cfg.Storage)
case "myprovider":
    storageProvider = storage.NewMyProvider(cfg.Storage)
default:
    storageProvider = storage.NewLocalProvider(cfg.Storage)
}
```

### Step 3: Add configuration

In `config/config.go`, add config fields for the new provider.
In `.env.example`, document the new environment variables.

---

## Recipe 10: Write a Test for a Handler

Tests live alongside source files. Naming convention: `TestFeature_WhenCondition_ExpectedResult`.

**Example:** testing `POST /tickets`:

```go
// ticket_handler_test.go

func TestCreateTicket_WhenQuotaExceeded_Returns422(t *testing.T) {
    // 1. Create a mock service that returns quota exceeded
    mockService := &mockTicketService{
        createErr: apperrors.QuotaExceeded(apperrors.CodeQuotaExceeded),
    }

    // 2. Build the handler
    h := NewTicketHandler(mockService, nil)

    // 3. Create a test request with injected claims
    body := `{"title":"Test ticket"}`
    req := httptest.NewRequest("POST", "/tickets", strings.NewReader(body))
    req = req.WithContext(middleware.TestInjectClaims(req.Context(), middleware.Claims{
        UserID: uuid.New(),
        OrgID:  uuid.New(),
        Permissions: []string{"tickets:create"},
    }))
    req.Header.Set("Content-Type", "application/json")

    // 4. Record the response
    rr := httptest.NewRecorder()
    h.Create(rr, req)

    // 5. Assert
    assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
    
    var resp map[string]any
    json.Unmarshal(rr.Body.Bytes(), &resp)
    assert.Equal(t, "QUOTA_EXCEEDED", resp["error"].(map[string]any)["code"])
}
```

Run: `make test-api`

---

## Architecture Decision Reference

When you are unsure where code belongs, use this table:

| Code type | Correct layer | Folder |
|---|---|---|
| Entity struct (e.g., `Ticket`) | Domain | `internal/domain/ticket/` |
| Repository interface (`Repository`) | Domain | `internal/domain/ticket/` |
| Business rule ("you can't close a ticket without resolving it first") | Application | `internal/application/ticket/service.go` |
| SQL query | Infrastructure | `internal/infrastructure/postgres/ticket_repo.go` |
| HTTP request parsing | Interfaces | `internal/interfaces/http/handlers/ticket_handler.go` |
| Route registration | Interfaces | `internal/interfaces/http/router.go` |
| Environment variable | Config | `config/config.go` |
| Dependency injection | Entry point | `cmd/server/main.go` |
| Reusable utility (pagination, errors, logger) | Pkg | `pkg/` |
| Frontend page | Routes | `apps/web/src/routes/(app)/` |
| Frontend data fetch | API client | `apps/web/src/lib/api/` |
| Frontend state | Store | `apps/web/src/lib/stores/` |
| Frontend permission check | Permissions helper | `apps/web/src/lib/permissions/index.ts` |
| Translation string | i18n | `apps/web/src/lib/i18n/en.json` + `pt.json` |
