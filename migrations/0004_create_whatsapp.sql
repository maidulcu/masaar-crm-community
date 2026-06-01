-- +goose Up
CREATE TABLE whatsapp_threads (
    id              UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    contact_id      UUID         NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    wa_account_id   TEXT         NOT NULL,
    thread_status   TEXT         NOT NULL DEFAULT 'open' CHECK (thread_status IN ('open','closed','pending')),
    last_message_at TIMESTAMPTZ,
    message_count   INT          NOT NULL DEFAULT 0,
    ai_summary      TEXT,
    embedding       vector(1536),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_contact_account UNIQUE (contact_id, wa_account_id)
);

CREATE INDEX idx_threads_contact    ON whatsapp_threads(contact_id);
CREATE INDEX idx_threads_status     ON whatsapp_threads(thread_status);
CREATE INDEX idx_threads_last_msg   ON whatsapp_threads(last_message_at DESC);
CREATE INDEX idx_threads_embedding  ON whatsapp_threads USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

CREATE TABLE whatsapp_messages (
    id            UUID             PRIMARY KEY DEFAULT uuid_generate_v4(),
    thread_id     UUID             NOT NULL REFERENCES whatsapp_threads(id) ON DELETE CASCADE,
    direction     TEXT             NOT NULL CHECK (direction IN ('inbound','outbound')),
    body          TEXT             NOT NULL DEFAULT '',
    media_url     TEXT,
    wa_message_id TEXT             UNIQUE,
    sent_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_thread  ON whatsapp_messages(thread_id);
CREATE INDEX idx_messages_sent_at ON whatsapp_messages(sent_at DESC);

-- +goose Down
DROP TABLE IF EXISTS whatsapp_messages;
DROP TABLE IF EXISTS whatsapp_threads;
