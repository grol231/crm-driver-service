//! Location API Integration Tests
//! 
//! This module contains comprehensive integration tests for location tracking functionality
//! including GPS updates, location history, nearby drivers search, and batch operations.

use anyhow::Result;
use chrono::{Duration, Utc};
use serial_test::serial;
use uuid::Uuid;

use crate::fixtures::{
    generate_test_drivers, generate_test_locations, create_location_request,
    UpdateLocationRequest, MOSCOW_COORDINATES, SPB_COORDINATES
};
use crate::helpers::{with_timeout, PerformanceMeasurement, calculate_distance};
use crate::{TestEnvironment, init_test_environment};

/// Test basic location update
#[tokio::test]
#[serial]
async fn test_update_location_success() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test driver
    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;

    // Update location
    let location_request = UpdateLocationRequest {
        latitude: MOSCOW_COORDINATES.0,
        longitude: MOSCOW_COORDINATES.1,
        altitude: Some(150.0),
        accuracy: Some(5.0),
        speed: Some(25.5),
        bearing: Some(45.0),
        timestamp: Some(Utc::now().timestamp()),
    };

    let response = env.api_client.update_location(created_driver.id, &location_request).await?;

    assert_eq!(response.driver_id, created_driver.id);
    assert_eq!(response.latitude, location_request.latitude);
    assert_eq!(response.longitude, location_request.longitude);
    assert_eq!(response.altitude, location_request.altitude);
    assert_eq!(response.accuracy, location_request.accuracy);
    assert_eq!(response.speed, location_request.speed);
    assert_eq!(response.bearing, location_request.bearing);

    Ok(())
}

/// Test location update validation
#[tokio::test]
#[serial]
async fn test_update_location_validation() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;

    // Test invalid coordinates
    let invalid_requests = vec![
        UpdateLocationRequest {
            latitude: 91.0, // invalid latitude > 90
            longitude: 0.0,
            altitude: None,
            accuracy: None,
            speed: None,
            bearing: None,
            timestamp: None,
        },
        UpdateLocationRequest {
            latitude: 0.0,
            longitude: 181.0, // invalid longitude > 180
            altitude: None,
            accuracy: None,
            speed: None,
            bearing: None,
            timestamp: None,
        },
        UpdateLocationRequest {
            latitude: 0.0,
            longitude: 0.0,
            altitude: None,
            accuracy: Some(-1.0), // negative accuracy
            speed: None,
            bearing: None,
            timestamp: None,
        }
    ];

    for invalid_request in invalid_requests {
        let result = env.api_client.update_location(created_driver.id, &invalid_request).await;
        assert!(result.is_err(), "Expected error for invalid location data");
    }

    Ok(())
}

/// Test batch location updates
#[tokio::test]
#[serial]
async fn test_batch_update_locations() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;

    // Generate test locations
    let test_locations = generate_test_locations(created_driver.id, 10);
    let location_requests: Vec<UpdateLocationRequest> = test_locations
        .iter()
        .map(|loc| create_location_request(loc))
        .collect();

    // Batch update
    let response = env.api_client.batch_update_locations(created_driver.id, &location_requests).await?;

    assert_eq!(response["count"], 10);
    assert_eq!(response["message"], "Locations updated successfully");

    Ok(())
}

/// Test get current location
#[tokio::test]
#[serial]
async fn test_get_current_location() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;

    // Update location first
    let location_request = UpdateLocationRequest {
        latitude: SPB_COORDINATES.0,
        longitude: SPB_COORDINATES.1,
        altitude: Some(10.0),
        accuracy: Some(3.0),
        speed: Some(0.0),
        bearing: Some(180.0),
        timestamp: Some(Utc::now().timestamp()),
    };

    env.api_client.update_location(created_driver.id, &location_request).await?;

    // Get current location
    let current = env.api_client.get_current_location(created_driver.id).await?;

    assert_eq!(current.driver_id, created_driver.id);
    assert_eq!(current.latitude, location_request.latitude);
    assert_eq!(current.longitude, location_request.longitude);

    Ok(())
}

