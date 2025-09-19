use anyhow::{anyhow, Result};
use chrono::{DateTime, Utc};
use sqlx::{PgPool, Row};
use std::collections::HashMap;
use tracing::{debug, error, info};
use uuid::Uuid;

use crate::config::TestConfig;
use crate::fixtures::*;

#[derive(Debug, Clone)]
pub struct DatabaseHelper {
    pool: PgPool,
}

impl DatabaseHelper {
    pub async fn new(config: &TestConfig) -> Result<Self> {
        let pool = PgPool::connect(&config.database_url()).await
            .map_err(|e| anyhow!("Failed to connect to database: {}", e))?;

        info!("Connected to database: {}", config.database.host);

        Ok(Self { pool })
    }

    /// Ensure database migrations are applied
    pub async fn ensure_migrations(&self) -> Result<()> {
        // Check if tables exist
        let table_exists = sqlx::query(
            "SELECT EXISTS (
                SELECT FROM information_schema.tables 
                WHERE table_schema = 'public' 
                AND table_name = 'drivers'
            )"
        )
        .fetch_one(&self.pool)
        .await?
        .get::<bool, _>(0);

        if !table_exists {
            error!("Database tables do not exist. Please run migrations first.");
            return Err(anyhow!("Database tables not found"));
        }

