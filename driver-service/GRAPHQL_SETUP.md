# GraphQL API Setup Instructions

Этот документ содержит инструкции по настройке и запуску GraphQL API для Driver Service.

## Установка зависимостей

1. Убедитесь, что у вас установлены все зависимости:
```bash
cd driver-service
go mod download
```

2. Установите gqlgen глобально (опционально):
```bash
go install github.com/99designs/gqlgen@latest
```

## Генерация кода GraphQL

После создания или изменения GraphQL схемы необходимо сгенерировать код:

```bash
cd driver-service

# Генерация кода с помощью go run
go run github.com/99designs/gqlgen generate

# Или если gqlgen установлен глобально
gqlgen generate
```

Эта команда создаст следующие файлы:
- `internal/interfaces/graphql/generated/generated.go` - основной исполняемый код
- `internal/interfaces/graphql/generated/models_gen.go` - сгенерированные модели
- `internal/interfaces/graphql/generated/resolver.go` - интерфейсы resolvers

## Структура файлов

После генерации структура GraphQL пакета будет выглядеть так:

```
internal/interfaces/graphql/
├── generated/           # Сгенерированные файлы
│   ├── generated.go
│   ├── models_gen.go
│   └── resolver.go
├── model/              # Пользовательские модели
│   ├── models.go
│   └── converters.go
├── resolver/           # Resolvers
│   ├── resolver.go
│   ├── interfaces.go
│   ├── query.go
│   ├── mutation.go
│   ├── subscription.go
│   ├── field_resolvers.go
│   └── *_test.go
├── schema/             # GraphQL схема
│   └── schema.graphql
└── handler.go          # HTTP handler
```

## Обновление handler.go

После генерации кода обновите файл `handler.go`:

```go
package graphql

import (
	"context"
	"net/http"

	"driver-service/internal/interfaces/graphql/generated"
	"driver-service/internal/interfaces/graphql/resolver"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewHandler создает GraphQL handler
func NewHandler(resolver *resolver.Resolver, logger *zap.Logger) http.Handler {
	// Создаем исполнитель GraphQL
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Настраиваем транспорты
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10,
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// Добавляем кэш для интроспекции
	srv.SetQueryCache(lru.New(1000))

	// Добавляем интроспекцию
	srv.Use(extension.Introspection{})

	// Добавляем автоматическое сохранение запросов
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	// Добавляем middleware для логирования
	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)
		logger.Info("GraphQL operation",
			zap.String("operation", oc.OperationName),
			zap.String("query", oc.RawQuery),
		)
		return next(ctx)
	})

	return srv
}

// PlaygroundHandler создает GraphQL playground handler
func PlaygroundHandler() gin.HandlerFunc {
	h := playground.Handler("GraphQL playground", "/graphql")
	return gin.WrapH(h)
}

// GraphQLHandler создает Gin handler для GraphQL
func GraphQLHandler(resolver *resolver.Resolver, logger *zap.Logger) gin.HandlerFunc {
	h := NewHandler(resolver, logger)
	return gin.WrapH(h)
}
```

## Запуск сервера

1. Сгенерируйте код:
```bash
go run github.com/99designs/gqlgen generate
```

2. Убедитесь, что все зависимости на месте:
```bash
go mod tidy
```

3. Запустите сервер:
```bash
go run cmd/server/main.go
```

4. Откройте GraphQL Playground:
```
http://localhost:8001/playground
```

## Тестирование

Запустите тесты для проверки работоспособности:

```bash
# Unit тесты
go test ./internal/interfaces/graphql/resolver/...

# Интеграционные тесты
go test ./tests/integration/...

# Все тесты
go test ./...
```

## Отладка

Если возникают проблемы:

1. Проверьте, что схема валидна:
```bash
go run github.com/99designs/gqlgen validate
```

2. Проверьте конфигурацию gqlgen:
```bash
cat gqlgen.yml
```

3. Пересоздайте код:
```bash
rm -rf internal/interfaces/graphql/generated/
go run github.com/99designs/gqlgen generate
```

## Развертывание

Для production окружения:

1. Убедитесь, что `introspection: false` в конфигурации
2. Отключите playground для production
3. Настройте правильные лимиты сложности запросов
4. Добавьте аутентификацию и авторизацию

## Полезные команды

```bash
# Инициализация gqlgen (только для новых проектов)
go run github.com/99designs/gqlgen init

# Генерация кода
go run github.com/99designs/gqlgen generate

# Валидация схемы
go run github.com/99designs/gqlgen validate

# Интроспекция схемы
go run github.com/99designs/gqlgen introspect

# Обновление зависимостей gqlgen
go get -u github.com/99designs/gqlgen
go get -u github.com/vektah/gqlparser/v2
```

## Дополнительная настройка

### Настройка CORS для GraphQL

Убедитесь, что CORS настроен правильно в middleware:

```go
router.Use(middleware.CORS())
```

### Настройка WebSocket для подписок

Для работы подписок убедитесь, что WebSocket transport включен в handler.

### Мониторинг и метрики

Рассмотрите добавление метрик для мониторинга производительности GraphQL:

```go
srv.Use(extension.FixedComplexityLimit(1000))
srv.Use(extension.IntrospectionLimit(100))
```

## Документация

- [Основная документация по GraphQL API](GRAPHQL_API.md)
- [Схема GraphQL](internal/interfaces/graphql/schema/schema.graphql)
- [Примеры запросов и мутаций](GRAPHQL_API.md#примеры-использования)