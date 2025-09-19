use chrono::{DateTime, Utc, Duration};
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use rand::prelude::*;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestDriver {
    pub id: Uuid,
    pub phone: String,
    pub email: String,
    pub first_name: String,
    pub last_name: String,
    pub middle_name: Option<String>,
    pub birth_date: DateTime<Utc>,
    pub passport_series: String,
    pub passport_number: String,
    pub license_number: String,
    pub license_expiry: DateTime<Utc>,
    pub status: String,
    pub current_rating: f64,
    pub total_trips: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestDocument {
    pub id: Uuid,
    pub driver_id: Uuid,
    pub document_type: String,
    pub document_number: String,
    pub issue_date: DateTime<Utc>,
    pub expiry_date: DateTime<Utc>,
    pub file_url: String,
    pub status: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestLocation {
    pub id: Uuid,
    pub driver_id: Uuid,
    pub latitude: f64,
    pub longitude: f64,
    pub altitude: Option<f64>,
    pub accuracy: Option<f64>,
    pub speed: Option<f64>,
    pub bearing: Option<f64>,
    pub recorded_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestShift {
    pub id: Uuid,
    pub driver_id: Uuid,
    pub vehicle_id: Option<Uuid>,
    pub start_time: DateTime<Utc>,
    pub end_time: Option<DateTime<Utc>>,
    pub status: String,
    pub total_trips: i32,
    pub total_distance: f64,
    pub total_earnings: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestRating {
    pub id: Uuid,
    pub driver_id: Uuid,
    pub order_id: Option<Uuid>,
    pub customer_id: Option<Uuid>,
    pub rating: i32,
    pub comment: Option<String>,
    pub rating_type: String,
}

// Driver API request/response structures
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateDriverRequest {
    pub phone: String,
    pub email: String,
    pub first_name: String,
    pub last_name: String,
    pub middle_name: Option<String>,
    pub birth_date: DateTime<Utc>,
    pub passport_series: String,
    pub passport_number: String,
    pub license_number: String,
    pub license_expiry: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DriverResponse {
    pub id: Uuid,
    pub phone: String,
    pub email: String,
    pub first_name: String,
    pub last_name: String,
    pub middle_name: Option<String>,
    pub birth_date: DateTime<Utc>,
    pub passport_series: String,
    pub passport_number: String,
    pub license_number: String,
    pub license_expiry: DateTime<Utc>,
    pub status: String,
    pub current_rating: f64,
    pub total_trips: i32,
    pub metadata: serde_json::Value,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateDriverRequest {
    pub email: Option<String>,
    pub first_name: Option<String>,
    pub last_name: Option<String>,
    pub middle_name: Option<String>,
    pub birth_date: Option<DateTime<Utc>>,
    pub passport_series: Option<String>,
    pub passport_number: Option<String>,
    pub license_expiry: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UpdateLocationRequest {
    pub latitude: f64,
    pub longitude: f64,
    pub altitude: Option<f64>,
    pub accuracy: Option<f64>,
    pub speed: Option<f64>,
    pub bearing: Option<f64>,
    pub timestamp: Option<i64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationResponse {
    pub id: Uuid,
    pub driver_id: Uuid,
    pub latitude: f64,
    pub longitude: f64,
    pub altitude: Option<f64>,
    pub accuracy: Option<f64>,
    pub speed: Option<f64>,
    pub bearing: Option<f64>,
    pub address: Option<String>,
    pub recorded_at: DateTime<Utc>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ListDriversResponse {
    pub drivers: Vec<DriverResponse>,
    pub total: i32,
    pub limit: i32,
    pub offset: i32,
    pub has_more: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorResponse {
    pub error: String,
    pub code: Option<String>,
    pub details: Option<String>,
}

// NATS event structures
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DriverRegisteredEvent {
    pub event_type: String,
    pub driver_id: String,
    pub phone: String,
    pub email: String,
    pub name: String,
    pub license_number: String,
    pub city: String,
    pub timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DriverLocationUpdatedEvent {
    pub event_type: String,
    pub driver_id: String,
    pub location: LocationData,
    pub speed: f64,
    pub bearing: f64,
    pub accuracy: f64,
    pub on_trip: bool,
    pub order_id: Option<String>,
    pub timestamp: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LocationData {
    pub latitude: f64,
    pub longitude: f64,
    pub address: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OrderAssignedEvent {
    pub event_type: String,
    pub order_id: String,
    pub driver_id: String,
    pub customer_id: String,
    pub pickup_location: LocationData,
    pub dropoff_location: LocationData,
    pub estimated_fare: f64,
    pub estimated_distance: f64,
    pub estimated_duration: i32,
    pub priority: i32,
    pub timestamp: DateTime<Utc>,
}

// Test data generators
pub fn generate_test_drivers(count: usize) -> Vec<TestDriver> {
    let mut rng = thread_rng();
    let mut drivers = Vec::with_capacity(count);

    for i in 0..count {
        drivers.push(TestDriver {
            id: Uuid::new_v4(),
            phone: format!("+7900123456{}", i),
            email: format!("test.driver.{}@example.com", i),
            first_name: format!("Тест{}", i),
            last_name: "Водителев".to_string(),
            middle_name: if rng.gen_bool(0.7) {
                Some("Тестович".to_string())
            } else {
                None
            },
            birth_date: Utc::now() - Duration::days(rng.gen_range(25..65) * 365),
            passport_series: format!("{:04}", rng.gen_range(1000..9999)),
            passport_number: format!("{:06}", rng.gen_range(100000..999999)),
            license_number: format!("TEST{:06}", rng.gen_range(100000..999999)),
            license_expiry: Utc::now() + Duration::days(rng.gen_range(365..1095)),
            status: "registered".to_string(),
            current_rating: rng.gen_range(3.0..5.0),
            total_trips: rng.gen_range(0..1000),
        });
    }

    drivers
}

pub fn generate_test_locations(driver_id: Uuid, count: usize) -> Vec<TestLocation> {
    let mut rng = thread_rng();
    let mut locations = Vec::with_capacity(count);

    // Moscow center coordinates
    let base_lat = 55.7558;
    let base_lng = 37.6176;

    for i in 0..count {
        locations.push(TestLocation {
            id: Uuid::new_v4(),
            driver_id,
            latitude: base_lat + rng.gen_range(-0.1..0.1),
            longitude: base_lng + rng.gen_range(-0.1..0.1),
            altitude: Some(rng.gen_range(100.0..200.0)),
            accuracy: Some(rng.gen_range(1.0..10.0)),
            speed: Some(rng.gen_range(0.0..80.0)),
            bearing: Some(rng.gen_range(0.0..360.0)),
            recorded_at: Utc::now() - Duration::minutes(count as i64 - i as i64),
        });
    }

    locations
}

pub fn create_driver_request(driver: &TestDriver) -> CreateDriverRequest {
    CreateDriverRequest {
        phone: driver.phone.clone(),
        email: driver.email.clone(),
        first_name: driver.first_name.clone(),
        last_name: driver.last_name.clone(),
        middle_name: driver.middle_name.clone(),
        birth_date: driver.birth_date,
        passport_series: driver.passport_series.clone(),
        passport_number: driver.passport_number.clone(),
        license_number: driver.license_number.clone(),
        license_expiry: driver.license_expiry,
    }
}

pub fn create_location_request(location: &TestLocation) -> UpdateLocationRequest {
    UpdateLocationRequest {
        latitude: location.latitude,
        longitude: location.longitude,
        altitude: location.altitude,
        accuracy: location.accuracy,
        speed: location.speed,
        bearing: location.bearing,
        timestamp: Some(location.recorded_at.timestamp()),
    }
}

// Test scenarios data
pub struct TestScenario {
    pub name: String,
    pub description: String,
    pub steps: Vec<TestStep>,
}

pub enum TestStep {
    CreateDriver(TestDriver),
    UpdateDriver(Uuid, UpdateDriverRequest),
    UpdateLocation(Uuid, UpdateLocationRequest),
    AssignOrder(OrderAssignedEvent),
    CompleteOrder(String),
    AddRating(TestRating),
}

pub fn generate_driver_lifecycle_scenario() -> TestScenario {
    let driver = generate_test_drivers(1).into_iter().next().unwrap();
    let driver_id = driver.id;

    TestScenario {
        name: "driver_lifecycle".to_string(),
        description: "Complete driver lifecycle from registration to rating".to_string(),
        steps: vec![
            TestStep::CreateDriver(driver),
            TestStep::UpdateLocation(driver_id, UpdateLocationRequest {
                latitude: 55.7558,
                longitude: 37.6176,
                altitude: Some(150.0),
                accuracy: Some(5.0),
                speed: Some(0.0),
                bearing: Some(0.0),
                timestamp: None,
            }),
        ],
    }
}

// Constants for testing
pub const VALID_STATUSES: &[&str] = &[
    "registered",
    "pending_verification",
    "verified",
    "rejected",
    "available",
    "on_shift",
    "busy",
    "inactive",
    "suspended",
    "blocked",
];

pub const MOSCOW_COORDINATES: (f64, f64) = (55.7558, 37.6176);
pub const SPB_COORDINATES: (f64, f64) = (59.9311, 30.3609);
pub const KAZAN_COORDINATES: (f64, f64) = (55.8304, 49.0661);