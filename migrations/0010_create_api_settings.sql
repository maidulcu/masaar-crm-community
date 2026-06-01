-- +goose Up
CREATE TABLE api_settings (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    setting_key     TEXT        NOT NULL UNIQUE,
    setting_value   TEXT        NOT NULL,
    description     TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID        REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_api_settings_key ON api_settings(setting_key);

-- Insert initial BuyOrSell24 API token setting (empty by default)
INSERT INTO api_settings (id, setting_key, setting_value, description)
VALUES (uuid_generate_v4(), 'bos24_api_token', '', 'BuyOrSell24 Real Estate API Token (Dynamic Web Lab)');

-- +goose Down
DROP TABLE IF EXISTS api_settings;
