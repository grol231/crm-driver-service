# Развертывание Driver Service в Portainer

Это руководство описывает развертывание Driver Service в Portainer.

## Подготовка

### 1. Сборка Docker образа

Сначала необходимо собрать и загрузить Docker образ в registry:

```bash
# Локальная сборка
cd driver-service
docker build -f deployments/docker/Dockerfile -t taxi-crm/driver-service:latest .

# Или push в Docker Hub (если настроен)
docker tag taxi-crm/driver-service:latest your-registry/driver-service:latest
docker push your-registry/driver-service:latest
```

### 2. Обновление конфигурации

Если используете внешний registry, обновите image в файле `docker-compose.portainer.yml`:

```yaml
services:
  driver-service:
    image: your-registry/driver-service:latest  # Замените на ваш registry
```

## Развертывание в Portainer

### Метод 1: Через веб-интерфейс Portainer

1. **Откройте Portainer** (обычно http://localhost:9000)

2. **Перейдите в Stacks** → **Add stack**

3. **Выберите способ загрузки:**
   - **Web editor**: скопируйте содержимое `docker-compose.portainer.yml`
   - **Upload**: загрузите файл `docker-compose.portainer.yml`
   - **Repository**: укажите Git репозиторий с файлом

4. **Настройте переменные окружения** (опционально):
   ```
   DRIVER_SERVICE_DATABASE_PASSWORD=your_secure_password
   DRIVER_SERVICE_SERVER_ENVIRONMENT=production
   ```

5. **Нажмите "Deploy the stack"**

### Метод 2: Через Docker CLI

```bash
# Если у вас есть доступ к Docker CLI на сервере
docker stack deploy -c deployments/docker/docker-compose.portainer.yml driver-service
```

## Проверка развертывания

### 1. Проверка статуса сервисов

В Portainer:
- Перейдите в **Stacks** → **driver-service**
- Проверьте статус всех сервисов
- Убедитесь, что все контейнеры в состоянии **Running**

### 2. Проверка health checks

```bash
# Health check основного сервиса
curl http://your-server:8001/health

# Проверка PostgreSQL
docker exec driver-service_postgres_1 pg_isready -U driver_service

# Проверка Redis
docker exec driver-service_redis_1 redis-cli ping
```

### 3. Тестирование API

```bash
# Создание водителя
curl -X POST http://your-server:8001/api/v1/drivers \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+79001234567",
    "email": "driver@example.com",
    "first_name": "Тест",
    "last_name": "Водитель",
    "birth_date": "1985-05-15T00:00:00Z",
    "passport_series": "1234",
    "passport_number": "567890",
    "license_number": "TEST123456",
    "license_expiry": "2026-12-31T00:00:00Z"
  }'

# Список водителей
curl http://your-server:8001/api/v1/drivers
```

## Конфигурация для production

### Переменные окружения

Рекомендуется использовать Docker secrets или переменные окружения:

```yaml
environment:
  - DRIVER_SERVICE_DATABASE_PASSWORD_FILE=/run/secrets/db_password
  - DRIVER_SERVICE_SERVER_ENVIRONMENT=production
  - DRIVER_SERVICE_LOGGER_LEVEL=warn
  - DRIVER_SERVICE_DATABASE_SSL_MODE=require
```

### Secrets (для production)

```yaml
secrets:
  db_password:
    external: true
  redis_password:
    external: true

services:
  driver-service:
    secrets:
      - db_password
      - redis_password
```

### Volumes для persistence

```yaml
volumes:
  postgres_data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /opt/driver-service/postgres
  
  redis_data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /opt/driver-service/redis
```

## Мониторинг

### Prometheus

Доступен по адресу: `http://your-server:9090`

Основные метрики:
- HTTP requests
- Database connections
- Application health

### Grafana

Доступен по адресу: `http://your-server:3000`
- Логин: `admin`
- Пароль: `admin`

## Troubleshooting

### Проблема: "config not found: prometheus_config"

**Решение:** Используйте `docker-compose.swarm-simple.yml` вместо `docker-compose.swarm.yml`

```bash
# Используйте упрощенную версию
docker stack deploy -c deployments/docker/docker-compose.swarm-simple.yml driver-service
```

### Проблема: "Since --detach=false was not specified"

**Решение:** Это предупреждение, не ошибка. Для подавления:

```bash
# Добавьте флаг --detach
docker stack deploy --detach -c docker-compose.swarm-simple.yml driver-service
```

### Проблема: Сервис не запускается

1. **Проверьте логи в Portainer:**
   - Stacks → driver-service → Logs

2. **Проверьте образ:**
   ```bash
   docker pull registry.starline.ru/crm-driver-service:latest
   ```

3. **Проверьте сеть:**
   ```bash
   docker network ls
   docker network inspect driver-service_driver-service-network
   ```

### Проблема: База данных недоступна

1. **Проверьте PostgreSQL:**
   ```bash
   docker logs driver-service_postgres_1
   ```

2. **Проверьте подключение:**
   ```bash
   docker exec driver-service_postgres_1 psql -U driver_service -d driver_service -c "SELECT version();"
   ```

### Проблема: Порты заняты

Измените порты в `docker-compose.portainer.yml`:

```yaml
ports:
  - "8002:8001"  # Вместо 8001
  - "5435:5432"  # Вместо 5434
  - "6382:6379"  # Вместо 6381
```

## Обновление сервиса

### 1. Обновление образа

```bash
# Сборка нового образа
docker build -t taxi-crm/driver-service:v1.1.0 .
docker push taxi-crm/driver-service:v1.1.0
```

### 2. Обновление в Portainer

1. Перейдите в **Stacks** → **driver-service**
2. Нажмите **Editor**
3. Измените версию образа:
   ```yaml
   image: taxi-crm/driver-service:v1.1.0
   ```
4. Нажмите **Update the stack**

### 3. Rolling update (для Swarm)

```bash
docker service update --image taxi-crm/driver-service:v1.1.0 driver-service_driver-service
```

## Backup и восстановление

### База данных

```bash
# Backup
docker exec driver-service_postgres_1 pg_dump -U driver_service driver_service > backup.sql

# Restore
docker exec -i driver-service_postgres_1 psql -U driver_service driver_service < backup.sql
```

### Redis данные

```bash
# Backup
docker exec driver-service_redis_1 redis-cli BGSAVE
docker cp driver-service_redis_1:/data/dump.rdb ./redis-backup.rdb

# Restore
docker cp ./redis-backup.rdb driver-service_redis_1:/data/dump.rdb
docker restart driver-service_redis_1
```

## Масштабирование

### Горизонтальное масштабирование

В Portainer:
1. Перейдите в **Services** → **driver-service_driver-service**
2. Нажмите **Scale**
3. Увеличьте количество реплик

Или через CLI:
```bash
docker service scale driver-service_driver-service=5
```

### Ресурсы

Настройте ресурсы в зависимости от нагрузки:

```yaml
deploy:
  resources:
    limits:
      memory: 1G      # Увеличьте для высокой нагрузки
      cpus: '1.0'
    reservations:
      memory: 512M
      cpus: '0.5'
```

## Security

### 1. Настройка сетей

```yaml
networks:
  driver-service-network:
    driver: overlay
    encrypted: true      # Шифрование трафика
    attachable: false    # Только для сервисов стека
```

### 2. Secrets

```yaml
secrets:
  db_password:
    external: true
  jwt_secret:
    external: true

services:
  driver-service:
    secrets:
      - source: db_password
        target: /run/secrets/db_password
      - source: jwt_secret
        target: /run/secrets/jwt_secret
```

### 3. Ограничения доступа

```yaml
deploy:
  placement:
    constraints:
      - node.labels.security == high
      - node.role == worker
```

## Мониторинг в production

### Настройка алертов

1. **Создайте alerting rules для Prometheus**
2. **Настройте Grafana дашборды**
3. **Интегрируйте с внешними системами мониторинга**

### Логи

```bash
# Просмотр логов через Portainer или CLI
docker service logs driver-service_driver-service
```

## Примечания

1. **Образ должен быть доступен** в registry, к которому имеет доступ Portainer
2. **Порты должны быть свободны** на целевой машине
3. **Volumes создаются автоматически** при первом запуске
4. **Для production** рекомендуется использовать внешние volumes и secrets
