-- +goose Up
CREATE TABLE listings (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,

  -- Core info
  title VARCHAR(255) NOT NULL,
  description TEXT,
  property_type VARCHAR(50) NOT NULL,
  listing_type VARCHAR(20) NOT NULL DEFAULT 'rent',

  -- Pricing
  price DECIMAL(15, 2) NOT NULL,
  currency VARCHAR(3) DEFAULT 'AED',
  rent_period VARCHAR(20),

  -- Location
  area VARCHAR(100),
  community VARCHAR(100),
  subcommunity VARCHAR(100),
  city VARCHAR(100),
  emirate VARCHAR(50),
  latitude DECIMAL(10, 7),
  longitude DECIMAL(10, 7),

  -- Specs
  bedrooms INTEGER,
  bathrooms INTEGER,
  total_sqft DECIMAL(12, 2),
  plot_sqft DECIMAL(12, 2),
  parking_spaces INTEGER DEFAULT 0,
  furnishing VARCHAR(50),
  amenities TEXT[],
  year_built INTEGER,

  -- Media
  cover_image_url VARCHAR(500),
  image_urls TEXT[],
  virtual_tour_url VARCHAR(500),
  video_url VARCHAR(500),

  -- Status & flags
  status VARCHAR(50) NOT NULL DEFAULT 'draft',
  featured BOOLEAN NOT NULL DEFAULT FALSE,
  reference_number VARCHAR(100),
  available_from DATE,

  -- Agent assignment
  assigned_to UUID REFERENCES users(id),

  -- Client / owner
  owner_name VARCHAR(255),
  owner_phone VARCHAR(50),
  owner_email VARCHAR(255),

  -- Portal sync status (for Phase 2)
  portal_sync_status JSONB NOT NULL DEFAULT '{}',

  -- Metadata
  published_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  created_by UUID REFERENCES users(id),
  updated_by UUID REFERENCES users(id)
);

CREATE INDEX idx_listings_company_id ON listings(company_id);
CREATE INDEX idx_listings_status ON listings(status);
CREATE INDEX idx_listings_property_type ON listings(property_type);
CREATE INDEX idx_listings_assigned_to ON listings(assigned_to);
CREATE INDEX idx_listings_area ON listings(area);
CREATE INDEX idx_listings_emirate ON listings(emirate);
CREATE INDEX idx_listings_featured ON listings(featured) WHERE featured = TRUE;

-- +goose Down
DROP TABLE listings;
