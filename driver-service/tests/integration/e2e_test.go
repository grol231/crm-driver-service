//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
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

// E2ETestSuite end-to-end тестовый suite
type E2ETestSuite struct {
	suite.Suite
	testDB          *helpers.TestDB
	server          *httpServer.Server
	router          *gin.Engine
	apiHelper       *helpers.APITestHelper
	driverService   services.DriverService
	locationService services.LocationService
	ctx             context.Context
}

// SetupSuite выполняется один раз перед всеми тестами
func (suite *E2ETestSuite) SetupSuite() {
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
	suite.apiHelper = helpers.NewAPITestHelper(suite.router, suite.T())
}

// TearDownSuite выполняется один раз после всех тестов
func (suite *E2ETestSuite) TearDownSuite() {
	suite.testDB.TeardownTestDB(suite.T())
}

// SetupTest выполняется перед каждым тестом
func (suite *E2ETestSuite) SetupTest() {
	suite.testDB.CleanupTables(suite.T())
}

// TestCompleteDriverWorkflow тестирует полный workflow водителя
func (suite *E2ETestSuite) TestCompleteDriverWorkflow() {
	// 1. Регистрация водителя
	suite.T().Log("Step 1: Driver registration")

	driverData := helpers.CreateDriverRequest()
	response := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/drivers",
		Body:   driverData,
	})

	suite.apiHelper.AssertStatusCode(response, http.StatusCreated)

	var createdDriver httpHandlers.DriverResponse
	suite.apiHelper.UnmarshalResponse(response, &createdDriver)

	assert.Equal(suite.T(), entities.StatusRegistered, createdDriver.Status)
	assert.Equal(suite.T(), driverData["phone"], createdDriver.Phone)

	driverID := createdDriver.ID

	// 2. Обновление статуса на pending_verification
	suite.T().Log("Step 2: Status change to pending verification")

	statusResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPatch,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/status", driverID),
		Body:   map[string]string{"status": "pending_verification"},
	})

	suite.apiHelper.AssertStatusCode(statusResponse, http.StatusOK)

	// 3. Добавление документов (через прямое обращение к сервису, так как API для документов не реализован в рамках этого задания)
	suite.T().Log("Step 3: Adding driver documents")

	// Здесь в реальном приложении были бы API вызовы для загрузки документов
	// Пока используем прямое обращение к репозиторию
	documentRepo := repositories.NewDocumentRepository(suite.testDB.DB, helpers.CreateTestLogger(suite.T()))

	licenseDoc := entities.NewDriverDocument(
		driverID,
		entities.DocumentTypeDriverLicense,
		"1234567890",
		time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		"https://example.com/license.pdf",
	)

	err := documentRepo.Create(suite.ctx, licenseDoc)
	require.NoError(suite.T(), err)

	// Верификация документа
	err = documentRepo.UpdateStatus(suite.ctx, licenseDoc.ID, entities.VerificationStatusVerified, stringPtr("admin"), nil)
	require.NoError(suite.T(), err)

	// 4. Переход к verified статусу
	suite.T().Log("Step 4: Status change to verified")

	statusResponse = suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPatch,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/status", driverID),
		Body:   map[string]string{"status": "verified"},
	})

	suite.apiHelper.AssertStatusCode(statusResponse, http.StatusOK)

	// 5. Переход к available статусу
	suite.T().Log("Step 5: Status change to available")

	statusResponse = suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPatch,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/status", driverID),
		Body:   map[string]string{"status": "available"},
	})

	suite.apiHelper.AssertStatusCode(statusResponse, http.StatusOK)

	// 6. Проверяем, что водитель появился в списке активных
	suite.T().Log("Step 6: Check driver in active list")

	activeResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    "/api/v1/drivers/active",
	})

	suite.apiHelper.AssertStatusCode(activeResponse, http.StatusOK)

	var activeDriversResp map[string]interface{}
	suite.apiHelper.UnmarshalResponse(activeResponse, &activeDriversResp)

	drivers := activeDriversResp["drivers"].([]interface{})
	assert.Len(suite.T(), drivers, 1)

	// 7. Обновление местоположения
	suite.T().Log("Step 7: Update driver location")

	locationData := helpers.CreateLocationRequest()
	locationResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPost,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/locations", driverID),
		Body:   locationData,
	})

	suite.apiHelper.AssertStatusCode(locationResponse, http.StatusOK)

	// 8. Получение текущего местоположения
	suite.T().Log("Step 8: Get current location")

	currentLocationResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/locations/current", driverID),
	})

	suite.apiHelper.AssertStatusCode(currentLocationResponse, http.StatusOK)

	var currentLocation httpHandlers.LocationResponse
	suite.apiHelper.UnmarshalResponse(currentLocationResponse, &currentLocation)

	assert.Equal(suite.T(), driverID, currentLocation.DriverID)
	assert.Equal(suite.T(), locationData["latitude"], currentLocation.Latitude)

	// 9. Поиск водителей поблизости
	suite.T().Log("Step 9: Search nearby drivers")

	nearbyResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    "/api/v1/locations/nearby",
		QueryParams: map[string]string{
			"latitude":  fmt.Sprintf("%f", locationData["latitude"]),
			"longitude": fmt.Sprintf("%f", locationData["longitude"]),
			"radius_km": "1",
		},
	})

	suite.apiHelper.AssertStatusCode(nearbyResponse, http.StatusOK)

	var nearbyDrivers httpHandlers.NearbyDriversResponse
	suite.apiHelper.UnmarshalResponse(nearbyResponse, &nearbyDrivers)

	assert.Len(suite.T(), nearbyDrivers.Drivers, 1)
	assert.Equal(suite.T(), driverID, nearbyDrivers.Drivers[0].DriverID)

	// 10. Обновление профиля водителя
	suite.T().Log("Step 10: Update driver profile")

	updateData := map[string]interface{}{
		"first_name": "Обновленное Имя",
		"email":      "updated@example.com",
	}

	updateResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPut,
		URL:    fmt.Sprintf("/api/v1/drivers/%s", driverID),
		Body:   updateData,
	})

	suite.apiHelper.AssertStatusCode(updateResponse, http.StatusOK)

	var updatedDriver httpHandlers.DriverResponse
	suite.apiHelper.UnmarshalResponse(updateResponse, &updatedDriver)

	assert.Equal(suite.T(), "Обновленное Имя", updatedDriver.FirstName)
	assert.Equal(suite.T(), "updated@example.com", updatedDriver.Email)

	suite.T().Log("Complete driver workflow test passed successfully!")
}

