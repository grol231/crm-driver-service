package integration

import (
	"context"
	"fmt"

	"driver-service/internal/domain/entities"

	"github.com/google/uuid"
)

// Моки для интеграционных тестов
// Эти моки хранят данные в памяти и предоставляют простую реализацию для тестирования

// mockDriverServiceIntegration мок для DriverService
type mockDriverServiceIntegration struct {
	drivers map[uuid.UUID]*entities.Driver
}

func newMockDriverServiceIntegration() *mockDriverServiceIntegration {
	return &mockDriverServiceIntegration{
		drivers: make(map[uuid.UUID]*entities.Driver),
	}
}

func (m *mockDriverServiceIntegration) CreateDriver(ctx context.Context, driver *entities.Driver) (*entities.Driver, error) {
	if driver.Phone == "" {
		return nil, entities.ErrInvalidPhone
	}
	
	driver.ID = uuid.New()
	driver.Status = entities.StatusRegistered
	driver.CurrentRating = 0.0
	driver.TotalTrips = 0
	
	m.drivers[driver.ID] = driver
	return driver, nil
}

func (m *mockDriverServiceIntegration) GetDriverByID(ctx context.Context, id uuid.UUID) (*entities.Driver, error) {
	driver, exists := m.drivers[id]
	if !exists {
		return nil, entities.ErrDriverNotFound
	}
	return driver, nil
}

func (m *mockDriverServiceIntegration) GetDriverByPhone(ctx context.Context, phone string) (*entities.Driver, error) {
	for _, driver := range m.drivers {
		if driver.Phone == phone {
			return driver, nil
		}
	}
	return nil, entities.ErrDriverNotFound
}

func (m *mockDriverServiceIntegration) GetDriverByEmail(ctx context.Context, email string) (*entities.Driver, error) {
	for _, driver := range m.drivers {
		if driver.Email == email {
			return driver, nil
		}
	}
	return nil, entities.ErrDriverNotFound
}

func (m *mockDriverServiceIntegration) UpdateDriver(ctx context.Context, driver *entities.Driver) (*entities.Driver, error) {
	existing, exists := m.drivers[driver.ID]
	if !exists {
		return nil, entities.ErrDriverNotFound
	}
	
	// Обновляем существующего водителя
	*existing = *driver
	return existing, nil
}

func (m *mockDriverServiceIntegration) DeleteDriver(ctx context.Context, id uuid.UUID) error {
	if _, exists := m.drivers[id]; !exists {
		return entities.ErrDriverNotFound
	}
	delete(m.drivers, id)
	return nil
}

func (m *mockDriverServiceIntegration) ListDrivers(ctx context.Context, filters *entities.DriverFilters) ([]*entities.Driver, error) {
	var result []*entities.Driver
	
	for _, driver := range m.drivers {
		result = append(result, driver)
	}
	
	// Простая пагинация
	if filters != nil {
		start := filters.Offset
		end := start + filters.Limit
		
		if start >= len(result) {
			return []*entities.Driver{}, nil
		}
		if end > len(result) {
			end = len(result)
		}
		
		return result[start:end], nil
	}
	
	return result, nil
}

func (m *mockDriverServiceIntegration) CountDrivers(ctx context.Context, filters *entities.DriverFilters) (int, error) {
	return len(m.drivers), nil
}

func (m *mockDriverServiceIntegration) ChangeDriverStatus(ctx context.Context, id uuid.UUID, status entities.Status) error {
	driver, exists := m.drivers[id]
	if !exists {
		return entities.ErrDriverNotFound
	}
	
	driver.Status = status
	return nil
}

func (m *mockDriverServiceIntegration) UpdateDriverRating(ctx context.Context, id uuid.UUID, rating float64) error {
	driver, exists := m.drivers[id]
	if !exists {
		return entities.ErrDriverNotFound
	}
	
	driver.CurrentRating = rating
	return nil
}

func (m *mockDriverServiceIntegration) IncrementTripCount(ctx context.Context, id uuid.UUID) error {
	driver, exists := m.drivers[id]
	if !exists {
		return entities.ErrDriverNotFound
	}
	
	driver.TotalTrips++
	return nil
}

func (m *mockDriverServiceIntegration) GetActiveDrivers(ctx context.Context) ([]*entities.Driver, error) {
	var activeDrivers []*entities.Driver
	
	for _, driver := range m.drivers {
		if driver.Status == entities.StatusAvailable || driver.Status == entities.StatusOnShift {
			activeDrivers = append(activeDrivers, driver)
		}
	}
	
	return activeDrivers, nil
}

func (m *mockDriverServiceIntegration) IsDriverAvailable(ctx context.Context, id uuid.UUID) (bool, error) {
	driver, exists := m.drivers[id]
	if !exists {
		return false, entities.ErrDriverNotFound
	}
	
	return driver.Status == entities.StatusAvailable, nil
}

func (m *mockDriverServiceIntegration) ValidateDriverForOrder(ctx context.Context, id uuid.UUID) error {
	driver, exists := m.drivers[id]
	if !exists {
		return entities.ErrDriverNotFound
	}
	
	if driver.Status != entities.StatusAvailable {
		return entities.ErrDriverNotAvailable
	}
	
	return nil
}

