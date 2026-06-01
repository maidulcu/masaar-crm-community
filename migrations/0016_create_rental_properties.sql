-- +goose Up
CREATE TABLE rental_properties (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

  -- Property details
  name VARCHAR(255) NOT NULL,
  description TEXT,
  property_type VARCHAR(50) NOT NULL, -- villa, apartment, townhouse, commercial
  units_count INTEGER DEFAULT 1,

  -- Location
  area VARCHAR(100), -- Dubai Marina, Downtown Dubai, etc
  street_address VARCHAR(255),
  building_number VARCHAR(50),
  unit_number VARCHAR(50),
  city VARCHAR(100),
  emirate VARCHAR(50),
  postal_code VARCHAR(20),

  -- Specifications
  total_sqft DECIMAL(12, 2),
  bedrooms INTEGER,
  bathrooms INTEGER,
  parking_spaces INTEGER,
  amenities TEXT[], -- pool, gym, security, etc

  -- Financial
  purchase_price DECIMAL(15, 2),
  purchase_date DATE,
  market_value DECIMAL(15, 2),
  currency VARCHAR(3) DEFAULT 'AED',

  -- Status
  status VARCHAR(50) DEFAULT 'active', -- active, inactive, sold, maintenance
  occupancy_status VARCHAR(50) DEFAULT 'vacant', -- vacant, occupied, maintenance
  total_occupied_units INTEGER DEFAULT 0,

  -- Documents
  property_deed_url VARCHAR(500),
  title_deed_number VARCHAR(100),
  municipality_registration VARCHAR(100),

  -- Metadata
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id),
  updated_by UUID REFERENCES users(id),

  CONSTRAINT check_units_occupied CHECK (total_occupied_units <= units_count)
);

CREATE INDEX idx_rental_properties_company_id ON rental_properties(company_id);
CREATE INDEX idx_rental_properties_area ON rental_properties(area);
CREATE INDEX idx_rental_properties_status ON rental_properties(status);
CREATE INDEX idx_rental_properties_emirate ON rental_properties(emirate);

-- +goose Down
DROP TABLE rental_properties;
