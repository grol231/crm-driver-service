package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"driver-service/internal/config"
	"driver-service/internal/interfaces/http/handlers"
	"driver-service/internal/interfaces/http/middleware"
	gqlhandler "driver-service/internal/interfaces/graphql"
	"driver-service/internal/interfaces/graphql/resolver"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server HTTP сервер
type Server struct {
	config     *config.Config
	logger     *zap.Logger
	httpServer *http.Server
	router     *gin.Engine
}

// NewServer создает новый HTTP сервер
func NewServer(
	cfg *config.Config,
	logger *zap.Logger,
	driverHandler *handlers.DriverHandler,
	locationHandler *handlers.LocationHandler,
	graphqlResolver *resolver.Resolver,
) *Server {
	// Настройка Gin
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	
	// Middleware
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "driver-service",
			"timestamp": time.Now().UTC(),
		})
	})

	// API routes
	api := router.Group("/api/v1")
	
	// Driver routes
	drivers := api.Group("/drivers")
	{
		drivers.POST("", driverHandler.CreateDriver)
		drivers.GET("", driverHandler.ListDrivers)
		drivers.GET("/active", driverHandler.GetActiveDrivers)
		drivers.GET("/:id", driverHandler.GetDriver)
		drivers.PUT("/:id", driverHandler.UpdateDriver)
		drivers.DELETE("/:id", driverHandler.DeleteDriver)
		drivers.PATCH("/:id/status", driverHandler.ChangeStatus)
		
		// Location routes for specific driver
		drivers.POST("/:id/locations", locationHandler.UpdateLocation)
		drivers.POST("/:id/locations/batch", locationHandler.BatchUpdateLocations)
		drivers.GET("/:id/locations/current", locationHandler.GetCurrentLocation)
		drivers.GET("/:id/locations/history", locationHandler.GetLocationHistory)
	}

	// Location routes
	locations := api.Group("/locations")
	{
		locations.GET("/nearby", locationHandler.GetNearbyDrivers)
	}

	// GraphQL routes
	if graphqlResolver != nil {
		router.POST("/graphql", gqlhandler.GraphQLHandler(graphqlResolver, logger))
		router.GET("/graphql", gqlhandler.GraphQLHandler(graphqlResolver, logger)) // для GET запросов
		
		// GraphQL Playground (только для development)
		if cfg.Server.Environment != "production" {
			router.GET("/playground", gqlhandler.PlaygroundHandler())
		}
	}

	server := &Server{
		config: cfg,
		logger: logger,
		router: router,
		httpServer: &http.Server{
			Addr:           fmt.Sprintf(":%d", cfg.Server.HTTPPort),
			Handler:        router,
			ReadTimeout:    cfg.Server.Timeout,
			WriteTimeout:   cfg.Server.Timeout,
			IdleTimeout:    2 * cfg.Server.Timeout,
			MaxHeaderBytes: 1 << 20, // 1 MB
		},
	}

	return server
}

// Start запускает HTTP сервер
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server",
		zap.Int("port", s.config.Server.HTTPPort),
		zap.String("environment", s.config.Server.Environment),
	)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	return nil
}

// Stop останавливает HTTP сервер
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	
	return s.httpServer.Shutdown(ctx)
}

// GetRouter возвращает router для тестирования
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}