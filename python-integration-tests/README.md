# Driver Service - Python Integration Tests

Комплексные интеграционные тесты для Driver Service, написанные на Python с использованием pytest.

## Обзор

Этот набор тестов покрывает:

- **HTTP REST API** - все endpoints Driver и Location API
- **WebSocket соединения** - real-time tracking и уведомления
- **NATS события** - публикация и потребление событий
- **gRPC API** - тестирование gRPC интерфейсов (если реализован)
- **Performance тесты** - нагрузочное тестирование с Locust

## Структура проекта

```
python-integration-tests/
├── tests/                          # Тесты
│   ├── test_driver_api.py          # Тесты Driver HTTP API
│   ├── test_location_api.py        # Тесты Location HTTP API
│   ├── test_websocket.py           # Тесты WebSocket соединений
│   ├── test_nats_events.py         # Тесты NATS событий
│   └── test_grpc_api.py            # Тесты gRPC API
├── utils/                          # Утилиты
│   ├── __init__.py
│   ├── logger.py                   # Настройка логирования
│   └── helpers.py                  # Вспомогательные функции
├── performance/                    # Performance тесты
│   ├── __init__.py
│   └── locustfile.py              # Locust конфигурация
├── config.py                      # Конфигурация тестов
├── conftest.py                    # Pytest fixtures
├── pytest.ini                    # Pytest конфигурация
├── requirements.txt              # Python зависимости
├── docker-compose.integration-tests.yml  # Docker Compose
├── Dockerfile.tests              # Docker образ для тестов
├── Dockerfile.performance        # Docker образ для performance тестов
├── Makefile                     # Команды автоматизации
└── README.md                    # Документация
```

## Быстрый старт

### Локальный запуск

1. **Установка зависимостей:**
```bash
make setup
```

2. **Настройка конфигурации:**
```bash
cp .env.example .env
# Отредактируйте .env файл под ваше окружение
```

3. **Запуск всех тестов:**
```bash
make test
```

### Docker запуск

1. **Запуск всех тестов в Docker:**
```bash
make test-docker
```

2. **Запуск отдельных наборов тестов:**
```bash
make docker-smoke      # Smoke тесты
make docker-api        # API тесты
make docker-websocket  # WebSocket тесты
make docker-nats       # NATS тесты
make docker-grpc       # gRPC тесты
```

## Доступные команды

### Основные команды

```bash
make help              # Показать справку
make setup             # Настроить окружение
make test              # Запустить все тесты локально
make test-docker       # Запустить тесты в Docker
make clean             # Очистить артефакты
```

### Отдельные наборы тестов

```bash
make smoke             # Быстрые smoke тесты
make api               # HTTP API тесты
make websocket         # WebSocket тесты
make nats              # NATS событий тесты
make grpc              # gRPC тесты
make performance       # Performance тесты
```

### Docker операции

```bash
make docker-build      # Собрать Docker образы
make docker-up         # Запустить сервисы
make docker-down       # Остановить сервисы
make docker-logs       # Показать логи
```

### Разработка

```bash
make dev-setup         # Настроить dev окружение
make lint              # Проверить код
make format            # Форматировать код
make ci-test           # Тесты для CI/CD
```

## Конфигурация

### Переменные окружения

Основные переменные в `.env` файле:

```bash
# Сервис
SERVICE_HOST=localhost
SERVICE_HTTP_PORT=8001
SERVICE_GRPC_PORT=9001

# База данных
TEST_DB_HOST=localhost
TEST_DB_PORT=5433
TEST_DB_USER=test_user
TEST_DB_PASSWORD=test_password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6380

# NATS
NATS_URL=nats://localhost:4222

# Настройки тестов
TEST_TIMEOUT=30
CLEANUP_AFTER_TEST=true
LOG_LEVEL=INFO
```

### Pytest маркеры

```bash
@pytest.mark.smoke       # Быстрые smoke тесты
@pytest.mark.api          # API тесты
@pytest.mark.websocket    # WebSocket тесты
@pytest.mark.nats         # NATS тесты
@pytest.mark.grpc         # gRPC тесты
@pytest.mark.performance  # Performance тесты
@pytest.mark.integration  # Интеграционные тесты
```

## Тестовые сценарии

### HTTP API тесты

