-- +goose Up
-- Tasks and follow-ups ("call this lead back on Thursday"), optionally linked to a lead,
-- contact, deal, listing, viewing or offer.
CREATE TABLE tasks (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id       UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    title            TEXT        NOT NULL,
    description      TEXT        NOT NULL DEFAULT '',
    type             TEXT        NOT NULL DEFAULT 'follow_up' CHECK (type IN ('call','meeting','follow_up','todo')),
    priority         TEXT        NOT NULL DEFAULT 'normal' CHECK (priority IN ('low','normal','high')),
    status           TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open','done','cancelled')),
    due_at           TIMESTAMPTZ,
    assigned_to      UUID        REFERENCES users(id) ON DELETE SET NULL,
    related_type     TEXT        CHECK (related_type IN ('lead','contact','deal','listing','viewing','offer')),
    related_id       UUID,
    created_by       UUID        REFERENCES users(id) ON DELETE SET NULL,
    completed_at     TIMESTAMPTZ,
    reminder_sent_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((related_type IS NULL) = (related_id IS NULL))
);
CREATE INDEX idx_tasks_assignee_open ON tasks(company_id, assigned_to, due_at) WHERE status = 'open';
CREATE INDEX idx_tasks_related ON tasks(company_id, related_type, related_id);
CREATE INDEX idx_tasks_due_reminder ON tasks(due_at) WHERE status = 'open' AND reminder_sent_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS tasks;
