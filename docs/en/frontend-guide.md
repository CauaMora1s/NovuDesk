# Frontend Guide

The frontend is a SvelteKit SPA (Single Page Application) with static rendering, TailwindCSS + DaisyUI for styling, svelte-i18n for translations, and a typed API client layer.

**Source root:** `apps/web/src/`

---

## Routing Overview

SvelteKit uses file-based routing. Files inside `routes/` define pages.

```
routes/
├── +layout.svelte     ← Root layout (theme setup)
├── +layout.ts         ← Disables SSR + initializes i18n
├── +page.svelte       ← Redirect page (→ /login or /dashboard)
├── (auth)/            ← Unauthenticated layout group (no sidebar)
│   ├── +layout.svelte
│   ├── +layout.ts     ← Redirects to /dashboard if already logged in
│   └── login/
│       └── +page.svelte
└── (app)/             ← Authenticated layout group (with sidebar)
    ├── +layout.svelte ← Sidebar wrapper
    ├── +layout.ts     ← Redirects to /login if not authenticated
    ├── dashboard/
    │   └── +page.svelte
    ├── tickets/
    │   ├── +page.svelte
    │   ├── new/
    │   │   └── +page.svelte
    │   └── [id]/
    │       └── +page.svelte
    ├── teams/
    │   ├── +page.svelte
    │   └── +page.ts
    └── settings/
        └── +page.svelte
```

Layout groups (`(auth)` and `(app)`) do not create URL segments — they only group pages under a shared layout.

---

## Pages

### `/` — Root Redirect

**File:** `routes/+page.svelte`

Immediately redirects to `/dashboard` (or `/login` if not authenticated). No content is rendered — the redirect happens in `+layout.ts`.

---

### `/login` — Login Page

**File:** `routes/(auth)/login/+page.svelte`
**Load function:** `routes/(auth)/+layout.ts` (redirects away if already authenticated)
**Permissions required:** none (public page)

**What it does:**
- Renders a login form with three fields: **Organization Slug**, **Email**, **Password**
- On submit, calls `POST /api/v1/auth/login`
- Decodes the returned JWT using `decodeJwt()` from `lib/stores/auth.ts`
- Saves decoded session to `localStorage` as `nd_session`
- Populates the `authStore` with user ID, org ID, role, permissions array, and team IDs
- Redirects to `/dashboard` on success

**Key data flows:**
```
Form submit → POST /auth/login → JWT → decodeJwt() → authStore.set() → navigate('/dashboard')
```

**Where to change it:**
- Add a field to the login form → add the HTML input + include in the fetch body
- Change where to redirect after login → change `goto('/dashboard')` at the bottom of the submit handler
- Add "forgot password" link → add the anchor below the form

---

### `/dashboard` — Dashboard

**File:** `routes/(app)/dashboard/+page.svelte`
**Permissions required:** authenticated (any role)

**What it does:**
- Shows three summary cards: **Open Tickets**, **SLA Breached**, **Resolved Today**
- Shows a table of the 10 most recently updated tickets
- Polls for updates at the interval set in `pollingInterval` store

**Key data flows:**
```
onMount → GET /tickets (status=open, per_page=1 for count) 
        → GET /tickets (sla_breached=true, per_page=1 for count)
        → GET /tickets (status=resolved, sorted by updated_at)
        → render cards + table
Polling interval change → re-fetch all data
```

**Where to change it:**
- Add a new summary card → fetch the relevant metric, add a card element using the same DaisyUI `stats` pattern
- Change the recent tickets table columns → edit the `<thead>` and `<td>` columns in the tickets table section
- Change polling behavior → the `pollingInterval` store drives the `setInterval` in `onMount`

---

### `/tickets` — Ticket List

**File:** `routes/(app)/tickets/+page.svelte`
**Permissions required:** `tickets:view`

