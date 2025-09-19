//! Integration Scenario Tests
//! 
//! This module contains comprehensive end-to-end integration tests that simulate
//! real-world scenarios and workflows involving multiple components and services.

use anyhow::Result;
use chrono::{Duration, Utc};
use serial_test::serial;
use std::time::Duration as StdDuration;
use tokio::time::sleep;
use uuid::Uuid;

use crate::fixtures::{
    generate_test_drivers, generate_test_locations, UpdateLocationRequest,
    TestRating, MOSCOW_COORDINATES, SPB_COORDINATES, KAZAN_COORDINATES
};
use crate::helpers::{with_timeout, calculate_distance, TestResults, EventTestHelper};
use crate::{TestEnvironment, init_test_environment};

/// Complete driver onboarding and verification scenario
#[tokio::test]
#[serial]
async fn test_complete_driver_onboarding_scenario() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    println!("Starting complete driver onboarding scenario...");

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    
    // Step 1: Driver registration
    println!("Step 1: Driver registration");
    let registered_driver = env.api_client.create_test_driver(&test_driver).await?;
    assert_eq!(registered_driver.status, "registered");
    println!("  ✓ Driver registered with ID: {}", registered_driver.id);

    // Step 2: Document upload simulation (change status to pending_verification)
    println!("Step 2: Document verification process");
    env.api_client.change_driver_status(registered_driver.id, "pending_verification").await?;
    
    // Simulate document verification delay
    sleep(StdDuration::from_millis(500)).await;
    
    // Step 3: Documents verified
    env.api_client.change_driver_status(registered_driver.id, "verified").await?;
    println!("  ✓ Driver documents verified");

    // Step 4: Driver becomes available
    println!("Step 3: Driver goes online");
    env.api_client.change_driver_status(registered_driver.id, "available").await?;
    
    // Step 5: Update initial location
    let initial_location = UpdateLocationRequest {
        latitude: MOSCOW_COORDINATES.0,
        longitude: MOSCOW_COORDINATES.1,
        altitude: Some(150.0),
        accuracy: Some(5.0),
        speed: Some(0.0),
        bearing: Some(0.0),
        timestamp: Some(Utc::now().timestamp()),
    };
    
    env.api_client.update_location(registered_driver.id, &initial_location).await?;
    println!("  ✓ Initial location set: Moscow center");

    // Step 6: Verify driver appears in active and nearby searches
    let active_drivers = env.api_client.get_active_drivers().await?;
    let active_count = active_drivers["count"].as_u64().unwrap();
    assert!(active_count >= 1, "Driver should appear in active drivers list");
    println!("  ✓ Driver appears in active drivers list");

    let nearby_drivers = env.api_client.get_nearby_drivers(
        MOSCOW_COORDINATES.0,
        MOSCOW_COORDINATES.1,
        Some(10.0),
        Some(10)
    ).await?;
    let nearby_count = nearby_drivers["count"].as_u64().unwrap();
    assert!(nearby_count >= 1, "Driver should appear in nearby drivers search");
    println!("  ✓ Driver appears in nearby search results");

    // Step 7: Verify database consistency
    let db_driver = env.database.get_driver(registered_driver.id).await?;
    assert!(db_driver.is_some());
    assert_eq!(db_driver.unwrap().status, "available");
    
    let current_location = env.api_client.get_current_location(registered_driver.id).await?;
    assert_eq!(current_location.latitude, initial_location.latitude);
    assert_eq!(current_location.longitude, initial_location.longitude);
    println!("  ✓ Database consistency verified");

    println!("Complete driver onboarding scenario: ✅ SUCCESS");
    Ok(())
}

