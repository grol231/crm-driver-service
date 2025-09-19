"""Locust performance tests for Driver Service."""
import json
import random
import time
from locust import HttpUser, task, between
from faker import Faker

fake = Faker(['ru_RU'])


class DriverServiceUser(HttpUser):
    """Simulate user interactions with Driver Service."""
    
    wait_time = between(1, 3)
    
    def on_start(self):
        """Setup for each user."""
        self.driver_ids = []
        self.api_base = "/api/v1"
        
        # Create a few test drivers for this user
        for _ in range(3):
            driver_id = self.create_test_driver()
            if driver_id:
                self.driver_ids.append(driver_id)
    
    def on_stop(self):
        """Cleanup for each user."""
        # Clean up created drivers
        for driver_id in self.driver_ids:
            self.client.delete(f"{self.api_base}/drivers/{driver_id}")
    
    def create_test_driver(self):
        """Create a test driver and return its ID."""
        driver_data = {
            "phone": f"+7{fake.random_int(min=9000000000, max=9999999999)}",
            "email": fake.email(),
            "first_name": fake.first_name_male(),
            "last_name": fake.last_name_male(),
            "birth_date": fake.date_of_birth(minimum_age=21, maximum_age=65).strftime("%Y-%m-%dT%H:%M:%SZ"),
            "passport_series": f"{fake.random_int(min=1000, max=9999)}",
            "passport_number": f"{fake.random_int(min=100000, max=999999)}",
            "license_number": f"{fake.random_element(['77', '97', '99'])}{fake.random_int(min=100000, max=999999)}",
            "license_expiry": fake.date_between(start_date='today', end_date='+5y').strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        
        with self.client.post(f"{self.api_base}/drivers", json=driver_data, catch_response=True) as response:
            if response.status_code == 201:
                return response.json()['id']
            else:
                response.failure(f"Failed to create driver: {response.status_code}")
                return None
    
    @task(10)
    def health_check(self):
        """Health check endpoint - most frequent."""
        self.client.get("/health")
    
    @task(8)
    def list_drivers(self):
        """List drivers with pagination."""
        limit = random.randint(10, 50)
        offset = random.randint(0, 100)
        self.client.get(f"{self.api_base}/drivers?limit={limit}&offset={offset}")
    
    @task(6)
    def get_driver(self):
        """Get specific driver."""
        if self.driver_ids:
            driver_id = random.choice(self.driver_ids)
            self.client.get(f"{self.api_base}/drivers/{driver_id}")
    
    @task(5)
    def update_driver_location(self):
        """Update driver location."""
        if self.driver_ids:
            driver_id = random.choice(self.driver_ids)
            location_data = {
                "latitude": 55.7558 + (random.random() - 0.5) * 0.1,
                "longitude": 37.6176 + (random.random() - 0.5) * 0.1,
                "accuracy": random.randint(1, 10),
                "speed": random.randint(0, 80),
                "bearing": random.randint(0, 359),
                "timestamp": int(time.time()),
            }
            self.client.post(f"{self.api_base}/drivers/{driver_id}/locations", json=location_data)
    
    @task(4)
    def get_current_location(self):
        """Get driver current location."""
        if self.driver_ids:
            driver_id = random.choice(self.driver_ids)
            self.client.get(f"{self.api_base}/drivers/{driver_id}/locations/current")
    
    @task(3)
    def change_driver_status(self):
        """Change driver status."""
        if self.driver_ids:
            driver_id = random.choice(self.driver_ids)
            statuses = ["available", "on_shift", "busy", "inactive"]
            status = random.choice(statuses)
            self.client.patch(f"{self.api_base}/drivers/{driver_id}/status", json={"status": status})
    
    @task(3)
    def get_nearby_drivers(self):
        """Get nearby drivers."""
        lat = 55.7558 + (random.random() - 0.5) * 0.1
        lon = 37.6176 + (random.random() - 0.5) * 0.1
        radius = random.randint(1, 10)
        limit = random.randint(5, 20)
        
        self.client.get(
            f"{self.api_base}/locations/nearby"
            f"?latitude={lat}&longitude={lon}&radius_km={radius}&limit={limit}"
        )
    
    @task(2)
    def get_location_history(self):
        """Get driver location history."""
        if self.driver_ids:
            driver_id = random.choice(self.driver_ids)
            from_time = int(time.time()) - 3600  # 1 hour ago
            to_time = int(time.time())
            
            self.client.get(
                f"{self.api_base}/drivers/{driver_id}/locations/history"
                f"?from={from_time}&to={to_time}"
            )
    
    @task(2)
    def update_driver_info(self):
        """Update driver information."""
        if self.driver_ids:
            driver_id = random.choice(self.driver_ids)
            update_data = {
                "first_name": fake.first_name_male(),
                "email": fake.email()
            }
            self.client.put(f"{self.api_base}/drivers/{driver_id}", json=update_data)
    
    @task(1)
    def batch_update_locations(self):
        """Batch update locations."""
        if self.driver_ids:
            driver_id = random.choice(self.driver_ids)
            
            locations = []
            base_time = int(time.time()) - 300  # 5 minutes ago
            
            for i in range(5):
                location = {
                    "latitude": 55.7558 + (random.random() - 0.5) * 0.01,
                    "longitude": 37.6176 + (random.random() - 0.5) * 0.01,
                    "accuracy": random.randint(1, 10),
                    "speed": random.randint(0, 80),
                    "bearing": random.randint(0, 359),
                    "timestamp": base_time + i * 60,
                }
                locations.append(location)
            
            self.client.post(
                f"{self.api_base}/drivers/{driver_id}/locations/batch",
                json={"locations": locations}
            )
    
    @task(1)
    def get_active_drivers(self):
        """Get active drivers."""
        self.client.get(f"{self.api_base}/drivers/active")
    
    @task(1)
    def filtered_drivers_search(self):
        """Search drivers with filters."""
        filters = {
            "min_rating": random.choice([3.0, 3.5, 4.0, 4.5]),
            "status": random.choice(["available", "on_shift"]),
            "limit": random.randint(10, 30)
        }
        
        query_params = "&".join([f"{k}={v}" for k, v in filters.items()])
        self.client.get(f"{self.api_base}/drivers?{query_params}")


class HighLoadUser(DriverServiceUser):
    """High-load user for stress testing."""
    
    wait_time = between(0.1, 0.5)  # Much faster requests
    
    @task(20)
    def rapid_location_updates(self):
        """Rapid location updates to stress test the system."""
        if self.driver_ids:
            driver_id = random.choice(self.driver_ids)
            location_data = {
                "latitude": 55.7558 + (random.random() - 0.5) * 0.001,
                "longitude": 37.6176 + (random.random() - 0.5) * 0.001,
                "timestamp": int(time.time()),
            }
            self.client.post(f"{self.api_base}/drivers/{driver_id}/locations", json=location_data)


class ReadOnlyUser(HttpUser):
    """Read-only user for testing read operations."""
    
    wait_time = between(0.5, 2)
    
    def on_start(self):
        self.api_base = "/api/v1"
    
    @task(15)
    def health_check(self):
        self.client.get("/health")
    
    @task(10)
    def list_drivers(self):
        limit = random.randint(10, 50)
        offset = random.randint(0, 200)
        self.client.get(f"{self.api_base}/drivers?limit={limit}&offset={offset}")
    
    @task(8)
    def get_nearby_drivers(self):
        lat = 55.7558 + (random.random() - 0.5) * 0.5
        lon = 37.6176 + (random.random() - 0.5) * 0.5
        radius = random.randint(1, 20)
        self.client.get(
            f"{self.api_base}/locations/nearby"
            f"?latitude={lat}&longitude={lon}&radius_km={radius}"
        )
    
    @task(5)
    def get_active_drivers(self):
        self.client.get(f"{self.api_base}/drivers/active")
    
    @task(3)
    def search_drivers(self):
        status = random.choice(["available", "on_shift", "busy"])
        min_rating = random.choice([3.0, 3.5, 4.0])
        self.client.get(f"{self.api_base}/drivers?status={status}&min_rating={min_rating}")


# Custom task sets for specific scenarios
class DriverRegistrationFlowUser(HttpUser):
    """User simulating driver registration and onboarding flow."""
    
    wait_time = between(2, 5)
    
    def on_start(self):
        self.api_base = "/api/v1"
        self.driver_id = None
    
    def on_stop(self):
        if self.driver_id:
            self.client.delete(f"{self.api_base}/drivers/{self.driver_id}")
    
    @task
    def registration_flow(self):
        """Complete driver registration flow."""
        # Step 1: Create driver
        driver_data = {
            "phone": f"+7{fake.random_int(min=9000000000, max=9999999999)}",
            "email": fake.email(),
            "first_name": fake.first_name_male(),
            "last_name": fake.last_name_male(),
            "birth_date": fake.date_of_birth(minimum_age=21, maximum_age=65).strftime("%Y-%m-%dT%H:%M:%SZ"),
            "passport_series": f"{fake.random_int(min=1000, max=9999)}",
            "passport_number": f"{fake.random_int(min=100000, max=999999)}",
            "license_number": f"77{fake.random_int(min=100000, max=999999)}",
            "license_expiry": fake.date_between(start_date='today', end_date='+5y').strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        
        response = self.client.post(f"{self.api_base}/drivers", json=driver_data)
        if response.status_code == 201:
            self.driver_id = response.json()['id']
            
            # Step 2: Change status to pending verification
            time.sleep(1)
            self.client.patch(
                f"{self.api_base}/drivers/{self.driver_id}/status",
                json={"status": "pending_verification"}
            )
            
            # Step 3: Verify driver (simulate approval)
            time.sleep(2)
            self.client.patch(
                f"{self.api_base}/drivers/{self.driver_id}/status",
                json={"status": "verified"}
            )
            
            # Step 4: Make driver available
            time.sleep(1)
            self.client.patch(
                f"{self.api_base}/drivers/{self.driver_id}/status",
                json={"status": "available"}
            )
            
            # Step 5: Update location (driver goes online)
            time.sleep(1)
            location_data = {
                "latitude": 55.7558 + (random.random() - 0.5) * 0.1,
                "longitude": 37.6176 + (random.random() - 0.5) * 0.1,
                "timestamp": int(time.time()),
            }
            self.client.post(f"{self.api_base}/drivers/{self.driver_id}/locations", json=location_data)