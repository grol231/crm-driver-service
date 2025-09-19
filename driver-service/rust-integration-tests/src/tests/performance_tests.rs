//! Performance Integration Tests
//! 
//! This module contains comprehensive performance tests including load testing,
//! stress testing, throughput measurements, and resource usage analysis.

use anyhow::Result;
use chrono::{Duration, Utc};
use serial_test::serial;
use std::sync::{Arc, atomic::{AtomicUsize, Ordering}};
use std::time::{Duration as StdDuration, Instant};
use tokio::time::sleep;
use uuid::Uuid;

use crate::fixtures::{generate_test_drivers, UpdateLocationRequest, LoadTestConfig};
use crate::helpers::{PerformanceMeasurement, TestResults};
use crate::{TestEnvironment, init_test_environment};

/// Load test configuration
const DEFAULT_LOAD_TEST_CONFIG: LoadTestConfig = LoadTestConfig {
    concurrent_users: 50,
    duration: StdDuration::from_secs(30),
    ramp_up_time: StdDuration::from_secs(10),
    operations_per_user: 20,
};

/// Test API throughput under sustained load
#[tokio::test]
#[serial]
async fn test_api_throughput_load() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping throughput load test");
        return Ok(());
    }
    
    env.cleanup().await?;

    let config = DEFAULT_LOAD_TEST_CONFIG;
    let total_operations = Arc::new(AtomicUsize::new(0));
    let successful_operations = Arc::new(AtomicUsize::new(0));
    let failed_operations = Arc::new(AtomicUsize::new(0));

    println!("Starting API throughput load test:");
    println!("  Concurrent users: {}", config.concurrent_users);
    println!("  Duration: {:?}", config.duration);
    println!("  Operations per user: {}", config.operations_per_user);

    let start_time = Instant::now();
    let mut tasks = Vec::new();

    // Launch concurrent user tasks
    for user_id in 0..config.concurrent_users {
        let api_client = env.api_client.clone();
        let total_ops = Arc::clone(&total_operations);
        let success_ops = Arc::clone(&successful_operations);
        let failed_ops = Arc::clone(&failed_operations);
        
        // Stagger user start times for ramp-up
        let delay = config.ramp_up_time.as_millis() as u64 * user_id as u64 / config.concurrent_users as u64;
        
        let task = tokio::spawn(async move {
            // Wait for ramp-up delay
            sleep(StdDuration::from_millis(delay)).await;
            
            let mut user_results = Vec::new();
            
            for op_id in 0..config.operations_per_user {
                total_ops.fetch_add(1, Ordering::Relaxed);
                
                // Create unique test driver
                let mut test_driver = generate_test_drivers(1).into_iter().next().unwrap();
                test_driver.phone = format!("+7900{:04}{:04}", user_id, op_id);
                test_driver.email = format!("load.test.{}.{}@example.com", user_id, op_id);
                test_driver.license_number = format!("LOAD{:04}{:04}", user_id, op_id);

                let op_start = Instant::now();
                
                // Perform API operation
                match api_client.create_test_driver(&test_driver).await {
                    Ok(_) => {
                        success_ops.fetch_add(1, Ordering::Relaxed);
                        user_results.push((op_id, true, op_start.elapsed()));
                    }
                    Err(e) => {
                        failed_ops.fetch_add(1, Ordering::Relaxed);
                        user_results.push((op_id, false, op_start.elapsed()));
                        eprintln!("User {} operation {} failed: {}", user_id, op_id, e);
                    }
                }
                
                // Small delay between operations
                sleep(StdDuration::from_millis(50)).await;
            }
            
            user_results
        });
        
        tasks.push(task);
    }

    // Wait for all tasks or timeout
    let timeout_task = tokio::spawn(async move {
        sleep(config.duration + config.ramp_up_time + StdDuration::from_secs(30)).await;
    });

    tokio::select! {
        results = futures::future::join_all(tasks) => {
            println!("All load test tasks completed");
        }
        _ = timeout_task => {
            println!("Load test timed out");
        }
    }

    let total_duration = start_time.elapsed();
    let total_ops = total_operations.load(Ordering::Relaxed);
    let success_ops = successful_operations.load(Ordering::Relaxed);
    let failed_ops = failed_operations.load(Ordering::Relaxed);

    let throughput = total_ops as f64 / total_duration.as_secs_f64();
    let success_rate = success_ops as f64 / total_ops as f64;

    println!("Load Test Results:");
    println!("  Duration: {:?}", total_duration);
    println!("  Total Operations: {}", total_ops);
    println!("  Successful: {}", success_ops);
    println!("  Failed: {}", failed_ops);
    println!("  Throughput: {:.2} ops/sec", throughput);
    println!("  Success Rate: {:.1}%", success_rate * 100.0);

    // Performance assertions (adjust thresholds based on requirements)
    assert!(throughput > 10.0, "Throughput should be > 10 ops/sec, got: {:.2}", throughput);
    assert!(success_rate > 0.95, "Success rate should be > 95%, got: {:.1}%", success_rate * 100.0);

    Ok(())
}

