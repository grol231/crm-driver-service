use anyhow::Result;
use std::time::{Duration, Instant};
use tokio::time::sleep;
use tracing::{debug, info, warn};

/// Test timeout wrapper
pub async fn with_timeout<F, T>(
    operation: F,
    timeout: Duration,
    description: &str,
) -> Result<T>
where
    F: std::future::Future<Output = Result<T>>,
{
    debug!("Starting operation: {}", description);
    let start_time = Instant::now();
    
    match tokio::time::timeout(timeout, operation).await {
        Ok(result) => {
            let elapsed = start_time.elapsed();
            debug!("Operation completed: {} in {:?}", description, elapsed);
            result
        }
        Err(_) => {
            warn!("Operation timed out: {} after {:?}", description, timeout);
            Err(anyhow::anyhow!("Operation '{}' timed out after {:?}", description, timeout))
        }
    }
}

/// Retry wrapper for flaky operations
pub async fn retry_with_backoff<F, T, E>(
    mut operation: F,
    max_attempts: u32,
    initial_delay: Duration,
    max_delay: Duration,
    description: &str,
) -> Result<T>
where
    F: FnMut() -> std::future::Ready<Result<T, E>> + Send,
    E: std::fmt::Display + Send + Sync + 'static,
{
    let mut delay = initial_delay;
    
    for attempt in 1..=max_attempts {
        debug!("Attempt {}/{} for: {}", attempt, max_attempts, description);
        
        match operation().await {
            Ok(result) => {
                if attempt > 1 {
                    info!("Operation succeeded on attempt {}: {}", attempt, description);
                }
                return Ok(result);
            }
            Err(e) => {
                warn!("Attempt {}/{} failed for '{}': {}", attempt, max_attempts, description, e);
                
                if attempt == max_attempts {
                    return Err(anyhow::anyhow!(
                        "Operation '{}' failed after {} attempts: {}",
                        description,
                        max_attempts,
                        e
                    ));
                }
                
                sleep(delay).await;
                delay = std::cmp::min(delay * 2, max_delay);
            }
        }
    }
    
    unreachable!()
}

/// Performance measurement utilities
#[derive(Debug, Clone)]
pub struct PerformanceMeasurement {
    pub name: String,
    pub start_time: Instant,
    pub end_time: Option<Instant>,
    pub operations: usize,
}

impl PerformanceMeasurement {
    pub fn start(name: &str, operations: usize) -> Self {
        debug!("Starting performance measurement: {} ({} operations)", name, operations);
        
        Self {
            name: name.to_string(),
            start_time: Instant::now(),
            end_time: None,
            operations,
        }
    }
    
    pub fn finish(&mut self) -> Duration {
        self.end_time = Some(Instant::now());
        let duration = self.duration();
        
        info!(
            "Performance measurement completed: {} - {} operations in {:?} ({:.2} ops/sec)",
            self.name,
            self.operations,
            duration,
            self.operations_per_second()
        );
        
        duration
    }
    
    pub fn duration(&self) -> Duration {
        match self.end_time {
            Some(end) => end - self.start_time,
            None => Instant::now() - self.start_time,
        }
    }
    
    pub fn operations_per_second(&self) -> f64 {
        let duration = self.duration();
        if duration.as_secs_f64() > 0.0 {
            self.operations as f64 / duration.as_secs_f64()
        } else {
            0.0
        }
    }
}

/// Test result aggregator
#[derive(Debug, Default)]
pub struct TestResults {
    pub passed: Vec<String>,
    pub failed: Vec<(String, String)>, // test name, error message
    pub skipped: Vec<String>,
    pub performance_measurements: Vec<PerformanceMeasurement>,
}

impl TestResults {
    pub fn new() -> Self {
        Self::default()
    }
    
    pub fn add_pass(&mut self, test_name: &str) {
        debug!("Test passed: {}", test_name);
        self.passed.push(test_name.to_string());
    }
    
