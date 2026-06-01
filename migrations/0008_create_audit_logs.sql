-- +goose Up
CREATE TABLE audit_logs (
    id          BIGSERIAL   PRIMARY KEY,
    entity_type TEXT        NOT NULL,
    entity_id   UUID        NOT NULL,
    action      TEXT        NOT NULL,
    actor_id    UUID        REFERENCES users(id) ON DELETE SET NULL,
    diff        JSONB,
    ts          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_ts     ON audit_logs(ts DESC);
CREATE INDEX idx_audit_actor  ON audit_logs(actor_id);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
