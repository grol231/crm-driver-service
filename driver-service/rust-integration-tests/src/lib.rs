pub mod config;
pub mod fixtures;
pub mod helpers;
pub mod clients;

// Test modules
pub mod tests {
    pub mod driver_api_tests;
    pub mod location_api_tests;
    pub mod database_tests;
    pub mod event_tests;
    pub mod performance_tests;
    pub mod integration_scenarios;
}

// Re-exports for easier testing
pub use config::TestConfig;
pub use fixtures::*;
pub use helpers::*;
pub use clients::*;

use anyhow::Result;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

/// Initialize test environment
pub async fn init_test_environment() -> Result<TestEnvironment> {
    // Initialize logging
    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::new(
            std::env::var("RUST_LOG").unwrap_or_else(|_| "info,driver_service_integration_tests=debug".into()),
        ))
        .with(tracing_subscriber::fmt::layer())
        .init();

    // Load configuration
    let config = TestConfig::load()?;
    
    // Initialize test environment
    TestEnvironment::new(config).await
}

/// Test environment that manages all test infrastructure
pub struct TestEnvironment {
    pub config: TestConfig,
    pub api_client: ApiClient,
    pub database: DatabaseHelper,
    pub nats_client: Option<NatsClient>,
    pub redis_client: Option<RedisHelper>,
    pub docker_helper: DockerHelper,
}

impl TestEnvironment {
    pub async fn new(config: TestConfig) -> Result<Self> {
        // Initialize Docker helper
        let docker_helper = DockerHelper::new().await?;
        
        // Start test containers if needed
        docker_helper.start_test_services(&config).await?;

        // Initialize database connection
        let database = DatabaseHelper::new(&config).await?;
        
        // Run migrations if needed
        database.ensure_migrations().await?;

        // Initialize API client
        let api_client = ApiClient::new(&config)?;
        
        // Initialize NATS client if configured
        let nats_client = if config.nats.enabled {
            Some(NatsClient::new(&config.nats.url).await?)
        } else {
            None
        };

        // Initialize Redis client if configured
        let redis_client = if config.redis.enabled {
            Some(RedisHelper::new(&config.redis.url).await?)
        } else {
            None
        };

        Ok(Self {
            config,
            api_client,
            database,
            nats_client,
            redis_client,
            docker_helper,
        })
    }

    /// Clean up test data between tests
    pub async fn cleanup(&self) -> Result<()> {
        self.database.cleanup_test_data().await?;
        
        if let Some(redis) = &self.redis_client {
            redis.flush_db().await?;
        }
        
        Ok(())
    }

    /// Setup test data
    pub async fn setup_test_data(&self) -> Result<TestDataSet> {
        let test_data = TestDataSet::new();
        
        // Create test drivers
        for driver in &test_data.drivers {
            self.database.create_test_driver(driver).await?;
        }
        
        Ok(test_data)
    }
}

impl Drop for TestEnvironment {
    fn drop(&mut self) {
        // Cleanup is handled by tokio runtime
        tokio::spawn(async move {
            // Any async cleanup would go here
        });
    }
}

#[derive(Debug)]
pub struct TestDataSet {
    pub drivers: Vec<TestDriver>,
    pub documents: Vec<TestDocument>,
    pub locations: Vec<TestLocation>,
    pub shifts: Vec<TestShift>,
    pub ratings: Vec<TestRating>,
}

impl TestDataSet {
    pub fn new() -> Self {
        Self {
            drivers: generate_test_drivers(5),
            documents: vec![],
            locations: vec![],
            shifts: vec![],
            ratings: vec![],
        }
    }
}