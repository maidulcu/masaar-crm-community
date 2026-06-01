-- +goose Up
CREATE TABLE custom_field_definitions (
    id UUID PRIMARY KEY,
    company_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    field_label VARCHAR(150),
    field_type VARCHAR(50),
    is_required BOOLEAN DEFAULT false,
    display_order INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(company_id, entity_type, field_name)
);

CREATE TABLE custom_field_values (
    id UUID PRIMARY KEY,
    entity_id UUID NOT NULL,
    entity_type VARCHAR(50),
    field_id UUID NOT NULL,
    value TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_custom_field_defs_company_entity ON custom_field_definitions(company_id, entity_type);
CREATE INDEX idx_custom_field_values_entity ON custom_field_values(entity_id, entity_type);

-- +goose Down
DROP TABLE IF EXISTS custom_field_values;
DROP TABLE IF EXISTS custom_field_definitions;
