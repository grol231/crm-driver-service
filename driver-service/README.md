# Driver Service

Driver Service является центральным компонентом микросервисной архитектуры CRM системы для таксопарков, отвечающим за полный жизненный цикл управления водителями.

## Основные функции

- 🚗 **Управление водителями**: регистрация, обновление профилей, управление статусами
- 📍 **GPS-трекинг**: отслеживание местоположения в реальном времени
- 📄 **Верификация документов**: проверка водительских удостоверений и других документов
- 👨‍💼 **Система смен**: управление рабочими сменами водителей
- ⭐ **Рейтинги**: система оценок и отзывов

## Архитектура

Сервис построен с использованием Domain Driven Design (DDD) и включает:

- **Domain Layer**: бизнес-логика и сущности
- **Repository Layer**: работа с данными
- **Service Layer**: доменные сервисы
- **Interface Layer**: HTTP/gRPC API, WebSocket
- **Infrastructure Layer**: внешние зависимости

## Технологический стек

- **Язык**: Go 1.21
- **База данных**: PostgreSQL 15
- **Кэш**: Redis
- **Message Broker**: NATS
- **Мониторинг**: Prometheus + Grafana
- **Контейнеризация**: Docker + Kubernetes

## Быстрый старт

### Предварительные требования

- Go 1.21+
- Docker и Docker Compose
- PostgreSQL 15+ (опционально для локальной разработки)

### Запуск с Docker Compose

```bash
# Клонируем репозиторий
git clone <repository-url>
cd driver-service

# Запускаем все сервисы
docker-compose -f deployments/docker/docker-compose.yml up -d

# Проверяем статус сервисов
docker-compose -f deployments/docker/docker-compose.yml ps

# Проверяем health check
curl http://localhost:8001/health
```

### Локальная разработка

```bash
# Устанавливаем зависимости
go mod download

# Настраиваем переменные окружения
export DRIVER_SERVICE_DATABASE_HOST=localhost
export DRIVER_SERVICE_DATABASE_USER=driver_service
export DRIVER_SERVICE_DATABASE_PASSWORD=password
export DRIVER_SERVICE_DATABASE_DATABASE=driver_service

# Запускаем миграции
make migrate-up

# Запускаем сервис
go run cmd/server/main.go
```

## API Документация

### REST API

Базовый URL: `http://localhost:8001/api/v1`

#### Водители

```bash
# Создание водителя
POST /drivers
{
  "phone": "+79001234567",
  "email": "driver@example.com",
  "first_name": "Иван",
  "last_name": "Иванов",
  "license_number": "1234567890",
  "license_expiry": "2025-12-31T00:00:00Z"
  // ... другие поля
}

# Получение водителя
GET /drivers/{id}

# Список водителей
GET /drivers?limit=20&offset=0&status=available

# Обновление водителя
PUT /drivers/{id}

# Изменение статуса
PATCH /drivers/{id}/status
{
  "status": "available"
}

# Удаление водителя
DELETE /drivers/{id}
```

#### Местоположения

```bash
# Обновление местоположения
POST /drivers/{id}/locations
{
  "latitude": 55.7558,
  "longitude": 37.6173,
  "speed": 60.5,
  "accuracy": 10.0
}

# Пакетное обновление
POST /drivers/{id}/locations/batch
{
  "locations": [
    {
      "latitude": 55.7558,
      "longitude": 37.6173,
      "timestamp": 1640995200
    }
  ]
}

# Текущее местоположение
GET /drivers/{id}/locations/current

# История местоположений
GET /drivers/{id}/locations/history?from=1640995200&to=1641081600

# Водители поблизости
GET /locations/nearby?latitude=55.7558&longitude=37.6173&radius_km=5
```

### Коды статусов водителей

- `registered` - Зарегистрирован
- `pending_verification` - Ожидает верификации
- `verified` - Верифицирован
- `available` - Доступен
- `on_shift` - На смене
- `busy` - Занят (выполняет заказ)
- `inactive` - Неактивен
- `suspended` - Приостановлен
- `blocked` - Заблокирован

## Конфигурация

### Переменные окружения

```bash
# Сервер
DRIVER_SERVICE_SERVER_HTTP_PORT=8001
DRIVER_SERVICE_SERVER_GRPC_PORT=9001
DRIVER_SERVICE_SERVER_METRICS_PORT=9002
DRIVER_SERVICE_SERVER_ENVIRONMENT=development

# База данных
DRIVER_SERVICE_DATABASE_HOST=localhost
DRIVER_SERVICE_DATABASE_PORT=5432
DRIVER_SERVICE_DATABASE_USER=driver_service
DRIVER_SERVICE_DATABASE_PASSWORD=password
DRIVER_SERVICE_DATABASE_DATABASE=driver_service

# Redis
DRIVER_SERVICE_REDIS_HOST=localhost
DRIVER_SERVICE_REDIS_PORT=6379

# NATS
DRIVER_SERVICE_NATS_URL=nats://localhost:4222

# Логирование
DRIVER_SERVICE_LOGGER_LEVEL=info
DRIVER_SERVICE_LOGGER_FORMAT=json
```

