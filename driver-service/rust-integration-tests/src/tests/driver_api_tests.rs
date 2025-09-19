//! Driver API Integration Tests
//! 
//! This module contains comprehensive integration tests for the Driver Service HTTP API.
//! Tests cover all CRUD operations, validation, error handling, and edge cases.

use anyhow::Result;
use chrono::{Duration, Utc};
use serial_test::serial;
use uuid::Uuid;

use crate::fixtures::{generate_test_drivers, create_driver_request, UpdateDriverRequest, VALID_STATUSES};
use crate::helpers::{with_timeout, PerformanceMeasurement, TestResults};
use crate::{TestEnvironment, init_test_environment};

/// Test driver creation via API
#[tokio::test]
#[serial]
async fn test_create_driver_success() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let request = create_driver_request(&test_driver);

    let response = env.api_client.create_driver(&request).await?;

    assert_eq!(response.phone, request.phone);
    assert_eq!(response.email, request.email);
    assert_eq!(response.first_name, request.first_name);
    assert_eq!(response.last_name, request.last_name);
    assert_eq!(response.status, "registered");
    assert!(response.current_rating >= 0.0);

    // Verify driver was created in database
    let db_driver = env.database.get_driver(response.id).await?;
    assert!(db_driver.is_some());

    Ok(())
}

/// Test driver creation validation
#[tokio::test]
#[serial]
async fn test_create_driver_validation() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Test missing required fields
    let invalid_requests = vec![
        serde_json::json!({
            "email": "test@example.com",
            "first_name": "Test",
            "last_name": "User",
            // missing phone
            "birth_date": "1990-01-01T00:00:00Z",
            "passport_series": "1234",
            "passport_number": "567890",
            "license_number": "TEST123",
            "license_expiry": "2026-01-01T00:00:00Z"
        }),
        serde_json::json!({
            "phone": "+79001234567",
            "email": "invalid-email", // invalid email format
            "first_name": "Test",
            "last_name": "User",
            "birth_date": "1990-01-01T00:00:00Z",
            "passport_series": "1234",
            "passport_number": "567890",
            "license_number": "TEST123",
            "license_expiry": "2026-01-01T00:00:00Z"
        }),
        serde_json::json!({
            "phone": "+79001234567",
            "email": "test@example.com",
            "first_name": "", // empty first name
            "last_name": "User",
            "birth_date": "1990-01-01T00:00:00Z",
            "passport_series": "1234",
            "passport_number": "567890",
            "license_number": "TEST123",
            "license_expiry": "2026-01-01T00:00:00Z"
        })
    ];

    for invalid_request in invalid_requests {
        let response = reqwest::Client::new()
            .post(&format!("{}/drivers", env.config.service_api_url()))
            .json(&invalid_request)
            .send()
            .await?;

        assert!(response.status().is_client_error(), "Expected 4xx status code for invalid request");
    }

    Ok(())
}

/// Test duplicate driver creation
#[tokio::test]
#[serial]
async fn test_create_duplicate_driver() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let request = create_driver_request(&test_driver);

    // Create driver first time
    let _response1 = env.api_client.create_driver(&request).await?;

    // Try to create same driver again - should fail
    let result = env.api_client.create_driver(&request).await;
    assert!(result.is_err(), "Expected error when creating duplicate driver");

    Ok(())
}

/// Test get driver by ID
#[tokio::test]
#[serial]
async fn test_get_driver_success() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test driver
    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created = env.api_client.create_test_driver(&test_driver).await?;

    // Get driver by ID
    let retrieved = env.api_client.get_driver(created.id).await?;

    assert_eq!(retrieved.id, created.id);
    assert_eq!(retrieved.phone, created.phone);
    assert_eq!(retrieved.email, created.email);

    Ok(())
}

/// Test get non-existent driver
#[tokio::test]
#[serial]
async fn test_get_driver_not_found() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let non_existent_id = Uuid::new_v4();
    let result = env.api_client.get_driver(non_existent_id).await;
    
    assert!(result.is_err(), "Expected error when getting non-existent driver");
    
    Ok(())
}

/// Test update driver
#[tokio::test]
#[serial]
async fn test_update_driver_success() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test driver
    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created = env.api_client.create_test_driver(&test_driver).await?;

    // Update driver
    let update_request = UpdateDriverRequest {
        email: Some("updated@example.com".to_string()),
        first_name: Some("Updated".to_string()),
        last_name: None,
        middle_name: None,
        birth_date: None,
        passport_series: None,
        passport_number: None,
        license_expiry: None,
    };

    let updated = env.api_client.update_driver(created.id, &update_request).await?;

    assert_eq!(updated.email, "updated@example.com");
    assert_eq!(updated.first_name, "Updated");
    assert_eq!(updated.last_name, created.last_name); // unchanged

    Ok(())
}

/// Test delete driver
#[tokio::test]
#[serial]
async fn test_delete_driver_success() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test driver
    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created = env.api_client.create_test_driver(&test_driver).await?;

    // Delete driver
    env.api_client.delete_driver(created.id).await?;

    // Verify driver is deleted
    let result = env.api_client.get_driver(created.id).await;
    assert!(result.is_err(), "Driver should be deleted");

    Ok(())
}

