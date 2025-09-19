//! Database Integration Tests
//! 
//! This module contains comprehensive database tests to verify data consistency,
//! constraints, triggers, and database-specific functionality.

use anyhow::Result;
use chrono::{Duration, Utc};
use serial_test::serial;
use sqlx::Row;
use uuid::Uuid;

use crate::fixtures::{generate_test_drivers, generate_test_locations, TestShift, TestRating};
use crate::helpers::{PerformanceMeasurement, TestResults};
use crate::{TestEnvironment, init_test_environment};

/// Test database connectivity and basic operations
#[tokio::test]
#[serial]
async fn test_database_connectivity() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Test basic query
    let result = sqlx::query("SELECT 1 as test_value")
        .fetch_one(env.database.get_pool())
        .await?;

    let test_value: i32 = result.get("test_value");
    assert_eq!(test_value, 1);

    Ok(())
}

/// Test driver CRUD operations at database level
#[tokio::test]
#[serial]
async fn test_driver_database_crud() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();

    // Create driver
    let created_id = env.database.create_test_driver(&test_driver).await?;
    assert_eq!(created_id, test_driver.id);

    // Read driver
    let retrieved = env.database.get_driver(test_driver.id).await?;
    assert!(retrieved.is_some());
    
    let driver = retrieved.unwrap();
    assert_eq!(driver.phone, test_driver.phone);
    assert_eq!(driver.email, test_driver.email);
    assert_eq!(driver.first_name, test_driver.first_name);
    assert_eq!(driver.status, test_driver.status);

    // Update driver rating directly in database
    sqlx::query!(
        "UPDATE drivers SET current_rating = $1, updated_at = NOW() WHERE id = $2",
        4.5_f64,
        test_driver.id
    )
    .execute(env.database.get_pool())
    .await?;

    // Verify update
    let updated = env.database.get_driver(test_driver.id).await?;
    assert!(updated.is_some());
    assert_eq!(updated.unwrap().current_rating, 4.5);

    // Delete (soft delete)
    sqlx::query!(
        "UPDATE drivers SET deleted_at = NOW() WHERE id = $1",
        test_driver.id
    )
    .execute(env.database.get_pool())
    .await?;

    // Verify soft delete
    let deleted = env.database.get_driver(test_driver.id).await?;
    assert!(deleted.is_none(), "Soft deleted driver should not be retrieved");

    Ok(())
}

/// Test database constraints and validations
#[tokio::test]
#[serial]
async fn test_database_constraints() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    env.database.create_test_driver(&test_driver).await?;

    // Test unique phone constraint
    let duplicate_phone_driver = generate_test_drivers(1).into_iter().next().unwrap();
    let result = sqlx::query!(
        r#"
        INSERT INTO drivers (
            id, phone, email, first_name, last_name,
            birth_date, passport_series, passport_number, license_number,
            license_expiry, status, current_rating, total_trips,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
        "#,
        duplicate_phone_driver.id,
        test_driver.phone, // Same phone as original driver
        duplicate_phone_driver.email,
        duplicate_phone_driver.first_name,
        duplicate_phone_driver.last_name,
        duplicate_phone_driver.birth_date.date_naive(),
        duplicate_phone_driver.passport_series,
        duplicate_phone_driver.passport_number,
        duplicate_phone_driver.license_number,
        duplicate_phone_driver.license_expiry.date_naive(),
        duplicate_phone_driver.status,
        duplicate_phone_driver.current_rating,
        duplicate_phone_driver.total_trips,
        Utc::now(),
        Utc::now()
    )
    .execute(env.database.get_pool())
    .await;

    assert!(result.is_err(), "Should fail due to unique phone constraint");

    // Test rating check constraint
    let invalid_rating_result = sqlx::query!(
        "UPDATE drivers SET current_rating = $1 WHERE id = $2",
        6.0_f64, // Invalid rating > 5.0
        test_driver.id
    )
    .execute(env.database.get_pool())
    .await;

    assert!(invalid_rating_result.is_err(), "Should fail due to rating check constraint");

    // Test status check constraint
    let invalid_status_result = sqlx::query!(
        "UPDATE drivers SET status = $1 WHERE id = $2",
        "invalid_status",
        test_driver.id
    )
    .execute(env.database.get_pool())
    .await;

    assert!(invalid_status_result.is_err(), "Should fail due to status check constraint");

    Ok(())
}

