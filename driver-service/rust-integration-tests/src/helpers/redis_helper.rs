use anyhow::{anyhow, Result};
use redis::{AsyncCommands, Client, Connection};
use serde::{Deserialize, Serialize};
use std::time::Duration;
use tracing::{debug, error, info};

#[derive(Debug, Clone)]
pub struct RedisHelper {
    client: Client,
}

impl RedisHelper {
    pub async fn new(redis_url: &str) -> Result<Self> {
        let client = Client::open(redis_url)
            .map_err(|e| anyhow!("Failed to create Redis client: {}", e))?;

        // Test connection
        let mut conn = client.get_async_connection().await
            .map_err(|e| anyhow!("Failed to connect to Redis: {}", e))?;
            
        redis::cmd("PING").query_async::<_, String>(&mut conn).await
            .map_err(|e| anyhow!("Redis ping failed: {}", e))?;

        info!("Connected to Redis at {}", redis_url);

        Ok(Self { client })
    }

    /// Get async connection
    async fn get_connection(&self) -> Result<redis::aio::Connection> {
        self.client.get_async_connection().await
            .map_err(|e| anyhow!("Failed to get Redis connection: {}", e))
    }

    /// Set key-value pair
    pub async fn set<T: Serialize>(&self, key: &str, value: &T, ttl: Option<Duration>) -> Result<()> {
        let mut conn = self.get_connection().await?;
        let serialized = serde_json::to_string(value)?;

        match ttl {
            Some(duration) => {
                conn.set_ex(key, serialized, duration.as_secs()).await
            }
            None => {
                conn.set(key, serialized).await
            }
        }
        .map_err(|e| anyhow!("Failed to set key '{}': {}", key, e))?;

        debug!("Set Redis key: {}", key);
        Ok(())
    }

    /// Get value by key
    pub async fn get<T: for<'a> Deserialize<'a>>(&self, key: &str) -> Result<Option<T>> {
        let mut conn = self.get_connection().await?;
        
        let value: Option<String> = conn.get(key).await
            .map_err(|e| anyhow!("Failed to get key '{}': {}", key, e))?;

        match value {
            Some(serialized) => {
                let deserialized = serde_json::from_str(&serialized)
                    .map_err(|e| anyhow!("Failed to deserialize value for key '{}': {}", key, e))?;
                Ok(Some(deserialized))
            }
            None => Ok(None),
        }
    }

    /// Delete key
    pub async fn delete(&self, key: &str) -> Result<bool> {
        let mut conn = self.get_connection().await?;
        
        let deleted: i32 = conn.del(key).await
            .map_err(|e| anyhow!("Failed to delete key '{}': {}", key, e))?;

        debug!("Deleted Redis key: {} (existed: {})", key, deleted > 0);
        Ok(deleted > 0)
    }

    /// Check if key exists
    pub async fn exists(&self, key: &str) -> Result<bool> {
        let mut conn = self.get_connection().await?;
        
        let exists: bool = conn.exists(key).await
            .map_err(|e| anyhow!("Failed to check existence of key '{}': {}", key, e))?;

        Ok(exists)
    }

    /// Set key expiration
    pub async fn expire(&self, key: &str, ttl: Duration) -> Result<bool> {
        let mut conn = self.get_connection().await?;
        
        let result: bool = conn.expire(key, ttl.as_secs() as usize).await
            .map_err(|e| anyhow!("Failed to set expiration for key '{}': {}", key, e))?;

        debug!("Set expiration for Redis key: {} -> {}s", key, ttl.as_secs());
        Ok(result)
    }

    /// Get time to live for key
    pub async fn ttl(&self, key: &str) -> Result<i32> {
        let mut conn = self.get_connection().await?;
        
        let ttl: i32 = conn.ttl(key).await
            .map_err(|e| anyhow!("Failed to get TTL for key '{}': {}", key, e))?;

        Ok(ttl)
    }

    /// Add item to list (left push)
    pub async fn list_push<T: Serialize>(&self, key: &str, value: &T) -> Result<usize> {
        let mut conn = self.get_connection().await?;
        let serialized = serde_json::to_string(value)?;

        let length: usize = conn.lpush(key, serialized).await
            .map_err(|e| anyhow!("Failed to push to list '{}': {}", key, e))?;

        debug!("Pushed to Redis list: {} (new length: {})", key, length);
        Ok(length)
    }

    /// Get items from list
    pub async fn list_range<T: for<'a> Deserialize<'a>>(&self, key: &str, start: isize, stop: isize) -> Result<Vec<T>> {
        let mut conn = self.get_connection().await?;
        
        let items: Vec<String> = conn.lrange(key, start, stop).await
            .map_err(|e| anyhow!("Failed to get range from list '{}': {}", key, e))?;

        let mut result = Vec::new();
        for item in items {
            let deserialized = serde_json::from_str(&item)
                .map_err(|e| anyhow!("Failed to deserialize item from list '{}': {}", key, e))?;
            result.push(deserialized);
        }

        Ok(result)
    }

    /// Get list length
    pub async fn list_length(&self, key: &str) -> Result<usize> {
        let mut conn = self.get_connection().await?;
        
        let length: usize = conn.llen(key).await
            .map_err(|e| anyhow!("Failed to get length of list '{}': {}", key, e))?;

        Ok(length)
    }

    /// Add item to set
    pub async fn set_add<T: Serialize>(&self, key: &str, value: &T) -> Result<bool> {
        let mut conn = self.get_connection().await?;
        let serialized = serde_json::to_string(value)?;

        let added: bool = conn.sadd(key, serialized).await
            .map_err(|e| anyhow!("Failed to add to set '{}': {}", key, e))?;

        debug!("Added to Redis set: {} (was new: {})", key, added);
        Ok(added)
    }