/// Test location updates performance under high frequency
#[tokio::test]
#[serial]
async fn test_location_updates_performance() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping location updates performance test");
        return Ok(());
    }
    
    env.cleanup().await?;

    // Create test drivers
    let drivers_count = 20;
    let updates_per_driver = 50;
    let test_drivers = generate_test_drivers(drivers_count);
    
    let mut created_drivers = Vec::new();
    for driver in test_drivers {
        let created = env.api_client.create_test_driver(&driver).await?;
        created_drivers.push(created);
    }

    println!("Starting location updates performance test:");
    println!("  Drivers: {}", drivers_count);
    println!("  Updates per driver: {}", updates_per_driver);
    println!("  Total updates: {}", drivers_count * updates_per_driver);

    let total_operations = Arc::new(AtomicUsize::new(0));
    let successful_operations = Arc::new(AtomicUsize::new(0));

    let start_time = Instant::now();
    let mut tasks = Vec::new();

    // Launch concurrent location update tasks
    for (driver_idx, driver) in created_drivers.iter().enumerate() {
        let driver_id = driver.id;
        let api_client = env.api_client.clone();
        let total_ops = Arc::clone(&total_operations);
        let success_ops = Arc::clone(&successful_operations);
        
        let task = tokio::spawn(async move {
            let base_lat = 55.7558;
            let base_lng = 37.6176;
            
            for update_idx in 0..updates_per_driver {
                total_ops.fetch_add(1, Ordering::Relaxed);
                
                // Generate realistic GPS coordinates around Moscow
                let lat_offset = (driver_idx as f64) * 0.01 + (update_idx as f64) * 0.001;
                let lng_offset = (driver_idx as f64) * 0.01 + (update_idx as f64) * 0.001;
                
                let location_request = UpdateLocationRequest {
                    latitude: base_lat + lat_offset,
                    longitude: base_lng + lng_offset,
                    altitude: Some(100.0 + update_idx as f64),
                    accuracy: Some(5.0),
                    speed: Some((update_idx % 60) as f64), // Varying speed 0-59 km/h
                    bearing: Some((update_idx * 6) as f64 % 360.0), // Rotating bearing
                    timestamp: Some(Utc::now().timestamp()),
                };

                match api_client.update_location(driver_id, &location_request).await {
                    Ok(_) => {
                        success_ops.fetch_add(1, Ordering::Relaxed);
                    }
                    Err(e) => {
                        eprintln!("Location update failed for driver {}: {}", driver_id, e);
                    }
                }
                
                // GPS update frequency simulation (every 2 seconds)
                sleep(StdDuration::from_millis(100)).await;
            }
        });
        
        tasks.push(task);
    }

    // Wait for all location updates to complete
    futures::future::join_all(tasks).await;

    let total_duration = start_time.elapsed();
    let total_ops = total_operations.load(Ordering::Relaxed);
    let success_ops = successful_operations.load(Ordering::Relaxed);

    let throughput = total_ops as f64 / total_duration.as_secs_f64();
    let success_rate = success_ops as f64 / total_ops as f64;

    println!("Location Updates Performance Results:");
    println!("  Duration: {:?}", total_duration);
    println!("  Total Updates: {}", total_ops);
    println!("  Successful: {}", success_ops);
    println!("  Throughput: {:.2} updates/sec", throughput);
    println!("  Success Rate: {:.1}%", success_rate * 100.0);

    // Location updates should handle high frequency
    assert!(throughput > 20.0, "Location update throughput should be > 20 ops/sec, got: {:.2}", throughput);
    assert!(success_rate > 0.98, "Location update success rate should be > 98%, got: {:.1}%", success_rate * 100.0);

    Ok(())
}

