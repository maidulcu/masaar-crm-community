-- +goose Up
-- Serves the Kanban board (newest N leads per stage) and "load more" keyset paging
-- (created_at, id) within a stage for one company.
CREATE INDEX IF NOT EXISTS idx_leads_company_stage_created
    ON leads (company_id, stage, created_at DESC, id DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_leads_company_stage_created;
