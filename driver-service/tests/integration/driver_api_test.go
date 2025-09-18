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

// DriverAPITestSuite тестовый suite для Driver HTTP API
type DriverAPITestSuite struct {
	suite.Suite
	testDB        *helpers.TestDB
	server        *httpServer.Server
	router        *gin.Engine
	driverService services.DriverService
	ctx           context.Context
}

// SetupSuite выполняется один раз перед всеми тестами
func (suite *DriverAPITestSuite) SetupSuite() {
	// Настройка тестового режима для Gin
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
	locationService := services.NewLocationService(locationRepo, driverRepo, eventBus, logger)

	// Создаем handlers
	driverHandler := httpHandlers.NewDriverHandler(suite.driverService, logger)
	locationHandler := httpHandlers.NewLocationHandler(locationService, logger)

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
func (suite *DriverAPITestSuite) TearDownSuite() {
	suite.testDB.TeardownTestDB(suite.T())
}

// SetupTest выполняется перед каждым тестом
func (suite *DriverAPITestSuite) SetupTest() {
	suite.testDB.CleanupTables(suite.T())
}

// TestCreateDriverAPI тестирует создание водителя через API
func (suite *DriverAPITestSuite) TestCreateDriverAPI() {
	// Arrange
	requestBody := map[string]interface{}{
		"phone":           "+79001234567",
		"email":           "test@example.com",
		"first_name":      "Иван",
		"last_name":       "Тестовый",
		"birth_date":      "1985-05-15T00:00:00Z",
		"passport_series": "1234",
		"passport_number": "567890",
		"license_number":  "TEST123456",
		"license_expiry":  "2026-12-31T00:00:00Z",
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var response httpHandlers.DriverResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "+79001234567", response.Phone)
	assert.Equal(suite.T(), "test@example.com", response.Email)
	assert.Equal(suite.T(), entities.StatusRegistered, response.Status)
	assert.NotEqual(suite.T(), uuid.Nil, response.ID)
}

// TestCreateDriverAPIValidation тестирует валидацию при создании водителя
func (suite *DriverAPITestSuite) TestCreateDriverAPIValidation() {
	// Arrange - невалидные данные
	testCases := []struct {
		name         string
		requestBody  map[string]interface{}
		expectedCode int
	}{
		{
			name: "missing phone",
			requestBody: map[string]interface{}{
				"email":      "test@example.com",
				"first_name": "Иван",
				"last_name":  "Тестовый",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "invalid email",
			requestBody: map[string]interface{}{
				"phone":      "+79001234567",
				"email":      "invalid-email",
				"first_name": "Иван",
				"last_name":  "Тестовый",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "empty first_name",
			requestBody: map[string]interface{}{
				"phone":      "+79001234567",
				"email":      "test@example.com",
				"first_name": "",
				"last_name":  "Тестовый",
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Act
			bodyBytes, _ := json.Marshal(tc.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tc.expectedCode, w.Code)
		})
	}
}

// TestGetDriverAPI тестирует получение водителя через API
func (suite *DriverAPITestSuite) TestGetDriverAPI() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	_, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/drivers/%s", driver.ID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.DriverResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), driver.Phone, response.Phone)
	assert.Equal(suite.T(), driver.Email, response.Email)
}

// TestGetDriverAPINotFound тестирует получение несуществующего водителя
func (suite *DriverAPITestSuite) TestGetDriverAPINotFound() {
	// Act
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/drivers/%s", uuid.New()), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	var errorResponse httpHandlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "DRIVER_NOT_FOUND", errorResponse.Code)
}

// TestListDriversAPI тестирует получение списка водителей через API
func (suite *DriverAPITestSuite) TestListDriversAPI() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(5)
	for _, driver := range drivers {
		_, err := suite.driverService.CreateDriver(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act
	req := httptest.NewRequest(http.MethodGet, "/api/v1/drivers?limit=3&offset=0", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.ListDriversResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), response.Drivers, 3)
	assert.Equal(suite.T(), 5, response.Total)
	assert.Equal(suite.T(), 3, response.Limit)
	assert.Equal(suite.T(), 0, response.Offset)
	assert.True(suite.T(), response.HasMore)
}

// TestUpdateDriverAPI тестирует обновление водителя через API
func (suite *DriverAPITestSuite) TestUpdateDriverAPI() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	updateData := map[string]interface{}{
		"first_name": "Обновленное Имя",
		"email":      "updated@example.com",
	}

	// Act
	bodyBytes, _ := json.Marshal(updateData)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/drivers/%s", createdDriver.ID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.DriverResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "Обновленное Имя", response.FirstName)
	assert.Equal(suite.T(), "updated@example.com", response.Email)
}

// TestChangeDriverStatusAPI тестирует изменение статуса водителя через API
func (suite *DriverAPITestSuite) TestChangeDriverStatusAPI() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	statusData := map[string]interface{}{
		"status": "pending_verification",
	}

	// Act
	bodyBytes, _ := json.Marshal(statusData)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/drivers/%s/status", createdDriver.ID), bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "pending_verification", response["status"])
}

