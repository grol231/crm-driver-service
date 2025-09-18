//go:build integration

package helpers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"driver-service/internal/config"
	"driver-service/internal/infrastructure/database"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestDB структура для тестовой базы данных
type TestDB struct {
	*database.DB
	dbName string
	logger *zap.Logger
}

// SetupTestDB создает тестовую базу данных
func SetupTestDB(t *testing.T) *TestDB {
	logger := zaptest.NewLogger(t)

	// Получаем конфигурацию для тестов
	cfg := getTestConfig()

	// Создаем уникальное имя для тестовой БД
	testDBName := fmt.Sprintf("test_%s_%d", t.Name(), time.Now().Unix())
	testDBName = sanitizeDBName(testDBName)

	// Подключаемся к основной БД для создания тестовой
	mainDB, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode,
	))
	if err != nil {
		t.Fatalf("Failed to connect to main database: %v", err)
	}
	defer mainDB.Close()

	// Создаем тестовую базу данных
	_, err = mainDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Подключаемся к тестовой БД
	cfg.Database.Database = testDBName
	testDB, err := database.NewPostgresDB(&cfg.Database, logger)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Выполняем миграции
	if err := runTestMigrations(testDB.DB.DB, logger); err != nil {
		t.Fatalf("Failed to run test migrations: %v", err)
	}

	logger.Info("Test database created",
		zap.String("db_name", testDBName),
	)

	return &TestDB{
		DB:     testDB,
		dbName: testDBName,
		logger: logger,
	}
}

// TeardownTestDB удаляет тестовую базу данных
func (tdb *TestDB) TeardownTestDB(t *testing.T) {
	if tdb == nil {
		return
	}

	// Закрываем соединение с тестовой БД
	tdb.DB.Close()

	// Получаем конфигурацию
	cfg := getTestConfig()

	// Подключаемся к основной БД для удаления тестовой
	mainDB, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode,
	))
	if err != nil {
		t.Errorf("Failed to connect to main database for cleanup: %v", err)
		return
	}
	defer mainDB.Close()

	// Закрываем все соединения к тестовой БД
	_, err = mainDB.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = '%s' AND pid <> pg_backend_pid()
	`, tdb.dbName))
	if err != nil {
		t.Logf("Warning: Failed to terminate connections to test database: %v", err)
	}

	// Удаляем тестовую базу данных
	_, err = mainDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", tdb.dbName))
	if err != nil {
		t.Errorf("Failed to drop test database: %v", err)
	} else {
		tdb.logger.Info("Test database dropped", zap.String("db_name", tdb.dbName))
	}
}

// CleanupTables очищает все таблицы в тестовой БД
func (tdb *TestDB) CleanupTables(t *testing.T) {
	tables := []string{
		"driver_ratings",
		"driver_rating_stats",
		"driver_locations",
		"driver_shifts",
		"driver_documents",
		"drivers",
	}

	for _, table := range tables {
		_, err := tdb.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Errorf("Failed to truncate table %s: %v", table, err)
		}
	}
}

// getTestConfig возвращает конфигурацию для тестов
func getTestConfig() *config.Config {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:            getEnvOrDefault("TEST_DB_HOST", "localhost"),
			Port:            5432,
			User:            getEnvOrDefault("TEST_DB_USER", "postgres"),
			Password:        getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
			Database:        "postgres", // Будет заменено на тестовую БД
			SSLMode:         "disable",
			MaxOpenConns:    5,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Server: config.ServerConfig{
			HTTPPort:    8001,
			GRPCPort:    9001,
			MetricsPort: 9002,
			Timeout:     30 * time.Second,
			Environment: "test",
		},
		Logger: config.LoggerConfig{
			Level:      "debug",
			Format:     "console",
			OutputPath: "stdout",
		},
	}

	return cfg
}

// runTestMigrations выполняет миграции для тестовой БД
func runTestMigrations(db *sql.DB, logger *zap.Logger) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	migrationsPath := "file://internal/infrastructure/database/migrations"
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Test migrations completed")
	return nil
}

// getEnvOrDefault получает переменную окружения или возвращает значение по умолчанию
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// sanitizeDBName очищает имя базы данных от недопустимых символов
func sanitizeDBName(name string) string {
	// Заменяем недопустимые символы на подчеркивания
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result += string(r)
		} else {
			result += "_"
		}
	}

	// Ограничиваем длину до 63 символов (лимит PostgreSQL)
	if len(result) > 63 {
		result = result[:63]
	}

	// Убираем подчеркивания в начале и конце
	for len(result) > 0 && result[0] == '_' {
		result = result[1:]
	}
	for len(result) > 0 && result[len(result)-1] == '_' {
		result = result[:len(result)-1]
	}

	// Если имя пустое, используем дефолтное
	if result == "" {
		result = "test_db"
	}

	return result
}

// WaitForDB ждет доступности базы данных
func WaitForDB(db *database.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("database not ready within %v", timeout)
		case <-ticker.C:
			if err := db.Health(); err == nil {
				return nil
			}
		}
	}
}

// CreateTestLogger создает логгер для тестов
func CreateTestLogger(t *testing.T) *zap.Logger {
	return zaptest.NewLogger(t, zaptest.Level(zap.DebugLevel))
}
