-- +goose Up
CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

  -- Personal info
  full_name_en VARCHAR(255) NOT NULL,
  full_name_ar VARCHAR(255),
  email VARCHAR(255),
  phone VARCHAR(20),
  phone_wa VARCHAR(20),

  -- Identification
  id_type VARCHAR(50) NOT NULL, -- emirati_id, passport, driving_license, trade_license
  id_number VARCHAR(100) UNIQUE,
  id_expiry_date DATE,
  id_document_url VARCHAR(500),

  -- Verification
  is_verified BOOLEAN DEFAULT FALSE,
  verification_status VARCHAR(50) DEFAULT 'pending', -- pending, verified, rejected
  verification_date TIMESTAMP,
  verified_by UUID REFERENCES users(id),
  verification_notes TEXT,

  -- Employment/Income
  employment_status VARCHAR(50), -- employed, self_employed, retired, student
  employer_name VARCHAR(255),
  annual_income DECIMAL(15, 2),
  income_currency VARCHAR(3) DEFAULT 'AED',
  salary_certificate_url VARCHAR(500),

  -- Address
  nationality VARCHAR(100),
  country_of_origin VARCHAR(100),
  permanent_address TEXT,

  -- Emergency contact
  emergency_contact_name VARCHAR(255),
  emergency_contact_phone VARCHAR(20),

  -- Status
  status VARCHAR(50) DEFAULT 'active', -- active, inactive, blacklisted
  notes TEXT,

  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id),
  updated_by UUID REFERENCES users(id)
);

CREATE INDEX idx_tenants_company_id ON tenants(company_id);
CREATE INDEX idx_tenants_email ON tenants(email);
CREATE INDEX idx_tenants_phone ON tenants(phone);
CREATE INDEX idx_tenants_id_number ON tenants(id_number);
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_verification_status ON tenants(verification_status);

-- +goose Down
DROP TABLE tenants;