- **Driver API:**
  - Создание, получение, обновление, удаление водителей
  - Изменение статуса водителя
  - Получение списка водителей с фильтрами и пагинацией
  - Получение активных водителей
  - Валидация входных данных
  - Обработка ошибок

- **Location API:**
  - Обновление местоположения водителя
  - Пакетное обновление местоположений
  - Получение текущего местоположения
  - Получение истории местоположений
  - Поиск водителей поблизости
  - Валидация координат

### WebSocket тесты

- Подключение к WebSocket endpoints
- Отправка и получение сообщений
- Обработка множественных соединений
- Восстановление соединения
- Валидация формата сообщений
- Тестирование больших сообщений

### NATS тесты

- **Исходящие события:**
  - driver.registered
  - driver.status.changed
  - driver.location.updated
  - driver.shift.started/ended

- **Входящие события:**
  - order.assigned
  - order.completed
  - customer.rated.driver
  - payment.processed

### Performance тесты

- **Нагрузочные сценарии:**
  - Создание и обновление водителей
  - Частые обновления местоположения
  - Поиск водителей поблизости
  - Множественные одновременные запросы

- **Пользовательские сценарии:**
  - Обычный пользователь (смешанная нагрузка)
  - Высоконагруженный пользователь
  - Только чтение данных
  - Процесс регистрации водителя

## Отчеты

После выполнения тестов генерируются отчеты:

- **HTML отчет:** `test-results/report.html`
- **JUnit XML:** `test-results/junit.xml`
- **Coverage:** `test-results/htmlcov/index.html`
- **Performance:** `test-results/performance-report.html`

## CI/CD интеграция

### GitHub Actions пример

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Integration Tests
        run: |
          cd python-integration-tests
          make ci-docker
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: python-integration-tests/test-results/
```

### Jenkins пример

```groovy
pipeline {
    agent any
    stages {
        stage('Integration Tests') {
            steps {
                dir('python-integration-tests') {
                    sh 'make ci-docker'
                }
            }
            post {
                always {
                    publishHTML([
                        allowMissing: false,
                        alwaysLinkToLastBuild: true,
                        keepAll: true,
                        reportDir: 'python-integration-tests/test-results',
                        reportFiles: 'report.html',
                        reportName: 'Integration Test Report'
                    ])
                    junit 'python-integration-tests/test-results/junit.xml'
                }
            }
        }
    }
}
```

## Troubleshooting

### Распространенные проблемы

1. **Сервис недоступен:**
```bash
make wait-for-services  # Ожидание готовности сервисов
make test-health        # Проверка health endpoint
```

2. **Docker проблемы:**
```bash
make docker-logs        # Просмотр логов
make clean-docker       # Очистка Docker ресурсов
make docker-build       # Пересборка образов
```

3. **База данных проблемы:**
```bash
make db-reset          # Сброс тестовой БД
```

4. **Зависимости:**
```bash
make check-deps        # Проверка версий
make update-deps       # Обновление зависимостей
```

### Отладка

Для детальной отладки:

```bash
# Запуск одного теста с подробным выводом
python -m pytest tests/test_driver_api.py::TestDriverAPI::test_create_driver_success -v -s

# Запуск с отладчиком
python -m pytest tests/test_driver_api.py --pdb

# Повышенный уровень логирования
LOG_LEVEL=DEBUG make test
```

### Производительность

Для оптимизации производительности тестов:

```bash
# Параллельный запуск
python -m pytest tests/ -n auto

# Запуск только быстрых тестов
python -m pytest tests/ -m "not slow"

# Остановка на первой ошибке
python -m pytest tests/ -x
```

## Разработка

### Добавление новых тестов

1. Создайте файл теста в директории `tests/`
2. Используйте fixtures из `conftest.py`
3. Добавьте соответствующие маркеры
4. Обновите документацию

### Создание новых fixtures

```python
@pytest.fixture
def my_fixture(http_client):
    # Setup
    data = create_test_data()
    yield data
    # Teardown
    cleanup_test_data(data)
```

### Добавление утилит

Добавьте функции в `utils/helpers.py` для повторного использования.

## Лицензия

MIT License - см. файл LICENSE в корне проекта.

## Поддержка

Для вопросов и проблем создавайте issues в репозитории проекта.