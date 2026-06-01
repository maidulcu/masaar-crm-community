-- +goose Up
CREATE TABLE pipeline_stages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  entity_type VARCHAR(20) NOT NULL DEFAULT 'lead',
  name VARCHAR(100) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  color VARCHAR(50) NOT NULL DEFAULT '#6366f1',
  is_won BOOLEAN NOT NULL DEFAULT FALSE,
  is_lost BOOLEAN NOT NULL DEFAULT FALSE,
  is_default BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(company_id, entity_type, name)
);

CREATE INDEX idx_pipeline_stages_company ON pipeline_stages(company_id, entity_type, sort_order);

-- Drop old CHECK constraint — stages are now dynamic
ALTER TABLE leads DROP CONSTRAINT IF EXISTS leads_stage_check;

-- Seed default lead stages for existing companies
INSERT INTO pipeline_stages (company_id, entity_type, name, sort_order, color, is_default)
SELECT c.id, 'lead', s.name, s.sort_order, s.color, TRUE
FROM companies c
CROSS JOIN (
  VALUES
    ('new', 0, '#a3a3a3'),
    ('contacted', 1, '#0ea5e9'),
    ('qualified', 2, '#6366f1'),
    ('proposal', 3, '#f59e0b'),
    ('won', 4, '#10b981'),
    ('lost', 5, '#ef4444')
) AS s(name, sort_order, color)
ON CONFLICT (company_id, entity_type, name) DO NOTHING;

-- Mark won/lost stages
UPDATE pipeline_stages SET is_won = TRUE WHERE name = 'won';
UPDATE pipeline_stages SET is_lost = TRUE WHERE name = 'lost';

-- +goose Down
DROP TABLE pipeline_stages;
ALTER TABLE leads ADD CONSTRAINT leads_stage_check CHECK (stage IN ('new','contacted','qualified','proposal','won','lost'));
