-- +goose Up
CREATE TABLE vat_invoices (
    id          UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    deal_id     UUID          NOT NULL REFERENCES deals(id) ON DELETE RESTRICT,
    invoice_no  TEXT          NOT NULL UNIQUE,
    subtotal    NUMERIC(12,2) NOT NULL,
    vat_rate    NUMERIC(4,3)  NOT NULL DEFAULT 0.05,
    vat_amount  NUMERIC(12,2) GENERATED ALWAYS AS (subtotal * vat_rate) STORED,
    total       NUMERIC(12,2) GENERATED ALWAYS AS (subtotal + subtotal * vat_rate) STORED,
    qr_payload  TEXT,
    status      TEXT          NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','sent','paid')),
    issued_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_deal   ON vat_invoices(deal_id);
CREATE INDEX idx_invoices_status ON vat_invoices(status);

-- +goose Down
DROP TABLE IF EXISTS vat_invoices;
