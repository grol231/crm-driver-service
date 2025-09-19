# Driver Service - Rust Integration Tests

Comprehensive integration test suite for the Driver Service written in Rust. This test suite provides thorough testing of all API endpoints, database operations, event-driven workflows, and real-world scenarios.

## Features

- **Complete API Testing**: Tests all HTTP endpoints with various scenarios
- **Database Integration**: Direct database testing with consistency checks
- **Event-Driven Testing**: NATS event publishing and consumption tests
- **Performance Testing**: Load testing and stress testing capabilities
- **Integration Scenarios**: End-to-end workflow testing
- **Docker Support**: Containerized test execution
- **Multiple Output Formats**: Console, JSON, and JUnit XML reports
- **Comprehensive Logging**: Detailed test execution logs

## Architecture

The test suite is organized into several modules:

```
src/
├── lib.rs                     # Main library and test environment setup
├── config.rs                  # Configuration management
├── fixtures.rs                # Test data generators and fixtures
├── clients/
│   ├── api_client.rs          # HTTP API client
│   ├── nats_client.rs         # NATS event client
│   └── database_client.rs     # Direct database client
├── helpers/
│   ├── docker_helper.rs       # Docker container management
│   ├── redis_helper.rs        # Redis operations
│   └── test_utilities.rs      # Common test utilities
├── tests/
│   ├── driver_api_tests.rs    # Driver API integration tests
│   ├── location_api_tests.rs  # Location API integration tests
│   ├── database_tests.rs      # Database integration tests
│   ├── event_tests.rs         # NATS event tests
│   ├── performance_tests.rs   # Performance and load tests
│   └── integration_scenarios.rs # End-to-end scenarios
└── main.rs                    # CLI test runner
```

## Quick Start

### Prerequisites

- Rust 1.75 or later
- Docker and Docker Compose
- PostgreSQL client tools (for database operations)
- Redis CLI (for cache operations)

### Setup

1. **Clone and navigate to the test directory:**
```bash
cd driver-service/rust-integration-tests
```

2. **Copy and configure environment variables:**
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. **Start test infrastructure:**
```bash
docker-compose -f docker-compose.test.yml up -d postgres-test redis-test nats-test
```

4. **Run database migrations:**
```bash
docker-compose -f docker-compose.test.yml run --rm migrate
```

5. **Start the Driver Service:**
```bash
cd ../
make run-test  # or docker-compose -f rust-integration-tests/docker-compose.test.yml up driver-service
```

### Running Tests

#### Basic Usage

```bash
# Run all tests
cargo run -- --mode all

# Run specific test categories
cargo run -- --mode api
cargo run -- --mode database
cargo run -- --mode events
cargo run -- --mode performance
cargo run -- --mode scenarios

# Run with custom filter
cargo run -- --mode custom --filter "driver_creation"

# Verbose output
cargo run -- --mode all -vv

# JSON output for CI/CD
cargo run -- --mode all --output json > test-results.json
```

#### Using Docker

```bash
# Run tests in containers
docker-compose -f docker-compose.test.yml --profile test up rust-integration-tests

# Run specific test mode
docker-compose -f docker-compose.test.yml --profile test run rust-integration-tests integration-tests --mode api

# Run with custom configuration
docker run --rm \
  -e DRIVER_SERVICE_BASE_URL=http://your-service:8001 \
  -e PERFORMANCE_TESTS_ENABLED=true \
  driver-service-rust-integration-tests \
  integration-tests --mode performance
```

#### Advanced Usage

```bash
# Run tests with timeout
cargo run -- --mode all --timeout 600

# Run tests in parallel (experimental)
cargo run -- --mode all --parallel

# Enable cleanup after tests
cargo run -- --mode all --cleanup

# Custom configuration file
cargo run -- --mode all --config custom-config.toml

# JUnit XML output for CI systems
cargo run -- --mode all --output junit > test-results.xml
```

## Test Categories

