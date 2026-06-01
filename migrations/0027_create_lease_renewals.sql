-- +goose Up
CREATE TABLE lease_renewal_workflows (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    lease_id UUID NOT NULL UNIQUE,
    renewal_date DATE NOT NULL,
    renewal_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    days_before_expiry INT DEFAULT 90,
    proposed_rent_amount DECIMAL(15,2),
    proposed_terms JSONB,
    tenant_response VARCHAR(50) DEFAULT 'pending',
    tenant_counter_offer DECIMAL(15,2),
    counter_offer_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE renewal_communication_templates (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    template_name VARCHAR(150) NOT NULL,
    email_subject VARCHAR(255),
    email_body TEXT,
    whatsapp_message TEXT,
    language VARCHAR(2) DEFAULT 'en',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, template_name, language)
);

CREATE TABLE renewal_communication_log (
    id UUID PRIMARY KEY,
    renewal_id UUID NOT NULL,
    communication_type VARCHAR(50),
    template_id UUID,
    sent_date TIMESTAMP,
    delivery_status VARCHAR(50),
    tenant_response_date TIMESTAMP,
    response_text TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_lease_renewal_status ON lease_renewal_workflows(company_id, renewal_status);
CREATE INDEX idx_lease_renewal_date ON lease_renewal_workflows(renewal_date);
CREATE INDEX idx_renewal_templates_company ON renewal_communication_templates(company_id);
CREATE INDEX idx_renewal_log_renewal ON renewal_communication_log(renewal_id);

-- +goose Down
DROP TABLE IF EXISTS renewal_communication_log;
DROP TABLE IF EXISTS renewal_communication_templates;
DROP TABLE IF EXISTS lease_renewal_workflows;
