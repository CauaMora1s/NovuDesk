# NovuDesk

Multi-tenant, open-source, production-ready helpdesk system.

> Read in: [Português](./README.md)

---

## About

NovuDesk is a customer support platform built for teams that need a robust, self-hostable, and extensible solution. Designed as both a real commercial product and a high-quality portfolio project.

### Key features

- **Multi-tenant** — multiple isolated organizations
- **Ticket system** — create, assign, status, priority, tags, custom fields
- **Roles and granular permissions** — owner, admin, agent, viewer + custom roles
- **Teams** — group agents with ticket assignment
- **SLA** — response and resolution time policies with alerts
- **Comments** — public and internal (agent-only)
- **Attachments** — local in dev, S3/R2 in production
- **Email notifications** — via configurable SMTP
- **Realtime** — live updates via SSE (Server-Sent Events)
- **Audit log** — full change history with before/after diffs
- **Public API** — versioned, with API keys
- **Internationalization** — Portuguese and English, easy to add more languages

### Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.23, Clean Architecture |
| Database | PostgreSQL 16 |
| Cache / Queues | Redis 7 |
| Frontend | SvelteKit, TailwindCSS, DaisyUI |
| Migrations | Goose |
| Containerization | Docker, Docker Compose |

---

## Quick start

### Prerequisites

- Docker and Docker Compose
- Make
- OpenSSL (for JWT key generation)

### Initial setup

```bash
git clone https://github.com/novudesk/novudesk.git
cd novudesk

# Copy .env and generate RSA keys for JWT
make setup

# Start all services
make dev
```

### Available services

| Service | URL |
|---|---|
| Frontend | http://localhost:5173 |
| API | http://localhost:8080 |
| MailHog (emails) | http://localhost:8025 |
| MinIO (storage) | http://localhost:9001 |

### Running migrations and seeds

```bash
make migrate
make seed
```

**Development credentials:**

| Field | Value |
|---|---|
| Organization | `acme` |
| Email (owner) | `admin@acme.com` |
| Email (agent) | `agent@acme.com` |
| Password | `password123` |

---

## Available commands

```bash
make dev          # Start all services
make stop         # Stop all services
make logs         # Tail logs
make migrate      # Run pending migrations
make seed         # Insert development data
make test         # Run all tests
make lint         # Check code quality
make build        # Build production images
make keys         # Generate RSA key pair
```

---

## Contributing

Contributions are very welcome! See [CONTRIBUTING.md](./CONTRIBUTING.md) for the full guide.

---

## License

NovuDesk is distributed under the [GNU AGPL v3](./LICENSE) license.

This means any publicly hosted version must make its source code available. For commercial use under different terms, please get in touch.
