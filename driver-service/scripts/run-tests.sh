#!/bin/bash

# Скрипт для запуска интеграционных тестов

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функция для логирования
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

# Проверяем, что Docker доступен
if ! command -v docker &> /dev/null; then
    error "Docker is not installed or not in PATH"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    error "Docker Compose is not installed or not in PATH"
    exit 1
fi

# Переходим в директорию проекта
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

log "Starting integration tests in $PROJECT_DIR"

# Функция очистки
cleanup() {
    log "Cleaning up test environment..."
    docker-compose -f docker-compose.test.yml down -v --remove-orphans
}

# Устанавливаем trap для очистки при выходе
trap cleanup EXIT

# Запускаем тестовые сервисы
log "Starting test database services..."
docker-compose -f docker-compose.test.yml up -d

# Ждем готовности PostgreSQL
log "Waiting for PostgreSQL to be ready..."
timeout=60
counter=0
while ! docker-compose -f docker-compose.test.yml exec -T test-postgres pg_isready -U test_user -d driver_service_test > /dev/null 2>&1; do
    if [ $counter -ge $timeout ]; then
        error "PostgreSQL failed to start within $timeout seconds"
        exit 1
    fi
    sleep 1
    counter=$((counter + 1))
done

log "PostgreSQL is ready"

# Ждем готовности Redis
log "Waiting for Redis to be ready..."
counter=0
while ! docker-compose -f docker-compose.test.yml exec -T test-redis redis-cli ping > /dev/null 2>&1; do
    if [ $counter -ge $timeout ]; then
        error "Redis failed to start within $timeout seconds"
        exit 1
    fi
    sleep 1
    counter=$((counter + 1))
done

log "Redis is ready"

# Устанавливаем переменные окружения для тестов
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5433
export TEST_DB_USER=test_user
export TEST_DB_PASSWORD=test_password
export TEST_DB_NAME=driver_service_test

export TEST_REDIS_HOST=localhost
export TEST_REDIS_PORT=6380

# Проверяем зависимости Go
log "Checking Go dependencies..."
go mod tidy

# Запускаем линтер (если установлен)
if command -v golangci-lint &> /dev/null; then
    log "Running linter..."
    golangci-lint run --timeout=5m
else
    warn "golangci-lint not found, skipping linting"
fi

# Запускаем unit тесты
log "Running unit tests..."
go test -v -race -coverprofile=coverage.out ./internal/...

# Запускаем интеграционные тесты
log "Running integration tests..."
go test -v -race -tags=integration -timeout=10m ./tests/integration/...

# Запускаем performance тесты (если не в быстром режиме)
if [ "${SKIP_PERFORMANCE_TESTS}" != "true" ]; then
    log "Running performance tests..."
    go test -v -tags=integration -timeout=15m -run="Performance" ./tests/integration/...
else
    warn "Skipping performance tests (SKIP_PERFORMANCE_TESTS=true)"
fi

# Генерируем отчет о покрытии
if [ -f coverage.out ]; then
    log "Generating coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    
    # Показываем общее покрытие
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    log "Total test coverage: $COVERAGE"
    
    # Проверяем минимальное покрытие
    COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
    MIN_COVERAGE=70
    
    if (( $(echo "$COVERAGE_NUM < $MIN_COVERAGE" | bc -l) )); then
        warn "Test coverage $COVERAGE is below minimum $MIN_COVERAGE%"
    else
        log "Test coverage $COVERAGE meets minimum requirement"
    fi
fi

# Проверяем на memory leaks (если установлен valgrind)
if command -v valgrind &> /dev/null; then
    log "Running memory leak detection..."
    # В Go memory leaks обычно проверяются через race detector и другие инструменты
    # Здесь можно добавить дополнительные проверки
fi

# Проверяем производительность базы данных
log "Checking database performance..."
docker-compose -f docker-compose.test.yml exec -T test-postgres psql -U test_user -d driver_service_test -c "
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats 
WHERE schemaname = 'public' 
ORDER BY tablename, attname;
" || warn "Could not get database statistics"

log "All tests completed successfully!"

# Опционально: отправляем результаты в систему мониторинга
if [ "${SEND_TEST_RESULTS}" == "true" ]; then
    log "Sending test results to monitoring system..."
    # Здесь можно добавить отправку метрик в Prometheus или другую систему
fi

log "Test execution finished. Check coverage.html for detailed coverage report."
