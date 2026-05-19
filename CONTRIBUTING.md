# Contributing to NovuDesk

Thank you for your interest in contributing! This document explains how to get started, what conventions to follow, and how to submit your work.

---

## Getting started

### 1. Fork and clone

```bash
git clone https://github.com/CauaMora1s/NovuDesk.git
cd NovuDesk
```

### 2. Set up the environment

```bash
make setup   # copies .env.example → .env and generates RSA keys
make dev     # starts all Docker services
make migrate # runs database migrations
make seed    # inserts development data
```

The application will be available at:
- Frontend: http://localhost:5173
- API: http://localhost:8080

### 3. Verify everything works

```bash
make test   # run all tests
make lint   # check code quality
```

---

## Workflow

1. Check the [Issues](https://github.com/novudesk/novudesk/issues) for something to work on
2. Comment on the issue to claim it (avoid duplicate work)
3. Create a branch from `main`: `git checkout -b feature/my-feature`
4. Make your changes
5. Run `make test` and `make lint` — both must pass
6. Open a Pull Request against `main`

---

## Branch naming

| Type | Format | Example |
|---|---|---|
| Feature | `feature/description` | `feature/ticket-attachments` |
| Bug fix | `fix/description` | `fix/sla-timezone-bug` |
| Documentation | `docs/description` | `docs/api-authentication` |
| Refactor | `refactor/description` | `refactor/audit-service` |

---

## Code conventions

### Go (backend)

- All comments in **English**
- Follow the Clean Architecture layer rules (see [ARCHITECTURE.md](./ARCHITECTURE.md))
- `domain/` must have zero external imports
- All repository calls must receive `ctx context.Context` and filter by `org_id`
- Use typed errors from `pkg/errors` — never return raw strings as errors
- Test file alongside source file: `ticket_service_test.go` next to `ticket_service.go`
- Test name format: `TestCreateTicket_WhenAssigneeNotFound_ReturnsError`

```bash
cd apps/api
golangci-lint run ./...
go test ./... -race
```

### Svelte (frontend)

- Use TypeScript in all `.svelte` files (`<script lang="ts">`)
- All user-visible strings must use `svelte-i18n` — no hardcoded text
- Permission checks use `can()`, `canAny()`, or `canAll()` — never hide elements with CSS
- Component files: PascalCase (`TicketCard.svelte`)
- Route files follow SvelteKit conventions (`+page.svelte`, `+layout.svelte`)

```bash
cd apps/web
pnpm lint
pnpm check
```

---

## Adding a new language

1. Copy `apps/web/src/lib/i18n/en.json` to your locale file, e.g. `es.json`
2. Translate all values (keep the keys in English)
3. Register the locale in `apps/web/src/lib/i18n/index.ts`
4. Open a PR with the title `i18n: add Spanish (es) translation`

---

## Database migrations

Create a new migration file:

```bash
# File name format: NNN_description.sql
# Example: 013_add_response_templates.sql
```

Every migration file must have both `-- +goose Up` and `-- +goose Down` sections.

Test your migration:

```bash
make migrate           # apply
make migrate-down      # roll back
make migrate           # apply again — must work cleanly
```

---

## Pull Request checklist

Before opening a PR, verify:

- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] New endpoints have corresponding handler tests
- [ ] New user-visible strings are in `pt.json` and `en.json`
- [ ] New database tables have migrations with Up and Down
- [ ] No secrets or `.env` files are committed
- [ ] The PR description explains what changed and why

---

## Commit message format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add SLA breach email notification
fix: resolve ticket number race condition on high load
docs: update API authentication guide
refactor: extract audit log writer to shared helper
test: add integration tests for ticket repository
chore: update goose to v3.22
```

---

## Questions?

Open an [Issue](https://github.com/novudesk/novudesk/issues) with the `question` label or start a [Discussion](https://github.com/novudesk/novudesk/discussions).
