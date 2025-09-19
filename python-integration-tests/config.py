"""Configuration for integration tests."""
import os
from typing import Optional, Dict, Any
from pydantic import BaseSettings, Field, validator
from dataclasses import dataclass


class TestConfig(BaseSettings):
    """Test configuration settings."""
    
    # Service endpoints
    service_host: str = Field(default="localhost", env="SERVICE_HOST")
    service_http_port: int = Field(default=8001, env="SERVICE_HTTP_PORT")
    service_grpc_port: int = Field(default=9001, env="SERVICE_GRPC_PORT")
    service_metrics_port: int = Field(default=9002, env="SERVICE_METRICS_PORT")
    
    # Database settings
    db_host: str = Field(default="localhost", env="TEST_DB_HOST")
    db_port: int = Field(default=5433, env="TEST_DB_PORT")
    db_user: str = Field(default="test_user", env="TEST_DB_USER")
    db_password: str = Field(default="test_password", env="TEST_DB_PASSWORD")
    db_name: str = Field(default="driver_service_test", env="TEST_DB_NAME")
    
    # Redis settings
    redis_host: str = Field(default="localhost", env="REDIS_HOST")
    redis_port: int = Field(default=6380, env="REDIS_PORT")
    redis_db: int = Field(default=0, env="REDIS_DB")
    redis_password: Optional[str] = Field(default=None, env="REDIS_PASSWORD")
    
    # NATS settings
    nats_url: str = Field(default="nats://localhost:4222", env="NATS_URL")
    nats_cluster_id: str = Field(default="driver-service-cluster", env="NATS_CLUSTER_ID")
    nats_client_id: str = Field(default="python-test-client", env="NATS_CLIENT_ID")
    
    # Test settings
    test_timeout: int = Field(default=30, env="TEST_TIMEOUT")
    test_parallel_workers: int = Field(default=4, env="TEST_WORKERS")
    cleanup_after_test: bool = Field(default=True, env="CLEANUP_AFTER_TEST")
    
    # Docker settings
    use_docker: bool = Field(default=True, env="USE_DOCKER")
    docker_compose_file: str = Field(default="docker-compose.test.yml", env="DOCKER_COMPOSE_FILE")
    
    # Logging settings
    log_level: str = Field(default="INFO", env="LOG_LEVEL")
    log_format: str = Field(default="json", env="LOG_FORMAT")
    
    class Config:
        env_file = ".env"
        case_sensitive = False
    
    @property
    def http_base_url(self) -> str:
        """Get base HTTP URL for the service."""
        return f"http://{self.service_host}:{self.service_http_port}"
    
    @property
    def grpc_address(self) -> str:
        """Get gRPC address for the service."""
        return f"{self.service_host}:{self.service_grpc_port}"
    
    @property
    def database_url(self) -> str:
        """Get database connection URL."""
        return f"postgresql://{self.db_user}:{self.db_password}@{self.db_host}:{self.db_port}/{self.db_name}"
    
    @property
    def redis_url(self) -> str:
        """Get Redis connection URL."""
        auth_part = f":{self.redis_password}@" if self.redis_password else ""
        return f"redis://{auth_part}{self.redis_host}:{self.redis_port}/{self.redis_db}"


@dataclass
class APIEndpoints:
    """API endpoints configuration."""
    
    # Health check
    health: str = "/health"
    
    # Driver endpoints
    drivers: str = "/api/v1/drivers"
    driver_by_id: str = "/api/v1/drivers/{driver_id}"
    driver_status: str = "/api/v1/drivers/{driver_id}/status"
    active_drivers: str = "/api/v1/drivers/active"
    
    # Location endpoints
    update_location: str = "/api/v1/drivers/{driver_id}/locations"
    batch_update_locations: str = "/api/v1/drivers/{driver_id}/locations/batch"
    current_location: str = "/api/v1/drivers/{driver_id}/locations/current"
    location_history: str = "/api/v1/drivers/{driver_id}/locations/history"
    nearby_drivers: str = "/api/v1/locations/nearby"
    
    # WebSocket endpoints
    ws_tracking: str = "/ws/tracking/{driver_id}"
    ws_orders: str = "/ws/orders/{driver_id}"


@dataclass
class NATSSubjects:
    """NATS subjects configuration."""
    
    # Outgoing events
    driver_registered: str = "driver.registered"
    driver_verified: str = "driver.verified"
    driver_status_changed: str = "driver.status.changed"
    driver_availability_changed: str = "driver.availability.changed"
    driver_shift_started: str = "driver.shift.started"
    driver_shift_ended: str = "driver.shift.ended"
    driver_location_updated: str = "driver.location.updated"
    driver_rating_updated: str = "driver.rating.updated"
    driver_performance_alert: str = "driver.performance.alert"
    
    # Incoming events
    order_assigned: str = "order.assigned"
    order_completed: str = "order.completed"
    order_cancelled: str = "order.cancelled"
    payment_processed: str = "payment.processed"
    payment_failed: str = "payment.failed"
    vehicle_assigned: str = "vehicle.assigned"
    vehicle_maintenance_required: str = "vehicle.maintenance.required"
    customer_rated_driver: str = "customer.rated.driver"


# Global configuration instance
config = TestConfig()
endpoints = APIEndpoints()
nats_subjects = NATSSubjects()


def get_test_config() -> TestConfig:
    """Get test configuration."""
    return config


def get_endpoints() -> APIEndpoints:
    """Get API endpoints configuration."""
    return endpoints


def get_nats_subjects() -> NATSSubjects:
    """Get NATS subjects configuration."""
    return nats_subjects