    pub fn add_fail(&mut self, test_name: &str, error: &str) {
        warn!("Test failed: {} - {}", test_name, error);
        self.failed.push((test_name.to_string(), error.to_string()));
    }
    
    pub fn add_skip(&mut self, test_name: &str) {
        info!("Test skipped: {}", test_name);
        self.skipped.push(test_name.to_string());
    }
    
    pub fn add_measurement(&mut self, measurement: PerformanceMeasurement) {
        self.performance_measurements.push(measurement);
    }
    
    pub fn total_tests(&self) -> usize {
        self.passed.len() + self.failed.len() + self.skipped.len()
    }
    
    pub fn success_rate(&self) -> f64 {
        let total = self.total_tests();
        if total == 0 {
            0.0
        } else {
            self.passed.len() as f64 / total as f64
        }
    }
    
    pub fn print_summary(&self) {
        info!("Test Results Summary:");
        info!("  Total: {}", self.total_tests());
        info!("  Passed: {}", self.passed.len());
        info!("  Failed: {}", self.failed.len());
        info!("  Skipped: {}", self.skipped.len());
        info!("  Success Rate: {:.1}%", self.success_rate() * 100.0);
        
        if !self.failed.is_empty() {
            warn!("Failed Tests:");
            for (name, error) in &self.failed {
                warn!("  {} - {}", name, error);
            }
        }
        
        if !self.performance_measurements.is_empty() {
            info!("Performance Measurements:");
            for measurement in &self.performance_measurements {
                info!(
                    "  {} - {} ops in {:?} ({:.2} ops/sec)",
                    measurement.name,
                    measurement.operations,
                    measurement.duration(),
                    measurement.operations_per_second()
                );
            }
        }
    }
}

/// Wait for condition with timeout
pub async fn wait_for_condition<F>(
    mut condition: F,
    timeout: Duration,
    check_interval: Duration,
    description: &str,
) -> Result<()>
where
    F: FnMut() -> std::future::Ready<bool> + Send,
{
    let start_time = Instant::now();
    
    while start_time.elapsed() < timeout {
        if condition().await {
            debug!("Condition met: {}", description);
            return Ok(());
        }
        
        sleep(check_interval).await;
    }
    
    Err(anyhow::anyhow!(
        "Condition not met within timeout: {} ({:?})",
        description,
        timeout
    ))
}

/// Generate test UUID with prefix
pub fn generate_test_id(prefix: &str) -> String {
    format!("{}_{}", prefix, uuid::Uuid::new_v4().as_simple())
}

/// Calculate distance between two coordinates (Haversine formula)
pub fn calculate_distance(lat1: f64, lon1: f64, lat2: f64, lon2: f64) -> f64 {
    const R: f64 = 6371.0; // Earth radius in kilometers
    
    let lat1_rad = lat1.to_radians();
    let lat2_rad = lat2.to_radians();
    let delta_lat = (lat2 - lat1).to_radians();
    let delta_lon = (lon2 - lon1).to_radians();
    
    let a = (delta_lat / 2.0).sin().powi(2) +
        lat1_rad.cos() * lat2_rad.cos() * (delta_lon / 2.0).sin().powi(2);
    let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());
    
    R * c
}

/// Test data cleanup tracker
#[derive(Debug, Default)]
pub struct CleanupTracker {
    pub drivers_to_cleanup: Vec<uuid::Uuid>,
    pub locations_to_cleanup: Vec<uuid::Uuid>,
    pub shifts_to_cleanup: Vec<uuid::Uuid>,
    pub ratings_to_cleanup: Vec<uuid::Uuid>,
    pub redis_keys_to_cleanup: Vec<String>,
}

impl CleanupTracker {
    pub fn new() -> Self {
        Self::default()
    }
    
