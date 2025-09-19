-- Create driver_ratings table
CREATE TABLE driver_ratings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    order_id UUID, -- Reference to order (external service)
    customer_id UUID, -- Reference to customer (external service)
    rating INTEGER NOT NULL,
    comment TEXT,
    rating_type VARCHAR(50) NOT NULL DEFAULT 'customer',
    criteria_scores JSONB DEFAULT '{}',
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    is_anonymous BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for driver_ratings table
CREATE INDEX idx_driver_ratings_driver_id ON driver_ratings(driver_id);
CREATE INDEX idx_driver_ratings_order_id ON driver_ratings(order_id);
CREATE INDEX idx_driver_ratings_customer_id ON driver_ratings(customer_id);
CREATE INDEX idx_driver_ratings_rating ON driver_ratings(rating);
CREATE INDEX idx_driver_ratings_type ON driver_ratings(rating_type);
CREATE INDEX idx_driver_ratings_created_at ON driver_ratings(created_at DESC);
CREATE INDEX idx_driver_ratings_verified ON driver_ratings(is_verified);

-- Create composite indexes for common queries
CREATE INDEX idx_driver_ratings_driver_rating ON driver_ratings(driver_id, rating, created_at DESC);
CREATE INDEX idx_driver_ratings_driver_type ON driver_ratings(driver_id, rating_type, created_at DESC);

-- Create unique constraint to prevent duplicate ratings for same order
CREATE UNIQUE INDEX idx_driver_ratings_unique_order ON driver_ratings(driver_id, order_id, customer_id) 
    WHERE order_id IS NOT NULL AND customer_id IS NOT NULL;

-- Add check constraints
ALTER TABLE driver_ratings ADD CONSTRAINT check_driver_ratings_rating 
    CHECK (rating >= 1 AND rating <= 5);

ALTER TABLE driver_ratings ADD CONSTRAINT check_driver_ratings_type 
    CHECK (rating_type IN ('customer', 'system', 'admin', 'peer', 'automatic'));

-- Create trigger for updated_at
CREATE TRIGGER update_driver_ratings_updated_at BEFORE UPDATE ON driver_ratings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create rating_stats table for aggregated statistics
CREATE TABLE driver_rating_stats (
    driver_id UUID PRIMARY KEY REFERENCES drivers(id) ON DELETE CASCADE,
    average_rating DECIMAL(3,2) NOT NULL DEFAULT 0.0,
    total_ratings INTEGER NOT NULL DEFAULT 0,
    rating_distribution JSONB DEFAULT '{}',
    criteria_averages JSONB DEFAULT '{}',
    last_rating_date TIMESTAMP WITH TIME ZONE,
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for rating_stats table
CREATE INDEX idx_driver_rating_stats_average ON driver_rating_stats(average_rating);
CREATE INDEX idx_driver_rating_stats_total ON driver_rating_stats(total_ratings);
CREATE INDEX idx_driver_rating_stats_updated ON driver_rating_stats(last_updated);

-- Add check constraints for rating_stats
ALTER TABLE driver_rating_stats ADD CONSTRAINT check_rating_stats_average 
    CHECK (average_rating >= 0.0 AND average_rating <= 5.0);

ALTER TABLE driver_rating_stats ADD CONSTRAINT check_rating_stats_total 
    CHECK (total_ratings >= 0);

-- Create trigger for rating_stats updated_at
CREATE TRIGGER update_driver_rating_stats_updated_at BEFORE UPDATE ON driver_rating_stats 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to update rating statistics
CREATE OR REPLACE FUNCTION update_driver_rating_stats(target_driver_id UUID) 
RETURNS void AS $$
DECLARE
    avg_rating DECIMAL(3,2);
    total_count INTEGER;
    rating_dist JSONB;
    criteria_avg JSONB;
    last_rating_dt TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Calculate average rating and total count
    SELECT 
        COALESCE(AVG(rating), 0.0),
        COUNT(*),
        MAX(created_at)
    INTO avg_rating, total_count, last_rating_dt
    FROM driver_ratings 
    WHERE driver_id = target_driver_id;
    
    -- Calculate rating distribution
    SELECT json_object_agg(rating, count)::jsonb
    INTO rating_dist
    FROM (
        SELECT rating, COUNT(*) as count
        FROM driver_ratings 
        WHERE driver_id = target_driver_id
        GROUP BY rating
    ) t;
    
    -- Calculate criteria averages
    WITH criteria_data AS (
        SELECT 
            key as criteria,
            AVG(value::integer) as avg_score
        FROM driver_ratings,
        LATERAL jsonb_each_text(criteria_scores)
        WHERE driver_id = target_driver_id 
        AND jsonb_typeof(criteria_scores) = 'object'
        GROUP BY key
    )
    SELECT json_object_agg(criteria, avg_score)::jsonb
    INTO criteria_avg
    FROM criteria_data;
    
    -- Insert or update statistics
    INSERT INTO driver_rating_stats (
        driver_id, 
        average_rating, 
        total_ratings, 
        rating_distribution,
        criteria_averages,
        last_rating_date,
        last_updated
    ) VALUES (
        target_driver_id, 
        avg_rating, 
        total_count, 
        COALESCE(rating_dist, '{}'::jsonb),
        COALESCE(criteria_avg, '{}'::jsonb),
        last_rating_dt,
        NOW()
    )
    ON CONFLICT (driver_id) DO UPDATE SET
        average_rating = EXCLUDED.average_rating,
        total_ratings = EXCLUDED.total_ratings,
        rating_distribution = EXCLUDED.rating_distribution,
        criteria_averages = EXCLUDED.criteria_averages,
        last_rating_date = EXCLUDED.last_rating_date,
        last_updated = NOW();
        
    -- Update driver's current_rating
    UPDATE drivers 
    SET current_rating = avg_rating, updated_at = NOW()
    WHERE id = target_driver_id;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to auto-update statistics when ratings change
CREATE OR REPLACE FUNCTION trigger_update_rating_stats() 
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        PERFORM update_driver_rating_stats(NEW.driver_id);
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        PERFORM update_driver_rating_stats(OLD.driver_id);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_driver_rating_stats_update
    AFTER INSERT OR UPDATE OR DELETE ON driver_ratings
    FOR EACH ROW EXECUTE FUNCTION trigger_update_rating_stats();