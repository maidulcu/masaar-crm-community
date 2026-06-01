-- +goose Up
CREATE TABLE bank_integrations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

  -- Bank details
  bank_name VARCHAR(255) NOT NULL,
  bank_code VARCHAR(50),
  account_number VARCHAR(100),
  account_name VARCHAR(255),
  iban VARCHAR(50),

  -- Integration config
  integration_type VARCHAR(50) NOT NULL DEFAULT 'manual', -- manual, api, file_import
  status VARCHAR(50) NOT NULL DEFAULT 'active', -- active, inactive, error
  api_key_encrypted TEXT,
  api_secret_encrypted TEXT,
  api_endpoint VARCHAR(500),

  -- Sync settings
  auto_sync BOOLEAN DEFAULT FALSE,
  sync_interval_hours INT DEFAULT 24,
  last_sync_date TIMESTAMP,
  last_sync_error TEXT,
  sync_error_count INT DEFAULT 0,

  -- Connection status
  is_connected BOOLEAN DEFAULT FALSE,
  connection_test_date TIMESTAMP,

  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id) ON DELETE SET NULL,
  updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_bank_integrations_company_id ON bank_integrations(company_id);
CREATE INDEX idx_bank_integrations_status ON bank_integrations(status);

-- +goose Down
DROP TABLE bank_integrations;