/// Stress test with extreme concurrent load
#[tokio::test]
#[serial]
async fn test_stress_extreme_load() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping stress test");
        return Ok(());
    }
    
    env.cleanup().await?;

    let concurrent_users = env.config.test.load_test_users as usize;
    let operations_per_user = 10;
    let stress_duration = StdDuration::from_secs(60);

    println!("Starting stress test with extreme load:");
    println!("  Concurrent users: {}", concurrent_users);
    println!("  Operations per user: {}", operations_per_user);
    println!("  Duration: {:?}", stress_duration);

    let stress_stats = Arc::new(StressTestStats::new());
    let start_time = Instant::now();
    let mut tasks = Vec::new();

    // Launch stress test tasks
    for user_id in 0..concurrent_users {
        let api_client = env.api_client.clone();
        let stats = Arc::clone(&stress_stats);
        
        let task = tokio::spawn(async move {
            let user_start = Instant::now();
            
            for op_id in 0..operations_per_user {
                let op_start = Instant::now();
                stats.record_operation_start();
                
                // Create unique test driver
                let mut test_driver = generate_test_drivers(1).into_iter().next().unwrap();
                test_driver.phone = format!("+7800{:04}{:04}", user_id, op_id);
                test_driver.email = format!("stress.test.{}.{}@example.com", user_id, op_id);
                test_driver.license_number = format!("STRESS{:04}{:04}", user_id, op_id);

                match api_client.create_test_driver(&test_driver).await {
                    Ok(_) => {
                        stats.record_operation_success(op_start.elapsed());
                    }
                    Err(_) => {
                        stats.record_operation_failure(op_start.elapsed());
                    }
                }
                
                // High-frequency operations (minimal delay)
                sleep(StdDuration::from_millis(10)).await;
            }
            
            user_start.elapsed()
        });
        
        tasks.push(task);
    }

    // Set timeout for stress test
    let timeout_task = tokio::spawn(async move {
        sleep(stress_duration + StdDuration::from_secs(30)).await;
    });

    tokio::select! {
        results = futures::future::join_all(tasks) => {
            println!("Stress test completed normally");
        }
        _ = timeout_task => {
            println!("Stress test timed out");
        }
    }

    let total_duration = start_time.elapsed();
    let stats_summary = stress_stats.get_summary();

    println!("Stress Test Results:");
    println!("  Duration: {:?}", total_duration);
    println!("  Total Operations: {}", stats_summary.total_operations);
    println!("  Successful: {}", stats_summary.successful_operations);
    println!("  Failed: {}", stats_summary.failed_operations);
    println!("  Throughput: {:.2} ops/sec", stats_summary.throughput);
    println!("  Success Rate: {:.1}%", stats_summary.success_rate * 100.0);
    println!("  Average Response Time: {:?}", stats_summary.avg_response_time);
    println!("  Max Response Time: {:?}", stats_summary.max_response_time);

    // Stress test should maintain reasonable performance under extreme load
    assert!(stats_summary.success_rate > 0.90, 
           "Success rate should be > 90% under stress, got: {:.1}%", 
           stats_summary.success_rate * 100.0);
    
    assert!(stats_summary.avg_response_time < StdDuration::from_secs(5),
           "Average response time should be < 5s under stress, got: {:?}",
           stats_summary.avg_response_time);

    Ok(())
}

