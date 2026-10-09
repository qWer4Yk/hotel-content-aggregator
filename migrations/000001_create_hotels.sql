-- 000001_create_hotels.sql
-- Hotel catalog schema with indexing for fast city search and deduplication

CREATE TABLE IF NOT EXISTS hotels (
    id VARCHAR(64) PRIMARY KEY,
    supplier_id VARCHAR(64) NOT NULL,
    external_id VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    city VARCHAR(128) NOT NULL,
    address TEXT NOT NULL DEFAULT '',
    latitude DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    stars SMALLINT NOT NULL DEFAULT 0 CHECK (stars >= 0 AND stars <= 5),
    rating DOUBLE PRECISION NOT NULL DEFAULT 0.0 CHECK (rating >= 0.0 AND rating <= 10.0),
    description TEXT NOT NULL DEFAULT '',
    amenities JSONB NOT NULL DEFAULT '[]',
    photos JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique constraint per supplier feed to avoid redundant records
CREATE UNIQUE INDEX IF NOT EXISTS uq_hotels_supplier_key 
    ON hotels (supplier_id, external_id);

-- Composite index for fast search queries by city ordered by rating
CREATE INDEX IF NOT EXISTS idx_hotels_city_rating 
    ON hotels (city, rating DESC);

-- Index for deduplication lookup by normalized name and city
CREATE INDEX IF NOT EXISTS idx_hotels_dedup 
    ON hotels (normalized_name, city);