/// Test list drivers with pagination
#[tokio::test]
#[serial]
async fn test_list_drivers_pagination() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create multiple test drivers
    let test_drivers = generate_test_drivers(5);
    for driver in test_drivers {
        env.api_client.create_test_driver(&driver).await?;
    }

    // Test pagination - first page
    let page1 = env.api_client.list_drivers(None, None, Some(3), Some(0)).await?;
    assert_eq!(page1.drivers.len(), 3);
    assert_eq!(page1.limit, 3);
    assert_eq!(page1.offset, 0);
    assert_eq!(page1.total, 5);
    assert!(page1.has_more);

    // Test pagination - second page
    let page2 = env.api_client.list_drivers(None, None, Some(3), Some(3)).await?;
    assert_eq!(page2.drivers.len(), 2);
    assert_eq!(page2.limit, 3);
    assert_eq!(page2.offset, 3);
    assert_eq!(page2.total, 5);
    assert!(!page2.has_more);

    Ok(())
}

/// Test list drivers with filters
#[tokio::test]
#[serial]
async fn test_list_drivers_with_filters() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test drivers with different ratings
    let mut test_drivers = generate_test_drivers(3);
    test_drivers[0].current_rating = 3.5;
    test_drivers[1].current_rating = 4.2;
    test_drivers[2].current_rating = 4.8;

    for driver in test_drivers {
        env.api_client.create_test_driver(&driver).await?;
    }

    // Filter by minimum rating
    let filtered = env.api_client.list_drivers(None, Some(4.0), None, None).await?;
    assert_eq!(filtered.drivers.len(), 2, "Should return drivers with rating >= 4.0");
    
    for driver in filtered.drivers {
        assert!(driver.current_rating >= 4.0);
    }

    Ok(())
}

/// Test change driver status
#[tokio::test]
#[serial]
async fn test_change_driver_status() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test driver
    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let created = env.api_client.create_test_driver(&test_driver).await?;

    // Test each valid status transition
    for &new_status in VALID_STATUSES {
        if new_status == "registered" { continue; } // skip initial status
        
        let result = env.api_client.change_driver_status(created.id, new_status).await?;
        
        // Verify response
        assert_eq!(result["status"], new_status);
        
        // Verify in database
        let updated_driver = env.api_client.get_driver(created.id).await?;
        assert_eq!(updated_driver.status, new_status);
    }

    Ok(())
}

/// Test get active drivers
#[tokio::test]
#[serial]
async fn test_get_active_drivers() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test drivers with different statuses
    let test_drivers = generate_test_drivers(5);
    let active_statuses = vec!["available", "on_shift", "busy"];
    let inactive_statuses = vec!["inactive", "blocked"];

    // Create active drivers
    for (i, driver) in test_drivers.iter().take(3).enumerate() {
        let created = env.api_client.create_test_driver(driver).await?;
        env.api_client.change_driver_status(created.id, active_statuses[i]).await?;
    }

    // Create inactive drivers
    for (i, driver) in test_drivers.iter().skip(3).enumerate() {
        let created = env.api_client.create_test_driver(driver).await?;
        env.api_client.change_driver_status(created.id, inactive_statuses[i]).await?;
    }

    // Get active drivers
    let response = env.api_client.get_active_drivers().await?;
    let count = response["count"].as_u64().unwrap();
    
    assert_eq!(count, 3, "Should return only active drivers");

    Ok(())
}

/// Test API performance under load
#[tokio::test]
#[serial]
async fn test_driver_api_performance() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping");
        return Ok(());
    }
    
    env.cleanup().await?;

    let operations_count = 50;
    let test_drivers = generate_test_drivers(operations_count);
    
    // Measure creation performance
    let mut create_measurement = PerformanceMeasurement::start("create_drivers", operations_count);
    
    let mut created_drivers = Vec::new();
    for driver in test_drivers {
        let response = env.api_client.create_test_driver(&driver).await?;
        created_drivers.push(response);
    }
    
    create_measurement.finish();
    
    // Measure retrieval performance
    let mut get_measurement = PerformanceMeasurement::start("get_drivers", operations_count);
    
    for driver in &created_drivers {
        let _retrieved = env.api_client.get_driver(driver.id).await?;
    }
    
    get_measurement.finish();
    
    // Assertions on performance (adjust thresholds as needed)
    assert!(
        create_measurement.operations_per_second() > 10.0,
        "Create operations should be > 10 ops/sec, got: {:.2}",
        create_measurement.operations_per_second()
    );
    
    assert!(
        get_measurement.operations_per_second() > 50.0,
        "Get operations should be > 50 ops/sec, got: {:.2}",
        get_measurement.operations_per_second()
    );

    Ok(())
}

