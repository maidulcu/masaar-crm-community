-- +goose Up
-- Property portals: inbound leads (Property Finder, Bayut, dubizzle, others) and an
-- outbound listing feed, each authenticated by a per-company secret token.

ALTER TABLE leads DROP CONSTRAINT IF EXISTS leads_source_check;
ALTER TABLE leads ADD CONSTRAINT leads_source_check
    CHECK (source IN ('whatsapp','web','referral','event','bos24','property_finder','bayut','dubizzle','portal'));

CREATE TABLE portal_connections (
    company_id         UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    portal             TEXT        NOT NULL CHECK (portal IN ('property_finder','bayut','dubizzle','portal')),
    inbound_token_hash TEXT        NOT NULL UNIQUE, -- sha256 hex of the webhook token
    feed_token_hash    TEXT        NOT NULL UNIQUE, -- sha256 hex of the feed token
    enabled            BOOLEAN     NOT NULL DEFAULT TRUE,
    last_lead_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, portal)
);

-- One row per received portal lead, so a redelivered webhook creates nothing new.
CREATE TABLE portal_leads (
    company_id  UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    portal      TEXT        NOT NULL,
    external_id TEXT        NOT NULL,
    lead_id     UUID        REFERENCES leads(id) ON DELETE SET NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, portal, external_id)
);

-- +goose Down
DROP TABLE IF EXISTS portal_leads;
DROP TABLE IF EXISTS portal_connections;
UPDATE leads SET source = 'web' WHERE source IN ('property_finder','bayut','dubizzle','portal');
ALTER TABLE leads DROP CONSTRAINT IF EXISTS leads_source_check;
ALTER TABLE leads ADD CONSTRAINT leads_source_check
    CHECK (source IN ('whatsapp','web','referral','event','bos24'));
