-- +goose Up

-- Add billing/plan fields to company_settings
ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS plan               VARCHAR(20)  NOT NULL DEFAULT 'community',
    ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS stripe_sub_id      VARCHAR(100),
    ADD COLUMN IF NOT EXISTS plan_started_at    TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS plan_expires_at    TIMESTAMPTZ;

-- Monthly usage counters per company per resource
CREATE TABLE IF NOT EXISTS usage_tracking (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID        NOT NULL,
    resource   VARCHAR(50) NOT NULL,   -- bos24 | ai | pdf
    period     VARCHAR(7)  NOT NULL,   -- YYYY-MM
    count      INT         NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (company_id, resource, period)
);

CREATE INDEX IF NOT EXISTS idx_usage_company_period ON usage_tracking (company_id, period);

-- +goose Down
ALTER TABLE company_settings
    DROP COLUMN IF EXISTS plan,
    DROP COLUMN IF EXISTS stripe_customer_id,
    DROP COLUMN IF EXISTS stripe_sub_id,
    DROP COLUMN IF EXISTS plan_started_at,
    DROP COLUMN IF EXISTS plan_expires_at;

DROP TABLE IF EXISTS usage_tracking;
