# API Reference

**Base URL:** `http://localhost:8080/api/v1` (development)

All endpoints return JSON. Protected endpoints require a valid JWT access token in the `Authorization` header.

---

## Authentication

### Headers

```
Authorization: Bearer <access_token>
```

The access token is obtained from `POST /auth/login`. It expires in 15 minutes.

### Refresh Token Flow

A `refresh_token` HttpOnly cookie is set on login. When the access token expires, call `POST /auth/logout` to clear it, or implement a refresh endpoint (see ROADMAP — currently tokens are refreshed via re-login in the MVP).

### Error Response Format

All errors follow this shape:

```json
{
  "error": {
    "code": "TICKET_NOT_FOUND",
    "message": "ticket not found",
    "details": {}
  }
}
```

| Field | Type | Description |
|---|---|---|
| `code` | string | Machine-readable error code (see table below) |
| `message` | string | Human-readable message |
| `details` | object | Optional field-level validation errors |

### Common Error Codes

| Code | HTTP Status | Meaning |
|---|---|---|
| `UNAUTHORIZED` | 401 | Missing or invalid JWT |
| `FORBIDDEN` | 403 | Valid JWT but missing required permission |
| `VALIDATION_ERROR` | 422 | Request body failed validation |
| `TICKET_NOT_FOUND` | 404 | Ticket does not exist or belongs to another org |
| `USER_NOT_FOUND` | 404 | User/member not found |
| `ROLE_NOT_FOUND` | 404 | Role not found |
| `TEAM_NOT_FOUND` | 404 | Team not found |
| `CATEGORY_NOT_FOUND` | 404 | Category not found |
| `SLA_NOT_FOUND` | 404 | SLA policy not found |
| `ORG_NOT_FOUND` | 404 | Organization not found |
| `EMAIL_ALREADY_EXISTS` | 409 | Email already registered |
| `ROLE_NAME_CONFLICT` | 409 | Role name already used in this org |
| `SYSTEM_ROLE_PROTECTED` | 409 | Cannot modify or delete a system role |
| `QUOTA_EXCEEDED` | 422 | Plan limit reached (tickets, members, storage) |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

### Pagination

List endpoints accept:

| Param | Type | Default | Description |
|---|---|---|---|
| `page` | int | `1` | Page number (1-based) |
| `per_page` | int | `25` | Items per page (max: 100) |

Responses include a `meta` object:

```json
{
  "data": [...],
  "meta": {
    "total": 143,
    "page": 1,
    "per_page": 25
  }
}
```

### Permission Strings

All permission checks use strings in the format `resource:action`. The complete list:

| Permission | Granted to (default) |
|---|---|
| `tickets:view` | agent, viewer, owner, admin |
| `tickets:create` | agent, owner, admin |
| `tickets:edit` | agent, owner, admin |
| `tickets:delete` | owner, admin |
| `comments:view_internal` | agent, owner, admin |
| `comments:create` | agent, owner, admin |
| `comments:create_internal` | agent, owner, admin |
| `teams:view` | agent, viewer, owner, admin |
| `teams:manage` | owner, admin |
| `users:view` | owner, admin |
| `users:create` | owner, admin |
| `users:edit` | owner, admin |
| `users:deactivate` | owner, admin |
| `organization:view_settings` | owner, admin |
| `organization:manage_settings` | owner, admin |
| `sla:view` | agent, owner, admin |
| `sla:manage` | owner, admin |
| `reports:view` | owner, admin |
| `api:manage` | owner, admin |

> Viewer role can only read their own tickets — they cannot see the full ticket list.

---

## Health Check

### GET /health

No authentication required.

**Response 200:**
```json
{ "status": "ok" }
```

**Where to find:** `apps/api/internal/interfaces/http/router.go`

---

## Auth

### POST /api/v1/auth/login

No authentication required.

**Description:** Authenticates a user against an organization and returns a JWT access token.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | yes | User email address |
| `password` | string | yes | User password (min 8 chars) |
| `org_slug` | string | yes | Organization slug (e.g., `acme`) |

```json
{
  "email": "admin@acme.com",
  "password": "password123",
  "org_slug": "acme"
}
```

**Response 200:**

```json
{
  "data": {
    "access_token": "eyJhbGciOiJSUzI1NiIs..."
  }
}
```

