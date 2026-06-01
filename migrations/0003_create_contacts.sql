-- +goose Up
CREATE TABLE contacts (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone_wa    VARCHAR(20) NOT NULL UNIQUE,
    full_name   TEXT        NOT NULL,
    email       TEXT,
    language    VARCHAR(2)  NOT NULL DEFAULT 'en',
    lead_score  SMALLINT    NOT NULL DEFAULT 0,
    assigned_to UUID        REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contacts_phone_wa   ON contacts(phone_wa);
CREATE INDEX idx_contacts_assigned   ON contacts(assigned_to);
CREATE INDEX idx_contacts_lead_score ON contacts(lead_score DESC);

-- +goose Down
DROP TABLE IF EXISTS contacts;
