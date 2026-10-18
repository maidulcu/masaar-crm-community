-- +goose Up
-- Money columns get a third decimal place so currencies with 3 minor-unit digits
-- (BHD, KWD, OMR, JOD) are stored exactly. Precision grows by one digit, so every
-- existing value fits and keeps its integer range. Values are unchanged.
-- vat_invoices is left alone: its VAT and total are generated columns rounded to 2
-- decimals, and the invoice model is reworked with explicit currency handling (#84).

ALTER TABLE agent_commissions
    ALTER COLUMN deals_revenue TYPE NUMERIC(16,3),
    ALTER COLUMN leases_revenue TYPE NUMERIC(16,3),
    ALTER COLUMN total_commission TYPE NUMERIC(16,3);
ALTER TABLE agent_targets
    ALTER COLUMN target_value TYPE NUMERIC(15,3);
ALTER TABLE approval_configs
    ALTER COLUMN deal_approval_above TYPE NUMERIC(15,3),
    ALTER COLUMN offer_approval_above TYPE NUMERIC(15,3);
ALTER TABLE bank_transactions
    ALTER COLUMN amount TYPE NUMERIC(13,3);
ALTER TABLE commission_transactions
    ALTER COLUMN transaction_amount TYPE NUMERIC(16,3);
ALTER TABLE deals
    ALTER COLUMN amount TYPE NUMERIC(13,3);
ALTER TABLE expenses
    ALTER COLUMN amount TYPE NUMERIC(16,3);
ALTER TABLE leads
    ALTER COLUMN deal_value TYPE NUMERIC(13,3);
ALTER TABLE lease_renewal_workflows
    ALTER COLUMN proposed_rent_amount TYPE NUMERIC(16,3),
    ALTER COLUMN tenant_counter_offer TYPE NUMERIC(16,3);
ALTER TABLE lease_templates
    ALTER COLUMN default_utility_charges TYPE NUMERIC(13,3);
ALTER TABLE leases
    ALTER COLUMN monthly_rent TYPE NUMERIC(13,3),
    ALTER COLUMN security_deposit TYPE NUMERIC(13,3),
    ALTER COLUMN utility_charges TYPE NUMERIC(13,3);
ALTER TABLE listings
    ALTER COLUMN price TYPE NUMERIC(16,3);
ALTER TABLE maintenance_tasks
    ALTER COLUMN actual_cost TYPE NUMERIC(16,3),
    ALTER COLUMN estimated_cost TYPE NUMERIC(16,3);
ALTER TABLE offers
    ALTER COLUMN offer_amount TYPE NUMERIC(15,3);
ALTER TABLE payments
    ALTER COLUMN amount TYPE NUMERIC(13,3),
    ALTER COLUMN late_fee_amount TYPE NUMERIC(13,3);
ALTER TABLE rental_properties
    ALTER COLUMN market_value TYPE NUMERIC(16,3),
    ALTER COLUMN purchase_price TYPE NUMERIC(16,3);
ALTER TABLE tenants
    ALTER COLUMN annual_income TYPE NUMERIC(16,3);

-- +goose Down
-- Rounds any third decimal away.
ALTER TABLE agent_commissions
    ALTER COLUMN deals_revenue TYPE NUMERIC(15,2),
    ALTER COLUMN leases_revenue TYPE NUMERIC(15,2),
    ALTER COLUMN total_commission TYPE NUMERIC(15,2);
ALTER TABLE agent_targets
    ALTER COLUMN target_value TYPE NUMERIC(14,2);
ALTER TABLE approval_configs
    ALTER COLUMN deal_approval_above TYPE NUMERIC(14,2),
    ALTER COLUMN offer_approval_above TYPE NUMERIC(14,2);
ALTER TABLE bank_transactions
    ALTER COLUMN amount TYPE NUMERIC(12,2);
ALTER TABLE commission_transactions
    ALTER COLUMN transaction_amount TYPE NUMERIC(15,2);
ALTER TABLE deals
    ALTER COLUMN amount TYPE NUMERIC(12,2);
ALTER TABLE expenses
    ALTER COLUMN amount TYPE NUMERIC(15,2);
ALTER TABLE leads
    ALTER COLUMN deal_value TYPE NUMERIC(12,2);
ALTER TABLE lease_renewal_workflows
    ALTER COLUMN proposed_rent_amount TYPE NUMERIC(15,2),
    ALTER COLUMN tenant_counter_offer TYPE NUMERIC(15,2);
ALTER TABLE lease_templates
    ALTER COLUMN default_utility_charges TYPE NUMERIC(12,2);
ALTER TABLE leases
    ALTER COLUMN monthly_rent TYPE NUMERIC(12,2),
    ALTER COLUMN security_deposit TYPE NUMERIC(12,2),
    ALTER COLUMN utility_charges TYPE NUMERIC(12,2);
ALTER TABLE listings
    ALTER COLUMN price TYPE NUMERIC(15,2);
ALTER TABLE maintenance_tasks
    ALTER COLUMN actual_cost TYPE NUMERIC(15,2),
    ALTER COLUMN estimated_cost TYPE NUMERIC(15,2);
ALTER TABLE offers
    ALTER COLUMN offer_amount TYPE NUMERIC(14,2);
ALTER TABLE payments
    ALTER COLUMN amount TYPE NUMERIC(12,2),
    ALTER COLUMN late_fee_amount TYPE NUMERIC(12,2);
ALTER TABLE rental_properties
    ALTER COLUMN market_value TYPE NUMERIC(15,2),
    ALTER COLUMN purchase_price TYPE NUMERIC(15,2);
ALTER TABLE tenants
    ALTER COLUMN annual_income TYPE NUMERIC(15,2);
