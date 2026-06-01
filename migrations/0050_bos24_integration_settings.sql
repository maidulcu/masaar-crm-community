-- +goose Up

-- Per-company BOS24 integration configuration.
-- Each company (tenant) that connects their BOS24 account stores their
-- integration API key and webhook credentials here.
CREATE TABLE IF NOT EXISTS bos24_integration_settings (
    id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id       UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    api_key          TEXT        NOT NULL DEFAULT '',
    webhook_secret   TEXT        NOT NULL DEFAULT '',
    webhook_id       TEXT        NOT NULL DEFAULT '',
    last_sync_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT bos24_settings_company_unique UNIQUE (company_id)
);

CREATE INDEX IF NOT EXISTS idx_bos24_settings_company ON bos24_integration_settings(company_id);
-- Index on webhook_secret for fast lookup during inbound webhook routing
CREATE INDEX IF NOT EXISTS idx_bos24_settings_secret  ON bos24_integration_settings(webhook_secret)
  WHERE webhook_secret <> '';

-- +goose Down
DROP TABLE IF EXISTS bos24_integration_settings;
