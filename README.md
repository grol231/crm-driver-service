# CRM Driver Service

Центральный сервис для управления водителями в микросервисной архитектуре CRM системы для таксопарков.

## Описание

Driver Service обеспечивает полный жизненный цикл управления водителями, включая:
- Регистрацию и верификацию водителей
- Управление документами и их проверку
- GPS-трекинг в реальном времени
- Систему рейтингов и отзывов
- Управление рабочими сменами

## Структура проекта

```
crm-driver-service/
└── driver-service/          # Основной сервис
    ├── cmd/server/          # Точка входа
    ├── internal/            # Внутренняя логика
    ├── deployments/         # Docker & Kubernetes
    ├── api/                 # API спецификации
    └── README.md            # Подробная документация
```

## Быстрый старт

```bash
cd driver-service

# Запуск с Docker Compose (рекомендуется)
docker-compose -f deployments/docker/docker-compose.yml up -d

# Проверка работы
curl http://localhost:8001/health
```

## Документация

Подробная документация находится в [driver-service/README.md](./driver-service/README.md)

## API Endpoints

- **HTTP API**: http://localhost:8001/api/v1
- **Metrics**: http://localhost:9002/metrics
- **Health Check**: http://localhost:8001/health

## Мониторинг

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

## Технологии

- Go 1.21
- PostgreSQL 15
- Redis 7
- NATS
- Docker & Kubernetes
- Prometheus & Grafana
