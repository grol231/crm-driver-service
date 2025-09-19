# Driver Service Rust Integration Tests Makefile

.PHONY: help build test test-api test-db test-events test-perf test-scenarios clean setup check lint fmt docker-build docker-test docker-clean

# Default target
help: ## Show this help message
	@echo "Driver Service Rust Integration Tests"
	@echo "====================================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Variables
CARGO := cargo
DOCKER := docker
DOCKER_COMPOSE := docker-compose -f docker-compose.test.yml
RUST_LOG ?= info,driver_service_integration_tests=debug
TEST_TIMEOUT ?= 300

# Build targets
build: ## Build the integration tests
	$(CARGO) build --release

build-dev: ## Build with debug symbols
	$(CARGO) build

check: ## Check code without building
	$(CARGO) check

lint: ## Run clippy linting
	$(CARGO) clippy -- -D warnings

fmt: ## Format code
	$(CARGO) fmt

fmt-check: ## Check code formatting
	$(CARGO) fmt -- --check

# Test infrastructure setup
setup: ## Setup test infrastructure (PostgreSQL, Redis, NATS)
	$(DOCKER_COMPOSE) up -d postgres-test redis-test nats-test
	@echo "Waiting for services to be ready..."
	sleep 10
	$(DOCKER_COMPOSE) run --rm migrate
	@echo "Test infrastructure ready!"

setup-minimal: ## Setup minimal infrastructure (PostgreSQL only)
	$(DOCKER_COMPOSE) up -d postgres-test
	@echo "Waiting for PostgreSQL..."
	sleep 5
	$(DOCKER_COMPOSE) run --rm migrate
	@echo "Minimal test infrastructure ready!"

teardown: ## Stop and remove test infrastructure
	$(DOCKER_COMPOSE) down -v
	docker system prune -f

# Test execution
test: ## Run all integration tests
	RUST_LOG=$(RUST_LOG) $(CARGO) run -- --mode all --timeout $(TEST_TIMEOUT)

test-quick: ## Run tests without performance tests
	RUST_LOG=$(RUST_LOG) $(CARGO) run -- --mode all --timeout 120

test-api: ## Run API integration tests only
	RUST_LOG=$(RUST_LOG) $(CARGO) run -- --mode api

test-db: ## Run database tests only
	RUST_LOG=$(RUST_LOG) $(CARGO) run -- --mode database

test-events: ## Run event/NATS tests only
	RUST_LOG=$(RUST_LOG) $(CARGO) run -- --mode events

test-perf: ## Run performance tests only
	PERFORMANCE_TESTS_ENABLED=true RUST_LOG=$(RUST_LOG) $(CARGO) run -- --mode performance --timeout 600

test-scenarios: ## Run integration scenarios only
	RUST_LOG=$(RUST_LOG) $(CARGO) run -- --mode scenarios

test-filter: ## Run tests matching filter (usage: make test-filter FILTER=driver_creation)
	RUST_LOG=$(RUST_LOG) $(CARGO) run -- --mode custom --filter "$(FILTER)"

test-verbose: ## Run tests with verbose output
	RUST_LOG=debug,driver_service_integration_tests=trace $(CARGO) run -- --mode all -vv

test-json: ## Run tests with JSON output
	$(CARGO) run -- --mode all --output json > test-results.json
	@echo "Results saved to test-results.json"

test-junit: ## Run tests with JUnit XML output
	$(CARGO) run -- --mode all --output junit > test-results.xml
	@echo "Results saved to test-results.xml"

# Load testing
load-test: ## Run load tests with default settings
	PERFORMANCE_TESTS_ENABLED=true LOAD_TEST_USERS=100 $(CARGO) run -- --mode performance

stress-test: ## Run stress tests with high load
	PERFORMANCE_TESTS_ENABLED=true LOAD_TEST_USERS=500 $(CARGO) run -- --mode performance --timeout 900

# Docker targets
docker-build: ## Build Docker image
	$(DOCKER) build -t driver-service-rust-integration-tests .

docker-test: ## Run tests in Docker containers
	$(DOCKER_COMPOSE) --profile test up --build --abort-on-container-exit rust-integration-tests

docker-test-api: ## Run API tests in Docker
	$(DOCKER_COMPOSE) --profile test run --rm rust-integration-tests integration-tests --mode api

docker-test-full: ## Run full test suite in Docker with services
	$(DOCKER_COMPOSE) up -d postgres-test redis-test nats-test driver-service
	sleep 15
	$(DOCKER_COMPOSE) --profile test run --rm rust-integration-tests integration-tests --mode all
	$(DOCKER_COMPOSE) down

docker-clean: ## Clean Docker resources
	$(DOCKER_COMPOSE) down -v
	$(DOCKER) system prune -f
	$(DOCKER) volume prune -f

# Development helpers
dev-setup: ## Setup development environment
	cp .env.example .env
	$(MAKE) setup
	@echo "Development environment ready!"
	@echo "Edit .env file to customize configuration"

dev-clean: ## Clean development environment
	$(MAKE) teardown
	rm -f .env test-results.json test-results.xml
	$(CARGO) clean

