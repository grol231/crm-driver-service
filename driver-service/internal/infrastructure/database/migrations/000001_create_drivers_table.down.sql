-- Drop triggers
DROP TRIGGER IF EXISTS update_drivers_updated_at ON drivers;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_drivers_phone;
DROP INDEX IF EXISTS idx_drivers_email;
DROP INDEX IF EXISTS idx_drivers_license;
DROP INDEX IF EXISTS idx_drivers_status;
DROP INDEX IF EXISTS idx_drivers_rating;
DROP INDEX IF EXISTS idx_drivers_created_at;
DROP INDEX IF EXISTS idx_drivers_deleted_at;
DROP INDEX IF EXISTS idx_drivers_names;

-- Drop table
DROP TABLE IF EXISTS drivers;