**What it does:**
- Renders a filterable, sortable ticket table
- Filter bar: Status (multi-select), Priority (multi-select), SLA Breached (toggle)
- Smart search: detects if input is a ticket number (digits only) → uses `number=` param; if UUID → uses `id=`; otherwise uses full-text `q=` param
- Sort options: Created At, Updated At, SLA Due
- Polls for live updates

**Key data flows:**
```
onMount → GET /tickets with current filters → render table
Filter change → update URL params → re-fetch
Search input → debounce 300ms → detect type → re-fetch
Polling → setInterval(re-fetch, pollingInterval)
```

**Where to change it:**
- Add a new filter → add a `<select>` or toggle to the filter bar, add the param to the `buildParams()` function, add it to the API call in `lib/api/tickets.ts` `ListParams` type
- Change table columns → edit the `<thead>` / `<td>` structure
- Change sort options → add option to the sort `<select>` and include the `sort` param in the fetch

---

### `/tickets/new` — Create Ticket

**File:** `routes/(app)/tickets/new/+page.svelte`
**Permissions required:** `tickets:create`

**What it does:**
- Form with: Title, Description, Priority (select), Category (grouped by team via `SearchSelect`)
- When a category is selected, fetches its SLA policy and shows a preview card: "Response due in X hours, Resolution due in Y hours"
- File upload section using `FileUpload` component (drag-and-drop, multi-file)
- On submit: creates the ticket first, then uploads each attachment sequentially

**Key data flows:**
```
onMount → GET /categories → GET /teams → group categories by team → populate SearchSelect
Category select → GET /sla-policies → find matching policy → render SLA preview
Submit → POST /tickets → for each file: POST /tickets/{id}/attachments → navigate('/tickets/{id}')
```

**Where to change it:**
- Add a new field (e.g., Tags) → add the form element, add the field to the submit payload, add the type to `CreateInput` in `lib/api/tickets.ts`
- Change the SLA preview format → edit the SLA info card below the category selector
- Change max file size or allowed types → these are enforced server-side in `attachment_handler.go`; the frontend `FileUpload` component can show client-side validation but the server is authoritative

---

### `/tickets/[id]` — Ticket Detail

**File:** `routes/(app)/tickets/[id]/+page.svelte`
**Permissions required:** `tickets:view`

**What it does:**
- Left panel: Ticket metadata (title, description, status badge, priority badge)
  - Inline status change: `<select>` calls `PATCH /tickets/{id}` on change (requires `tickets:edit`)
  - Inline priority change: same pattern
  - Inline assignee change: searchable dropdown (requires `tickets:edit`)
  - Category and team display
- Right panel: Ticket info sidebar (requester, created date, SLA due dates, tags)
- Bottom: Unified timeline (comments + activity events from `GET /tickets/{id}/comments`)
  - Comment composer: textarea + internal toggle (hidden if no `comments:create_internal` permission)
  - Attachment list with download links
  - File upload for new attachments

**Key data flows:**
```
onMount → GET /tickets/{id} → render metadata
        → GET /tickets/{id}/comments → render timeline
        → GET /tickets/{id}/attachments → render attachment list
Status/Priority select → PATCH /tickets/{id} → re-fetch ticket
Comment submit → POST /tickets/{id}/comments → re-fetch timeline
File upload → POST /tickets/{id}/attachments → re-fetch attachments
Polling → setInterval(re-fetch ticket + timeline, pollingInterval)
```

**Where to change it:**
- Add a new editable field in the sidebar → add a `<select>` or input that calls `PATCH /tickets/{id}` on change
- Change the timeline item display → edit the `{#each timeline as item}` block; `item.type === 'comment'` vs `item.type === 'activity'` branches control the render
- Add a new activity event type display → add a case in the activity rendering section
- Show internal comments differently → they already have `is_internal: true`; adjust the CSS class on the comment card

---

### `/teams` — Team Management

**File:** `routes/(app)/teams/+page.svelte`
**Load function:** `routes/(app)/teams/+page.ts`
**Permissions required:** `teams:view` (read); `teams:manage` (create/edit)

