-- Drop triggers
DROP TRIGGER IF EXISTS trigger_driver_rating_stats_update ON driver_ratings;
DROP TRIGGER IF EXISTS update_driver_rating_stats_updated_at ON driver_rating_stats;
DROP TRIGGER IF EXISTS update_driver_ratings_updated_at ON driver_ratings;

-- Drop functions
DROP FUNCTION IF EXISTS trigger_update_rating_stats();
DROP FUNCTION IF EXISTS update_driver_rating_stats(UUID);

-- Drop indexes for rating_stats
DROP INDEX IF EXISTS idx_driver_rating_stats_average;
DROP INDEX IF EXISTS idx_driver_rating_stats_total;
DROP INDEX IF EXISTS idx_driver_rating_stats_updated;

-- Drop rating_stats table
DROP TABLE IF EXISTS driver_rating_stats;

-- Drop indexes for ratings
DROP INDEX IF EXISTS idx_driver_ratings_driver_id;
DROP INDEX IF EXISTS idx_driver_ratings_order_id;
DROP INDEX IF EXISTS idx_driver_ratings_customer_id;
DROP INDEX IF EXISTS idx_driver_ratings_rating;
DROP INDEX IF EXISTS idx_driver_ratings_type;
DROP INDEX IF EXISTS idx_driver_ratings_created_at;
DROP INDEX IF EXISTS idx_driver_ratings_verified;
DROP INDEX IF EXISTS idx_driver_ratings_driver_rating;
DROP INDEX IF EXISTS idx_driver_ratings_driver_type;
DROP INDEX IF EXISTS idx_driver_ratings_unique_order;

-- Drop table
DROP TABLE IF EXISTS driver_ratings;