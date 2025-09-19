use anyhow::{anyhow, Result};
use std::collections::HashMap;
use std::process::{Command, Stdio};
use std::time::Duration;
use testcontainers::{clients, Container, Docker, Image};
use testcontainers_modules::{postgres::Postgres, redis::Redis};
use tokio::time::sleep;
use tracing::{debug, error, info, warn};

use crate::config::TestConfig;

pub struct DockerHelper {
    docker: clients::Cli,
    containers: HashMap<String, Box<dyn ContainerHandle>>,
}

trait ContainerHandle {
    fn stop(&mut self) -> Result<()>;
}

struct PostgresHandle {
    container: Container<'static, Postgres>,
}

struct RedisHandle {
    container: Container<'static, Redis>,
}

impl ContainerHandle for PostgresHandle {
    fn stop(&mut self) -> Result<()> {
        // Container automatically stops when dropped
        Ok(())
    }
}

impl ContainerHandle for RedisHandle {
    fn stop(&mut self) -> Result<()> {
        // Container automatically stops when dropped
        Ok(())
    }
}

impl DockerHelper {
    pub async fn new() -> Result<Self> {
        let docker = clients::Cli::default();
        
        Ok(Self {
            docker,
            containers: HashMap::new(),
        })
    }

    /// Start test services using Docker Compose
    pub async fn start_test_services(&self, config: &TestConfig) -> Result<()> {
        info!("Starting test services with Docker Compose");

        // Check if docker-compose.test.yml exists
        let compose_file = "../docker-compose.test.yml";
        if !std::path::Path::new(compose_file).exists() {
            warn!("Docker compose test file not found, skipping service startup");
            return Ok(());
        }

        // Start services
        let output = Command::new("docker-compose")
            .args(&["-f", compose_file, "up", "-d", "--remove-orphans"])
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .output()?;

        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            error!("Failed to start test services: {}", stderr);
            return Err(anyhow!("Docker compose failed: {}", stderr));
        }

        info!("Test services started successfully");
        
        // Wait for services to be ready
        self.wait_for_services(config).await?;
        
