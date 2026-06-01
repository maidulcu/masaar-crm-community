-- +goose Up

-- Stores round-robin rotation state and per-company rotation config.
-- rotation_index tracks which agent was assigned last so the next assignment
-- picks the next active agent in alphabetical order of user ID.
CREATE TABLE IF NOT EXISTS lead_rotation_settings (
    id              UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id      UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    mode            TEXT        NOT NULL DEFAULT 'manual'
                                CHECK (mode IN ('manual','round_robin','capacity')),
    enabled         BOOLEAN     NOT NULL DEFAULT false,
    rotation_index  INT         NOT NULL DEFAULT 0,  -- index into sorted agent list
    max_per_agent   INT         NOT NULL DEFAULT 0,  -- 0 = unlimited (capacity mode)
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT lead_rotation_company_unique UNIQUE (company_id)
);

-- Insert a default row for the default company so settings always exist
INSERT INTO lead_rotation_settings (company_id, mode, enabled)
VALUES ('00000000-0000-0000-0000-000000000001', 'manual', false)
ON CONFLICT (company_id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS lead_rotation_settings;
