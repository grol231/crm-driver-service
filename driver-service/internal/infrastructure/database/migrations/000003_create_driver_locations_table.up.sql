-- Create driver_locations table
CREATE TABLE driver_locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    latitude DECIMAL(10, 7) NOT NULL,
    longitude DECIMAL(10, 7) NOT NULL,
    altitude DECIMAL(8, 2),
    accuracy DECIMAL(8, 2),
    speed DECIMAL(8, 2),
    bearing DECIMAL(6, 2),
    address TEXT,
    metadata JSONB DEFAULT '{}',
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for driver_locations table
CREATE INDEX idx_driver_locations_driver_time ON driver_locations(driver_id, recorded_at DESC);
CREATE INDEX idx_driver_locations_recorded_at ON driver_locations(recorded_at DESC);
CREATE INDEX idx_driver_locations_created_at ON driver_locations(created_at DESC);

-- Create spatial index for geographical queries (PostGIS style without PostGIS)
CREATE INDEX idx_driver_locations_spatial ON driver_locations USING GIST(point(longitude, latitude));

-- Create partial index for recent locations (last 24 hours)
-- Note: Using a static timestamp instead of NOW() for immutable index
CREATE INDEX idx_driver_locations_recent ON driver_locations(driver_id, recorded_at) 
    WHERE recorded_at > CURRENT_DATE - INTERVAL '1 day';

-- Add check constraints for valid coordinates
ALTER TABLE driver_locations ADD CONSTRAINT check_driver_locations_latitude 
    CHECK (latitude >= -90.0 AND latitude <= 90.0);

ALTER TABLE driver_locations ADD CONSTRAINT check_driver_locations_longitude 
    CHECK (longitude >= -180.0 AND longitude <= 180.0);

ALTER TABLE driver_locations ADD CONSTRAINT check_driver_locations_accuracy 
    CHECK (accuracy IS NULL OR accuracy >= 0);

ALTER TABLE driver_locations ADD CONSTRAINT check_driver_locations_speed 
    CHECK (speed IS NULL OR speed >= 0);

ALTER TABLE driver_locations ADD CONSTRAINT check_driver_locations_bearing 
    CHECK (bearing IS NULL OR (bearing >= 0 AND bearing < 360));

-- Create function to clean old location data (older than 30 days)
CREATE OR REPLACE FUNCTION cleanup_old_locations() RETURNS void AS $$
BEGIN
    DELETE FROM driver_locations 
    WHERE recorded_at < NOW() - INTERVAL '30 days';
END;
$$ LANGUAGE plpgsql;

-- Note: Partitioning can be added later for high-volume locations
-- Example partitioning by date:
-- CREATE TABLE driver_locations_202401 PARTITION OF driver_locations
-- FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');