-- +goose Up
-- PDPL: per-channel marketing/contact consent history, and erasure of a contact.

CREATE TABLE contact_consents (
    id          BIGSERIAL   PRIMARY KEY,
    company_id  UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    contact_id  UUID        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    channel     TEXT        NOT NULL CHECK (channel IN ('whatsapp','email','sms','phone')),
    status      TEXT        NOT NULL CHECK (status IN ('granted','withdrawn')),
    source      TEXT        NOT NULL CHECK (source IN ('manual','public_form','import','api','inbound_message','erasure')),
    captured_by UUID        REFERENCES users(id) ON DELETE SET NULL,
    api_key_id  UUID,
    note        TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- The current status of a channel is its most recent row.
CREATE INDEX idx_contact_consents_current ON contact_consents(contact_id, channel, created_at DESC, id DESC);
CREATE INDEX idx_contact_consents_company ON contact_consents(company_id);

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS erased_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE contacts DROP COLUMN IF EXISTS erased_at;
DROP TABLE IF EXISTS contact_consents;
