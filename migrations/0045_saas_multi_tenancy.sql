-- +goose Up
-- Masaar CRM Multi-Tenant SaaS: adds company scaffolding, plan/trial fields,
-- and binds users to their company.

-- 1. Expand companies table with SaaS fields
ALTER TABLE companies
    ADD COLUMN IF NOT EXISTS subdomain          VARCHAR(100) UNIQUE,
    ADD COLUMN IF NOT EXISTS plan               VARCHAR(20)  NOT NULL DEFAULT 'starter',
    ADD COLUMN IF NOT EXISTS trial_started_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS trial_ends_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS on_trial           BOOLEAN      NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS stripe_sub_id      VARCHAR(100);

-- 2. Link users to companies
ALTER TABLE users ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id);
CREATE INDEX IF NOT EXISTS idx_users_company_id ON users(company_id);

-- 3. Link company_settings to companies
ALTER TABLE company_settings ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id);

-- 4. Backfill existing seed company (single-tenant migration)
UPDATE companies SET
    plan      = COALESCE((SELECT plan FROM company_settings LIMIT 1), 'community'),
    on_trial  = FALSE,
    is_active = TRUE
WHERE id = '00000000-0000-0000-0000-000000000001';

-- Assign all existing users to the seed company
UPDATE users SET company_id = '00000000-0000-0000-0000-000000000001'
WHERE company_id IS NULL;

-- Assign existing company_settings to the seed company
UPDATE company_settings SET company_id = '00000000-0000-0000-0000-000000000001'
WHERE company_id IS NULL;

-- 5. NOW safe to enforce NOT NULL (backfill completed above)
ALTER TABLE users ALTER COLUMN company_id SET NOT NULL;

-- 6. Remove billing fields from company_settings (moved to companies table)
ALTER TABLE company_settings
    DROP COLUMN IF EXISTS plan,
    DROP COLUMN IF EXISTS stripe_customer_id,
    DROP COLUMN IF EXISTS stripe_sub_id,
    DROP COLUMN IF EXISTS plan_started_at,
    DROP COLUMN IF EXISTS plan_expires_at;

-- +goose Down
-- Revert: restore billing fields to company_settings, drop SaaS columns.

ALTER TABLE company_settings
    ADD COLUMN IF NOT EXISTS plan               VARCHAR(20)  NOT NULL DEFAULT 'community',
    ADD COLUMN IF NOT EXISTS stripe_customer_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS stripe_sub_id      VARCHAR(100),
    ADD COLUMN IF NOT EXISTS plan_started_at    TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS plan_expires_at    TIMESTAMPTZ;

-- Re-populate from companies (best-effort)
UPDATE company_settings SET
    plan = COALESCE((SELECT plan FROM companies WHERE id = '00000000-0000-0000-0000-000000000001'), 'community')
WHERE company_id = '00000000-0000-0000-0000-000000000001';

ALTER TABLE users ALTER COLUMN company_id DROP NOT NULL;
ALTER TABLE company_settings DROP COLUMN IF EXISTS company_id;
DROP INDEX IF EXISTS idx_users_company_id;
ALTER TABLE users DROP COLUMN IF EXISTS company_id;

ALTER TABLE companies
    DROP COLUMN IF EXISTS subdomain,
    DROP COLUMN IF EXISTS plan,
    DROP COLUMN IF EXISTS trial_started_at,
    DROP COLUMN IF EXISTS trial_ends_at,
    DROP COLUMN IF EXISTS on_trial,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS stripe_customer_id,
    DROP COLUMN IF EXISTS stripe_sub_id;
