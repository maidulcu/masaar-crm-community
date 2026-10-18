-- +goose Up
-- Pre-filled RERA brokerage forms (A, B, I, F/MOU). Each is a document (so it can go
-- through the e-signature flow) with a snapshot of the form's fields.
CREATE TABLE rera_forms (
    document_id UUID        PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    company_id  UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    form_type   CHAR(1)     NOT NULL CHECK (form_type IN ('A','B','I','F')),
    listing_id  UUID        REFERENCES listings(id) ON DELETE SET NULL,
    contact_id  UUID        REFERENCES contacts(id) ON DELETE SET NULL,
    offer_id    UUID        REFERENCES offers(id) ON DELETE SET NULL,
    data        JSONB       NOT NULL,
    created_by  UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_rera_forms_listing ON rera_forms(company_id, listing_id) WHERE listing_id IS NOT NULL;
CREATE INDEX idx_rera_forms_contact ON rera_forms(company_id, contact_id) WHERE contact_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS rera_forms;