/// Test get location history
#[tokio::test]
#[serial]
async fn test_get_location_history() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;

    // Create location history by updating location multiple times
    let locations_count = 5;
    let now = Utc::now();
    
    for i in 0..locations_count {
        let location_request = UpdateLocationRequest {
            latitude: MOSCOW_COORDINATES.0 + (i as f64) * 0.001, // slightly different positions
            longitude: MOSCOW_COORDINATES.1 + (i as f64) * 0.001,
            altitude: Some(100.0 + i as f64),
            accuracy: Some(5.0),
            speed: Some(i as f64 * 10.0),
            bearing: Some(i as f64 * 45.0),
            timestamp: Some((now - Duration::minutes(locations_count - i)).timestamp()),
        };

        env.api_client.update_location(created_driver.id, &location_request).await?;
        
        // Small delay to ensure different timestamps
        tokio::time::sleep(std::time::Duration::from_millis(100)).await;
    }

    // Get location history
    let from_timestamp = (now - Duration::hours(1)).timestamp();
    let to_timestamp = (now + Duration::minutes(5)).timestamp();
    
    let history = env.api_client.get_location_history(
        created_driver.id,
        Some(from_timestamp),
        Some(to_timestamp)
    ).await?;

    let locations = history["locations"].as_array().unwrap();
    let count = history["count"].as_u64().unwrap();
    
    assert_eq!(count, locations_count as u64);
    assert_eq!(locations.len(), locations_count);

    // Verify locations are ordered by time (most recent first)
    for i in 0..locations.len()-1 {
        let current_time = locations[i]["recorded_at"].as_str().unwrap();
        let next_time = locations[i+1]["recorded_at"].as_str().unwrap();
        assert!(current_time >= next_time, "Locations should be ordered by time desc");
    }

    Ok(())
}

/// Test nearby drivers search
#[tokio::test]
#[serial]
async fn test_get_nearby_drivers() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create multiple test drivers
    let test_drivers = generate_test_drivers(3);
    let mut created_drivers = Vec::new();
    
    for driver in test_drivers {
        let created = env.api_client.create_test_driver(&driver).await?;
        created_drivers.push(created);
    }

    // Set drivers as available
    for driver in &created_drivers {
        env.api_client.change_driver_status(driver.id, "available").await?;
    }

    // Update locations for drivers - some near, some far
    let center = MOSCOW_COORDINATES;
    let nearby_positions = vec![
        (center.0 + 0.001, center.1 + 0.001), // ~100m away
        (center.0 + 0.003, center.1 + 0.002), // ~300m away
        (center.0 + 0.1, center.1 + 0.1),     // ~10km away (far)
    ];

    for (i, driver) in created_drivers.iter().enumerate() {
        let pos = nearby_positions[i];
        let location_request = UpdateLocationRequest {
            latitude: pos.0,
            longitude: pos.1,
            altitude: Some(100.0),
            accuracy: Some(5.0),
            speed: Some(0.0),
            bearing: Some(0.0),
            timestamp: Some(Utc::now().timestamp()),
        };

        env.api_client.update_location(driver.id, &location_request).await?;
    }

    // Search for nearby drivers within 5km radius
    let nearby = env.api_client.get_nearby_drivers(
        center.0,
        center.1,
        Some(5.0), // 5km radius
        Some(10)   // limit 10
    ).await?;

    let drivers = nearby["drivers"].as_array().unwrap();
    let count = nearby["count"].as_u64().unwrap();

    // Should find first 2 drivers (within 5km), not the 3rd one (10km away)
    assert_eq!(count, 2);
    assert_eq!(drivers.len(), 2);

    // Verify distances are calculated correctly
    for driver in drivers {
        let lat = driver["latitude"].as_f64().unwrap();
        let lon = driver["longitude"].as_f64().unwrap();
        let distance = calculate_distance(center.0, center.1, lat, lon);
        assert!(distance <= 5.0, "Driver should be within 5km radius, but was {}km", distance);
    }

    Ok(())
}

/// Test location updates for non-existent driver
#[tokio::test]
#[serial]
async fn test_update_location_driver_not_found() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let non_existent_driver_id = Uuid::new_v4();
    let location_request = UpdateLocationRequest {
        latitude: MOSCOW_COORDINATES.0,
        longitude: MOSCOW_COORDINATES.1,
        altitude: None,
        accuracy: None,
        speed: None,
        bearing: None,
        timestamp: None,
    };

    let result = env.api_client.update_location(non_existent_driver_id, &location_request).await;
    assert!(result.is_err(), "Expected error for non-existent driver");

    Ok(())
}

