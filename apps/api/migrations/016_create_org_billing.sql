-- +goose Up

-- Billing state lives 1:1 on the organization (no JOIN needed).
ALTER TABLE organizations
    ADD COLUMN plan_renews_at        TIMESTAMPTZ,
    ADD COLUMN billing_status        TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN billing_provider      TEXT NOT NULL DEFAULT 'manual',
    ADD COLUMN billing_customer_ref  TEXT,
    ADD COLUMN payment_method_brand  TEXT,
    ADD COLUMN payment_method_last4  TEXT;

-- History of every plan-change attempt. The organization's active plan
-- (plan_tier / plan_renews_at) only changes when a session reaches 'completed'.
CREATE TABLE payment_sessions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    from_tier       TEXT,
    to_tier         TEXT        NOT NULL,
    status          TEXT        NOT NULL DEFAULT 'pending', -- pending|completed|cancelled|failed|expired
    amount_cents    BIGINT      NOT NULL DEFAULT 0,
    proration_cents BIGINT      NOT NULL DEFAULT 0,
    currency        TEXT        NOT NULL DEFAULT 'BRL',
    provider        TEXT        NOT NULL DEFAULT 'manual',
    provider_ref    TEXT,
    created_by      UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ
);

CREATE INDEX idx_payment_sessions_org ON payment_sessions (org_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS payment_sessions;

ALTER TABLE organizations
    DROP COLUMN IF EXISTS plan_renews_at,
    DROP COLUMN IF EXISTS billing_status,
    DROP COLUMN IF EXISTS billing_provider,
    DROP COLUMN IF EXISTS billing_customer_ref,
    DROP COLUMN IF EXISTS payment_method_brand,
    DROP COLUMN IF EXISTS payment_method_last4;
