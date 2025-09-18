//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/domain/services"
	"driver-service/internal/repositories"
	"driver-service/tests/fixtures"
	"driver-service/tests/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ServiceIntegrationTestSuite тестирует интеграцию между сервисами
type ServiceIntegrationTestSuite struct {
	suite.Suite
	testDB          *helpers.TestDB
	driverService   services.DriverService
	locationService services.LocationService
	driverRepo      repositories.DriverRepository
	locationRepo    repositories.LocationRepository
	documentRepo    repositories.DocumentRepository
	ctx             context.Context
}

// SetupSuite выполняется один раз перед всеми тестами
func (suite *ServiceIntegrationTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDB(suite.T())
	logger := helpers.CreateTestLogger(suite.T())
	suite.ctx = context.Background()

	// Инициализируем репозитории
	suite.driverRepo = repositories.NewDriverRepository(suite.testDB.DB, logger)
	suite.documentRepo = repositories.NewDocumentRepository(suite.testDB.DB, logger)
	suite.locationRepo = repositories.NewLocationRepository(suite.testDB.DB, logger)

	// Создаем mock EventPublisher
	eventBus := &mockEventPublisher{logger: logger}

	// Инициализируем сервисы
	suite.driverService = services.NewDriverService(suite.driverRepo, suite.documentRepo, eventBus, logger)
	suite.locationService = services.NewLocationService(suite.locationRepo, suite.driverRepo, eventBus, logger)
}

// TearDownSuite выполняется один раз после всех тестов
func (suite *ServiceIntegrationTestSuite) TearDownSuite() {
	suite.testDB.TeardownTestDB(suite.T())
}

// SetupTest выполняется перед каждым тестом
func (suite *ServiceIntegrationTestSuite) SetupTest() {
	suite.testDB.CleanupTables(suite.T())
}

// TestDriverLifecycle тестирует полный жизненный цикл водителя
func (suite *ServiceIntegrationTestSuite) TestDriverLifecycle() {
	// 1. Создание водителя
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.StatusRegistered, createdDriver.Status)

	// 2. Переход к верификации
	err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, entities.StatusPendingVerification)
	require.NoError(suite.T(), err)

	// 3. Добавление документов
	document := fixtures.CreateTestDocument(createdDriver.ID, entities.DocumentTypeDriverLicense)
	err = suite.documentRepo.Create(suite.ctx, document)
	require.NoError(suite.T(), err)

	// 4. Верификация документа
	err = suite.documentRepo.UpdateStatus(suite.ctx, document.ID, entities.VerificationStatusVerified, stringPtr("admin"), nil)
	require.NoError(suite.T(), err)

	// 5. Переход к верифицированному статусу
	err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, entities.StatusVerified)
	require.NoError(suite.T(), err)

	// 6. Переход к доступному статусу
	err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, entities.StatusAvailable)
	require.NoError(suite.T(), err)

	// 7. Проверка готовности к заказам
	err = suite.driverService.ValidateDriverForOrder(suite.ctx, createdDriver.ID)
	require.NoError(suite.T(), err)

	// 8. Добавление местоположения
	location := fixtures.CreateTestLocation(createdDriver.ID)
	err = suite.locationService.UpdateLocation(suite.ctx, location)
	require.NoError(suite.T(), err)

	// 9. Проверка текущего местоположения
	currentLocation, err := suite.locationService.GetCurrentLocation(suite.ctx, createdDriver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), createdDriver.ID, currentLocation.DriverID)

	// 10. Финальная проверка статуса
	finalDriver, err := suite.driverService.GetDriverByID(suite.ctx, createdDriver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.StatusAvailable, finalDriver.Status)
	assert.True(suite.T(), finalDriver.CanReceiveOrders())
}

// TestDriverValidationWithDocuments тестирует валидацию водителя с документами
func (suite *ServiceIntegrationTestSuite) TestDriverValidationWithDocuments() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Переводим в доступный статус
	err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, entities.StatusAvailable)
	require.NoError(suite.T(), err)

	// Act - проверяем валидацию без документов
	err = suite.driverService.ValidateDriverForOrder(suite.ctx, createdDriver.ID)

	// Assert - должна быть ошибка, так как нет верифицированных документов
	assert.Equal(suite.T(), entities.ErrDocumentNotVerified, err)

	// Arrange - добавляем верифицированный документ
	document := fixtures.CreateTestDocument(createdDriver.ID, entities.DocumentTypeDriverLicense)
	err = suite.documentRepo.Create(suite.ctx, document)
	require.NoError(suite.T(), err)

	err = suite.documentRepo.UpdateStatus(suite.ctx, document.ID, entities.VerificationStatusVerified, stringPtr("admin"), nil)
	require.NoError(suite.T(), err)

	// Act - повторная проверка валидации
	err = suite.driverService.ValidateDriverForOrder(suite.ctx, createdDriver.ID)

	// Assert - теперь должно быть OK
	assert.NoError(suite.T(), err)
}