    /// Get all items from set
    pub async fn set_members<T: for<'a> Deserialize<'a>>(&self, key: &str) -> Result<Vec<T>> {
        let mut conn = self.get_connection().await?;
        
        let members: Vec<String> = conn.smembers(key).await
            .map_err(|e| anyhow!("Failed to get members of set '{}': {}", key, e))?;

        let mut result = Vec::new();
        for member in members {
            let deserialized = serde_json::from_str(&member)
                .map_err(|e| anyhow!("Failed to deserialize member from set '{}': {}", key, e))?;
            result.push(deserialized);
        }

        Ok(result)
    }

    /// Check if item is in set
    pub async fn set_contains<T: Serialize>(&self, key: &str, value: &T) -> Result<bool> {
        let mut conn = self.get_connection().await?;
        let serialized = serde_json::to_string(value)?;

        let is_member: bool = conn.sismember(key, serialized).await
            .map_err(|e| anyhow!("Failed to check membership in set '{}': {}", key, e))?;

        Ok(is_member)
    }

    /// Increment counter
    pub async fn increment(&self, key: &str, increment: i64) -> Result<i64> {
        let mut conn = self.get_connection().await?;
        
        let new_value: i64 = if increment == 1 {
            conn.incr(key, 1).await
        } else {
            conn.incr(key, increment).await
        }
        .map_err(|e| anyhow!("Failed to increment key '{}': {}", key, e))?;

        debug!("Incremented Redis key: {} by {} -> {}", key, increment, new_value);
        Ok(new_value)
    }

    /// Get multiple keys
    pub async fn get_multiple<T: for<'a> Deserialize<'a>>(&self, keys: &[&str]) -> Result<Vec<Option<T>>> {
        let mut conn = self.get_connection().await?;
        
        let values: Vec<Option<String>> = conn.get(keys).await
            .map_err(|e| anyhow!("Failed to get multiple keys: {:?}", e))?;

        let mut result = Vec::new();
        for value in values {
            match value {
                Some(serialized) => {
                    let deserialized = serde_json::from_str(&serialized)?;
                    result.push(Some(deserialized));
                }
                None => result.push(None),
            }
        }

        Ok(result)
    }

    /// Flush all data from current database
    pub async fn flush_db(&self) -> Result<()> {
        let mut conn = self.get_connection().await?;
        
        redis::cmd("FLUSHDB").query_async(&mut conn).await
            .map_err(|e| anyhow!("Failed to flush database: {}", e))?;

        info!("Flushed Redis database");
        Ok(())
    }

    /// Get database info
    pub async fn info(&self) -> Result<String> {
        let mut conn = self.get_connection().await?;
        
        let info: String = redis::cmd("INFO").query_async(&mut conn).await
            .map_err(|e| anyhow!("Failed to get Redis info: {}", e))?;

        Ok(info)
    }

    /// Execute raw Redis command
    pub async fn execute_command(&self, cmd: &str, args: &[&str]) -> Result<redis::Value> {
        let mut conn = self.get_connection().await?;
        
        let mut redis_cmd = redis::cmd(cmd);
        for arg in args {
            redis_cmd.arg(*arg);
        }

        let result = redis_cmd.query_async(&mut conn).await
            .map_err(|e| anyhow!("Failed to execute Redis command '{}': {}", cmd, e))?;

        Ok(result)
    }

    /// Test Redis performance
    pub async fn benchmark(&self, operations: usize) -> Result<RedisBenchmarkResult> {
        let start_time = std::time::Instant::now();
        
        // Set operations
        let set_start = std::time::Instant::now();
        for i in 0..operations {
            self.set(&format!("benchmark_key_{}", i), &format!("value_{}", i), None).await?;
        }
        let set_duration = set_start.elapsed();

        // Get operations
        let get_start = std::time::Instant::now();
        for i in 0..operations {
            self.get::<String>(&format!("benchmark_key_{}", i)).await?;
        }
        let get_duration = get_start.elapsed();

        // Cleanup
        let mut conn = self.get_connection().await?;
        for i in 0..operations {
            let _: () = conn.del(format!("benchmark_key_{}", i)).await?;
        }

        let total_duration = start_time.elapsed();

        Ok(RedisBenchmarkResult {
            operations,
            total_duration,
            set_duration,
            get_duration,
            set_ops_per_sec: operations as f64 / set_duration.as_secs_f64(),
            get_ops_per_sec: operations as f64 / get_duration.as_secs_f64(),
        })
    }
}

#[derive(Debug)]
pub struct RedisBenchmarkResult {
    pub operations: usize,
    pub total_duration: Duration,
    pub set_duration: Duration,
    pub get_duration: Duration,
    pub set_ops_per_sec: f64,
    pub get_ops_per_sec: f64,
}

impl RedisBenchmarkResult {
    pub fn print_summary(&self) {
        info!("Redis Benchmark Results:");
        info!("  Operations: {}", self.operations);
        info!("  Total Time: {:?}", self.total_duration);
        info!("  SET: {:?} ({:.2} ops/sec)", self.set_duration, self.set_ops_per_sec);
        info!("  GET: {:?} ({:.2} ops/sec)", self.get_duration, self.get_ops_per_sec);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_redis_operations() -> Result<()> {
        let redis = RedisHelper::new("redis://localhost:6380").await?;
        
        // Test basic operations
        redis.set("test_key", &"test_value", None).await?;
        let value: Option<String> = redis.get("test_key").await?;
        assert_eq!(value, Some("test_value".to_string()));
        
        // Test deletion
        let deleted = redis.delete("test_key").await?;
        assert!(deleted);
        
        let value: Option<String> = redis.get("test_key").await?;
        assert!(value.is_none());

        Ok(())
    }
}