// mockLocationServiceIntegration мок для LocationService
type mockLocationServiceIntegration struct {
	locations map[uuid.UUID]*entities.DriverLocation
}

func newMockLocationServiceIntegration() *mockLocationServiceIntegration {
	return &mockLocationServiceIntegration{
		locations: make(map[uuid.UUID]*entities.DriverLocation),
	}
}

func (m *mockLocationServiceIntegration) UpdateLocation(ctx context.Context, location *entities.DriverLocation) error {
	if location.ID == uuid.Nil {
		location.ID = uuid.New()
	}
	m.locations[location.ID] = location
	return nil
}

func (m *mockLocationServiceIntegration) BatchUpdateLocations(ctx context.Context, locations []*entities.DriverLocation) error {
	for _, location := range locations {
		if location.ID == uuid.Nil {
			location.ID = uuid.New()
		}
		m.locations[location.ID] = location
	}
	return nil
}

func (m *mockLocationServiceIntegration) GetNearbyDrivers(ctx context.Context, latitude, longitude, radiusKM float64, limit int) ([]*entities.Driver, error) {
	// Простая реализация - возвращаем пустой список
	return []*entities.Driver{}, nil
}

// Остальные моки для repositories
type mockDriverRepoIntegration struct{}
type mockLocationRepoIntegration struct{}
type mockRatingRepoIntegration struct{}
type mockShiftRepoIntegration struct{}
type mockDocumentRepoIntegration struct{}

func newMockDriverRepoIntegration() *mockDriverRepoIntegration {
	return &mockDriverRepoIntegration{}
}

func newMockLocationRepoIntegration() *mockLocationRepoIntegration {
	return &mockLocationRepoIntegration{}
}

func newMockRatingRepoIntegration() *mockRatingRepoIntegration {
	return &mockRatingRepoIntegration{}
}

func newMockShiftRepoIntegration() *mockShiftRepoIntegration {
	return &mockShiftRepoIntegration{}
}

func newMockDocumentRepoIntegration() *mockDocumentRepoIntegration {
	return &mockDocumentRepoIntegration{}
}

// Простые реализации для тестирования
func (m *mockLocationRepoIntegration) GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverLocation, error) {
	return nil, entities.ErrLocationNotFound
}

func (m *mockLocationRepoIntegration) GetLatestByDriverID(ctx context.Context, driverID uuid.UUID) (*entities.DriverLocation, error) {
	return nil, entities.ErrLocationNotFound
}

func (m *mockLocationRepoIntegration) List(ctx context.Context, filters *entities.LocationFilters) ([]*entities.DriverLocation, error) {
	return []*entities.DriverLocation{}, nil
}

func (m *mockLocationRepoIntegration) Count(ctx context.Context, filters *entities.LocationFilters) (int, error) {
	return 0, nil
}

func (m *mockLocationRepoIntegration) Create(ctx context.Context, location *entities.DriverLocation) error {
	return nil
}

func (m *mockLocationRepoIntegration) BatchCreate(ctx context.Context, locations []*entities.DriverLocation) error {
	return nil
}

func (m *mockRatingRepoIntegration) GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverRating, error) {
	return nil, entities.ErrRatingNotFound
}

func (m *mockRatingRepoIntegration) List(ctx context.Context, filters *entities.RatingFilters) ([]*entities.DriverRating, error) {
	return []*entities.DriverRating{}, nil
}

func (m *mockRatingRepoIntegration) Count(ctx context.Context, filters *entities.RatingFilters) (int, error) {
	return 0, nil
}

func (m *mockRatingRepoIntegration) GetDriverStats(ctx context.Context, driverID uuid.UUID) (*entities.RatingStats, error) {
	return &entities.RatingStats{
		DriverID:      driverID,
		AverageRating: 4.5,
		TotalRatings:  10,
	}, nil
}

func (m *mockRatingRepoIntegration) Create(ctx context.Context, rating *entities.DriverRating) error {
	return nil
}

func (m *mockRatingRepoIntegration) Update(ctx context.Context, rating *entities.DriverRating) error {
	return nil
}

func (m *mockShiftRepoIntegration) GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverShift, error) {
	return nil, entities.ErrShiftNotFound
}

func (m *mockShiftRepoIntegration) GetActiveByDriverID(ctx context.Context, driverID uuid.UUID) (*entities.DriverShift, error) {
	return nil, entities.ErrShiftNotFound
}

func (m *mockShiftRepoIntegration) List(ctx context.Context, filters *entities.ShiftFilters) ([]*entities.DriverShift, error) {
	return []*entities.DriverShift{}, nil
}

func (m *mockShiftRepoIntegration) Count(ctx context.Context, filters *entities.ShiftFilters) (int, error) {
	return 0, nil
}

func (m *mockShiftRepoIntegration) GetActiveShifts(ctx context.Context) ([]*entities.DriverShift, error) {
	return []*entities.DriverShift{}, nil
}

func (m *mockShiftRepoIntegration) Create(ctx context.Context, shift *entities.DriverShift) error {
	return nil
}

func (m *mockShiftRepoIntegration) Update(ctx context.Context, shift *entities.DriverShift) error {
	return nil
}