-- +goose Up
-- (Numbered 0065 when first merged, which collided with 0065_locale_and_jobs; renumbered so
-- it runs after the M0–M5 migrations. Written to be safe to re-run.)

-- Leads pushed by another system (a listing portal, a website, an automation) carry that
-- system's name and its own id for the lead. Unique per company, so a retried push finds
-- the lead it already made instead of creating a second one, and two companies can each
-- hold a lead with the same external id.
ALTER TABLE leads ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE leads ADD COLUMN IF NOT EXISTS external_ref TEXT;
ALTER TABLE leads DROP CONSTRAINT IF EXISTS leads_external_ref_pair;
ALTER TABLE leads ADD CONSTRAINT leads_external_ref_pair
  CHECK ((external_source IS NULL) = (external_ref IS NULL));
CREATE UNIQUE INDEX IF NOT EXISTS idx_leads_company_external_ref
  ON leads(company_id, external_source, external_ref)
  WHERE external_ref IS NOT NULL;

-- 'portal': a lead from a listing portal or marketplace; external_source names which one.
-- Keeps the named portals added in 0073.
ALTER TABLE leads DROP CONSTRAINT IF EXISTS leads_source_check;
ALTER TABLE leads ADD CONSTRAINT leads_source_check
  CHECK (source IN ('whatsapp','web','referral','event','bos24','property_finder','bayut','dubizzle','portal'));

-- Actions taken with an API key have no user actor; record the key instead.
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS api_key_id UUID
  REFERENCES api_keys(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE audit_logs DROP COLUMN IF EXISTS api_key_id;

-- 'portal' stays: 0073 allows it too.

DROP INDEX IF EXISTS idx_leads_company_external_ref;
ALTER TABLE leads DROP CONSTRAINT IF EXISTS leads_external_ref_pair;
ALTER TABLE leads DROP COLUMN IF EXISTS external_ref;
ALTER TABLE leads DROP COLUMN IF EXISTS external_source;