A `refresh_token` HttpOnly cookie is also set on the response.

The access token payload (decoded) contains:
```json
{
  "sub": "<user_id>",
  "org_id": "<org_id>",
  "role": "admin",
  "permissions": ["tickets:view", "tickets:create", "..."],
  "team_ids": ["<team_id_1>"],
  "exp": 1234567890
}
```

**Error codes:** `UNAUTHORIZED` (wrong password), `USER_NOT_FOUND`, `ORG_NOT_FOUND`

**Where to find:** `apps/api/internal/interfaces/http/handlers/auth_handler.go` → `Login()`
**Service:** `apps/api/internal/application/auth/service.go` → `Login()`

---

### POST /api/v1/auth/register

No authentication required.

**Description:** Creates a new user account (not tied to an organization yet).

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | yes | User email address (must be unique) |
| `password` | string | yes | Password (min 8 chars) |
| `full_name` | string | yes | Display name |
| `locale` | string | no | `en` or `pt` (defaults to `pt`) |

**Response 201:**

```json
{
  "data": {
    "id": "<user_id>",
    "email": "user@example.com"
  }
}
```

**Error codes:** `EMAIL_ALREADY_EXISTS`, `VALIDATION_ERROR`

**Where to find:** `auth_handler.go` → `Register()`, `auth/service.go` → `Register()`

---

### POST /api/v1/auth/logout

No authentication required (but clears the cookie).

**Description:** Clears the refresh token cookie.

**Request body:** none

**Response 204:** no content

**Where to find:** `auth_handler.go` → `Logout()`

---

## Real-time Events

### GET /api/v1/events

**Permission required:** authenticated (any valid JWT)

**Description:** Opens a Server-Sent Events stream. The browser receives push notifications when tickets are created or updated in the organization.

**Event format:**

```
event: ticket.updated
data: {"ticket_id":"<id>","org_id":"<org_id>"}
```

**Event types:**
- `ticket.created`
- `ticket.updated`
- `comment.created`

**Notes:**
- Keep the connection open — it is a long-lived HTTP/1.1 stream
- The Nginx production config sets `proxy_buffering off` for this endpoint
- Redis pub/sub is used internally so multiple API instances can share the same SSE channel

**Where to find:** `apps/api/internal/interfaces/sse/manager.go`

---

## Tickets

### GET /api/v1/tickets

**Permission required:** `tickets:view`

**Description:** Returns a paginated list of tickets in the organization. Non-privileged users (viewers) only see tickets they created (requester) or are assigned to.

**Query parameters:**

| Param | Type | Description |
|---|---|---|
| `status` | string | Filter by status: `open`, `pending`, `on_hold`, `resolved`, `closed` |
| `priority` | string | Filter by priority: `low`, `normal`, `high`, `urgent` |
| `assignee` | string (UUID) | Filter by assignee user ID |
| `team` | string (UUID) | Filter by team ID |
| `category` | string (UUID) | Filter by category ID |
| `sla_breached` | bool | `true` to show only SLA-breached tickets |
| `number` | int | Find by ticket number (exact match) |
| `q` | string | Full-text search in title and description |
| `sort` | string | `created_at` (default), `updated_at`, `sla_due` |
| `page` | int | Page number (default: 1) |
| `per_page` | int | Items per page (default: 25, max: 100) |

**Response 200:**

```json
{
  "data": [
    {
      "id": "<uuid>",
      "number": 42,
      "title": "Login button not working",
      "description": "Detailed description...",
      "status": "open",
      "priority": "high",
      "org_id": "<uuid>",
      "assignee": {
        "id": "<uuid>",
        "full_name": "Jane Agent",
        "avatar_url": null
      },
      "requester": {
        "id": "<uuid>",
        "full_name": "John Customer",
        "avatar_url": null
      },
      "team": { "id": "<uuid>", "name": "Support" },
      "category": { "id": "<uuid>", "name": "Bug" },
      "tags": ["login", "critical"],
      "sla_response_due_at": "2026-05-30T10:00:00Z",
      "sla_resolution_due_at": "2026-05-30T18:00:00Z",
      "sla_breached": false,
      "resolved_at": null,
      "closed_at": null,
      "created_at": "2026-05-30T08:00:00Z",
      "updated_at": "2026-05-30T09:00:00Z"
    }
  ],
  "meta": { "total": 87, "page": 1, "per_page": 25 }
}
```

