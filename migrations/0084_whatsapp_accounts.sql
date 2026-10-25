-- +goose Up

-- WhatsApp accounts are per company. Today the connection is deployment-wide: every thread in
-- a company is keyed by Meta's phone_number_id from the environment, so a second company cannot
-- have its own number. This table makes the account an entity, which the official (Cloud API /
-- BSP) path and the optional community gateway can both use.
CREATE TABLE IF NOT EXISTS wa_accounts (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           UUID        NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    provider             TEXT        NOT NULL CHECK (provider IN ('cloud_api','bsp','waha')),
    -- For provider='bsp': which partner, and which of its clients this maps to.
    bsp_slug             TEXT,
    bsp_client_ref       TEXT,
    status               TEXT        NOT NULL CHECK (status IN ('pending','linking','connected','degraded','logged_out','banned')),
    phone_e164           TEXT,
    display_name         TEXT,
    -- Meta identifiers. Keep all three: the account model is mid-migration.
    waba_id              TEXT,       -- now a Messaging account ID
    messaging_account_id TEXT,       -- required for multi-account sends from H1 2027
    waac_id              TEXT,       -- replaces phone_number_id in paths by H1 2028
    phone_number_id      TEXT,       -- legacy identifier; never the account key
    external_id          TEXT,       -- provider-side account or session identifier
    session_ref          TEXT,       -- gateway session id, if any
    billing_mode         TEXT        CHECK (billing_mode IN ('client_card','masaar_credit_line')),
    linked_by            UUID        REFERENCES users(id) ON DELETE SET NULL,
    linked_at            TIMESTAMPTZ,
    last_seen_at         TIMESTAMPTZ,
    last_error           TEXT        NOT NULL DEFAULT '',
    risk_flags           JSONB       NOT NULL DEFAULT '{}',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wa_accounts_company ON wa_accounts(company_id, created_at DESC);

-- Legacy data can contain the same deployment-wide number under more than one company, so
-- uniqueness is per company. Inbound routing resolves within the company from APP_COMPANY_ID;
-- official per-company numbers are unique in practice anyway.
CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_accounts_company_phone_number
    ON wa_accounts(company_id, phone_number_id) WHERE phone_number_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_accounts_company_phone
    ON wa_accounts(company_id, phone_e164) WHERE phone_e164 IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_wa_accounts_company_external
    ON wa_accounts(company_id, provider, external_id) WHERE external_id IS NOT NULL;

-- Threads are re-parented to an account but keep wa_account_id (the legacy Meta phone number id)
-- during the transition, so older code paths and existing data remain readable.
ALTER TABLE whatsapp_threads
    ADD COLUMN IF NOT EXISTS account_id UUID REFERENCES wa_accounts(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_threads_account ON whatsapp_threads(account_id);

-- Backfill one account per company + Meta phone number id found on existing threads. The
-- deployment-wide configuration is also ensured at startup, for deployments with no threads yet.
INSERT INTO wa_accounts (company_id, provider, status, phone_number_id, linked_at, last_seen_at, created_at, updated_at)
SELECT t.company_id,
       'cloud_api',
       'connected',
       NULLIF(t.wa_account_id, ''),
       MIN(t.created_at),
       MAX(t.last_message_at),
       MIN(t.created_at),
       NOW()
FROM whatsapp_threads t
WHERE t.company_id IS NOT NULL
  AND t.wa_account_id <> ''
GROUP BY t.company_id, t.wa_account_id
ON CONFLICT DO NOTHING;

UPDATE whatsapp_threads t
SET account_id = a.id
FROM wa_accounts a
WHERE t.account_id IS NULL
  AND a.company_id = t.company_id
  AND a.provider = 'cloud_api'
  AND a.phone_number_id = t.wa_account_id;

-- +goose Down
DROP INDEX IF EXISTS idx_threads_account;
ALTER TABLE whatsapp_threads DROP COLUMN IF EXISTS account_id;
DROP TABLE IF EXISTS wa_accounts;
