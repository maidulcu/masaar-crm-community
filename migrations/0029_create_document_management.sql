-- +goose Up
CREATE TABLE document_templates (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    template_name VARCHAR(150) NOT NULL,
    document_type VARCHAR(50),
    template_content TEXT,
    language VARCHAR(2),
    signature_required BOOLEAN DEFAULT false,
    signature_fields JSONB,
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, template_name, language)
);

CREATE TABLE documents (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    document_type VARCHAR(100),
    original_template_id UUID,
    related_entity_type VARCHAR(50),
    related_entity_id UUID,
    document_title VARCHAR(255),
    file_url VARCHAR(500),
    file_size_bytes INT,
    content_hash VARCHAR(64),
    signature_status VARCHAR(50),
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_classification VARCHAR(20),
    retention_until DATE,
    deleted_at TIMESTAMP
);

CREATE TABLE document_signatures (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL,
    signer_name VARCHAR(150),
    signer_email VARCHAR(255),
    signature_field_name VARCHAR(100),
    signature_status VARCHAR(50),
    signed_at TIMESTAMP,
    signature_image_url VARCHAR(500),
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE document_audit_log (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL,
    action VARCHAR(100),
    actor_id UUID,
    actor_name VARCHAR(150),
    old_values JSONB,
    new_values JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_document_templates_company ON document_templates(company_id);
CREATE INDEX idx_documents_company_type ON documents(company_id, document_type);
CREATE INDEX idx_documents_entity ON documents(related_entity_type, related_entity_id);
CREATE INDEX idx_document_signatures_document ON document_signatures(document_id);
CREATE INDEX idx_document_audit_document ON document_audit_log(document_id);

-- +goose Down
DROP TABLE IF EXISTS document_audit_log;
DROP TABLE IF EXISTS document_signatures;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS document_templates;
