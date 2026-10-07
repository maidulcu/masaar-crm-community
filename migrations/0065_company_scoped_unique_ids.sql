-- +goose Up
-- Tenant ID numbers and bank transaction ids were unique across the WHOLE database. In a
-- multi-company deployment that meant company A could not register a tenant whose ID number company B
-- already had (and the 409 revealed that B has such a tenant), and two companies importing the
-- same bank's transaction ids collided. Uniqueness is per company. Blank values are not unique keys.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_id_number_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_company_id_number
    ON tenants (company_id, id_number) WHERE id_number IS NOT NULL AND id_number <> '';

ALTER TABLE bank_transactions DROP CONSTRAINT IF EXISTS bank_transactions_external_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_bank_transactions_company_external
    ON bank_transactions (company_id, external_id) WHERE external_id IS NOT NULL AND external_id <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_bank_transactions_company_external;
DROP INDEX IF EXISTS idx_tenants_company_id_number;
-- The global constraints are not restored: they cannot be re-created once two companies share a value.
