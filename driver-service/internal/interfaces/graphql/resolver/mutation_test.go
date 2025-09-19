package resolver

import (
	"context"
	"testing"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/interfaces/graphql/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// createMutationResolver создает mutation resolver с моками для тестирования
func createMutationResolver() (*mutationResolver, *mockDriverService, *mockLocationService, *mockLocationRepository, *mockRatingRepository, *mockShiftRepository) {
	mockDriverSvc := new(mockDriverService)
	mockLocationSvc := new(mockLocationService)
	mockDriverRepo := new(mockDriverRepository)
	mockLocationRepo := new(mockLocationRepository)
	mockRatingRepo := new(mockRatingRepository)
	mockShiftRepo := new(mockShiftRepository)
	mockDocumentRepo := new(mockDocumentRepository)
	
	logger := zap.NewNop()
	
	resolver := &Resolver{
		driverService:   mockDriverSvc,
		locationService: mockLocationSvc,
		driverRepo:      mockDriverRepo,
		locationRepo:    mockLocationRepo,
		ratingRepo:      mockRatingRepo,
		shiftRepo:       mockShiftRepo,
		documentRepo:    mockDocumentRepo,
		logger:          logger,
	}
	
	mutationRes := &mutationResolver{resolver}
	
	return mutationRes, mockDriverSvc, mockLocationSvc, mockLocationRepo, mockRatingRepo, mockShiftRepo
}

func TestMutationResolver_CreateDriver(t *testing.T) {
	mutationRes, mockDriverSvc, _, _, _, _ := createMutationResolver()
	
	// Подготавливаем тестовые данные
	now := time.Now()
	input := model.CreateDriverInput{
		Phone:          "+1234567890",
		Email:          "test@example.com",
		FirstName:      "John",
		LastName:       "Doe",
		BirthDate:      now.AddDate(-30, 0, 0),
		PassportSeries: "1234",
		PassportNumber: "567890",
		LicenseNumber:  "DL123456",
		LicenseExpiry:  now.AddDate(5, 0, 0),
	}
	
	createdDriver := &entities.Driver{
		ID:              uuid.New(),
		Phone:           input.Phone,
		Email:           input.Email,
		FirstName:       input.FirstName,
		LastName:        input.LastName,
		BirthDate:       input.BirthDate,
		PassportSeries:  input.PassportSeries,
		PassportNumber:  input.PassportNumber,
		LicenseNumber:   input.LicenseNumber,
		LicenseExpiry:   input.LicenseExpiry,
		Status:          entities.StatusRegistered,
		CurrentRating:   0.0,
		TotalTrips:      0,
		Metadata:        make(entities.Metadata),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	
	// Настраиваем мок
	mockDriverSvc.On("CreateDriver", mock.Anything, mock.AnythingOfType("*entities.Driver")).Return(createdDriver, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.CreateDriver(ctx, input)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, createdDriver.ID, result.ID)
	assert.Equal(t, input.Phone, result.Phone)
	assert.Equal(t, input.Email, result.Email)
	assert.Equal(t, input.FirstName, result.FirstName)
	assert.Equal(t, input.LastName, result.LastName)
	
	// Проверяем, что мок был вызван
	mockDriverSvc.AssertExpectations(t)
}

func TestMutationResolver_CreateDriver_ValidationError(t *testing.T) {
	mutationRes, mockDriverSvc, _, _, _, _ := createMutationResolver()
	
	// Подготавливаем невалидные тестовые данные
	input := model.CreateDriverInput{
		Phone:     "", // пустой телефон должен вызвать ошибку валидации
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}
	
	// Настраиваем мок для возврата ошибки валидации
	mockDriverSvc.On("CreateDriver", mock.Anything, mock.AnythingOfType("*entities.Driver")).Return(nil, entities.ErrInvalidPhone)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.CreateDriver(ctx, input)
	
	// Проверяем результат
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, entities.ErrInvalidPhone, err)
	
	// Проверяем, что мок был вызван
	mockDriverSvc.AssertExpectations(t)
}

func TestMutationResolver_UpdateDriver(t *testing.T) {
	mutationRes, mockDriverSvc, _, _, _, _ := createMutationResolver()
	
	// Подготавливаем тестовые данные
	existingDriver := createTestDriver()
	driverID := existingDriver.ID
	
	newEmail := "updated@example.com"
	input := model.UpdateDriverInput{
		Email: &newEmail,
	}
	
	updatedDriver := *existingDriver
	updatedDriver.Email = newEmail
	
	// Настраиваем моки
	mockDriverSvc.On("GetDriverByID", mock.Anything, driverID).Return(existingDriver, nil)
	mockDriverSvc.On("UpdateDriver", mock.Anything, mock.AnythingOfType("*entities.Driver")).Return(&updatedDriver, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.UpdateDriver(ctx, driverID, input)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, driverID, result.ID)
	assert.Equal(t, newEmail, result.Email)
	
	// Проверяем, что моки были вызваны
	mockDriverSvc.AssertExpectations(t)
}

func TestMutationResolver_DeleteDriver(t *testing.T) {
	mutationRes, mockDriverSvc, _, _, _, _ := createMutationResolver()
	
	// Подготавливаем тестовые данные
	driverID := uuid.New()
	
	// Настраиваем мок
	mockDriverSvc.On("DeleteDriver", mock.Anything, driverID).Return(nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.DeleteDriver(ctx, driverID)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.True(t, result)
	
	// Проверяем, что мок был вызван
	mockDriverSvc.AssertExpectations(t)
}

func TestMutationResolver_ChangeDriverStatus(t *testing.T) {
	mutationRes, mockDriverSvc, _, _, _, _ := createMutationResolver()
	
	// Подготавливаем тестовые данные
	testDriver := createTestDriver()
	driverID := testDriver.ID
	newStatus := model.StatusBusy
	
	updatedDriver := *testDriver
	updatedDriver.Status = entities.StatusBusy
	
	// Настраиваем моки
	mockDriverSvc.On("ChangeDriverStatus", mock.Anything, driverID, entities.StatusBusy).Return(nil)
	mockDriverSvc.On("GetDriverByID", mock.Anything, driverID).Return(&updatedDriver, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.ChangeDriverStatus(ctx, driverID, newStatus)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, driverID, result.ID)
	assert.Equal(t, model.StatusBusy, result.Status)
	
	// Проверяем, что моки были вызваны
	mockDriverSvc.AssertExpectations(t)
}

func TestMutationResolver_UpdateDriverLocation(t *testing.T) {
	mutationRes, _, mockLocationSvc, _, _, _ := createMutationResolver()
	
	// Подготавливаем тестовые данные
	driverID := uuid.New()
	input := model.LocationUpdateInput{
		Latitude:  55.7558,
		Longitude: 37.6176,
	}
	
	// Настраиваем мок
	mockLocationSvc.On("UpdateLocation", mock.Anything, mock.AnythingOfType("*entities.DriverLocation")).Return(nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.UpdateDriverLocation(ctx, driverID, input)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, driverID, result.DriverID)
	assert.Equal(t, input.Latitude, result.Latitude)
	assert.Equal(t, input.Longitude, result.Longitude)
	
	// Проверяем, что мок был вызван
	mockLocationSvc.AssertExpectations(t)
}

func TestMutationResolver_BatchUpdateDriverLocations(t *testing.T) {
	mutationRes, _, mockLocationSvc, _, _, _ := createMutationResolver()
	
	// Подготавливаем тестовые данные
	driverID := uuid.New()
	locations := []model.LocationUpdateInput{
		{Latitude: 55.7558, Longitude: 37.6176},
		{Latitude: 55.7600, Longitude: 37.6200},
	}
	
	// Настраиваем мок
	mockLocationSvc.On("BatchUpdateLocations", mock.Anything, mock.AnythingOfType("[]*entities.DriverLocation")).Return(nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.BatchUpdateDriverLocations(ctx, driverID, locations)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, driverID, result[0].DriverID)
	assert.Equal(t, locations[0].Latitude, result[0].Latitude)
	assert.Equal(t, locations[0].Longitude, result[0].Longitude)
	
	// Проверяем, что мок был вызван
	mockLocationSvc.AssertExpectations(t)
}

func TestMutationResolver_AddDriverRating(t *testing.T) {
	mutationRes, mockDriverSvc, _, _, mockRatingRepo, _ := createMutationResolver()
	
	// Подготавливаем тестовые данные
	driverID := uuid.New()
	orderID := uuid.New()
	customerID := uuid.New()
	input := model.RatingInput{
		Rating:  5,
		Comment: stringPtr("Excellent service!"),
	}
	
	// Настраиваем моки
	mockRatingRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.DriverRating")).Return(nil)
	mockDriverSvc.On("UpdateDriverRating", mock.Anything, driverID, float64(5)).Return(nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.AddDriverRating(ctx, driverID, &orderID, &customerID, input)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, driverID, result.DriverID)
	assert.Equal(t, orderID, *result.OrderID)
	assert.Equal(t, customerID, *result.CustomerID)
	assert.Equal(t, 5, result.Rating)
	assert.Equal(t, "Excellent service!", *result.Comment)
	
	// Проверяем, что моки были вызваны
	mockRatingRepo.AssertExpectations(t)
	mockDriverSvc.AssertExpectations(t)
}

func TestMutationResolver_StartShift(t *testing.T) {
	mutationRes, mockDriverSvc, _, _, _, mockShiftRepo := createMutationResolver()
	
	// Подготавливаем тестовые данные
	driverID := uuid.New()
	vehicleID := uuid.New()
	input := &model.ShiftStartInput{
		VehicleID: &vehicleID,
		Latitude:  floatPtr(55.7558),
		Longitude: floatPtr(37.6176),
		Notes:     stringPtr("Starting shift"),
	}
	
	// Настраиваем моки
	mockShiftRepo.On("Create", mock.Anything, mock.AnythingOfType("*entities.DriverShift")).Return(nil)
	mockDriverSvc.On("ChangeDriverStatus", mock.Anything, driverID, entities.StatusOnShift).Return(nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.StartShift(ctx, driverID, input)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, driverID, result.DriverID)
	assert.Equal(t, vehicleID, *result.VehicleID)
	assert.Equal(t, model.ShiftStatusActive, result.Status)
	
	// Проверяем, что моки были вызваны
	mockShiftRepo.AssertExpectations(t)
	mockDriverSvc.AssertExpectations(t)
}

func TestMutationResolver_EndShift(t *testing.T) {
	mutationRes, mockDriverSvc, _, _, _, mockShiftRepo := createMutationResolver()
	
	// Подготавливаем тестовые данные
	driverID := uuid.New()
	activeShift := &entities.DriverShift{
		ID:       uuid.New(),
		DriverID: driverID,
		Status:   entities.ShiftStatusActive,
		StartTime: time.Now().Add(-2 * time.Hour),
	}
	input := &model.ShiftEndInput{
		Latitude:  floatPtr(55.7600),
		Longitude: floatPtr(37.6200),
		Notes:     stringPtr("Ending shift"),
	}
	
	// Настраиваем моки
	mockShiftRepo.On("GetActiveByDriverID", mock.Anything, driverID).Return(activeShift, nil)
	mockShiftRepo.On("Update", mock.Anything, mock.AnythingOfType("*entities.DriverShift")).Return(nil)
	mockDriverSvc.On("ChangeDriverStatus", mock.Anything, driverID, entities.StatusAvailable).Return(nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := mutationRes.EndShift(ctx, driverID, input)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, driverID, result.DriverID)
	assert.Equal(t, model.ShiftStatusCompleted, result.Status)
	assert.NotNil(t, result.EndTime)
	
	// Проверяем, что моки были вызваны
	mockShiftRepo.AssertExpectations(t)
	mockDriverSvc.AssertExpectations(t)
}

// Вспомогательные функции для создания указателей на примитивные типы
func stringPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}