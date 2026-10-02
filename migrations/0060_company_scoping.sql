-- +goose Up
-- +goose StatementBegin
-- Company scoping for the core CRM tables. Before this migration contacts, leads,
-- deals, invoices, WhatsApp, notifications, email history, viewings, offers,
-- communication history, audit logs and api_settings were shared by every company.
--
-- Existing rows are attributed to a company through their parent/owner where
-- possible; otherwise to the only company that has users, else the seed company.

ALTER TABLE contacts              ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE leads                 ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE deals                 ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE vat_invoices          ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE notifications         ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE email_history         ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE viewings              ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE offers                ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE whatsapp_threads      ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE whatsapp_outbound     ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE communication_history ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE audit_logs            ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;
ALTER TABLE api_settings          ADD COLUMN IF NOT EXISTS company_id UUID REFERENCES companies(id) ON DELETE CASCADE;

DO $$
DECLARE
    def UUID;
BEGIN
    IF (SELECT COUNT(DISTINCT company_id) FROM users) = 1 THEN
        SELECT DISTINCT company_id INTO def FROM users;
    ELSE
        def := '00000000-0000-0000-0000-000000000001';
    END IF;

    UPDATE contacts c SET company_id = COALESCE((SELECT u.company_id FROM users u WHERE u.id = c.assigned_to), def)
        WHERE c.company_id IS NULL;
    UPDATE leads l SET company_id = COALESCE(
            (SELECT c.company_id FROM contacts c WHERE c.id = l.contact_id),
            (SELECT u.company_id FROM users u WHERE u.id = l.assigned_to), def)
        WHERE l.company_id IS NULL;
    UPDATE whatsapp_threads t SET company_id = COALESCE((SELECT c.company_id FROM contacts c WHERE c.id = t.contact_id), def)
        WHERE t.company_id IS NULL;
    UPDATE deals d SET company_id = COALESCE(
            (SELECT l.company_id FROM leads l WHERE l.id = d.lead_id),
            (SELECT u.company_id FROM users u WHERE u.id = d.owner_id), def)
        WHERE d.company_id IS NULL;
    UPDATE vat_invoices v SET company_id = COALESCE((SELECT d.company_id FROM deals d WHERE d.id = v.deal_id), def)
        WHERE v.company_id IS NULL;
    UPDATE notifications n SET company_id = COALESCE((SELECT u.company_id FROM users u WHERE u.id = n.user_id), def)
        WHERE n.company_id IS NULL;
    UPDATE email_history e SET company_id = COALESCE((SELECT u.company_id FROM users u WHERE u.id = e.created_by), def)
        WHERE e.company_id IS NULL;
    UPDATE viewings v SET company_id = COALESCE(
            (SELECT u.company_id FROM users u WHERE u.id = v.agent_id),
            (SELECT c.company_id FROM contacts c WHERE c.id = v.contact_id), def)
        WHERE v.company_id IS NULL;
    UPDATE offers o SET company_id = COALESCE(
            (SELECT u.company_id FROM users u WHERE u.id = o.agent_id),
            (SELECT c.company_id FROM contacts c WHERE c.id = o.contact_id), def)
        WHERE o.company_id IS NULL;
    UPDATE whatsapp_outbound w SET company_id = COALESCE(
            (SELECT t.company_id FROM whatsapp_threads t WHERE t.id = w.thread_id),
            (SELECT u.company_id FROM users u WHERE u.id = w.created_by), def)
        WHERE w.company_id IS NULL;
    UPDATE communication_history h SET company_id = COALESCE(
            (SELECT l.company_id FROM leads l WHERE l.id = h.lead_id),
            (SELECT c.company_id FROM contacts c WHERE c.id = h.contact_id), def)
        WHERE h.company_id IS NULL;
    UPDATE audit_logs a SET company_id = COALESCE((SELECT u.company_id FROM users u WHERE u.id = a.actor_id), def)
        WHERE a.company_id IS NULL;
    UPDATE api_settings SET company_id = def WHERE company_id IS NULL;
