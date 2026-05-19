-- +goose Up
CREATE TABLE ticket_comments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID        NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    author_id   UUID        REFERENCES users(id) ON DELETE SET NULL,
    body        TEXT        NOT NULL,
    is_internal BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_comments_ticket_id ON ticket_comments (ticket_id);
CREATE INDEX idx_comments_org_id    ON ticket_comments (org_id);

CREATE TABLE ticket_attachments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id   UUID        NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    comment_id  UUID        REFERENCES ticket_comments(id) ON DELETE SET NULL,
    uploader_id UUID        REFERENCES users(id) ON DELETE SET NULL,
    filename    TEXT        NOT NULL,
    mime_type   TEXT        NOT NULL,
    size_bytes  BIGINT      NOT NULL,
    storage_key TEXT        NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_attachments_ticket_id ON ticket_attachments (ticket_id);

-- +goose Down
DROP TABLE IF EXISTS ticket_attachments;
DROP TABLE IF EXISTS ticket_comments;
