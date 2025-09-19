use anyhow::{anyhow, Result};
use reqwest::{Client, Response, StatusCode};
use serde::de::DeserializeOwned;
use serde::Serialize;
use std::time::Duration;
use tracing::{debug, error, info, warn};
use uuid::Uuid;

use crate::config::TestConfig;
use crate::fixtures::*;

#[derive(Debug, Clone)]
pub struct ApiClient {
    client: Client,
    base_url: String,
}

impl ApiClient {
    pub fn new(config: &TestConfig) -> Result<Self> {
        let client = Client::builder()
            .timeout(Duration::from_secs(config.driver_service.timeout_seconds))
            .build()?;

        Ok(Self {
            client,
            base_url: config.service_api_url(),
        })
    }

    /// Health check endpoint
    pub async fn health_check(&self) -> Result<serde_json::Value> {
        let url = format!("{}/health", self.base_url.trim_end_matches("/api/v1"));
        debug!("Health check request: {}", url);

        let response = self.client.get(&url).send().await?;
        self.handle_response(response).await
    }

    /// Create a new driver
    pub async fn create_driver(&self, request: &CreateDriverRequest) -> Result<DriverResponse> {
        let url = format!("{}/drivers", self.base_url);
        debug!("Create driver request: {} -> {:?}", url, request);

        let response = self.client
            .post(&url)
            .json(request)
            .send()
            .await?;

        self.handle_response(response).await
    }

    /// Get driver by ID
    pub async fn get_driver(&self, driver_id: Uuid) -> Result<DriverResponse> {
        let url = format!("{}/drivers/{}", self.base_url, driver_id);
        debug!("Get driver request: {}", url);

        let response = self.client.get(&url).send().await?;
        self.handle_response(response).await
    }

    /// Update driver
    pub async fn update_driver(&self, driver_id: Uuid, request: &UpdateDriverRequest) -> Result<DriverResponse> {
        let url = format!("{}/drivers/{}", self.base_url, driver_id);
        debug!("Update driver request: {} -> {:?}", url, request);

        let response = self.client
            .put(&url)
            .json(request)
            .send()
            .await?;

        self.handle_response(response).await
    }

    /// Delete driver
    pub async fn delete_driver(&self, driver_id: Uuid) -> Result<()> {
        let url = format!("{}/drivers/{}", self.base_url, driver_id);
        debug!("Delete driver request: {}", url);

        let response = self.client.delete(&url).send().await?;
        
        match response.status() {
            StatusCode::NO_CONTENT => {
                info!("Driver {} deleted successfully", driver_id);
                Ok(())
            }
            status => {
                let error_text = response.text().await.unwrap_or_default();
                error!("Delete driver failed: {} - {}", status, error_text);
                Err(anyhow!("Delete driver failed: {}", status))
            }
        }
    }

    /// List drivers with optional filters
    pub async fn list_drivers(
        &self,
        status: Option<&str>,
        min_rating: Option<f64>,
        limit: Option<i32>,
        offset: Option<i32>,
    ) -> Result<ListDriversResponse> {
        let mut url = format!("{}/drivers", self.base_url);
        let mut query_params = vec![];

        if let Some(status) = status {
            query_params.push(format!("status={}", status));
        }
        if let Some(min_rating) = min_rating {
            query_params.push(format!("min_rating={}", min_rating));
        }
        if let Some(limit) = limit {
            query_params.push(format!("limit={}", limit));
        }
        if let Some(offset) = offset {
            query_params.push(format!("offset={}", offset));
        }

        if !query_params.is_empty() {
            url.push('?');
            url.push_str(&query_params.join("&"));
        }

        debug!("List drivers request: {}", url);

        let response = self.client.get(&url).send().await?;
        self.handle_response(response).await
    }

    /// Change driver status
    pub async fn change_driver_status(&self, driver_id: Uuid, status: &str) -> Result<serde_json::Value> {
        let url = format!("{}/drivers/{}/status", self.base_url, driver_id);
        let request = serde_json::json!({ "status": status });
        
        debug!("Change driver status request: {} -> {:?}", url, request);

        let response = self.client
            .patch(&url)
            .json(&request)
            .send()
            .await?;

        self.handle_response(response).await
    }

