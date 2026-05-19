-- +goose Up
CREATE TABLE sla_policies (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name             TEXT        NOT NULL,
    response_hours   INT         NOT NULL DEFAULT 4,
    resolution_hours INT         NOT NULL DEFAULT 24,
    conditions       JSONB       NOT NULL DEFAULT '{}',
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sla_policies_org_id ON sla_policies (org_id);

-- +goose Down
DROP TABLE IF EXISTS sla_policies;