// TestLocationTrackingWorkflow тестирует workflow отслеживания местоположения
func (suite *ServiceIntegrationTestSuite) TestLocationTrackingWorkflow() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	orderID := uuid.New()

	// 1. Начальное местоположение
	initialLocation := fixtures.CreateTestLocation(createdDriver.ID)
	err = suite.locationService.UpdateLocation(suite.ctx, initialLocation)
	require.NoError(suite.T(), err)

	// 2. Начало отслеживания заказа
	err = suite.locationService.StartOrderTracking(suite.ctx, createdDriver.ID, orderID)
	require.NoError(suite.T(), err)

	// 3. Обновления местоположения во время поездки
	tripLocations := fixtures.CreateTestLocationHistory(createdDriver.ID, 5, 1*time.Minute)
	for _, location := range tripLocations {
		location.Metadata = entities.Metadata{
			"on_trip":  true,
			"order_id": orderID.String(),
		}
		err = suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// 4. Получение статистики поездки
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now()
	stats, err := suite.locationService.GetLocationStats(suite.ctx, createdDriver.ID, from, to)
	require.NoError(suite.T(), err)

	assert.Greater(suite.T(), stats.TotalPoints, 5)
	assert.Greater(suite.T(), stats.DistanceTraveled, 0.0)

	// 5. Завершение отслеживания заказа
	err = suite.locationService.StopOrderTracking(suite.ctx, createdDriver.ID, orderID)
	require.NoError(suite.T(), err)

	// 6. Проверка текущего местоположения
	currentLocation, err := suite.locationService.GetCurrentLocation(suite.ctx, createdDriver.ID)
	require.NoError(suite.T(), err)

	// Метаданные должны указывать, что поездка завершена
	assert.Equal(suite.T(), false, currentLocation.Metadata["on_trip"])
	_, hasOrderID := currentLocation.Metadata["order_id"]
	assert.False(suite.T(), hasOrderID)
}

// TestNearbyDriversIntegration тестирует интеграцию поиска водителей поблизости
func (suite *ServiceIntegrationTestSuite) TestNearbyDriversIntegration() {
	// Arrange
	// Создаем водителей с разными статусами
	drivers := fixtures.CreateMultipleTestDrivers(4)
	statuses := []entities.Status{
		entities.StatusAvailable,
		entities.StatusOnShift,
		entities.StatusInactive,
		entities.StatusBlocked,
	}

	var activeDriverIDs []uuid.UUID

	for i, driver := range drivers {
		createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
		require.NoError(suite.T(), err)

		// Обновляем статус
		err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, statuses[i])
		require.NoError(suite.T(), err)

		if statuses[i] == entities.StatusAvailable || statuses[i] == entities.StatusOnShift {
			activeDriverIDs = append(activeDriverIDs, createdDriver.ID)
		}

		// Добавляем местоположение рядом с центром
		location := fixtures.CreateTestLocationWithCoords(createdDriver.ID, 55.7558+float64(i)*0.001, 37.6173+float64(i)*0.001)
		err = suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act
	nearbyLocations, err := suite.locationService.GetNearbyDrivers(suite.ctx, 55.7558, 37.6173, 1.0, 10)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), nearbyLocations, 2) // Только активные водители

	// Проверяем, что возвращены только активные водители
	for _, location := range nearbyLocations {
		found := false
		for _, activeID := range activeDriverIDs {
			if location.DriverID == activeID {
				found = true
				break
			}
		}
		assert.True(suite.T(), found, "Returned driver should be active")
	}
}