/// Full ride lifecycle scenario
#[tokio::test]
#[serial]
async fn test_complete_ride_lifecycle_scenario() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    println!("Starting complete ride lifecycle scenario...");

    // Setup: Create and prepare driver
    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let driver = env.api_client.create_test_driver(&test_driver).await?;
    env.api_client.change_driver_status(driver.id, "available").await?;
    
    // Set driver at pickup location
    let pickup_location = MOSCOW_COORDINATES;
    let pickup_request = UpdateLocationRequest {
        latitude: pickup_location.0,
        longitude: pickup_location.1,
        altitude: Some(150.0),
        accuracy: Some(5.0),
        speed: Some(0.0),
        bearing: Some(0.0),
        timestamp: Some(Utc::now().timestamp()),
    };
    env.api_client.update_location(driver.id, &pickup_request).await?;
    println!("  ✓ Driver positioned at pickup location");

    // Step 1: Order assignment (simulated via events if NATS is available)
    println!("Step 1: Order assignment");
    let order_id = Uuid::new_v4();
    
    if let Some(nats_client) = &env.nats_client {
        nats_client.simulate_order_assigned(driver.id, order_id).await?;
        println!("  ✓ Order assignment event sent");
        
        // Allow time for event processing
        sleep(StdDuration::from_millis(200)).await;
    }

    // Manually change driver status to busy (simulating order acceptance)
    env.api_client.change_driver_status(driver.id, "busy").await?;
    println!("  ✓ Driver accepts order and status changes to busy");

    // Step 2: Driver moves to pickup location (already there in this case)
    println!("Step 2: En route to pickup");
    
    // Simulate small movement to exact pickup
    let exact_pickup = UpdateLocationRequest {
        latitude: pickup_location.0 + 0.0001,
        longitude: pickup_location.1 + 0.0001,
        altitude: Some(150.0),
        accuracy: Some(3.0),
        speed: Some(15.0),
        bearing: Some(90.0),
        timestamp: Some(Utc::now().timestamp()),
    };
    env.api_client.update_location(driver.id, &exact_pickup).await?;
    println!("  ✓ Driver navigating to exact pickup point");

    // Step 3: Arrival at pickup
    println!("Step 3: Arrived at pickup");
    let arrival_location = UpdateLocationRequest {
        latitude: pickup_location.0 + 0.0002,
        longitude: pickup_location.1 + 0.0002,
        altitude: Some(150.0),
        accuracy: Some(2.0),
        speed: Some(0.0),
        bearing: Some(90.0),
        timestamp: Some(Utc::now().timestamp()),
    };
    env.api_client.update_location(driver.id, &arrival_location).await?;
    println!("  ✓ Driver arrived at pickup location");

    // Step 4: Trip starts - simulate route to destination
    println!("Step 4: Trip in progress");
    let destination = SPB_COORDINATES; // Moscow to Saint Petersburg
    let route_points = generate_route_points(
        (arrival_location.latitude, arrival_location.longitude),
        destination,
        10 // number of points along route
    );

    for (i, (lat, lon)) in route_points.iter().enumerate() {
        let speed = if i == 0 { 0.0 } else { 60.0 }; // Start from 0, then highway speed
        let bearing = calculate_bearing(
            if i == 0 { (arrival_location.latitude, arrival_location.longitude) } else { route_points[i-1] },
            (*lat, *lon)
        );

        let route_location = UpdateLocationRequest {
            latitude: *lat,
            longitude: *lon,
            altitude: Some(100.0),
            accuracy: Some(5.0),
            speed: Some(speed),
            bearing: Some(bearing),
            timestamp: Some((Utc::now() + Duration::minutes(i as i64)).timestamp()),
        };

        env.api_client.update_location(driver.id, &route_location).await?;
        
        // Small delay between location updates
        sleep(StdDuration::from_millis(50)).await;
    }
    println!("  ✓ Trip route completed ({} GPS updates)", route_points.len());

    // Step 5: Arrival at destination
    println!("Step 5: Arrival at destination");
    let final_location = UpdateLocationRequest {
        latitude: destination.0,
        longitude: destination.1,
        altitude: Some(50.0),
        accuracy: Some(3.0),
        speed: Some(0.0),
        bearing: Some(0.0),
        timestamp: Some(Utc::now().timestamp()),
    };
    env.api_client.update_location(driver.id, &final_location).await?;
    println!("  ✓ Driver arrived at destination");

    // Step 6: Order completion
    println!("Step 6: Order completion");
    if let Some(nats_client) = &env.nats_client {
        // Simulate order completion event
        let completion_event = serde_json::json!({
            "event_type": "order.completed",
            "order_id": order_id.to_string(),
            "driver_id": driver.id.to_string(),
            "customer_id": Uuid::new_v4().to_string(),
            "actual_fare": 2500.0,
            "actual_distance": calculate_distance(pickup_location.0, pickup_location.1, destination.0, destination.1),
            "duration": 180, // 3 hours
            "timestamp": Utc::now()
        });
        nats_client.publish_event("order.completed", &completion_event).await?;
        println!("  ✓ Order completion event sent");
    }

    // Change driver status back to available
    env.api_client.change_driver_status(driver.id, "available").await?;
    println!("  ✓ Driver status changed back to available");

    // Step 7: Customer rating simulation
    println!("Step 7: Customer rating");
    if let Some(nats_client) = &env.nats_client {
        nats_client.simulate_customer_rating(driver.id, order_id, 5).await?;
        println!("  ✓ Customer rating submitted (5 stars)");
    }

    // Step 8: Payment processing simulation
    println!("Step 8: Payment processing");
    if let Some(nats_client) = &env.nats_client {
        nats_client.simulate_payment_processed(driver.id, order_id, 2500.0).await?;
        println!("  ✓ Payment processed");
    }

    // Step 9: Verification of final state
    println!("Step 9: Final state verification");
    
    // Check driver location history
    let history = env.api_client.get_location_history(
        driver.id,
        Some((Utc::now() - Duration::hours(1)).timestamp()),
        Some(Utc::now().timestamp())
    ).await?;
    
    let locations = history["locations"].as_array().unwrap();
    assert!(locations.len() > 10, "Should have comprehensive location history");
    println!("  ✓ Location history contains {} points", locations.len());

    // Verify final location
    let current_location = env.api_client.get_current_location(driver.id).await?;
    let distance_to_destination = calculate_distance(
        current_location.latitude,
        current_location.longitude,
        destination.0,
        destination.1
    );
    assert!(distance_to_destination < 1.0, "Driver should be at destination");
    println!("  ✓ Driver final location verified ({}km from destination)", distance_to_destination);

    // Check driver is available for next order
    let final_driver = env.api_client.get_driver(driver.id).await?;
    assert_eq!(final_driver.status, "available");
    println!("  ✓ Driver ready for next order");

    println!("Complete ride lifecycle scenario: ✅ SUCCESS");
    Ok(())
}

