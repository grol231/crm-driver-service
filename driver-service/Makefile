.PHONY: build run test clean docker-build docker-run migrate-up migrate-down migrate-create

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=driver-service
BINARY_PATH=./cmd/server

# Docker parameters
DOCKER_IMAGE=taxi-crm/driver-service
DOCKER_TAG=latest

# Database parameters
DB_HOST=localhost
DB_PORT=5432
DB_USER=driver_service
DB_PASSWORD=password
DB_NAME=driver_service
DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v $(BINARY_PATH)

# Build for production
build-prod:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) \
		-ldflags='-w -s -extldflags "-static"' \
		-a -installsuffix cgo \
		-o $(BINARY_NAME) $(BINARY_PATH)

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v $(BINARY_PATH) && ./$(BINARY_NAME)

# Run with live reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

# Test all packages
test:
	$(GOTEST) -v ./...

# Test with coverage
test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Integration tests
test-integration:
	$(GOTEST) -tags=integration -v ./tests/integration/...

# Performance tests
test-performance:
	$(GOTEST) -tags=integration -v -run="Performance" ./tests/integration/...

# End-to-end tests
test-e2e:
	$(GOTEST) -tags=integration -v -run="E2E" ./tests/integration/...

# All tests including integration
test-all: test test-integration

# Quick tests (skip performance)
test-quick:
	$(GOTEST) -short -v ./...
	$(GOTEST) -short -tags=integration -v ./tests/integration/...

# Test with race detection
test-race:
	$(GOTEST) -race -v ./...
	$(GOTEST) -race -tags=integration -v ./tests/integration/...

# Setup test environment
test-setup:
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for test services to be ready..."
	@sleep 10

# Teardown test environment
test-teardown:
	docker-compose -f docker-compose.test.yml down -v

# Run tests with script (recommended)
test-script:
	./scripts/run-tests.sh

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Docker build
docker-build:
	docker build -f deployments/docker/Dockerfile -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Docker run
docker-run:
	docker run -p 8001:8001 -p 9001:9001 -p 9002:9002 \
		-e DRIVER_SERVICE_DATABASE_HOST=host.docker.internal \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Docker compose up
docker-up:
	docker-compose -f deployments/docker/docker-compose.yml up -d

# Docker compose down
docker-down:
	docker-compose -f deployments/docker/docker-compose.yml down

# Docker compose logs
docker-logs:
	docker-compose -f deployments/docker/docker-compose.yml logs -f

# Install migrate tool (run once)
install-migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run database migrations up
migrate-up:
	migrate -path internal/infrastructure/database/migrations -database "$(DB_URL)" up

# Run database migrations down
migrate-down:
	migrate -path internal/infrastructure/database/migrations -database "$(DB_URL)" down

# Create new migration
migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	migrate create -ext sql -dir internal/infrastructure/database/migrations -seq $(NAME)

# Force migration version
migrate-force:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make migrate-force VERSION=version_number"; exit 1; fi
	migrate -path internal/infrastructure/database/migrations -database "$(DB_URL)" force $(VERSION)

# Drop database
migrate-drop:
	migrate -path internal/infrastructure/database/migrations -database "$(DB_URL)" drop

# Kubernetes deploy
k8s-deploy:
	kubectl apply -f deployments/k8s/

# Kubernetes delete
k8s-delete:
	kubectl delete -f deployments/k8s/

# Kubernetes logs
k8s-logs:
	kubectl logs -f deployment/driver-service

# Generate mocks (requires mockgen)
generate-mocks:
	go generate ./...

# Security scan
security-scan:
	gosec ./...

# Benchmark tests
benchmark:
	$(GOTEST) -bench=. -benchmem ./...

# Load test (requires hey: go install github.com/rakyll/hey@latest)
load-test:
	hey -n 1000 -c 10 http://localhost:8001/health

# Check for vulnerabilities
vuln-check:
	govulncheck ./...

# All checks (lint, test, security)
check: lint test security-scan

# Full CI pipeline
ci: deps check build

# Help
help:
	@echo "Available commands:"
	@echo "  build          - Build the binary"
	@echo "  build-prod     - Build production binary"
	@echo "  run            - Build and run the application"
	@echo "  dev            - Run with live reload (requires air)"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Download dependencies"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-up      - Start with docker-compose"
	@echo "  docker-down    - Stop docker-compose"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback migrations"
	@echo "  k8s-deploy     - Deploy to Kubernetes"
	@echo "  help           - Show this help"
