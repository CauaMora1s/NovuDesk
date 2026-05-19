-- +goose Up
CREATE TYPE custom_field_type AS ENUM ('text', 'number', 'boolean', 'select', 'multi_select', 'date');

CREATE TABLE custom_field_definitions (
    id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID              NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name          TEXT              NOT NULL,
    key           TEXT              NOT NULL,
    field_type    custom_field_type NOT NULL,
    options       JSONB             NOT NULL DEFAULT '[]',
    is_required   BOOLEAN           NOT NULL DEFAULT FALSE,
    is_active     BOOLEAN           NOT NULL DEFAULT TRUE,
    display_order INT               NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, key)
);

CREATE INDEX idx_custom_field_defs_org_id ON custom_field_definitions (org_id);

-- +goose Down
DROP TABLE IF EXISTS custom_field_definitions;
DROP TYPE  IF EXISTS custom_field_type;