        info!("Database tables verified");
        Ok(())
    }

    /// Clean up all test data
    pub async fn cleanup_test_data(&self) -> Result<()> {
        let mut tx = self.pool.begin().await?;

        // Delete in order to respect foreign key constraints
        sqlx::query("DELETE FROM driver_ratings").execute(&mut *tx).await?;
        sqlx::query("DELETE FROM driver_shifts").execute(&mut *tx).await?;
        sqlx::query("DELETE FROM driver_locations").execute(&mut *tx).await?;
        sqlx::query("DELETE FROM driver_documents").execute(&mut *tx).await?;
        sqlx::query("DELETE FROM driver_rating_stats").execute(&mut *tx).await?;
        sqlx::query("DELETE FROM drivers").execute(&mut *tx).await?;

        tx.commit().await?;
        
        debug!("Cleaned up all test data from database");
        Ok(())
    }

    /// Create a test driver directly in the database
    pub async fn create_test_driver(&self, driver: &TestDriver) -> Result<Uuid> {
        let driver_id = sqlx::query_scalar!(
            r#"
            INSERT INTO drivers (
                id, phone, email, first_name, last_name, middle_name,
                birth_date, passport_series, passport_number, license_number,
                license_expiry, status, current_rating, total_trips,
                created_at, updated_at
            ) VALUES (
                $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
            ) RETURNING id
            "#,
            driver.id,
            driver.phone,
            driver.email,
            driver.first_name,
            driver.last_name,
            driver.middle_name,
            driver.birth_date.date_naive(),
            driver.passport_series,
            driver.passport_number,
            driver.license_number,
            driver.license_expiry.date_naive(),
            driver.status,
            driver.current_rating as f64,
            driver.total_trips,
            Utc::now(),
            Utc::now()
        )
        .fetch_one(&self.pool)
        .await?;

        debug!("Created test driver in database: {}", driver_id);
        Ok(driver_id)
    }

    /// Get driver by ID
    pub async fn get_driver(&self, driver_id: Uuid) -> Result<Option<DatabaseDriver>> {
        let row = sqlx::query!(
            r#"
            SELECT id, phone, email, first_name, last_name, middle_name,
                   birth_date, passport_series, passport_number, license_number,
                   license_expiry, status, current_rating, total_trips,
                   created_at, updated_at, deleted_at
            FROM drivers 
            WHERE id = $1 AND deleted_at IS NULL
            "#,
            driver_id
        )
        .fetch_optional(&self.pool)
        .await?;

        if let Some(row) = row {
            Ok(Some(DatabaseDriver {
                id: row.id,
                phone: row.phone,
                email: row.email,
                first_name: row.first_name,
                last_name: row.last_name,
                middle_name: row.middle_name,
                birth_date: DateTime::from_naive_utc_and_offset(row.birth_date.and_hms_opt(0, 0, 0).unwrap(), Utc),
                passport_series: row.passport_series,
                passport_number: row.passport_number,
                license_number: row.license_number,
                license_expiry: DateTime::from_naive_utc_and_offset(row.license_expiry.and_hms_opt(0, 0, 0).unwrap(), Utc),
                status: row.status,
                current_rating: row.current_rating,
                total_trips: row.total_trips,
                created_at: row.created_at,
                updated_at: row.updated_at,
                deleted_at: row.deleted_at,
            }))
        } else {
            Ok(None)
        }
    }

    /// Create test locations for a driver
    pub async fn create_test_locations(&self, locations: &[TestLocation]) -> Result<()> {
        let mut tx = self.pool.begin().await?;

        for location in locations {
            sqlx::query!(
                r#"
                INSERT INTO driver_locations (
                    id, driver_id, latitude, longitude, altitude, accuracy,
                    speed, bearing, recorded_at, created_at
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                "#,
                location.id,
                location.driver_id,
                location.latitude,
                location.longitude,
                location.altitude,
                location.accuracy,
                location.speed,
                location.bearing,
                location.recorded_at,
                Utc::now()
            )
            .execute(&mut *tx)
            .await?;
        }

        tx.commit().await?;
        debug!("Created {} test locations in database", locations.len());
        Ok(())
    }

    /// Get driver locations
    pub async fn get_driver_locations(&self, driver_id: Uuid, limit: Option<i32>) -> Result<Vec<DatabaseLocation>> {
        let locations = sqlx::query!(
            r#"
            SELECT id, driver_id, latitude, longitude, altitude, accuracy,
                   speed, bearing, address, recorded_at, created_at
            FROM driver_locations
            WHERE driver_id = $1
            ORDER BY recorded_at DESC
            LIMIT $2
            "#,
            driver_id,
            limit.unwrap_or(100) as i64
        )
        .fetch_all(&self.pool)
        .await?;

        Ok(locations
            .into_iter()
            .map(|row| DatabaseLocation {
                id: row.id,
                driver_id: row.driver_id,
                latitude: row.latitude,
                longitude: row.longitude,
                altitude: row.altitude,
                accuracy: row.accuracy,
                speed: row.speed,
                bearing: row.bearing,
                address: row.address,
                recorded_at: row.recorded_at,
                created_at: row.created_at,
            })
            .collect())
    }

    /// Create test shift
    pub async fn create_test_shift(&self, shift: &TestShift) -> Result<Uuid> {
        let shift_id = sqlx::query_scalar!(
            r#"
            INSERT INTO driver_shifts (
                id, driver_id, vehicle_id, start_time, end_time, status,
                total_trips, total_distance, total_earnings, created_at, updated_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            RETURNING id
            "#,
            shift.id,
            shift.driver_id,
            shift.vehicle_id,
            shift.start_time,
            shift.end_time,
            shift.status,
            shift.total_trips,
            shift.total_distance,
            shift.total_earnings,
            Utc::now(),
            Utc::now()
        )
        .fetch_one(&self.pool)
        .await?;

        debug!("Created test shift in database: {}", shift_id);
        Ok(shift_id)
    }

    /// Create test rating
    pub async fn create_test_rating(&self, rating: &TestRating) -> Result<Uuid> {
        let rating_id = sqlx::query_scalar!(
            r#"
            INSERT INTO driver_ratings (
                id, driver_id, order_id, customer_id, rating, comment,
                rating_type, created_at, updated_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            RETURNING id
            "#,
            rating.id,
            rating.driver_id,
            rating.order_id,
            rating.customer_id,
            rating.rating,
            rating.comment,
            rating.rating_type,
            Utc::now(),
            Utc::now()
        )
        .fetch_one(&self.pool)
        .await?;

        debug!("Created test rating in database: {}", rating_id);
        Ok(rating_id)
    }

    /// Get driver statistics
    pub async fn get_driver_stats(&self, driver_id: Uuid) -> Result<Option<DriverStats>> {
        let row = sqlx::query!(
            r#"
            SELECT 
                d.id,
                d.current_rating,
                d.total_trips,
                COUNT(dl.id) as total_locations,
                COUNT(ds.id) as total_shifts,
                COUNT(dr.id) as total_ratings,
                AVG(dr.rating) as avg_rating
            FROM drivers d
            LEFT JOIN driver_locations dl ON d.id = dl.driver_id
            LEFT JOIN driver_shifts ds ON d.id = ds.driver_id
            LEFT JOIN driver_ratings dr ON d.id = dr.driver_id
            WHERE d.id = $1 AND d.deleted_at IS NULL
            GROUP BY d.id, d.current_rating, d.total_trips
            "#,
            driver_id
        )
        .fetch_optional(&self.pool)
        .await?;

        if let Some(row) = row {
            Ok(Some(DriverStats {
                driver_id: row.id,
                current_rating: row.current_rating,
                total_trips: row.total_trips,
                total_locations: row.total_locations.unwrap_or(0),
                total_shifts: row.total_shifts.unwrap_or(0),
                total_ratings: row.total_ratings.unwrap_or(0),
                average_rating: row.avg_rating,
            }))
        } else {
            Ok(None)
        }
    }

    /// Verify data consistency
    pub async fn verify_data_consistency(&self) -> Result<ConsistencyReport> {
        let mut report = ConsistencyReport::new();

        // Check drivers with missing rating stats
        let drivers_without_stats = sqlx::query!(
            r#"
            SELECT d.id 
            FROM drivers d
            LEFT JOIN driver_rating_stats rs ON d.id = rs.driver_id
            WHERE d.deleted_at IS NULL AND rs.driver_id IS NULL
            "#
        )
        .fetch_all(&self.pool)
        .await?;

        report.add_check("drivers_without_rating_stats", drivers_without_stats.len() == 0);

        // Check rating consistency
        let rating_inconsistencies = sqlx::query!(
            r#"
            SELECT d.id 
            FROM drivers d
            JOIN driver_rating_stats rs ON d.id = rs.driver_id
            WHERE ABS(d.current_rating - rs.average_rating) > 0.01
            "#
        )
        .fetch_all(&self.pool)
        .await?;

        report.add_check("rating_consistency", rating_inconsistencies.len() == 0);

        // Check location data integrity
        let invalid_locations = sqlx::query!(
            r#"
            SELECT COUNT(*) as count
            FROM driver_locations
            WHERE latitude NOT BETWEEN -90 AND 90
               OR longitude NOT BETWEEN -180 AND 180
            "#
        )
        .fetch_one(&self.pool)
        .await?;

        report.add_check("valid_location_coordinates", invalid_locations.count.unwrap_or(0) == 0);

        Ok(report)
    }

    /// Get pool for direct access
    pub fn get_pool(&self) -> &PgPool {
        &self.pool
    }
}

#[derive(Debug, Clone)]
pub struct DatabaseDriver {
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
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
    pub deleted_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
pub struct DatabaseLocation {
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

#[derive(Debug)]
pub struct DriverStats {
    pub driver_id: Uuid,
    pub current_rating: f64,
    pub total_trips: i32,
    pub total_locations: i64,
    pub total_shifts: i64,
    pub total_ratings: i64,
    pub average_rating: Option<f64>,
}

#[derive(Debug)]
pub struct ConsistencyReport {
    pub checks: HashMap<String, bool>,
    pub total_checks: usize,
    pub passed_checks: usize,
}

impl ConsistencyReport {
    pub fn new() -> Self {
        Self {
            checks: HashMap::new(),
            total_checks: 0,
            passed_checks: 0,
        }
    }

    pub fn add_check(&mut self, name: &str, passed: bool) {
        self.checks.insert(name.to_string(), passed);
        self.total_checks += 1;
        if passed {
            self.passed_checks += 1;
        }
    }

    pub fn is_all_passed(&self) -> bool {
        self.passed_checks == self.total_checks
    }

    pub fn get_failed_checks(&self) -> Vec<String> {
        self.checks
            .iter()
            .filter_map(|(name, passed)| if !passed { Some(name.clone()) } else { None })
            .collect()
    }
}