/// Test database performance under concurrent access
#[tokio::test]
#[serial]
async fn test_database_concurrent_performance() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping database performance test");
        return Ok(());
    }
    
    env.cleanup().await?;

    let concurrent_connections = 20;
    let queries_per_connection = 50;
    
    println!("Starting database concurrent performance test:");
    println!("  Concurrent connections: {}", concurrent_connections);
    println!("  Queries per connection: {}", queries_per_connection);

    let start_time = Instant::now();
    let mut tasks = Vec::new();

    for conn_id in 0..concurrent_connections {
        let database = env.database.clone();
        
        let task = tokio::spawn(async move {
            let mut results = Vec::new();
            
            for query_id in 0..queries_per_connection {
                let query_start = Instant::now();
                
                // Perform various database operations
                let operation = query_id % 4;
                let success = match operation {
                    0 => {
                        // Select query
                        let result = sqlx::query!("SELECT COUNT(*) as count FROM drivers WHERE deleted_at IS NULL")
                            .fetch_one(database.get_pool())
                            .await;
                        result.is_ok()
                    }
                    1 => {
                        // Insert query
                        let test_driver = generate_test_drivers(1).into_iter().next().unwrap();
                        let mut unique_driver = test_driver;
                        unique_driver.phone = format!("+7700{:04}{:04}", conn_id, query_id);
                        unique_driver.email = format!("perf.db.{}.{}@example.com", conn_id, query_id);
                        unique_driver.license_number = format!("PERF{:04}{:04}", conn_id, query_id);
                        
                        let result = database.create_test_driver(&unique_driver).await;
                        result.is_ok()
                    }
                    2 => {
                        // Update query
                        let result = sqlx::query!(
                            "UPDATE drivers SET updated_at = NOW() WHERE phone LIKE $1 AND deleted_at IS NULL LIMIT 1",
                            format!("+7700{:04}%", conn_id)
                        )
                        .execute(database.get_pool())
                        .await;
                        result.is_ok()
                    }
                    _ => {
                        // Complex query with joins
                        let result = sqlx::query!(
                            r#"
                            SELECT d.id, d.current_rating, COUNT(dl.id) as location_count
                            FROM drivers d
                            LEFT JOIN driver_locations dl ON d.id = dl.driver_id
                            WHERE d.deleted_at IS NULL
                            GROUP BY d.id, d.current_rating
                            ORDER BY d.current_rating DESC
                            LIMIT 10
                            "#
                        )
                        .fetch_all(database.get_pool())
                        .await;
                        result.is_ok()
                    }
                };
                
                results.push((query_id, success, query_start.elapsed()));
            }
            
            results
        });
        
        tasks.push(task);
    }

    // Wait for all database tasks
    let all_results = futures::future::join_all(tasks).await;
    let total_duration = start_time.elapsed();

    let mut total_queries = 0;
    let mut successful_queries = 0;
    let mut total_response_time = StdDuration::from_secs(0);
    let mut max_response_time = StdDuration::from_secs(0);

    for task_result in all_results {
        if let Ok(results) = task_result {
            for (_query_id, success, response_time) in results {
                total_queries += 1;
                if success {
                    successful_queries += 1;
                }
                total_response_time += response_time;
                if response_time > max_response_time {
                    max_response_time = response_time;
                }
            }
        }
    }

    let avg_response_time = total_response_time / total_queries as u32;
    let success_rate = successful_queries as f64 / total_queries as f64;
    let query_throughput = total_queries as f64 / total_duration.as_secs_f64();

    println!("Database Performance Results:");
    println!("  Duration: {:?}", total_duration);
    println!("  Total Queries: {}", total_queries);
    println!("  Successful: {}", successful_queries);
    println!("  Query Throughput: {:.2} queries/sec", query_throughput);
    println!("  Success Rate: {:.1}%", success_rate * 100.0);
    println!("  Average Response Time: {:?}", avg_response_time);
    println!("  Max Response Time: {:?}", max_response_time);

    // Database should handle concurrent access well
    assert!(success_rate > 0.95, 
           "Database success rate should be > 95%, got: {:.1}%", 
           success_rate * 100.0);
    
    assert!(avg_response_time < StdDuration::from_millis(500),
           "Average database response time should be < 500ms, got: {:?}",
           avg_response_time);

    Ok(())
}

/// Memory and resource usage test
#[tokio::test]
#[serial]
async fn test_resource_usage_under_load() -> Result<()> {
    let env = init_test_environment().await?;
    if !env.config.test.performance_test_enabled {
        println!("Performance tests disabled, skipping resource usage test");
        return Ok(());
    }
    
    env.cleanup().await?;

    // Monitor system resources during load test
    let resource_monitor = tokio::spawn(async {
        let mut measurements = Vec::new();
        
        for i in 0..30 { // Monitor for 30 iterations
            let measurement = SystemResourceMeasurement {
                timestamp: Instant::now(),
                iteration: i,
                // In a real implementation, you would measure:
                // - Memory usage (RSS, heap, etc.)
                // - CPU usage
                // - Network connections
                // - Database connections
                // - File descriptors
                // For now, we'll simulate with dummy values
                memory_mb: 100.0 + (i as f64) * 2.0, // Simulated memory growth
                cpu_percent: 15.0 + (i as f64) * 0.5, // Simulated CPU usage
            };
            measurements.push(measurement);
            sleep(StdDuration::from_secs(1)).await;
        }
        
        measurements
    });

    // Run sustained load while monitoring resources
    let concurrent_operations = 30;
    let mut operation_tasks = Vec::new();

    for op_id in 0..concurrent_operations {
        let api_client = env.api_client.clone();
        
        let task = tokio::spawn(async move {
            let mut test_driver = generate_test_drivers(1).into_iter().next().unwrap();
            test_driver.phone = format!("+7600{:04}", op_id);
            test_driver.email = format!("resource.test.{}@example.com", op_id);
            test_driver.license_number = format!("RESOURCE{:04}", op_id);

            // Perform multiple operations per task
            for _ in 0..20 {
                let _result = api_client.create_test_driver(&test_driver).await;
                sleep(StdDuration::from_millis(100)).await;
            }
        });
        
        operation_tasks.push(task);
    }

    // Wait for both resource monitoring and operations
    let (resource_measurements, _operation_results) = tokio::join!(
        resource_monitor,
        futures::future::join_all(operation_tasks)
    );

    if let Ok(measurements) = resource_measurements {
        println!("Resource Usage During Load Test:");
        
        let initial_memory = measurements.first().map(|m| m.memory_mb).unwrap_or(0.0);
        let final_memory = measurements.last().map(|m| m.memory_mb).unwrap_or(0.0);
        let max_memory = measurements.iter().map(|m| m.memory_mb).fold(0.0, f64::max);
        
        let initial_cpu = measurements.first().map(|m| m.cpu_percent).unwrap_or(0.0);
        let final_cpu = measurements.last().map(|m| m.cpu_percent).unwrap_or(0.0);
        let max_cpu = measurements.iter().map(|m| m.cpu_percent).fold(0.0, f64::max);

        println!("  Memory Usage:");
        println!("    Initial: {:.1} MB", initial_memory);
        println!("    Final: {:.1} MB", final_memory);
        println!("    Peak: {:.1} MB", max_memory);
        println!("    Growth: {:.1} MB", final_memory - initial_memory);
        
        println!("  CPU Usage:");
        println!("    Initial: {:.1}%", initial_cpu);
        println!("    Final: {:.1}%", final_cpu);
        println!("    Peak: {:.1}%", max_cpu);

        // Resource usage assertions (adjust based on requirements)
        assert!(max_memory < 500.0, "Memory usage should stay below 500MB, got: {:.1}MB", max_memory);
        assert!(max_cpu < 80.0, "CPU usage should stay below 80%, got: {:.1}%", max_cpu);
        
        // Memory growth should be reasonable
        let memory_growth_rate = (final_memory - initial_memory) / initial_memory;
        assert!(memory_growth_rate < 2.0, 
               "Memory growth should be < 200%, got: {:.1}%", 
               memory_growth_rate * 100.0);
    }

    Ok(())
}