// TestBatchLocationUpdates тестирует пакетные обновления местоположений
func (suite *ServiceIntegrationTestSuite) TestBatchLocationUpdates() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Создаем большой набор местоположений
	locations := fixtures.CreateTestLocationHistory(createdDriver.ID, 100, 10*time.Second)

	// Act
	start := time.Now()
	err = suite.locationService.BatchUpdateLocations(suite.ctx, locations)
	duration := time.Since(start)

	// Assert
	require.NoError(suite.T(), err)
	suite.T().Logf("Batch update of 100 locations took: %v", duration)

	// Проверяем, что операция быстрая
	assert.Less(suite.T(), duration, 5*time.Second)

	// Проверяем, что все местоположения сохранены
	history, err := suite.locationService.GetLocationHistory(suite.ctx, createdDriver.ID,
		time.Now().Add(-2*time.Hour), time.Now())
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), history, 100)
}

// TestDataConsistency тестирует консистентность данных между сервисами
func (suite *ServiceIntegrationTestSuite) TestDataConsistency() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act - обновляем рейтинг через сервис
	newRating := 4.5
	err = suite.driverService.UpdateDriverRating(suite.ctx, createdDriver.ID, newRating)
	require.NoError(suite.T(), err)

	// Assert - проверяем, что изменения видны через разные методы доступа

	// Через DriverService
	updatedDriver, err := suite.driverService.GetDriverByID(suite.ctx, createdDriver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newRating, updatedDriver.CurrentRating)

	// Через DriverRepository напрямую
	repoDriver, err := suite.driverRepo.GetByID(suite.ctx, createdDriver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newRating, repoDriver.CurrentRating)

	// Проверяем, что updated_at изменился
	assert.True(suite.T(), updatedDriver.UpdatedAt.After(createdDriver.UpdatedAt))
}

// TestLocationHistoryAnalytics тестирует аналитику по истории местоположений
func (suite *ServiceIntegrationTestSuite) TestLocationHistoryAnalytics() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Создаем маршрут с известными координатами для расчета расстояния
	locations := []*entities.DriverLocation{
		fixtures.CreateTestLocationWithCoords(createdDriver.ID, 55.7558, 37.6173), // Красная площадь
		fixtures.CreateTestLocationWithCoords(createdDriver.ID, 55.7614, 37.6193), // ~600м на север
		fixtures.CreateTestLocationWithCoords(createdDriver.ID, 55.7670, 37.6213), // еще ~600м на север
	}

	// Устанавливаем времена и скорости
	baseTime := time.Now().Add(-30 * time.Minute)
	speeds := []float64{0, 30, 45} // км/ч

	for i, location := range locations {
		location.RecordedAt = baseTime.Add(time.Duration(i*10) * time.Minute)
		location.CreatedAt = location.RecordedAt
		location.Speed = &speeds[i]

		err = suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act
	from := baseTime.Add(-5 * time.Minute)
	to := time.Now()
	stats, err := suite.locationService.GetLocationStats(suite.ctx, createdDriver.ID, from, to)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, stats.TotalPoints)
	assert.Greater(suite.T(), stats.DistanceTraveled, 1.0) // Примерно 1.2км
	assert.Less(suite.T(), stats.DistanceTraveled, 2.0)
	assert.Equal(suite.T(), 45.0, stats.MaxSpeed)
	assert.Greater(suite.T(), stats.AverageSpeed, 0.0)
	assert.Equal(suite.T(), int64(20), stats.TimeSpan) // 20 минут между первой и последней точкой
}

// TestConcurrentDriverOperations тестирует конкурентные операции с водителем
func (suite *ServiceIntegrationTestSuite) TestConcurrentDriverOperations() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act - одновременные операции
	const goroutineCount = 10
	done := make(chan error, goroutineCount*2)

	// Одновременные обновления рейтинга
	for i := 0; i < goroutineCount; i++ {
		go func(rating float64) {
			done <- suite.driverService.UpdateDriverRating(suite.ctx, createdDriver.ID, rating)
		}(4.0 + float64(i%2)*0.5) // Рейтинги 4.0 или 4.5
	}

	// Одновременные увеличения счетчика поездок
	for i := 0; i < goroutineCount; i++ {
		go func() {
			done <- suite.driverService.IncrementTripCount(suite.ctx, createdDriver.ID)
		}()
	}

	// Ждем завершения всех операций
	for i := 0; i < goroutineCount*2; i++ {
		err := <-done
		assert.NoError(suite.T(), err)
	}

	// Assert
	finalDriver, err := suite.driverService.GetDriverByID(suite.ctx, createdDriver.ID)
	require.NoError(suite.T(), err)

	// Рейтинг должен быть одним из ожидаемых значений
	assert.True(suite.T(), finalDriver.CurrentRating == 4.0 || finalDriver.CurrentRating == 4.5)

	// Счетчик поездок должен увеличиться на количество горутин
	assert.Equal(suite.T(), goroutineCount, finalDriver.TotalTrips)
}

