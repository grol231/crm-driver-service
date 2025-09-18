package services

import (
	"context"
	"fmt"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/repositories"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LocationService интерфейс для управления местоположениями водителей
type LocationService interface {
	UpdateLocation(ctx context.Context, location *entities.DriverLocation) error
	GetCurrentLocation(ctx context.Context, driverID uuid.UUID) (*entities.DriverLocation, error)
	GetLocationHistory(ctx context.Context, driverID uuid.UUID, from, to time.Time) ([]*entities.DriverLocation, error)
	GetLocationStats(ctx context.Context, driverID uuid.UUID, from, to time.Time) (*entities.LocationStats, error)
	StreamLocations(ctx context.Context, driverID uuid.UUID) (<-chan *entities.DriverLocation, error)
	StartOrderTracking(ctx context.Context, driverID, orderID uuid.UUID) error
	StopOrderTracking(ctx context.Context, driverID, orderID uuid.UUID) error
	GetNearbyDrivers(ctx context.Context, lat, lon, radiusKm float64, limit int) ([]*entities.DriverLocation, error)
	BatchUpdateLocations(ctx context.Context, locations []*entities.DriverLocation) error
	CleanupOldLocations(ctx context.Context) error
}

// locationService реализация LocationService
type locationService struct {
	locationRepo repositories.LocationRepository
	driverRepo   repositories.DriverRepository
	eventBus     EventPublisher
	logger       *zap.Logger
}

// NewLocationService создает новый LocationService
func NewLocationService(
	locationRepo repositories.LocationRepository,
	driverRepo repositories.DriverRepository,
	eventBus EventPublisher,
	logger *zap.Logger,
) LocationService {
	return &locationService{
		locationRepo: locationRepo,
		driverRepo:   driverRepo,
		eventBus:     eventBus,
		logger:       logger,
	}
}

// UpdateLocation обновляет местоположение водителя
func (s *locationService) UpdateLocation(ctx context.Context, location *entities.DriverLocation) error {
	s.logger.Debug("Updating driver location",
		zap.String("driver_id", location.DriverID.String()),
		zap.Float64("latitude", location.Latitude),
		zap.Float64("longitude", location.Longitude),
	)

	// Валидация местоположения
	if err := location.Validate(); err != nil {
		s.logger.Error("Location validation failed",
			zap.Error(err),
			zap.String("driver_id", location.DriverID.String()),
		)
		return fmt.Errorf("location validation failed: %w", err)
	}

	// Проверяем, существует ли водитель
	_, err := s.driverRepo.GetByID(ctx, location.DriverID)
	if err != nil {
		s.logger.Error("Driver not found for location update",
			zap.Error(err),
			zap.String("driver_id", location.DriverID.String()),
		)
		return err
	}

	// Устанавливаем ID и время создания
	if location.ID == uuid.Nil {
		location.ID = uuid.New()
	}
	if location.CreatedAt.IsZero() {
		location.CreatedAt = time.Now()
	}
	if location.RecordedAt.IsZero() {
		location.RecordedAt = time.Now()
	}

	// Сохраняем местоположение в базе данных
	if err := s.locationRepo.Create(ctx, location); err != nil {
		s.logger.Error("Failed to save location",
			zap.Error(err),
			zap.String("driver_id", location.DriverID.String()),
		)
		return fmt.Errorf("failed to save location: %w", err)
	}

	// Публикуем событие об обновлении местоположения
	eventData := map[string]interface{}{
		"location": location.ToLocation(),
		"speed":    location.GetSpeed(),
		"bearing":  location.GetBearing(),
		"accuracy": location.GetAccuracy(),
	}

	if err := s.eventBus.PublishDriverEvent(ctx, "driver.location.updated", location.DriverID, eventData); err != nil {
		s.logger.Error("Failed to publish location updated event",
			zap.Error(err),
			zap.String("driver_id", location.DriverID.String()),
		)
		// Не возвращаем ошибку, так как местоположение уже сохранено
	}

	return nil
}

// GetCurrentLocation получает текущее местоположение водителя
func (s *locationService) GetCurrentLocation(ctx context.Context, driverID uuid.UUID) (*entities.DriverLocation, error) {
	location, err := s.locationRepo.GetLatestByDriverID(ctx, driverID)
	if err != nil {
		s.logger.Error("Failed to get current location",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
		)
		return nil, err
	}

	// Проверяем, не слишком ли старые данные (больше 10 минут)
	if time.Since(location.RecordedAt) > 10*time.Minute {
		s.logger.Warn("Location data is too old",
			zap.String("driver_id", driverID.String()),
			zap.Time("recorded_at", location.RecordedAt),
		)
		return nil, entities.ErrLocationTooOld
	}

	return location, nil
}

// GetLocationHistory получает историю местоположений водителя
func (s *locationService) GetLocationHistory(ctx context.Context, driverID uuid.UUID, from, to time.Time) ([]*entities.DriverLocation, error) {
	if to.Before(from) {
		return nil, fmt.Errorf("invalid time range: 'to' time is before 'from' time")
	}

	locations, err := s.locationRepo.GetByDriverIDInTimeRange(ctx, driverID, from, to)
	if err != nil {
		s.logger.Error("Failed to get location history",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
			zap.Time("from", from),
			zap.Time("to", to),
		)
		return nil, err
	}

	return locations, nil
}

// GetLocationStats вычисляет статистику по местоположениям
func (s *locationService) GetLocationStats(ctx context.Context, driverID uuid.UUID, from, to time.Time) (*entities.LocationStats, error) {
	locations, err := s.GetLocationHistory(ctx, driverID, from, to)
	if err != nil {
		return nil, err
	}

	stats := entities.CalculateLocationStats(locations)
	return stats, nil
}

// StreamLocations создает поток для получения местоположений водителя в реальном времени
func (s *locationService) StreamLocations(ctx context.Context, driverID uuid.UUID) (<-chan *entities.DriverLocation, error) {
	// Проверяем, существует ли водитель
	_, err := s.driverRepo.GetByID(ctx, driverID)
	if err != nil {
		return nil, err
	}

	// Создаем канал для передачи местоположений
	locationChan := make(chan *entities.DriverLocation, 100)

	// В реальном приложении здесь должна быть реализация
	// подписки на обновления через Redis/WebSocket/NATS
	// Для демонстрации создаем заглушку
	go func() {
		defer close(locationChan)
		
		// Получаем последнее местоположение
		location, err := s.GetCurrentLocation(ctx, driverID)
		if err == nil {
			select {
			case locationChan <- location:
			case <-ctx.Done():
				return
			}
		}

		// Ждем отмены контекста
		<-ctx.Done()
	}()

	return locationChan, nil
}

// StartOrderTracking начинает отслеживание заказа
func (s *locationService) StartOrderTracking(ctx context.Context, driverID, orderID uuid.UUID) error {
	s.logger.Info("Starting order tracking",
		zap.String("driver_id", driverID.String()),
		zap.String("order_id", orderID.String()),
	)

	// Получаем текущее местоположение водителя
	location, err := s.GetCurrentLocation(ctx, driverID)
	if err != nil {
		s.logger.Error("Failed to get current location for order tracking",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
		)
		return err
	}

	// Добавляем информацию о заказе в метаданные
	if location.Metadata == nil {
		location.Metadata = make(entities.Metadata)
	}
	location.Metadata["on_trip"] = true
	location.Metadata["order_id"] = orderID.String()

	// Обновляем местоположение с информацией о заказе
	location.ID = uuid.New() // Создаем новую запись
	location.CreatedAt = time.Now()
	
	if err := s.locationRepo.Create(ctx, location); err != nil {
		s.logger.Error("Failed to create tracking location record",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
		)
		return fmt.Errorf("failed to create tracking location record: %w", err)
	}

	return nil
}

// StopOrderTracking прекращает отслеживание заказа
func (s *locationService) StopOrderTracking(ctx context.Context, driverID, orderID uuid.UUID) error {
	s.logger.Info("Stopping order tracking",
		zap.String("driver_id", driverID.String()),
		zap.String("order_id", orderID.String()),
	)

	// Получаем текущее местоположение
	location, err := s.GetCurrentLocation(ctx, driverID)
	if err != nil {
		s.logger.Error("Failed to get current location for stopping tracking",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
		)
		return err
	}

	// Обновляем метаданные
	if location.Metadata == nil {
		location.Metadata = make(entities.Metadata)
	}
	location.Metadata["on_trip"] = false
	delete(location.Metadata, "order_id")

	// Создаем новую запись о завершении отслеживания
	location.ID = uuid.New()
	location.CreatedAt = time.Now()
	location.RecordedAt = time.Now()

	if err := s.locationRepo.Create(ctx, location); err != nil {
		s.logger.Error("Failed to create stop tracking location record",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
		)
		return fmt.Errorf("failed to create stop tracking location record: %w", err)
	}

	return nil
}

// GetNearbyDrivers получает водителей поблизости от указанной точки
func (s *locationService) GetNearbyDrivers(ctx context.Context, lat, lon, radiusKm float64, limit int) ([]*entities.DriverLocation, error) {
	if radiusKm <= 0 {
		return nil, fmt.Errorf("radius must be positive")
	}

	if limit <= 0 {
		limit = 50 // Значение по умолчанию
	}

	locations, err := s.locationRepo.GetNearby(ctx, lat, lon, radiusKm, limit)
	if err != nil {
		s.logger.Error("Failed to get nearby drivers",
			zap.Error(err),
			zap.Float64("latitude", lat),
			zap.Float64("longitude", lon),
			zap.Float64("radius_km", radiusKm),
		)
		return nil, err
	}

	// Фильтруем только активных водителей
	var activeDriverLocations []*entities.DriverLocation
	for _, location := range locations {
		driver, err := s.driverRepo.GetByID(ctx, location.DriverID)
		if err != nil {
			continue
		}

		if driver.IsActive() {
			activeDriverLocations = append(activeDriverLocations, location)
		}
	}

	return activeDriverLocations, nil
}

// BatchUpdateLocations обновляет множество местоположений за один запрос
func (s *locationService) BatchUpdateLocations(ctx context.Context, locations []*entities.DriverLocation) error {
	if len(locations) == 0 {
		return nil
	}

	s.logger.Info("Batch updating locations",
		zap.Int("count", len(locations)),
	)

	// Валидация всех местоположений
	now := time.Now()
	for _, location := range locations {
		if err := location.Validate(); err != nil {
			s.logger.Error("Location validation failed in batch",
				zap.Error(err),
				zap.String("driver_id", location.DriverID.String()),
			)
			return fmt.Errorf("location validation failed: %w", err)
		}

		// Устанавливаем значения по умолчанию
		if location.ID == uuid.Nil {
			location.ID = uuid.New()
		}
		if location.CreatedAt.IsZero() {
			location.CreatedAt = now
		}
		if location.RecordedAt.IsZero() {
			location.RecordedAt = now
		}
	}

	// Сохраняем все местоположения
	if err := s.locationRepo.CreateBatch(ctx, locations); err != nil {
		s.logger.Error("Failed to batch update locations",
			zap.Error(err),
			zap.Int("count", len(locations)),
		)
		return fmt.Errorf("failed to batch update locations: %w", err)
	}

	s.logger.Info("Batch location update completed successfully",
		zap.Int("count", len(locations)),
	)

	return nil
}

// CleanupOldLocations удаляет старые данные о местоположении
func (s *locationService) CleanupOldLocations(ctx context.Context) error {
	// Удаляем данные старше 30 дней
	cutoffTime := time.Now().AddDate(0, 0, -30)

	s.logger.Info("Starting location cleanup",
		zap.Time("cutoff_time", cutoffTime),
	)

	if err := s.locationRepo.DeleteOld(ctx, cutoffTime); err != nil {
		s.logger.Error("Failed to cleanup old locations",
			zap.Error(err),
		)
		return fmt.Errorf("failed to cleanup old locations: %w", err)
	}

	s.logger.Info("Location cleanup completed")
	return nil
}