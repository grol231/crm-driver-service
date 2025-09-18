//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"driver-service/internal/config"
	"driver-service/internal/domain/entities"
	"driver-service/internal/domain/services"
	httpServer "driver-service/internal/interfaces/http"
	httpHandlers "driver-service/internal/interfaces/http/handlers"
	"driver-service/internal/repositories"
	"driver-service/tests/fixtures"
	"driver-service/tests/helpers"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// LocationAPITestSuite тестовый suite для Location HTTP API
type LocationAPITestSuite struct {
	suite.Suite
	testDB          *helpers.TestDB
	server          *httpServer.Server
	router          *gin.Engine
	driverService   services.DriverService
	locationService services.LocationService
	ctx             context.Context
	testDriverID    uuid.UUID
}

// SetupSuite выполняется один раз перед всеми тестами
func (suite *LocationAPITestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

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

	// Создаем handlers
	driverHandler := httpHandlers.NewDriverHandler(suite.driverService, logger)
	locationHandler := httpHandlers.NewLocationHandler(suite.locationService, logger)

	// Создаем тестовую конфигурацию
	cfg := &config.Config{
		Server: config.ServerConfig{
			HTTPPort:    8001,
			Environment: "test",
			Timeout:     30 * time.Second,
		},
	}

	// Создаем HTTP сервер
	suite.server = httpServer.NewServer(cfg, logger, driverHandler, locationHandler)
	suite.router = suite.server.GetRouter()
}

// TearDownSuite выполняется один раз после всех тестов
func (suite *LocationAPITestSuite) TearDownSuite() {
	suite.testDB.TeardownTestDB(suite.T())
}

// SetupTest выполняется перед каждым тестом
func (suite *LocationAPITestSuite) SetupTest() {
	suite.testDB.CleanupTables(suite.T())

	// Создаем тестового водителя
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)
	suite.testDriverID = createdDriver.ID
}

// TestUpdateLocationAPI тестирует обновление местоположения через API
func (suite *LocationAPITestSuite) TestUpdateLocationAPI() {
	// Arrange
	locationData := map[string]interface{}{
		"latitude":  55.7558,
		"longitude": 37.6173,
		"altitude":  150.0,
		"accuracy":  10.0,
		"speed":     60.5,
		"bearing":   45.0,
	}

	// Act
	bodyBytes, _ := json.Marshal(locationData)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/drivers/%s/locations", suite.testDriverID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.LocationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), suite.testDriverID, response.DriverID)
	assert.Equal(suite.T(), 55.7558, response.Latitude)
	assert.Equal(suite.T(), 37.6173, response.Longitude)
	assert.Equal(suite.T(), 150.0, *response.Altitude)
	assert.Equal(suite.T(), 60.5, *response.Speed)
}

// TestBatchUpdateLocationsAPI тестирует пакетное обновление местоположений
func (suite *LocationAPITestSuite) TestBatchUpdateLocationsAPI() {
	// Arrange
	batchData := map[string]interface{}{
		"locations": []map[string]interface{}{
			{
				"latitude":  55.7558,
				"longitude": 37.6173,
				"speed":     50.0,
				"timestamp": time.Now().Unix(),
			},
			{
				"latitude":  55.7600,
				"longitude": 37.6200,
				"speed":     55.0,
				"timestamp": time.Now().Unix() + 60,
			},
			{
				"latitude":  55.7650,
				"longitude": 37.6250,
				"speed":     60.0,
				"timestamp": time.Now().Unix() + 120,
			},
		},
	}

	// Act
	bodyBytes, _ := json.Marshal(batchData)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/drivers/%s/locations/batch", suite.testDriverID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Locations updated successfully", response["message"])
	assert.Equal(suite.T(), float64(3), response["count"])
}