        Ok(())
    }

    /// Wait for services to become ready
    pub async fn wait_for_services(&self, config: &TestConfig) -> Result<()> {
        info!("Waiting for test services to be ready");

        // Wait for PostgreSQL
        self.wait_for_postgres(config).await?;

        // Wait for Redis if enabled
        if config.redis.enabled {
            self.wait_for_redis(config).await?;
        }

        info!("All test services are ready");
        Ok(())
    }

    /// Wait for PostgreSQL to be ready
    async fn wait_for_postgres(&self, config: &TestConfig) -> Result<()> {
        let max_attempts = 30;
        let delay = Duration::from_secs(2);

        for attempt in 1..=max_attempts {
            match self.check_postgres_health(config).await {
                Ok(_) => {
                    info!("PostgreSQL is ready after {} attempts", attempt);
                    return Ok(());
                }
                Err(e) => {
                    debug!("PostgreSQL health check attempt {}/{} failed: {}", attempt, max_attempts, e);
                    if attempt == max_attempts {
                        return Err(anyhow!("PostgreSQL not ready after {} attempts", max_attempts));
                    }
                    sleep(delay).await;
                }
            }
        }

        Ok(())
    }

    /// Wait for Redis to be ready
    async fn wait_for_redis(&self, config: &TestConfig) -> Result<()> {
        let max_attempts = 15;
        let delay = Duration::from_secs(1);

        for attempt in 1..=max_attempts {
            match self.check_redis_health(config).await {
                Ok(_) => {
                    info!("Redis is ready after {} attempts", attempt);
                    return Ok(());
                }
                Err(e) => {
                    debug!("Redis health check attempt {}/{} failed: {}", attempt, max_attempts, e);
                    if attempt == max_attempts {
                        return Err(anyhow!("Redis not ready after {} attempts", max_attempts));
                    }
                    sleep(delay).await;
                }
            }
        }

        Ok(())
    }

    /// Check PostgreSQL health
    async fn check_postgres_health(&self, config: &TestConfig) -> Result<()> {
        let output = Command::new("pg_isready")
            .args(&[
                "-h", &config.database.host,
                "-p", &config.database.port.to_string(),
                "-U", &config.database.username,
                "-d", &config.database.database,
            ])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()?;

        if output.success() {
            Ok(())
        } else {
            Err(anyhow!("PostgreSQL not ready"))
        }
    }

    /// Check Redis health
    async fn check_redis_health(&self, config: &TestConfig) -> Result<()> {
        let output = Command::new("redis-cli")
            .args(&["-u", &config.redis.url, "ping"])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()?;

        if output.success() {
            Ok(())
        } else {
            Err(anyhow!("Redis not ready"))
        }
    }

    /// Start individual PostgreSQL container (for testing without compose)
    pub async fn start_postgres_container(&mut self) -> Result<(String, u16)> {
        info!("Starting PostgreSQL test container");

        let postgres_image = Postgres::default()
            .with_db_name("driver_service_test")
            .with_user("test_user")
            .with_password("test_password");

        let container = self.docker.run(postgres_image);
        let host_port = container.get_host_port_ipv4(5432);

        let handle = PostgresHandle { container };
        self.containers.insert("postgres".to_string(), Box::new(handle));

        info!("PostgreSQL container started on port {}", host_port);
        Ok(("localhost".to_string(), host_port))
    }

    /// Start individual Redis container (for testing without compose)
    pub async fn start_redis_container(&mut self) -> Result<(String, u16)> {
        info!("Starting Redis test container");

        let redis_image = Redis::default();
        let container = self.docker.run(redis_image);
        let host_port = container.get_host_port_ipv4(6379);

        let handle = RedisHandle { container };
        self.containers.insert("redis".to_string(), Box::new(handle));

        info!("Redis container started on port {}", host_port);
        Ok(("localhost".to_string(), host_port))
    }

    /// Stop test services
    pub async fn stop_test_services(&self) -> Result<()> {
        info!("Stopping test services");

        let compose_file = "../docker-compose.test.yml";
        if std::path::Path::new(compose_file).exists() {
            let output = Command::new("docker-compose")
                .args(&["-f", compose_file, "down", "-v"])
                .stdout(Stdio::piped())
                .stderr(Stdio::piped())
                .output()?;

            if !output.status.success() {
                let stderr = String::from_utf8_lossy(&output.stderr);
                warn!("Failed to stop test services cleanly: {}", stderr);
            } else {
                info!("Test services stopped successfully");
            }
        }

        Ok(())
    }

    /// Clean up all test containers and networks
    pub async fn cleanup(&self) -> Result<()> {
        info!("Cleaning up Docker test resources");

        // Remove test containers
        let output = Command::new("docker")
            .args(&["ps", "-aq", "--filter", "name=driver-service-test"])
            .stdout(Stdio::piped())
            .output()?;

        if output.status.success() {
            let container_ids = String::from_utf8_lossy(&output.stdout);
            let ids: Vec<&str> = container_ids.lines().collect();

            if !ids.is_empty() {
                Command::new("docker")
                    .args(&["rm", "-f"])
                    .args(&ids)
                    .stdout(Stdio::null())
                    .status()?;
            }
        }

        // Remove test volumes
        let output = Command::new("docker")
            .args(&["volume", "ls", "-q", "--filter", "name=driver-service-test"])
            .stdout(Stdio::piped())
            .output()?;

        if output.status.success() {
            let volume_names = String::from_utf8_lossy(&output.stdout);
            let names: Vec<&str> = volume_names.lines().collect();

            if !names.is_empty() {
                Command::new("docker")
                    .args(&["volume", "rm"])
                    .args(&names)
                    .stdout(Stdio::null())
                    .status()?;
            }
        }

        // Remove test networks
        let output = Command::new("docker")
            .args(&["network", "ls", "-q", "--filter", "name=driver-service-test"])
            .stdout(Stdio::piped())
            .output()?;

        if output.status.success() {
            let network_names = String::from_utf8_lossy(&output.stdout);
            let names: Vec<&str> = network_names.lines().collect();

            if !names.is_empty() {
                Command::new("docker")
                    .args(&["network", "rm"])
                    .args(&names)
                    .stdout(Stdio::null())
                    .status()?;
            }
        }

        info!("Docker cleanup completed");
        Ok(())
    }

    /// Check if Docker is available
    pub fn is_docker_available() -> bool {
        Command::new("docker")
            .args(&["version"])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()
            .map(|status| status.success())
            .unwrap_or(false)
    }

    /// Check if Docker Compose is available
    pub fn is_docker_compose_available() -> bool {
        Command::new("docker-compose")
            .args(&["version"])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status()
            .map(|status| status.success())
            .unwrap_or(false)
    }

    /// Get service logs
    pub async fn get_service_logs(&self, service_name: &str) -> Result<String> {
        let compose_file = "../docker-compose.test.yml";
        
        let output = Command::new("docker-compose")
            .args(&["-f", compose_file, "logs", service_name])
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .output()?;

        if output.status.success() {
            Ok(String::from_utf8_lossy(&output.stdout).to_string())
        } else {
            let stderr = String::from_utf8_lossy(&output.stderr);
            Err(anyhow!("Failed to get logs for {}: {}", service_name, stderr))
        }
    }

    /// Execute command in service container
    pub async fn exec_in_service(&self, service_name: &str, command: &[&str]) -> Result<String> {
        let compose_file = "../docker-compose.test.yml";
        
        let mut cmd = Command::new("docker-compose");
        cmd.args(&["-f", compose_file, "exec", "-T", service_name]);
        cmd.args(command);
        
        let output = cmd
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .output()?;

        if output.status.success() {
            Ok(String::from_utf8_lossy(&output.stdout).to_string())
        } else {
            let stderr = String::from_utf8_lossy(&output.stderr);
            Err(anyhow!("Failed to execute command in {}: {}", service_name, stderr))
        }
    }

    /// Restart service
    pub async fn restart_service(&self, service_name: &str) -> Result<()> {
        let compose_file = "../docker-compose.test.yml";
        
        let output = Command::new("docker-compose")
            .args(&["-f", compose_file, "restart", service_name])
            .stdout(Stdio::piped())
            .stderr(Stdio::piped())
            .output()?;

        if output.status.success() {
            info!("Service {} restarted successfully", service_name);
            Ok(())
        } else {
            let stderr = String::from_utf8_lossy(&output.stderr);
            Err(anyhow!("Failed to restart {}: {}", service_name, stderr))
        }
    }
}

impl Drop for DockerHelper {
    fn drop(&mut self) {
        // Cleanup is handled by the async cleanup method
        // Individual containers will be stopped automatically
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_docker_availability() {
        let available = DockerHelper::is_docker_available();
        println!("Docker available: {}", available);
    }

    #[tokio::test]
    async fn test_docker_compose_availability() {
        let available = DockerHelper::is_docker_compose_available();
        println!("Docker Compose available: {}", available);
    }
}