    /// Get active drivers
    pub async fn get_active_drivers(&self) -> Result<serde_json::Value> {
        let url = format!("{}/drivers/active", self.base_url);
        debug!("Get active drivers request: {}", url);

        let response = self.client.get(&url).send().await?;
        self.handle_response(response).await
    }

    /// Update driver location
    pub async fn update_location(&self, driver_id: Uuid, location: &UpdateLocationRequest) -> Result<LocationResponse> {
        let url = format!("{}/drivers/{}/location", self.base_url, driver_id);
        debug!("Update location request: {} -> {:?}", url, location);

        let response = self.client
            .post(&url)
            .json(location)
            .send()
            .await?;

        self.handle_response(response).await
    }

    /// Batch update locations
    pub async fn batch_update_locations(&self, driver_id: Uuid, locations: &[UpdateLocationRequest]) -> Result<serde_json::Value> {
        let url = format!("{}/drivers/{}/locations/batch", self.base_url, driver_id);
        let request = serde_json::json!({ "locations": locations });
        
        debug!("Batch update locations request: {} -> {} locations", url, locations.len());

        let response = self.client
            .post(&url)
            .json(&request)
            .send()
            .await?;

        self.handle_response(response).await
    }

    /// Get current location
    pub async fn get_current_location(&self, driver_id: Uuid) -> Result<LocationResponse> {
        let url = format!("{}/drivers/{}/location", self.base_url, driver_id);
        debug!("Get current location request: {}", url);

        let response = self.client.get(&url).send().await?;
        self.handle_response(response).await
    }

    /// Get location history
    pub async fn get_location_history(
        &self,
        driver_id: Uuid,
        from: Option<i64>,
        to: Option<i64>,
    ) -> Result<serde_json::Value> {
        let mut url = format!("{}/drivers/{}/location/history", self.base_url, driver_id);
        let mut query_params = vec![];

        if let Some(from) = from {
            query_params.push(format!("from={}", from));
        }
        if let Some(to) = to {
            query_params.push(format!("to={}", to));
        }

        if !query_params.is_empty() {
            url.push('?');
            url.push_str(&query_params.join("&"));
        }

        debug!("Get location history request: {}", url);

        let response = self.client.get(&url).send().await?;
        self.handle_response(response).await
    }

    /// Get nearby drivers
    pub async fn get_nearby_drivers(
        &self,
        latitude: f64,
        longitude: f64,
        radius_km: Option<f64>,
        limit: Option<i32>,
    ) -> Result<serde_json::Value> {
        let mut url = format!("{}/drivers/nearby", self.base_url);
        let mut query_params = vec![
            format!("latitude={}", latitude),
            format!("longitude={}", longitude),
        ];

        if let Some(radius_km) = radius_km {
            query_params.push(format!("radius_km={}", radius_km));
        }
        if let Some(limit) = limit {
            query_params.push(format!("limit={}", limit));
        }

        url.push('?');
        url.push_str(&query_params.join("&"));

        debug!("Get nearby drivers request: {}", url);

        let response = self.client.get(&url).send().await?;
        self.handle_response(response).await
    }

    /// Handle HTTP response and deserialize JSON
    async fn handle_response<T: DeserializeOwned>(&self, response: Response) -> Result<T> {
        let status = response.status();
        let response_text = response.text().await?;

        debug!("API response: {} - {}", status, response_text);

        match status {
            StatusCode::OK | StatusCode::CREATED => {
                serde_json::from_str(&response_text)
                    .map_err(|e| anyhow!("Failed to deserialize response: {}", e))
            }
            StatusCode::BAD_REQUEST => {
                let error: ErrorResponse = serde_json::from_str(&response_text)
                    .unwrap_or_else(|_| ErrorResponse {
                        error: "Bad Request".to_string(),
                        code: Some("BAD_REQUEST".to_string()),
                        details: Some(response_text),
                    });
                Err(anyhow!("Bad Request: {} - {:?}", error.error, error.details))
            }
            StatusCode::NOT_FOUND => {
                let error: ErrorResponse = serde_json::from_str(&response_text)
                    .unwrap_or_else(|_| ErrorResponse {
                        error: "Not Found".to_string(),
                        code: Some("NOT_FOUND".to_string()),
                        details: Some(response_text),
                    });
                Err(anyhow!("Not Found: {} - {:?}", error.error, error.details))
            }
            StatusCode::CONFLICT => {
                let error: ErrorResponse = serde_json::from_str(&response_text)
                    .unwrap_or_else(|_| ErrorResponse {
                        error: "Conflict".to_string(),
                        code: Some("CONFLICT".to_string()),
                        details: Some(response_text),
                    });
                Err(anyhow!("Conflict: {} - {:?}", error.error, error.details))
            }
            StatusCode::INTERNAL_SERVER_ERROR => {
                error!("Internal server error: {}", response_text);
                Err(anyhow!("Internal Server Error: {}", response_text))
            }
            _ => {
                warn!("Unexpected status code: {} - {}", status, response_text);
                Err(anyhow!("Unexpected response: {} - {}", status, response_text))
            }
        }
    }

