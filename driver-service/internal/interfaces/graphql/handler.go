package graphql

import (
	"context"
	"net/http"

	"driver-service/internal/interfaces/graphql/resolver"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler создает GraphQL handler
func NewHandler(resolver *resolver.Resolver, logger *zap.Logger) http.Handler {
	// Временно возвращаем простой handler до генерации кода
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"message":"GraphQL endpoint ready, but schema generation is pending"}}`))
	})
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