-- +goose Up

-- Automation rules (schema ready for post-MVP)
CREATE TABLE automation_rules (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name           TEXT        NOT NULL,
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    trigger_event  TEXT        NOT NULL,
    conditions     JSONB       NOT NULL DEFAULT '[]',
    actions        JSONB       NOT NULL DEFAULT '[]',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_automation_rules_org_id        ON automation_rules (org_id);
CREATE INDEX idx_automation_rules_trigger_event ON automation_rules (org_id, trigger_event) WHERE is_active = TRUE;

CREATE TABLE automation_logs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id      UUID        NOT NULL REFERENCES automation_rules(id) ON DELETE CASCADE,
    ticket_id    UUID        REFERENCES tickets(id) ON DELETE SET NULL,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    result       TEXT        NOT NULL DEFAULT 'success',
    error_message TEXT
);

CREATE INDEX idx_automation_logs_rule_id ON automation_logs (rule_id);

-- Webhooks (schema ready for post-MVP)
CREATE TABLE webhooks (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    url         TEXT        NOT NULL,
    secret_hash TEXT        NOT NULL,
    events      TEXT[]      NOT NULL DEFAULT '{}',
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhooks_org_id ON webhooks (org_id);

CREATE TABLE webhook_deliveries (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id      UUID        NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    event           TEXT        NOT NULL,
    payload         JSONB       NOT NULL,
    status_code     INT,
    attempts        INT         NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    next_retry_at   TIMESTAMPTZ,
    status          TEXT        NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_webhook_id    ON webhook_deliveries (webhook_id);
CREATE INDEX idx_webhook_deliveries_next_retry_at ON webhook_deliveries (next_retry_at) WHERE status = 'pending';

-- +goose Down
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS automation_logs;
DROP TABLE IF EXISTS automation_rules;
