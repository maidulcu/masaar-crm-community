-- +goose Up
-- WhatsApp broadcasts, opt-out by reply, inbox assignment and business-hours auto-reply (#87).

-- Inbox: who handles a thread, and when the business-hours auto-reply last went out.
-- (The 24-hour service window is read from the thread's latest inbound message.)
ALTER TABLE whatsapp_threads
    ADD COLUMN IF NOT EXISTS assigned_to   UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS auto_reply_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_whatsapp_messages_inbound ON whatsapp_messages(thread_id, sent_at DESC) WHERE direction = 'inbound';

-- A broadcast sends an approved template to a segment of contacts who agreed to WhatsApp.
CREATE TABLE whatsapp_broadcasts (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id    UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    template_name TEXT        NOT NULL,          -- approved in Meta's WhatsApp Manager
    language      TEXT        NOT NULL DEFAULT 'en',
    params        TEXT[]      NOT NULL DEFAULT '{}', -- body parameters; {{name}} is the contact's name
    segment       JSONB       NOT NULL DEFAULT '{}',
    status        TEXT        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','sending','done','cancelled')),
    created_by    UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ
);
CREATE INDEX idx_wa_broadcasts_company ON whatsapp_broadcasts(company_id, created_at DESC);

CREATE TABLE whatsapp_broadcast_recipients (
    broadcast_id  UUID        NOT NULL REFERENCES whatsapp_broadcasts(id) ON DELETE CASCADE,
    contact_id    UUID        NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    company_id    UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    phone         TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','sending','sent','delivered','read','failed','skipped')),
    skip_reason   TEXT        NOT NULL DEFAULT '',
    wa_message_id TEXT,
    error         TEXT        NOT NULL DEFAULT '',
    sent_at       TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (broadcast_id, contact_id)
);
CREATE INDEX idx_wa_broadcast_pending ON whatsapp_broadcast_recipients(company_id, status) WHERE status IN ('pending','sending');
CREATE UNIQUE INDEX idx_wa_broadcast_waid ON whatsapp_broadcast_recipients(wa_message_id) WHERE wa_message_id IS NOT NULL;

-- Settings: sending rate, and the business-hours auto-reply.
ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS wa_broadcast_per_minute INTEGER NOT NULL DEFAULT 30 CHECK (wa_broadcast_per_minute BETWEEN 1 AND 1000),
    ADD COLUMN IF NOT EXISTS wa_auto_reply_enabled   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS wa_auto_reply_en        TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS wa_auto_reply_ar        TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS wa_business_hours       JSONB;  -- bizhours.Schedule; NULL = 09:00–18:00 on working days

-- +goose Down
ALTER TABLE company_settings
    DROP COLUMN IF EXISTS wa_business_hours, DROP COLUMN IF EXISTS wa_auto_reply_ar, DROP COLUMN IF EXISTS wa_auto_reply_en,
    DROP COLUMN IF EXISTS wa_auto_reply_enabled, DROP COLUMN IF EXISTS wa_broadcast_per_minute;
DROP TABLE IF EXISTS whatsapp_broadcast_recipients;
DROP TABLE IF EXISTS whatsapp_broadcasts;
DROP INDEX IF EXISTS idx_whatsapp_messages_inbound;
ALTER TABLE whatsapp_threads DROP COLUMN IF EXISTS auto_reply_at, DROP COLUMN IF EXISTS assigned_to;
