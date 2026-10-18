-- +goose Up

-- The BOS24 marketplace integration is retired (it was a stub in this edition). Leads and
-- listings it imported keep their marketplace id as the generic external_source/external_ref
-- pair that every pushed lead uses (0079); listings get the same pair for portal publishing.

-- Leads: bos24_inquiry_id → external_source 'bos24' + external_ref.
UPDATE leads
   SET external_source = 'bos24', external_ref = bos24_inquiry_id::text
 WHERE bos24_inquiry_id IS NOT NULL AND external_ref IS NULL;
DROP INDEX IF EXISTS idx_leads_bos24_inquiry;
ALTER TABLE leads DROP COLUMN IF EXISTS bos24_inquiry_id;

-- Listings: the same pair, unique per company.
ALTER TABLE listings ADD COLUMN IF NOT EXISTS external_source TEXT;
ALTER TABLE listings ADD COLUMN IF NOT EXISTS external_ref TEXT;
ALTER TABLE listings ADD CONSTRAINT listings_external_ref_pair
  CHECK ((external_source IS NULL) = (external_ref IS NULL));
CREATE UNIQUE INDEX IF NOT EXISTS idx_listings_company_external_ref
  ON listings(company_id, external_source, external_ref)
  WHERE external_ref IS NOT NULL;
UPDATE listings
   SET external_source = 'bos24', external_ref = bos24_listing_uuid
 WHERE bos24_listing_uuid IS NOT NULL AND bos24_listing_uuid <> '';
DROP INDEX IF EXISTS idx_listings_bos24_uuid;
ALTER TABLE listings DROP COLUMN IF EXISTS bos24_listing_uuid;

-- Integration credentials: nothing reads them any more.
DROP TABLE IF EXISTS bos24_integration_settings;

-- +goose Down
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
CREATE INDEX IF NOT EXISTS idx_bos24_settings_secret  ON bos24_integration_settings(webhook_secret)
  WHERE webhook_secret <> '';

ALTER TABLE listings ADD COLUMN IF NOT EXISTS bos24_listing_uuid TEXT;
UPDATE listings SET bos24_listing_uuid = external_ref WHERE external_source = 'bos24';
CREATE UNIQUE INDEX IF NOT EXISTS idx_listings_bos24_uuid
  ON listings(company_id, bos24_listing_uuid)
  WHERE bos24_listing_uuid IS NOT NULL;
DROP INDEX IF EXISTS idx_listings_company_external_ref;
ALTER TABLE listings DROP CONSTRAINT IF EXISTS listings_external_ref_pair;
ALTER TABLE listings DROP COLUMN IF EXISTS external_ref;
ALTER TABLE listings DROP COLUMN IF EXISTS external_source;

ALTER TABLE leads ADD COLUMN IF NOT EXISTS bos24_inquiry_id INT;
UPDATE leads SET bos24_inquiry_id = external_ref::int
 WHERE external_source = 'bos24' AND external_ref ~ '^[0-9]{1,9}$';
CREATE UNIQUE INDEX IF NOT EXISTS idx_leads_bos24_inquiry
  ON leads(bos24_inquiry_id)
  WHERE bos24_inquiry_id IS NOT NULL;
