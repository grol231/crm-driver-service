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

// Моки для тестирования
type mockDriverService struct {
	mock.Mock
}

func (m *mockDriverService) GetDriverByID(ctx context.Context, id uuid.UUID) (*entities.Driver, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Driver), args.Error(1)
}

func (m *mockDriverService) CreateDriver(ctx context.Context, driver *entities.Driver) (*entities.Driver, error) {
	args := m.Called(ctx, driver)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Driver), args.Error(1)
}

func (m *mockDriverService) GetDriverByPhone(ctx context.Context, phone string) (*entities.Driver, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Driver), args.Error(1)
}

func (m *mockDriverService) GetDriverByEmail(ctx context.Context, email string) (*entities.Driver, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Driver), args.Error(1)
}

func (m *mockDriverService) UpdateDriver(ctx context.Context, driver *entities.Driver) (*entities.Driver, error) {
	args := m.Called(ctx, driver)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Driver), args.Error(1)
}

func (m *mockDriverService) DeleteDriver(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockDriverService) ListDrivers(ctx context.Context, filters *entities.DriverFilters) ([]*entities.Driver, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Driver), args.Error(1)
}

func (m *mockDriverService) CountDrivers(ctx context.Context, filters *entities.DriverFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}

func (m *mockDriverService) ChangeDriverStatus(ctx context.Context, id uuid.UUID, status entities.Status) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *mockDriverService) UpdateDriverRating(ctx context.Context, id uuid.UUID, rating float64) error {
	args := m.Called(ctx, id, rating)
	return args.Error(0)
}

func (m *mockDriverService) IncrementTripCount(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockDriverService) GetActiveDrivers(ctx context.Context) ([]*entities.Driver, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Driver), args.Error(1)
}

func (m *mockDriverService) IsDriverAvailable(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *mockDriverService) ValidateDriverForOrder(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

type mockLocationService struct {
	mock.Mock
}

func (m *mockLocationService) UpdateLocation(ctx context.Context, location *entities.DriverLocation) error {
	args := m.Called(ctx, location)
	return args.Error(0)
}

func (m *mockLocationService) BatchUpdateLocations(ctx context.Context, locations []*entities.DriverLocation) error {
	args := m.Called(ctx, locations)
	return args.Error(0)
}

func (m *mockLocationService) GetNearbyDrivers(ctx context.Context, latitude, longitude, radiusKM float64, limit int) ([]*entities.Driver, error) {
	args := m.Called(ctx, latitude, longitude, radiusKM, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Driver), args.Error(1)
}

// Дополнительные моки для repositories
type mockDriverRepository struct {
	mock.Mock
}

type mockLocationRepository struct {
	mock.Mock
}

func (m *mockLocationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverLocation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DriverLocation), args.Error(1)
}

func (m *mockLocationRepository) GetLatestByDriverID(ctx context.Context, driverID uuid.UUID) (*entities.DriverLocation, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DriverLocation), args.Error(1)
}

func (m *mockLocationRepository) List(ctx context.Context, filters *entities.LocationFilters) ([]*entities.DriverLocation, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.DriverLocation), args.Error(1)
}

func (m *mockLocationRepository) Count(ctx context.Context, filters *entities.LocationFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}

func (m *mockLocationRepository) Create(ctx context.Context, location *entities.DriverLocation) error {
	args := m.Called(ctx, location)
	return args.Error(0)
}

func (m *mockLocationRepository) BatchCreate(ctx context.Context, locations []*entities.DriverLocation) error {
	args := m.Called(ctx, locations)
	return args.Error(0)
}

type mockRatingRepository struct {
	mock.Mock
}

func (m *mockRatingRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverRating, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DriverRating), args.Error(1)
}

func (m *mockRatingRepository) List(ctx context.Context, filters *entities.RatingFilters) ([]*entities.DriverRating, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.DriverRating), args.Error(1)
}

func (m *mockRatingRepository) Count(ctx context.Context, filters *entities.RatingFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}

func (m *mockRatingRepository) GetDriverStats(ctx context.Context, driverID uuid.UUID) (*entities.RatingStats, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.RatingStats), args.Error(1)
}

func (m *mockRatingRepository) Create(ctx context.Context, rating *entities.DriverRating) error {
	args := m.Called(ctx, rating)
	return args.Error(0)
}

func (m *mockRatingRepository) Update(ctx context.Context, rating *entities.DriverRating) error {
	args := m.Called(ctx, rating)
	return args.Error(0)
}

type mockShiftRepository struct {
	mock.Mock
}

func (m *mockShiftRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverShift, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DriverShift), args.Error(1)
}

func (m *mockShiftRepository) GetActiveByDriverID(ctx context.Context, driverID uuid.UUID) (*entities.DriverShift, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.DriverShift), args.Error(1)
}

func (m *mockShiftRepository) List(ctx context.Context, filters *entities.ShiftFilters) ([]*entities.DriverShift, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.DriverShift), args.Error(1)
}

func (m *mockShiftRepository) Count(ctx context.Context, filters *entities.ShiftFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}

func (m *mockShiftRepository) GetActiveShifts(ctx context.Context) ([]*entities.DriverShift, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.DriverShift), args.Error(1)
}

func (m *mockShiftRepository) Create(ctx context.Context, shift *entities.DriverShift) error {
	args := m.Called(ctx, shift)
	return args.Error(0)
}

func (m *mockShiftRepository) Update(ctx context.Context, shift *entities.DriverShift) error {
	args := m.Called(ctx, shift)
	return args.Error(0)
}

type mockDocumentRepository struct {
	mock.Mock
}

