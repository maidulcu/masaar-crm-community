-- +goose Up

-- Per-agent monthly targets for KPI tracking.
-- period is stored as YYYY-MM (e.g. '2026-05').
CREATE TABLE agent_targets (
    id           UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id     UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id   UUID          NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    metric       TEXT          NOT NULL
                               CHECK (metric IN ('deals_won','revenue','listings_added','leads_converted','leads_assigned')),
    target_value NUMERIC(14,2) NOT NULL DEFAULT 0,
    period       CHAR(7)       NOT NULL,  -- YYYY-MM
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT agent_targets_unique UNIQUE (agent_id, metric, period)
);

CREATE INDEX idx_agent_targets_agent   ON agent_targets(agent_id);
CREATE INDEX idx_agent_targets_company ON agent_targets(company_id);
CREATE INDEX idx_agent_targets_period  ON agent_targets(period);

-- +goose Down
DROP TABLE IF EXISTS agent_targets;
