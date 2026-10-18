-- +goose Up
-- Two-factor authentication with an authenticator app (#87).

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS totp_secret     TEXT        NOT NULL DEFAULT '', -- sealed with TWO_FACTOR_KEY
    ADD COLUMN IF NOT EXISTS totp_enabled    BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS totp_enabled_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS totp_last_step  BIGINT      NOT NULL DEFAULT 0;      -- last accepted time step (no replay)

-- Single-use backup codes, stored as hashes.
CREATE TABLE user_backup_codes (
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, code_hash)
);

-- Admins can require two-factor authentication for every user of the company.
ALTER TABLE company_settings ADD COLUMN IF NOT EXISTS require_2fa BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE company_settings DROP COLUMN IF EXISTS require_2fa;
DROP TABLE IF EXISTS user_backup_codes;
ALTER TABLE users
    DROP COLUMN IF EXISTS totp_last_step,
    DROP COLUMN IF EXISTS totp_enabled_at,
    DROP COLUMN IF EXISTS totp_enabled,
    DROP COLUMN IF EXISTS totp_secret;
