-- +goose Up
CREATE TABLE bulk_import_jobs (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    entity_type VARCHAR(50),
    job_status VARCHAR(50),
    file_url VARCHAR(500),
    total_rows INT,
    processed_rows INT,
    failed_rows INT,
    error_message TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE TABLE bulk_import_errors (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL,
    row_number INT,
    error_message TEXT,
    data_snapshot JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE bulk_export_jobs (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    entity_type VARCHAR(50),
    export_format VARCHAR(10),
    filters JSONB,
    file_url VARCHAR(500),
    status VARCHAR(50),
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_bulk_import_jobs_company ON bulk_import_jobs(company_id);
CREATE INDEX idx_bulk_import_jobs_status ON bulk_import_jobs(job_status);
CREATE INDEX idx_bulk_import_errors_job ON bulk_import_errors(job_id);
CREATE INDEX idx_bulk_export_jobs_company ON bulk_export_jobs(company_id);

-- +goose Down
DROP TABLE IF EXISTS bulk_import_errors;
DROP TABLE IF EXISTS bulk_import_jobs;
DROP TABLE IF EXISTS bulk_export_jobs;