// createTestDriver создает тестового водителя
func createTestDriver() *entities.Driver {
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

// createQueryResolver создает query resolver с моками для тестирования
func createQueryResolver() (*queryResolver, *mockDriverService, *mockLocationService, *mockLocationRepository, *mockRatingRepository, *mockShiftRepository) {
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
	
	queryRes := &queryResolver{resolver}
	
	return queryRes, mockDriverSvc, mockLocationSvc, mockLocationRepo, mockRatingRepo, mockShiftRepo
}

func TestQueryResolver_Driver(t *testing.T) {
	queryRes, mockDriverSvc, _, _, _, _ := createQueryResolver()
	
	// Подготавливаем тестовые данные
	testDriver := createTestDriver()
	driverID := testDriver.ID
	
	// Настраиваем мок
	mockDriverSvc.On("GetDriverByID", mock.Anything, driverID).Return(testDriver, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := queryRes.Driver(ctx, driverID)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testDriver.ID, result.ID)
	assert.Equal(t, testDriver.Phone, result.Phone)
	assert.Equal(t, testDriver.Email, result.Email)
	assert.Equal(t, testDriver.FirstName, result.FirstName)
	assert.Equal(t, testDriver.LastName, result.LastName)
	
	// Проверяем, что мок был вызван
	mockDriverSvc.AssertExpectations(t)
}

func TestQueryResolver_Driver_NotFound(t *testing.T) {
	queryRes, mockDriverSvc, _, _, _, _ := createQueryResolver()
	
	// Подготавливаем тестовые данные
	driverID := uuid.New()
	
	// Настраиваем мок для возврата ошибки
	mockDriverSvc.On("GetDriverByID", mock.Anything, driverID).Return(nil, entities.ErrDriverNotFound)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := queryRes.Driver(ctx, driverID)
	
	// Проверяем результат
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, entities.ErrDriverNotFound, err)
	
	// Проверяем, что мок был вызван
	mockDriverSvc.AssertExpectations(t)
}

func TestQueryResolver_Drivers(t *testing.T) {
	queryRes, mockDriverSvc, _, _, _, _ := createQueryResolver()
	
	// Подготавливаем тестовые данные
	testDrivers := []*entities.Driver{createTestDriver(), createTestDriver()}
	
	// Настраиваем моки
	mockDriverSvc.On("ListDrivers", mock.Anything, mock.AnythingOfType("*entities.DriverFilters")).Return(testDrivers, nil)
	mockDriverSvc.On("CountDrivers", mock.Anything, mock.AnythingOfType("*entities.DriverFilters")).Return(2, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := queryRes.Drivers(ctx, nil, nil, nil)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Drivers, 2)
	assert.Equal(t, 2, result.PageInfo.Total)
	assert.Equal(t, 20, result.PageInfo.Limit) // значение по умолчанию
	assert.Equal(t, 0, result.PageInfo.Offset)  // значение по умолчанию
	assert.False(t, result.PageInfo.HasMore)
	
	// Проверяем, что моки были вызваны
	mockDriverSvc.AssertExpectations(t)
}

func TestQueryResolver_ActiveDrivers(t *testing.T) {
	queryRes, mockDriverSvc, _, _, _, _ := createQueryResolver()
	
	// Подготавливаем тестовые данные
	testDrivers := []*entities.Driver{createTestDriver()}
	
	// Настраиваем мок
	mockDriverSvc.On("GetActiveDrivers", mock.Anything).Return(testDrivers, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := queryRes.ActiveDrivers(ctx)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, testDrivers[0].ID, result[0].ID)
	
	// Проверяем, что мок был вызван
	mockDriverSvc.AssertExpectations(t)
}

func TestQueryResolver_DriverByPhone(t *testing.T) {
	queryRes, mockDriverSvc, _, _, _, _ := createQueryResolver()
	
	// Подготавливаем тестовые данные
	testDriver := createTestDriver()
	phone := testDriver.Phone
	
	// Настраиваем мок
	mockDriverSvc.On("GetDriverByPhone", mock.Anything, phone).Return(testDriver, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := queryRes.DriverByPhone(ctx, phone)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testDriver.ID, result.ID)
	assert.Equal(t, phone, result.Phone)
	
	// Проверяем, что мок был вызван
	mockDriverSvc.AssertExpectations(t)
}

func TestQueryResolver_DriverByEmail(t *testing.T) {
	queryRes, mockDriverSvc, _, _, _, _ := createQueryResolver()
	
	// Подготавливаем тестовые данные
	testDriver := createTestDriver()
	email := testDriver.Email
	
	// Настраиваем мок
	mockDriverSvc.On("GetDriverByEmail", mock.Anything, email).Return(testDriver, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := queryRes.DriverByEmail(ctx, email)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testDriver.ID, result.ID)
	assert.Equal(t, email, result.Email)
	
	// Проверяем, что мок был вызван
	mockDriverSvc.AssertExpectations(t)
}

func TestQueryResolver_NearbyDrivers(t *testing.T) {
	queryRes, _, mockLocationSvc, _, _, _ := createQueryResolver()
	
	// Подготавливаем тестовые данные
	testDrivers := []*entities.Driver{createTestDriver()}
	latitude := 55.7558
	longitude := 37.6176
	
	// Настраиваем мок
	mockLocationSvc.On("GetNearbyDrivers", mock.Anything, latitude, longitude, 5.0, 10).Return(testDrivers, nil)
	
	// Выполняем тест
	ctx := context.Background()
	result, err := queryRes.NearbyDrivers(ctx, latitude, longitude, nil, nil)
	
	// Проверяем результат
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, testDrivers[0].ID, result[0].ID)
	
	// Проверяем, что мок был вызван
	mockLocationSvc.AssertExpectations(t)
}