// TestDriverLocationTracking тестирует отслеживание местоположения водителя
func (suite *E2ETestSuite) TestDriverLocationTracking() {
	// 1. Создаем водителя
	driverData := helpers.CreateDriverRequest()
	response := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/drivers",
		Body:   driverData,
	})

	suite.apiHelper.AssertStatusCode(response, http.StatusCreated)

	var driver httpHandlers.DriverResponse
	suite.apiHelper.UnmarshalResponse(response, &driver)

	// 2. Отправляем серию обновлений местоположения (имитация поездки)
	suite.T().Log("Simulating trip with location updates")

	// Маршрут: от Красной площади до Большого театра
	route := []map[string]interface{}{
		{"latitude": 55.7558, "longitude": 37.6173, "speed": 0.0},  // Старт
		{"latitude": 55.7580, "longitude": 37.6180, "speed": 25.0}, // Начало движения
		{"latitude": 55.7600, "longitude": 37.6190, "speed": 40.0}, // Ускорение
		{"latitude": 55.7620, "longitude": 37.6200, "speed": 35.0}, // Поворот
		{"latitude": 55.7640, "longitude": 37.6210, "speed": 30.0}, // Замедление
		{"latitude": 55.7655, "longitude": 37.6220, "speed": 0.0},  // Остановка
	}

	for i, point := range route {
		locationResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
			Method: http.MethodPost,
			URL:    fmt.Sprintf("/api/v1/drivers/%s/locations", driver.ID),
			Body:   point,
		})

		suite.apiHelper.AssertStatusCode(locationResponse, http.StatusOK)

		// Небольшая пауза между обновлениями
		if i < len(route)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// 3. Получаем историю местоположений
	suite.T().Log("Getting location history")

	from := time.Now().Add(-1 * time.Hour).Unix()
	to := time.Now().Unix()

	historyResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/locations/history", driver.ID),
		QueryParams: map[string]string{
			"from": fmt.Sprintf("%d", from),
			"to":   fmt.Sprintf("%d", to),
		},
	})

	suite.apiHelper.AssertStatusCode(historyResponse, http.StatusOK)

	var history httpHandlers.LocationHistoryResponse
	suite.apiHelper.UnmarshalResponse(historyResponse, &history)

	assert.Len(suite.T(), history.Locations, len(route))
	assert.NotNil(suite.T(), history.Stats)
	assert.Greater(suite.T(), history.Stats.DistanceTraveled, 0.0)
	assert.Equal(suite.T(), 40.0, history.Stats.MaxSpeed)

	// 4. Проверяем текущее местоположение
	suite.T().Log("Checking current location")

	currentResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/locations/current", driver.ID),
	})

	suite.apiHelper.AssertStatusCode(currentResponse, http.StatusOK)

	var currentLocation httpHandlers.LocationResponse
	suite.apiHelper.UnmarshalResponse(currentResponse, &currentLocation)

	// Текущее местоположение должно быть последней точкой маршрута
	lastPoint := route[len(route)-1]
	assert.Equal(suite.T(), lastPoint["latitude"], currentLocation.Latitude)
	assert.Equal(suite.T(), lastPoint["longitude"], currentLocation.Longitude)
}

