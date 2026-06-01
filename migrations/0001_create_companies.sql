-- +goose Up
CREATE TABLE companies (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    vat_number TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Default company for single-tenant installations.
-- Set APP_COMPANY_ID in .env to this UUID or your own.
INSERT INTO companies (id, name) VALUES ('00000000-0000-0000-0000-000000000001', 'My Company');

-- +goose Down
DROP TABLE IF EXISTS companies;
