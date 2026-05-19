-- +goose Up
CREATE TABLE categories (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, name)
);

CREATE INDEX idx_categories_org_id ON categories (org_id);

CREATE TABLE team_categories (
    team_id     UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, category_id)
);

ALTER TABLE tickets ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE SET NULL;

CREATE INDEX idx_tickets_category_id ON tickets (category_id);

-- +goose Down
ALTER TABLE tickets DROP COLUMN IF EXISTS category_id;
DROP TABLE IF EXISTS team_categories;
DROP TABLE IF EXISTS categories;
