# Интеграционные тесты Driver Service

Этот каталог содержит интеграционные тесты для Driver Service, которые проверяют работу всех компонентов системы в связке.

## Структура тестов

```
tests/
├── integration/           # Интеграционные тесты
│   ├── driver_api_test.go        # Тесты HTTP API для водителей
│   ├── location_api_test.go      # Тесты HTTP API для местоположений
│   ├── driver_repository_test.go # Тесты репозитория водителей
│   ├── location_repository_test.go # Тесты репозитория местоположений
│   ├── document_repository_test.go # Тесты репозитория документов
│   ├── service_integration_test.go # Тесты интеграции сервисов
│   ├── performance_test.go       # Тесты производительности
│   └── e2e_test.go              # End-to-end тесты
├── helpers/              # Вспомогательные функции
│   ├── test_helpers.go          # Основные хелперы
│   ├── api_helpers.go           # Хелперы для API тестов
│   └── performance_helpers.go   # Хелперы для performance тестов
├── fixtures/             # Тестовые данные
│   └── driver_fixtures.go       # Фикстуры для водителей
└── README.md            # Этот файл
```

## Типы тестов

### 🔧 **Unit Tests**
Тестируют отдельные компоненты в изоляции:
```bash
go test ./internal/...
```

### 🔗 **Integration Tests**
Тестируют взаимодействие компонентов с реальной БД:
```bash
go test -tags=integration ./tests/integration/...
```

### ⚡ **Performance Tests**
Тестируют производительность под нагрузкой:
```bash
go test -tags=integration -run="Performance" ./tests/integration/...
```

### 🎯 **End-to-End Tests**
Тестируют полные пользовательские сценарии:
```bash
go test -tags=integration -run="E2E" ./tests/integration/...
```

## Настройка тестовой среды

### Автоматическая настройка
```bash
# Запуск всех тестов с автоматической настройкой БД
./scripts/run-tests.sh
```

### Ручная настройка

1. **Запуск тестовой БД:**
```bash
docker-compose -f docker-compose.test.yml up -d
```

2. **Установка переменных окружения:**
```bash
export TEST_DB_HOST=localhost
export TEST_DB_PORT=5433
export TEST_DB_USER=test_user
export TEST_DB_PASSWORD=test_password
```

3. **Запуск тестов:**
```bash
go test -tags=integration ./tests/integration/...
```

## Что тестируется

### 📊 **Database Layer**
- CRUD операции для всех сущностей
- Транзакции и откаты
- Constraint'ы и валидация
- Индексы и производительность запросов
- Каскадные удаления

### 🔄 **Service Layer**  
- Бизнес-логика водителей
- Переходы статусов
- Валидация для получения заказов
- GPS-трекинг и аналитика
- Event publishing

### 🌐 **HTTP API**
- REST endpoints
- Валидация запросов
- Обработка ошибок
- Пагинация и фильтрация
- CORS и middleware

### 🏃‍♂️ **Performance**
- Пропускная способность
- Время отклика
- Потребление памяти
- Конкурентные операции
- Нагрузочное тестирование

### 🎬 **End-to-End Scenarios**
- Полный жизненный цикл водителя
- Workflow отслеживания местоположения
- Сценарии с несколькими водителями
- Обработка ошибок

## Тестовые данные

### Фикстуры
Используются предопределенные тестовые данные:
- `CreateTestDriver()` - создает тестового водителя
- `CreateTestLocation()` - создает тестовое местоположение
- `CreateTestDocument()` - создает тестовый документ
- `CreateFullTestDataSet()` - создает полный набор связанных данных

### Тестовая БД
Каждый тест использует изолированную тестовую базу данных:
- Автоматическое создание уникальной БД для каждого теста
- Выполнение миграций
- Автоматическая очистка после тестов

## Метрики тестирования

### Покрытие кода
- **Минимальное покрытие**: 70%
- **Цель**: 85%+
- **Отчет**: `coverage.html`

### Производительность
- **Создание водителя**: < 100ms
- **Обновление местоположения**: < 50ms
- **Поиск поблизости**: < 200ms
- **Пакетные операции**: > 100 ops/sec

### Надежность
- **Максимальная частота ошибок**: 1%
- **Время отклика 95-го процентиля**: < 500ms
- **Успешность транзакций**: > 99%

## Запуск тестов

### Все тесты
```bash
make test-all
```

### Только интеграционные
```bash
make test-integration
```

### Только производительность
```bash
make test-performance
```

### С покрытием
```bash
make test-coverage
```

### В CI/CD
```bash
# GitHub Actions автоматически запускает все тесты
# при push в main/develop ветки
```

## Отладка тестов

### Логирование
Тесты используют структурированное логирование:
```go
logger := helpers.CreateTestLogger(t)
// Логи выводятся только при падении тестов
```

### Изоляция тестов
Каждый тест работает с чистой БД:
```go
func (suite *TestSuite) SetupTest() {
    suite.testDB.CleanupTables(suite.T())
}
```

### Отладка конкретного теста
```bash
# Запуск одного теста с подробным выводом
go test -v -tags=integration -run="TestSpecificTest" ./tests/integration/

# С отладочными логами
go test -v -tags=integration -run="TestSpecificTest" ./tests/integration/ -args -test.v
```

## Continuous Integration

### GitHub Actions
- Автоматический запуск при PR
- Проверка покрытия кода
- Сборка Docker образа
- Security scanning

### Локальная проверка перед commit
```bash
# Быстрые тесты
make check

# Полная проверка
make ci
```

## Troubleshooting

### Проблемы с БД
```bash
# Проверка подключения к тестовой БД
docker-compose -f docker-compose.test.yml exec test-postgres psql -U test_user -d driver_service_test -c "SELECT version();"

# Просмотр логов БД
docker-compose -f docker-compose.test.yml logs test-postgres
```

### Проблемы с тестами
```bash
# Очистка тестового окружения
docker-compose -f docker-compose.test.yml down -v

# Пересборка тестовых образов
docker-compose -f docker-compose.test.yml build --no-cache
```

### Медленные тесты
```bash
# Пропуск performance тестов
go test -short -tags=integration ./tests/integration/

# Профилирование тестов
go test -cpuprofile=cpu.prof -memprofile=mem.prof -tags=integration ./tests/integration/
```

## Расширение тестов

### Добавление нового теста
1. Создайте файл `*_test.go` в соответствующей директории
2. Используйте build tag `//go:build integration`
3. Наследуйтесь от `suite.Suite` для интеграционных тестов
4. Используйте хелперы из `tests/helpers/`

### Добавление новых фикстур
1. Добавьте функции в `tests/fixtures/`
2. Следуйте паттерну `CreateTest*`
3. Используйте реалистичные данные

### Добавление performance тестов
1. Используйте `PerformanceTestHelper`
2. Устанавливайте четкие пороги производительности
3. Логируйте подробную статистику

## Best Practices

1. **Изоляция**: каждый тест должен быть независимым
2. **Очистка**: всегда очищайте данные после тестов
3. **Реалистичность**: используйте реальные сценарии
4. **Производительность**: тесты должны выполняться быстро
5. **Покрытие**: стремитесь к высокому покрытию кода
6. **Документация**: документируйте сложные тестовые сценарии
