# NovuDesk

Sistema de helpdesk multi-tenant, open-source e pronto para produção.

> Leia em: [English](./README.en.md)

---

## Sobre

NovuDesk é uma plataforma de atendimento ao cliente construída para times que precisam de uma solução robusta, auto-hospedável e extensível. Projetada como produto comercial real e como projeto de portfólio de alta qualidade.

### Principais recursos

- **Multi-tenant** — suporte a múltiplas organizações isoladas
- **Sistema de tickets** — criação, atribuição, status, prioridade, tags, campos customizados
- **Papéis e permissões granulares** — owner, admin, agente, viewer + papéis customizados
- **Times** — agrupamento de agentes com atribuição de tickets
- **SLA** — políticas de tempo de resposta e resolução com alertas
- **Comentários** — públicos e internos (apenas para agentes)
- **Anexos** — local em dev, S3/R2 em produção
- **Notificações por e-mail** — via SMTP configurável
- **Tempo real** — atualizações via SSE (Server-Sent Events)
- **Registro de auditoria** — histórico completo de alterações com antes/depois
- **API pública** — versionada, com API keys
- **Internacionalização** — Português e Inglês, fácil de adicionar novos idiomas

### Stack

| Camada | Tecnologia |
|---|---|
| Backend | Go 1.23, Clean Architecture |
| Banco de dados | PostgreSQL 16 |
| Cache / Filas | Redis 7 |
| Frontend | SvelteKit, TailwindCSS, DaisyUI |
| Migrations | Goose |
| Containerização | Docker, Docker Compose |

---

## Início rápido

### Pré-requisitos

- Docker e Docker Compose
- Make
- OpenSSL (para gerar chaves JWT)

### Configuração inicial

```bash
git clone https://github.com/novudesk/novudesk.git
cd novudesk

# Copia o .env e gera as chaves RSA para JWT
make setup

# Sobe todos os serviços
make dev
```

### Serviços disponíveis

| Serviço | URL |
|---|---|
| Frontend | http://localhost:5173 |
| API | http://localhost:8080 |
| MailHog (e-mails) | http://localhost:8025 |
| MinIO (storage) | http://localhost:9001 |

### Rodando as migrations e seeds

```bash
make migrate
make seed
```

**Credenciais de desenvolvimento:**

| Campo | Valor |
|---|---|
| Organização | `acme` |
| E-mail (owner) | `admin@acme.com` |
| E-mail (agente) | `agent@acme.com` |
| Senha | `password123` |

---

## Comandos disponíveis

```bash
make dev          # Sobe todos os serviços
make stop         # Para todos os serviços
make logs         # Acompanha os logs
make migrate      # Roda as migrations pendentes
make seed         # Insere dados de desenvolvimento
make test         # Roda todos os testes
make lint         # Verifica o código
make build        # Gera as imagens de produção
make keys         # Gera o par de chaves RSA
```

---

## Estrutura do projeto

```
novudesk/
├── apps/
│   ├── api/              # Backend Go
│   │   ├── cmd/server/   # Ponto de entrada e DI
│   │   ├── config/       # Configuração via variáveis de ambiente
│   │   ├── internal/
│   │   │   ├── domain/         # Entidades e interfaces (sem dependências externas)
│   │   │   ├── application/    # Casos de uso
│   │   │   ├── infrastructure/ # Postgres, Redis, SMTP, Storage
│   │   │   └── interfaces/     # HTTP handlers, middleware, SSE
│   │   ├── migrations/   # Migrations SQL (goose)
│   │   ├── pkg/          # Utilitários reutilizáveis
│   │   └── seeds/        # Dados de desenvolvimento
│   └── web/              # Frontend SvelteKit
│       └── src/
│           ├── lib/
│           │   ├── api/          # Clientes de API tipados
│           │   ├── components/   # Componentes reutilizáveis
│           │   ├── i18n/         # Arquivos de tradução
│           │   ├── permissions/  # Sistema de permissões
│           │   └── stores/       # Estado global
│           └── routes/   # Páginas (SvelteKit file-based routing)
├── infra/
│   └── docker/           # Dockerfiles
├── docs/                 # Documentação técnica
└── .github/              # CI/CD e templates
```

---

## Contribuindo

Contribuições são muito bem-vindas! Veja [CONTRIBUTING.md](./CONTRIBUTING.md) para o guia completo.

---

## Licença

NovuDesk é distribuído sob a licença [GNU AGPL v3](./LICENSE).

Isso significa que qualquer versão hospedada publicamente deve ter seu código-fonte disponibilizado. Para uso comercial com outras condições, entre em contato.
