package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Logger middleware для логирования HTTP запросов
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: func(param gin.LogFormatterParams) string {
			logger.Info("HTTP Request",
				zap.String("client_ip", param.ClientIP),
				zap.String("method", param.Method),
				zap.String("path", param.Path),
				zap.Int("status_code", param.StatusCode),
				zap.Duration("latency", param.Latency),
				zap.String("user_agent", param.Request.UserAgent()),
				zap.Int("body_size", param.BodySize),
				zap.String("request_id", param.Request.Header.Get("X-Request-ID")),
			)
			return ""
		},
		Output: nil,
	})
}

// RequestID middleware для добавления уникального ID к каждому запросу
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// CORS middleware для обработки CORS запросов
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// В production среде здесь должны быть проверки разрешенных доменов
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RateLimit middleware для ограничения частоты запросов
func RateLimit() gin.HandlerFunc {
	// В реальном приложении здесь должна быть реализация rate limiting
	// с использованием Redis или in-memory cache
	return func(c *gin.Context) {
		c.Next()
	}
}

// Timeout middleware для установки таймаута запроса
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		
		// Устанавливаем таймаут для контекста
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		
		// Заменяем контекст запроса
		c.Request = c.Request.WithContext(timeoutCtx)
		
		c.Next()
	}
}

// Metrics middleware для сбора метрик
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		method := c.Request.Method
		
		c.Next()
		
		duration := time.Since(start)
		status := c.Writer.Status()
		
		// Здесь должна быть отправка метрик в Prometheus
		// Например:
		// httpRequestDuration.WithLabelValues(method, path, fmt.Sprintf("%d", status)).Observe(duration.Seconds())
		// httpRequestsTotal.WithLabelValues(method, path, fmt.Sprintf("%d", status)).Inc()
		
		_ = duration
		_ = status
		_ = path
		_ = method
	}
}

// Auth middleware для аутентификации
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// В реальном приложении здесь должна быть проверка JWT токена
		// или другой механизм аутентификации
		
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}
		
		// Проверка токена...
		// В данном примере пропускаем все запросы
		c.Set("user_id", "authenticated_user")
		c.Next()
	}
}

// Recovery middleware для обработки паник
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("Panic recovered",
			zap.Any("panic", recovered),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.String("request_id", c.GetString("request_id")),
		)
		
		c.JSON(500, gin.H{
			"error": "Internal server error",
			"code":  "PANIC_RECOVERED",
		})
	})
}