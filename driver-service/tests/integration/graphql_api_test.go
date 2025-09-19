package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"driver-service/internal/config"
	"driver-service/internal/domain/entities"
	"driver-service/internal/domain/services"
	"driver-service/internal/interfaces/graphql/resolver"
	httpserver "driver-service/internal/interfaces/http"
	"driver-service/internal/interfaces/http/handlers"
	"driver-service/internal/repositories"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// GraphQLAPITestSuite тестовый набор для GraphQL API
type GraphQLAPITestSuite struct {
	suite.Suite
	server    *httpserver.Server
	logger    *zap.Logger
	cfg       *config.Config
	
	// Моки сервисов и репозиториев для тестирования
	driverService   *mockDriverServiceIntegration
	locationService *mockLocationServiceIntegration
	driverRepo      *mockDriverRepoIntegration
	locationRepo    *mockLocationRepoIntegration
	ratingRepo      *mockRatingRepoIntegration
	shiftRepo       *mockShiftRepoIntegration
	documentRepo    *mockDocumentRepoIntegration
}

// SetupSuite настройка тестового набора
func (suite *GraphQLAPITestSuite) SetupSuite() {
	// Настройка логера
	suite.logger = zap.NewNop()
	
	// Настройка конфигурации
	suite.cfg = &config.Config{
		Server: config.ServerConfig{
			HTTPPort:    8080,
			Environment: "test",
			Timeout:     30 * time.Second,
		},
	}
	
	// Переключаем Gin в тестовый режим
	gin.SetMode(gin.TestMode)
	
	// Создаем моки
	suite.driverService = newMockDriverServiceIntegration()
	suite.locationService = newMockLocationServiceIntegration()
	suite.driverRepo = newMockDriverRepoIntegration()
	suite.locationRepo = newMockLocationRepoIntegration()
	suite.ratingRepo = newMockRatingRepoIntegration()
	suite.shiftRepo = newMockShiftRepoIntegration()
	suite.documentRepo = newMockDocumentRepoIntegration()
	
	// Создаем handlers
	driverHandler := handlers.NewDriverHandler(suite.driverService, suite.logger)
	locationHandler := handlers.NewLocationHandler(suite.locationService, suite.logger)
	
	// Создаем GraphQL resolver
	graphqlResolver := resolver.NewResolver(
		suite.driverService,
		suite.locationService,
		suite.driverRepo,
		suite.locationRepo,
		suite.ratingRepo,
		suite.shiftRepo,
		suite.documentRepo,
		suite.logger,
	)
	
	// Создаем HTTP сервер
	suite.server = httpserver.NewServer(
		suite.cfg,
		suite.logger,
		driverHandler,
		locationHandler,
		graphqlResolver,
	)
}

// TestGraphQLQuery_Driver тестирует GraphQL запрос водителя
func (suite *GraphQLAPITestSuite) TestGraphQLQuery_Driver() {
	// Подготавливаем тестовые данные
	testDriver := createTestDriverEntity()
	suite.driverService.drivers[testDriver.ID] = testDriver
	
	// GraphQL запрос
	query := `
		query GetDriver($id: UUID!) {
			driver(id: $id) {
				id
				phone
				email
				firstName
				lastName
				fullName
				status
				currentRating
				totalTrips
				isActive
				canReceiveOrders
				isLicenseExpired
			}
		}
	`
	
	variables := map[string]interface{}{
		"id": testDriver.ID.String(),
	}
	
	// Выполняем запрос
	response := suite.executeGraphQLQuery(query, variables)
	
	// Проверяем ответ
	suite.Assert().Equal(http.StatusOK, response.Code)
	
	var result map[string]interface{}
	err := json.Unmarshal(response.Body.Bytes(), &result)
	suite.Assert().NoError(err)
	
	// Проверяем данные
	data := result["data"].(map[string]interface{})
	driver := data["driver"].(map[string]interface{})
	
	suite.Assert().Equal(testDriver.ID.String(), driver["id"])
	suite.Assert().Equal(testDriver.Phone, driver["phone"])
	suite.Assert().Equal(testDriver.Email, driver["email"])
	suite.Assert().Equal(testDriver.FirstName, driver["firstName"])
	suite.Assert().Equal(testDriver.LastName, driver["lastName"])
	suite.Assert().Equal("Doe John", driver["fullName"])
	suite.Assert().Equal("AVAILABLE", driver["status"])
	suite.Assert().Equal(testDriver.CurrentRating, driver["currentRating"])
	suite.Assert().Equal(float64(testDriver.TotalTrips), driver["totalTrips"])
	suite.Assert().True(driver["isActive"].(bool))
	suite.Assert().True(driver["canReceiveOrders"].(bool))
	suite.Assert().False(driver["isLicenseExpired"].(bool))
}

