-- +goose Up
-- Uploaded bank statements are kept on local disk (see internal/mediastore). storage_key is the
-- opaque stored file name; it is NULL for rows created before the file was actually saved.
ALTER TABLE bank_statements ADD COLUMN IF NOT EXISTS storage_key VARCHAR(64);

-- +goose Down
ALTER TABLE bank_statements DROP COLUMN IF EXISTS storage_key;
