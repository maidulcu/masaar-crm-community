-- +goose Up
-- FTA-style tax invoices: line items with VAT categories, supplier and customer details
-- captured at issue time, currency, credit notes and void, and gap-free numbering.

-- Amounts are now computed per line (in Go) and stored, rounded to the currency.
ALTER TABLE vat_invoices ALTER COLUMN vat_amount DROP EXPRESSION;
ALTER TABLE vat_invoices ALTER COLUMN total DROP EXPRESSION;
ALTER TABLE vat_invoices
    ALTER COLUMN subtotal   TYPE NUMERIC(15,3),
    ALTER COLUMN vat_amount TYPE NUMERIC(15,3),
    ALTER COLUMN total      TYPE NUMERIC(15,3),
    ALTER COLUMN vat_amount SET NOT NULL,
    ALTER COLUMN total      SET NOT NULL;

ALTER TABLE vat_invoices
    ADD COLUMN kind                TEXT        NOT NULL DEFAULT 'invoice' CHECK (kind IN ('invoice','credit_note')),
    ADD COLUMN original_invoice_id UUID        REFERENCES vat_invoices(id) ON DELETE RESTRICT,
    ADD COLUMN currency            CHAR(3)     NOT NULL DEFAULT 'AED',
    ADD COLUMN supply_date         DATE,
    ADD COLUMN supplier_name       TEXT        NOT NULL DEFAULT '',
    ADD COLUMN supplier_address    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN supplier_trn        TEXT        NOT NULL DEFAULT '',
    ADD COLUMN customer_name       TEXT        NOT NULL DEFAULT '',
    ADD COLUMN customer_address    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN customer_trn        TEXT        NOT NULL DEFAULT '',
    ADD COLUMN notes               TEXT        NOT NULL DEFAULT '',
    ADD COLUMN reason              TEXT        NOT NULL DEFAULT '', -- credit-note reason
    ADD COLUMN void_reason         TEXT        NOT NULL DEFAULT '',
    ADD COLUMN voided_at           TIMESTAMPTZ,
    ADD COLUMN created_by          UUID        REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT vat_invoices_credit_note_has_original
        CHECK ((kind = 'credit_note') = (original_invoice_id IS NOT NULL));

ALTER TABLE vat_invoices DROP CONSTRAINT IF EXISTS vat_invoices_status_check;
ALTER TABLE vat_invoices ADD CONSTRAINT vat_invoices_status_check
    CHECK (status IN ('draft','sent','paid','void'));

CREATE INDEX idx_vat_invoices_original ON vat_invoices(original_invoice_id) WHERE original_invoice_id IS NOT NULL;

CREATE TABLE vat_invoice_lines (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id   UUID          NOT NULL REFERENCES vat_invoices(id) ON DELETE CASCADE,
    company_id   UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    position     INTEGER       NOT NULL,
    description  TEXT          NOT NULL,
    quantity     NUMERIC(15,4) NOT NULL CHECK (quantity > 0),
    unit_price   NUMERIC(15,3) NOT NULL CHECK (unit_price >= 0),
    vat_category TEXT          NOT NULL CHECK (vat_category IN ('standard','zero_rated','exempt','out_of_scope')),
    vat_rate     NUMERIC(5,4)  NOT NULL,
    net_amount   NUMERIC(15,3) NOT NULL,
    vat_amount   NUMERIC(15,3) NOT NULL,
    UNIQUE (invoice_id, position)
);
CREATE INDEX idx_vat_invoice_lines_company ON vat_invoice_lines(company_id);

-- One counter per company, document kind and year. Incremented inside the issuing
-- transaction, so a rolled-back issue does not leave a gap.
CREATE TABLE invoice_sequences (
    company_id UUID    NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    kind       TEXT    NOT NULL,
    year       INTEGER NOT NULL,
    last_no    INTEGER NOT NULL,
    PRIMARY KEY (company_id, kind, year)
);

-- Existing invoices: one line for the deal, currency from the deal, supplier from the
-- company settings and customer from the deal's contact.
INSERT INTO vat_invoice_lines (invoice_id, company_id, position, description, quantity, unit_price,
                               vat_category, vat_rate, net_amount, vat_amount)
SELECT v.id, v.company_id, 1, COALESCE(NULLIF(d.title, ''), v.invoice_no), 1, v.subtotal,
       CASE WHEN v.vat_rate > 0 THEN 'standard' ELSE 'zero_rated' END, v.vat_rate,
       v.subtotal, v.vat_amount
  FROM vat_invoices v JOIN deals d ON d.id = v.deal_id;

UPDATE vat_invoices v SET currency = d.currency
  FROM deals d WHERE d.id = v.deal_id;

UPDATE vat_invoices v SET supplier_name = s.name, supplier_address = s.business_address,
                          supplier_trn = COALESCE(s.vat_number, '')
  FROM company_settings s WHERE s.company_id = v.company_id;

UPDATE vat_invoices v SET customer_name = COALESCE(c.full_name, '')
  FROM deals d JOIN leads l ON l.id = d.lead_id JOIN contacts c ON c.id = l.contact_id
 WHERE d.id = v.deal_id;

-- Continue numbering after existing INV-YYYY-NNNN numbers.
INSERT INTO invoice_sequences (company_id, kind, year, last_no)
SELECT company_id, 'invoice', SUBSTRING(invoice_no FROM '^INV-(\d{4})-')::INT,
       MAX(SUBSTRING(invoice_no FROM '-(\d+)$')::INT)
  FROM vat_invoices
 WHERE invoice_no ~ '^INV-\d{4}-\d+$'
 GROUP BY company_id, SUBSTRING(invoice_no FROM '^INV-(\d{4})-')::INT;

-- +goose Down
DROP TABLE IF EXISTS invoice_sequences;
DROP TABLE IF EXISTS vat_invoice_lines;
DELETE FROM vat_invoices WHERE kind = 'credit_note';
UPDATE vat_invoices SET status = 'sent' WHERE status = 'void';
ALTER TABLE vat_invoices DROP CONSTRAINT IF EXISTS vat_invoices_status_check;
ALTER TABLE vat_invoices ADD CONSTRAINT vat_invoices_status_check CHECK (status IN ('draft','sent','paid'));
ALTER TABLE vat_invoices
    DROP CONSTRAINT IF EXISTS vat_invoices_credit_note_has_original,
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS voided_at,
    DROP COLUMN IF EXISTS void_reason,
    DROP COLUMN IF EXISTS reason,
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS customer_trn,
    DROP COLUMN IF EXISTS customer_address,
    DROP COLUMN IF EXISTS customer_name,
    DROP COLUMN IF EXISTS supplier_trn,
    DROP COLUMN IF EXISTS supplier_address,
    DROP COLUMN IF EXISTS supplier_name,
    DROP COLUMN IF EXISTS supply_date,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS original_invoice_id,
    DROP COLUMN IF EXISTS kind;
ALTER TABLE vat_invoices DROP COLUMN vat_amount, DROP COLUMN total;
ALTER TABLE vat_invoices ALTER COLUMN subtotal TYPE NUMERIC(12,2);
ALTER TABLE vat_invoices
    ADD COLUMN vat_amount NUMERIC(12,2) GENERATED ALWAYS AS (subtotal * vat_rate) STORED,
    ADD COLUMN total      NUMERIC(12,2) GENERATED ALWAYS AS (subtotal + subtotal * vat_rate) STORED;