/// Multi-driver coordination scenario
#[tokio::test]
#[serial]
async fn test_multi_driver_coordination_scenario() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    println!("Starting multi-driver coordination scenario...");

    let drivers_count = 5;
    let test_drivers = generate_test_drivers(drivers_count);
    let mut created_drivers = Vec::new();

    // Step 1: Onboard multiple drivers
    println!("Step 1: Onboarding {} drivers", drivers_count);
    for (i, driver) in test_drivers.iter().enumerate() {
        let created = env.api_client.create_test_driver(driver).await?;
        env.api_client.change_driver_status(created.id, "available").await?;
        created_drivers.push(created);
        println!("  ✓ Driver {} onboarded: {}", i + 1, created.id);
    }

    // Step 2: Position drivers in different areas of Moscow
    println!("Step 2: Positioning drivers across Moscow");
    let moscow_areas = vec![
        (55.7558, 37.6176), // Red Square (center)
        (55.7739, 37.6208), // Pushkin Square (north)
        (55.7378, 37.6173), // Gorky Park (south) 
        (55.7558, 37.5862), // Victory Park (west)
        (55.7558, 37.6490), // Sokolniki Park (east)
    ];

    for (i, driver) in created_drivers.iter().enumerate() {
        let (lat, lon) = moscow_areas[i];
        let location = UpdateLocationRequest {
            latitude: lat,
            longitude: lon,
            altitude: Some(150.0),
            accuracy: Some(5.0),
            speed: Some(0.0),
            bearing: Some(0.0),
            timestamp: Some(Utc::now().timestamp()),
        };

        env.api_client.update_location(driver.id, &location).await?;
        println!("  ✓ Driver {} positioned at ({:.4}, {:.4})", i + 1, lat, lon);
    }

    // Step 3: Test nearby driver search from different locations
    println!("Step 3: Testing nearby driver discovery");
    
    for (i, &(search_lat, search_lon)) in moscow_areas.iter().enumerate() {
        let nearby = env.api_client.get_nearby_drivers(
            search_lat,
            search_lon,
            Some(5.0), // 5km radius
            Some(10)
        ).await?;

        let nearby_drivers = nearby["drivers"].as_array().unwrap();
        let nearby_count = nearby["count"].as_u64().unwrap();
        
        // Should find at least the driver in this area, possibly others
        assert!(nearby_count >= 1, "Should find at least 1 driver near area {}", i + 1);
        
        // Verify distances
        for driver_data in nearby_drivers {
            let distance = driver_data["distance_km"].as_f64().unwrap_or(0.0);
            assert!(distance <= 5.0, "Driver should be within 5km radius");
        }
        
        println!("  ✓ Area {}: Found {} nearby drivers", i + 1, nearby_count);
    }

    // Step 4: Simulate simultaneous order assignments
    println!("Step 4: Simultaneous order assignments");
    let mut order_tasks = Vec::new();
    
    for (i, driver) in created_drivers.iter().take(3).enumerate() { // Assign orders to first 3 drivers
        let driver_id = driver.id;
        let api_client = env.api_client.clone();
        let nats_client = env.nats_client.clone();
        
        let task = tokio::spawn(async move {
            // Simulate order assignment
            if let Some(nats) = nats_client {
                let order_id = Uuid::new_v4();
                nats.simulate_order_assigned(driver_id, order_id).await?;
            }
            
            // Change to busy status
            api_client.change_driver_status(driver_id, "busy").await?;
            
            // Simulate short trip
            sleep(StdDuration::from_millis(100 * i as u64)).await;
            
            // Complete trip
            api_client.change_driver_status(driver_id, "available").await?;
            
            Ok::<_, anyhow::Error>(driver_id)
        });
        
        order_tasks.push(task);
    }

    // Wait for all order simulations
    let order_results = futures::future::join_all(order_tasks).await;
    for (i, result) in order_results.iter().enumerate() {
        match result {
            Ok(Ok(driver_id)) => println!("  ✓ Driver {} order completed: {}", i + 1, driver_id),
            _ => println!("  ✗ Driver {} order failed", i + 1),
        }
    }

    // Step 5: Verify system state after concurrent operations
    println!("Step 5: System state verification");
    
    let active_drivers = env.api_client.get_active_drivers().await?;
    let active_count = active_drivers["count"].as_u64().unwrap();
    assert_eq!(active_count, drivers_count as u64, "All drivers should be active again");
    println!("  ✓ All {} drivers returned to available state", drivers_count);

    // Check location data consistency
    for (i, driver) in created_drivers.iter().enumerate() {
        let current_location = env.api_client.get_current_location(driver.id).await?;
        let expected_area = moscow_areas[i];
        let distance_from_expected = calculate_distance(
            current_location.latitude,
            current_location.longitude,
            expected_area.0,
            expected_area.1
        );
        
        // Should be close to original position (allowing for small movements)
        assert!(distance_from_expected < 2.0, "Driver should be near original position");
        println!("  ✓ Driver {} location consistent", i + 1);
    }

    println!("Multi-driver coordination scenario: ✅ SUCCESS");
    Ok(())
}

