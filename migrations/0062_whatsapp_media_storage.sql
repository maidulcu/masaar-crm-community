-- +goose Up
-- Inbound WhatsApp media is downloaded from Meta and kept on local disk (see internal/mediastore).
-- media_path is the opaque stored file name (NULL until downloaded), so a file that failed to
-- download can be fetched on demand later using wa_media_id.
ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS media_path     TEXT;
ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS media_size     BIGINT;
ALTER TABLE whatsapp_messages ADD COLUMN IF NOT EXISTS media_filename TEXT;

-- +goose Down
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS media_filename;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS media_size;
ALTER TABLE whatsapp_messages DROP COLUMN IF EXISTS media_path;
