-- Create driver_documents table
CREATE TABLE driver_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    document_type VARCHAR(50) NOT NULL,
    document_number VARCHAR(100) NOT NULL,
    issue_date DATE NOT NULL,
    expiry_date DATE NOT NULL,
    file_url VARCHAR(500) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    verified_by VARCHAR(255),
    verified_at TIMESTAMP WITH TIME ZONE,
    rejection_reason TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for driver_documents table
CREATE INDEX idx_driver_documents_driver_id ON driver_documents(driver_id);
CREATE INDEX idx_driver_documents_type ON driver_documents(document_type);
CREATE INDEX idx_driver_documents_status ON driver_documents(status);
CREATE INDEX idx_driver_documents_expiry ON driver_documents(expiry_date);
CREATE INDEX idx_driver_documents_created_at ON driver_documents(created_at);

-- Create unique index to prevent duplicate document types per driver
CREATE UNIQUE INDEX idx_driver_documents_unique_type ON driver_documents(driver_id, document_type);

-- Add check constraints
ALTER TABLE driver_documents ADD CONSTRAINT check_driver_documents_status 
    CHECK (status IN ('pending', 'verified', 'rejected', 'expired', 'processing'));

ALTER TABLE driver_documents ADD CONSTRAINT check_driver_documents_type 
    CHECK (document_type IN ('driver_license', 'medical_certificate', 'vehicle_registration', 'insurance', 'passport', 'taxi_permit', 'work_permit'));

ALTER TABLE driver_documents ADD CONSTRAINT check_driver_documents_dates 
    CHECK (expiry_date > issue_date);

-- Create trigger for updated_at
CREATE TRIGGER update_driver_documents_updated_at BEFORE UPDATE ON driver_documents 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();