#!/bin/bash

# Driver Service Integration Tests Runner
# This script provides a convenient way to run integration tests with proper setup

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCKER_COMPOSE_FILE="$SCRIPT_DIR/docker-compose.test.yml"
ENV_FILE="$SCRIPT_DIR/.env"

# Default values
TEST_MODE="all"
CLEANUP_AFTER=true
VERBOSE=false
DOCKER_MODE=false
PERFORMANCE_ENABLED=false
TIMEOUT=300

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show help
show_help() {
    cat << EOF
Driver Service Integration Tests Runner

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -m, --mode MODE         Test mode: all, api, database, events, performance, scenarios (default: all)
    -t, --timeout SECONDS  Test timeout in seconds (default: 300)
    -d, --docker           Run tests in Docker containers
    -p, --performance      Enable performance tests
    -c, --no-cleanup       Skip cleanup after tests
    -v, --verbose          Verbose output
    -h, --help             Show this help message
    
    --quick                Quick test run (api + database only, short timeout)
    --full                 Full test suite with performance tests
    --ci                   CI mode (minimal output, no interactive prompts)

EXAMPLES:
    $0                              # Run all tests
    $0 -m api -v                   # Run API tests with verbose output
    $0 --docker --performance     # Run all tests in Docker with performance tests
    $0 --quick                     # Quick development test run
    $0 --full                      # Complete test suite
    
ENVIRONMENT VARIABLES:
    RUST_LOG                Log level (default: info,driver_service_integration_tests=debug)
    DRIVER_SERVICE_BASE_URL Service URL (default: http://localhost:8001)
    POSTGRES_HOST           PostgreSQL host (default: localhost)
    POSTGRES_PORT           PostgreSQL port (default: 5433)
    
For more configuration options, edit the .env file.
EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -m|--mode)
                TEST_MODE="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -d|--docker)
                DOCKER_MODE=true
                shift
                ;;
            -p|--performance)
                PERFORMANCE_ENABLED=true
                shift
                ;;
            -c|--no-cleanup)
                CLEANUP_AFTER=false
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            --quick)
                TEST_MODE="api"
                TIMEOUT=120
                PERFORMANCE_ENABLED=false
                shift
                ;;
            --full)
                TEST_MODE="all"
                TIMEOUT=600
                PERFORMANCE_ENABLED=true
                shift
                ;;
            --ci)
                export CI=true
                VERBOSE=false
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# Check dependencies
check_dependencies() {
    print_info "Checking dependencies..."
    
    local missing_deps=()
    
    if ! command -v cargo &> /dev/null; then
        missing_deps+=("cargo (Rust)")
    fi
    
    if [[ "$DOCKER_MODE" == true ]]; then
        if ! command -v docker &> /dev/null; then
            missing_deps+=("docker")
        fi
        if ! command -v docker-compose &> /dev/null; then
            missing_deps+=("docker-compose")
        fi
    else
        if ! command -v psql &> /dev/null; then
            missing_deps+=("psql (PostgreSQL client)")
        fi
        if ! command -v redis-cli &> /dev/null; then
            print_warning "redis-cli not found (Redis tests will be skipped)"
        fi
    fi
    
    if [[ ${#missing_deps[@]} -ne 0 ]]; then
        print_error "Missing dependencies: ${missing_deps[*]}"
        exit 1
    fi
    
    print_success "All dependencies satisfied"
}

# Setup environment
setup_environment() {
    print_info "Setting up test environment..."
    
    # Load environment variables
    if [[ -f "$ENV_FILE" ]]; then
        print_info "Loading environment from $ENV_FILE"
        source "$ENV_FILE"
    else
        print_info "Creating .env file from example"
        cp "$SCRIPT_DIR/.env.example" "$ENV_FILE"
        source "$ENV_FILE"
    fi
    
    # Set additional environment variables
    export PERFORMANCE_TESTS_ENABLED="$PERFORMANCE_ENABLED"
    
    if [[ "$VERBOSE" == true ]]; then
        export RUST_LOG="debug,driver_service_integration_tests=trace"
    else
        export RUST_LOG="${RUST_LOG:-info,driver_service_integration_tests=debug}"
    fi
    
    print_success "Environment configured"
}

# Start infrastructure services
start_infrastructure() {
    if [[ "$DOCKER_MODE" == true ]]; then
        print_info "Starting infrastructure with Docker..."
        docker-compose -f "$DOCKER_COMPOSE_FILE" up -d postgres-test redis-test nats-test
        
        print_info "Waiting for services to be ready..."
        sleep 15
        
        # Run migrations
        print_info "Running database migrations..."
        docker-compose -f "$DOCKER_COMPOSE_FILE" run --rm migrate
    else
        print_info "Infrastructure should be running externally"
        print_info "Expected services:"
        print_info "  - PostgreSQL on ${POSTGRES_HOST:-localhost}:${POSTGRES_PORT:-5433}"
        print_info "  - Redis on localhost:6380 (optional)"
        print_info "  - NATS on localhost:4222 (optional)"
        print_info "  - Driver Service on ${DRIVER_SERVICE_BASE_URL:-http://localhost:8001}"
        
        # Health check
        if ! curl -sf "${DRIVER_SERVICE_BASE_URL:-http://localhost:8001}/health" > /dev/null; then
            print_warning "Driver Service health check failed"
            print_warning "Make sure the service is running before continuing"
            
            if [[ "$CI" != true ]]; then
                read -p "Continue anyway? (y/N): " -n 1 -r
                echo
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    exit 1
                fi
            fi
        else
            print_success "Driver Service is healthy"
        fi
    fi
}

# Run integration tests
run_tests() {
    print_info "Running integration tests (mode: $TEST_MODE, timeout: ${TIMEOUT}s)..."
    
    local test_command
    local output_args=""
    
    if [[ "$DOCKER_MODE" == true ]]; then
        # Docker-based test execution
        test_command="docker-compose -f $DOCKER_COMPOSE_FILE --profile test run --rm rust-integration-tests"
        test_command="$test_command integration-tests --mode $TEST_MODE --timeout $TIMEOUT"
    else
        # Native test execution
        cd "$SCRIPT_DIR"
        test_command="cargo run --release -- --mode $TEST_MODE --timeout $TIMEOUT"
    fi
    
    # Add verbose flag if needed
    if [[ "$VERBOSE" == true ]]; then
        test_command="$test_command -vv"
    fi
    
    # Add JSON output for CI
    if [[ "$CI" == true ]]; then
        test_command="$test_command --output json"
        output_args="> test-results.json"
    fi
    
    # Execute tests
    print_info "Executing: $test_command $output_args"
    
    if eval "$test_command $output_args"; then
        print_success "Integration tests passed!"
        return 0
    else
        print_error "Integration tests failed!"
        return 1
    fi
}

# Cleanup function
cleanup() {
    if [[ "$CLEANUP_AFTER" == true ]]; then
        print_info "Cleaning up..."
        
        if [[ "$DOCKER_MODE" == true ]]; then
            docker-compose -f "$DOCKER_COMPOSE_FILE" down -v
            print_success "Docker resources cleaned up"
        fi
        
        # Clean up test results in CI mode
        if [[ "$CI" == true ]] && [[ -f "test-results.json" ]]; then
            print_info "Test results saved to test-results.json"
        fi
    else
        print_info "Skipping cleanup (--no-cleanup specified)"
    fi
}

# Signal handler for cleanup
trap cleanup EXIT INT TERM

# Main execution
main() {
    print_info "Driver Service Integration Tests Runner"
    print_info "======================================="
    
    parse_args "$@"
    check_dependencies
    setup_environment
    start_infrastructure
    
    local test_result=0
    run_tests || test_result=$?
    
    if [[ $test_result -eq 0 ]]; then
        print_success "🎉 All tests completed successfully!"
    else
        print_error "❌ Tests failed with exit code $test_result"
    fi
    
    exit $test_result
}

# Run main function with all arguments
main "$@"