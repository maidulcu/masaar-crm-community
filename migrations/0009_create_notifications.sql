-- +goose Up
CREATE TABLE IF NOT EXISTS notifications (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       TEXT        NOT NULL DEFAULT 'info',
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL DEFAULT '',
    read       BOOLEAN     NOT NULL DEFAULT false,
    data       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user   ON notifications(user_id);
CREATE INDEX idx_notifications_unread ON notifications(user_id) WHERE read = false;
CREATE INDEX idx_notifications_ts     ON notifications(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS notifications;
