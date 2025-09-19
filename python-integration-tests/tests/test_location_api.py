"""Tests for Location API endpoints."""
import pytest
import time
import uuid
from typing import Dict, Any, List

from config import get_test_config
from utils.helpers import (
    generate_test_location_data,
    generate_batch_location_data,
    assert_status_code,
    assert_location_data_valid,
    assert_valid_uuid,
)

config = get_test_config()


class TestLocationAPI:
    """Test Location API endpoints."""
    
    def test_update_driver_location_success(self, http_client, created_driver):
        """Test successful location update."""
        driver_id = created_driver['id']
        location_data = generate_test_location_data()
        
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations",
            json=location_data
        )
        
        assert_status_code(response, 200)
        location_response = response.json()
        
        # Validate response structure
        assert_location_data_valid(location_response)
        
        # Validate data matches request
        assert location_response['driver_id'] == driver_id
        assert location_response['latitude'] == location_data['latitude']
        assert location_response['longitude'] == location_data['longitude']
        if 'accuracy' in location_data:
            assert location_response['accuracy'] == location_data['accuracy']
        if 'speed' in location_data:
            assert location_response['speed'] == location_data['speed']
        if 'bearing' in location_data:
            assert location_response['bearing'] == location_data['bearing']
    
    def test_update_location_validation_errors(self, http_client, created_driver):
        """Test location update with validation errors."""
        driver_id = created_driver['id']
        
        test_cases = [
            {
                "data": {"longitude": 37.6176},  # Missing latitude
                "description": "missing latitude"
            },
            {
                "data": {"latitude": 55.7558},  # Missing longitude
                "description": "missing longitude"
            },
            {
                "data": {"latitude": 91, "longitude": 37.6176},  # Invalid latitude
                "description": "invalid latitude > 90"
            },
            {
                "data": {"latitude": -91, "longitude": 37.6176},  # Invalid latitude
                "description": "invalid latitude < -90"
            },
            {
                "data": {"latitude": 55.7558, "longitude": 181},  # Invalid longitude
                "description": "invalid longitude > 180"
            },
            {
                "data": {"latitude": 55.7558, "longitude": -181},  # Invalid longitude
                "description": "invalid longitude < -180"
            },
            {
                "data": {"latitude": "invalid", "longitude": 37.6176},  # Invalid type
                "description": "invalid latitude type"
            }
        ]
        
        for case in test_cases:
            response = http_client.post(
                f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations",
                json=case["data"]
            )
            
            assert response.status_code == 400, \
                f"Failed for {case['description']}: expected 400, got {response.status_code}"
    
    def test_update_location_driver_not_found(self, http_client):
        """Test location update for non-existent driver."""
        non_existent_id = str(uuid.uuid4())
        location_data = generate_test_location_data()
        
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{non_existent_id}/locations",
            json=location_data
        )
        
        assert_status_code(response, 404)
        error_data = response.json()
        assert error_data['code'] == 'DRIVER_NOT_FOUND'
    
    def test_batch_update_locations_success(self, http_client, created_driver):
        """Test successful batch location update."""
        driver_id = created_driver['id']
        locations = generate_batch_location_data(count=3)
        batch_data = {"locations": locations}
        
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/batch",
            json=batch_data
        )
        
        assert_status_code(response, 200)
        response_data = response.json()
        
        assert response_data['count'] == 3
        assert "successfully" in response_data['message'].lower()
    
    def test_batch_update_locations_validation(self, http_client, created_driver):
        """Test batch location update validation."""
        driver_id = created_driver['id']
        
        # Test empty locations array
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/batch",
            json={"locations": []}
        )
        assert_status_code(response, 400)
        
        # Test missing locations field
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/batch",
            json={}
        )
        assert_status_code(response, 400)
        
        # Test invalid location in batch
        invalid_batch = {
            "locations": [
                generate_test_location_data(),
                {"latitude": 91, "longitude": 37.6176},  # Invalid
                generate_test_location_data()
            ]
        }
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/batch",
            json=invalid_batch
        )
        assert_status_code(response, 400)
    
    def test_get_current_location_success(self, http_client, driver_with_location):
        """Test getting current driver location."""
        driver_id = driver_with_location['id']
        
        response = http_client.get(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/current"
        )
        
        assert_status_code(response, 200)
        location_data = response.json()
        
        assert_location_data_valid(location_data)
        assert location_data['driver_id'] == driver_id
    
    def test_get_current_location_not_found(self, http_client, created_driver):
        """Test getting current location for driver without location data."""
        driver_id = created_driver['id']
        
        response = http_client.get(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/current"
        )
        
        assert_status_code(response, 404)
        error_data = response.json()
        assert error_data['code'] == 'LOCATION_NOT_FOUND'
    
    def test_get_location_history_success(self, http_client, driver_with_location):
        """Test getting location history."""
        driver_id = driver_with_location['id']
        
        # Add a few more location updates
        for i in range(3):
            location_data = generate_test_location_data()
            http_client.post(
                f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations",
                json=location_data
            )
            time.sleep(0.1)  # Small delay to ensure different timestamps
        
        response = http_client.get(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/history"
        )
        
        assert_status_code(response, 200)
        history_data = response.json()
        
        assert 'locations' in history_data
        assert 'count' in history_data
        assert isinstance(history_data['locations'], list)
        assert history_data['count'] >= 1
        
        # Validate each location in history
        for location in history_data['locations']:
            assert_location_data_valid(location)
            assert location['driver_id'] == driver_id
    
    def test_get_location_history_with_time_filters(self, http_client, driver_with_location):
        """Test location history with time filters."""
        driver_id = driver_with_location['id']
        
        # Get history for last hour
        from_time = int(time.time()) - 3600  # 1 hour ago
        to_time = int(time.time())
        
        response = http_client.get(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/history?"
            f"from={from_time}&to={to_time}"
        )
        
        assert_status_code(response, 200)
        history_data = response.json()
        
        assert 'locations' in history_data
        assert 'count' in history_data
        
        # All locations should be within the time range
        for location in history_data['locations']:
            location_time = time.strptime(
                location['recorded_at'].replace('Z', ''),
                '%Y-%m-%dT%H:%M:%S'
            )
            location_timestamp = int(time.mktime(location_time))
            assert from_time <= location_timestamp <= to_time
    
    def test_get_location_history_empty(self, http_client, created_driver):
        """Test getting location history for driver without locations."""
        driver_id = created_driver['id']
        
        response = http_client.get(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations/history"
        )
        
        assert_status_code(response, 200)
        history_data = response.json()
        
        assert history_data['count'] == 0
        assert len(history_data['locations']) == 0
    
    def test_get_nearby_drivers_success(self, http_client, multiple_drivers):
        """Test getting nearby drivers."""
        # Add locations to multiple drivers
        moscow_center = {"latitude": 55.7558, "longitude": 37.6176}
        
        for i, driver in enumerate(multiple_drivers[:3]):
            # Add locations around Moscow center
            location_data = generate_test_location_data(
                latitude=moscow_center["latitude"] + (i * 0.01),
                longitude=moscow_center["longitude"] + (i * 0.01)
            )
            http_client.post(
                f"{config.http_base_url}/api/v1/drivers/{driver['id']}/locations",
                json=location_data
            )
        
        # Search for nearby drivers
        response = http_client.get(
            f"{config.http_base_url}/api/v1/locations/nearby?"
            f"latitude={moscow_center['latitude']}&longitude={moscow_center['longitude']}"
            f"&radius_km=5&limit=10"
        )
        
        assert_status_code(response, 200)
        nearby_data = response.json()
        
        assert 'drivers' in nearby_data
        assert 'count' in nearby_data
        assert isinstance(nearby_data['drivers'], list)
        
        # Validate nearby driver data
        for driver_info in nearby_data['drivers']:
            assert_valid_uuid(driver_info['driver_id'])
            assert 'latitude' in driver_info
            assert 'longitude' in driver_info
            assert 'updated_at' in driver_info
            
            # Distance should be calculated
            if 'distance_km' in driver_info:
                assert isinstance(driver_info['distance_km'], (int, float))
                assert driver_info['distance_km'] >= 0
    
    def test_get_nearby_drivers_validation(self, http_client):
        """Test nearby drivers endpoint validation."""
        # Missing latitude
        response = http_client.get(
            f"{config.http_base_url}/api/v1/locations/nearby?longitude=37.6176"
        )
        assert_status_code(response, 400)
        
        # Missing longitude  
        response = http_client.get(
            f"{config.http_base_url}/api/v1/locations/nearby?latitude=55.7558"
        )
        assert_status_code(response, 400)
        
        # Invalid latitude
        response = http_client.get(
            f"{config.http_base_url}/api/v1/locations/nearby?latitude=91&longitude=37.6176"
        )
        assert_status_code(response, 400)
        
        # Invalid longitude
        response = http_client.get(
            f"{config.http_base_url}/api/v1/locations/nearby?latitude=55.7558&longitude=181"
        )
        assert_status_code(response, 400)
    
    def test_get_nearby_drivers_with_radius_and_limit(self, http_client, multiple_drivers):
        """Test nearby drivers with radius and limit parameters."""
        # Add locations to drivers
        center = {"latitude": 55.7558, "longitude": 37.6176}
        
        for i, driver in enumerate(multiple_drivers):
            location_data = generate_test_location_data(
                latitude=center["latitude"] + (i * 0.005),  # Small increments
                longitude=center["longitude"] + (i * 0.005)
            )
            http_client.post(
                f"{config.http_base_url}/api/v1/drivers/{driver['id']}/locations",
                json=location_data
            )
        
        # Test with small radius (should find fewer drivers)
        response = http_client.get(
            f"{config.http_base_url}/api/v1/locations/nearby?"
            f"latitude={center['latitude']}&longitude={center['longitude']}"
            f"&radius_km=0.5&limit=2"
        )
        
        assert_status_code(response, 200)
        nearby_data = response.json()
        
        assert len(nearby_data['drivers']) <= 2  # Respects limit
        assert nearby_data['count'] == len(nearby_data['drivers'])
    
    def test_location_with_optional_fields(self, http_client, created_driver):
        """Test location update with all optional fields."""
        driver_id = created_driver['id']
        location_data = {
            "latitude": 55.7558,
            "longitude": 37.6176,
            "altitude": 156.5,
            "accuracy": 5.2,
            "speed": 45.8,
            "bearing": 180.0,
            "timestamp": int(time.time())
        }
        
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations",
            json=location_data
        )
        
        assert_status_code(response, 200)
        location_response = response.json()
        
        # Validate all fields are preserved
        assert location_response['altitude'] == location_data['altitude']
        assert location_response['accuracy'] == location_data['accuracy']
        assert location_response['speed'] == location_data['speed']
        assert location_response['bearing'] == location_data['bearing']
    
    def test_location_timestamp_handling(self, http_client, created_driver):
        """Test location timestamp handling."""
        driver_id = created_driver['id']
        
        # Test with explicit timestamp
        custom_timestamp = int(time.time()) - 300  # 5 minutes ago
        location_data = generate_test_location_data()
        location_data['timestamp'] = custom_timestamp
        
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/locations",
            json=location_data
        )
        
        assert_status_code(response, 200)
        location_response = response.json()
        
        # Recorded time should reflect the provided timestamp
        recorded_time = time.strptime(
            location_response['recorded_at'].replace('Z', ''),
            '%Y-%m-%dT%H:%M:%S'
        )
        recorded_timestamp = int(time.mktime(recorded_time))
        
        # Should be close to the provided timestamp (within a few seconds for processing)
        assert abs(recorded_timestamp - custom_timestamp) < 10
    
    def test_location_api_invalid_driver_id(self, http_client):
        """Test location endpoints with invalid driver ID."""
        invalid_id = "not-a-uuid"
        location_data = generate_test_location_data()
        
        # Test update location
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{invalid_id}/locations",
            json=location_data
        )
        assert_status_code(response, 400)
        
        # Test batch update
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers/{invalid_id}/locations/batch",
            json={"locations": [location_data]}
        )
        assert_status_code(response, 400)
        
        # Test get current location
        response = http_client.get(
            f"{config.http_base_url}/api/v1/drivers/{invalid_id}/locations/current"
        )
        assert_status_code(response, 400)
        
        # Test get history
        response = http_client.get(
            f"{config.http_base_url}/api/v1/drivers/{invalid_id}/locations/history"
        )
        assert_status_code(response, 400)