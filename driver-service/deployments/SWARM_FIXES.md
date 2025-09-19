# Исправление ошибок Docker Swarm

## 🚨 **Ошибки и решения**

### 1. "config not found: prometheus_config"

**Проблема:** Swarm не может найти внешний config `prometheus_config`

**Решение:** Используйте `docker-compose.swarm-simple.yml`

```bash
# ❌ Неправильно (с configs)
docker stack deploy -c docker-compose.swarm.yml driver-service

# ✅ Правильно (без configs)
docker stack deploy -c docker-compose.swarm-simple.yml driver-service
```

### 2. "Since --detach=false was not specified"

**Проблема:** Предупреждение о том, что задачи создаются в фоне

**Решение:** Добавьте флаг `--detach`

```bash
# ✅ С флагом --detach
docker stack deploy --detach -c docker-compose.swarm-simple.yml driver-service
```

### 3. "invalid reference format"

**Проблема:** Неправильный формат ссылки на образ

**Решение:** Проверьте имя образа

```yaml
# ❌ Неправильно
image: taxi-crm/driver-service:latest

# ✅ Правильно (с registry)
image: registry.starline.ru/crm-driver-service:latest
```

## 📋 **Доступные конфигурации**

### Для Portainer (рекомендуется)
```bash
# Простая версия без внешних зависимостей
docker-compose.simple.yml
```

### Для Docker Swarm
```bash
# Полная версия (может требовать configs)
docker-compose.swarm.yml

# Упрощенная версия (рекомендуется)
docker-compose.swarm-simple.yml
```

### Для локальной разработки
```bash
# Полная версия с build
docker-compose.yml
```

## 🔧 **Быстрое исправление**

### Шаг 1: Остановите текущий стек
```bash
docker stack rm driver-service
```

### Шаг 2: Используйте упрощенную версию
```bash
docker stack deploy --detach -c deployments/docker/docker-compose.swarm-simple.yml driver-service
```

### Шаг 3: Проверьте статус
```bash
docker stack services driver-service
docker service logs driver-service_driver-service
```

## 📝 **Основные отличия версий**

| Файл | Configs | Build | Restart Policy | Использование |
|------|---------|-------|----------------|---------------|
| `docker-compose.yml` | ❌ | ✅ | `restart: unless-stopped` | Локальная разработка |
| `docker-compose.simple.yml` | ❌ | ❌ | ❌ | Portainer |
| `docker-compose.portainer.yml` | ❌ | ❌ | ❌ | Portainer с health checks |
| `docker-compose.swarm.yml` | ✅ | ❌ | `deploy.restart_policy` | Docker Swarm |
| `docker-compose.swarm-simple.yml` | ❌ | ❌ | `deploy.restart_policy` | Docker Swarm (рекомендуется) |

## 🚀 **Команды для развертывания**

### Portainer
```bash
# Используйте содержимое docker-compose.simple.yml
# В Portainer: Stacks → Add stack → Web editor
```

### Docker Swarm
```bash
# Упрощенная версия (рекомендуется)
docker stack deploy --detach -c deployments/docker/docker-compose.swarm-simple.yml driver-service

# Или через Portainer с docker-compose.swarm-simple.yml
```

### Локальная разработка
```bash
# Полная версия с build
docker-compose -f deployments/docker/docker-compose.yml up -d
```

## 🔍 **Диагностика проблем**

### Проверка стека
```bash
docker stack ls
docker stack services driver-service
docker stack ps driver-service
```

### Проверка сервисов
```bash
docker service ls
docker service ps driver-service_driver-service
docker service logs driver-service_driver-service
```

### Проверка сети
```bash
docker network ls
docker network inspect driver-service_driver-service-network
```

### Проверка volumes
```bash
docker volume ls
docker volume inspect driver-service_postgres_data
```

## ⚠️ **Важные замечания**

1. **Образ должен быть доступен** в registry
2. **Порты должны быть свободны** на всех нодах
3. **Volumes создаются автоматически** при первом запуске
4. **Для production** используйте внешние volumes и secrets
5. **PostgreSQL размещается только на manager нодах** (constraint: `node.role == manager`)

## 🎯 **Рекомендации**

- **Для Portainer**: используйте `docker-compose.simple.yml`
- **Для Swarm**: используйте `docker-compose.swarm-simple.yml`
- **Для разработки**: используйте `docker-compose.yml`
- **При ошибках configs**: переходите на упрощенные версии
