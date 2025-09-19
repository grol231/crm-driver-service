.PHONY: help install test test-docker clean setup lint format

# Default target
help:
	@echo "Driver Service Python Integration Tests"
	@echo ""
	@echo "Available targets:"
	@echo "  help              Show this help message"
	@echo "  install           Install Python dependencies"
	@echo "  setup             Setup test environment"
	@echo "  test              Run all tests locally"
	@echo "  test-docker       Run tests in Docker containers"
	@echo "  smoke             Run smoke tests only"
	@echo "  api               Run API tests only"
	@echo "  websocket         Run WebSocket tests only"
	@echo "  nats              Run NATS event tests only"
	@echo "  grpc              Run gRPC tests only"
	@echo "  performance       Run performance tests"
	@echo "  lint              Run linting checks"
	@echo "  format            Format code"
	@echo "  clean             Clean up test artifacts"
	@echo "  docker-build      Build Docker images"
	@echo "  docker-up         Start Docker services"
	@echo "  docker-down       Stop Docker services"
	@echo "  docker-logs       Show Docker logs"

# Python virtual environment
VENV ?= venv
PYTHON = $(VENV)/bin/python
PIP = $(VENV)/bin/pip

# Install dependencies
install:
	@echo "Installing Python dependencies..."
	python -m venv $(VENV)
	$(PIP) install --upgrade pip
	$(PIP) install -r requirements.txt

# Setup test environment
setup: install
	@echo "Setting up test environment..."
	cp .env.example .env
	mkdir -p test-results/htmlcov
	@echo "Environment setup complete. Edit .env file with your configuration."

# Run all tests locally
test:
	@echo "Running all integration tests..."
	$(PYTHON) -m pytest tests/ \
		--maxfail=10 \
		--tb=short \
		--html=test-results/report.html \
		--self-contained-html \
		--junitxml=test-results/junit.xml \
		--cov=. \
		--cov-report=html:test-results/htmlcov \
		--cov-report=term-missing \
		-v

# Run tests in Docker
test-docker: docker-build
	@echo "Running tests in Docker..."
	docker-compose -f docker-compose.integration-tests.yml run --rm python-tests

# Run specific test suites
smoke:
	@echo "Running smoke tests..."
	$(PYTHON) -m pytest tests/ -m smoke \
		--maxfail=5 \
		--tb=line \
		--html=test-results/smoke-report.html \
		--self-contained-html \
		-v

api:
	@echo "Running API tests..."
	$(PYTHON) -m pytest tests/test_driver_api.py tests/test_location_api.py \
		--maxfail=10 \
		--tb=short \
		--html=test-results/api-report.html \
		--self-contained-html \
		-v

websocket:
	@echo "Running WebSocket tests..."
	$(PYTHON) -m pytest tests/test_websocket.py -m websocket \
		--maxfail=5 \
		--tb=short \
		--html=test-results/websocket-report.html \
		--self-contained-html \
		-v

nats:
	@echo "Running NATS event tests..."
	$(PYTHON) -m pytest tests/test_nats_events.py -m nats \
		--maxfail=5 \
		--tb=short \
		--html=test-results/nats-report.html \
		--self-contained-html \
		-v

grpc:
	@echo "Running gRPC tests..."
	$(PYTHON) -m pytest tests/test_grpc_api.py -m grpc \
		--maxfail=5 \
		--tb=short \
		--html=test-results/grpc-report.html \
		--self-contained-html \
		-v

# Performance tests
performance:
	@echo "Running performance tests..."
	docker-compose -f docker-compose.integration-tests.yml up -d driver-service
	@sleep 30
	docker-compose -f docker-compose.integration-tests.yml run --rm performance-tests
	docker-compose -f docker-compose.integration-tests.yml down

# Code quality
lint:
	@echo "Running linting checks..."
	$(PYTHON) -m flake8 tests/ utils/ config.py
	$(PYTHON) -m black --check tests/ utils/ config.py
	$(PYTHON) -m isort --check-only tests/ utils/ config.py

format:
	@echo "Formatting code..."
	$(PYTHON) -m black tests/ utils/ config.py
	$(PYTHON) -m isort tests/ utils/ config.py

# Docker operations
docker-build:
	@echo "Building Docker images..."
	docker-compose -f docker-compose.integration-tests.yml build

docker-up:
	@echo "Starting Docker services..."
	docker-compose -f docker-compose.integration-tests.yml up -d

