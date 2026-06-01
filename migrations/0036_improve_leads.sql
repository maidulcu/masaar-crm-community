-- +goose Up
ALTER TABLE leads
    ADD COLUMN IF NOT EXISTS assigned_to       UUID        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS closed_reason     TEXT        DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_contacted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deleted_at        TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_leads_assigned_to    ON leads(assigned_to) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_leads_deleted_at     ON leads(deleted_at);
CREATE INDEX IF NOT EXISTS idx_leads_last_contacted ON leads(last_contacted_at DESC) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_leads_last_contacted;
DROP INDEX IF EXISTS idx_leads_deleted_at;
DROP INDEX IF EXISTS idx_leads_assigned_to;

ALTER TABLE leads
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS last_contacted_at,
    DROP COLUMN IF EXISTS closed_reason,
    DROP COLUMN IF EXISTS assigned_to;
