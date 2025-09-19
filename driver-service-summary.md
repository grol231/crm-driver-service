# Driver Service - Краткое резюме

## Основные характеристики

| Параметр | Значение |
|----------|----------|
| **Порт** | 8001 |
| **Язык** | Go (Golang) |
| **База данных** | PostgreSQL |
| **Message Broker** | NATS |
| **API** | REST + gRPC + WebSocket |
| **Архитектура** | DDD (Domain Driven Design) |

## Ключевые функции

### 🚗 Управление водителями
- Регистрация и профили водителей
- Верификация документов через ГИБДД API
- Управление статусами (доступен, занят, оффлайн)
- Система блокировок и разрешений

### 📍 GPS-трекинг
- Отслеживание местоположения в реальном времени
- Мониторинг скорости и стиля вождения  
- Геозоны и контроль маршрутов
- WebSocket для live-трекинга

### 👨‍💼 Система смен
- Автоматическое начало/окончание смен
- Привязка к транспортным средствам
- Учет рабочего времени и статистики
- Расчет заработка за смену

### ⭐ Рейтинги и отзывы
- Агрегация оценок от пассажиров
- Многокритериальная система оценки
- Расчет средних рейтингов
- Система поощрений/штрафов

## База данных PostgreSQL

### Основные таблицы
- `drivers` - Профили водителей
- `driver_documents` - Документы и верификация
- `driver_locations` - GPS координаты 
- `driver_shifts` - Рабочие смены
- `driver_ratings` - Оценки и отзывы

### Производительность
- Индексы для быстрого поиска
- Партиционирование по времени для locations
- Пространственные индексы для геозапросов
- Оптимизация под высокие нагрузки

## REST API Endpoints

```
├── /api/v1/drivers
│   ├── POST   /                     # Регистрация
│   ├── GET    /                     # Список водителей
│   ├── GET    /{id}                 # Профиль водителя
│   ├── PUT    /{id}                 # Обновление профиля
│   └── PATCH  /{id}/status          # Изменение статуса
├── /api/v1/drivers/{id}/documents
│   ├── POST   /                     # Загрузка документа
│   ├── GET    /                     # Список документов
│   └── POST   /{doc_id}/verify      # Верификация
├── /api/v1/drivers/{id}/locations
│   ├── POST   /                     # Отправка координат
│   ├── GET    /                     # История локаций
│   └── GET    /current              # Текущее местоположение
├── /api/v1/drivers/{id}/shifts
│   ├── POST   /start                # Начало смены
│   ├── POST   /end                  # Окончание смены
│   └── GET    /                     # История смен
└── /api/v1/drivers/{id}/ratings
    ├── POST   /                     # Добавление оценки
    ├── GET    /                     # Список оценок
    └── GET    /stats                # Статистика рейтинга
```

## NATS События

### Исходящие события
```go
"driver.registered"        // Регистрация водителя
"driver.verified"          // Верификация документов
"driver.status.changed"    // Изменение статуса
"driver.shift.started"     // Начало смены
"driver.shift.ended"       // Окончание смены
"driver.location.updated"  // Обновление GPS
"driver.rating.updated"    // Изменение рейтинга
"driver.performance.alert" // Предупреждения
```

### Входящие события
```go
"order.assigned"           // Назначен заказ
"order.completed"          // Заказ завершен
"order.cancelled"          // Заказ отменен
"payment.processed"        // Обработан платеж
"vehicle.assigned"         // Назначен автомобиль
"customer.rated.driver"    // Оценка от клиента
```

## Интеграции с другими сервисами

### Core Services
- **Order Service** - Управление заказами и назначениями
- **Payment Service** - Расчеты и выплаты
- **Customer Service** - Обратная связь от клиентов
- **Notification Service** - Уведомления водителям
- **Analytics Service** - Аналитика и отчетность

### Supporting Services
- **Fleet Service** - Управление автопарком
- **Auth Service** - Аутентификация и авторизация
- **Compliance Service** - Соответствие требованиям

### External APIs
- **ГИБДД API** - Проверка документов
- **Yandex Maps** - Геокодирование
- **SMS Gateway** - Отправка уведомлений

## Структура проекта

```
driver-service/
├── cmd/server/           # Точка входа приложения
├── internal/
│   ├── domain/          # Бизнес-логика и сущности
│   ├── infrastructure/  # Внешние зависимости
│   ├── interfaces/      # HTTP, gRPC, WebSocket
│   └── repositories/    # Работа с данными
├── api/                 # API спецификации
├── pkg/                 # Общие пакеты
└── deployments/         # Docker, Kubernetes
```

## Основные Go пакеты

```go
// Domain Entities
type Driver struct {
    ID            uuid.UUID
    Phone         string
    Email         string
    FirstName     string
    LastName      string
    LicenseNumber string
    Status        Status
    CurrentRating float64
    TotalTrips    int
    // ...
}

// Services
type DriverService interface {
    CreateDriver(ctx context.Context, driver *Driver) (*Driver, error)
    GetDriverByID(ctx context.Context, id uuid.UUID) (*Driver, error)
    UpdateDriverStatus(ctx context.Context, id uuid.UUID, status Status) error
    // ...
}

// Event Types
type DriverEvent struct {
    EventType string
    DriverID  string
    Timestamp time.Time
    Data      interface{}
}
```

## Мониторинг и метрики

### Prometheus метрики
- `drivers_registered_total` - Количество зарегистрированных водителей
- `location_updates_total` - Обновления GPS координат
- `active_shifts_current` - Активные смены
- `driver_average_rating` - Средние рейтинги

### Health Checks
- Database connectivity
- NATS connectivity  
- Redis cache status
- External APIs status

## Развертывание

### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
# Build process...
FROM alpine:latest
EXPOSE 8001 9001
CMD ["./driver-service"]
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: driver-service
spec:
  replicas: 3
  # Configuration...
```

## Производительность и масштабирование

### Характеристики нагрузки
- **GPS Updates**: 10,000+ в секунду
- **API Requests**: 5,000+ RPS
- **Database**: Оптимизировано для 100,000+ водителей
- **Events**: 50,000+ событий в минуту

### Оптимизации
- Connection pooling для БД
- Redis кэширование частых запросов
- Batch processing для GPS данных
- Асинхронная обработка событий

## Безопасность

### Аутентификация
- JWT токены для API доступа
- OAuth 2.0 интеграция
- Role-based access control (RBAC)

### Защита данных
- Шифрование персональных данных
- SSL/TLS для всех соединений
- Audit logging действий

Driver Service является центральным компонентом системы CRM для таксопарков, обеспечивая полное управление жизненным циклом водителей от регистрации до аналитики производительности.