    /// Wait for service to be ready (health check with retries)
    pub async fn wait_for_service(&self, max_retries: u32, delay: Duration) -> Result<()> {
        for attempt in 1..=max_retries {
            match self.health_check().await {
                Ok(_) => {
                    info!("Service is ready after {} attempts", attempt);
                    return Ok(());
                }
                Err(e) => {
                    warn!("Health check attempt {}/{} failed: {}", attempt, max_retries, e);
                    if attempt == max_retries {
                        return Err(anyhow!("Service not ready after {} attempts", max_retries));
                    }
                    tokio::time::sleep(delay).await;
                }
            }
        }
        Ok(())
    }
}

// Test-specific extensions
impl ApiClient {
    /// Create test driver with automatic cleanup tracking
    pub async fn create_test_driver(&self, driver: &TestDriver) -> Result<DriverResponse> {
        let request = create_driver_request(driver);
        self.create_driver(&request).await
    }

    /// Bulk create drivers for load testing
    pub async fn bulk_create_drivers(&self, drivers: Vec<TestDriver>) -> Result<Vec<Result<DriverResponse>>> {
        let mut tasks = Vec::new();
        
        for driver in drivers {
            let client = self.clone();
            let task = tokio::spawn(async move {
                client.create_test_driver(&driver).await
            });
            tasks.push(task);
        }

        let results = futures::future::join_all(tasks).await;
        let responses = results
            .into_iter()
            .map(|task_result| task_result.unwrap_or_else(|e| Err(anyhow!("Task failed: {}", e))))
            .collect();

        Ok(responses)
    }

    /// Test API endpoint availability
    pub async fn test_endpoint_availability(&self) -> Result<EndpointTestReport> {
        let mut report = EndpointTestReport::new();

        // Test health endpoint
        report.add_test("health", self.health_check().await.is_ok());

        // Test drivers list endpoint
        report.add_test("list_drivers", self.list_drivers(None, None, Some(1), Some(0)).await.is_ok());

        // Test active drivers endpoint
        report.add_test("active_drivers", self.get_active_drivers().await.is_ok());

        // Test nearby drivers endpoint
        report.add_test("nearby_drivers", 
            self.get_nearby_drivers(55.7558, 37.6176, Some(5.0), Some(10)).await.is_ok());

        Ok(report)
    }
}

#[derive(Debug)]
pub struct EndpointTestReport {
    pub tests: Vec<(String, bool)>,
    pub total_tests: usize,
    pub passed_tests: usize,
}

impl EndpointTestReport {
    pub fn new() -> Self {
        Self {
            tests: Vec::new(),
            total_tests: 0,
            passed_tests: 0,
        }
    }

    pub fn add_test(&mut self, name: &str, passed: bool) {
        self.tests.push((name.to_string(), passed));
        self.total_tests += 1;
        if passed {
            self.passed_tests += 1;
        }
    }

    pub fn success_rate(&self) -> f64 {
        if self.total_tests == 0 {
            0.0
        } else {
            self.passed_tests as f64 / self.total_tests as f64
        }
    }

    pub fn is_all_passed(&self) -> bool {
        self.passed_tests == self.total_tests
    }
}