**Where to find:** `ticket_handler.go` → `List()`, `ticket_repo.go` → `List()`

---

### POST /api/v1/tickets

**Permission required:** `tickets:create`

**Description:** Creates a new ticket. Checks quota against the organization's plan. Auto-applies SLA if the selected category has an SLA policy.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | yes | Ticket title (max 500 chars) |
| `description` | string | no | Detailed description |
| `priority` | string | no | `low`, `normal` (default), `high`, `urgent` |
| `category_id` | string (UUID) | no | Ticket category |
| `team_id` | string (UUID) | no | Assigned team |
| `assignee_id` | string (UUID) | no | Assigned agent user ID |
| `tags` | string[] | no | Array of tag strings |
| `custom_fields` | object | no | Key-value pairs for custom fields |

**Response 201:**

```json
{
  "data": {
    "id": "<uuid>",
    "number": 43,
    "title": "Login button not working",
    "status": "open",
    "priority": "normal",
    "...": "..."
  }
}
```

**Error codes:** `QUOTA_EXCEEDED`, `VALIDATION_ERROR`, `USER_NOT_FOUND` (invalid assignee), `CATEGORY_NOT_FOUND`

**Where to find:** `ticket_handler.go` → `Create()`, `application/ticket/service.go` → `Create()`

---

### GET /api/v1/tickets/{id}

**Permission required:** `tickets:view`

**Description:** Returns a single ticket by UUID. Non-privileged users can only view tickets they are requester or assignee of.

**Path parameter:** `id` — ticket UUID

**Response 200:** same shape as a single item from the list endpoint.

**Error codes:** `TICKET_NOT_FOUND`, `FORBIDDEN`

**Where to find:** `ticket_handler.go` → `Get()`

---

### PATCH /api/v1/tickets/{id}

**Permission required:** `tickets:edit`

**Description:** Partially updates a ticket. Non-privileged users cannot change `status` or `assignee_id`, and cannot update tickets they do not own. Status transitions to `resolved` and `closed` automatically set `resolved_at` / `closed_at` timestamps.

**Path parameter:** `id` — ticket UUID

**Request body (all fields optional):**

| Field | Type | Description |
|---|---|---|
| `title` | string | New title |
| `description` | string | New description |
| `status` | string | `open`, `pending`, `on_hold`, `resolved`, `closed` |
| `priority` | string | `low`, `normal`, `high`, `urgent` |
| `assignee_id` | string (UUID) | New assignee (or `null` to unassign) |
| `team_id` | string (UUID) | Move to team (or `null`) |
| `category_id` | string (UUID) | Change category (recalculates SLA) |
| `tags` | string[] | Replace tags array |
| `custom_fields` | object | Merge custom field values |

**Response 200:** updated ticket object.

**Error codes:** `TICKET_NOT_FOUND`, `FORBIDDEN`, `VALIDATION_ERROR`

**Where to find:** `ticket_handler.go` → `Update()`, `application/ticket/service.go` → `Update()`

---

### DELETE /api/v1/tickets/{id}

**Permission required:** `tickets:delete`

**Description:** Deletes a ticket and all associated comments, attachments, and audit log entries.

**Path parameter:** `id` — ticket UUID

**Response 204:** no content.

**Error codes:** `TICKET_NOT_FOUND`, `FORBIDDEN`

**Where to find:** `ticket_handler.go` → `Delete()`

---

## Comments / Timeline

### GET /api/v1/tickets/{id}/comments

**Permission required:** `tickets:view`

**Description:** Returns the unified timeline for a ticket — a chronological mix of comments and audit activity events. Internal comments are filtered out for users without `comments:view_internal`.

**Path parameter:** `id` — ticket UUID

**Response 200:**

```json
{
  "data": [
    {
      "type": "comment",
      "id": "<uuid>",
      "body": "Have you tried clearing the cache?",
      "is_internal": false,
      "author": { "id": "<uuid>", "full_name": "Jane Agent", "avatar_url": null },
      "created_at": "2026-05-30T09:30:00Z"
    },
    {
      "type": "activity",
      "action": "status_changed",
      "actor": { "id": "<uuid>", "full_name": "Jane Agent" },
      "changes": { "status": { "from": "open", "to": "pending" } },
      "created_at": "2026-05-30T10:00:00Z"
    }
  ]
}
```

