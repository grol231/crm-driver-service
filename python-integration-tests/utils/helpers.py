"""Helper functions for integration tests."""
import time
import uuid
import requests
import psycopg2
from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List
from faker import Faker
import re

from config import get_test_config
from .logger import get_logger

fake = Faker(['ru_RU'])
fake.add_provider(Faker.providers.automotive.ru_RU.Provider)
config = get_test_config()
logger = get_logger(__name__)


def wait_for_service(url: str, timeout: int = 30, interval: float = 1.0) -> bool:
    """Wait for service to be available."""
    start_time = time.time()
    
    while time.time() - start_time < timeout:
        try:
            response = requests.get(f"{url}/health", timeout=5)
            if response.status_code == 200:
                logger.info(f"Service is available", url=url)
                return True
        except requests.RequestException as e:
            logger.debug(f"Service not yet available", url=url, error=str(e))
        
        time.sleep(interval)
    
    logger.error(f"Service did not become available within timeout", url=url, timeout=timeout)
    return False


def generate_test_driver_data(phone: Optional[str] = None) -> Dict[str, Any]:
    """Generate test driver data."""
    if phone is None:
        phone = f"+7{fake.random_int(min=9000000000, max=9999999999)}"
    
    # Generate Russian passport data
    passport_series = f"{fake.random_int(min=1000, max=9999)}"
    passport_number = f"{fake.random_int(min=100000, max=999999)}"
    
    # Generate license number in Russian format
    license_number = f"{fake.random_element(['77', '97', '99', '177'])}{fake.random_int(min=100000, max=999999)}"
    
    birth_date = fake.date_of_birth(minimum_age=21, maximum_age=65)
    license_expiry = fake.date_between(start_date='today', end_date='+10y')
    
    return {
        "phone": phone,
        "email": fake.email(),
        "first_name": fake.first_name_male(),
        "last_name": fake.last_name_male(), 
        "middle_name": fake.middle_name_male() if fake.boolean() else None,
        "birth_date": birth_date.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "passport_series": passport_series,
        "passport_number": passport_number,
        "license_number": license_number,
        "license_expiry": license_expiry.strftime("%Y-%m-%dT%H:%M:%SZ"),
    }


def generate_test_location_data(
    latitude: Optional[float] = None,
    longitude: Optional[float] = None,
    **kwargs
) -> Dict[str, Any]:
    """Generate test location data."""
    # Moscow region coordinates by default
    if latitude is None:
        latitude = fake.latitude()
        # Ensure it's in Moscow region roughly
        latitude = 55.7558 + (float(latitude) % 1.0 - 0.5) * 0.5
        
    if longitude is None:
        longitude = fake.longitude()
        # Ensure it's in Moscow region roughly
        longitude = 37.6176 + (float(longitude) % 1.0 - 0.5) * 0.5
    
    data = {
        "latitude": round(latitude, 6),
        "longitude": round(longitude, 6),
        "accuracy": fake.random_int(min=1, max=10),
        "speed": fake.random_int(min=0, max=80),
        "bearing": fake.random_int(min=0, max=359),
        "timestamp": int(time.time()),
    }
    
    # Add optional fields
    if "altitude" in kwargs:
        data["altitude"] = kwargs["altitude"]
    
    return data


def generate_batch_location_data(count: int = 5, **kwargs) -> List[Dict[str, Any]]:
    """Generate batch of location data."""
    locations = []
    base_time = int(time.time()) - count * 10  # 10 seconds apart
    
    for i in range(count):
        location = generate_test_location_data(**kwargs)
        location["timestamp"] = base_time + i * 10
        locations.append(location)
    
    return locations


def cleanup_test_data(driver_ids: List[str] = None) -> None:
    """Cleanup test data from database."""
    try:
        conn = psycopg2.connect(
            host=config.db_host,
            port=config.db_port,
            user=config.db_user,
            password=config.db_password,
            database=config.db_name,
        )
        
        with conn.cursor() as cur:
            if driver_ids:
                # Clean specific drivers
                placeholders = ','.join(['%s'] * len(driver_ids))
                
                # Clean in order of foreign key dependencies
                tables = [
                    'driver_ratings',
                    'driver_locations',
                    'driver_shifts',
                    'driver_documents',
                    'drivers'
                ]
                
                for table in tables:
                    cur.execute(f"DELETE FROM {table} WHERE driver_id IN ({placeholders})", driver_ids)
            else:
                # Clean all test data
                cur.execute("DELETE FROM driver_ratings WHERE driver_id IN (SELECT id FROM drivers WHERE email LIKE '%test%' OR email LIKE '%example%')")
                cur.execute("DELETE FROM driver_locations WHERE driver_id IN (SELECT id FROM drivers WHERE email LIKE '%test%' OR email LIKE '%example%')")
                cur.execute("DELETE FROM driver_shifts WHERE driver_id IN (SELECT id FROM drivers WHERE email LIKE '%test%' OR email LIKE '%example%')")
                cur.execute("DELETE FROM driver_documents WHERE driver_id IN (SELECT id FROM drivers WHERE email LIKE '%test%' OR email LIKE '%example%')")
                cur.execute("DELETE FROM drivers WHERE email LIKE '%test%' OR email LIKE '%example%'")
        
        conn.commit()
        conn.close()
        logger.info("Test data cleaned up successfully")
        
    except Exception as e:
        logger.error(f"Failed to cleanup test data: {e}")


