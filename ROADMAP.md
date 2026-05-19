# Roadmap

## v1.0 — MVP (em desenvolvimento)

Objetivo: um helpdesk completo e funcional, pronto para uso em produção.

### Backend
- [x] Autenticação JWT com refresh tokens
- [x] Multi-tenancy por linha (row-level isolation)
- [x] Organizações e membros
- [x] Sistema de papéis e permissões granulares
- [x] Times
- [x] Tickets (CRUD, status, prioridade, tags, campos customizados)
- [x] Comentários públicos e internos
- [x] Políticas de SLA com cálculo de prazo
- [x] Registro de auditoria com diff antes/depois
- [x] Notificações por e-mail (SMTP)
- [x] Uploads com suporte a LocalFS e S3
- [x] Realtime via SSE com Redis pub/sub
- [x] Sistema de convites por e-mail
- [x] API versionada `/api/v1/`
- [ ] API keys para acesso público
- [ ] Rate limiting por IP e por tenant
- [ ] Endpoint de saúde com checagem de dependências

### Frontend
- [x] Autenticação (login, logout, refresh)
- [x] Sistema de permissões no cliente
- [x] Internacionalização (PT e EN)
- [x] Dashboard com métricas básicas
- [x] Lista e detalhe de tickets
- [x] Criação de ticket
- [x] Troca de status e prioridade
- [x] Comentários públicos e internos
- [x] Modo escuro/claro
- [ ] Página de configurações (organização, membros, papéis, SLA)
- [ ] Página de times
- [ ] Fluxo de aceite de convite
- [ ] Página de perfil do usuário
- [ ] Filtros avançados de tickets

---

## v1.1 — Automações e Webhooks

- [ ] Motor de regras de automação
  - Auto-atribuição de tickets
  - Troca automática de status
  - Escalonamento de SLA
  - Envio de notificações por regra
- [ ] Webhooks com retry e log de entregas
- [ ] Templates de resposta (macros)
- [ ] Campos customizados na UI

---

## v1.2 — API Pública e Integrações

- [ ] API keys com escopos
- [ ] Documentação OpenAPI/Swagger
- [ ] Portal do cliente (envio externo de tickets)
- [ ] Integração com e-mail de entrada (parse de e-mails em tickets)

---

## v2.0 — Enterprise

- [ ] SSO (Google OAuth, Microsoft, SAML, OIDC)
- [ ] Modo white-label (logo, cores, domínio customizados)
- [ ] Feature flags por plano
- [ ] Relatórios avançados e exportação
- [ ] Faturamento e assinaturas
- [ ] Sistema de plugins/módulos
- [ ] Suporte a múltiplos idiomas (contribuição da comunidade)
