-- +goose Up
CREATE TABLE bank_statements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    bank_integration_id UUID NOT NULL REFERENCES bank_integrations(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    file_size_bytes INTEGER NOT NULL,
    file_url VARCHAR(2048) NOT NULL,
    file_format VARCHAR(20) NOT NULL CHECK (file_format IN ('csv', 'pdf', 'xlsx')),
    uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    upload_date TIMESTAMP WITH TIME ZONE NOT NULL,
    processing_status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (processing_status IN ('pending', 'processing', 'completed', 'failed')),
    transactions_imported INTEGER DEFAULT 0,
    import_error TEXT,
    data_classification VARCHAR(50) NOT NULL DEFAULT 'confidential' CHECK (data_classification IN ('public', 'internal', 'confidential')),
    retention_until DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_bank_statements_company_id ON bank_statements(company_id);
CREATE INDEX idx_bank_statements_bank_integration_id ON bank_statements(bank_integration_id);
CREATE INDEX idx_bank_statements_upload_date ON bank_statements(upload_date DESC);
CREATE INDEX idx_bank_statements_processing_status ON bank_statements(processing_status);

CREATE TABLE payment_confirmations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    confirmation_number VARCHAR(50) NOT NULL UNIQUE,
    tenant_email VARCHAR(255) NOT NULL,
    tenant_phone VARCHAR(20),
    sent_at TIMESTAMP WITH TIME ZONE,
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (delivery_status IN ('pending', 'sent', 'failed', 'bounced')),
    delivery_method VARCHAR(20) NOT NULL DEFAULT 'email' CHECK (delivery_method IN ('email', 'whatsapp', 'sms')),
    pdf_url VARCHAR(2048),
    data_classification VARCHAR(50) NOT NULL DEFAULT 'confidential' CHECK (data_classification IN ('public', 'internal', 'confidential')),
    retention_until DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_payment_confirmations_company_id ON payment_confirmations(company_id);
CREATE INDEX idx_payment_confirmations_payment_id ON payment_confirmations(payment_id);
CREATE INDEX idx_payment_confirmations_sent_at ON payment_confirmations(sent_at DESC);
CREATE INDEX idx_payment_confirmations_delivery_status ON payment_confirmations(delivery_status);

-- +goose Down
DROP TABLE IF EXISTS payment_confirmations;
DROP TABLE IF EXISTS bank_statements;