// TestMultipleDriversScenario тестирует сценарий с несколькими водителями
func (suite *E2ETestSuite) TestMultipleDriversScenario() {
	// 1. Создаем несколько водителей
	suite.T().Log("Creating multiple drivers")

	var driverIDs []uuid.UUID

	for i := 0; i < 5; i++ {
		driverData := helpers.CreateDriverRequest()
		driverData["phone"] = fmt.Sprintf("+7900123456%d", i)
		driverData["email"] = fmt.Sprintf("driver%d@example.com", i)
		driverData["license_number"] = fmt.Sprintf("TEST12345%d", i)

		response := suite.apiHelper.MakeRequest(helpers.APIRequest{
			Method: http.MethodPost,
			URL:    "/api/v1/drivers",
			Body:   driverData,
		})

		suite.apiHelper.AssertStatusCode(response, http.StatusCreated)

		var driver httpHandlers.DriverResponse
		suite.apiHelper.UnmarshalResponse(response, &driver)
		driverIDs = append(driverIDs, driver.ID)
	}

	// 2. Переводим водителей в available статус
	suite.T().Log("Changing drivers status to available")

	for _, driverID := range driverIDs {
		// Сначала pending_verification
		statusResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/v1/drivers/%s/status", driverID),
			Body:   map[string]string{"status": "pending_verification"},
		})
		suite.apiHelper.AssertStatusCode(statusResponse, http.StatusOK)

		// Затем verified
		statusResponse = suite.apiHelper.MakeRequest(helpers.APIRequest{
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/v1/drivers/%s/status", driverID),
			Body:   map[string]string{"status": "verified"},
		})
		suite.apiHelper.AssertStatusCode(statusResponse, http.StatusOK)

		// Наконец available
		statusResponse = suite.apiHelper.MakeRequest(helpers.APIRequest{
			Method: http.MethodPatch,
			URL:    fmt.Sprintf("/api/v1/drivers/%s/status", driverID),
			Body:   map[string]string{"status": "available"},
		})
		suite.apiHelper.AssertStatusCode(statusResponse, http.StatusOK)
	}

	// 3. Добавляем местоположения для всех водителей
	suite.T().Log("Adding locations for all drivers")

	centerLat := 55.7558
	centerLon := 37.6173

	for i, driverID := range driverIDs {
		locationData := map[string]interface{}{
			"latitude":  centerLat + float64(i)*0.01, // Распределяем водителей
			"longitude": centerLon + float64(i)*0.01,
			"speed":     float64(30 + i*5),
		}

		locationResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
			Method: http.MethodPost,
			URL:    fmt.Sprintf("/api/v1/drivers/%s/locations", driverID),
			Body:   locationData,
		})

		suite.apiHelper.AssertStatusCode(locationResponse, http.StatusOK)
	}

	// 4. Проверяем список активных водителей
	suite.T().Log("Checking active drivers list")

	activeResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    "/api/v1/drivers/active",
	})

	suite.apiHelper.AssertStatusCode(activeResponse, http.StatusOK)

	var activeDriversResp map[string]interface{}
	suite.apiHelper.UnmarshalResponse(activeResponse, &activeDriversResp)

	drivers := activeDriversResp["drivers"].([]interface{})
	assert.Len(suite.T(), drivers, 5)

	// 5. Тестируем поиск водителей поблизости
	suite.T().Log("Testing nearby drivers search")

	nearbyResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    "/api/v1/locations/nearby",
		QueryParams: map[string]string{
			"latitude":  fmt.Sprintf("%f", centerLat),
			"longitude": fmt.Sprintf("%f", centerLon),
			"radius_km": "5",
			"limit":     "10",
		},
	})

	suite.apiHelper.AssertStatusCode(nearbyResponse, http.StatusOK)

	var nearbyDrivers httpHandlers.NearbyDriversResponse
	suite.apiHelper.UnmarshalResponse(nearbyResponse, &nearbyDrivers)

	assert.Greater(suite.T(), nearbyDrivers.Count, 0)
	assert.LessOrEqual(suite.T(), nearbyDrivers.Count, 5)

	// 6. Тестируем пагинацию списка водителей
	suite.T().Log("Testing drivers list pagination")

	// Первая страница
	page1Response := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    "/api/v1/drivers",
		QueryParams: map[string]string{
			"limit":  "3",
			"offset": "0",
		},
	})

	suite.apiHelper.AssertStatusCode(page1Response, http.StatusOK)

	var page1 httpHandlers.ListDriversResponse
	suite.apiHelper.UnmarshalResponse(page1Response, &page1)

	assert.Len(suite.T(), page1.Drivers, 3)
	assert.Equal(suite.T(), 5, page1.Total)
	assert.True(suite.T(), page1.HasMore)

	// Вторая страница
	page2Response := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    "/api/v1/drivers",
		QueryParams: map[string]string{
			"limit":  "3",
			"offset": "3",
		},
	})

	suite.apiHelper.AssertStatusCode(page2Response, http.StatusOK)

	var page2 httpHandlers.ListDriversResponse
	suite.apiHelper.UnmarshalResponse(page2Response, &page2)

	assert.Len(suite.T(), page2.Drivers, 2)
	assert.Equal(suite.T(), 5, page2.Total)
	assert.False(suite.T(), page2.HasMore)
}

