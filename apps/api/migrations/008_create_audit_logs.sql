-- +goose Up
CREATE TABLE audit_logs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    actor_id      UUID,
    actor_type    TEXT        NOT NULL DEFAULT 'user', -- user | system | automation
    resource_type TEXT        NOT NULL,
    resource_id   UUID        NOT NULL,
    action        TEXT        NOT NULL,
    before        JSONB,
    after         JSONB,
    metadata      JSONB       NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_org_id      ON audit_logs (org_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource    ON audit_logs (org_id, resource_type, resource_id);
CREATE INDEX idx_audit_logs_actor       ON audit_logs (org_id, actor_id);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
