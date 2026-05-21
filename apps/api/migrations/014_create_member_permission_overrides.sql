-- +goose Up

-- Per-member permission overrides: allow granting or denying individual permissions
-- beyond what the member's role provides.
CREATE TABLE member_permission_overrides (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id     UUID        NOT NULL REFERENCES organization_members(id) ON DELETE CASCADE,
    permission_id UUID        NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    is_granted    BOOLEAN     NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (member_id, permission_id)
);

CREATE INDEX idx_member_perm_overrides_member ON member_permission_overrides (member_id);

-- +goose Down
DROP TABLE IF EXISTS member_permission_overrides;