/// Test location table operations and constraints
#[tokio::test]
#[serial]
async fn test_location_database_operations() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    env.database.create_test_driver(&test_driver).await?;

    // Create test locations
    let test_locations = generate_test_locations(test_driver.id, 5);
    env.database.create_test_locations(&test_locations).await?;

    // Retrieve locations
    let retrieved_locations = env.database.get_driver_locations(test_driver.id, Some(10)).await?;
    assert_eq!(retrieved_locations.len(), 5);

    // Verify locations are ordered by recorded_at DESC
    for i in 0..retrieved_locations.len()-1 {
        assert!(
            retrieved_locations[i].recorded_at >= retrieved_locations[i+1].recorded_at,
            "Locations should be ordered by recorded_at DESC"
        );
    }

    // Test coordinate constraints
    let invalid_location_result = sqlx::query!(
        r#"
        INSERT INTO driver_locations (
            id, driver_id, latitude, longitude, recorded_at, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6)
        "#,
        Uuid::new_v4(),
        test_driver.id,
        91.0_f64, // Invalid latitude > 90
        0.0_f64,
        Utc::now(),
        Utc::now()
    )
    .execute(env.database.get_pool())
    .await;

    assert!(invalid_location_result.is_err(), "Should fail due to latitude constraint");

    Ok(())
}

/// Test database triggers and automatic updates
#[tokio::test]
#[serial]
async fn test_database_triggers() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    env.database.create_test_driver(&test_driver).await?;

    // Get initial updated_at timestamp
    let initial = env.database.get_driver(test_driver.id).await?;
    let initial_updated_at = initial.unwrap().updated_at;

    // Small delay to ensure timestamp difference
    tokio::time::sleep(std::time::Duration::from_millis(100)).await;

    // Update driver - should trigger updated_at update
    sqlx::query!(
        "UPDATE drivers SET first_name = $1 WHERE id = $2",
        "Updated Name",
        test_driver.id
    )
    .execute(env.database.get_pool())
    .await?;

    // Verify updated_at was automatically updated
    let updated = env.database.get_driver(test_driver.id).await?;
    let updated_updated_at = updated.unwrap().updated_at;

    assert!(
        updated_updated_at > initial_updated_at,
        "updated_at should be automatically updated by trigger"
    );

    Ok(())
}

/// Test rating statistics calculations
#[tokio::test]
#[serial]
async fn test_rating_statistics() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    env.database.create_test_driver(&test_driver).await?;

    // Create test ratings
    let ratings = vec![
        TestRating {
            id: Uuid::new_v4(),
            driver_id: test_driver.id,
            order_id: Some(Uuid::new_v4()),
            customer_id: Some(Uuid::new_v4()),
            rating: 5,
            comment: Some("Excellent!".to_string()),
            rating_type: "customer".to_string(),
        },
        TestRating {
            id: Uuid::new_v4(),
            driver_id: test_driver.id,
            order_id: Some(Uuid::new_v4()),
            customer_id: Some(Uuid::new_v4()),
            rating: 4,
            comment: Some("Good".to_string()),
            rating_type: "customer".to_string(),
        },
        TestRating {
            id: Uuid::new_v4(),
            driver_id: test_driver.id,
            order_id: Some(Uuid::new_v4()),
            customer_id: Some(Uuid::new_v4()),
            rating: 3,
            comment: Some("Average".to_string()),
            rating_type: "customer".to_string(),
        }
    ];

    for rating in &ratings {
        env.database.create_test_rating(rating).await?;
    }

    // Check if rating statistics were automatically calculated
    let stats = env.database.get_driver_stats(test_driver.id).await?;
    assert!(stats.is_some());

    let driver_stats = stats.unwrap();
    assert_eq!(driver_stats.total_ratings, 3);
    assert_eq!(driver_stats.average_rating, Some(4.0)); // (5+4+3)/3 = 4.0

    // Verify driver's current_rating was updated
    let updated_driver = env.database.get_driver(test_driver.id).await?;
    assert_eq!(updated_driver.unwrap().current_rating, 4.0);

    Ok(())
}

