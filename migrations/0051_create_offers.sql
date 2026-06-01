-- +goose Up

CREATE TYPE offer_status AS ENUM (
    'submitted',
    'under_review',
    'countered',
    'accepted',
    'rejected',
    'expired'
);

CREATE TABLE offers (
    id              UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id      UUID            NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    contact_id      UUID            NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    agent_id        UUID            REFERENCES users(id) ON DELETE SET NULL,
    parent_offer_id UUID            REFERENCES offers(id) ON DELETE SET NULL,  -- counter-offer chain

    offer_amount    NUMERIC(14,2)   NOT NULL,
    currency        CHAR(3)         NOT NULL DEFAULT 'AED',
    status          offer_status    NOT NULL DEFAULT 'submitted',
    terms           TEXT            NOT NULL DEFAULT '',
    notes           TEXT            NOT NULL DEFAULT '',
    valid_until     TIMESTAMPTZ,

    -- Auto-conversion: populated when offer is accepted and a deal is created
    deal_id         UUID            REFERENCES deals(id) ON DELETE SET NULL,

    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_offers_listing   ON offers(listing_id);
CREATE INDEX idx_offers_contact   ON offers(contact_id);
CREATE INDEX idx_offers_agent     ON offers(agent_id) WHERE agent_id IS NOT NULL;
CREATE INDEX idx_offers_status    ON offers(status);
CREATE INDEX idx_offers_parent    ON offers(parent_offer_id) WHERE parent_offer_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS offers;
DROP TYPE IF EXISTS offer_status;
