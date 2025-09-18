-- Drop triggers
DROP TRIGGER IF EXISTS update_driver_documents_updated_at ON driver_documents;

-- Drop indexes
DROP INDEX IF EXISTS idx_driver_documents_driver_id;
DROP INDEX IF EXISTS idx_driver_documents_type;
DROP INDEX IF EXISTS idx_driver_documents_status;
DROP INDEX IF EXISTS idx_driver_documents_expiry;
DROP INDEX IF EXISTS idx_driver_documents_created_at;
DROP INDEX IF EXISTS idx_driver_documents_unique_type;

-- Drop table
DROP TABLE IF EXISTS driver_documents;