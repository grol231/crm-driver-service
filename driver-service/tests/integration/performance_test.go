//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"driver-service/internal/domain/services"
	"driver-service/internal/repositories"
	"driver-service/tests/fixtures"
	"driver-service/tests/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PerformanceTestSuite тестовый suite для проверки производительности
type PerformanceTestSuite struct {
	suite.Suite
	testDB          *helpers.TestDB
	driverService   services.DriverService
	locationService services.LocationService
	perfHelper      *helpers.PerformanceTestHelper
	ctx             context.Context
}

// SetupSuite выполняется один раз перед всеми тестами
func (suite *PerformanceTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDB(suite.T())
	logger := helpers.CreateTestLogger(suite.T())
	suite.ctx = context.Background()

	// Инициализируем репозитории
	driverRepo := repositories.NewDriverRepository(suite.testDB.DB, logger)
	documentRepo := repositories.NewDocumentRepository(suite.testDB.DB, logger)
	locationRepo := repositories.NewLocationRepository(suite.testDB.DB, logger)

	// Создаем mock EventPublisher
	eventBus := &mockEventPublisher{logger: logger}

	// Инициализируем сервисы
	suite.driverService = services.NewDriverService(driverRepo, documentRepo, eventBus, logger)
	suite.locationService = services.NewLocationService(locationRepo, driverRepo, eventBus, logger)

	// Создаем helper для тестирования производительности
	suite.perfHelper = helpers.NewPerformanceTestHelper(suite.T(), suite.driverService, suite.locationService)
}

// TearDownSuite выполняется один раз после всех тестов
func (suite *PerformanceTestSuite) TearDownSuite() {
	suite.testDB.TeardownTestDB(suite.T())
}

// SetupTest выполняется перед каждым тестом
func (suite *PerformanceTestSuite) SetupTest() {
	suite.testDB.CleanupTables(suite.T())
}

// TestDriverCreationPerformance тестирует производительность создания водителей
func (suite *PerformanceTestSuite) TestDriverCreationPerformance() {
	// Тест с разными уровнями concurrency
	testCases := []struct {
		name         string
		count        int
		concurrency  int
		maxAvgTime   time.Duration
		minOpsPerSec float64
	}{
		{
			name:         "Low load",
			count:        50,
			concurrency:  5,
			maxAvgTime:   100 * time.Millisecond,
			minOpsPerSec: 10,
		},
		{
			name:         "Medium load",
			count:        200,
			concurrency:  10,
			maxAvgTime:   200 * time.Millisecond,
			minOpsPerSec: 20,
		},
		{
			name:         "High load",
			count:        500,
			concurrency:  20,
			maxAvgTime:   500 * time.Millisecond,
			minOpsPerSec: 30,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := suite.perfHelper.BenchmarkDriverCreation(suite.ctx, tc.count, tc.concurrency)
			suite.perfHelper.AssertPerformanceThresholds(result, tc.maxAvgTime, tc.minOpsPerSec)
		})
	}
}

// TestLocationUpdatePerformance тестирует производительность обновления местоположений
func (suite *PerformanceTestSuite) TestLocationUpdatePerformance() {
	// Arrange - создаем тестового водителя
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Test cases
	testCases := []struct {
		name         string
		count        int
		concurrency  int
		maxAvgTime   time.Duration
		minOpsPerSec float64
	}{
		{
			name:         "Single location updates",
			count:        100,
			concurrency:  5,
			maxAvgTime:   50 * time.Millisecond,
			minOpsPerSec: 50,
		},
		{
			name:         "High frequency updates",
			count:        1000,
			concurrency:  15,
			maxAvgTime:   100 * time.Millisecond,
			minOpsPerSec: 100,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := suite.perfHelper.BenchmarkLocationUpdates(suite.ctx, createdDriver.ID, tc.count, tc.concurrency)
			suite.perfHelper.AssertPerformanceThresholds(result, tc.maxAvgTime, tc.minOpsPerSec)
		})
	}
}

// TestBatchLocationPerformance тестирует производительность пакетных обновлений
func (suite *PerformanceTestSuite) TestBatchLocationPerformance() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Test cases для разных размеров батчей
	testCases := []struct {
		name       string
		batchSize  int
		batchCount int
	}{
		{"Small batches", 10, 10},
		{"Medium batches", 50, 5},
		{"Large batches", 100, 3},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := suite.perfHelper.BenchmarkBatchLocationUpdates(suite.ctx, createdDriver.ID, tc.batchSize, tc.batchCount)

			// Пакетные операции должны быть эффективнее одиночных
			expectedMinOpsPerSec := float64(tc.batchSize * 2) // Минимум в 2 раза быстрее одиночных
			suite.perfHelper.AssertPerformanceThresholds(result, 1*time.Second, expectedMinOpsPerSec)
		})
	}
}

