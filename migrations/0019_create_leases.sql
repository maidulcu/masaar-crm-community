-- +goose Up
CREATE TABLE leases (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  property_id UUID NOT NULL REFERENCES rental_properties(id) ON DELETE RESTRICT,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
  template_id UUID REFERENCES lease_templates(id) ON DELETE SET NULL,

  -- Lease period
  start_date DATE NOT NULL,
  end_date DATE NOT NULL,
  renewal_start_date DATE,
  renewal_end_date DATE,

  -- Financial terms
  monthly_rent DECIMAL(12, 2) NOT NULL,
  currency VARCHAR(3) DEFAULT 'AED',
  security_deposit DECIMAL(12, 2),
  utility_charges DECIMAL(12, 2),
  late_fee_percent DECIMAL(5, 2),

  -- Payment schedule
  payment_frequency VARCHAR(50) NOT NULL, -- monthly, quarterly, semi_annual, annual
  payment_day_of_month INTEGER, -- 1-31, relevant for monthly
  auto_generate_payments BOOLEAN DEFAULT TRUE,
  last_generated_payment_date DATE,

  -- Notice & termination
  notice_period_days INTEGER,
  move_out_date DATE,
  move_out_inspection_date DATE,

  -- Documents
  lease_document_url VARCHAR(500),
  signed_by_landlord_date DATE,
  signed_by_tenant_date DATE,
  ejari_number VARCHAR(100), -- UAE Ejari registration number
  ejari_url VARCHAR(500),

  -- Status
  status VARCHAR(50) DEFAULT 'active', -- active, renewed, terminated, expired
  termination_reason VARCHAR(255),
  termination_date DATE,

  -- Notes
  notes TEXT,

  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id),
  updated_by UUID REFERENCES users(id),

  CONSTRAINT check_dates CHECK (start_date < end_date),
  CONSTRAINT check_rent CHECK (monthly_rent > 0)
);

CREATE INDEX idx_leases_company_id ON leases(company_id);
CREATE INDEX idx_leases_property_id ON leases(property_id);
CREATE INDEX idx_leases_tenant_id ON leases(tenant_id);
CREATE INDEX idx_leases_status ON leases(status);
CREATE INDEX idx_leases_start_date ON leases(start_date);
CREATE INDEX idx_leases_end_date ON leases(end_date);

-- +goose Down
DROP TABLE leases;