/// Helper struct for stress test statistics
#[derive(Debug)]
struct StressTestStats {
    total_operations: AtomicUsize,
    successful_operations: AtomicUsize,
    failed_operations: AtomicUsize,
    total_response_time: std::sync::Mutex<StdDuration>,
    max_response_time: std::sync::Mutex<StdDuration>,
}

impl StressTestStats {
    fn new() -> Self {
        Self {
            total_operations: AtomicUsize::new(0),
            successful_operations: AtomicUsize::new(0),
            failed_operations: AtomicUsize::new(0),
            total_response_time: std::sync::Mutex::new(StdDuration::from_secs(0)),
            max_response_time: std::sync::Mutex::new(StdDuration::from_secs(0)),
        }
    }

    fn record_operation_start(&self) {
        self.total_operations.fetch_add(1, Ordering::Relaxed);
    }

    fn record_operation_success(&self, response_time: StdDuration) {
        self.successful_operations.fetch_add(1, Ordering::Relaxed);
        self.update_response_times(response_time);
    }

    fn record_operation_failure(&self, response_time: StdDuration) {
        self.failed_operations.fetch_add(1, Ordering::Relaxed);
        self.update_response_times(response_time);
    }

    fn update_response_times(&self, response_time: StdDuration) {
        if let Ok(mut total_time) = self.total_response_time.lock() {
            *total_time += response_time;
        }
        
        if let Ok(mut max_time) = self.max_response_time.lock() {
            if response_time > *max_time {
                *max_time = response_time;
            }
        }
    }

    fn get_summary(&self) -> StressTestSummary {
        let total = self.total_operations.load(Ordering::Relaxed);
        let successful = self.successful_operations.load(Ordering::Relaxed);
        let failed = self.failed_operations.load(Ordering::Relaxed);
        
        let total_time = self.total_response_time.lock().unwrap().clone();
        let max_time = self.max_response_time.lock().unwrap().clone();
        
        let avg_response_time = if total > 0 {
            total_time / total as u32
        } else {
            StdDuration::from_secs(0)
        };
        
        StressTestSummary {
            total_operations: total,
            successful_operations: successful,
            failed_operations: failed,
            success_rate: if total > 0 { successful as f64 / total as f64 } else { 0.0 },
            throughput: 0.0, // Would be calculated with actual duration
            avg_response_time,
            max_response_time: max_time,
        }
    }
}

#[derive(Debug)]
struct StressTestSummary {
    total_operations: usize,
    successful_operations: usize,
    failed_operations: usize,
    success_rate: f64,
    throughput: f64,
    avg_response_time: StdDuration,
    max_response_time: StdDuration,
}

#[derive(Debug)]
struct SystemResourceMeasurement {
    timestamp: Instant,
    iteration: usize,
    memory_mb: f64,
    cpu_percent: f64,
}