/// Test concurrent driver operations
#[tokio::test]
#[serial]
async fn test_concurrent_driver_operations() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let concurrent_users = 10;
    let operations_per_user = 5;
    
    let mut tasks = Vec::new();
    
    for user_id in 0..concurrent_users {
        let env_clone = env.clone(); // Assuming TestEnvironment implements Clone
        let task = tokio::spawn(async move {
            let mut results = Vec::new();
            
            for op_id in 0..operations_per_user {
                let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
                
                // Make phone unique for each operation
                let mut driver = test_driver;
                driver.phone = format!("+7900123{:04}{:04}", user_id, op_id);
                driver.email = format!("test.user.{}.{}.@example.com", user_id, op_id);
                driver.license_number = format!("TEST{:04}{:04}", user_id, op_id);
                
                match env_clone.api_client.create_test_driver(&driver).await {
                    Ok(response) => results.push((user_id, op_id, Ok(response))),
                    Err(e) => results.push((user_id, op_id, Err(e))),
                }
            }
            
            results
        });
        
        tasks.push(task);
    }
    
    // Wait for all tasks to complete
    let all_results = futures::future::join_all(tasks).await;
    
    let mut total_operations = 0;
    let mut successful_operations = 0;
    
    for task_result in all_results {
        let operations = task_result?;
        for (user_id, op_id, result) in operations {
            total_operations += 1;
            if result.is_ok() {
                successful_operations += 1;
            } else {
                println!("Operation failed for user {} op {}: {:?}", user_id, op_id, result);
            }
        }
    }
    
    let success_rate = successful_operations as f64 / total_operations as f64;
    
    println!("Concurrent operations: {} total, {} successful ({:.1}% success rate)", 
             total_operations, successful_operations, success_rate * 100.0);
    
    // Expect high success rate for concurrent operations
    assert!(success_rate > 0.95, "Success rate should be > 95%, got: {:.1}%", success_rate * 100.0);

    Ok(())
}

/// Test API error handling and resilience
#[tokio::test]
#[serial]
async fn test_api_error_handling() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Test invalid UUIDs
    let invalid_uuid = "not-a-uuid";
    let response = reqwest::Client::new()
        .get(&format!("{}/drivers/{}", env.config.service_api_url(), invalid_uuid))
        .send()
        .await?;
    
    assert_eq!(response.status(), 400, "Should return 400 for invalid UUID");

    // Test non-existent resource
    let non_existent_id = Uuid::new_v4();
    let result = env.api_client.get_driver(non_existent_id).await;
    assert!(result.is_err(), "Should return error for non-existent driver");

    // Test malformed JSON
    let response = reqwest::Client::new()
        .post(&format!("{}/drivers", env.config.service_api_url()))
        .header("Content-Type", "application/json")
        .body("invalid json")
        .send()
        .await?;
    
    assert!(response.status().is_client_error(), "Should return 4xx for malformed JSON");

    Ok(())
}

/// Test driver API with timeout scenarios
#[tokio::test]
#[serial]
async fn test_api_with_timeout() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();

    // Test operation with timeout
    let result = with_timeout(
        env.api_client.create_test_driver(&test_driver),
        std::time::Duration::from_secs(10),
        "create_driver_with_timeout"
    ).await?;

    assert!(result.id != Uuid::nil(), "Driver should be created successfully");

    Ok(())
}

// Helper function to set up test environment for driver API tests
async fn setup_driver_api_test_env() -> Result<TestEnvironment> {
    let env = init_test_environment().await?;
    env.cleanup().await?;
    Ok(env)
}

// Integration test that combines multiple driver operations
#[tokio::test]
#[serial]
async fn test_driver_lifecycle_integration() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    
    // 1. Create driver
    let created = env.api_client.create_test_driver(&test_driver).await?;
    assert_eq!(created.status, "registered");

    // 2. Update driver information
    let update_request = UpdateDriverRequest {
        email: Some("updated@example.com".to_string()),
        first_name: Some("Updated".to_string()),
        last_name: None,
        middle_name: Some("Updateovich".to_string()),
        birth_date: None,
        passport_series: None,
        passport_number: None,
        license_expiry: Some(Utc::now() + Duration::days(1000)),
    };
    
    let updated = env.api_client.update_driver(created.id, &update_request).await?;
    assert_eq!(updated.email, "updated@example.com");
    assert_eq!(updated.first_name, "Updated");

    // 3. Change status to available
    env.api_client.change_driver_status(created.id, "available").await?;
    
    // 4. Verify driver appears in active drivers list
    let active_drivers = env.api_client.get_active_drivers().await?;
    let count = active_drivers["count"].as_u64().unwrap();
    assert!(count >= 1, "Driver should appear in active drivers list");

    // 5. Change status to busy (simulate taking an order)
    env.api_client.change_driver_status(created.id, "busy").await?;
    
    // 6. Change back to available (simulate completing order)
    env.api_client.change_driver_status(created.id, "available").await?;

    // 7. Finally delete the driver
    env.api_client.delete_driver(created.id).await?;

    // 8. Verify driver is deleted
    let result = env.api_client.get_driver(created.id).await;
    assert!(result.is_err(), "Driver should be deleted");

    Ok(())
}