-- +goose Up
CREATE TABLE webhook_subscriptions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  UUID NOT NULL REFERENCES companies(id),
    name        TEXT NOT NULL,
    url         TEXT NOT NULL,
    -- Comma-separated event names: lead.created,lead.stage_changed,payment.received
    events      TEXT NOT NULL DEFAULT '',
    secret      TEXT NOT NULL, -- HMAC signing secret (stored in plaintext for signing)
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_fired_at   TIMESTAMPTZ,
    failure_count   INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_webhook_subs_company ON webhook_subscriptions(company_id);
CREATE INDEX idx_webhook_subs_active  ON webhook_subscriptions(company_id, active);

CREATE TABLE webhook_deliveries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    event           TEXT NOT NULL,
    payload         JSONB,
    status          TEXT NOT NULL DEFAULT 'pending', -- pending|success|failed
    response_code   INT,
    attempts        INT NOT NULL DEFAULT 0,
    delivered_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_sub ON webhook_deliveries(subscription_id);
CREATE INDEX idx_webhook_deliveries_status ON webhook_deliveries(status);

-- +goose Down
DROP TABLE webhook_deliveries;
DROP TABLE webhook_subscriptions;
