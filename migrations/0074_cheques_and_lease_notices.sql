-- +goose Up
-- Property management (#86): post-dated cheque register, lease alerts and notices, rent
-- reminders.

-- The agent who looks after a lease (alerts, bounced cheques). Defaults to its creator.
ALTER TABLE leases
    ADD COLUMN IF NOT EXISTS agent_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS ejari_expiry DATE;
UPDATE leases SET agent_id = created_by WHERE agent_id IS NULL;

-- Post-dated cheques. A cheque usually pays one scheduled payment.
CREATE TABLE cheques (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id     UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    lease_id       UUID          NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    payment_id     UUID          REFERENCES payments(id) ON DELETE SET NULL,
    cheque_number  TEXT          NOT NULL,
    bank_name      TEXT          NOT NULL DEFAULT '',
    drawer_name    TEXT          NOT NULL DEFAULT '',
    amount         NUMERIC(15,3) NOT NULL CHECK (amount > 0),
    currency       CHAR(3)       NOT NULL DEFAULT 'AED',
    cheque_date    DATE          NOT NULL,
    status         TEXT          NOT NULL DEFAULT 'received'
                   CHECK (status IN ('received','deposited','cleared','bounced','replaced','returned','on_hold')),
    scan_key       TEXT          NOT NULL DEFAULT '', -- file in the bank statement store
    deposited_at   DATE,
    cleared_at     DATE,
    bounced_at     DATE,
    bounce_reason  TEXT          NOT NULL DEFAULT '',
    replaced_by    UUID          REFERENCES cheques(id) ON DELETE SET NULL,
    notes          TEXT          NOT NULL DEFAULT '',
    created_by     UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_cheques_lease ON cheques(lease_id, cheque_date);
CREATE INDEX idx_cheques_company_status_date ON cheques(company_id, status, cheque_date);

-- Notices served on a tenant (rent change, non-renewal…), with when and how they were sent.
CREATE TABLE lease_notices (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id     UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    lease_id       UUID          NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    kind           TEXT          NOT NULL CHECK (kind IN ('rent_change','non_renewal','renewal_terms','other')),
    current_rent   NUMERIC(15,3),
    new_rent       NUMERIC(15,3),
    effective_date DATE,
    body_en        TEXT          NOT NULL DEFAULT '',
    body_ar        TEXT          NOT NULL DEFAULT '',
    sent_at        TIMESTAMPTZ,
    sent_via       TEXT          NOT NULL DEFAULT ''
                   CHECK (sent_via IN ('','email','whatsapp','hand','registered_mail','notary','other')),
    created_by     UUID          REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_lease_notices_lease ON lease_notices(lease_id, created_at DESC);

-- Rent reminders. The repository for this table existed without a migration.
CREATE TABLE IF NOT EXISTS payment_reminders (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    payment_id      UUID        NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    reminder_type   TEXT        NOT NULL,
    reminder_date   DATE        NOT NULL,
    sent_at         TIMESTAMPTZ,
    delivery_method TEXT        NOT NULL CHECK (delivery_method IN ('email','whatsapp','sms')),
    delivery_status TEXT        NOT NULL DEFAULT 'pending' CHECK (delivery_status IN ('pending','sent','failed','skipped')),
    delivery_error  TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (payment_id, reminder_type, delivery_method)
);

-- A tenant can be excluded from rent reminders.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS rent_reminders BOOLEAN NOT NULL DEFAULT TRUE;

-- Company settings. Notice periods differ by emirate and change: these are defaults to check.
ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS lease_alert_days          INTEGER[] NOT NULL DEFAULT '{120,90,60,30}',
    ADD COLUMN IF NOT EXISTS rent_notice_days          INTEGER   NOT NULL DEFAULT 90,
    ADD COLUMN IF NOT EXISTS rent_reminders_enabled    BOOLEAN   NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS rent_reminder_days_before INTEGER   NOT NULL DEFAULT 3,
    ADD COLUMN IF NOT EXISTS rent_reminder_wa_template TEXT      NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS rent_reminder_wa_language TEXT      NOT NULL DEFAULT 'en';

-- +goose Down
ALTER TABLE company_settings
    DROP COLUMN IF EXISTS rent_reminder_wa_language,
    DROP COLUMN IF EXISTS rent_reminder_wa_template,
    DROP COLUMN IF EXISTS rent_reminder_days_before,
    DROP COLUMN IF EXISTS rent_reminders_enabled,
    DROP COLUMN IF EXISTS rent_notice_days,
    DROP COLUMN IF EXISTS lease_alert_days;
ALTER TABLE tenants DROP COLUMN IF EXISTS rent_reminders;
DROP TABLE IF EXISTS payment_reminders;
DROP TABLE IF EXISTS lease_notices;
DROP TABLE IF EXISTS cheques;
ALTER TABLE leases DROP COLUMN IF EXISTS ejari_expiry, DROP COLUMN IF EXISTS agent_id;