/// Test location tracking performance under load
#[tokio::test]
#[serial]
async fn test_location_tracking_performance() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping");
        return Ok(());
    }
    
    env.cleanup().await?;

    // Create test driver
    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;

    let updates_count = 100;
    let mut performance = PerformanceMeasurement::start("location_updates", updates_count);

    // Simulate GPS updates
    for i in 0..updates_count {
        let location_request = UpdateLocationRequest {
            latitude: MOSCOW_COORDINATES.0 + (i as f64) * 0.0001,
            longitude: MOSCOW_COORDINATES.1 + (i as f64) * 0.0001,
            altitude: Some(100.0),
            accuracy: Some(5.0),
            speed: Some((i % 60) as f64), // varying speed
            bearing: Some((i * 3) as f64 % 360.0), // rotating bearing
            timestamp: Some(Utc::now().timestamp()),
        };

        env.api_client.update_location(created_driver.id, &location_request).await?;
    }

    performance.finish();

    // Should handle at least 10 location updates per second
    assert!(
        performance.operations_per_second() > 10.0,
        "Location updates should be > 10 ops/sec, got: {:.2}",
        performance.operations_per_second()
    );

    Ok(())
}

/// Test concurrent location updates from multiple drivers
#[tokio::test]
#[serial]
async fn test_concurrent_location_updates() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let drivers_count = 5;
    let updates_per_driver = 10;

    // Create test drivers
    let test_drivers = generate_test_drivers(drivers_count);
    let mut created_drivers = Vec::new();
    
    for driver in test_drivers {
        let created = env.api_client.create_test_driver(&driver).await?;
        created_drivers.push(created);
    }

    // Launch concurrent location update tasks
    let mut tasks = Vec::new();
    
    for (driver_idx, driver) in created_drivers.iter().enumerate() {
        let driver_id = driver.id;
        let api_client = env.api_client.clone();
        
        let task = tokio::spawn(async move {
            let mut results = Vec::new();
            
            for update_idx in 0..updates_per_driver {
                let location_request = UpdateLocationRequest {
                    latitude: MOSCOW_COORDINATES.0 + (driver_idx as f64) * 0.01 + (update_idx as f64) * 0.001,
                    longitude: MOSCOW_COORDINATES.1 + (driver_idx as f64) * 0.01 + (update_idx as f64) * 0.001,
                    altitude: Some(100.0 + update_idx as f64),
                    accuracy: Some(5.0),
                    speed: Some(update_idx as f64 * 2.0),
                    bearing: Some((update_idx * 30) as f64 % 360.0),
                    timestamp: Some(Utc::now().timestamp()),
                };

                match api_client.update_location(driver_id, &location_request).await {
                    Ok(_) => results.push(true),
                    Err(e) => {
                        println!("Location update failed for driver {}: {}", driver_id, e);
                        results.push(false);
                    }
                }
                
                // Small delay to simulate realistic GPS update frequency
                tokio::time::sleep(std::time::Duration::from_millis(50)).await;
            }
            
            results
        });
        
        tasks.push(task);
    }

    // Wait for all tasks to complete
    let all_results = futures::future::join_all(tasks).await;
    
    let mut total_updates = 0;
    let mut successful_updates = 0;
    
    for task_result in all_results {
        let updates = task_result?;
        for success in updates {
            total_updates += 1;
            if success {
                successful_updates += 1;
            }
        }
    }

    let success_rate = successful_updates as f64 / total_updates as f64;
    
    println!("Concurrent location updates: {} total, {} successful ({:.1}% success rate)", 
             total_updates, successful_updates, success_rate * 100.0);

    // Expect high success rate
    assert!(success_rate > 0.95, "Success rate should be > 95%, got: {:.1}%", success_rate * 100.0);

    Ok(())
}

/// Test location history with time filtering
#[tokio::test]
#[serial]
async fn test_location_history_time_filtering() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;

    let now = Utc::now();
    let base_time = now - Duration::hours(2);

    // Create location history with specific timestamps
    let timestamps = vec![
        base_time,                    // 2 hours ago
        base_time + Duration::minutes(30), // 1.5 hours ago
        base_time + Duration::hours(1),     // 1 hour ago
        base_time + Duration::minutes(90),  // 30 minutes ago
        now,                              // now
    ];

    for (i, timestamp) in timestamps.iter().enumerate() {
        let location_request = UpdateLocationRequest {
            latitude: MOSCOW_COORDINATES.0 + (i as f64) * 0.001,
            longitude: MOSCOW_COORDINATES.1 + (i as f64) * 0.001,
            altitude: Some(100.0),
            accuracy: Some(5.0),
            speed: Some(i as f64 * 5.0),
            bearing: Some(i as f64 * 45.0),
            timestamp: Some(timestamp.timestamp()),
        };

        env.api_client.update_location(created_driver.id, &location_request).await?;
        tokio::time::sleep(std::time::Duration::from_millis(100)).await;
    }

    // Test filtering - get locations from last hour
    let one_hour_ago = now - Duration::hours(1);
    let history = env.api_client.get_location_history(
        created_driver.id,
        Some(one_hour_ago.timestamp()),
        Some(now.timestamp() + 60) // small buffer for timing
    ).await?;

    let locations = history["locations"].as_array().unwrap();
    let count = history["count"].as_u64().unwrap();

    // Should get 3 locations (1 hour ago, 30 minutes ago, now)
    assert_eq!(count, 3);
    assert_eq!(locations.len(), 3);

    // Verify all returned locations are within the time range
    for location in locations {
        let recorded_at_str = location["recorded_at"].as_str().unwrap();
        let recorded_at = chrono::DateTime::parse_from_rfc3339(recorded_at_str).unwrap();
        assert!(recorded_at >= one_hour_ago.into(), "Location should be within time range");
    }

    Ok(())
}

