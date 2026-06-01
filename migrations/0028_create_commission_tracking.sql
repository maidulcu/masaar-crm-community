-- +goose Up
CREATE TABLE commission_structures (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    structure_name VARCHAR(100) NOT NULL,
    commission_type VARCHAR(50) NOT NULL,
    applicable_to VARCHAR(50),
    effective_from DATE,
    effective_to DATE,
    rules JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, structure_name)
);

CREATE TABLE agent_commissions (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    agent_id UUID NOT NULL,
    commission_period_start DATE NOT NULL,
    commission_period_end DATE NOT NULL,
    commission_structure_id UUID,
    deals_count INT DEFAULT 0,
    deals_revenue DECIMAL(15,2) DEFAULT 0,
    leases_count INT DEFAULT 0,
    leases_revenue DECIMAL(15,2) DEFAULT 0,
    total_commission DECIMAL(15,2),
    status VARCHAR(50),
    approval_date TIMESTAMP,
    payment_date TIMESTAMP,
    payment_reference VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE commission_transactions (
    id UUID PRIMARY KEY,
    commission_id UUID NOT NULL,
    deal_id UUID,
    lease_id UUID,
    transaction_amount DECIMAL(15,2),
    transaction_type VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_commission_structures_company ON commission_structures(company_id);
CREATE INDEX idx_agent_commissions_agent_period ON agent_commissions(agent_id, commission_period_start);
CREATE INDEX idx_agent_commissions_status ON agent_commissions(status);
CREATE INDEX idx_commission_transactions_commission ON commission_transactions(commission_id);

-- +goose Down
DROP TABLE IF EXISTS commission_transactions;
DROP TABLE IF EXISTS agent_commissions;
DROP TABLE IF EXISTS commission_structures;
