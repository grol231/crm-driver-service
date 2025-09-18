-- Database initialization script for Docker
-- This script runs when PostgreSQL container starts for the first time

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create database if it doesn't exist (this is handled by POSTGRES_DB env var)
-- But we can add any additional initialization here

-- Set timezone
SET timezone = 'UTC';

-- Grant permissions (if needed)
-- GRANT ALL PRIVILEGES ON DATABASE driver_service TO driver_service;
