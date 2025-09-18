package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"driver-service/internal/config"
	"driver-service/internal/domain/services"
	httpHandlers "driver-service/internal/interfaces/http/handlers"
	httpServer "driver-service/internal/interfaces/http"
	"driver-service/internal/infrastructure/database"
	"driver-service/internal/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Application основная структура приложения
type Application struct {
	config   *config.Config
	logger   *zap.Logger
	db       *database.DB
	
	// Repositories
	driverRepo   repositories.DriverRepository
	documentRepo repositories.DocumentRepository
	locationRepo repositories.LocationRepository
	
	// Services
	driverService   services.DriverService
	locationService services.LocationService
	
	// Servers
	httpServer *httpServer.Server
	
	// Shutdown
	shutdown chan struct{}
	wg       sync.WaitGroup
}

func main() {
	app, err := NewApplication()
	if err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		app.logger.Fatal("Application failed", zap.Error(err))
	}

	app.logger.Info("Application stopped gracefully")
}

// NewApplication создает новый экземпляр приложения
func NewApplication() (*Application, error) {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Валидируем конфигурацию
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Инициализируем логгер
	logger, err := initLogger(cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger.Info("Starting Driver Service",
		zap.String("version", "1.0.0"),
		zap.String("environment", cfg.Server.Environment),
		zap.Int("http_port", cfg.Server.HTTPPort),
	)

	// Инициализируем базу данных
	db, err := database.NewPostgresDB(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Выполняем миграции
	migrationsPath := filepath.Join("internal", "infrastructure", "database", "migrations")
	if err := db.RunMigrations(migrationsPath); err != nil {
		logger.Error("Failed to run migrations", zap.Error(err))
		// Не прерываем выполнение, так как миграции могут быть уже выполнены
	}

	app := &Application{
		config:   cfg,
		logger:   logger,
		db:       db,
		shutdown: make(chan struct{}),
	}

	// Инициализируем компоненты
	if err := app.initRepositories(); err != nil {
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	if err := app.initServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	if err := app.initServers(); err != nil {
		return nil, fmt.Errorf("failed to initialize servers: %w", err)
	}

	return app, nil
}

// initLogger инициализирует логгер
func initLogger(cfg config.LoggerConfig) (*zap.Logger, error) {
	var zapConfig zap.Config

	if cfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
	}

	// Устанавливаем уровень логирования
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	zapConfig.Level.SetLevel(level)

	// Устанавливаем путь вывода
	if cfg.OutputPath != "" && cfg.OutputPath != "stdout" {
		zapConfig.OutputPaths = []string{cfg.OutputPath}
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}

// initRepositories инициализирует репозитории
func (app *Application) initRepositories() error {
	app.driverRepo = repositories.NewDriverRepository(app.db, app.logger)
	app.documentRepo = repositories.NewDocumentRepository(app.db, app.logger)
	app.locationRepo = repositories.NewLocationRepository(app.db, app.logger)

	app.logger.Info("Repositories initialized")
	return nil
}

// initServices инициализирует сервисы
func (app *Application) initServices() error {
	// Создаем заглушку для EventPublisher
	eventBus := &mockEventPublisher{logger: app.logger}

	app.driverService = services.NewDriverService(
		app.driverRepo,
		app.documentRepo,
		eventBus,
		app.logger,
	)

	app.locationService = services.NewLocationService(
		app.locationRepo,
		app.driverRepo,
		eventBus,
		app.logger,
	)

	app.logger.Info("Services initialized")
	return nil
}

// initServers инициализирует серверы
func (app *Application) initServers() error {
	// HTTP handlers
	driverHandler := httpHandlers.NewDriverHandler(app.driverService, app.logger)
	locationHandler := httpHandlers.NewLocationHandler(app.locationService, app.logger)

	// HTTP server
	app.httpServer = httpServer.NewServer(
		app.config,
		app.logger,
		driverHandler,
		locationHandler,
	)

	app.logger.Info("Servers initialized")
	return nil
}

// Run запускает приложение
func (app *Application) Run() error {
	// Запускаем background задачи
	app.wg.Add(1)
	go app.runBackgroundTasks()

	// Запускаем HTTP сервер
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		if err := app.httpServer.Start(); err != nil {
			app.logger.Error("HTTP server failed", zap.Error(err))
		}
	}()

	// Ждем сигнал для завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		app.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-app.shutdown:
		app.logger.Info("Received shutdown from internal source")
	}

	// Graceful shutdown
	return app.gracefulShutdown()
}

// runBackgroundTasks запускает фоновые задачи
func (app *Application) runBackgroundTasks() {
	defer app.wg.Done()

	// Cleanup старых местоположений
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-cleanupTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			if err := app.locationService.CleanupOldLocations(ctx); err != nil {
				app.logger.Error("Failed to cleanup old locations", zap.Error(err))
			}
			cancel()

		case <-app.shutdown:
			app.logger.Info("Stopping background tasks")
			return
		}
	}
}

// gracefulShutdown выполняет graceful shutdown
func (app *Application) gracefulShutdown() error {
	app.logger.Info("Starting graceful shutdown")

	// Создаем контекст с таймаутом для shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Закрываем канал для уведомления background задач
	close(app.shutdown)

	// Останавливаем HTTP сервер
	if err := app.httpServer.Stop(ctx); err != nil {
		app.logger.Error("Failed to stop HTTP server", zap.Error(err))
	}

	// Ждем завершения background задач
	done := make(chan struct{})
	go func() {
		app.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		app.logger.Info("All goroutines stopped")
	case <-ctx.Done():
		app.logger.Error("Shutdown timeout exceeded")
	}

	// Закрываем подключение к базе данных
	if err := app.db.Close(); err != nil {
		app.logger.Error("Failed to close database connection", zap.Error(err))
	}

	app.logger.Info("Graceful shutdown completed")
	return nil
}

// mockEventPublisher заглушка для EventPublisher
// В реальном приложении здесь должна быть реализация с NATS
type mockEventPublisher struct {
	logger *zap.Logger
}

func (m *mockEventPublisher) PublishDriverEvent(ctx context.Context, eventType string, driverID uuid.UUID, data interface{}) error {
	m.logger.Info("Publishing driver event",
		zap.String("event_type", eventType),
		zap.String("driver_id", driverID.String()),
		zap.Any("data", data),
	)
	return nil
}