// TestErrorHandlingScenarios тестирует различные сценарии обработки ошибок
func (suite *E2ETestSuite) TestErrorHandlingScenarios() {
	// 1. Невалидные данные при создании
	suite.T().Log("Testing invalid data handling")

	invalidDriverData := map[string]interface{}{
		"phone":      "invalid-phone",
		"email":      "invalid-email",
		"first_name": "",
	}

	response := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/drivers",
		Body:   invalidDriverData,
	})

	suite.apiHelper.AssertStatusCode(response, http.StatusBadRequest)

	// 2. Несуществующий водитель
	suite.T().Log("Testing non-existent driver")

	nonExistentID := uuid.New()
	response = suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("/api/v1/drivers/%s", nonExistentID),
	})

	suite.apiHelper.AssertStatusCode(response, http.StatusNotFound)
	suite.apiHelper.AssertErrorResponse(response, "DRIVER_NOT_FOUND")

	// 3. Невалидные координаты
	suite.T().Log("Testing invalid coordinates")

	// Сначала создаем валидного водителя
	validDriverData := helpers.CreateDriverRequest()
	driverResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPost,
		URL:    "/api/v1/drivers",
		Body:   validDriverData,
	})

	var driver httpHandlers.DriverResponse
	suite.apiHelper.UnmarshalResponse(driverResponse, &driver)

	// Отправляем невалидные координаты
	invalidLocationData := map[string]interface{}{
		"latitude":  91.0, // Больше 90
		"longitude": 37.6173,
	}

	locationResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPost,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/locations", driver.ID),
		Body:   invalidLocationData,
	})

	suite.apiHelper.AssertStatusCode(locationResponse, http.StatusBadRequest)

	// 4. Невалидный переход статуса
	suite.T().Log("Testing invalid status transition")

	// Пытаемся перейти напрямую от registered к available (недопустимо)
	statusResponse := suite.apiHelper.MakeRequest(helpers.APIRequest{
		Method: http.MethodPatch,
		URL:    fmt.Sprintf("/api/v1/drivers/%s/status", driver.ID),
		Body:   map[string]string{"status": "available"},
	})

	suite.apiHelper.AssertStatusCode(statusResponse, http.StatusInternalServerError) // Или другой код ошибки в зависимости от реализации
}

