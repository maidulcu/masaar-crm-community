-- +goose Up

-- Per-company approval workflow configuration.
CREATE TABLE approval_configs (
    id                   UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id           UUID         NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    listing_approval     BOOLEAN      NOT NULL DEFAULT false,  -- require admin approval before publish
    deal_approval_above  NUMERIC(14,2) NOT NULL DEFAULT 0,     -- 0 = no threshold (always approve)
    offer_approval_above NUMERIC(14,2) NOT NULL DEFAULT 0,     -- 0 = agents can accept any offer
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT approval_configs_company_unique UNIQUE (company_id)
);

-- Insert default for the default company
INSERT INTO approval_configs (company_id) VALUES ('00000000-0000-0000-0000-000000000001')
ON CONFLICT (company_id) DO NOTHING;

-- Individual approval requests.
CREATE TYPE approval_status AS ENUM ('pending', 'approved', 'rejected');
CREATE TYPE approval_entity AS ENUM ('listing', 'deal', 'offer');

CREATE TABLE approval_requests (
    id            UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id    UUID            NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    entity_type   approval_entity NOT NULL,
    entity_id     UUID            NOT NULL,
    requested_by  UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewed_by   UUID            REFERENCES users(id) ON DELETE SET NULL,
    status        approval_status NOT NULL DEFAULT 'pending',
    notes         TEXT            NOT NULL DEFAULT '',
    reviewer_note TEXT            NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    reviewed_at   TIMESTAMPTZ
);

CREATE INDEX idx_approvals_company   ON approval_requests(company_id);
CREATE INDEX idx_approvals_entity    ON approval_requests(entity_type, entity_id);
CREATE INDEX idx_approvals_status    ON approval_requests(status);
CREATE INDEX idx_approvals_requester ON approval_requests(requested_by);

-- +goose Down
DROP TABLE IF EXISTS approval_requests;
DROP TYPE  IF EXISTS approval_entity;
DROP TYPE  IF EXISTS approval_status;
DROP TABLE IF EXISTS approval_configs;