// TestGetCurrentLocationAPI тестирует получение текущего местоположения
func (suite *LocationAPITestSuite) TestGetCurrentLocationAPI() {
	// Arrange - создаем несколько местоположений
	locations := fixtures.CreateTestLocationHistory(suite.testDriverID, 3, 1*time.Minute)

	for _, location := range locations {
		err := suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/drivers/%s/locations/current", suite.testDriverID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.LocationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), suite.testDriverID, response.DriverID)
	assert.NotEqual(suite.T(), uuid.Nil, response.ID)
}

// TestGetLocationHistoryAPI тестирует получение истории местоположений
func (suite *LocationAPITestSuite) TestGetLocationHistoryAPI() {
	// Arrange
	locations := fixtures.CreateTestLocationHistory(suite.testDriverID, 5, 10*time.Minute)

	for _, location := range locations {
		err := suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act - получаем историю за последние 2 часа
	from := time.Now().Add(-2 * time.Hour).Unix()
	to := time.Now().Unix()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/drivers/%s/locations/history?from=%d&to=%d", suite.testDriverID, from, to), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.LocationHistoryResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), response.Locations, 5)
	assert.Equal(suite.T(), 5, response.Count)
	assert.NotNil(suite.T(), response.Stats)
	assert.Equal(suite.T(), 5, response.Stats.TotalPoints)
}

// TestGetNearbyDriversAPI тестирует поиск водителей поблизости
func (suite *LocationAPITestSuite) TestGetNearbyDriversAPI() {
	// Arrange
	// Создаем нескольких водителей
	drivers := fixtures.CreateMultipleTestDrivers(3)
	var driverIDs []uuid.UUID

	for _, driver := range drivers {
		driver.Status = entities.StatusAvailable
		createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
		require.NoError(suite.T(), err)

		// Обновляем статус на available
		err = suite.driverService.ChangeDriverStatus(suite.ctx, createdDriver.ID, entities.StatusAvailable)
		require.NoError(suite.T(), err)

		driverIDs = append(driverIDs, createdDriver.ID)
	}

	// Создаем местоположения на разных расстояниях
	centerLat := 55.7558
	centerLon := 37.6173

	locations := []*entities.DriverLocation{
		fixtures.CreateTestLocationWithCoords(driverIDs[0], centerLat+0.001, centerLon+0.001), // ~100м
		fixtures.CreateTestLocationWithCoords(driverIDs[1], centerLat+0.01, centerLon+0.01),   // ~1км
		fixtures.CreateTestLocationWithCoords(driverIDs[2], centerLat+0.1, centerLon+0.1),     // ~10км
	}

	for _, location := range locations {
		err := suite.locationService.UpdateLocation(suite.ctx, location)
		require.NoError(suite.T(), err)
	}

	// Act - ищем в радиусе 5км
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/locations/nearby?latitude=%f&longitude=%f&radius_km=5&limit=10", centerLat, centerLon), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.NearbyDriversResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), response.Drivers, 2) // Первые два водителя в радиусе 5км
	assert.Equal(suite.T(), 2, response.Count)

	// Проверяем, что расстояния рассчитаны
	for _, driver := range response.Drivers {
		assert.Greater(suite.T(), driver.Distance, 0.0)
		assert.Less(suite.T(), driver.Distance, 5.0) // В пределах радиуса
	}
}

// TestLocationAPIValidation тестирует валидацию координат
func (suite *LocationAPITestSuite) TestLocationAPIValidation() {
	// Arrange - тестовые случаи с невалидными данными
	testCases := []struct {
		name         string
		locationData map[string]interface{}
		expectedCode int
	}{
		{
			name: "missing latitude",
			locationData: map[string]interface{}{
				"longitude": 37.6173,
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "missing longitude",
			locationData: map[string]interface{}{
				"latitude": 55.7558,
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid latitude",
			locationData: map[string]interface{}{
				"latitude":  91.0, // Больше 90
				"longitude": 37.6173,
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid longitude",
			locationData: map[string]interface{}{
				"latitude":  55.7558,
				"longitude": 181.0, // Больше 180
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Act
			bodyBytes, _ := json.Marshal(tc.locationData)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/drivers/%s/locations", suite.testDriverID), bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tc.expectedCode, w.Code)
		})
	}
}

// TestLocationAPIInvalidDriverID тестирует обработку невалидного ID водителя
func (suite *LocationAPITestSuite) TestLocationAPIInvalidDriverID() {
	// Arrange
	locationData := map[string]interface{}{
		"latitude":  55.7558,
		"longitude": 37.6173,
	}

	// Act
	bodyBytes, _ := json.Marshal(locationData)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers/invalid-uuid/locations", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var errorResponse httpHandlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), errorResponse.Error, "Invalid driver ID format")
}

// TestLocationAPIDriverNotFound тестирует обновление местоположения для несуществующего водителя
func (suite *LocationAPITestSuite) TestLocationAPIDriverNotFound() {
	// Arrange
	nonExistentDriverID := uuid.New()
	locationData := map[string]interface{}{
		"latitude":  55.7558,
		"longitude": 37.6173,
	}

	// Act
	bodyBytes, _ := json.Marshal(locationData)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/drivers/%s/locations", nonExistentDriverID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var errorResponse httpHandlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "DRIVER_NOT_FOUND", errorResponse.Code)
}

// TestGetCurrentLocationAPINotFound тестирует получение местоположения для водителя без GPS данных
func (suite *LocationAPITestSuite) TestGetCurrentLocationAPINotFound() {
	// Act - водитель создан, но местоположения нет
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/drivers/%s/locations/current", suite.testDriverID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var errorResponse httpHandlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "LOCATION_NOT_FOUND", errorResponse.Code)
}

// TestLocationHistoryAPITimeFormats тестирует разные форматы времени в API
func (suite *LocationAPITestSuite) TestLocationHistoryAPITimeFormats() {
	// Arrange
	location := fixtures.CreateTestLocation(suite.testDriverID)
	err := suite.locationService.UpdateLocation(suite.ctx, location)
	require.NoError(suite.T(), err)

	// Test cases с разными форматами времени
	testCases := []struct {
		name     string
		fromTime string
		toTime   string
		expected int // HTTP status code
	}{
		{
			name:     "Unix timestamp",
			fromTime: fmt.Sprintf("%d", time.Now().Add(-1*time.Hour).Unix()),
			toTime:   fmt.Sprintf("%d", time.Now().Unix()),
			expected: http.StatusOK,
		},
		{
			name:     "RFC3339 format",
			fromTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			toTime:   time.Now().Format(time.RFC3339),
			expected: http.StatusOK,
		},
		{
			name:     "Invalid from format",
			fromTime: "invalid-time",
			toTime:   fmt.Sprintf("%d", time.Now().Unix()),
			expected: http.StatusBadRequest,
		},
		{
			name:     "Invalid to format",
			fromTime: fmt.Sprintf("%d", time.Now().Add(-1*time.Hour).Unix()),
			toTime:   "invalid-time",
			expected: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Act
			url := fmt.Sprintf("/api/v1/drivers/%s/locations/history?from=%s&to=%s", suite.testDriverID, tc.fromTime, tc.toTime)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tc.expected, w.Code)
		})
	}
}

// TestNearbyDriversAPIValidation тестирует валидацию параметров поиска поблизости
func (suite *LocationAPITestSuite) TestNearbyDriversAPIValidation() {
	// Test cases с невалидными параметрами
	testCases := []struct {
		name         string
		queryParams  string
		expectedCode int
	}{
		{
			name:         "missing coordinates",
			queryParams:  "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "missing longitude",
			queryParams:  "latitude=55.7558",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "missing latitude",
			queryParams:  "longitude=37.6173",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid latitude",
			queryParams:  "latitude=invalid&longitude=37.6173",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid longitude",
			queryParams:  "latitude=55.7558&longitude=invalid",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "valid coordinates",
			queryParams:  "latitude=55.7558&longitude=37.6173",
			expectedCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Act
			url := "/api/v1/locations/nearby"
			if tc.queryParams != "" {
				url += "?" + tc.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tc.expectedCode, w.Code)
		})
	}
}

// TestLocationAPIWithTimestamp тестирует API с пользовательским timestamp
func (suite *LocationAPITestSuite) TestLocationAPIWithTimestamp() {
	// Arrange
	customTimestamp := time.Now().Add(-30 * time.Minute).Unix()
	locationData := map[string]interface{}{
		"latitude":  55.7558,
		"longitude": 37.6173,
		"timestamp": customTimestamp,
	}

	// Act
	bodyBytes, _ := json.Marshal(locationData)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/drivers/%s/locations", suite.testDriverID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.LocationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	expectedTime := time.Unix(customTimestamp, 0)
	assert.Equal(suite.T(), expectedTime.Unix(), response.RecordedAt.Unix())
}

// TestLocationAPIRateLimit тестирует ограничение частоты запросов (если реализовано)
func (suite *LocationAPITestSuite) TestLocationAPIRateLimit() {
	// Arrange
	locationData := map[string]interface{}{
		"latitude":  55.7558,
		"longitude": 37.6173,
	}
	bodyBytes, _ := json.Marshal(locationData)

	// Act - отправляем много запросов подряд
	successCount := 0
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/drivers/%s/locations", suite.testDriverID), bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	// Assert
	// В данной реализации rate limiting не настроен, поэтому все запросы должны проходить
	assert.Equal(suite.T(), 100, successCount)

	// Примечание: когда rate limiting будет реализован, здесь нужно будет
	// проверить, что часть запросов отклоняется с кодом 429
}

// Запуск тестового suite
func TestLocationAPITestSuite(t *testing.T) {
	suite.Run(t, new(LocationAPITestSuite))
}