// TestDeleteDriverAPI тестирует удаление водителя через API
func (suite *DriverAPITestSuite) TestDeleteDriverAPI() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/drivers/%s", createdDriver.ID), nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// Проверяем, что водитель действительно удален
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/drivers/%s", createdDriver.ID), nil)
	w2 := httptest.NewRecorder()
	suite.router.ServeHTTP(w2, req2)
	assert.Equal(suite.T(), http.StatusNotFound, w2.Code)
}

// TestGetActiveDriversAPI тестирует получение активных водителей через API
func (suite *DriverAPITestSuite) TestGetActiveDriversAPI() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(5)

	// Устанавливаем разные статусы
	statuses := []entities.Status{
		entities.StatusAvailable,
		entities.StatusOnShift,
		entities.StatusBusy,
		entities.StatusInactive,
		entities.StatusBlocked,
	}

	for i, driver := range drivers {
		driver.Status = statuses[i]
		_, err := suite.driverService.CreateDriver(suite.ctx, driver)
		require.NoError(suite.T(), err)

		// Обновляем статус после создания
		if statuses[i] != entities.StatusRegistered {
			err = suite.driverService.ChangeDriverStatus(suite.ctx, driver.ID, statuses[i])
			require.NoError(suite.T(), err)
		}
	}

	// Act
	req := httptest.NewRequest(http.MethodGet, "/api/v1/drivers/active", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	drivers_list := response["drivers"].([]interface{})
	count := response["count"].(float64)

	assert.Equal(suite.T(), float64(3), count) // available, on_shift, busy
	assert.Len(suite.T(), drivers_list, 3)
}

// TestDriverAPIFilters тестирует фильтрацию водителей через API
func (suite *DriverAPITestSuite) TestDriverAPIFilters() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(4)

	// Устанавливаем разные рейтинги
	ratings := []float64{3.5, 4.2, 4.8, 2.1}

	for i, driver := range drivers {
		driver.CurrentRating = ratings[i]
		_, err := suite.driverService.CreateDriver(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act - фильтр по минимальному рейтингу
	req := httptest.NewRequest(http.MethodGet, "/api/v1/drivers?min_rating=4.0", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response httpHandlers.ListDriversResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), response.Drivers, 2) // Рейтинги 4.2 и 4.8

	for _, driver := range response.Drivers {
		assert.GreaterOrEqual(suite.T(), driver.CurrentRating, 4.0)
	}
}

