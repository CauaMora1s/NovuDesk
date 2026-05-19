-- +goose Up
CREATE TYPE ticket_status   AS ENUM ('open', 'pending', 'on_hold', 'resolved', 'closed');
CREATE TYPE ticket_priority AS ENUM ('low', 'normal', 'high', 'urgent');

-- Per-org auto-incrementing ticket number
CREATE SEQUENCE ticket_number_seq START 1;

CREATE TABLE tickets (
    id                    UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                UUID            NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    number                BIGINT          NOT NULL,
    title                 TEXT            NOT NULL,
    description           TEXT,
    status                ticket_status   NOT NULL DEFAULT 'open',
    priority              ticket_priority NOT NULL DEFAULT 'normal',
    assignee_id           UUID            REFERENCES users(id) ON DELETE SET NULL,
    team_id               UUID            REFERENCES teams(id) ON DELETE SET NULL,
    requester_id          UUID            REFERENCES users(id) ON DELETE SET NULL,
    sla_policy_id         UUID            REFERENCES sla_policies(id) ON DELETE SET NULL,
    sla_response_due_at   TIMESTAMPTZ,
    sla_resolution_due_at TIMESTAMPTZ,
    sla_breached          BOOLEAN         NOT NULL DEFAULT FALSE,
    tags                  TEXT[]          NOT NULL DEFAULT '{}',
    custom_fields         JSONB           NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    resolved_at           TIMESTAMPTZ,
    closed_at             TIMESTAMPTZ,
    UNIQUE (org_id, number)
);

-- Core query patterns
CREATE INDEX idx_tickets_org_id      ON tickets (org_id);
CREATE INDEX idx_tickets_status      ON tickets (org_id, status);
CREATE INDEX idx_tickets_assignee    ON tickets (org_id, assignee_id);
CREATE INDEX idx_tickets_team        ON tickets (org_id, team_id);
CREATE INDEX idx_tickets_created_at  ON tickets (org_id, created_at DESC);
CREATE INDEX idx_tickets_sla_breach  ON tickets (org_id, sla_resolution_due_at) WHERE sla_breached = FALSE;

-- GIN index for JSONB custom_fields and tags array
CREATE INDEX idx_tickets_custom_fields ON tickets USING GIN (custom_fields);
CREATE INDEX idx_tickets_tags          ON tickets USING GIN (tags);

-- Full-text search on title + description
CREATE INDEX idx_tickets_fts ON tickets USING GIN (
    to_tsvector('portuguese', coalesce(title, '') || ' ' || coalesce(description, ''))
);

-- +goose Down
DROP TABLE IF EXISTS tickets;
DROP TYPE  IF EXISTS ticket_priority;
DROP TYPE  IF EXISTS ticket_status;
DROP SEQUENCE IF EXISTS ticket_number_seq;
