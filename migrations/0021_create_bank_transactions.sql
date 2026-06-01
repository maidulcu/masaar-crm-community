-- +goose Up
CREATE TABLE bank_transactions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  bank_integration_id UUID REFERENCES bank_integrations(id) ON DELETE SET NULL,

  -- Transaction details
  external_id VARCHAR(255) UNIQUE, -- Bank's transaction ID
  transaction_date DATE NOT NULL,
  amount DECIMAL(12, 2) NOT NULL,
  currency VARCHAR(3) DEFAULT 'AED',

  -- Parties
  from_account VARCHAR(100),
  to_account VARCHAR(100),
  from_name VARCHAR(255),
  to_name VARCHAR(255),

  -- Transaction info
  reference VARCHAR(500), -- Bank reference, description
  transaction_type VARCHAR(50), -- credit, debit, transfer, check
  status VARCHAR(50) DEFAULT 'completed', -- pending, completed, failed, reversed

  -- Reconciliation (matched_payment_id added later via 0023 after payments table exists)
  match_confidence DECIMAL(3, 2), -- 0.0 to 1.0 confidence score
  matched_at TIMESTAMP,

  -- Sync metadata
  imported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_checked TIMESTAMP,
  sync_error TEXT,

  CONSTRAINT check_amount CHECK (amount >= 0)
);

CREATE INDEX idx_bank_transactions_company_id ON bank_transactions(company_id);
CREATE INDEX idx_bank_transactions_bank_integration_id ON bank_transactions(bank_integration_id);
CREATE INDEX idx_bank_transactions_external_id ON bank_transactions(external_id);
CREATE INDEX idx_bank_transactions_transaction_date ON bank_transactions(transaction_date);
CREATE INDEX idx_bank_transactions_status ON bank_transactions(status);

-- +goose Down
DROP TABLE bank_transactions;