/// Test database foreign key constraints
#[tokio::test]
#[serial]
async fn test_foreign_key_constraints() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
    env.database.create_test_driver(&test_driver).await?;

    // Test location foreign key constraint
    let non_existent_driver_id = Uuid::new_v4();
    let result = sqlx::query!(
        r#"
        INSERT INTO driver_locations (
            id, driver_id, latitude, longitude, recorded_at, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6)
        "#,
        Uuid::new_v4(),
        non_existent_driver_id,
        55.0_f64,
        37.0_f64,
        Utc::now(),
        Utc::now()
    )
    .execute(env.database.get_pool())
    .await;

    assert!(result.is_err(), "Should fail due to foreign key constraint");

    // Test cascade delete
    let location_id = Uuid::new_v4();
    sqlx::query!(
        r#"
        INSERT INTO driver_locations (
            id, driver_id, latitude, longitude, recorded_at, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6)
        "#,
        location_id,
        test_driver.id,
        55.0_f64,
        37.0_f64,
        Utc::now(),
        Utc::now()
    )
    .execute(env.database.get_pool())
    .await?;

    // Delete driver (hard delete)
    sqlx::query!("DELETE FROM drivers WHERE id = $1", test_driver.id)
        .execute(env.database.get_pool())
        .await?;

    // Verify location was cascade deleted
    let location_count = sqlx::query_scalar!(
        "SELECT COUNT(*) FROM driver_locations WHERE driver_id = $1",
        test_driver.id
    )
    .fetch_one(env.database.get_pool())
    .await?;

    assert_eq!(location_count, Some(0), "Location should be cascade deleted");

    Ok(())
}

/// Test database indexing and query performance
#[tokio::test]
#[serial]
async fn test_database_indexing_performance() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping");
        return Ok(());
    }
    
    env.cleanup().await?;

    // Create many test drivers for performance testing
    let drivers_count = 1000;
    let test_drivers = generate_test_drivers(drivers_count);
    
    let mut create_measurement = PerformanceMeasurement::start("bulk_create_drivers", drivers_count);
    
    for driver in &test_drivers {
        env.database.create_test_driver(driver).await?;
    }
    
    create_measurement.finish();

    // Test indexed queries performance
    let query_count = 100;
    let mut query_measurement = PerformanceMeasurement::start("indexed_queries", query_count);
    
    for _ in 0..query_count {
        // Query by phone (indexed)
        let random_driver = &test_drivers[rand::random::<usize>() % test_drivers.len()];
        let _result = sqlx::query!(
            "SELECT id FROM drivers WHERE phone = $1 AND deleted_at IS NULL",
            random_driver.phone
        )
        .fetch_optional(env.database.get_pool())
        .await?;
    }
    
    query_measurement.finish();

    // Test range queries
    let range_measurement = PerformanceMeasurement::start("rating_range_query", 1);
    
    let _high_rated_drivers = sqlx::query!(
        "SELECT id, current_rating FROM drivers WHERE current_rating >= $1 AND deleted_at IS NULL ORDER BY current_rating DESC LIMIT 100",
        4.0_f64
    )
    .fetch_all(env.database.get_pool())
    .await?;
    
    range_measurement.finish();

    // Performance assertions (adjust thresholds based on your requirements)
    assert!(
        query_measurement.operations_per_second() > 50.0,
        "Indexed queries should be > 50 ops/sec, got: {:.2}",
        query_measurement.operations_per_second()
    );

    Ok(())
}

/// Test database transaction handling
#[tokio::test]
#[serial]
async fn test_database_transactions() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    let test_driver = generate_test_drivers(1).into_iter().next().unwrap();

    // Test successful transaction
    let mut tx = env.database.get_pool().begin().await?;
    
    let created_id = sqlx::query_scalar!(
        r#"
        INSERT INTO drivers (
            id, phone, email, first_name, last_name,
            birth_date, passport_series, passport_number, license_number,
            license_expiry, status, current_rating, total_trips,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
        RETURNING id
        "#,
        test_driver.id,
        test_driver.phone,
        test_driver.email,
        test_driver.first_name,
        test_driver.last_name,
        test_driver.birth_date.date_naive(),
        test_driver.passport_series,
        test_driver.passport_number,
        test_driver.license_number,
        test_driver.license_expiry.date_naive(),
        test_driver.status,
        test_driver.current_rating,
        test_driver.total_trips,
        Utc::now(),
        Utc::now()
    )
    .fetch_one(&mut *tx)
    .await?;

    assert_eq!(created_id, test_driver.id);

    // Commit transaction
    tx.commit().await?;

    // Verify driver was created
    let created_driver = env.database.get_driver(test_driver.id).await?;
    assert!(created_driver.is_some());

    // Test rollback transaction
    let mut tx2 = env.database.get_pool().begin().await?;
    
    sqlx::query!(
        "UPDATE drivers SET first_name = $1 WHERE id = $2",
        "Rolled Back Name",
        test_driver.id
    )
    .execute(&mut *tx2)
    .await?;

    // Rollback transaction
    tx2.rollback().await?;

    // Verify rollback - name should be unchanged
    let unchanged_driver = env.database.get_driver(test_driver.id).await?;
    assert_ne!(unchanged_driver.unwrap().first_name, "Rolled Back Name");

    Ok(())
}

