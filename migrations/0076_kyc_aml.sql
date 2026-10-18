-- +goose Up
-- KYC (customer due diligence) records and AML review support (#87). This is record
-- keeping and workflow: nothing here makes a compliance decision on its own.

-- A compliance officer role: reads like a viewer, plus the KYC/AML screens.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin','agent','viewer','compliance'));

-- Identity documents. The copy lives in the private media store (storage_key).
CREATE TABLE kyc_documents (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id      UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    contact_id      UUID        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    doc_type        TEXT        NOT NULL CHECK (doc_type IN ('emirates_id','passport','residence_visa','trade_licence','other')),
    number          TEXT        NOT NULL,
    issuing_country CHAR(2)     NOT NULL DEFAULT '',
    issue_date      DATE,
    expiry_date     DATE,
    storage_key     TEXT        NOT NULL DEFAULT '',
    file_name       TEXT        NOT NULL DEFAULT '',
    created_by      UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_kyc_documents_contact ON kyc_documents(contact_id);
CREATE INDEX idx_kyc_documents_expiry ON kyc_documents(company_id, expiry_date) WHERE expiry_date IS NOT NULL;

-- Due-diligence fields on the contact.
ALTER TABLE contacts
    ADD COLUMN IF NOT EXISTS date_of_birth        DATE,
    ADD COLUMN IF NOT EXISTS kyc_risk             TEXT        NOT NULL DEFAULT '' CHECK (kyc_risk IN ('','low','medium','high')),
    ADD COLUMN IF NOT EXISTS kyc_risk_reason      TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS kyc_is_pep           BOOLEAN     NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS kyc_screening        TEXT        NOT NULL DEFAULT 'not_screened'
                                                  CHECK (kyc_screening IN ('not_screened','clear','possible_match','match')),
    ADD COLUMN IF NOT EXISTS kyc_screening_source TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS kyc_screened_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS kyc_notes            TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS kyc_reviewed_by      UUID        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS kyc_reviewed_at      TIMESTAMPTZ;

-- Beneficial owners of corporate clients.
CREATE TABLE beneficial_owners (
    company_id      UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    organisation_id UUID         NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    contact_id      UUID         NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    ownership_pct   NUMERIC(5,2) CHECK (ownership_pct IS NULL OR (ownership_pct > 0 AND ownership_pct <= 100)),
    role            TEXT         NOT NULL DEFAULT 'owner' CHECK (role IN ('owner','controller','signatory')),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organisation_id, contact_id)
);

-- How a deal is paid, the KYC override, and the AML review.
ALTER TABLE deals
    ADD COLUMN IF NOT EXISTS payment_method      TEXT          NOT NULL DEFAULT ''
                                                 CHECK (payment_method IN ('','cash','cheque','transfer','virtual_asset','mixed')),
    ADD COLUMN IF NOT EXISTS cash_amount         NUMERIC(15,3) NOT NULL DEFAULT 0 CHECK (cash_amount >= 0),
    ADD COLUMN IF NOT EXISTS kyc_override_reason TEXT          NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS kyc_override_by     UUID          REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS kyc_override_at     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS aml_status          TEXT          NOT NULL DEFAULT ''
                                                 CHECK (aml_status IN ('','flagged','no_report','reported')),
    ADD COLUMN IF NOT EXISTS aml_reasons         TEXT[]        NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS aml_flagged_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS aml_reviewed_by     UUID          REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS aml_reviewed_at     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS aml_notes           TEXT          NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS goaml_reference     TEXT          NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_deals_aml ON deals(company_id, aml_status) WHERE aml_status <> '';

-- Thresholds (check the current figures with the Ministry of Economy / UAE FIU).
ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS aml_cash_threshold     NUMERIC(15,3) NOT NULL DEFAULT 55000,
    ADD COLUMN IF NOT EXISTS aml_flag_virtual_asset BOOLEAN       NOT NULL DEFAULT TRUE;

-- +goose Down
ALTER TABLE company_settings DROP COLUMN IF EXISTS aml_flag_virtual_asset, DROP COLUMN IF EXISTS aml_cash_threshold;
DROP INDEX IF EXISTS idx_deals_aml;
ALTER TABLE deals
    DROP COLUMN IF EXISTS goaml_reference, DROP COLUMN IF EXISTS aml_notes, DROP COLUMN IF EXISTS aml_reviewed_at,
    DROP COLUMN IF EXISTS aml_reviewed_by, DROP COLUMN IF EXISTS aml_flagged_at, DROP COLUMN IF EXISTS aml_reasons,
    DROP COLUMN IF EXISTS aml_status, DROP COLUMN IF EXISTS kyc_override_at, DROP COLUMN IF EXISTS kyc_override_by,
    DROP COLUMN IF EXISTS kyc_override_reason, DROP COLUMN IF EXISTS cash_amount, DROP COLUMN IF EXISTS payment_method;
DROP TABLE IF EXISTS beneficial_owners;
ALTER TABLE contacts
    DROP COLUMN IF EXISTS kyc_reviewed_at, DROP COLUMN IF EXISTS kyc_reviewed_by, DROP COLUMN IF EXISTS kyc_notes,
    DROP COLUMN IF EXISTS kyc_screened_at, DROP COLUMN IF EXISTS kyc_screening_source, DROP COLUMN IF EXISTS kyc_screening,
    DROP COLUMN IF EXISTS kyc_is_pep, DROP COLUMN IF EXISTS kyc_risk_reason, DROP COLUMN IF EXISTS kyc_risk,
    DROP COLUMN IF EXISTS date_of_birth;
DROP TABLE IF EXISTS kyc_documents;
UPDATE users SET role = 'viewer' WHERE role = 'compliance';
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin','agent','viewer'));
