-- Driver Service Test Database Initialization
-- This script sets up the test database with proper permissions and settings

-- Create additional test-specific extensions if needed
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create test schemas if needed for different test categories
CREATE SCHEMA IF NOT EXISTS test_data;
CREATE SCHEMA IF NOT EXISTS performance_test;

-- Grant permissions to test user
GRANT ALL PRIVILEGES ON SCHEMA public TO test_user;
GRANT ALL PRIVILEGES ON SCHEMA test_data TO test_user;
GRANT ALL PRIVILEGES ON SCHEMA performance_test TO test_user;

-- Create test-specific helper functions
CREATE OR REPLACE FUNCTION test_data.generate_test_phone(prefix TEXT DEFAULT '+7900')
RETURNS TEXT AS $$
BEGIN
    RETURN prefix || LPAD(FLOOR(RANDOM() * 10000000)::TEXT, 7, '0');
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION test_data.generate_test_email(domain TEXT DEFAULT 'test.example.com')
RETURNS TEXT AS $$
BEGIN
    RETURN 'test' || FLOOR(RANDOM() * 100000) || '@' || domain;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION test_data.generate_test_license()
RETURNS TEXT AS $$
BEGIN
    RETURN 'TEST' || LPAD(FLOOR(RANDOM() * 1000000)::TEXT, 6, '0');
END;
$$ LANGUAGE plpgsql;

-- Create function to clean test data
CREATE OR REPLACE FUNCTION test_data.cleanup_all_test_data()
RETURNS VOID AS $$
BEGIN
    -- Delete in proper order to respect foreign key constraints
    DELETE FROM driver_ratings;
    DELETE FROM driver_shifts; 
    DELETE FROM driver_locations;
    DELETE FROM driver_documents;
    DELETE FROM driver_rating_stats;
    DELETE FROM drivers;
    
    -- Reset sequences if they exist
    -- ALTER SEQUENCE IF EXISTS some_sequence RESTART WITH 1;
    
    RAISE NOTICE 'All test data cleaned up';
END;
$$ LANGUAGE plpgsql;

-- Create function to generate test data for performance testing
CREATE OR REPLACE FUNCTION performance_test.create_test_drivers(driver_count INTEGER DEFAULT 100)
RETURNS VOID AS $$
DECLARE
    i INTEGER;
    test_driver_id UUID;
BEGIN
    FOR i IN 1..driver_count LOOP
        test_driver_id := uuid_generate_v4();
        
        INSERT INTO drivers (
            id, phone, email, first_name, last_name,
            birth_date, passport_series, passport_number, license_number,
            license_expiry, status, current_rating, total_trips,
            created_at, updated_at
        ) VALUES (
            test_driver_id,
            test_data.generate_test_phone(),
            test_data.generate_test_email(),
            'TestDriver' || i,
            'Generated',
            CURRENT_DATE - INTERVAL '25 years' - (RANDOM() * INTERVAL '15 years'),
            LPAD(FLOOR(RANDOM() * 10000)::TEXT, 4, '0'),
            LPAD(FLOOR(RANDOM() * 1000000)::TEXT, 6, '0'),
            test_data.generate_test_license(),
            CURRENT_DATE + INTERVAL '2 years' + (RANDOM() * INTERVAL '3 years'),
            (ARRAY['registered', 'verified', 'available'])[FLOOR(RANDOM() * 3 + 1)],
            ROUND((RANDOM() * 2 + 3)::NUMERIC, 2), -- Rating between 3.0 and 5.0
            FLOOR(RANDOM() * 1000),
            NOW() - (RANDOM() * INTERVAL '30 days'),
            NOW()
        );
        
        -- Add some location history for each driver
        INSERT INTO driver_locations (
            id, driver_id, latitude, longitude, altitude, accuracy, speed, bearing,
            recorded_at, created_at
        ) VALUES (
            uuid_generate_v4(),
            test_driver_id,
            55.7558 + (RANDOM() - 0.5) * 0.2, -- Moscow area coordinates
            37.6176 + (RANDOM() - 0.5) * 0.2,
            150.0 + (RANDOM() - 0.5) * 100,
            5.0 + RANDOM() * 10,
            RANDOM() * 60,
            RANDOM() * 360,
            NOW() - (RANDOM() * INTERVAL '1 hour'),
            NOW()
        );
    END LOOP;
    
    RAISE NOTICE 'Created % test drivers with location data', driver_count;
END;
$$ LANGUAGE plpgsql;

-- Create indexes for test performance
CREATE INDEX IF NOT EXISTS idx_test_drivers_phone_lookup ON drivers(phone) WHERE phone LIKE '+7900%';
CREATE INDEX IF NOT EXISTS idx_test_locations_recent ON driver_locations(recorded_at) WHERE recorded_at > CURRENT_TIMESTAMP - INTERVAL '1 day';

-- Grant execute permissions on test functions
GRANT EXECUTE ON FUNCTION test_data.generate_test_phone(TEXT) TO test_user;
GRANT EXECUTE ON FUNCTION test_data.generate_test_email(TEXT) TO test_user;
GRANT EXECUTE ON FUNCTION test_data.generate_test_license() TO test_user;
GRANT EXECUTE ON FUNCTION test_data.cleanup_all_test_data() TO test_user;
GRANT EXECUTE ON FUNCTION performance_test.create_test_drivers(INTEGER) TO test_user;

-- Create test statistics view
CREATE OR REPLACE VIEW test_data.test_statistics AS
SELECT 
    'drivers' as table_name,
    COUNT(*) as record_count,
    COUNT(*) FILTER (WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour') as recent_count
FROM drivers
WHERE phone LIKE '+7900%' OR email LIKE '%test.example.com'

UNION ALL

SELECT 
    'driver_locations' as table_name,
    COUNT(*) as record_count,
    COUNT(*) FILTER (WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour') as recent_count
FROM driver_locations dl
JOIN drivers d ON dl.driver_id = d.id
WHERE d.phone LIKE '+7900%' OR d.email LIKE '%test.example.com'

UNION ALL

SELECT 
    'driver_ratings' as table_name,
    COUNT(*) as record_count,
    COUNT(*) FILTER (WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour') as recent_count
FROM driver_ratings dr
JOIN drivers d ON dr.driver_id = d.id  
WHERE d.phone LIKE '+7900%' OR d.email LIKE '%test.example.com';

GRANT SELECT ON test_data.test_statistics TO test_user;

-- Log initialization completion
DO $$
BEGIN
    RAISE NOTICE 'Driver Service test database initialized successfully';
    RAISE NOTICE 'Available test functions:';
    RAISE NOTICE '  - test_data.cleanup_all_test_data()';
    RAISE NOTICE '  - performance_test.create_test_drivers(count)';
    RAISE NOTICE '  - test_data.generate_test_phone(prefix)';
    RAISE NOTICE '  - test_data.generate_test_email(domain)';
    RAISE NOTICE '  - test_data.generate_test_license()';
    RAISE NOTICE 'Available test views:';
    RAISE NOTICE '  - test_data.test_statistics';
END $$;