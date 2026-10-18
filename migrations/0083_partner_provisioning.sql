-- +goose Up

-- A company created by a partner system (e.g. a portal that bundles the CRM with its paid
-- plans) carries the partner's client id and its own reference for the account, unique per
-- partner, so a repeated provisioning call finds the company it already made.
ALTER TABLE companies ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE companies ADD COLUMN IF NOT EXISTS external_ref TEXT;
ALTER TABLE companies ADD CONSTRAINT companies_external_ref_pair
  CHECK ((external_source IS NULL) = (external_ref IS NULL));
CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_external_ref
  ON companies(external_source, external_ref) WHERE external_ref IS NOT NULL;

-- Seats: the most active users a company may have. NULL means no limit.
ALTER TABLE companies ADD COLUMN IF NOT EXISTS max_users INT CHECK (max_users IS NULL OR max_users > 0);

-- Enforce seats on every path (partner, admin UI, invitations) atomically: activating a user
-- locks the company row, so two concurrent additions cannot both take the last seat.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_company_seats() RETURNS trigger AS $$
DECLARE
  seats INT;
  active_count INT;
BEGIN
  IF NOT NEW.is_active OR (TG_OP = 'UPDATE' AND OLD.is_active AND OLD.company_id = NEW.company_id) THEN
    RETURN NEW;
  END IF;
  SELECT max_users INTO seats FROM companies WHERE id = NEW.company_id FOR UPDATE;
  IF seats IS NULL THEN
    RETURN NEW;
  END IF;
  SELECT COUNT(*) INTO active_count FROM users
   WHERE company_id = NEW.company_id AND is_active AND id <> NEW.id;
  IF active_count >= seats THEN
    RAISE EXCEPTION 'seat_limit' USING ERRCODE = 'P0001', HINT = 'company has no free seat';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
CREATE TRIGGER users_enforce_company_seats
  BEFORE INSERT OR UPDATE OF is_active, company_id ON users
  FOR EACH ROW EXECUTE FUNCTION enforce_company_seats();

-- The partner's reference for a user, unique within the company.
ALTER TABLE users ADD COLUMN IF NOT EXISTS external_ref TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_company_external_ref
  ON users(company_id, external_ref) WHERE external_ref IS NOT NULL;

-- Access tokens issued to partner clients (OAuth 2.0 client credentials). Partner clients are
-- configured per deployment (PARTNER_CLIENTS), not owned by a company, so this platform table
-- has no company_id. Only the SHA-256 of a token is stored.
CREATE TABLE IF NOT EXISTS partner_tokens (
    token_hash  TEXT        PRIMARY KEY,
    client_id   TEXT        NOT NULL,
    scopes      TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_partner_tokens_expires ON partner_tokens(expires_at);

-- Actions taken by a partner client have no user or API key actor; record the client id.
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS partner_client_id TEXT;

-- +goose Down
-- Only provisioning metadata and short-lived tokens are removed; companies and users stay.
DROP TRIGGER IF EXISTS users_enforce_company_seats ON users;
DROP FUNCTION IF EXISTS enforce_company_seats();
ALTER TABLE audit_logs DROP COLUMN IF EXISTS partner_client_id;
DROP TABLE IF EXISTS partner_tokens;
DROP INDEX IF EXISTS idx_users_company_external_ref;
ALTER TABLE users DROP COLUMN IF EXISTS external_ref;
ALTER TABLE companies DROP COLUMN IF EXISTS max_users;
DROP INDEX IF EXISTS idx_companies_external_ref;
ALTER TABLE companies DROP CONSTRAINT IF EXISTS companies_external_ref_pair;
ALTER TABLE companies DROP COLUMN IF EXISTS external_ref;
ALTER TABLE companies DROP COLUMN IF EXISTS external_source;
