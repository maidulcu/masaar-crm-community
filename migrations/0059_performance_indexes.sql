-- +goose Up
-- +goose StatementBegin
-- Lease filtering by date range (renewal, upcoming overdue)
CREATE INDEX IF NOT EXISTS idx_leases_start_date ON leases(start_date DESC);
-- +goose StatementEnd

-- +goose StatementBegin
-- Payment overdue detection (common dashboard query)
CREATE INDEX IF NOT EXISTS idx_payments_due_date_status ON payments(due_date, status);
-- +goose StatementEnd

-- Audit log time-range queries are already served by idx_audit_ts (0008) on audit_logs(ts DESC).

-- +goose StatementBegin
-- Lead soft-delete queries (most lead queries exclude deleted_at IS NOT NULL)
CREATE INDEX IF NOT EXISTS idx_leads_deleted_at ON leads(deleted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_leases_start_date;
DROP INDEX IF EXISTS idx_payments_due_date_status;
DROP INDEX IF EXISTS idx_leads_deleted_at;
-- +goose StatementEnd
