-- +goose Up
ALTER TABLE document_signatures
    ADD COLUMN IF NOT EXISTS envelope_id TEXT;

CREATE INDEX IF NOT EXISTS idx_document_signatures_envelope
    ON document_signatures(envelope_id)
    WHERE envelope_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_document_signatures_envelope;
ALTER TABLE document_signatures DROP COLUMN IF EXISTS envelope_id;