// TestDriverAPIPagination тестирует пагинацию через API
func (suite *DriverAPITestSuite) TestDriverAPIPagination() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(7)
	for _, driver := range drivers {
		_, err := suite.driverService.CreateDriver(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act - первая страница
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/drivers?limit=3&offset=0", nil)
	w1 := httptest.NewRecorder()
	suite.router.ServeHTTP(w1, req1)

	// Act - вторая страница
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/drivers?limit=3&offset=3", nil)
	w2 := httptest.NewRecorder()
	suite.router.ServeHTTP(w2, req2)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w1.Code)
	assert.Equal(suite.T(), http.StatusOK, w2.Code)

	var page1, page2 httpHandlers.ListDriversResponse
	err := json.Unmarshal(w1.Body.Bytes(), &page1)
	require.NoError(suite.T(), err)
	err = json.Unmarshal(w2.Body.Bytes(), &page2)
	require.NoError(suite.T(), err)

	assert.Len(suite.T(), page1.Drivers, 3)
	assert.Len(suite.T(), page2.Drivers, 3)
	assert.Equal(suite.T(), 7, page1.Total)
	assert.Equal(suite.T(), 7, page2.Total)
	assert.True(suite.T(), page1.HasMore)
	assert.True(suite.T(), page2.HasMore)
}

// TestDriverAPIInvalidID тестирует обработку невалидного ID
func (suite *DriverAPITestSuite) TestDriverAPIInvalidID() {
	// Act
	req := httptest.NewRequest(http.MethodGet, "/api/v1/drivers/invalid-uuid", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var errorResponse httpHandlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Contains(suite.T(), errorResponse.Error, "Invalid driver ID format")
}

// TestDriverAPIDuplicateCreation тестирует создание дублирующего водителя
func (suite *DriverAPITestSuite) TestDriverAPIDuplicateCreation() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	_, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Пытаемся создать водителя с тем же телефоном
	requestBody := map[string]interface{}{
		"phone":           driver.Phone, // Дублируем телефон
		"email":           "another@example.com",
		"first_name":      "Другой",
		"last_name":       "Водитель",
		"birth_date":      "1990-01-01T00:00:00Z",
		"passport_series": "5678",
		"passport_number": "123456",
		"license_number":  "ANOTHER123",
		"license_expiry":  "2026-12-31T00:00:00Z",
	}

	// Act
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/drivers", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusConflict, w.Code)

	var errorResponse httpHandlers.ErrorResponse
	err = json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "DRIVER_EXISTS", errorResponse.Code)
}

// TestHealthCheckAPI тестирует health check endpoint
func (suite *DriverAPITestSuite) TestHealthCheckAPI() {
	// Act
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "healthy", response["status"])
	assert.Equal(suite.T(), "driver-service", response["service"])
	assert.NotNil(suite.T(), response["timestamp"])
}

// TestCORSHeaders тестирует CORS заголовки
func (suite *DriverAPITestSuite) TestCORSHeaders() {
	// Act - OPTIONS запрос
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/drivers", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Assert
	assert.Equal(suite.T(), http.StatusNoContent, w.Code)
	assert.Equal(suite.T(), "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(suite.T(), w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(suite.T(), w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
}

// TestRequestIDMiddleware тестирует middleware для Request ID
func (suite *DriverAPITestSuite) TestRequestIDMiddleware() {
	// Act - без X-Request-ID заголовка
	req1 := httptest.NewRequest(http.MethodGet, "/health", nil)
	w1 := httptest.NewRecorder()
	suite.router.ServeHTTP(w1, req1)

	// Act - с X-Request-ID заголовком
	customRequestID := uuid.New().String()
	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	req2.Header.Set("X-Request-ID", customRequestID)
	w2 := httptest.NewRecorder()
	suite.router.ServeHTTP(w2, req2)

	// Assert
	assert.Equal(suite.T(), http.StatusOK, w1.Code)
	assert.Equal(suite.T(), http.StatusOK, w2.Code)

	// Проверяем, что Request ID установлен
	assert.NotEmpty(suite.T(), w1.Header().Get("X-Request-ID"))
	assert.Equal(suite.T(), customRequestID, w2.Header().Get("X-Request-ID"))
}

// mockEventPublisher заглушка для EventPublisher в тестах
type mockEventPublisher struct {
	logger interface{} // zap.Logger, но не импортируем zap здесь
}

func (m *mockEventPublisher) PublishDriverEvent(ctx context.Context, eventType string, driverID uuid.UUID, data interface{}) error {
	// В тестах просто логируем события
	return nil
}

// Запуск тестового suite
func TestDriverAPITestSuite(t *testing.T) {
	suite.Run(t, new(DriverAPITestSuite))
}