    pub fn track_driver(&mut self, driver_id: uuid::Uuid) {
        self.drivers_to_cleanup.push(driver_id);
    }
    
    pub fn track_location(&mut self, location_id: uuid::Uuid) {
        self.locations_to_cleanup.push(location_id);
    }
    
    pub fn track_shift(&mut self, shift_id: uuid::Uuid) {
        self.shifts_to_cleanup.push(shift_id);
    }
    
    pub fn track_rating(&mut self, rating_id: uuid::Uuid) {
        self.ratings_to_cleanup.push(rating_id);
    }
    
    pub fn track_redis_key(&mut self, key: String) {
        self.redis_keys_to_cleanup.push(key);
    }
    
    pub fn cleanup_count(&self) -> usize {
        self.drivers_to_cleanup.len() +
        self.locations_to_cleanup.len() +
        self.shifts_to_cleanup.len() +
        self.ratings_to_cleanup.len() +
        self.redis_keys_to_cleanup.len()
    }
}

/// Load test configuration
#[derive(Debug, Clone)]
pub struct LoadTestConfig {
    pub concurrent_users: usize,
    pub duration: Duration,
    pub ramp_up_time: Duration,
    pub operations_per_user: usize,
}

impl Default for LoadTestConfig {
    fn default() -> Self {
        Self {
            concurrent_users: 10,
            duration: Duration::from_secs(60),
            ramp_up_time: Duration::from_secs(10),
            operations_per_user: 100,
        }
    }
}

/// Test environment verification
pub async fn verify_test_environment(config: &crate::config::TestConfig) -> Result<EnvironmentStatus> {
    let mut status = EnvironmentStatus::new();
    
    // Check service availability
    let api_client = crate::ApiClient::new(config)?;
    match api_client.health_check().await {
        Ok(_) => status.add_check("api_service", true),
        Err(e) => {
            status.add_check("api_service", false);
            status.add_issue(format!("API service not available: {}", e));
        }
    }
    
    // Check database connectivity
    match crate::DatabaseHelper::new(config).await {
        Ok(_) => status.add_check("database", true),
        Err(e) => {
            status.add_check("database", false);
            status.add_issue(format!("Database not available: {}", e));
        }
    }
    
    // Check Redis if enabled
    if config.redis.enabled {
        match crate::RedisHelper::new(&config.redis.url).await {
            Ok(_) => status.add_check("redis", true),
            Err(e) => {
                status.add_check("redis", false);
                status.add_issue(format!("Redis not available: {}", e));
            }
        }
    }
    
    // Check NATS if enabled
    if config.nats.enabled {
        match crate::NatsClient::new(&config.nats.url).await {
            Ok(_) => status.add_check("nats", true),
            Err(e) => {
                status.add_check("nats", false);
                status.add_issue(format!("NATS not available: {}", e));
            }
        }
    }
    
    Ok(status)
}

#[derive(Debug)]
pub struct EnvironmentStatus {
    pub checks: std::collections::HashMap<String, bool>,
    pub issues: Vec<String>,
}

impl EnvironmentStatus {
    pub fn new() -> Self {
        Self {
            checks: std::collections::HashMap::new(),
            issues: Vec::new(),
        }
    }
    
    pub fn add_check(&mut self, component: &str, passed: bool) {
        self.checks.insert(component.to_string(), passed);
    }
    
    pub fn add_issue(&mut self, issue: String) {
        self.issues.push(issue);
    }
    
    pub fn is_healthy(&self) -> bool {
        self.checks.values().all(|&passed| passed) && self.issues.is_empty()
    }
    
    pub fn print_status(&self) {
        info!("Environment Status:");
        for (component, passed) in &self.checks {
            let status = if *passed { "✓" } else { "✗" };
            info!("  {} {}", status, component);
        }
        
        if !self.issues.is_empty() {
            warn!("Issues:");
            for issue in &self.issues {
                warn!("  - {}", issue);
            }
        }
    }
}