# Быстрый старт Driver Service

## Для Portainer

### Шаг 1: Подготовка образа

```bash
# Если у вас есть доступ к Docker на сервере
docker build -f deployments/docker/Dockerfile -t taxi-crm/driver-service:latest .
```

### Шаг 2: Развертывание в Portainer

1. Откройте Portainer
2. Перейдите в **Stacks** → **Add stack**
3. Выберите **Web editor**
4. Скопируйте содержимое файла `deployments/docker/docker-compose.simple.yml`
5. Нажмите **Deploy the stack**

### Шаг 3: Проверка

```bash
curl http://your-server:8001/health
```

## Для локальной разработки

### Быстрый запуск

```bash
cd driver-service

# Запуск всех сервисов
docker-compose -f deployments/docker/docker-compose.yml up -d

# Проверка
curl http://localhost:8001/health
```

### Создание тестового водителя

```bash
curl -X POST http://localhost:8001/api/v1/drivers \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+79001234567",
    "email": "test@example.com",
    "first_name": "Тест",
    "last_name": "Водитель",
    "birth_date": "1985-05-15T00:00:00Z",
    "passport_series": "1234",
    "passport_number": "567890",
    "license_number": "TEST123456",
    "license_expiry": "2026-12-31T00:00:00Z"
  }'
```

### Обновление местоположения

```bash
# Замените {DRIVER_ID} на ID созданного водителя
curl -X POST "http://localhost:8001/api/v1/drivers/{DRIVER_ID}/locations" \
  -H "Content-Type: application/json" \
  -d '{
    "latitude": 55.7558,
    "longitude": 37.6173,
    "speed": 60.5
  }'
```

## Доступные файлы конфигурации

- `docker-compose.yml` - Полная версия для разработки
- `docker-compose.simple.yml` - Упрощенная версия для Portainer
- `docker-compose.portainer.yml` - Версия с deploy конфигурацией
- `docker-compose.swarm.yml` - Версия для Docker Swarm

## Порты

- **8001** - HTTP API
- **9001** - gRPC API (будет реализован)
- **9002** - Prometheus метрики
- **5434** - PostgreSQL
- **6381** - Redis
- **4222** - NATS
- **9090** - Prometheus
- **3000** - Grafana

## Переменные окружения

Основные переменные для настройки:

```bash
DRIVER_SERVICE_DATABASE_HOST=postgres
DRIVER_SERVICE_DATABASE_USER=driver_service
DRIVER_SERVICE_DATABASE_PASSWORD=driver_service_password
DRIVER_SERVICE_DATABASE_DATABASE=driver_service
DRIVER_SERVICE_SERVER_ENVIRONMENT=production
DRIVER_SERVICE_LOGGER_LEVEL=info
```

## Troubleshooting

### Ошибка "build not supported"
Используйте `docker-compose.simple.yml` или `docker-compose.portainer.yml`

### Ошибка "container_name not supported"  
Используйте версии без `container_name`

### Ошибка "restart not supported"
Используйте версии с `deploy.restart_policy`

### Порты заняты
Измените порты в конфигурации на свободные
