-- +goose Up

-- Portal webhooks now de-duplicate on leads.external_source/external_ref (0079), the
-- same as every other pushed lead. Carry existing portal lead ids over, then drop the
-- separate claim table.
UPDATE leads l
   SET external_source = p.portal, external_ref = p.external_id
  FROM portal_leads p
 WHERE p.lead_id = l.id
   AND p.company_id = l.company_id
   AND l.external_ref IS NULL;

DROP TABLE IF EXISTS portal_leads;

-- +goose Down
CREATE TABLE portal_leads (
    company_id  UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    portal      TEXT        NOT NULL,
    external_id TEXT        NOT NULL,
    lead_id     UUID        REFERENCES leads(id) ON DELETE SET NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, portal, external_id)
);

INSERT INTO portal_leads (company_id, portal, external_id, lead_id, received_at)
SELECT company_id, external_source, external_ref, id, created_at
  FROM leads
 WHERE external_source IN ('property_finder','bayut','dubizzle','portal')
   AND external_ref IS NOT NULL
ON CONFLICT DO NOTHING;