// TestConcurrentAPIRequests тестирует конкурентные API запросы
func (suite *E2ETestSuite) TestConcurrentAPIRequests() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	createdDriver, err := suite.driverService.CreateDriver(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act - одновременные запросы разных типов
	const concurrency = 20
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(requestID int) {
			var err error

			switch requestID % 4 {
			case 0:
				// GET запрос
				response := suite.apiHelper.MakeRequest(helpers.APIRequest{
					Method: http.MethodGet,
					URL:    fmt.Sprintf("/api/v1/drivers/%s", createdDriver.ID),
				})
				if response.StatusCode != http.StatusOK {
					err = fmt.Errorf("GET request failed with status %d", response.StatusCode)
				}

			case 1:
				// Обновление местоположения
				locationData := map[string]interface{}{
					"latitude":  55.7558 + float64(requestID)*0.0001,
					"longitude": 37.6173 + float64(requestID)*0.0001,
				}
				response := suite.apiHelper.MakeRequest(helpers.APIRequest{
					Method: http.MethodPost,
					URL:    fmt.Sprintf("/api/v1/drivers/%s/locations", createdDriver.ID),
					Body:   locationData,
				})
				if response.StatusCode != http.StatusOK {
					err = fmt.Errorf("Location update failed with status %d", response.StatusCode)
				}

			case 2:
				// Получение текущего местоположения
				response := suite.apiHelper.MakeRequest(helpers.APIRequest{
					Method: http.MethodGet,
					URL:    fmt.Sprintf("/api/v1/drivers/%s/locations/current", createdDriver.ID),
				})
				if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
					err = fmt.Errorf("Get current location failed with status %d", response.StatusCode)
				}

			case 3:
				// Список водителей
				response := suite.apiHelper.MakeRequest(helpers.APIRequest{
					Method:      http.MethodGet,
					URL:         "/api/v1/drivers",
					QueryParams: map[string]string{"limit": "10"},
				})
				if response.StatusCode != http.StatusOK {
					err = fmt.Errorf("List drivers failed with status %d", response.StatusCode)
				}
			}

			done <- err
		}(i)
	}

	// Ждем завершения всех запросов
	errors := 0
	for i := 0; i < concurrency; i++ {
		if err := <-done; err != nil {
			errors++
			suite.T().Logf("Concurrent request error: %v", err)
		}
	}

	// Assert
	errorRate := float64(errors) / float64(concurrency)
	assert.Less(suite.T(), errorRate, 0.1, "Error rate should be less than 10%% for concurrent requests")
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

// Запуск тестового suite
func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