def assert_valid_uuid(value: str) -> None:
    """Assert that value is a valid UUID."""
    try:
        uuid.UUID(value)
    except ValueError:
        raise AssertionError(f"Invalid UUID: {value}")


def assert_valid_timestamp(timestamp_str: str) -> None:
    """Assert that timestamp string is valid ISO format."""
    try:
        datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
    except ValueError:
        raise AssertionError(f"Invalid timestamp format: {timestamp_str}")


def assert_status_code(response: requests.Response, expected_code: int) -> None:
    """Assert HTTP response status code with detailed error info."""
    if response.status_code != expected_code:
        try:
            error_details = response.json()
        except:
            error_details = response.text
        
        raise AssertionError(
            f"Expected status code {expected_code}, got {response.status_code}. "
            f"Response: {error_details}"
        )


def assert_driver_data_valid(driver_data: Dict[str, Any]) -> None:
    """Assert that driver data contains required fields with valid values."""
    required_fields = [
        'id', 'phone', 'email', 'first_name', 'last_name',
        'birth_date', 'passport_series', 'passport_number',
        'license_number', 'license_expiry', 'status',
        'current_rating', 'total_trips', 'created_at', 'updated_at'
    ]
    
    for field in required_fields:
        assert field in driver_data, f"Missing required field: {field}"
    
    # Validate specific fields
    assert_valid_uuid(driver_data['id'])
    assert_valid_timestamp(driver_data['created_at'])
    assert_valid_timestamp(driver_data['updated_at'])
    
    # Validate phone format (Russian mobile)
    phone_pattern = r'^\+7\d{10}$'
    assert re.match(phone_pattern, driver_data['phone']), f"Invalid phone format: {driver_data['phone']}"
    
    # Validate email
    email_pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    assert re.match(email_pattern, driver_data['email']), f"Invalid email format: {driver_data['email']}"
    
    # Validate rating
    assert 0.0 <= driver_data['current_rating'] <= 5.0, f"Invalid rating: {driver_data['current_rating']}"
    
    # Validate trip count
    assert driver_data['total_trips'] >= 0, f"Invalid trip count: {driver_data['total_trips']}"


def assert_location_data_valid(location_data: Dict[str, Any]) -> None:
    """Assert that location data contains required fields with valid values."""
    required_fields = ['id', 'driver_id', 'latitude', 'longitude', 'recorded_at', 'created_at']
    
    for field in required_fields:
        assert field in location_data, f"Missing required field: {field}"
    
    # Validate UUIDs
    assert_valid_uuid(location_data['id'])
    assert_valid_uuid(location_data['driver_id'])
    
    # Validate timestamps
    assert_valid_timestamp(location_data['recorded_at'])
    assert_valid_timestamp(location_data['created_at'])
    
    # Validate coordinates
    assert -90 <= location_data['latitude'] <= 90, f"Invalid latitude: {location_data['latitude']}"
    assert -180 <= location_data['longitude'] <= 180, f"Invalid longitude: {location_data['longitude']}"
    
    # Validate optional fields if present
    if 'accuracy' in location_data and location_data['accuracy'] is not None:
        assert location_data['accuracy'] >= 0, f"Invalid accuracy: {location_data['accuracy']}"
    
    if 'speed' in location_data and location_data['speed'] is not None:
        assert location_data['speed'] >= 0, f"Invalid speed: {location_data['speed']}"
    
    if 'bearing' in location_data and location_data['bearing'] is not None:
        assert 0 <= location_data['bearing'] < 360, f"Invalid bearing: {location_data['bearing']}"


def create_test_driver(http_client) -> Dict[str, Any]:
    """Create a test driver and return the response data."""
    driver_data = generate_test_driver_data()
    
    response = http_client.post(
        f"{config.http_base_url}/api/v1/drivers",
        json=driver_data
    )
    
    assert_status_code(response, 201)
    created_driver = response.json()
    assert_driver_data_valid(created_driver)
    
    return created_driver


def update_driver_location(http_client, driver_id: str, location_data: Dict[str, Any] = None) -> Dict[str, Any]:
    """Update driver location and return the response data."""
    if location_data is None:
        location_data = generate_test_location_data()
    
    response = http_client.post(
        f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations",
        json=location_data
    )
    
    assert_status_code(response, 200)
    location_response = response.json()
    assert_location_data_valid(location_response)
    
    return location_response


def wait_for_postgres(timeout: int = 30) -> bool:
    """Wait for PostgreSQL to be ready."""
    start_time = time.time()
    
    while time.time() - start_time < timeout:
        try:
            conn = psycopg2.connect(
                host=config.db_host,
                port=config.db_port,
                user=config.db_user,
                password=config.db_password,
                database=config.db_name,
                connect_timeout=5,
            )
            conn.close()
            logger.info("PostgreSQL is ready")
            return True
        except psycopg2.OperationalError as e:
            logger.debug(f"PostgreSQL not ready yet: {e}")
            time.sleep(1)
    
    logger.error("PostgreSQL did not become ready within timeout")
    return False