// TestLocationCleanupIntegration тестирует интеграцию очистки старых местоположений
func (suite *ServiceIntegrationTestSuite) TestLocationCleanupIntegration() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	now := time.Now()

	// Создаем старые местоположения (старше 30 дней)
	oldLocations := fixtures.CreateTestLocationHistory(createdDriver.ID, 5, 1*time.Hour)
	for _, location := range oldLocations {
		location.RecordedAt = now.Add(-35 * 24 * time.Hour) // 35 дней назад
		location.CreatedAt = location.RecordedAt
		err = suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Создаем новые местоположения
	newLocations := fixtures.CreateTestLocationHistory(createdDriver.ID, 3, 30*time.Minute)
	for _, location := range newLocations {
		location.RecordedAt = now.Add(-time.Duration(len(newLocations)-1) * 30 * time.Minute)
		location.CreatedAt = location.RecordedAt
		err = suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act
	err = suite.locationService.CleanupOldLocations(suite.ctx)

	// Assert
	require.NoError(suite.T(), err)

	// Проверяем, что старые местоположения удалены
	for _, location := range oldLocations {
		_, err := suite.locationRepo.GetByID(suite.ctx, location.ID)
		assert.Equal(suite.T(), entities.ErrLocationNotFound, err)
	}

	// Проверяем, что новые местоположения остались
	for _, location := range newLocations {
		_, err := suite.locationRepo.GetByID(suite.ctx, location.ID)
		assert.NoError(suite.T(), err)
	}
}

// TestDriverStatusTransitions тестирует все возможные переходы статусов
func (suite *ServiceIntegrationTestSuite) TestDriverStatusTransitions() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Test valid transitions
	validTransitions := []struct {
		from entities.Status
		to   entities.Status
	}{
		{entities.StatusRegistered, entities.StatusPendingVerification},
		{entities.StatusPendingVerification, entities.StatusVerified},
		{entities.StatusVerified, entities.StatusAvailable},
		{entities.StatusAvailable, entities.StatusOnShift},
		{entities.StatusOnShift, entities.StatusBusy},
		{entities.StatusBusy, entities.StatusOnShift},
		{entities.StatusOnShift, entities.StatusAvailable},
		{entities.StatusAvailable, entities.StatusInactive},
		{entities.StatusInactive, entities.StatusAvailable},
	}

	currentStatus := entities.StatusRegistered
	for _, transition := range validTransitions {
		// Убеждаемся, что текущий статус соответствует ожидаемому
		if currentStatus != transition.from {
			err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, transition.from)
			require.NoError(suite.T(), err)
		}

		// Act
		err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, transition.to)

		// Assert
		assert.NoError(suite.T(), err, "Transition from %s to %s should be valid", transition.from, transition.to)
		currentStatus = transition.to
	}

	// Test invalid transition
	err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, entities.StatusRegistered)
	assert.Error(suite.T(), err, "Invalid transition should fail")
}

// TestLocationAccuracyFiltering тестирует фильтрацию по точности GPS
func (suite *ServiceIntegrationTestSuite) TestLocationAccuracyFiltering() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Создаем местоположения с разной точностью
	accuracies := []float64{5.0, 15.0, 30.0, 60.0, 100.0} // метры

	for i, accuracy := range accuracies {
		location := fixtures.CreateTestLocation(createdDriver.ID)
		location.Accuracy = &accuracy
		location.Latitude += float64(i) * 0.001 // Немного сдвигаем координаты

		err = suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act - получаем все местоположения
	history, err := suite.locationService.GetLocationHistory(suite.ctx, createdDriver.ID,
		time.Now().Add(-1*time.Hour), time.Now())

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), history, 5)

	// Проверяем классификацию точности
	highAccuracyCount := 0
	for _, location := range history {
		if location.IsHighAccuracy() {
			highAccuracyCount++
		}
	}

	assert.Equal(suite.T(), 3, highAccuracyCount) // 5м, 15м, 30м считаются высокой точностью
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

// Запуск тестового suite
func TestServiceIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceIntegrationTestSuite))
}
