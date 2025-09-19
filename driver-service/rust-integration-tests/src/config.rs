use anyhow::Result;
use serde::{Deserialize, Serialize};
use std::env;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestConfig {
    pub driver_service: ServiceConfig,
    pub database: DatabaseConfig,
    pub nats: NatsConfig,
    pub redis: RedisConfig,
    pub test: TestSettings,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceConfig {
    pub base_url: String,
    pub port: u16,
    pub timeout_seconds: u64,
    pub health_check_path: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DatabaseConfig {
    pub host: String,
    pub port: u16,
    pub database: String,
    pub username: String,
    pub password: String,
    pub max_connections: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NatsConfig {
    pub enabled: bool,
    pub url: String,
    pub subjects: NatsSubjects,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NatsSubjects {
    pub driver_registered: String,
    pub driver_verified: String,
    pub driver_status_changed: String,
    pub driver_location_updated: String,
    pub driver_rating_updated: String,
    pub order_assigned: String,
    pub order_completed: String,
    pub payment_processed: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RedisConfig {
    pub enabled: bool,
    pub url: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestSettings {
    pub cleanup_after_each: bool,
    pub parallel_execution: bool,
    pub max_test_duration_seconds: u64,
    pub performance_test_enabled: bool,
    pub load_test_users: u32,
}

impl TestConfig {
    pub fn load() -> Result<Self> {
        dotenv::dotenv().ok();

        Ok(Self {
            driver_service: ServiceConfig {
                base_url: env::var("DRIVER_SERVICE_BASE_URL")
                    .unwrap_or_else(|_| "http://localhost:8001".to_string()),
                port: env::var("DRIVER_SERVICE_PORT")
                    .unwrap_or_else(|_| "8001".to_string())
                    .parse()?,
                timeout_seconds: env::var("HTTP_TIMEOUT_SECONDS")
                    .unwrap_or_else(|_| "30".to_string())
                    .parse()?,
                health_check_path: "/health".to_string(),
            },
            database: DatabaseConfig {
                host: env::var("POSTGRES_HOST")
                    .unwrap_or_else(|_| "localhost".to_string()),
                port: env::var("POSTGRES_PORT")
                    .unwrap_or_else(|_| "5433".to_string())
                    .parse()?,
                database: env::var("POSTGRES_DB")
                    .unwrap_or_else(|_| "driver_service_test".to_string()),
                username: env::var("POSTGRES_USER")
                    .unwrap_or_else(|_| "test_user".to_string()),
                password: env::var("POSTGRES_PASSWORD")
                    .unwrap_or_else(|_| "test_password".to_string()),
                max_connections: env::var("DB_MAX_CONNECTIONS")
                    .unwrap_or_else(|_| "10".to_string())
                    .parse()?,
            },
            nats: NatsConfig {
                enabled: env::var("NATS_ENABLED")
                    .unwrap_or_else(|_| "false".to_string())
                    .parse()?,
                url: env::var("NATS_URL")
                    .unwrap_or_else(|_| "nats://localhost:4222".to_string()),
                subjects: NatsSubjects {
                    driver_registered: "driver.registered".to_string(),
                    driver_verified: "driver.verified".to_string(),
                    driver_status_changed: "driver.status.changed".to_string(),
                    driver_location_updated: "driver.location.updated".to_string(),
                    driver_rating_updated: "driver.rating.updated".to_string(),
                    order_assigned: "order.assigned".to_string(),
                    order_completed: "order.completed".to_string(),
                    payment_processed: "payment.processed".to_string(),
                },
            },
            redis: RedisConfig {
                enabled: env::var("REDIS_ENABLED")
                    .unwrap_or_else(|_| "false".to_string())
                    .parse()?,
                url: env::var("REDIS_URL")
                    .unwrap_or_else(|_| "redis://localhost:6380".to_string()),
            },
            test: TestSettings {
                cleanup_after_each: env::var("TEST_CLEANUP_AFTER_EACH")
                    .unwrap_or_else(|_| "true".to_string())
                    .parse()?,
                parallel_execution: env::var("TEST_PARALLEL_EXECUTION")
                    .unwrap_or_else(|_| "false".to_string())
                    .parse()?,
                max_test_duration_seconds: env::var("TEST_MAX_DURATION_SECONDS")
                    .unwrap_or_else(|_| "300".to_string())
                    .parse()?,
                performance_test_enabled: env::var("PERFORMANCE_TESTS_ENABLED")
                    .unwrap_or_else(|_| "false".to_string())
                    .parse()?,
                load_test_users: env::var("LOAD_TEST_USERS")
                    .unwrap_or_else(|_| "100".to_string())
                    .parse()?,
            },
        })
    }

    pub fn database_url(&self) -> String {
        format!(
            "postgresql://{}:{}@{}:{}/{}",
            self.database.username,
            self.database.password,
            self.database.host,
            self.database.port,
            self.database.database
        )
    }

    pub fn service_api_url(&self) -> String {
        format!("{}/api/v1", self.driver_service.base_url)
    }

    pub fn service_health_url(&self) -> String {
        format!("{}{}", self.driver_service.base_url, self.driver_service.health_check_path)
    }
}