**Where to find:** `comment_handler.go` → `List()`

---

### POST /api/v1/tickets/{id}/comments

**Permission required:** `comments:create`

**Description:** Adds a comment to the ticket. Internal comments (visible only to agents) require `comments:create_internal`.

**Path parameter:** `id` — ticket UUID

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `body` | string | yes | Comment text |
| `is_internal` | bool | no | `true` for agent-only comments (default: `false`) |

**Response 201:**

```json
{
  "data": {
    "id": "<uuid>",
    "body": "Have you tried clearing the cache?",
    "is_internal": false,
    "author": { "id": "<uuid>", "full_name": "Jane Agent", "avatar_url": null },
    "created_at": "2026-05-30T09:30:00Z"
  }
}
```

**Error codes:** `TICKET_NOT_FOUND`, `FORBIDDEN` (creating internal without permission)

**Where to find:** `comment_handler.go` → `Create()`

---

## Attachments

### GET /api/v1/tickets/{id}/attachments

**Permission required:** `tickets:view`

**Description:** Returns all file attachments for a ticket.

**Response 200:**

```json
{
  "data": [
    {
      "id": "<uuid>",
      "name": "screenshot.png",
      "mime_type": "image/png",
      "size_bytes": 204800,
      "url": "http://localhost:9000/novudesk/attachments/<path>",
      "created_at": "2026-05-30T09:00:00Z"
    }
  ]
}
```

**Where to find:** `attachment_handler.go` → `List()`

---

### POST /api/v1/tickets/{id}/attachments

**Permission required:** `tickets:edit`

**Description:** Uploads a file attachment to a ticket. Checks storage quota against the organization's plan.

**Path parameter:** `id` — ticket UUID

**Request:** `multipart/form-data`

| Field | Type | Required | Description |
|---|---|---|---|
| `file` | file | yes | The file to upload (max 25 MB) |
| `comment_id` | string (UUID) | no | Link attachment to a specific comment |

