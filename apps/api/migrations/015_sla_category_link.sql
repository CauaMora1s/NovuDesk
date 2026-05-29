-- +goose Up
ALTER TABLE sla_policies
  ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
  ADD COLUMN resolution_value INT NOT NULL DEFAULT 24,
  ADD COLUMN resolution_unit  TEXT NOT NULL DEFAULT 'hours';

UPDATE sla_policies SET resolution_value = resolution_hours, resolution_unit = 'hours';

CREATE UNIQUE INDEX idx_sla_policies_category
  ON sla_policies(org_id, category_id) WHERE category_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_sla_policies_category;
ALTER TABLE sla_policies DROP COLUMN IF EXISTS resolution_unit;
ALTER TABLE sla_policies DROP COLUMN IF EXISTS resolution_value;
ALTER TABLE sla_policies DROP COLUMN IF EXISTS category_id;
