-- +goose Up
-- Add cross-references between bank_transactions and payments (circular FK resolved by deferring to this migration)
ALTER TABLE bank_transactions
  ADD COLUMN matched_payment_id UUID REFERENCES payments(id) ON DELETE SET NULL;

ALTER TABLE payments
  ADD COLUMN bank_transaction_id UUID REFERENCES bank_transactions(id) ON DELETE SET NULL;

CREATE INDEX idx_bank_transactions_matched_payment_id ON bank_transactions(matched_payment_id);
CREATE INDEX idx_payments_bank_transaction_id ON payments(bank_transaction_id);

-- +goose Down
ALTER TABLE payments DROP COLUMN IF EXISTS bank_transaction_id;
ALTER TABLE bank_transactions DROP COLUMN IF EXISTS matched_payment_id;
