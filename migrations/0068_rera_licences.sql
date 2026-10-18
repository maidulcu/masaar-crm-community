-- +goose Up
-- Brokerage licences and listing advertising permits (Dubai Trakheesi, Abu Dhabi
-- Madhmoun), plus a record of expiry alerts so each one is sent once.

ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS orn                  VARCHAR(20) NOT NULL DEFAULT '', -- RERA office registration number
    ADD COLUMN IF NOT EXISTS trade_license_no     VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS trade_license_expiry DATE;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS brn        VARCHAR(20) NOT NULL DEFAULT '', -- broker registration number
    ADD COLUMN IF NOT EXISTS brn_expiry DATE;

ALTER TABLE listings
    ADD COLUMN IF NOT EXISTS permit_number          VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS permit_expiry          DATE,
    ADD COLUMN IF NOT EXISTS permit_qr_url          TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS permit_override_by     UUID        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS permit_override_at     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS permit_override_reason TEXT        NOT NULL DEFAULT '';

-- One row per alert sent. The expiry date is part of the key, so renewing a licence (a
-- new expiry date) re-arms its alerts.
CREATE TABLE compliance_alerts (
    company_id     UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    entity_type    TEXT        NOT NULL, -- 'listing_permit' | 'brn' | 'trade_license' | …
    entity_id      UUID        NOT NULL,
    expiry_date    DATE        NOT NULL,
    days_before    INTEGER     NOT NULL,
    sent_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (company_id, entity_type, entity_id, expiry_date, days_before)
);

-- +goose Down
DROP TABLE IF EXISTS compliance_alerts;
ALTER TABLE listings
    DROP COLUMN IF EXISTS permit_override_reason,
    DROP COLUMN IF EXISTS permit_override_at,
    DROP COLUMN IF EXISTS permit_override_by,
    DROP COLUMN IF EXISTS permit_qr_url,
    DROP COLUMN IF EXISTS permit_expiry,
    DROP COLUMN IF EXISTS permit_number;
ALTER TABLE users DROP COLUMN IF EXISTS brn_expiry, DROP COLUMN IF EXISTS brn;
ALTER TABLE company_settings
    DROP COLUMN IF EXISTS trade_license_expiry,
    DROP COLUMN IF EXISTS trade_license_no,
    DROP COLUMN IF EXISTS orn;