dev-reset: ## Reset development environment
	$(MAKE) dev-clean
	$(MAKE) dev-setup

# Database management
db-reset: ## Reset test database
	$(DOCKER_COMPOSE) stop postgres-test
	$(DOCKER_COMPOSE) rm -f postgres-test
	docker volume rm -f rust-integration-tests_postgres_test_data
	$(DOCKER_COMPOSE) up -d postgres-test
	sleep 5
	$(DOCKER_COMPOSE) run --rm migrate

db-migrate: ## Run database migrations
	$(DOCKER_COMPOSE) run --rm migrate

db-connect: ## Connect to test database
	$(DOCKER_COMPOSE) exec postgres-test psql -U test_user -d driver_service_test

db-logs: ## Show database logs
	$(DOCKER_COMPOSE) logs postgres-test

# Service management
services-up: ## Start all services (DB, Redis, NATS, Driver Service)
	$(DOCKER_COMPOSE) up -d postgres-test redis-test nats-test driver-service

services-down: ## Stop all services
	$(DOCKER_COMPOSE) down

services-logs: ## Show logs from all services
	$(DOCKER_COMPOSE) logs -f

services-status: ## Show service status
	$(DOCKER_COMPOSE) ps

driver-service-logs: ## Show Driver Service logs
	$(DOCKER_COMPOSE) logs -f driver-service

# Monitoring and debugging
logs: ## Show test execution logs
	tail -f test-execution.log

monitor: ## Monitor test infrastructure
	watch "$(DOCKER_COMPOSE) ps && echo '---' && docker stats --no-stream"

health-check: ## Check health of all services
	@echo "Checking service health..."
	@curl -s http://localhost:8001/health && echo " ✓ Driver Service" || echo " ✗ Driver Service"
	@$(DOCKER_COMPOSE) exec -T postgres-test pg_isready -U test_user && echo " ✓ PostgreSQL" || echo " ✗ PostgreSQL"
	@$(DOCKER_COMPOSE) exec -T redis-test redis-cli ping >/dev/null && echo " ✓ Redis" || echo " ✗ Redis"
	@curl -s http://localhost:8222/healthz >/dev/null && echo " ✓ NATS" || echo " ✗ NATS"

# Cleanup targets
clean: ## Clean build artifacts
	$(CARGO) clean
	rm -f test-results.json test-results.xml test-execution.log

clean-all: ## Clean everything including Docker resources
	$(MAKE) clean
	$(MAKE) docker-clean

# CI/CD helpers
ci-setup: ## Setup for CI environment
	$(DOCKER_COMPOSE) up -d postgres-test redis-test nats-test
	sleep 10
	$(DOCKER_COMPOSE) run --rm migrate

ci-test: ## Run tests for CI environment
	$(CARGO) run -- --mode all --output junit --timeout 600 > test-results.xml

ci-cleanup: ## Cleanup after CI run
	$(DOCKER_COMPOSE) down -v

# Performance profiling
profile: ## Run tests with profiling
	RUST_LOG=info $(CARGO) run --release -- --mode performance --timeout 300

benchmark: ## Run benchmark tests
	$(CARGO) bench

# Security scanning
security-audit: ## Run security audit
	$(CARGO) audit

# Dependencies
deps-update: ## Update dependencies
	$(CARGO) update

deps-check: ## Check for outdated dependencies
	$(CARGO) outdated

# Documentation
docs: ## Generate and open documentation
	$(CARGO) doc --open

docs-check: ## Check documentation
	$(CARGO) doc --no-deps

# Example usage targets
example-basic: ## Example: Run basic API tests
	@echo "Running basic API tests..."
	$(MAKE) setup-minimal
	RUST_LOG=info $(CARGO) run -- --mode api
	@echo "Basic test completed!"

example-full: ## Example: Run full test suite with all services
	@echo "Running full test suite..."
	$(MAKE) setup
	PERFORMANCE_TESTS_ENABLED=true $(CARGO) run -- --mode all --timeout 600
	@echo "Full test suite completed!"

example-load: ## Example: Run load testing
	@echo "Running load tests..."
	$(MAKE) setup
	PERFORMANCE_TESTS_ENABLED=true LOAD_TEST_USERS=50 $(CARGO) run -- --mode performance
	@echo "Load testing completed!"

# Help for common workflows
workflow-dev: ## Workflow: Development testing
	@echo "Development Testing Workflow:"
	@echo "1. make dev-setup    # Setup environment"
	@echo "2. make test-api     # Test API changes"
	@echo "3. make test-db      # Test database changes"
	@echo "4. make test         # Full test suite"

workflow-ci: ## Workflow: CI/CD testing
	@echo "CI/CD Testing Workflow:"
	@echo "1. make ci-setup     # Setup CI environment"
	@echo "2. make ci-test      # Run tests with reports"
	@echo "3. make ci-cleanup   # Clean up resources"

workflow-perf: ## Workflow: Performance testing
	@echo "Performance Testing Workflow:"
	@echo "1. make setup        # Setup full infrastructure"
	@echo "2. make load-test    # Run load tests"
	@echo "3. make stress-test  # Run stress tests"
	@echo "4. make profile      # Profile performance"