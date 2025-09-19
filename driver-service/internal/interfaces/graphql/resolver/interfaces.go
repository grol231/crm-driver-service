package resolver

import (
	"context"

	"driver-service/internal/domain/entities"

	"github.com/google/uuid"
)

// Недостающие интерфейсы для repositories

// RatingRepository интерфейс для работы с рейтингами
type RatingRepository interface {
	Create(ctx context.Context, rating *entities.DriverRating) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverRating, error)
	Update(ctx context.Context, rating *entities.DriverRating) error
	List(ctx context.Context, filters *entities.RatingFilters) ([]*entities.DriverRating, error)
	Count(ctx context.Context, filters *entities.RatingFilters) (int, error)
	GetDriverStats(ctx context.Context, driverID uuid.UUID) (*entities.RatingStats, error)
}

// ShiftRepository интерфейс для работы со сменами
type ShiftRepository interface {
	Create(ctx context.Context, shift *entities.DriverShift) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverShift, error)
	GetActiveByDriverID(ctx context.Context, driverID uuid.UUID) (*entities.DriverShift, error)
	Update(ctx context.Context, shift *entities.DriverShift) error
	List(ctx context.Context, filters *entities.ShiftFilters) ([]*entities.DriverShift, error)
	Count(ctx context.Context, filters *entities.ShiftFilters) (int, error)
	GetActiveShifts(ctx context.Context) ([]*entities.DriverShift, error)
}

// LocationRepository интерфейс для работы с местоположениями
type LocationRepository interface {
	Create(ctx context.Context, location *entities.DriverLocation) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverLocation, error)
	GetLatestByDriverID(ctx context.Context, driverID uuid.UUID) (*entities.DriverLocation, error)
	List(ctx context.Context, filters *entities.LocationFilters) ([]*entities.DriverLocation, error)
	Count(ctx context.Context, filters *entities.LocationFilters) (int, error)
	BatchCreate(ctx context.Context, locations []*entities.DriverLocation) error
}

// LocationService интерфейс для работы с местоположениями
type LocationService interface {
	UpdateLocation(ctx context.Context, location *entities.DriverLocation) error
	BatchUpdateLocations(ctx context.Context, locations []*entities.DriverLocation) error
	GetNearbyDrivers(ctx context.Context, latitude, longitude, radiusKM float64, limit int) ([]*entities.Driver, error)
}