/// Peak hours stress scenario
#[tokio::test]
#[serial]
async fn test_peak_hours_stress_scenario() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping peak hours stress scenario");
        return Ok(());
    }
    
    env.cleanup().await?;

    println!("Starting peak hours stress scenario...");

    // Simulate a busy period with many drivers and frequent updates
    let peak_drivers_count = 20;
    let location_updates_per_driver = 30;
    let update_frequency_ms = 100; // 10 updates per second per driver

    // Step 1: Create peak-time driver fleet
    println!("Step 1: Creating peak-time driver fleet ({} drivers)", peak_drivers_count);
    let test_drivers = generate_test_drivers(peak_drivers_count);
    let mut created_drivers = Vec::new();

    for driver in test_drivers {
        let created = env.api_client.create_test_driver(&driver).await?;
        env.api_client.change_driver_status(created.id, "available").await?;
        created_drivers.push(created);
    }
    println!("  ✓ {} drivers created and activated", peak_drivers_count);

    // Step 2: Distribute drivers across Moscow metropolitan area
    println!("Step 2: Distributing drivers across metropolitan area");
    for (i, driver) in created_drivers.iter().enumerate() {
        // Generate positions in a grid around Moscow
        let base_lat = 55.7558;
        let base_lon = 37.6176;
        let offset_lat = ((i / 5) as f64 - 2.0) * 0.05; // Grid: -0.1 to +0.1 degrees
        let offset_lon = ((i % 5) as f64 - 2.0) * 0.05;

        let location = UpdateLocationRequest {
            latitude: base_lat + offset_lat,
            longitude: base_lon + offset_lon,
            altitude: Some(150.0),
            accuracy: Some(5.0),
            speed: Some(0.0),
            bearing: Some(0.0),
            timestamp: Some(Utc::now().timestamp()),
        };

        env.api_client.update_location(driver.id, &location).await?;
    }
    println!("  ✓ Drivers distributed across metropolitan area");

    // Step 3: Simulate peak-hour traffic with high-frequency location updates
    println!("Step 3: Simulating peak-hour traffic (high-frequency GPS updates)");
    let start_time = std::time::Instant::now();
    let mut location_tasks = Vec::new();

    for (driver_idx, driver) in created_drivers.iter().enumerate() {
        let driver_id = driver.id;
        let api_client = env.api_client.clone();
        
        let task = tokio::spawn(async move {
            let base_lat = 55.7558 + ((driver_idx / 5) as f64 - 2.0) * 0.05;
            let base_lon = 37.6176 + ((driver_idx % 5) as f64 - 2.0) * 0.05;
            let mut success_count = 0;
            
            for update_idx in 0..location_updates_per_driver {
                // Simulate realistic movement patterns
                let movement_radius = 0.01; // ~1km movement radius
                let angle = (update_idx as f64) * 0.2; // Slow rotation
                let distance_offset = (update_idx as f64) * 0.0001; // Gradual movement
                
                let lat = base_lat + (angle.sin() * movement_radius) + distance_offset;
                let lon = base_lon + (angle.cos() * movement_radius) + distance_offset;
                
                let speed = 20.0 + (update_idx as f64 % 40.0); // Variable speed 20-60 km/h
                let bearing = (angle.to_degrees() + 90.0) % 360.0;

                let location_request = UpdateLocationRequest {
                    latitude: lat,
                    longitude: lon,
                    altitude: Some(150.0),
                    accuracy: Some(5.0),
                    speed: Some(speed),
                    bearing: Some(bearing),
                    timestamp: Some(Utc::now().timestamp()),
                };

                match api_client.update_location(driver_id, &location_request).await {
                    Ok(_) => success_count += 1,
                    Err(_) => {
                        // Continue on error during stress test
                    }
                }

                sleep(StdDuration::from_millis(update_frequency_ms)).await;
            }
            
            (driver_idx, success_count, location_updates_per_driver)
        });

        location_tasks.push(task);
    }

    // Wait for all location update tasks
    let location_results = futures::future::join_all(location_tasks).await;
    let total_duration = start_time.elapsed();

    // Analyze results
    let mut total_updates = 0;
    let mut successful_updates = 0;
    
    for task_result in location_results {
        if let Ok((driver_idx, success_count, total_count)) = task_result {
            total_updates += total_count;
            successful_updates += success_count;
            println!("  ✓ Driver {}: {}/{} successful updates", driver_idx, success_count, total_count);
        }
    }

    let success_rate = successful_updates as f64 / total_updates as f64;
    let update_throughput = total_updates as f64 / total_duration.as_secs_f64();

    println!("Peak Hours Stress Results:");
    println!("  Duration: {:?}", total_duration);
    println!("  Total Updates: {}", total_updates);
    println!("  Successful: {}", successful_updates);
    println!("  Success Rate: {:.1}%", success_rate * 100.0);
    println!("  Update Throughput: {:.1} updates/sec", update_throughput);

    // Step 4: Test system responsiveness during peak load
    println!("Step 4: Testing system responsiveness during peak load");
    
    // Perform API queries while location updates are ongoing
    let responsiveness_tasks = vec![
        // Test active drivers query
        async {
            let start = std::time::Instant::now();
            let result = env.api_client.get_active_drivers().await;
            (start.elapsed(), result.is_ok(), "active_drivers")
        },
        
        // Test nearby drivers search
        async {
            let start = std::time::Instant::now();
            let result = env.api_client.get_nearby_drivers(55.7558, 37.6176, Some(10.0), Some(20)).await;
            (start.elapsed(), result.is_ok(), "nearby_drivers")
        },
        
        // Test driver list with pagination
        async {
            let start = std::time::Instant::now();
            let result = env.api_client.list_drivers(None, None, Some(10), Some(0)).await;
            (start.elapsed(), result.is_ok(), "list_drivers")
        }
    ];

    for responsiveness_task in responsiveness_tasks {
        let (duration, success, operation) = responsiveness_task.await;
        println!("  ✓ {} - {:?} (success: {})", operation, duration, success);
        
        // System should remain responsive even under load
        assert!(duration < StdDuration::from_secs(5), 
               "{} should complete within 5 seconds during peak load, took: {:?}", 
               operation, duration);
    }

    // Assertions for peak hours performance
    assert!(success_rate > 0.90, 
           "Success rate should be > 90% during peak hours, got: {:.1}%", 
           success_rate * 100.0);
    
    assert!(update_throughput > 50.0,
           "System should handle > 50 location updates/sec during peak, got: {:.1}",
           update_throughput);

    println!("Peak hours stress scenario: ✅ SUCCESS");
    Ok(())
}

