-- +goose Up
CREATE TABLE lead_tags (
  id BIGSERIAL PRIMARY KEY,
  lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
  tag VARCHAR(100) NOT NULL,
  category VARCHAR(50), -- segment, quality, interest, timeline, status
  auto_applied BOOLEAN DEFAULT false,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_lead_tag_unique ON lead_tags(lead_id, tag);
CREATE INDEX idx_lead_tags_by_lead ON lead_tags(lead_id);
CREATE INDEX idx_lead_tags_by_category ON lead_tags(category);

-- Add lead_score column to leads table for automatic scoring
ALTER TABLE leads ADD COLUMN lead_score INT DEFAULT 0;
ALTER TABLE leads ADD COLUMN score_updated_at TIMESTAMP;

-- +goose Down
ALTER TABLE leads DROP COLUMN IF EXISTS lead_score;
ALTER TABLE leads DROP COLUMN IF EXISTS score_updated_at;
DROP TABLE IF EXISTS lead_tags;
