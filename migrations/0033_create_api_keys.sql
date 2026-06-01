-- +goose Up
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE, -- SHA256 hash of the full key
    key_prefix TEXT NOT NULL, -- First 8 chars for display (e.g., "sk_live_abcd1234...")
    scopes TEXT NOT NULL DEFAULT '', -- JSON array of scopes (comma-separated for now)
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,

    CONSTRAINT company_key_unique UNIQUE (company_id, name)
);

CREATE INDEX idx_api_keys_company_id ON api_keys(company_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_revoked ON api_keys(revoked_at);

-- +goose Down
DROP TABLE api_keys;
