# Исправление Production проблем

## 🚨 **Проблемы и решения**

### 1. "Connection refused" к PostgreSQL

**Причина:** PostgreSQL не готов или недоступен

**Решение:**
```bash
# Проверьте статус PostgreSQL
docker service ps driver-service_postgres

# Проверьте логи PostgreSQL
docker service logs driver-service_postgres

# Перезапустите PostgreSQL
docker service update --force driver-service_postgres
```

### 2. "Dirty database version 3"

**Причина:** Миграции в "грязном" состоянии после неудачного выполнения

**Решение:**
```bash
# 1. Остановите driver-service
docker service scale driver-service_driver-service=0

# 2. Исправьте миграции
./scripts/fix-migrations.sh

# 3. Запустите driver-service
docker service scale driver-service_driver-service=1
```

### 3. "bind source path does not exist: prometheus.yml"

**Причина:** Внешний файл конфигурации недоступен в Swarm

**Решение:** Используйте `docker-compose.production-minimal.yml` без внешних файлов

### 4. Несколько реплик пытаются подключиться одновременно

**Причина:** Swarm запускает все реплики одновременно

**Решение:** Используйте `docker-compose.production.yml` с:
- `replicas: 1` - только одна реплика
- `order: start-first` - последовательное обновление
- Улучшенные health checks

## 🔧 **Пошаговое исправление**

### Шаг 1: Остановите текущий стек
```bash
docker stack rm driver-service
```

### Шаг 2: Очистите volumes (если нужно)
```bash
# ⚠️ ВНИМАНИЕ: Это удалит все данные!
docker volume prune -f
```

### Шаг 3: Исправьте миграции (если БД существует)
```bash
# Установите переменные окружения
export DB_HOST=your-postgres-host
export DB_PORT=5434
export DB_USER=driver_service
export DB_PASSWORD=driver_service_password
export DB_NAME=driver_service

# Запустите скрипт исправления
./scripts/fix-migrations.sh
```

### Шаг 4: Разверните production конфигурацию
```bash
# Если есть проблемы с внешними файлами, используйте минимальную версию:
docker stack deploy --detach -c deployments/docker/docker-compose.production-minimal.yml driver-service

# Или полную версию (если все файлы доступны):
docker stack deploy --detach -c deployments/docker/docker-compose.production.yml driver-service
```

### Шаг 5: Проверьте статус
```bash
# Проверьте сервисы
docker stack services driver-service

# Проверьте логи
docker service logs driver-service_driver-service
docker service logs driver-service_postgres
```

## 📋 **Production конфигурация**

### Основные изменения в `docker-compose.production.yml`:

1. **Одна реплика driver-service** - избегаем конфликтов миграций
2. **Улучшенные health checks** - более длительные таймауты
3. **Больше ресурсов** - для стабильной работы
4. **Последовательное обновление** - `order: start-first`
5. **Улучшенные restart policies** - более стабильные перезапуски

### Порты:
- **8001** - HTTP API
- **9001** - gRPC API
- **9002** - Metrics
- **5434** - PostgreSQL
- **6381** - Redis
- **14222** - NATS Client
- **18222** - NATS HTTP

## 🔍 **Диагностика проблем**

### Проверка подключения к БД
```bash
# Проверьте доступность PostgreSQL
docker exec -it $(docker ps -q -f name=driver-service_postgres) pg_isready -U driver_service

# Проверьте подключение из driver-service
docker exec -it $(docker ps -q -f name=driver-service_driver-service) curl -f http://postgres:5432
```

### Проверка миграций
```bash
# Подключитесь к БД
docker exec -it $(docker ps -q -f name=driver-service_postgres) psql -U driver_service -d driver_service

# Проверьте состояние миграций
SELECT * FROM schema_migrations ORDER BY version DESC;

# Если dirty = true, исправьте:
UPDATE schema_migrations SET dirty = false WHERE version = (SELECT MAX(version) FROM schema_migrations);
```

### Проверка логов
```bash
# Логи driver-service
docker service logs -f driver-service_driver-service

# Логи PostgreSQL
docker service logs -f driver-service_postgres

# Логи Redis
docker service logs -f driver-service_redis
```

## ⚠️ **Важные замечания**

1. **Всегда используйте production конфигурацию** для production
2. **Проверяйте health checks** перед масштабированием
3. **Делайте backup БД** перед изменениями
4. **Мониторьте логи** во время развертывания
5. **Используйте rolling updates** для обновлений

## 🚀 **Команды для быстрого исправления**

```bash
# Полное исправление
docker stack rm driver-service
sleep 10
docker stack deploy --detach -c deployments/docker/docker-compose.production.yml driver-service

# Проверка
docker stack services driver-service
curl http://your-server:8001/health
```

## 📊 **Мониторинг**

### Health checks
```bash
# Driver service
curl http://your-server:8001/health

# PostgreSQL
docker exec $(docker ps -q -f name=driver-service_postgres) pg_isready -U driver_service

# Redis
docker exec $(docker ps -q -f name=driver-service_redis) redis-cli ping

# NATS
curl http://your-server:18222/varz
```

### Логи
```bash
# Все сервисы
docker stack services driver-service

# Конкретный сервис
docker service logs driver-service_driver-service
```

## 🎯 **Рекомендации для production**

1. **Используйте внешние volumes** для данных
2. **Настройте backup** для PostgreSQL
3. **Мониторьте ресурсы** (CPU, память)
4. **Настройте алерты** для критических ошибок
5. **Используйте load balancer** для масштабирования
