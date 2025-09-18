-- Drop functions
DROP FUNCTION IF EXISTS calculate_shift_duration(TIMESTAMP WITH TIME ZONE, TIMESTAMP WITH TIME ZONE);

-- Drop triggers
DROP TRIGGER IF EXISTS update_driver_shifts_updated_at ON driver_shifts;

-- Drop indexes
DROP INDEX IF EXISTS idx_driver_shifts_driver_id;
DROP INDEX IF EXISTS idx_driver_shifts_vehicle_id;
DROP INDEX IF EXISTS idx_driver_shifts_status;
DROP INDEX IF EXISTS idx_driver_shifts_start_time;
DROP INDEX IF EXISTS idx_driver_shifts_end_time;
DROP INDEX IF EXISTS idx_driver_shifts_created_at;
DROP INDEX IF EXISTS idx_driver_shifts_driver_time;
DROP INDEX IF EXISTS idx_driver_shifts_active;

-- Drop table
DROP TABLE IF EXISTS driver_shifts;