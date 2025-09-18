-- Drop functions
DROP FUNCTION IF EXISTS cleanup_old_locations();

-- Drop indexes
DROP INDEX IF EXISTS idx_driver_locations_driver_time;
DROP INDEX IF EXISTS idx_driver_locations_recorded_at;
DROP INDEX IF EXISTS idx_driver_locations_created_at;
DROP INDEX IF EXISTS idx_driver_locations_spatial;
DROP INDEX IF EXISTS idx_driver_locations_recent;

-- Drop table
DROP TABLE IF EXISTS driver_locations;