/// Test batch location updates with validation
#[tokio::test]
#[serial]
async fn test_batch_location_updates_validation() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;

    // Create batch with some invalid locations
    let mixed_locations = vec![
        UpdateLocationRequest {
            latitude: MOSCOW_COORDINATES.0,
            longitude: MOSCOW_COORDINATES.1,
            altitude: Some(100.0),
            accuracy: Some(5.0),
            speed: Some(10.0),
            bearing: Some(45.0),
            timestamp: Some(Utc::now().timestamp()),
        },
        UpdateLocationRequest {
            latitude: 91.0, // Invalid latitude
            longitude: MOSCOW_COORDINATES.1,
            altitude: None,
            accuracy: None,
            speed: None,
            bearing: None,
            timestamp: None,
        }
    ];

    let result = env.api_client.batch_update_locations(created_driver.id, &mixed_locations).await;
    
    // Should fail due to invalid location in batch
    assert!(result.is_err(), "Batch should fail with invalid location data");

    Ok(())
}

/// Integration test for complete location tracking scenario
#[tokio::test]
#[serial]
async fn test_location_tracking_integration() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // 1. Create driver and set as available
    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created_driver = env.api_client.create_test_driver(&test_driver).await?;
    env.api_client.change_driver_status(created_driver.id, "available").await?;

    // 2. Start location tracking - simulate driver movement
    let route_points = vec![
        (MOSCOW_COORDINATES.0, MOSCOW_COORDINATES.1), // Start at Red Square
        (55.7540, 37.6200), // Move towards Kremlin
        (55.7520, 37.6175), // Continue movement
        (55.7500, 37.6150), // Final position
    ];

    for (i, (lat, lon)) in route_points.iter().enumerate() {
        let location_request = UpdateLocationRequest {
            latitude: *lat,
            longitude: *lon,
            altitude: Some(150.0),
            accuracy: Some(5.0),
            speed: Some(if i == 0 { 0.0 } else { 30.0 }), // Start stationary, then moving
            bearing: Some(if i == 0 { 0.0 } else { 45.0 }),
            timestamp: Some((Utc::now() + Duration::minutes(i as i64 * 5)).timestamp()),
        };

        env.api_client.update_location(created_driver.id, &location_request).await?;
        tokio::time::sleep(std::time::Duration::from_millis(200)).await;
    }

    // 3. Verify current location is the last updated position
    let current = env.api_client.get_current_location(created_driver.id).await?;
    let last_point = route_points.last().unwrap();
    assert_eq!(current.latitude, last_point.0);
    assert_eq!(current.longitude, last_point.1);

    // 4. Check location history contains all points
    let history = env.api_client.get_location_history(
        created_driver.id,
        Some((Utc::now() - Duration::hours(1)).timestamp()),
        Some((Utc::now() + Duration::hours(1)).timestamp())
    ).await?;

    let locations = history["locations"].as_array().unwrap();
    assert!(locations.len() >= route_points.len(), "History should contain all route points");

    // 5. Test nearby search - should find our driver
    let search_center = MOSCOW_COORDINATES;
    let nearby = env.api_client.get_nearby_drivers(
        search_center.0,
        search_center.1,
        Some(10.0), // 10km radius
        Some(10)
    ).await?;

    let drivers = nearby["drivers"].as_array().unwrap();
    let found_our_driver = drivers.iter().any(|d| {
        d["driver_id"].as_str().unwrap() == created_driver.id.to_string()
    });

    assert!(found_our_driver, "Our driver should be found in nearby search");

    Ok(())
}