### 1. API Integration Tests (`driver_api_tests.rs`)

Tests all HTTP API endpoints:

- **CRUD Operations**: Create, read, update, delete drivers
- **Validation**: Input validation and error handling
- **Authentication**: API security and permissions
- **Pagination**: List endpoints with pagination
- **Filtering**: Search and filter functionality
- **Status Management**: Driver status transitions
- **Error Handling**: Various error scenarios

**Key Tests:**
- `test_create_driver_success`
- `test_create_driver_validation`
- `test_list_drivers_pagination`
- `test_change_driver_status`
- `test_concurrent_driver_operations`

### 2. Location API Tests (`location_api_tests.rs`)

Tests GPS tracking and location functionality:

- **Location Updates**: Single and batch GPS updates
- **Location History**: Historical location data
- **Nearby Search**: Find drivers near coordinates
- **Real-time Tracking**: WebSocket connections
- **Geospatial Queries**: Distance and area calculations

**Key Tests:**
- `test_update_location_success`
- `test_batch_update_locations`
- `test_get_nearby_drivers`
- `test_location_tracking_performance`

### 3. Database Tests (`database_tests.rs`)

Direct database testing:

- **CRUD Operations**: Low-level database operations
- **Constraints**: Foreign keys, unique constraints, check constraints
- **Triggers**: Automatic timestamp updates
- **Transactions**: ACID compliance
- **Performance**: Query optimization and indexing
- **Data Consistency**: Referential integrity

**Key Tests:**
- `test_database_constraints`
- `test_rating_statistics`
- `test_database_transactions`
- `test_data_consistency`

### 4. Event Tests (`event_tests.rs`)

NATS event-driven testing:

- **Event Publishing**: Outgoing events from Driver Service
- **Event Consumption**: Processing incoming events
- **Message Ordering**: Event sequence verification
- **Error Handling**: Event processing failures
- **High Volume**: Event throughput testing

**Key Tests:**
- `test_driver_registration_events`
- `test_location_update_events`
- `test_order_assignment_events`
- `test_high_volume_events`

### 5. Performance Tests (`performance_tests.rs`)

Load and stress testing:

- **API Throughput**: Requests per second under load
- **Database Performance**: Concurrent database operations
- **Memory Usage**: Resource consumption monitoring
- **Stress Testing**: System behavior under extreme load
- **Response Times**: Latency measurements

**Key Tests:**
- `test_api_throughput_load`
- `test_stress_extreme_load`
- `test_database_concurrent_performance`
- `test_resource_usage_under_load`

### 6. Integration Scenarios (`integration_scenarios.rs`)

End-to-end workflow testing:

- **Driver Onboarding**: Complete registration process
- **Ride Lifecycle**: Full trip from assignment to completion
- **Multi-Driver Coordination**: Concurrent driver management
- **Shift Tracking**: Working hours and earnings
- **Peak Hours**: High-traffic simulation

**Key Tests:**
- `test_complete_driver_onboarding_scenario`
- `test_complete_ride_lifecycle_scenario`
- `test_multi_driver_coordination_scenario`
- `test_peak_hours_stress_scenario`

## Configuration

### Environment Variables

```bash
# Service Configuration
DRIVER_SERVICE_BASE_URL=http://localhost:8001
DRIVER_SERVICE_PORT=8001
HTTP_TIMEOUT_SECONDS=30

# Database Configuration  
POSTGRES_HOST=localhost
POSTGRES_PORT=5433
POSTGRES_DB=driver_service_test
POSTGRES_USER=test_user
POSTGRES_PASSWORD=test_password

# Event System Configuration
NATS_ENABLED=true
NATS_URL=nats://localhost:4222

# Cache Configuration
REDIS_ENABLED=true
REDIS_URL=redis://localhost:6380

# Test Behavior
TEST_CLEANUP_AFTER_EACH=true
PERFORMANCE_TESTS_ENABLED=false
LOAD_TEST_USERS=100
TEST_MAX_DURATION_SECONDS=300
```

