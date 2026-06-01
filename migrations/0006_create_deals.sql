-- +goose Up
CREATE TABLE deals (
    id          UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    lead_id     UUID          NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    title       TEXT          NOT NULL,
    stage       TEXT          NOT NULL DEFAULT 'open' CHECK (stage IN ('open','won','lost')),
    amount      NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency    CHAR(3)       NOT NULL DEFAULT 'AED',
    close_date  DATE,
    probability SMALLINT      NOT NULL DEFAULT 50 CHECK (probability BETWEEN 0 AND 100),
    owner_id    UUID          NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deals_lead    ON deals(lead_id);
CREATE INDEX idx_deals_owner   ON deals(owner_id);
CREATE INDEX idx_deals_stage   ON deals(stage);

-- +goose Down
DROP TABLE IF EXISTS deals;