// TestNearbyDriversSearchPerformance тестирует производительность поиска водителей поблизости
func (suite *PerformanceTestSuite) TestNearbyDriversSearchPerformance() {
	// Test cases с разным количеством водителей
	testCases := []struct {
		name         string
		driversCount int
		searchCount  int
		maxAvgTime   time.Duration
		minOpsPerSec float64
	}{
		{
			name:         "Small dataset",
			driversCount: 100,
			searchCount:  50,
			maxAvgTime:   100 * time.Millisecond,
			minOpsPerSec: 20,
		},
		{
			name:         "Medium dataset",
			driversCount: 500,
			searchCount:  100,
			maxAvgTime:   200 * time.Millisecond,
			minOpsPerSec: 15,
		},
		{
			name:         "Large dataset",
			driversCount: 1000,
			searchCount:  50,
			maxAvgTime:   500 * time.Millisecond,
			minOpsPerSec: 10,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Очищаем таблицы перед каждым тестом
			suite.testDB.CleanupTables(t)

			result := suite.perfHelper.BenchmarkNearbyDriversSearch(suite.ctx, tc.searchCount, tc.driversCount)
			suite.perfHelper.AssertPerformanceThresholds(result, tc.maxAvgTime, tc.minOpsPerSec)
		})
	}
}

// TestMemoryUsage тестирует использование памяти
func (suite *PerformanceTestSuite) TestMemoryUsage() {
	// Тест создания большого количества объектов
	suite.perfHelper.MemoryUsageTest(suite.ctx, 1000)

	// Дополнительная проверка - память не должна расти бесконтрольно
	// (это проверяется визуально через логи)
}

// TestStressTest выполняет стресс-тестирование
func (suite *PerformanceTestSuite) TestStressTest() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	suite.perfHelper.StressTest(suite.ctx, createdDriver.ID)

	// Assert - основные проверки выполняются внутри StressTest
	// Здесь можно добавить дополнительные проверки состояния системы
}

// TestDatabaseConnectionPool тестирует пул соединений БД под нагрузкой
func (suite *PerformanceTestSuite) TestDatabaseConnectionPool() {
	suite.perfHelper.DatabaseConnectionPoolTest(suite.ctx, suite.testDB)
}

// TestLongRunningOperations тестирует долгосрочные операции
func (suite *PerformanceTestSuite) TestLongRunningOperations() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act - запускаем нагрузочный тест на 30 секунд
	result := suite.perfHelper.LoadTest(suite.ctx, createdDriver.ID, 30*time.Second, 10)

	// Assert
	suite.T().Logf("Long running test completed: %d operations in %v", result.OperationCount, result.TotalTime)

	// Проверяем, что система стабильна под продолжительной нагрузкой
	errorRate := float64(result.Errors) / float64(result.OperationCount)
	assert.Less(suite.T(), errorRate, 0.05, "Error rate should be less than 5%% in long running test")

	// Проверяем минимальную производительность
	assert.Greater(suite.T(), result.OpsPerSecond, 10.0, "Should maintain at least 10 ops/sec under load")
}

// TestResourceCleanup тестирует очистку ресурсов
func (suite *PerformanceTestSuite) TestResourceCleanup() {
	// Arrange - создаем много данных
	driversCount := 50
	locationsPerDriver := 20

	var driverIDs []uuid.UUID

	// Создаем водителей
	for i := 0; i < driversCount; i++ {
		driver := fixtures.CreateTestDriver()
		driver.Phone = fmt.Sprintf("+7900%07d", i)
		driver.Email = fmt.Sprintf("cleanup_test_%d@example.com", i)
		driver.LicenseNumber = fmt.Sprintf("CLEANUP%04d", i)

		createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
		require.NoError(suite.T(), err)
		driverIDs = append(driverIDs, createdDriver.ID)
	}

	// Создаем местоположения
	for _, driverID := range driverIDs {
		locations := fixtures.CreateTestLocationHistory(driverID, locationsPerDriver, 1*time.Minute)

		// Устанавливаем старые времена (старше 30 дней)
		baseTime := time.Now().Add(-35 * 24 * time.Hour)
		for i, location := range locations {
			location.RecordedAt = baseTime.Add(time.Duration(i) * time.Minute)
			location.CreatedAt = location.RecordedAt
		}

		err := suite.locationService.BatchUpdateLocations(suite.ctx, locations)
		require.NoError(suite.T(), err)
	}

	// Act - выполняем очистку
	start := time.Now()
	err := suite.locationService.CleanupOldLocations(suite.ctx)
	cleanupTime := time.Since(start)

	// Assert
	require.NoError(suite.T(), err)
	suite.T().Logf("Cleanup of %d locations took: %v", driversCount*locationsPerDriver, cleanupTime)

	// Проверяем, что очистка выполняется за разумное время
	suite.Assert().Less(cleanupTime, 30*time.Second, "Cleanup should complete within 30 seconds")

	// Проверяем, что данные действительно удалены
	for _, driverID := range driverIDs {
		history, err := suite.locationService.GetLocationHistory(suite.ctx, driverID,
			time.Now().Add(-40*24*time.Hour), time.Now())
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), history, 0, "Old locations should be cleaned up")
	}
}

// Запуск тестового suite
func TestPerformanceTestSuite(t *testing.T) {
	// Пропускаем performance тесты в быстром режиме
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	suite.Run(t, new(PerformanceTestSuite))
}