**What it does:**
- Left panel: List of teams (user's teams for non-admins, all teams for admins/owners)
- Right panel: Selected team detail with:
  - Members tab: list of team members + add member button (modal)
  - Categories tab: list of linked categories + create/link category buttons (modals)
- Modals:
  - Create Team
  - Add Member to Team (SearchSelect of org members)
  - Create Category
  - Link existing Category to Team

**Key data flows:**
```
onMount → GET /teams → render list
Team click → GET /teams/{id}/members + GET /teams/{id}/categories → render detail
Create team modal submit → POST /teams → re-fetch list
Add member modal submit → POST /teams/{id}/members → re-fetch members
Create category modal submit → POST /categories → POST /teams/{id}/categories → re-fetch
```

**Where to change it:**
- Add team edit functionality → add an "Edit" button that opens a modal with PATCH /teams/{id}
- Add delete team button → add button (gated with `can('teams:manage')`), call `DELETE /teams/{id}`
- Add category edit functionality → add edit modal with PATCH /categories/{id}

---

### `/settings` — Organization Settings

**File:** `routes/(app)/settings/+page.svelte`
**Permissions required:** `organization:view_settings` (read); `organization:manage_settings` (write)

**What it does:**
This is the largest page. It has four tabs:

#### Tab 1: Organization
- Shows org name, slug, logo, current plan, usage progress bars (members used/limit, tickets used/limit, storage used/limit)
- Billing session status: if a plan change is pending, shows "Confirm" and "Cancel" buttons
- Plan catalog: list of available plans with "Switch to this plan" button
- Rename org form (requires `organization:manage_settings`)

#### Tab 2: Members
- Paginated member table: name, email, role badge, active status
- Per-member actions (dropdown):
  - Edit Profile → inline form with PATCH /members/{id}/profile
  - Change Password → modal with PATCH /members/{id}/password
  - Change Role → dropdown with PATCH /members/{id}
  - Manage Permissions → opens `PermissionMatrix` component with PUT /members/{id}/permissions
  - Deactivate / Activate → DELETE or POST /members/{id}/activate
- Add Member button → modal with POST /members

#### Tab 3: Roles
- List of custom org roles with permission badges
- Create Role button → form with name input + `PermissionMatrix` component → POST /roles
- Per-role: Edit → PATCH /roles/{id} | Delete → DELETE /roles/{id}
- System roles (owner, admin, agent, viewer) are shown but cannot be edited or deleted

#### Tab 4: SLA
- List of categories with their SLA policies (response hours, resolution)
- Per-category: "Set SLA" or "Edit" → form with resolution value/unit + response hours → PUT /sla-policies/category/{id}
- Delete SLA → DELETE /sla-policies/{id}

**Key data flows:**
```
onMount → GET /organization → GET /organization/plans → GET /organization/plan/sessions
        → GET /members → GET /roles → GET /permissions → GET /categories → GET /sla-policies
Tab switch → use already-loaded data (no re-fetch unless stale)
```

**Where to change it:**
- Add a new organization setting field → add to Tab 1, call PATCH /organization in submit handler
- Add a new member action → add menu item in the actions dropdown, wire up the API call
- Add a new system permission → add to `GET /permissions` response (server side), it will automatically appear in `PermissionMatrix`
- Add a new SLA field → update the SLA form and the `PUT /sla-policies/category/{id}` request

---

## Stores

Stores are Svelte's reactive state containers. They live in `apps/web/src/lib/stores/`.

### `auth.ts` — Authentication Store

**Exports:**
- `authStore` — writable store containing the full session object
- `isAuthenticated` — derived boolean (true if `authStore` has a non-null user)
- `currentUser` — derived object with just the user fields

**Session shape (stored in `localStorage` as `nd_session`):**

```typescript
{
  user_id: string;
  org_id: string;
  role: string;           // 'owner' | 'admin' | 'agent' | 'viewer' | custom
  permissions: string[];  // e.g. ['tickets:view', 'tickets:create']
  team_ids: string[];     // teams the user belongs to
  access_token: string;   // the JWT itself
}
```

**`decodeJwt(token: string)`** — utility that base64-decodes the JWT payload (no signature verification — the server validates the signature on every request).

**How auth state gets populated:**
1. Login page calls `POST /auth/login`
2. Response contains `access_token` (JWT)
3. `decodeJwt()` extracts the payload
4. `authStore.set({ ...payload, access_token })` updates the store
5. Store subscriber writes to `localStorage`
6. On page load, `+layout.ts` reads from `localStorage` to restore session

**Where to change it:**
- Add a new field to the session → add it to the JWT claims in `application/auth/service.go`, decode it in `decodeJwt()`, add to the store shape
- Change the localStorage key → update `nd_session` constant in this file

---

### `polling.ts` — Polling Interval Store

**Exports:**
- `pollingInterval` — writable store of type `PollingInterval`

```typescript
type PollingInterval = 10_000 | 30_000 | 60_000 | 0;
// 10 seconds | 30 seconds | 1 minute | disabled
```

Pages that display live data use `setInterval` driven by this store value. When `pollingInterval === 0`, polling is disabled (user prefers SSE only or wants static view).

**Where to change it:** edit `polling.ts` to add new interval options or change defaults.

---

## Permission System

**File:** `apps/web/src/lib/permissions/index.ts`

### Functions

```typescript
can(permission: Permission): boolean
canAny(permissions: Permission[]): boolean
canAll(permissions: Permission[]): boolean
```

All three functions read from `authStore.permissions` at call time.

- `can('tickets:create')` → true if the user has that exact permission
- `canAny(['tickets:edit', 'tickets:delete'])` → true if the user has at least one
- `canAll(['users:view', 'users:edit'])` → true if the user has all of them

### Usage in Svelte templates

```svelte
{#if can('tickets:create')}
  <button>New Ticket</button>
{/if}

{#if canAny(['teams:manage', 'organization:manage_settings'])}
  <AdminPanel />
{/if}
```

**Important:** Elements are completely removed from the DOM when the condition is false — not just hidden with CSS. This is intentional: hiding elements with CSS can be inspected and overridden; removing them from the DOM is safer.

### Complete Permission Type

```typescript
type Permission =
  | 'tickets:view'
  | 'tickets:create'
  | 'tickets:edit'
  | 'tickets:delete'
  | 'comments:create'
  | 'comments:view_internal'
  | 'comments:create_internal'
  | 'teams:view'
  | 'teams:manage'
  | 'users:view'
  | 'users:create'
  | 'users:edit'
  | 'users:deactivate'
  | 'organization:view_settings'
  | 'organization:manage_settings'
  | 'sla:view'
  | 'sla:manage'
  | 'reports:view'
  | 'api:manage';
```

---

## API Client Layer

**Folder:** `apps/web/src/lib/api/`

### `client.ts` — Base HTTP Client

The `request<T>()` function handles:
- Adding `Authorization: Bearer <token>` from `authStore`
- 15-second request timeout
- Redirecting to `/login` on 401 (token expired)
- Throwing a typed error on non-ok responses

```typescript
const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  delete: <T>(path: string) => request<T>('DELETE', path),
};
```

All API modules import and use this `api` object.

### API Modules

| File | Exports | Endpoints used |
|---|---|---|
| `tickets.ts` | `ticketsApi` | GET/POST/PATCH/DELETE /tickets |
| `comments.ts` | `commentsApi` | GET/POST /tickets/{id}/comments |
| `attachments.ts` | `attachmentsApi` | GET /tickets/{id}/attachments, POST (multipart) |
| `members.ts` | `membersApi` | All /members endpoints |
| `roles.ts` | `rolesApi` | All /roles + /permissions endpoints |
| `teams.ts` | `teamsApi` | All /teams endpoints |
| `categories.ts` | `categoriesApi` | All /categories endpoints |
| `sla.ts` | `slaApi` | All /sla-policies endpoints |
| `organization.ts` | `organizationApi` | All /organization endpoints |

### Adding a New API Call

1. Find the relevant module file (e.g., `tickets.ts`)
2. Add a TypeScript type for the request/response if needed
3. Add a new method to the exported API object:
   ```typescript
   export const ticketsApi = {
     // existing methods...
     export: (params: ExportParams) =>
       api.post<ExportResponse>('/tickets/export', params),
   };
   ```
4. Call it from the Svelte page: `const result = await ticketsApi.export(params)`

---

## Components

**Folder:** `apps/web/src/lib/components/`

### `layout/Sidebar.svelte`

The main navigation sidebar visible on all `(app)` pages.

**Nav items and their permission gates:**

| Item | Route | Permission required |
|---|---|---|
| Dashboard | `/dashboard` | none |
| Tickets | `/tickets` | `tickets:view` |
| Teams | `/teams` | `teams:view` |
| Settings | `/settings` | `organization:view_settings` |

The user's name and role are shown at the bottom with a Logout button.

**Where to change it:**
- Add a nav item → add an `<a>` tag inside the `<nav>` section, wrap with `{#if can('permission')}` if needed
- Change the logo → edit the logo section at the top of the sidebar

---

### `ui/PermissionMatrix.svelte`

A grid of toggle checkboxes used to assign permissions to roles or individual members. Used in Settings → Roles (create/edit) and Settings → Members (manage permissions).

**Props:**
- `permissions: Permission[]` — all available system permissions
- `selected: Permission[]` — currently selected permissions
- `on:change` — event emitting the new selected array

**Where to change it:** the component receives the full permission list from `GET /permissions`. Adding new permissions server-side automatically makes them appear here.

---

### `ui/SearchSelect.svelte`

A searchable dropdown input. Used for category selection in the new ticket form and for assignee selection in the ticket detail page.

**Props:**
- `options: { id: string, label: string }[]` — available options
- `value: string | null` — currently selected option ID
- `placeholder: string` — placeholder text
- `on:select` — event emitting the selected option

---

### `FileUpload.svelte`

A drag-and-drop file uploader. Used in the create ticket form and the ticket detail page.

**Props:**
- `multiple: boolean` — allow multiple files (default: false)
- `accept: string` — MIME type filter for the file picker
- `on:files` — event emitting an array of `File` objects

**Note:** The component does not upload files itself — it emits them to the parent page which handles the actual `POST /attachments` call.

---

### `PollingControl.svelte`

A small UI control (usually a select or toggle) that lets the user change the polling interval. Updates the `pollingInterval` store which all live-updating pages subscribe to.

---

## Internationalization (i18n)

**Folder:** `apps/web/src/lib/i18n/`

NovuDesk uses [svelte-i18n](https://github.com/kaisermann/svelte-i18n) for translations.

### How to use translations in a component

```svelte
<script>
  import { t } from 'svelte-i18n';
</script>

<h1>{$t('tickets.title')}</h1>
<button>{$t('common.save')}</button>
```

**Never hardcode user-visible strings.** Always use `$t('key')`.

### Translation files

- `en.json` — English (primary development language)
- `pt.json` — Portuguese

Both files must have identical keys. Values differ by language.

### Adding a new translation key

1. Add the key and English value to `en.json`:
   ```json
   { "tickets": { "export_button": "Export CSV" } }
   ```
2. Add the same key and Portuguese value to `pt.json`:
   ```json
   { "tickets": { "export_button": "Exportar CSV" } }
   ```
3. Use in your component: `{$t('tickets.export_button')}`

### Adding a new language

1. Copy `en.json` to `<lang>.json` (e.g., `es.json` for Spanish)
2. Translate all values
3. Register in `i18n/index.ts`:
   ```typescript
   register('es', () => import('./es.json'));
   ```
4. The browser's `navigator.language` is used for auto-detection with `pt` as the fallback

### i18n initialization

`apps/web/src/lib/i18n/index.ts` calls `register()` for each locale and `init()` with the browser locale and fallback. This runs before any page renders (called from `routes/+layout.ts`).
