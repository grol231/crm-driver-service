//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/repositories"
	"driver-service/tests/fixtures"
	"driver-service/tests/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// LocationRepositoryTestSuite тестовый suite для LocationRepository
type LocationRepositoryTestSuite struct {
	suite.Suite
	testDB       *helpers.TestDB
	locationRepo repositories.LocationRepository
	driverRepo   repositories.DriverRepository
	ctx          context.Context
	testDriverID uuid.UUID
}

// SetupSuite выполняется один раз перед всеми тестами
func (suite *LocationRepositoryTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDB(suite.T())
	logger := helpers.CreateTestLogger(suite.T())

	suite.locationRepo = repositories.NewLocationRepository(suite.testDB.DB, logger)
	suite.driverRepo = repositories.NewDriverRepository(suite.testDB.DB, logger)
	suite.ctx = context.Background()
}

// TearDownSuite выполняется один раз после всех тестов
func (suite *LocationRepositoryTestSuite) TearDownSuite() {
	suite.testDB.TeardownTestDB(suite.T())
}

// SetupTest выполняется перед каждым тестом
func (suite *LocationRepositoryTestSuite) SetupTest() {
	suite.testDB.CleanupTables(suite.T())

	// Создаем тестового водителя
	driver := fixtures.CreateTestDriver()
	err := suite.driverRepo.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)
	suite.testDriverID = driver.ID
}

// TestCreateLocation тестирует создание местоположения
func (suite *LocationRepositoryTestSuite) TestCreateLocation() {
	// Arrange
	location := fixtures.CreateTestLocation(suite.testDriverID)

	// Act
	err := suite.locationRepo.Create(suite.ctx, location)

	// Assert
	require.NoError(suite.T(), err)

	// Проверяем, что местоположение создано
	createdLocation, err := suite.locationRepo.GetByID(suite.ctx, location.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), location.DriverID, createdLocation.DriverID)
	assert.Equal(suite.T(), location.Latitude, createdLocation.Latitude)
	assert.Equal(suite.T(), location.Longitude, createdLocation.Longitude)
}

// TestGetLatestLocation тестирует получение последнего местоположения
func (suite *LocationRepositoryTestSuite) TestGetLatestLocation() {
	// Arrange
	locations := fixtures.CreateTestLocationHistory(suite.testDriverID, 5, 1*time.Minute)

	for _, location := range locations {
		err := suite.locationRepo.Create(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act
	latestLocation, err := suite.locationRepo.GetLatestByDriverID(suite.ctx, suite.testDriverID)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testDriverID, latestLocation.DriverID)

	// Проверяем, что это действительно последнее местоположение
	for _, location := range locations {
		if location.RecordedAt.After(latestLocation.RecordedAt) {
			suite.T().Errorf("Found location with later timestamp than 'latest'")
		}
	}
}

// TestGetLocationHistory тестирует получение истории местоположений
func (suite *LocationRepositoryTestSuite) TestGetLocationHistory() {
	// Arrange
	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now

	// Создаем местоположения: некоторые в диапазоне, некоторые вне
	locations := []*entities.DriverLocation{
		fixtures.CreateTestLocationWithCoords(suite.testDriverID, 55.7558, 37.6173), // Вне диапазона (старше)
		fixtures.CreateTestLocationWithCoords(suite.testDriverID, 55.7600, 37.6200), // В диапазоне
		fixtures.CreateTestLocationWithCoords(suite.testDriverID, 55.7650, 37.6250), // В диапазоне
		fixtures.CreateTestLocationWithCoords(suite.testDriverID, 55.7700, 37.6300), // Вне диапазона (новее)
	}

	// Устанавливаем времена записи
	locations[0].RecordedAt = now.Add(-2 * time.Hour)    // Вне диапазона
	locations[1].RecordedAt = now.Add(-45 * time.Minute) // В диапазоне
	locations[2].RecordedAt = now.Add(-30 * time.Minute) // В диапазоне
	locations[3].RecordedAt = now.Add(30 * time.Minute)  // Вне диапазона

	for _, location := range locations {
		err := suite.locationRepo.Create(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act
	historyLocations, err := suite.locationRepo.GetByDriverIDInTimeRange(suite.ctx, suite.testDriverID, from, to)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), historyLocations, 2) // Только locations[1] и locations[2]

	// Проверяем сортировку по времени (ASC)
	if len(historyLocations) > 1 {
		assert.True(suite.T(), historyLocations[0].RecordedAt.Before(historyLocations[1].RecordedAt))
	}
}

// TestBatchCreateLocations тестирует пакетное создание местоположений
func (suite *LocationRepositoryTestSuite) TestBatchCreateLocations() {
	// Arrange
	locations := fixtures.CreateTestLocationHistory(suite.testDriverID, 10, 30*time.Second)

	// Act
	err := suite.locationRepo.CreateBatch(suite.ctx, locations)

	// Assert
	require.NoError(suite.T(), err)

	// Проверяем, что все местоположения созданы
	for _, location := range locations {
		createdLocation, err := suite.locationRepo.GetByID(suite.ctx, location.ID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), location.DriverID, createdLocation.DriverID)
		assert.Equal(suite.T(), location.Latitude, createdLocation.Latitude)
		assert.Equal(suite.T(), location.Longitude, createdLocation.Longitude)
	}
}

