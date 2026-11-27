-- +goose Up
-- Signing links can expire and be cancelled; a cancelled link keeps signature_status 'cancelled'.
ALTER TABLE document_signatures ADD COLUMN IF NOT EXISTS expires_at   TIMESTAMPTZ;
ALTER TABLE document_signatures ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;

-- API keys may be given an expiry; NULL keeps a key valid until it is revoked.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

-- Which client secret a partner token was issued under: rotating the secret ends older tokens.
-- NULL for tokens issued before this migration (they expire within the hour).
ALTER TABLE partner_tokens ADD COLUMN IF NOT EXISTS secret_fingerprint TEXT;

-- +goose Down
-- Only expiry metadata is removed. Cancelled signatures keep signature_status 'cancelled', so
-- they stay unsignable; API keys and signatures themselves are untouched.
ALTER TABLE partner_tokens DROP COLUMN IF EXISTS secret_fingerprint;
ALTER TABLE api_keys DROP COLUMN IF EXISTS expires_at;
ALTER TABLE document_signatures DROP COLUMN IF EXISTS cancelled_at;
ALTER TABLE document_signatures DROP COLUMN IF EXISTS expires_at;
