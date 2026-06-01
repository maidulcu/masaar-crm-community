-- +goose Up
-- +goose StatementBegin
-- Users: fast lookup by email (login, password reset, magic link)
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
-- +goose StatementEnd

-- +goose StatementBegin
-- Contacts: fast lookup by WhatsApp number (webhook message routing)
CREATE INDEX IF NOT EXISTS idx_contacts_phone_wa ON contacts(phone_wa);
-- +goose StatementEnd

-- +goose StatementBegin
-- WhatsApp threads: filter by status (open/pending/closed inbox views)
CREATE INDEX IF NOT EXISTS idx_wa_threads_status ON whatsapp_threads(thread_status);
-- +goose StatementEnd

-- +goose StatementBegin
-- WhatsApp threads: contact drilldown
CREATE INDEX IF NOT EXISTS idx_wa_threads_contact_id ON whatsapp_threads(contact_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- WhatsApp messages: conversation history (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_wa_messages_thread_id ON whatsapp_messages(thread_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Leads: pipeline board (stage column is the primary kanban filter)
CREATE INDEX IF NOT EXISTS idx_leads_stage ON leads(stage);
-- +goose StatementEnd

-- +goose StatementBegin
-- Leads: contact ownership lookups
CREATE INDEX IF NOT EXISTS idx_leads_contact_id ON leads(contact_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Notifications: per-user unread count (polled frequently)
CREATE INDEX IF NOT EXISTS idx_notifications_user_id_read ON notifications(user_id, read);
-- +goose StatementEnd

-- +goose StatementBegin
-- Payments: lease drilldown (payment history per lease)
CREATE INDEX IF NOT EXISTS idx_payments_lease_id ON payments(lease_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Payments: status filter (overdue detection)
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
-- +goose StatementEnd

-- +goose StatementBegin
-- Leases: property and tenant lookups
CREATE INDEX IF NOT EXISTS idx_leases_property_id ON leases(property_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_leases_tenant_id ON leases(tenant_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Documents: entity scoping (most common document query)
CREATE INDEX IF NOT EXISTS idx_documents_entity ON documents(related_entity_type, related_entity_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Document signatures: lookup by status for dashboard counts
CREATE INDEX IF NOT EXISTS idx_document_signatures_status ON document_signatures(signature_status);
-- +goose StatementEnd

-- +goose StatementBegin
-- Audit logs: user activity trail
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(actor_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_contacts_phone_wa;
DROP INDEX IF EXISTS idx_wa_threads_status;
DROP INDEX IF EXISTS idx_wa_threads_contact_id;
DROP INDEX IF EXISTS idx_wa_messages_thread_id;
DROP INDEX IF EXISTS idx_leads_stage;
DROP INDEX IF EXISTS idx_leads_contact_id;
DROP INDEX IF EXISTS idx_notifications_user_id_read;
DROP INDEX IF EXISTS idx_payments_lease_id;
DROP INDEX IF EXISTS idx_payments_status;
DROP INDEX IF EXISTS idx_leases_property_id;
DROP INDEX IF EXISTS idx_leases_tenant_id;
DROP INDEX IF EXISTS idx_documents_entity;
DROP INDEX IF EXISTS idx_document_signatures_status;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
-- +goose StatementEnd
