-- Create drivers table
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE drivers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone VARCHAR(20) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    birth_date DATE NOT NULL,
    passport_series VARCHAR(10) NOT NULL,
    passport_number VARCHAR(20) NOT NULL,
    license_number VARCHAR(50) NOT NULL UNIQUE,
    license_expiry DATE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'registered',
    current_rating DECIMAL(3,2) NOT NULL DEFAULT 0.0,
    total_trips INTEGER NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for drivers table
CREATE INDEX idx_drivers_phone ON drivers(phone);
CREATE INDEX idx_drivers_email ON drivers(email);
CREATE INDEX idx_drivers_license ON drivers(license_number);
CREATE INDEX idx_drivers_status ON drivers(status);
CREATE INDEX idx_drivers_rating ON drivers(current_rating);
CREATE INDEX idx_drivers_created_at ON drivers(created_at);
CREATE INDEX idx_drivers_deleted_at ON drivers(deleted_at) WHERE deleted_at IS NULL;

-- Create index for full-text search on names
CREATE INDEX idx_drivers_names ON drivers USING gin(to_tsvector('russian', first_name || ' ' || last_name || ' ' || COALESCE(middle_name, '')));

-- Add check constraints
ALTER TABLE drivers ADD CONSTRAINT check_drivers_rating CHECK (current_rating >= 0.0 AND current_rating <= 5.0);
ALTER TABLE drivers ADD CONSTRAINT check_drivers_total_trips CHECK (total_trips >= 0);
ALTER TABLE drivers ADD CONSTRAINT check_drivers_status CHECK (status IN ('registered', 'pending_verification', 'verified', 'rejected', 'available', 'on_shift', 'busy', 'inactive', 'suspended', 'blocked'));

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for updated_at
CREATE TRIGGER update_drivers_updated_at BEFORE UPDATE ON drivers 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();