### Конфигурационный файл

Можно использовать YAML файл конфигурации:

```yaml
# config.yaml
server:
  http_port: 8001
  grpc_port: 9001
  environment: development

database:
  host: localhost
  port: 5432
  user: driver_service
  password: password
  database: driver_service

logger:
  level: info
  format: json
```

## База данных

### Миграции

```bash
# Применить миграции
make migrate-up

# Откатить последнюю миграцию
make migrate-down

# Создать новую миграцию
make migrate-create NAME=add_new_field
```

### Структура таблиц

- `drivers` - Основная информация о водителях
- `driver_documents` - Документы водителей
- `driver_locations` - GPS координаты
- `driver_shifts` - Рабочие смены
- `driver_ratings` - Оценки и отзывы
- `driver_rating_stats` - Статистика рейтингов

## События NATS

### Исходящие события

```go
// Регистрация водителя
"driver.registered" {
  "driver_id": "uuid",
  "phone": "+79001234567",
  "name": "Иван Иванов"
}

// Изменение статуса
"driver.status.changed" {
  "driver_id": "uuid",
  "old_status": "registered",
  "new_status": "available"
}

// Обновление местоположения
"driver.location.updated" {
  "driver_id": "uuid",
  "location": {
    "latitude": 55.7558,
    "longitude": 37.6173
  },
  "speed": 60.5
}
```

### Входящие события

```go
// Назначение заказа
"order.assigned" {
  "order_id": "uuid",
  "driver_id": "uuid",
  "pickup_location": {...}
}

// Завершение заказа
"order.completed" {
  "order_id": "uuid",
  "driver_id": "uuid",
  "rating": 5
}
```

## Мониторинг

### Prometheus метрики

- `drivers_registered_total` - Количество зарегистрированных водителей
- `location_updates_total` - Обновления GPS
- `active_shifts_current` - Активные смены
- `http_requests_total` - HTTP запросы
- `http_request_duration_seconds` - Длительность запросов

### Health Checks

```bash
# Проверка состояния сервиса
curl http://localhost:8001/health

# Prometheus метрики
curl http://localhost:9002/metrics
```

### Grafana Dashboard

Дашборды доступны по адресу: http://localhost:3000
- Логин: `admin`
- Пароль: `admin`

## Развертывание

### Kubernetes

```bash
# Применить все манифесты
kubectl apply -f deployments/k8s/

# Проверить статус подов
kubectl get pods -l app=driver-service

# Посмотреть логи
kubectl logs -f deployment/driver-service

# Port forwarding для локального доступа
kubectl port-forward svc/driver-service 8001:8001
```

### Docker

```bash
# Сборка образа
docker build -f deployments/docker/Dockerfile -t driver-service:latest .

# Запуск контейнера
docker run -p 8001:8001 -e DATABASE_HOST=host.docker.internal driver-service:latest
```

## Разработка

### Структура проекта

```
driver-service/
├── cmd/server/           # Точка входа приложения
├── internal/
│   ├── config/          # Конфигурация
│   ├── domain/          # Доменная логика
│   │   ├── entities/    # Сущности
│   │   └── services/    # Доменные сервисы
│   ├── infrastructure/  # Инфраструктура
│   │   └── database/    # БД и миграции
│   ├── interfaces/      # Интерфейсы
│   │   └── http/        # HTTP API
│   └── repositories/    # Репозитории
├── api/                 # API спецификации
├── deployments/         # Развертывание
└── docs/               # Документация
```

### Тестирование

```bash
# Запуск всех тестов
go test ./...

# Тестирование с покрытием
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Интеграционные тесты
go test -tags=integration ./...
```

### Линтинг

```bash
# Запуск golangci-lint
golangci-lint run

# Форматирование кода
gofmt -s -w .
goimports -w .
```

## Производительность

### Рекомендуемые настройки

- **CPU**: 0.5-1 core на реплику
- **Memory**: 256-512 MB на реплику
- **Database**: Connection pool 25-50 соединений
- **Redis**: 10-20 соединений в пуле

### Масштабирование

- Горизонтальное масштабирование через HPA
- Автоскейлинг по CPU/Memory метрикам
- Партиционирование таблицы `driver_locations` по дате

## Безопасность

- JWT токены для аутентификации
- RBAC для авторизации
- SSL/TLS для всех соединений
- Шифрование персональных данных
- Audit logging всех действий

## FAQ

### Q: Как добавить новое поле в сущность водителя?

A: Создайте миграцию, обновите entity структуру, добавьте поле в repository и API handlers.

### Q: Как настроить алерты в Prometheus?

A: Настройте правила алертинга в конфигурации Prometheus и подключите AlertManager.

### Q: Как масштабировать сервис под высокой нагрузкой?

A: Используйте HPA для автоскейлинга, настройте партиционирование БД, оптимизируйте индексы.

## Поддержка

- 📧 Email: support@example.com
- 💬 Slack: #driver-service
- 📖 Wiki: [внутренняя документация]
- 🐛 Issues: [система багтрекинга]

## Лицензия

Этот проект является собственностью компании и предназначен только для внутреннего использования.