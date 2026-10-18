-- +goose Up
-- UAE e-invoicing (#88): the data a PINT AE e-invoice needs beyond the tax invoice, and
-- each invoice's submission to the company's accredited service provider.

ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS einvoice_provider       TEXT NOT NULL DEFAULT 'off', -- off (PDF only) | mock | an installed connector
    ADD COLUMN IF NOT EXISTS einvoice_tin            TEXT NOT NULL DEFAULT '',    -- 10-digit TIN: the e-invoicing participant ID
    ADD COLUMN IF NOT EXISTS legal_id_type           TEXT NOT NULL DEFAULT 'TL' CHECK (legal_id_type IN ('TL','EID','PAS','CD')),
    ADD COLUMN IF NOT EXISTS trade_license_authority TEXT NOT NULL DEFAULT '',    -- issuing authority of the trade licence
    ADD COLUMN IF NOT EXISTS business_city           TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS einvoice_service_code   TEXT NOT NULL DEFAULT '',    -- service accounting code (SAC) of the lines
    ADD COLUMN IF NOT EXISTS einvoice_exemption_code TEXT NOT NULL DEFAULT '',    -- VAT exemption reason code for exempt lines
    -- Saudi groundwork: also show dates in the Umm al-Qura (Hijri) calendar.
    ADD COLUMN IF NOT EXISTS show_hijri              BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE einvoices (
    invoice_id    UUID        PRIMARY KEY REFERENCES vat_invoices(id) ON DELETE CASCADE,
    company_id    UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    provider      TEXT        NOT NULL,
    status        TEXT        NOT NULL CHECK (status IN ('submitted','accepted','rejected','error')),
    provider_ref  TEXT        NOT NULL DEFAULT '',
    reason        TEXT        NOT NULL DEFAULT '', -- rejection or error message
    buyer         JSONB       NOT NULL DEFAULT '{}',
    credit_reason TEXT        NOT NULL DEFAULT '',
    due_date      DATE,
    xml           TEXT        NOT NULL,
    submitted_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    submitted_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_einvoices_pending ON einvoices(company_id) WHERE status = 'submitted';

-- +goose Down
DROP TABLE IF EXISTS einvoices;
ALTER TABLE company_settings
    DROP COLUMN IF EXISTS show_hijri,
    DROP COLUMN IF EXISTS einvoice_exemption_code, DROP COLUMN IF EXISTS einvoice_service_code,
    DROP COLUMN IF EXISTS business_city, DROP COLUMN IF EXISTS trade_license_authority,
    DROP COLUMN IF EXISTS legal_id_type, DROP COLUMN IF EXISTS einvoice_tin, DROP COLUMN IF EXISTS einvoice_provider;
