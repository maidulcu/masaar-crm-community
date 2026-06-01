-- +goose Up

CREATE TYPE viewing_status AS ENUM (
    'scheduled',
    'confirmed',
    'checked_in',
    'completed',
    'cancelled',
    'no_show'
);

CREATE TABLE viewings (
    id           UUID            PRIMARY KEY DEFAULT uuid_generate_v4(),
    listing_id   UUID            REFERENCES listings(id) ON DELETE SET NULL,
    contact_id   UUID            NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    agent_id     UUID            REFERENCES users(id) ON DELETE SET NULL,
    lead_id      UUID            REFERENCES leads(id) ON DELETE SET NULL,

    scheduled_at TIMESTAMPTZ     NOT NULL,
    duration_min INT             NOT NULL DEFAULT 30,
    status       viewing_status  NOT NULL DEFAULT 'scheduled',

    -- Location: either from the linked listing or a manual address
    address      TEXT            NOT NULL DEFAULT '',

    notes        TEXT            NOT NULL DEFAULT '',

    -- Check-in / check-out tracking
    checked_in_at  TIMESTAMPTZ,
    checked_out_at TIMESTAMPTZ,

    -- Reminder tracking
    reminder_sent  BOOLEAN       NOT NULL DEFAULT false,

    created_at   TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_viewings_agent        ON viewings(agent_id)     WHERE agent_id IS NOT NULL;
CREATE INDEX idx_viewings_contact      ON viewings(contact_id);
CREATE INDEX idx_viewings_listing      ON viewings(listing_id)   WHERE listing_id IS NOT NULL;
CREATE INDEX idx_viewings_scheduled_at ON viewings(scheduled_at);
CREATE INDEX idx_viewings_status       ON viewings(status);

-- +goose Down
DROP TABLE IF EXISTS viewings;
DROP TYPE  IF EXISTS viewing_status;
