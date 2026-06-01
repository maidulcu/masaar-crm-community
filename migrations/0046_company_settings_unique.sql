-- +goose Up
-- Enforce one company_settings row per company.
-- The backfill in 0045 guarantees at most one row per company_id before this runs.
ALTER TABLE company_settings
    ADD CONSTRAINT uq_company_settings_company UNIQUE (company_id);

-- +goose Down
ALTER TABLE company_settings
    DROP CONSTRAINT IF EXISTS uq_company_settings_company;
