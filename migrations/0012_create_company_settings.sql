-- +goose Up
CREATE TABLE company_settings (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  vat_number VARCHAR(50) NOT NULL,
  business_address TEXT NOT NULL,
  business_phone VARCHAR(20),
  business_email VARCHAR(255),
  bank_name VARCHAR(255),
  bank_account VARCHAR(100),
  bank_iban VARCHAR(50),
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Initialize with default values (must be filled by admin)
INSERT INTO company_settings (name, vat_number, business_address)
VALUES ('Your Company Name', '', 'Dubai, United Arab Emirates')
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS company_settings;
