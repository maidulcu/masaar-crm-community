-- +goose Up
CREATE TABLE payments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
  lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE RESTRICT,

  -- Payment details
  amount DECIMAL(12, 2) NOT NULL,
  currency VARCHAR(3) DEFAULT 'AED',
  due_date DATE NOT NULL,
  paid_date DATE,

  -- Payment method & status
  payment_method VARCHAR(50) NOT NULL, -- transfer, check, cash, card, other
  payment_reference VARCHAR(255), -- cheque number, transaction ID, etc
  status VARCHAR(50) DEFAULT 'pending', -- pending, received, overdue, failed, refunded

  -- Reconciliation (bank_transaction_id added later via 0023 after bank_transactions exists)
  reconciled_at TIMESTAMP,
  reconciled_by UUID REFERENCES users(id),

  -- Notes & attachments
  notes TEXT,
  receipt_url VARCHAR(500),

  -- Late fees
  late_fee_applied BOOLEAN DEFAULT FALSE,
  late_fee_amount DECIMAL(12, 2),

  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id),
  updated_by UUID REFERENCES users(id),

  CONSTRAINT check_amount CHECK (amount > 0)
);

CREATE INDEX idx_payments_company_id ON payments(company_id);
CREATE INDEX idx_payments_lease_id ON payments(lease_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_due_date ON payments(due_date);
CREATE INDEX idx_payments_paid_date ON payments(paid_date);

-- +goose Down
DROP TABLE payments;