// TestGraphQLQuery_Drivers тестирует GraphQL запрос списка водителей
func (suite *GraphQLAPITestSuite) TestGraphQLQuery_Drivers() {
	// Подготавливаем тестовые данные
	testDrivers := []*entities.Driver{
		createTestDriverEntity(),
		createTestDriverEntity(),
	}
	
	for _, driver := range testDrivers {
		suite.driverService.drivers[driver.ID] = driver
	}
	
	// GraphQL запрос
	query := `
		query GetDrivers($limit: Int, $offset: Int) {
			drivers(limit: $limit, offset: $offset) {
				drivers {
					id
					phone
					firstName
					lastName
				}
				pageInfo {
					total
					limit
					offset
					hasMore
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"limit":  10,
		"offset": 0,
	}
	
	// Выполняем запрос
	response := suite.executeGraphQLQuery(query, variables)
	
	// Проверяем ответ
	suite.Assert().Equal(http.StatusOK, response.Code)
	
	var result map[string]interface{}
	err := json.Unmarshal(response.Body.Bytes(), &result)
	suite.Assert().NoError(err)
	
	// Проверяем данные
	data := result["data"].(map[string]interface{})
	driversResult := data["drivers"].(map[string]interface{})
	drivers := driversResult["drivers"].([]interface{})
	pageInfo := driversResult["pageInfo"].(map[string]interface{})
	
	suite.Assert().Len(drivers, 2)
	suite.Assert().Equal(float64(2), pageInfo["total"])
	suite.Assert().Equal(float64(10), pageInfo["limit"])
	suite.Assert().Equal(float64(0), pageInfo["offset"])
	suite.Assert().False(pageInfo["hasMore"].(bool))
}

// TestGraphQLMutation_CreateDriver тестирует GraphQL мутацию создания водителя
func (suite *GraphQLAPITestSuite) TestGraphQLMutation_CreateDriver() {
	// GraphQL мутация
	mutation := `
		mutation CreateDriver($input: CreateDriverInput!) {
			createDriver(input: $input) {
				id
				phone
				email
				firstName
				lastName
				status
			}
		}
	`
	
	now := time.Now()
	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"phone":          "+1234567890",
			"email":          "newdriver@example.com",
			"firstName":      "Jane",
			"lastName":       "Smith",
			"birthDate":      now.AddDate(-25, 0, 0).Format(time.RFC3339),
			"passportSeries": "5678",
			"passportNumber": "123456",
			"licenseNumber":  "DL987654",
			"licenseExpiry":  now.AddDate(3, 0, 0).Format(time.RFC3339),
		},
	}
	
	// Выполняем запрос
	response := suite.executeGraphQLQuery(mutation, variables)
	
	// Проверяем ответ
	suite.Assert().Equal(http.StatusOK, response.Code)
	
	var result map[string]interface{}
	err := json.Unmarshal(response.Body.Bytes(), &result)
	suite.Assert().NoError(err)
	
	// Проверяем данные
	data := result["data"].(map[string]interface{})
	driver := data["createDriver"].(map[string]interface{})
	
	suite.Assert().NotEmpty(driver["id"])
	suite.Assert().Equal("+1234567890", driver["phone"])
	suite.Assert().Equal("newdriver@example.com", driver["email"])
	suite.Assert().Equal("Jane", driver["firstName"])
	suite.Assert().Equal("Smith", driver["lastName"])
	suite.Assert().Equal("REGISTERED", driver["status"])
}

// TestGraphQLMutation_UpdateDriverLocation тестирует GraphQL мутацию обновления местоположения
func (suite *GraphQLAPITestSuite) TestGraphQLMutation_UpdateDriverLocation() {
	// Подготавливаем тестовые данные
	testDriver := createTestDriverEntity()
	suite.driverService.drivers[testDriver.ID] = testDriver
	
	// GraphQL мутация
	mutation := `
		mutation UpdateDriverLocation($driverId: UUID!, $input: LocationUpdateInput!) {
			updateDriverLocation(driverId: $driverId, input: $input) {
				id
				driverId
				latitude
				longitude
				speed
				bearing
				isValidLocation
			}
		}
	`
	
	variables := map[string]interface{}{
		"driverId": testDriver.ID.String(),
		"input": map[string]interface{}{
			"latitude":  55.7558,
			"longitude": 37.6176,
			"speed":     60.0,
			"bearing":   180.0,
		},
	}
	
	// Выполняем запрос
	response := suite.executeGraphQLQuery(mutation, variables)
	
	// Проверяем ответ
	suite.Assert().Equal(http.StatusOK, response.Code)
	
	var result map[string]interface{}
	err := json.Unmarshal(response.Body.Bytes(), &result)
	suite.Assert().NoError(err)
	
	// Проверяем данные
	data := result["data"].(map[string]interface{})
	location := data["updateDriverLocation"].(map[string]interface{})
	
	suite.Assert().NotEmpty(location["id"])
	suite.Assert().Equal(testDriver.ID.String(), location["driverId"])
	suite.Assert().Equal(55.7558, location["latitude"])
	suite.Assert().Equal(37.6176, location["longitude"])
	suite.Assert().Equal(60.0, location["speed"])
	suite.Assert().Equal(180.0, location["bearing"])
	suite.Assert().True(location["isValidLocation"].(bool))
}

// TestGraphQLError_DriverNotFound тестирует обработку ошибок
func (suite *GraphQLAPITestSuite) TestGraphQLError_DriverNotFound() {
	// GraphQL запрос несуществующего водителя
	query := `
		query GetDriver($id: UUID!) {
			driver(id: $id) {
				id
				phone
			}
		}
	`
	
	variables := map[string]interface{}{
		"id": uuid.New().String(),
	}
	
	// Выполняем запрос
	response := suite.executeGraphQLQuery(query, variables)
	
	// Проверяем ответ
	suite.Assert().Equal(http.StatusOK, response.Code)
	
	var result map[string]interface{}
	err := json.Unmarshal(response.Body.Bytes(), &result)
	suite.Assert().NoError(err)
	
	// Проверяем, что есть ошибки
	errors := result["errors"].([]interface{})
	suite.Assert().Len(errors, 1)
	
	firstError := errors[0].(map[string]interface{})
	suite.Assert().Contains(firstError["message"], "driver not found")
}

// executeGraphQLQuery выполняет GraphQL запрос
func (suite *GraphQLAPITestSuite) executeGraphQLQuery(query string, variables map[string]interface{}) *httptest.ResponseRecorder {
	// Подготавливаем тело запроса
	requestBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	
	jsonBody, _ := json.Marshal(requestBody)
	
	// Создаем HTTP запрос
	req, _ := http.NewRequest("POST", "/graphql", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	// Выполняем запрос
	recorder := httptest.NewRecorder()
	suite.server.GetRouter().ServeHTTP(recorder, req)
	
	return recorder
}

// Запуск тестового набора
func TestGraphQLAPITestSuite(t *testing.T) {
	suite.Run(t, new(GraphQLAPITestSuite))
}

// Вспомогательные функции и моки для интеграционных тестов

func createTestDriverEntity() *entities.Driver {
	now := time.Now()
	return &entities.Driver{
		ID:              uuid.New(),
		Phone:           "+1234567890",
		Email:           "test@example.com",
		FirstName:       "John",
		LastName:        "Doe",
		BirthDate:       now.AddDate(-30, 0, 0),
		PassportSeries:  "1234",
		PassportNumber:  "567890",
		LicenseNumber:   "DL123456",
		LicenseExpiry:   now.AddDate(5, 0, 0),
		Status:          entities.StatusAvailable,
		CurrentRating:   4.5,
		TotalTrips:      10,
		Metadata:        make(entities.Metadata),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}