/// Test database data consistency
#[tokio::test]
#[serial]
async fn test_data_consistency() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test data
    let test_drivers = generate_test_drivers(3);
    for driver in &test_drivers {
        env.database.create_test_driver(driver).await?;
    }

    // Create locations for drivers
    for driver in &test_drivers {
        let locations = generate_test_locations(driver.id, 5);
        env.database.create_test_locations(&locations).await?;
    }

    // Run consistency checks
    let consistency_report = env.database.verify_data_consistency().await?;
    
    if !consistency_report.is_all_passed() {
        println!("Data consistency issues found:");
        for failed_check in consistency_report.get_failed_checks() {
            println!("  - {}", failed_check);
        }
    }

    assert!(consistency_report.is_all_passed(), "All data consistency checks should pass");

    Ok(())
}

/// Test database backup and restore scenarios
#[tokio::test]
#[serial]
async fn test_database_backup_simulation() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;

    // Create test data
    let test_drivers = generate_test_drivers(5);
    let mut created_ids = Vec::new();
    
    for driver in &test_drivers {
        let id = env.database.create_test_driver(driver).await?;
        created_ids.push(id);
    }

    // Simulate backup by counting records
    let driver_count_before = sqlx::query_scalar!(
        "SELECT COUNT(*) FROM drivers WHERE deleted_at IS NULL"
    )
    .fetch_one(env.database.get_pool())
    .await?
    .unwrap_or(0);

    assert_eq!(driver_count_before, 5);

    // Simulate some operations (updates)
    for id in &created_ids {
        sqlx::query!(
            "UPDATE drivers SET current_rating = current_rating + 0.1 WHERE id = $1",
            id
        )
        .execute(env.database.get_pool())
        .await?;
    }

    // Verify data integrity after operations
    let driver_count_after = sqlx::query_scalar!(
        "SELECT COUNT(*) FROM drivers WHERE deleted_at IS NULL"
    )
    .fetch_one(env.database.get_pool())
    .await?
    .unwrap_or(0);

    assert_eq!(driver_count_after, driver_count_before, "Record count should remain same after updates");

    // Verify all drivers still exist with updated ratings
    for id in &created_ids {
        let driver = env.database.get_driver(*id).await?;
        assert!(driver.is_some(), "Driver should exist after updates");
        assert!(driver.unwrap().current_rating > 0.0, "Rating should be updated");
    }

    Ok(())
}

/// Test database connection pooling under load
#[tokio::test]
#[serial]
async fn test_connection_pool_under_load() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping");
        return Ok(());
    }
    
    env.cleanup().await?;

    let concurrent_operations = 20;
    let operations_per_task = 10;
    
    let mut tasks = Vec::new();
    
    for task_id in 0..concurrent_operations {
        let pool = env.database.get_pool().clone();
        
        let task = tokio::spawn(async move {
            let mut results = Vec::new();
            
            for op_id in 0..operations_per_task {
                // Simulate database operations with connection pool
                let query_result = sqlx::query!(
                    "SELECT $1 as task_id, $2 as op_id, NOW() as timestamp",
                    task_id,
                    op_id
                )
                .fetch_one(&pool)
                .await;
                
                results.push(query_result.is_ok());
            }
            
            results
        });
        
        tasks.push(task);
    }
    
    // Wait for all tasks
    let all_results = futures::future::join_all(tasks).await;
    
    let mut total_ops = 0;
    let mut successful_ops = 0;
    
    for task_result in all_results {
        let ops = task_result?;
        for success in ops {
            total_ops += 1;
            if success {
                successful_ops += 1;
            }
        }
    }
    
    let success_rate = successful_ops as f64 / total_ops as f64;
    
    println!("Connection pool test: {} total ops, {} successful ({:.1}% success rate)", 
             total_ops, successful_ops, success_rate * 100.0);
    
    // All operations should succeed under normal connection pool load
    assert!(success_rate > 0.99, "Success rate should be > 99%, got: {:.1}%", success_rate * 100.0);

    Ok(())
}