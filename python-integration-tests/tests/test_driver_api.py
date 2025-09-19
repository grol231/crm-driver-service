"""Tests for Driver API endpoints."""
import pytest
import uuid
from typing import Dict, Any

from config import get_test_config
from utils.helpers import (
    generate_test_driver_data,
    assert_status_code,
    assert_driver_data_valid,
    assert_valid_uuid,
)

config = get_test_config()


class TestDriverAPI:
    """Test Driver API endpoints."""
    
    @pytest.mark.smoke
    def test_health_check(self, http_client):
        """Test health check endpoint."""
        response = http_client.get(f"{config.http_base_url}/health")
        assert_status_code(response, 200)
        
        data = response.json()
        assert data["status"] == "healthy"
        assert data["service"] == "driver-service"
        assert "timestamp" in data
    
    def test_create_driver_success(self, http_client, test_drivers):
        """Test successful driver creation."""
        driver_data = generate_test_driver_data()
        
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers",
            json=driver_data
        )
        
        assert_status_code(response, 201)
        created_driver = response.json()
        test_drivers.append(created_driver['id'])
        
        # Validate response structure
        assert_driver_data_valid(created_driver)
        
        # Validate data matches request
        assert created_driver['phone'] == driver_data['phone']
        assert created_driver['email'] == driver_data['email']
        assert created_driver['first_name'] == driver_data['first_name']
        assert created_driver['last_name'] == driver_data['last_name']
        assert created_driver['status'] == 'registered'
        assert created_driver['total_trips'] == 0
        assert created_driver['current_rating'] == 0.0
    
    def test_create_driver_validation_errors(self, http_client):
        """Test driver creation with validation errors."""
        test_cases = [
            # Missing required fields
            {
                "data": {"email": "test@example.com"},
                "expected_status": 400,
                "description": "missing phone"
            },
            {
                "data": {"phone": "+79001234567"},
                "expected_status": 400,
                "description": "missing email"
            },
            # Invalid data formats
            {
                "data": {
                    "phone": "+79001234567",
                    "email": "invalid-email",
                    "first_name": "Test",
                    "last_name": "User"
                },
                "expected_status": 400,
                "description": "invalid email format"
            },
            {
                "data": {
                    "phone": "invalid-phone",
                    "email": "test@example.com",
                    "first_name": "Test",
                    "last_name": "User"
                },
                "expected_status": 400,
                "description": "invalid phone format"
            }
        ]
        
        for case in test_cases:
            response = http_client.post(
                f"{config.http_base_url}/api/v1/drivers",
                json=case["data"]
            )
            
            assert response.status_code == case["expected_status"], \
                f"Failed for {case['description']}: expected {case['expected_status']}, got {response.status_code}"
    
    def test_create_duplicate_driver(self, http_client, created_driver):
        """Test creation of duplicate driver (same phone)."""
        # Try to create driver with same phone
        duplicate_data = generate_test_driver_data(phone=created_driver['phone'])
        
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers",
            json=duplicate_data
        )
        
        assert_status_code(response, 409)
        error_data = response.json()
        assert error_data['code'] == 'DRIVER_EXISTS'
    
    def test_get_driver_success(self, http_client, created_driver):
        """Test successful driver retrieval."""
        driver_id = created_driver['id']
        
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers/{driver_id}")
        assert_status_code(response, 200)
        
        driver_data = response.json()
        assert_driver_data_valid(driver_data)
        assert driver_data['id'] == driver_id
        assert driver_data['phone'] == created_driver['phone']
    
    def test_get_driver_not_found(self, http_client):
        """Test getting non-existent driver."""
        non_existent_id = str(uuid.uuid4())
        
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers/{non_existent_id}")
        assert_status_code(response, 404)
        
        error_data = response.json()
        assert error_data['code'] == 'DRIVER_NOT_FOUND'
    
    def test_get_driver_invalid_id(self, http_client):
        """Test getting driver with invalid ID format."""
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers/invalid-uuid")
        assert_status_code(response, 400)
        
        error_data = response.json()
        assert "Invalid driver ID format" in error_data['error']
    
    def test_update_driver_success(self, http_client, created_driver):
        """Test successful driver update."""
        driver_id = created_driver['id']
        update_data = {
            "first_name": "Updated Name",
            "email": "updated@example.com"
        }
        
        response = http_client.put(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}",
            json=update_data
        )
        
        assert_status_code(response, 200)
        updated_driver = response.json()
        
        assert_driver_data_valid(updated_driver)
        assert updated_driver['first_name'] == update_data['first_name']
        assert updated_driver['email'] == update_data['email']
        # Other fields should remain unchanged
        assert updated_driver['phone'] == created_driver['phone']
        assert updated_driver['last_name'] == created_driver['last_name']
    
    def test_update_driver_not_found(self, http_client):
        """Test updating non-existent driver."""
        non_existent_id = str(uuid.uuid4())
        update_data = {"first_name": "Updated"}
        
        response = http_client.put(
            f"{config.http_base_url}/api/v1/drivers/{non_existent_id}",
            json=update_data
        )
        
        assert_status_code(response, 404)
        error_data = response.json()
        assert error_data['code'] == 'DRIVER_NOT_FOUND'
    
    def test_delete_driver_success(self, http_client, test_drivers):
        """Test successful driver deletion."""
        from utils.helpers import create_test_driver
        
        driver = create_test_driver(http_client)
        driver_id = driver['id']
        test_drivers.append(driver_id)
        
        # Delete the driver
        response = http_client.delete(f"{config.http_base_url}/api/v1/drivers/{driver_id}")
        assert_status_code(response, 204)
        
        # Verify driver is deleted
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers/{driver_id}")
        assert_status_code(response, 404)
    
    def test_change_driver_status(self, http_client, created_driver):
        """Test changing driver status."""
        driver_id = created_driver['id']
        status_data = {"status": "pending_verification"}
        
        response = http_client.patch(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/status",
            json=status_data
        )
        
        assert_status_code(response, 200)
        response_data = response.json()
        assert response_data['status'] == 'pending_verification'
        
        # Verify status was updated
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers/{driver_id}")
        driver_data = response.json()
        assert driver_data['status'] == 'pending_verification'
    
    def test_list_drivers(self, http_client, multiple_drivers):
        """Test listing drivers with pagination."""
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers?limit=3&offset=0")
        assert_status_code(response, 200)
        
        data = response.json()
        assert 'drivers' in data
        assert 'total' in data
        assert 'limit' in data
        assert 'offset' in data
        assert 'has_more' in data
        
        assert len(data['drivers']) <= 3
        assert data['limit'] == 3
        assert data['offset'] == 0
        
        # Validate each driver in the list
        for driver in data['drivers']:
            assert_driver_data_valid(driver)
    
    def test_list_drivers_with_filters(self, http_client, multiple_drivers):
        """Test listing drivers with filters."""
        # Test status filter
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers?status=registered")
        assert_status_code(response, 200)
        
        data = response.json()
        for driver in data['drivers']:
            assert driver['status'] == 'registered'
        
        # Test rating filter
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers?min_rating=0&max_rating=1")
        assert_status_code(response, 200)
        
        data = response.json()
        for driver in data['drivers']:
            assert 0 <= driver['current_rating'] <= 1
    
    def test_get_active_drivers(self, http_client, multiple_drivers):
        """Test getting active drivers."""
        # First, change some drivers to active statuses
        active_statuses = ['available', 'on_shift', 'busy']
        for i, driver in enumerate(multiple_drivers[:3]):
            if i < len(active_statuses):
                status_data = {"status": active_statuses[i]}
                http_client.patch(
                    f"{config.http_base_url}/api/v1/drivers/{driver['id']}/status",
                    json=status_data
                )
        
        # Get active drivers
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers/active")
        assert_status_code(response, 200)
        
        data = response.json()
        assert 'drivers' in data
        assert 'count' in data
        
        active_count = data['count']
        assert active_count >= 3  # At least the ones we just set
        
        # Validate that all returned drivers are active
        for driver in data['drivers']:
            assert driver['status'] in active_statuses
    
    def test_list_drivers_pagination(self, http_client, multiple_drivers):
        """Test pagination functionality."""
        # Get first page
        response1 = http_client.get(f"{config.http_base_url}/api/v1/drivers?limit=2&offset=0")
        assert_status_code(response1, 200)
        page1 = response1.json()
        
        # Get second page
        response2 = http_client.get(f"{config.http_base_url}/api/v1/drivers?limit=2&offset=2")
        assert_status_code(response2, 200)
        page2 = response2.json()
        
        # Verify pagination
        assert len(page1['drivers']) <= 2
        assert len(page2['drivers']) <= 2
        assert page1['limit'] == 2
        assert page2['limit'] == 2
        assert page1['offset'] == 0
        assert page2['offset'] == 2
        
        # Verify no duplicates between pages
        page1_ids = {driver['id'] for driver in page1['drivers']}
        page2_ids = {driver['id'] for driver in page2['drivers']}
        assert len(page1_ids.intersection(page2_ids)) == 0
    
    def test_driver_api_error_handling(self, http_client):
        """Test API error handling for various scenarios."""
        # Test malformed JSON
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers",
            data="invalid json",
            headers={"Content-Type": "application/json"}
        )
        assert response.status_code == 400
        
        # Test empty request body
        response = http_client.post(
            f"{config.http_base_url}/api/v1/drivers",
            json={}
        )
        assert response.status_code == 400
        
        # Test invalid UUID in URL
        response = http_client.get(f"{config.http_base_url}/api/v1/drivers/not-a-uuid")
        assert response.status_code == 400
    
    @pytest.mark.parametrize("invalid_status", [
        "invalid_status",
        "",
        "REGISTERED",  # Case sensitivity
        123,  # Wrong type
    ])
    def test_invalid_status_change(self, http_client, created_driver, invalid_status):
        """Test changing driver to invalid status."""
        driver_id = created_driver['id']
        status_data = {"status": invalid_status}
        
        response = http_client.patch(
            f"{config.http_base_url}/api/v1/drivers/{driver_id}/status",
            json=status_data
        )
        
        # Should return 400 for invalid status
        assert response.status_code == 400