### Configuration File

Create `config.toml` for advanced configuration:

```toml
[driver_service]
base_url = "http://localhost:8001"
port = 8001
timeout_seconds = 30

[database]
host = "localhost"
port = 5433
database = "driver_service_test"
username = "test_user"
password = "test_password"
max_connections = 10

[nats]
enabled = true
url = "nats://localhost:4222"

[test]
cleanup_after_each = true
parallel_execution = false
performance_test_enabled = false
load_test_users = 100
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: driver_service_test
          POSTGRES_USER: test_user
          POSTGRES_PASSWORD: test_password
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      redis:
        image: redis:7-alpine
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Rust
      uses: actions-rs/toolchain@v1
      with:
        toolchain: stable
        
    - name: Start Driver Service
      run: |
        cd driver-service
        make build
        make run-test &
        sleep 10
        
    - name: Run Integration Tests
      run: |
        cd rust-integration-tests
        cargo run -- --mode all --output junit > test-results.xml
        
    - name: Publish Test Results
      uses: EnricoMi/publish-unit-test-result-action@v2
      if: always()
      with:
        files: rust-integration-tests/test-results.xml
```

### Docker-based CI

```dockerfile
# Dockerfile.ci
FROM rust:1.75-slim

WORKDIR /app
COPY . .

RUN cargo build --release

CMD ["./target/release/test_runner", "--mode", "all", "--output", "json"]
```

## Troubleshooting

### Common Issues

1. **Service Not Available**
   ```bash
   # Check if Driver Service is running
   curl http://localhost:8001/health
   
   # Check Docker services
   docker-compose -f docker-compose.test.yml ps
   ```

2. **Database Connection Issues**
   ```bash
   # Test database connection
   psql -h localhost -p 5433 -U test_user -d driver_service_test
   
   # Check migrations
   docker-compose -f docker-compose.test.yml run --rm migrate
   ```

3. **NATS Connection Issues**
   ```bash
   # Check NATS server
   docker-compose -f docker-compose.test.yml logs nats-test
   
   # Disable NATS tests if not needed
   export NATS_ENABLED=false
   ```

4. **Performance Test Timeouts**
   ```bash
   # Increase timeout
   cargo run -- --mode performance --timeout 600
   
   # Reduce load test size
   export LOAD_TEST_USERS=10
   ```

### Debug Mode

```bash
# Enable debug logging
export RUST_LOG=debug,driver_service_integration_tests=trace

# Run single test with full output
cargo test test_create_driver_success -- --nocapture

# Use test-specific environment
cp .env.debug .env
```

### Log Analysis

```bash
# View test execution logs
cargo run -- --mode all -vv 2>&1 | tee test-execution.log

# Filter specific component logs
grep "api_client" test-execution.log

# Performance analysis
grep "Performance" test-execution.log | sort -k2
```

## Contributing

### Adding New Tests

1. **Create test function:**
```rust
#[tokio::test]
#[serial]
async fn test_your_new_feature() -> Result<()> {
    let env = init_test_environment().await?;
    env.cleanup().await?;
    
    // Your test logic here
    
    Ok(())
}
```

2. **Add to appropriate module** (`driver_api_tests.rs`, etc.)

3. **Update test runner** if needed in `main.rs`

4. **Add documentation** and examples

### Code Style

- Use `#[serial]` for tests that modify shared state
- Always clean up test data with `env.cleanup().await?`
- Use meaningful assertion messages
- Add `println!` statements for test progress in scenarios
- Handle errors appropriately with `Result<()>`

### Performance Considerations

- Mark expensive tests with performance gates
- Use `#[ignore]` for very slow tests
- Consider parallel execution where safe
- Monitor resource usage in CI

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## Support

For issues and questions:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review existing [GitHub Issues](https://github.com/your-org/driver-service/issues)
3. Create a new issue with detailed information
4. Join our [Discord/Slack channel] for real-time help