-- +goose Up
CREATE TABLE api_keys (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name         TEXT        NOT NULL,
    key_prefix   TEXT        NOT NULL,
    key_hash     TEXT        NOT NULL UNIQUE,
    scopes       TEXT[]      NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_by   UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_org_id   ON api_keys (org_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys (key_hash);

CREATE TABLE feature_flags (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID        REFERENCES organizations(id) ON DELETE CASCADE,
    flag_key   TEXT        NOT NULL,
    enabled    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, flag_key)
);

CREATE INDEX idx_feature_flags_org_id ON feature_flags (org_id);

-- +goose Down
DROP TABLE IF EXISTS feature_flags;
DROP TABLE IF EXISTS api_keys;
