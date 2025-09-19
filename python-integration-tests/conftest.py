"""Pytest configuration and fixtures."""
import pytest
import requests
import time
from typing import Generator, Dict, Any, List

from config import get_test_config
from utils import setup_logger, wait_for_service, cleanup_test_data, wait_for_postgres

config = get_test_config()
logger = setup_logger()


@pytest.fixture(scope="session", autouse=True)
def setup_logging():
    """Setup logging for the test session."""
    setup_logger()


@pytest.fixture(scope="session")
def wait_for_services():
    """Wait for all required services to be available."""
    logger.info("Waiting for services to be available...")
    
    # Wait for PostgreSQL
    assert wait_for_postgres(timeout=30), "PostgreSQL is not available"
    
    # Wait for Driver Service HTTP API
    assert wait_for_service(config.http_base_url, timeout=60), "Driver Service HTTP API is not available"
    
    logger.info("All services are ready")


@pytest.fixture(scope="session")
def http_client(wait_for_services) -> Generator[requests.Session, None, None]:
    """HTTP client for API testing."""
    session = requests.Session()
    session.headers.update({
        "Content-Type": "application/json",
        "Accept": "application/json",
    })
    yield session
    session.close()


@pytest.fixture(scope="function")
def test_drivers() -> Generator[List[str], None, None]:
    """List to track created test drivers for cleanup."""
    driver_ids = []
    yield driver_ids
    
    # Cleanup after test
    if config.cleanup_after_test and driver_ids:
        cleanup_test_data(driver_ids)


@pytest.fixture(scope="function")
def created_driver(http_client, test_drivers) -> Dict[str, Any]:
    """Create a test driver for testing."""
    from utils.helpers import create_test_driver
    
    driver = create_test_driver(http_client)
    test_drivers.append(driver['id'])
    return driver


@pytest.fixture(scope="function")
def multiple_drivers(http_client, test_drivers) -> List[Dict[str, Any]]:
    """Create multiple test drivers."""
    from utils.helpers import create_test_driver
    
    drivers = []
    for i in range(5):
        driver = create_test_driver(http_client)
        drivers.append(driver)
        test_drivers.append(driver['id'])
    
    return drivers


@pytest.fixture
def driver_with_location(http_client, test_drivers) -> Dict[str, Any]:
    """Create a driver with location data."""
    from utils.helpers import create_test_driver, update_driver_location
    
    driver = create_test_driver(http_client)
    test_drivers.append(driver['id'])
    
    # Add location
    location = update_driver_location(http_client, driver['id'])
    driver['last_location'] = location
    
    return driver


@pytest.fixture
def api_endpoints():
    """API endpoints configuration."""
    from config import get_endpoints
    return get_endpoints()


@pytest.fixture
def nats_subjects():
    """NATS subjects configuration."""
    from config import get_nats_subjects
    return get_nats_subjects()


@pytest.fixture(autouse=True)
def log_test_info(request):
    """Log test information."""
    logger.info(f"Starting test: {request.node.name}")
    yield
    logger.info(f"Finished test: {request.node.name}")


# Test markers
def pytest_configure(config):
    """Configure pytest markers."""
    config.addinivalue_line("markers", "smoke: smoke tests")
    config.addinivalue_line("markers", "api: API tests")
    config.addinivalue_line("markers", "websocket: WebSocket tests")
    config.addinivalue_line("markers", "nats: NATS event tests")
    config.addinivalue_line("markers", "performance: Performance tests")
    config.addinivalue_line("markers", "integration: Full integration tests")


# Command line options
def pytest_addoption(parser):
    """Add command line options."""
    parser.addoption(
        "--service-url",
        action="store",
        default=config.http_base_url,
        help="Driver service base URL"
    )
    parser.addoption(
        "--skip-cleanup",
        action="store_true",
        default=False,
        help="Skip cleanup after tests"
    )
    parser.addoption(
        "--test-timeout",
        action="store",
        type=int,
        default=config.test_timeout,
        help="Test timeout in seconds"
    )


@pytest.fixture(scope="session")
def service_url(request):
    """Get service URL from command line or config."""
    return request.config.getoption("--service-url")


@pytest.fixture(scope="session")
def skip_cleanup(request):
    """Get skip cleanup flag from command line."""
    return request.config.getoption("--skip-cleanup")


@pytest.fixture(scope="session")
def test_timeout(request):
    """Get test timeout from command line or config."""
    return request.config.getoption("--test-timeout")


# Pytest collection hooks
def pytest_collection_modifyitems(config, items):
    """Modify test collection."""
    # Add smoke marker to fast tests
    for item in items:
        if "health" in item.name.lower() or "ping" in item.name.lower():
            item.add_marker(pytest.mark.smoke)
        
        # Add api marker to API tests
        if "api" in str(item.fspath).lower():
            item.add_marker(pytest.mark.api)