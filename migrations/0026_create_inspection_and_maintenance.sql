-- +goose Up
CREATE TABLE inspection_templates (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    template_name VARCHAR(150) NOT NULL,
    inspection_type VARCHAR(50) NOT NULL,
    checklist_items JSONB,
    estimated_duration_minutes INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, template_name)
);

CREATE TABLE inspections (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    property_id UUID NOT NULL,
    template_id UUID,
    inspection_type VARCHAR(50),
    scheduled_date TIMESTAMP NOT NULL,
    completed_date TIMESTAMP,
    inspector_id UUID,
    tenant_id UUID,
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    findings TEXT,
    severity_level VARCHAR(20),
    photos_urls TEXT[],
    checklist_results JSONB,
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE maintenance_tasks (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    property_id UUID NOT NULL,
    inspection_id UUID,
    maintenance_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL,
    scheduled_date DATE,
    due_date DATE,
    completion_date DATE,
    contractor_name VARCHAR(150),
    contractor_contact VARCHAR(255),
    estimated_cost DECIMAL(15,2),
    actual_cost DECIMAL(15,2),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    assigned_to UUID,
    notes TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE maintenance_photos (
    id UUID PRIMARY KEY,
    task_id UUID NOT NULL,
    photo_url VARCHAR(500) NOT NULL,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    photo_stage VARCHAR(20)
);

CREATE INDEX idx_inspection_templates_company ON inspection_templates(company_id);
CREATE INDEX idx_inspections_property_date ON inspections(property_id, scheduled_date);
CREATE INDEX idx_inspections_status ON inspections(company_id, status);
CREATE INDEX idx_maintenance_property_status ON maintenance_tasks(property_id, status);
CREATE INDEX idx_maintenance_company ON maintenance_tasks(company_id);
CREATE INDEX idx_maintenance_photos_task ON maintenance_photos(task_id);

-- +goose Down
DROP TABLE IF EXISTS maintenance_photos;
DROP TABLE IF EXISTS maintenance_tasks;
DROP TABLE IF EXISTS inspections;
DROP TABLE IF EXISTS inspection_templates;
