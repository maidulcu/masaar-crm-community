-- +goose Up
ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS logo_url    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS disclaimer  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS primary_color VARCHAR(7) NOT NULL DEFAULT '#1a3a5c';

-- +goose Down
ALTER TABLE company_settings
    DROP COLUMN IF EXISTS primary_color,
    DROP COLUMN IF EXISTS disclaimer,
    DROP COLUMN IF EXISTS logo_url;