docker-down:
	@echo "Stopping Docker services..."
	docker-compose -f docker-compose.integration-tests.yml down -v

docker-logs:
	@echo "Showing Docker logs..."
	docker-compose -f docker-compose.integration-tests.yml logs -f

# Specific Docker test suites
docker-smoke:
	@echo "Running smoke tests in Docker..."
	docker-compose -f docker-compose.integration-tests.yml run --rm smoke-tests

docker-api:
	@echo "Running API tests in Docker..."
	docker-compose -f docker-compose.integration-tests.yml run --rm api-tests

docker-websocket:
	@echo "Running WebSocket tests in Docker..."
	docker-compose -f docker-compose.integration-tests.yml run --rm websocket-tests

docker-nats:
	@echo "Running NATS tests in Docker..."
	docker-compose -f docker-compose.integration-tests.yml run --rm nats-tests

docker-grpc:
	@echo "Running gRPC tests in Docker..."
	docker-compose -f docker-compose.integration-tests.yml run --rm grpc-tests

# Test utilities
test-health:
	@echo "Testing service health..."
	curl -f http://localhost:8001/health || echo "Service not available"

wait-for-services:
	@echo "Waiting for services to be ready..."
	@timeout 60 bash -c 'until curl -f http://localhost:8001/health; do sleep 2; done'
	@echo "Services are ready!"

# Database operations
db-reset:
	@echo "Resetting test database..."
	docker-compose -f docker-compose.integration-tests.yml stop test-postgres
	docker-compose -f docker-compose.integration-tests.yml rm -f test-postgres
	docker volume rm python-integration-tests_test_postgres_data 2>/dev/null || true
	docker-compose -f docker-compose.integration-tests.yml up -d test-postgres

# Cleanup
clean:
	@echo "Cleaning up test artifacts..."
	rm -rf test-results/
	rm -rf .pytest_cache/
	rm -rf __pycache__/
	rm -rf tests/__pycache__/
	rm -rf utils/__pycache__/
	rm -rf performance/__pycache__/
	rm -rf *.log
	rm -rf .coverage
	rm -rf htmlcov/
	docker system prune -f 2>/dev/null || true

clean-docker:
	@echo "Cleaning up Docker resources..."
	docker-compose -f docker-compose.integration-tests.yml down -v --remove-orphans
	docker image prune -f
	docker volume prune -f

# Development helpers
dev-setup: setup
	@echo "Setting up development environment..."
	$(PIP) install black flake8 isort mypy
	@echo "Development environment ready!"

check-deps:
	@echo "Checking dependency versions..."
	$(PIP) list --outdated

update-deps:
	@echo "Updating dependencies..."
	$(PIP) install --upgrade -r requirements.txt

# CI/CD helpers
ci-test:
	@echo "Running tests for CI/CD..."
	$(PYTHON) -m pytest tests/ \
		--maxfail=5 \
		--tb=line \
		--junitxml=test-results/junit.xml \
		--cov=. \
		--cov-report=xml:test-results/coverage.xml \
		--cov-report=term

ci-docker:
	@echo "Running Docker tests for CI/CD..."
	docker-compose -f docker-compose.integration-tests.yml run --rm \
		-e CI=true python-tests \
		python -m pytest tests/ \
			--maxfail=5 \
			--tb=line \
			--junitxml=test-results/junit.xml \
			--cov=. \
			--cov-report=xml:test-results/coverage.xml

# Generate test report
report:
	@echo "Generating comprehensive test report..."
	mkdir -p test-results
	@echo "# Integration Test Report" > test-results/README.md
	@echo "" >> test-results/README.md
	@echo "Generated on: $$(date)" >> test-results/README.md
	@echo "" >> test-results/README.md
	@echo "## Test Results" >> test-results/README.md
	@echo "- HTML Report: report.html" >> test-results/README.md
	@echo "- JUnit XML: junit.xml" >> test-results/README.md
	@echo "- Coverage Report: htmlcov/index.html" >> test-results/README.md
	@echo "" >> test-results/README.md
	@echo "Test report generated in test-results/ directory"

# Show status
status:
	@echo "Docker Services Status:"
	@docker-compose -f docker-compose.integration-tests.yml ps
	@echo ""
	@echo "Test Results:"
	@ls -la test-results/ 2>/dev/null || echo "No test results found"