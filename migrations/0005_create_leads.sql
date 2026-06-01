-- +goose Up
CREATE TABLE leads (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    contact_id  UUID        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    stage       TEXT        NOT NULL DEFAULT 'new'
                            CHECK (stage IN ('new','contacted','qualified','proposal','won','lost')),
    source      TEXT        CHECK (source IN ('whatsapp','web','referral','event')),
    deal_value  NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency    CHAR(3)     NOT NULL DEFAULT 'AED',
    notes       TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_leads_contact   ON leads(contact_id);
CREATE INDEX idx_leads_stage     ON leads(stage);
CREATE INDEX idx_leads_created   ON leads(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS leads;
