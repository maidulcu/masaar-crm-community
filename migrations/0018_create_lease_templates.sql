-- +goose Up
CREATE TABLE lease_templates (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

  -- Template details
  name VARCHAR(255) NOT NULL,
  description TEXT,
  is_default BOOLEAN DEFAULT FALSE,

  -- Payment schedule
  payment_frequency VARCHAR(50) NOT NULL, -- monthly, quarterly, semi_annual, annual
  payment_day_of_month INTEGER, -- 1-31, relevant for monthly
  auto_generate_payments BOOLEAN DEFAULT TRUE,

  -- Financial
  default_security_deposit_percent DECIMAL(5, 2), -- e.g., 5.00 for 5%
  default_utility_charges DECIMAL(12, 2),
  default_late_fee_percent DECIMAL(5, 2), -- e.g., 2.00 for 2%

  -- Terms
  default_lease_duration_months INTEGER, -- e.g., 12
  default_notice_period_days INTEGER, -- e.g., 30
  default_renewal_duration_months INTEGER, -- e.g., 12

  -- Document
  template_document_url VARCHAR(500),
  terms_conditions TEXT,

  -- Status
  status VARCHAR(50) DEFAULT 'active', -- active, inactive, archived

  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id),
  updated_by UUID REFERENCES users(id)
);

CREATE INDEX idx_lease_templates_company_id ON lease_templates(company_id);
CREATE INDEX idx_lease_templates_is_default ON lease_templates(is_default);
CREATE INDEX idx_lease_templates_status ON lease_templates(status);

-- +goose Down
DROP TABLE lease_templates;