END $$;

ALTER TABLE contacts              ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE leads                 ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE deals                 ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE vat_invoices          ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE notifications         ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE email_history         ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE viewings              ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE offers                ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE whatsapp_threads      ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE whatsapp_outbound     ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE communication_history ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE audit_logs            ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE api_settings          ALTER COLUMN company_id SET NOT NULL;

-- A phone number / setting key is unique per company, not globally.
ALTER TABLE contacts     DROP CONSTRAINT IF EXISTS contacts_phone_wa_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_company_phone ON contacts(company_id, phone_wa);
ALTER TABLE vat_invoices DROP CONSTRAINT IF EXISTS vat_invoices_invoice_no_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_vat_invoices_company_no ON vat_invoices(company_id, invoice_no);
ALTER TABLE api_settings DROP CONSTRAINT IF EXISTS api_settings_setting_key_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_settings_company_key ON api_settings(company_id, setting_key);

CREATE INDEX IF NOT EXISTS idx_leads_company              ON leads(company_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_deals_company              ON deals(company_id);
CREATE INDEX IF NOT EXISTS idx_vat_invoices_company       ON vat_invoices(company_id);
CREATE INDEX IF NOT EXISTS idx_notifications_company_user ON notifications(company_id, user_id);
CREATE INDEX IF NOT EXISTS idx_email_history_company      ON email_history(company_id);
CREATE INDEX IF NOT EXISTS idx_viewings_company           ON viewings(company_id);
CREATE INDEX IF NOT EXISTS idx_offers_company             ON offers(company_id);
CREATE INDEX IF NOT EXISTS idx_wa_threads_company         ON whatsapp_threads(company_id);
CREATE INDEX IF NOT EXISTS idx_wa_outbound_company        ON whatsapp_outbound(company_id);
CREATE INDEX IF NOT EXISTS idx_comm_history_company       ON communication_history(company_id);
CREATE INDEX IF NOT EXISTS idx_audit_company_ts           ON audit_logs(company_id, ts DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_audit_company_ts, idx_comm_history_company, idx_wa_outbound_company, idx_wa_threads_company,
    idx_offers_company, idx_viewings_company, idx_email_history_company, idx_notifications_company_user,
    idx_vat_invoices_company_no, idx_vat_invoices_company, idx_deals_company, idx_leads_company, idx_api_settings_company_key, idx_contacts_company_phone;
ALTER TABLE contacts     ADD CONSTRAINT contacts_phone_wa_key UNIQUE (phone_wa);
ALTER TABLE vat_invoices ADD CONSTRAINT vat_invoices_invoice_no_key UNIQUE (invoice_no);
ALTER TABLE api_settings ADD CONSTRAINT api_settings_setting_key_key UNIQUE (setting_key);
ALTER TABLE contacts DROP COLUMN IF EXISTS company_id;
ALTER TABLE leads DROP COLUMN IF EXISTS company_id;
ALTER TABLE deals DROP COLUMN IF EXISTS company_id;
ALTER TABLE vat_invoices DROP COLUMN IF EXISTS company_id;
ALTER TABLE notifications DROP COLUMN IF EXISTS company_id;
ALTER TABLE email_history DROP COLUMN IF EXISTS company_id;
ALTER TABLE viewings DROP COLUMN IF EXISTS company_id;
ALTER TABLE offers DROP COLUMN IF EXISTS company_id;
ALTER TABLE whatsapp_threads DROP COLUMN IF EXISTS company_id;
ALTER TABLE whatsapp_outbound DROP COLUMN IF EXISTS company_id;
ALTER TABLE communication_history DROP COLUMN IF EXISTS company_id;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS company_id;
ALTER TABLE api_settings DROP COLUMN IF EXISTS company_id;
-- +goose StatementEnd