// TestGetNearbyLocations тестирует поиск ближайших местоположений
func (suite *LocationRepositoryTestSuite) TestGetNearbyLocations() {
	// Arrange
	// Создаем нескольких водителей
	drivers := fixtures.CreateMultipleTestDrivers(4)
	for _, driver := range drivers {
		err := suite.driverRepo.Create(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Создаем местоположения на разных расстояниях от центра (Красная площадь)
	centerLat := 55.7558
	centerLon := 37.6173

	locations := []*entities.DriverLocation{
		fixtures.CreateTestLocationWithCoords(drivers[0].ID, centerLat+0.001, centerLon+0.001), // ~100м
		fixtures.CreateTestLocationWithCoords(drivers[1].ID, centerLat+0.01, centerLon+0.01),   // ~1км
		fixtures.CreateTestLocationWithCoords(drivers[2].ID, centerLat+0.05, centerLon+0.05),   // ~5км
		fixtures.CreateTestLocationWithCoords(drivers[3].ID, centerLat+0.1, centerLon+0.1),     // ~10км
	}

	for _, location := range locations {
		err := suite.locationRepo.Create(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act - ищем в радиусе 3км
	nearbyLocations, err := suite.locationRepo.GetNearby(suite.ctx, centerLat, centerLon, 3.0, 10)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), nearbyLocations, 2) // Первые два местоположения
}

// TestDeleteOldLocations тестирует удаление старых местоположений
func (suite *LocationRepositoryTestSuite) TestDeleteOldLocations() {
	// Arrange
	now := time.Now()
	cutoffTime := now.Add(-1 * time.Hour)

	// Создаем местоположения: старые и новые
	oldLocations := fixtures.CreateTestLocationHistory(suite.testDriverID, 3, 10*time.Minute)
	newLocations := fixtures.CreateTestLocationHistory(suite.testDriverID, 2, 5*time.Minute)

	// Устанавливаем времена для старых местоположений
	for i, location := range oldLocations {
		location.RecordedAt = now.Add(-time.Duration(2+i) * time.Hour)
		err := suite.locationRepo.Create(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Устанавливаем времена для новых местоположений
	for i, location := range newLocations {
		location.RecordedAt = now.Add(-time.Duration(i*10) * time.Minute)
		err := suite.locationRepo.Create(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act
	err := suite.locationRepo.DeleteOld(suite.ctx, cutoffTime)

	// Assert
	require.NoError(suite.T(), err)

	// Проверяем, что старые удалены, а новые остались
	for _, location := range oldLocations {
		_, err := suite.locationRepo.GetByID(suite.ctx, location.ID)
		assert.Equal(suite.T(), entities.ErrLocationNotFound, err)
	}

	for _, location := range newLocations {
		_, err := suite.locationRepo.GetByID(suite.ctx, location.ID)
		assert.NoError(suite.T(), err)
	}
}

// TestLocationFilters тестирует фильтрацию местоположений
func (suite *LocationRepositoryTestSuite) TestLocationFilters() {
	// Arrange
	now := time.Now()
	locations := fixtures.CreateTestLocationHistory(suite.testDriverID, 10, 5*time.Minute)

	for _, location := range locations {
		err := suite.locationRepo.Create(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act - фильтр с лимитом
	filters := &entities.LocationFilters{
		DriverID: &suite.testDriverID,
		Limit:    5,
	}

	result, err := suite.locationRepo.List(suite.ctx, filters)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 5)

	// Проверяем, что все местоположения принадлежат нужному водителю
	for _, location := range result {
		assert.Equal(suite.T(), suite.testDriverID, location.DriverID)
	}
}

// TestLocationTimeRangeFilter тестирует фильтрацию по времени
func (suite *LocationRepositoryTestSuite) TestLocationTimeRangeFilter() {
	// Arrange
	now := time.Now()
	from := now.Add(-2 * time.Hour)
	to := now.Add(-1 * time.Hour)

	locations := fixtures.CreateTestLocationHistory(suite.testDriverID, 6, 30*time.Minute)

	// Устанавливаем времена так, чтобы только некоторые попали в диапазон
	baseTimes := []time.Duration{-3 * time.Hour, -2 * time.Hour, -90 * time.Minute, -60 * time.Minute, -30 * time.Minute, 0}

	for i, location := range locations {
		location.RecordedAt = now.Add(baseTimes[i])
		err := suite.locationRepo.Create(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act
	filters := &entities.LocationFilters{
		DriverID: &suite.testDriverID,
		From:     &from,
		To:       &to,
	}

	result, err := suite.locationRepo.List(suite.ctx, filters)

	// Assert
	require.NoError(suite.T(), err)

	// Должны попасть locations с индексами 1, 2, 3 (времена -2h, -1.5h, -1h)
	assert.Len(suite.T(), result, 3)

	for _, location := range result {
		assert.True(suite.T(), location.RecordedAt.After(from) || location.RecordedAt.Equal(from))
		assert.True(suite.T(), location.RecordedAt.Before(to) || location.RecordedAt.Equal(to))
	}
}

// TestLocationValidation тестирует валидацию координат
func (suite *LocationRepositoryTestSuite) TestLocationValidation() {
	// Arrange - невалидные координаты
	invalidLocations := []*entities.DriverLocation{
		fixtures.CreateTestLocationWithCoords(suite.testDriverID, 91.0, 37.6173),   // Широта > 90
		fixtures.CreateTestLocationWithCoords(suite.testDriverID, 55.7558, 181.0),  // Долгота > 180
		fixtures.CreateTestLocationWithCoords(suite.testDriverID, -91.0, 37.6173),  // Широта < -90
		fixtures.CreateTestLocationWithCoords(suite.testDriverID, 55.7558, -181.0), // Долгота < -180
	}

	// Act & Assert
	for i, location := range invalidLocations {
		err := suite.locationRepo.Create(suite.ctx, location)
		assert.Error(suite.T(), err, "Invalid location %d should cause error", i)
	}
}

// TestLocationPerformance тестирует производительность при большом количестве записей
func (suite *LocationRepositoryTestSuite) TestLocationPerformance() {
	// Arrange
	const locationCount = 1000
	locations := fixtures.CreateTestLocationHistory(suite.testDriverID, locationCount, 1*time.Second)

	// Act - измеряем время пакетной вставки
	start := time.Now()
	err := suite.locationRepo.CreateBatch(suite.ctx, locations)
	batchDuration := time.Since(start)

	// Assert
	require.NoError(suite.T(), err)
	suite.T().Logf("Batch insert of %d locations took: %v", locationCount, batchDuration)

	// Проверяем, что пакетная вставка быстрее определенного порога
	assert.Less(suite.T(), batchDuration, 5*time.Second, "Batch insert should be fast")

	// Проверяем получение последнего местоположения
	start = time.Now()
	_, err = suite.locationRepo.GetLatestByDriverID(suite.ctx, suite.testDriverID)
	queryDuration := time.Since(start)

	require.NoError(suite.T(), err)
	suite.T().Logf("Get latest location query took: %v", queryDuration)
	assert.Less(suite.T(), queryDuration, 100*time.Millisecond, "Latest location query should be fast")
}

// TestLocationConcurrency тестирует конкурентные операции с местоположениями
func (suite *LocationRepositoryTestSuite) TestLocationConcurrency() {
	// Arrange
	const goroutineCount = 10
	const locationsPerGoroutine = 5

	// Act - одновременно создаем местоположения из разных горутин
	done := make(chan error, goroutineCount)

	for i := 0; i < goroutineCount; i++ {
		go func(routineID int) {
			locations := make([]*entities.DriverLocation, locationsPerGoroutine)
			for j := 0; j < locationsPerGoroutine; j++ {
				location := fixtures.CreateTestLocation(suite.testDriverID)
				location.Latitude += float64(routineID) * 0.001
				location.Longitude += float64(j) * 0.001
				locations[j] = location
			}

			done <- suite.locationRepo.CreateBatch(suite.ctx, locations)
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < goroutineCount; i++ {
		err := <-done
		assert.NoError(suite.T(), err)
	}

	// Assert - проверяем общее количество записей
	filters := &entities.LocationFilters{
		DriverID: &suite.testDriverID,
	}

	allLocations, err := suite.locationRepo.List(suite.ctx, filters)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), allLocations, goroutineCount*locationsPerGoroutine)
}

// TestLocationMetadata тестирует работу с метаданными
func (suite *LocationRepositoryTestSuite) TestLocationMetadata() {
	// Arrange
	location := fixtures.CreateTestLocation(suite.testDriverID)
	location.Metadata = entities.Metadata{
		"on_trip":     true,
		"order_id":    uuid.New().String(),
		"speed_limit": 60,
		"road_type":   "highway",
	}

	// Act
	err := suite.locationRepo.Create(suite.ctx, location)
	require.NoError(suite.T(), err)

	// Assert
	createdLocation, err := suite.locationRepo.GetByID(suite.ctx, location.ID)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), location.Metadata["on_trip"], createdLocation.Metadata["on_trip"])
	assert.Equal(suite.T(), location.Metadata["order_id"], createdLocation.Metadata["order_id"])
	assert.Equal(suite.T(), location.Metadata["speed_limit"], createdLocation.Metadata["speed_limit"])
	assert.Equal(suite.T(), location.Metadata["road_type"], createdLocation.Metadata["road_type"])
}

// TestLocationAccuracy тестирует работу с точностью GPS
func (suite *LocationRepositoryTestSuite) TestLocationAccuracy() {
	// Arrange
	location := fixtures.CreateTestLocation(suite.testDriverID)

	// Тестируем разные уровни точности
	testCases := []struct {
		accuracy float64
		expected bool // высокая точность или нет
	}{
		{5.0, true},    // Высокая точность
		{15.0, true},   // Хорошая точность
		{45.0, true},   // Средняя точность
		{55.0, false},  // Низкая точность
		{100.0, false}, // Очень низкая точность
	}

	for _, tc := range testCases {
		location.ID = uuid.New()
		location.Accuracy = &tc.accuracy

		err := suite.locationRepo.Create(suite.ctx, location)
		require.NoError(suite.T(), err)

		createdLocation, err := suite.locationRepo.GetByID(suite.ctx, location.ID)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), tc.expected, createdLocation.IsHighAccuracy())
	}
}

// TestLocationDistanceCalculation тестирует расчет расстояний
func (suite *LocationRepositoryTestSuite) TestLocationDistanceCalculation() {
	// Arrange
	// Красная площадь, Москва
	location1 := fixtures.CreateTestLocationWithCoords(suite.testDriverID, 55.7558, 37.6173)
	// Примерно в 1км от Красной площади
	location2 := fixtures.CreateTestLocationWithCoords(suite.testDriverID, 55.7650, 37.6250)

	err := suite.locationRepo.Create(suite.ctx, location1)
	require.NoError(suite.T(), err)

	err = suite.locationRepo.Create(suite.ctx, location2)
	require.NoError(suite.T(), err)

	// Act
	distance := location1.DistanceTo(location2)

	// Assert
	// Расстояние должно быть примерно 1км (с погрешностью)
	assert.Greater(suite.T(), distance, 0.8) // Минимум 800м
	assert.Less(suite.T(), distance, 1.5)    // Максимум 1.5км
}

// Запуск тестового suite
func TestLocationRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(LocationRepositoryTestSuite))
}
