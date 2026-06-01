-- +goose Up

-- Track which BOS24 marketplace listings we've imported into the listings table.
-- Unique per (company_id, bos24_listing_uuid) so different companies can import
-- the same BOS24 listing independently.
ALTER TABLE listings ADD COLUMN IF NOT EXISTS bos24_listing_uuid TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_listings_bos24_uuid
  ON listings(company_id, bos24_listing_uuid)
  WHERE bos24_listing_uuid IS NOT NULL;

-- Track which BOS24 inquiry IDs we've imported as leads to prevent duplicate
-- lead creation when the same inquiry arrives via both webhook and nightly poll.
ALTER TABLE leads ADD COLUMN IF NOT EXISTS bos24_inquiry_id INT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_leads_bos24_inquiry
  ON leads(bos24_inquiry_id)
  WHERE bos24_inquiry_id IS NOT NULL;

-- Extend the leads source check constraint to include 'bos24'.
-- Postgres requires dropping and re-adding CHECK constraints.
ALTER TABLE leads DROP CONSTRAINT IF EXISTS leads_source_check;
ALTER TABLE leads ADD CONSTRAINT leads_source_check
  CHECK (source IN ('whatsapp','web','referral','event','bos24'));

-- +goose Down
ALTER TABLE leads DROP CONSTRAINT IF EXISTS leads_source_check;
ALTER TABLE leads ADD CONSTRAINT leads_source_check
  CHECK (source IN ('whatsapp','web','referral','event'));

DROP INDEX IF EXISTS idx_leads_bos24_inquiry;
ALTER TABLE leads DROP COLUMN IF EXISTS bos24_inquiry_id;

DROP INDEX IF EXISTS idx_listings_bos24_uuid;
ALTER TABLE listings DROP COLUMN IF EXISTS bos24_listing_uuid;