/// Driver shift and earnings tracking scenario
#[tokio::test]
#[serial]
async fn test_driver_shift_and_earnings_scenario() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    println!("Starting driver shift and earnings tracking scenario...");

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let driver = env.api_client.create_test_driver(&test_driver).await?;
    env.api_client.change_driver_status(driver.id, "available").await?;
    
    // Step 1: Driver starts shift
    println!("Step 1: Driver starts shift");
    let shift_start_location = MOSCOW_COORDINATES;
    let shift_start_request = UpdateLocationRequest {
        latitude: shift_start_location.0,
        longitude: shift_start_location.1,
        altitude: Some(150.0),
        accuracy: Some(5.0),
        speed: Some(0.0),
        bearing: Some(0.0),
        timestamp: Some(Utc::now().timestamp()),
    };
    
    env.api_client.update_location(driver.id, &shift_start_request).await?;
    env.api_client.change_driver_status(driver.id, "on_shift").await?;
    println!("  ✓ Driver shift started at Moscow center");

    // Step 2: Simulate multiple rides during shift
    println!("Step 2: Processing multiple rides during shift");
    let ride_count = 3;
    let mut total_distance = 0.0;
    let mut total_earnings = 0.0;

    let destinations = vec![
        (55.7739, 37.6208), // Pushkin Square
        (55.7378, 37.6173), // Gorky Park
        (55.7558, 37.5862), // Victory Park
    ];

    for (ride_idx, destination) in destinations.iter().enumerate() {
        println!("  Ride {}: Starting", ride_idx + 1);
        
        // Get current location
        let current_location = env.api_client.get_current_location(driver.id).await?;
        let start_point = (current_location.latitude, current_location.longitude);
        
        // Calculate ride distance
        let ride_distance = calculate_distance(start_point.0, start_point.1, destination.0, destination.1);
        total_distance += ride_distance;
        
        // Simulate ride fare (base + distance)
        let ride_fare = 200.0 + (ride_distance * 25.0); // 200 RUB base + 25 RUB per km
        total_earnings += ride_fare;

        // Driver accepts order
        env.api_client.change_driver_status(driver.id, "busy").await?;
        
        // Simulate trip with intermediate points
        let trip_points = generate_route_points(start_point, *destination, 5);
        
        for (point_idx, (lat, lon)) in trip_points.iter().enumerate() {
            let speed = if point_idx == 0 { 0.0 } else { 40.0 + (point_idx as f64) * 5.0 };
            let bearing = calculate_bearing(
                if point_idx == 0 { start_point } else { trip_points[point_idx - 1] },
                (*lat, *lon)
            );

            let trip_location = UpdateLocationRequest {
                latitude: *lat,
                longitude: *lon,
                altitude: Some(150.0),
                accuracy: Some(5.0),
                speed: Some(speed),
                bearing: Some(bearing),
                timestamp: Some(Utc::now().timestamp()),
            };

            env.api_client.update_location(driver.id, &trip_location).await?;
            sleep(StdDuration::from_millis(200)).await;
        }

        // Complete ride
        env.api_client.change_driver_status(driver.id, "on_shift").await?;
        
        // Simulate customer rating and payment
        if let Some(nats_client) = &env.nats_client {
            let order_id = Uuid::new_v4();
            nats_client.simulate_customer_rating(driver.id, order_id, 4 + (ride_idx % 2)).await?;
            nats_client.simulate_payment_processed(driver.id, order_id, ride_fare).await?;
        }

        println!("    ✓ Ride completed: {:.1}km, {:.0} RUB", ride_distance, ride_fare);
        sleep(StdDuration::from_millis(300)).await;
    }

    println!("  ✓ {} rides completed during shift", ride_count);
    println!("  ✓ Total distance: {:.1}km", total_distance);
    println!("  ✓ Total earnings: {:.0} RUB", total_earnings);

    // Step 3: Driver ends shift
    println!("Step 3: Driver ends shift");
    let shift_end_location = env.api_client.get_current_location(driver.id).await?;
    env.api_client.change_driver_status(driver.id, "available").await?;
    println!("  ✓ Shift ended at location ({:.4}, {:.4})", 
             shift_end_location.latitude, shift_end_location.longitude);

    // Step 4: Verify shift data and statistics
    println!("Step 4: Shift statistics verification");
    
    // Check location history for the shift
    let shift_history = env.api_client.get_location_history(
        driver.id,
        Some((Utc::now() - Duration::hours(1)).timestamp()),
        Some(Utc::now().timestamp())
    ).await?;

    let history_locations = shift_history["locations"].as_array().unwrap();
    assert!(history_locations.len() > 15, "Should have comprehensive location history during shift");
    
    // Calculate actual distance traveled from GPS data
    let mut gps_distance = 0.0;
    for i in 1..history_locations.len() {
        let prev_lat = history_locations[i-1]["latitude"].as_f64().unwrap();
        let prev_lon = history_locations[i-1]["longitude"].as_f64().unwrap();
        let curr_lat = history_locations[i]["latitude"].as_f64().unwrap();
        let curr_lon = history_locations[i]["longitude"].as_f64().unwrap();
        
        gps_distance += calculate_distance(prev_lat, prev_lon, curr_lat, curr_lon);
    }

    println!("  ✓ Location history: {} GPS points", history_locations.len());
    println!("  ✓ GPS calculated distance: {:.1}km", gps_distance);
    
    // GPS distance should be reasonable (may include routing inefficiencies)
    assert!(gps_distance > total_distance * 0.8, "GPS distance should be close to calculated distance");
    assert!(gps_distance < total_distance * 2.0, "GPS distance should not be excessively higher");

    // Step 5: Driver statistics verification
    println!("Step 5: Driver statistics verification");
    let driver_stats = env.database.get_driver_stats(driver.id).await?;
    
    if let Some(stats) = driver_stats {
        println!("  ✓ Total ratings: {}", stats.total_ratings);
        println!("  ✓ Average rating: {:.1}", stats.average_rating.unwrap_or(0.0));
        println!("  ✓ Total locations recorded: {}", stats.total_locations);
        
        // Should have received ratings from rides
        assert!(stats.total_ratings >= ride_count as i64, "Should have ratings from completed rides");
        
        // Should have comprehensive location data
        assert!(stats.total_locations > 15, "Should have many location records from shift");
    }

    println!("Driver shift and earnings scenario: ✅ SUCCESS");
    Ok(())
}

// Helper functions for scenario tests

fn generate_route_points(start: (f64, f64), end: (f64, f64), num_points: usize) -> Vec<(f64, f64)> {
    let mut points = Vec::new();
    
    for i in 0..=num_points {
        let progress = i as f64 / num_points as f64;
        let lat = start.0 + (end.0 - start.0) * progress;
        let lon = start.1 + (end.1 - start.1) * progress;
        points.push((lat, lon));
    }
    
    points
}

fn calculate_bearing(from: (f64, f64), to: (f64, f64)) -> f64 {
    let lat1 = from.0.to_radians();
    let lat2 = to.0.to_radians();
    let delta_lon = (to.1 - from.1).to_radians();
    
    let y = delta_lon.sin() * lat2.cos();
    let x = lat1.cos() * lat2.sin() - lat1.sin() * lat2.cos() * delta_lon.cos();
    
    let bearing_rad = y.atan2(x);
    let bearing_deg = bearing_rad.to_degrees();
    
    (bearing_deg + 360.0) % 360.0
}