**Allowed MIME types:** `image/png`, `image/jpeg`, `image/gif`, `image/webp`, `application/pdf`, `text/plain`, `application/zip`, `application/msword`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document`

**Response 201:**

```json
{
  "data": {
    "id": "<uuid>",
    "name": "screenshot.png",
    "mime_type": "image/png",
    "size_bytes": 204800,
    "url": "http://localhost:9000/novudesk/attachments/<path>"
  }
}
```

**Error codes:** `QUOTA_EXCEEDED`, `VALIDATION_ERROR` (file too large or bad MIME type)

**Where to find:** `attachment_handler.go` → `Upload()`

---

## Members

### GET /api/v1/members

**Permission required:** `users:view`

**Description:** Returns all active and inactive members of the organization.

**Query parameters:** `page`, `per_page`

**Response 200:**

```json
{
  "data": [
    {
      "id": "<member_id>",
      "user_id": "<user_id>",
      "full_name": "Jane Agent",
      "email": "agent@acme.com",
      "role_id": "<uuid>",
      "role_name": "agent",
      "is_active": true,
      "joined_at": "2026-05-01T00:00:00Z",
      "avatar_url": null
    }
  ],
  "meta": { "total": 5, "page": 1, "per_page": 25 }
}
```

**Where to find:** `member_handler.go` → `List()`

---

### POST /api/v1/members

**Permission required:** `users:create`

**Description:** Adds a new member to the organization. Creates the user account if the email does not already exist, then links them to the organization. Checks member quota against the plan.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `email` | string | yes | User email |
| `full_name` | string | yes | Display name |
| `role_id` | string (UUID) | yes | Role to assign |
| `team_id` | string (UUID) | no | Optionally add to a team immediately |

**Response 201:**

```json
{
  "data": {
    "member_id": "<uuid>",
    "user_id": "<uuid>",
    "email": "newagent@acme.com"
  }
}
```

**Error codes:** `QUOTA_EXCEEDED`, `ROLE_NOT_FOUND`, `TEAM_NOT_FOUND`

**Where to find:** `member_handler.go` → `Create()`

---

### PATCH /api/v1/members/{id}

**Permission required:** `users:edit`

**Description:** Changes the role of an organization member.

**Path parameter:** `id` — member UUID

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `role_id` | string (UUID) | yes | New role to assign |

**Response 200:** `{ "data": { "ok": true } }`

**Where to find:** `member_handler.go` → `UpdateRole()`

---

### DELETE /api/v1/members/{id}

**Permission required:** `users:deactivate`

**Description:** Deactivates a member (soft delete — preserves history). The member can no longer log in but their ticket history is preserved.

**Response 204:** no content.

**Where to find:** `member_handler.go` → `Deactivate()`

---

### POST /api/v1/members/{id}/activate

**Permission required:** `users:edit`

**Description:** Re-activates a previously deactivated member.

**Response 200:** `{ "data": { "ok": true } }`

**Where to find:** `member_handler.go` → `Activate()`

---

### PATCH /api/v1/members/{id}/profile

**Permission required:** `users:edit`

**Description:** Updates a member's full name or email address.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `full_name` | string | no | New display name |
| `email` | string | no | New email address |

**Response 200:** `{ "data": { "ok": true } }`

**Error codes:** `EMAIL_ALREADY_EXISTS`

**Where to find:** `member_handler.go` → `UpdateProfile()`

---

### PATCH /api/v1/members/{id}/password

**Permission required:** `users:edit`

**Description:** Sets a new password for a member.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `password` | string | yes | New password (min 8 chars) |

**Response 200:** `{ "data": { "ok": true } }`

**Where to find:** `member_handler.go` → `UpdatePassword()`

---

### GET /api/v1/members/{id}/permissions

**Permission required:** `users:view`

**Description:** Returns the effective permissions for a member: the role's base permissions merged with any per-member permission overrides (additions and removals).

**Response 200:**

```json
{
  "data": {
    "role_permissions": ["tickets:view", "tickets:create", "..."],
    "overrides": [
      { "permission_key": "users:view", "granted": true },
      { "permission_key": "tickets:delete", "granted": false }
    ],
    "effective": ["tickets:view", "tickets:create", "users:view"]
  }
}
```

**Where to find:** `member_handler.go` → `GetPermissions()`

---

### PUT /api/v1/members/{id}/permissions

**Permission required:** `users:edit`

**Description:** Replaces the per-member permission overrides for a member. This does not change the role — it creates exceptions on top of the role's permissions.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `overrides` | array | yes | Array of `{ permission_key, granted }` objects |

```json
{
  "overrides": [
    { "permission_key": "users:view", "granted": true },
    { "permission_key": "tickets:delete", "granted": false }
  ]
}
```

**Response 200:** `{ "data": { "ok": true } }`

**Where to find:** `member_handler.go` → `SetPermissions()`

---

## Roles

### GET /api/v1/permissions

**Permission required:** `organization:view_settings`

**Description:** Returns the full list of all permission strings available in the system. Used to populate the permission matrix UI.

**Response 200:**

```json
{
  "data": [
    { "key": "tickets:view", "description": "View tickets" },
    { "key": "tickets:create", "description": "Create tickets" },
    "..."
  ]
}
```

**Where to find:** `role_handler.go` → `ListPermissions()`

---

### GET /api/v1/roles

**Permission required:** `organization:view_settings`

**Description:** Lists all roles for the organization (system roles + custom org roles), each with their assigned permissions.

**Response 200:**

```json
{
  "data": [
    {
      "id": "<uuid>",
      "name": "agent",
      "is_system_role": true,
      "permissions": ["tickets:view", "tickets:create", "..."]
    },
    {
      "id": "<uuid>",
      "name": "Tier 2 Support",
      "is_system_role": false,
      "permissions": ["tickets:view", "tickets:edit", "users:view"]
    }
  ]
}
```

**Where to find:** `role_handler.go` → `List()`

---

### GET /api/v1/roles/{id}

**Permission required:** `organization:view_settings`

**Description:** Returns a single role with its permissions.

**Where to find:** `role_handler.go` → `Get()`

---

### POST /api/v1/roles

**Permission required:** `organization:manage_settings`

**Description:** Creates a custom role for the organization.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Role name (must be unique in the org) |
| `permissions` | string[] | yes | Array of permission key strings |

**Response 201:** created role object.

**Error codes:** `ROLE_NAME_CONFLICT`, `VALIDATION_ERROR`

**Where to find:** `role_handler.go` → `Create()`

---

### PATCH /api/v1/roles/{id}

**Permission required:** `organization:manage_settings`

**Description:** Updates a custom role's name or permissions. System roles cannot be modified.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | no | New role name |
| `permissions` | string[] | no | New permission list (replaces existing) |

**Error codes:** `SYSTEM_ROLE_PROTECTED`, `ROLE_NAME_CONFLICT`

**Where to find:** `role_handler.go` → `Update()`

---

### DELETE /api/v1/roles/{id}

**Permission required:** `organization:manage_settings`

**Description:** Deletes a custom role. System roles cannot be deleted.

**Response 204:** no content.

**Error codes:** `SYSTEM_ROLE_PROTECTED`, `FORBIDDEN`

**Where to find:** `role_handler.go` → `Delete()`

---

## Teams

### GET /api/v1/teams

**Permission required:** `teams:view`

**Description:** Returns teams the user belongs to. Admins and owners see all teams.

**Response 200:**

```json
{
  "data": [
    {
      "id": "<uuid>",
      "name": "Support",
      "description": "Customer support team",
      "created_at": "2026-05-01T00:00:00Z"
    }
  ]
}
```

**Where to find:** `team_handler.go` → `List()`

---

### POST /api/v1/teams

**Permission required:** `teams:manage`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Team name |
| `description` | string | no | Team description |

**Response 201:** created team object.

**Where to find:** `team_handler.go` → `Create()`

---

### GET /api/v1/teams/{id}

**Permission required:** `teams:view` (non-admins can only view their own teams)

**Response 200:** single team object.

---

### PATCH /api/v1/teams/{id}

**Permission required:** `teams:manage`

**Request body:** `name`, `description` (both optional).

**Response 200:** updated team object.

---

### DELETE /api/v1/teams/{id}

**Permission required:** `teams:manage`

**Response 204:** no content.

---

### GET /api/v1/teams/{id}/members

**Permission required:** `teams:view` (non-admin can only view their own team)

**Response 200:** array of member objects (same shape as `GET /members` items).

---

### POST /api/v1/teams/{id}/members

**Permission required:** `teams:manage`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `user_id` | string (UUID) | yes | Member to add |

**Response 201:** `{ "data": { "ok": true } }`

---

### DELETE /api/v1/teams/{id}/members/{userId}

**Permission required:** `teams:manage`

**Response 204:** no content.

---

### GET /api/v1/teams/{id}/categories

**Permission required:** `teams:view`

**Response 200:** array of category objects linked to this team.

---

### POST /api/v1/teams/{id}/categories

**Permission required:** `teams:manage`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `category_id` | string (UUID) | yes | Category to link to this team |

**Response 201:** `{ "data": { "ok": true } }`

---

### DELETE /api/v1/teams/{id}/categories/{catId}

**Permission required:** `teams:manage`

**Response 204:** no content.

---

## Categories

### GET /api/v1/categories

**Permission required:** `teams:view`

**Description:** Returns all ticket categories for the organization.

**Response 200:**

```json
{
  "data": [
    {
      "id": "<uuid>",
      "name": "Bug",
      "description": "Software defects",
      "created_at": "2026-05-01T00:00:00Z"
    }
  ]
}
```

**Where to find:** `category_handler.go` → `List()`

---

### POST /api/v1/categories

**Permission required:** `teams:manage`

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Category name |
| `description` | string | no | Category description |

**Response 201:** created category object.

---

### PATCH /api/v1/categories/{id}

**Permission required:** `teams:manage`

**Request body:** `name`, `description` (both optional).

**Response 200:** updated category object.

---

### DELETE /api/v1/categories/{id}

**Permission required:** `teams:manage`

**Response 204:** no content.

---

## SLA Policies

### GET /api/v1/sla-policies

**Permission required:** `sla:view`

**Description:** Returns all SLA policies with per-category statistics (average resolution time, count of breached tickets).

**Response 200:**

```json
{
  "data": [
    {
      "id": "<uuid>",
      "name": "SLA",
      "category_id": "<uuid>",
      "category_name": "Bug",
      "response_hours": 4,
      "resolution_value": 24,
      "resolution_unit": "hours",
      "resolution_hours": 24,
      "is_active": true,
      "stats": {
        "avg_resolution_hours": 18.5,
        "breach_count": 3
      }
    }
  ]
}
```

**Where to find:** `sla_handler.go` → `List()`, `application/sla/service.go` → `ListWithCategoryStats()`

---

### PUT /api/v1/sla-policies/category/{categoryId}

**Permission required:** `sla:manage`

**Description:** Creates or updates the SLA policy for a ticket category. If a policy already exists for the category it is updated; otherwise a new one is created.

**Path parameter:** `categoryId` — category UUID

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `resolution_value` | int | yes | SLA resolution time value (e.g., `24`) |
| `resolution_unit` | string | yes | `hours`, `days`, `weeks` |
| `response_hours` | int | no | Optional first-response SLA in hours |

**Response 200 / 201:** upserted SLA policy object.

**Where to find:** `sla_handler.go` → `UpsertForCategory()`

---

### DELETE /api/v1/sla-policies/{id}

**Permission required:** `sla:manage`

**Response 204:** no content.

**Error codes:** `SLA_NOT_FOUND`

---

## Organization

### GET /api/v1/organization

**Permission required:** `organization:view_settings`

**Description:** Returns the organization details, current plan, usage metrics, and any pending billing session.

**Response 200:**

```json
{
  "data": {
    "id": "<uuid>",
    "name": "Acme Corp",
    "slug": "acme",
    "logo_url": null,
    "plan_tier": "free",
    "plan_renews_at": null,
    "billing_status": "active",
    "usage": {
      "members": 3,
      "tickets": 45,
      "storage_bytes": 10485760,
      "teams": 2,
      "categories": 4,
      "api_keys": 0
    },
    "pending_session": null
  }
}
```

**Where to find:** `organization_handler.go` → `Get()`

---

### PATCH /api/v1/organization

**Permission required:** `organization:manage_settings`

**Description:** Updates the organization name or logo URL.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | no | New organization display name |
| `logo_url` | string | no | URL to logo image |

**Response 200:** updated organization object.

---

### GET /api/v1/organization/plans

**Permission required:** `organization:view_settings`

**Description:** Returns the available subscription plans and their limits.

**Response 200:** array of plan objects with name, limits (members, tickets, storage), and pricing.

---

### GET /api/v1/organization/plan/sessions

**Permission required:** `organization:view_settings`

**Description:** Returns the billing session history (plan change log).

**Response 200:** array of billing session objects.

---

### POST /api/v1/organization/plan/sessions

**Permission required:** `organization:manage_settings`

**Description:** Initiates a plan change request. Creates a pending billing session that must be confirmed.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `plan_tier` | string | yes | Target plan identifier (e.g., `pro`, `enterprise`) |

**Response 201:** pending billing session object with `id`.

---

### POST /api/v1/organization/plan/sessions/{id}/confirm

**Permission required:** `organization:manage_settings`

**Description:** Confirms a pending plan change session and applies the new plan to the organization.

**Response 200:** `{ "data": { "ok": true } }`

---

### POST /api/v1/organization/plan/sessions/{id}/cancel

**Permission required:** `organization:manage_settings`

**Description:** Cancels a pending plan change session.

**Response 200:** `{ "data": { "ok": true } }`

---

## Standard Response Wrappers

All responses use these shapes (defined in `apps/api/internal/interfaces/http/respond/respond.go`):

| Function | Status | Shape |
|---|---|---|
| `respond.Ok(w, data, meta)` | 200 | `{ data, meta? }` |
| `respond.Created(w, data)` | 201 | `{ data }` |
| `respond.NoContent(w)` | 204 | (empty body) |
| `respond.Error(w, err)` | varies | `{ error: { code, message, details? } }` |
| `respond.ValidationError(w, errors)` | 422 | `{ error: { code: "VALIDATION_ERROR", details: { field: "message" } } }` |
| `respond.Unauthorized(w)` | 401 | `{ error: { code: "UNAUTHORIZED" } }` |
| `respond.Forbidden(w)` | 403 | `{ error: { code: "FORBIDDEN" } }` |
