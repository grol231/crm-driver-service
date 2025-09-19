-- Create driver_shifts table
CREATE TABLE driver_shifts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    vehicle_id UUID, -- Reference to vehicle (external service)
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    start_latitude DECIMAL(10, 7),
    start_longitude DECIMAL(10, 7),
    end_latitude DECIMAL(10, 7),
    end_longitude DECIMAL(10, 7),
    total_trips INTEGER NOT NULL DEFAULT 0,
    total_distance DECIMAL(10, 2) NOT NULL DEFAULT 0.0,
    total_earnings DECIMAL(10, 2) NOT NULL DEFAULT 0.0,
    fuel_consumed DECIMAL(8, 2),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for driver_shifts table
CREATE INDEX idx_driver_shifts_driver_id ON driver_shifts(driver_id);
CREATE INDEX idx_driver_shifts_vehicle_id ON driver_shifts(vehicle_id);
CREATE INDEX idx_driver_shifts_status ON driver_shifts(status);
CREATE INDEX idx_driver_shifts_start_time ON driver_shifts(start_time DESC);
CREATE INDEX idx_driver_shifts_end_time ON driver_shifts(end_time DESC);
CREATE INDEX idx_driver_shifts_created_at ON driver_shifts(created_at DESC);

-- Create composite index for driver shifts by time period
CREATE INDEX idx_driver_shifts_driver_time ON driver_shifts(driver_id, start_time DESC, end_time DESC);

-- Create partial index for active shifts
CREATE UNIQUE INDEX idx_driver_shifts_active ON driver_shifts(driver_id) 
    WHERE status = 'active' AND end_time IS NULL;

-- Add check constraints
ALTER TABLE driver_shifts ADD CONSTRAINT check_driver_shifts_status 
    CHECK (status IN ('active', 'completed', 'suspended', 'cancelled'));

ALTER TABLE driver_shifts ADD CONSTRAINT check_driver_shifts_times 
    CHECK (end_time IS NULL OR end_time > start_time);

ALTER TABLE driver_shifts ADD CONSTRAINT check_driver_shifts_totals 
    CHECK (total_trips >= 0 AND total_distance >= 0.0 AND total_earnings >= 0.0);

ALTER TABLE driver_shifts ADD CONSTRAINT check_driver_shifts_coordinates 
    CHECK (
        (start_latitude IS NULL AND start_longitude IS NULL) OR
        (start_latitude IS NOT NULL AND start_longitude IS NOT NULL AND
         start_latitude >= -90.0 AND start_latitude <= 90.0 AND
         start_longitude >= -180.0 AND start_longitude <= 180.0)
    );

ALTER TABLE driver_shifts ADD CONSTRAINT check_driver_shifts_end_coordinates 
    CHECK (
        (end_latitude IS NULL AND end_longitude IS NULL) OR
        (end_latitude IS NOT NULL AND end_longitude IS NOT NULL AND
         end_latitude >= -90.0 AND end_latitude <= 90.0 AND
         end_longitude >= -180.0 AND end_longitude <= 180.0)
    );

-- Create trigger for updated_at
CREATE TRIGGER update_driver_shifts_updated_at BEFORE UPDATE ON driver_shifts 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to calculate shift duration
CREATE OR REPLACE FUNCTION calculate_shift_duration(shift_start TIMESTAMP WITH TIME ZONE, shift_end TIMESTAMP WITH TIME ZONE DEFAULT NULL)
RETURNS INTEGER AS $$
BEGIN
    IF shift_end IS NULL THEN
        RETURN EXTRACT(EPOCH FROM (NOW() - shift_start))::INTEGER / 60; -- minutes
    ELSE
        RETURN EXTRACT(EPOCH FROM (shift_end - shift_start))::INTEGER / 60; -- minutes
    END IF;
END;
$$ LANGUAGE plpgsql;