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

// DriverService интерфейс для управления водителями
type DriverService interface {
	CreateDriver(ctx context.Context, driver *entities.Driver) (*entities.Driver, error)
	GetDriverByID(ctx context.Context, id uuid.UUID) (*entities.Driver, error)
	GetDriverByPhone(ctx context.Context, phone string) (*entities.Driver, error)
	GetDriverByEmail(ctx context.Context, email string) (*entities.Driver, error)
	UpdateDriver(ctx context.Context, driver *entities.Driver) (*entities.Driver, error)
	DeleteDriver(ctx context.Context, id uuid.UUID) error
	ListDrivers(ctx context.Context, filters *entities.DriverFilters) ([]*entities.Driver, error)
	CountDrivers(ctx context.Context, filters *entities.DriverFilters) (int, error)
	ChangeDriverStatus(ctx context.Context, id uuid.UUID, status entities.Status) error
	UpdateDriverRating(ctx context.Context, id uuid.UUID, rating float64) error
	IncrementTripCount(ctx context.Context, id uuid.UUID) error
	GetActiveDrivers(ctx context.Context) ([]*entities.Driver, error)
	IsDriverAvailable(ctx context.Context, id uuid.UUID) (bool, error)
	ValidateDriverForOrder(ctx context.Context, id uuid.UUID) error
}

// driverService реализация DriverService
type driverService struct {
	driverRepo   repositories.DriverRepository
	documentRepo repositories.DocumentRepository
	logger       *zap.Logger
	eventBus     EventPublisher // Интерфейс для публикации событий
}

// EventPublisher интерфейс для публикации событий
type EventPublisher interface {
	PublishDriverEvent(ctx context.Context, eventType string, driverID uuid.UUID, data interface{}) error
}

// NewDriverService создает новый DriverService
func NewDriverService(
	driverRepo repositories.DriverRepository,
	documentRepo repositories.DocumentRepository,
	eventBus EventPublisher,
	logger *zap.Logger,
) DriverService {
	return &driverService{
		driverRepo:   driverRepo,
		documentRepo: documentRepo,
		eventBus:     eventBus,
		logger:       logger,
	}
}

// CreateDriver создает нового водителя
func (s *driverService) CreateDriver(ctx context.Context, driver *entities.Driver) (*entities.Driver, error) {
	s.logger.Info("Creating new driver",
		zap.String("phone", driver.Phone),
		zap.String("email", driver.Email),
	)

	// Валидация входных данных
	if err := driver.Validate(); err != nil {
		s.logger.Error("Driver validation failed",
			zap.Error(err),
			zap.String("phone", driver.Phone),
		)
		return nil, fmt.Errorf("driver validation failed: %w", err)
	}

	// Проверяем, не существует ли уже водитель с таким телефоном или лицензией
	exists, err := s.driverRepo.Exists(ctx, driver.Phone, driver.LicenseNumber)
	if err != nil {
		s.logger.Error("Failed to check driver existence",
			zap.Error(err),
			zap.String("phone", driver.Phone),
		)
		return nil, fmt.Errorf("failed to check driver existence: %w", err)
	}

	if exists {
		s.logger.Warn("Driver already exists",
			zap.String("phone", driver.Phone),
			zap.String("license", driver.LicenseNumber),
		)
		return nil, entities.ErrDriverExists
	}

	// Устанавливаем начальные значения
	now := time.Now()
	driver.ID = uuid.New()
	driver.Status = entities.StatusRegistered
	driver.CurrentRating = 0.0
	driver.TotalTrips = 0
	driver.CreatedAt = now
	driver.UpdatedAt = now

	if driver.Metadata == nil {
		driver.Metadata = make(entities.Metadata)
	}

	// Создаем водителя в базе данных
	if err := s.driverRepo.Create(ctx, driver); err != nil {
		s.logger.Error("Failed to create driver",
			zap.Error(err),
			zap.String("driver_id", driver.ID.String()),
		)
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	// Публикуем событие о создании водителя
	eventData := map[string]interface{}{
		"phone":          driver.Phone,
		"email":          driver.Email,
		"name":           driver.GetFullName(),
		"license_number": driver.LicenseNumber,
	}

	if err := s.eventBus.PublishDriverEvent(ctx, "driver.registered", driver.ID, eventData); err != nil {
		s.logger.Error("Failed to publish driver registered event",
			zap.Error(err),
			zap.String("driver_id", driver.ID.String()),
		)
		// Не возвращаем ошибку, так как водитель уже создан
	}

	s.logger.Info("Driver created successfully",
		zap.String("driver_id", driver.ID.String()),
		zap.String("phone", driver.Phone),
	)

	return driver, nil
}

// GetDriverByID получает водителя по ID
func (s *driverService) GetDriverByID(ctx context.Context, id uuid.UUID) (*entities.Driver, error) {
	driver, err := s.driverRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get driver by ID",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
		return nil, err
	}

	return driver, nil
}

// GetDriverByPhone получает водителя по номеру телефона
func (s *driverService) GetDriverByPhone(ctx context.Context, phone string) (*entities.Driver, error) {
	driver, err := s.driverRepo.GetByPhone(ctx, phone)
	if err != nil {
		s.logger.Error("Failed to get driver by phone",
			zap.Error(err),
			zap.String("phone", phone),
		)
		return nil, err
	}

	return driver, nil
}

// GetDriverByEmail получает водителя по email
func (s *driverService) GetDriverByEmail(ctx context.Context, email string) (*entities.Driver, error) {
	driver, err := s.driverRepo.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to get driver by email",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, err
	}

	return driver, nil
}

// UpdateDriver обновляет данные водителя
func (s *driverService) UpdateDriver(ctx context.Context, driver *entities.Driver) (*entities.Driver, error) {
	s.logger.Info("Updating driver",
		zap.String("driver_id", driver.ID.String()),
	)

	// Валидация входных данных
	if err := driver.Validate(); err != nil {
		s.logger.Error("Driver validation failed",
			zap.Error(err),
			zap.String("driver_id", driver.ID.String()),
		)
		return nil, fmt.Errorf("driver validation failed: %w", err)
	}

	// Проверяем, существует ли водитель
	existing, err := s.driverRepo.GetByID(ctx, driver.ID)
	if err != nil {
		return nil, err
	}

	// Сохраняем некоторые поля, которые не должны изменяться через Update
	driver.CreatedAt = existing.CreatedAt
	driver.UpdatedAt = time.Now()

	// Обновляем водителя в базе данных
	if err := s.driverRepo.Update(ctx, driver); err != nil {
		s.logger.Error("Failed to update driver",
			zap.Error(err),
			zap.String("driver_id", driver.ID.String()),
		)
		return nil, fmt.Errorf("failed to update driver: %w", err)
	}

	s.logger.Info("Driver updated successfully",
		zap.String("driver_id", driver.ID.String()),
	)

	return driver, nil
}

// DeleteDriver удаляет водителя
func (s *driverService) DeleteDriver(ctx context.Context, id uuid.UUID) error {
	s.logger.Info("Deleting driver",
		zap.String("driver_id", id.String()),
	)

	// Проверяем, существует ли водитель
	driver, err := s.driverRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Мягкое удаление
	if err := s.driverRepo.SoftDelete(ctx, id); err != nil {
		s.logger.Error("Failed to delete driver",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
		return fmt.Errorf("failed to delete driver: %w", err)
	}

	// Публикуем событие о блокировке водителя
	eventData := map[string]interface{}{
		"reason": "account_deleted",
	}

	if err := s.eventBus.PublishDriverEvent(ctx, "driver.blocked", driver.ID, eventData); err != nil {
		s.logger.Error("Failed to publish driver blocked event",
			zap.Error(err),
			zap.String("driver_id", driver.ID.String()),
		)
	}

	s.logger.Info("Driver deleted successfully",
		zap.String("driver_id", id.String()),
	)

	return nil
}

// ListDrivers получает список водителей с фильтрами
func (s *driverService) ListDrivers(ctx context.Context, filters *entities.DriverFilters) ([]*entities.Driver, error) {
	drivers, err := s.driverRepo.List(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to list drivers",
			zap.Error(err),
		)
		return nil, err
	}

	return drivers, nil
}

// CountDrivers возвращает количество водителей с фильтрами
func (s *driverService) CountDrivers(ctx context.Context, filters *entities.DriverFilters) (int, error) {
	count, err := s.driverRepo.Count(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to count drivers",
			zap.Error(err),
		)
		return 0, err
	}

	return count, nil
}

// ChangeDriverStatus изменяет статус водителя
func (s *driverService) ChangeDriverStatus(ctx context.Context, id uuid.UUID, status entities.Status) error {
	s.logger.Info("Changing driver status",
		zap.String("driver_id", id.String()),
		zap.String("new_status", string(status)),
	)

	// Получаем текущего водителя
	driver, err := s.driverRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	oldStatus := driver.Status

	// Проверяем валидность перехода статуса
	if err := s.validateStatusTransition(oldStatus, status); err != nil {
		s.logger.Error("Invalid status transition",
			zap.Error(err),
			zap.String("driver_id", id.String()),
			zap.String("from_status", string(oldStatus)),
			zap.String("to_status", string(status)),
		)
		return err
	}

	// Обновляем статус
	if err := s.driverRepo.UpdateStatus(ctx, id, status); err != nil {
		s.logger.Error("Failed to update driver status",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
		return fmt.Errorf("failed to update driver status: %w", err)
	}

	// Публикуем событие об изменении статуса
	eventData := map[string]interface{}{
		"old_status": string(oldStatus),
		"new_status": string(status),
		"changed_by": "system", // В реальном приложении здесь должен быть ID пользователя
	}

	if err := s.eventBus.PublishDriverEvent(ctx, "driver.status.changed", id, eventData); err != nil {
		s.logger.Error("Failed to publish driver status changed event",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
	}

	s.logger.Info("Driver status changed successfully",
		zap.String("driver_id", id.String()),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(status)),
	)

	return nil
}

// UpdateDriverRating обновляет рейтинг водителя
func (s *driverService) UpdateDriverRating(ctx context.Context, id uuid.UUID, rating float64) error {
	if rating < 0 || rating > 5 {
		return fmt.Errorf("invalid rating: %f (must be between 0 and 5)", rating)
	}

	// Получаем старый рейтинг
	driver, err := s.driverRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	oldRating := driver.CurrentRating

	// Обновляем рейтинг
	if err := s.driverRepo.UpdateRating(ctx, id, rating); err != nil {
		s.logger.Error("Failed to update driver rating",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
		return fmt.Errorf("failed to update driver rating: %w", err)
	}

	// Публикуем событие об обновлении рейтинга
	eventData := map[string]interface{}{
		"new_rating":      rating,
		"previous_rating": oldRating,
	}

	if err := s.eventBus.PublishDriverEvent(ctx, "driver.rating.updated", id, eventData); err != nil {
		s.logger.Error("Failed to publish driver rating updated event",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
	}

	return nil
}

// IncrementTripCount увеличивает счетчик поездок
func (s *driverService) IncrementTripCount(ctx context.Context, id uuid.UUID) error {
	return s.driverRepo.IncrementTripCount(ctx, id)
}

// GetActiveDrivers получает список активных водителей
func (s *driverService) GetActiveDrivers(ctx context.Context) ([]*entities.Driver, error) {
	return s.driverRepo.GetActiveDrivers(ctx)
}

// IsDriverAvailable проверяет доступность водителя
func (s *driverService) IsDriverAvailable(ctx context.Context, id uuid.UUID) (bool, error) {
	driver, err := s.driverRepo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	return driver.CanReceiveOrders(), nil
}

// ValidateDriverForOrder проверяет, может ли водитель получить заказ
func (s *driverService) ValidateDriverForOrder(ctx context.Context, id uuid.UUID) error {
	driver, err := s.driverRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Проверяем статус водителя
	if !driver.CanReceiveOrders() {
		return entities.ErrDriverNotAvailable
	}

	// Проверяем, не истекла ли лицензия
	if driver.IsLicenseExpired() {
		return entities.ErrLicenseExpired
	}

	// Проверяем наличие верифицированных документов
	documents, err := s.documentRepo.GetByDriverID(ctx, id)
	if err != nil {
		return err
	}

	hasValidLicense := false
	for _, doc := range documents {
		if doc.DocumentType == entities.DocumentTypeDriverLicense && doc.IsVerified() {
			hasValidLicense = true
			break
		}
	}

	if !hasValidLicense {
		return entities.ErrDocumentNotVerified
	}

	return nil
}

// validateStatusTransition проверяет валидность перехода между статусами
func (s *driverService) validateStatusTransition(from, to entities.Status) error {
	// Разрешенные переходы между статусами
	allowedTransitions := map[entities.Status][]entities.Status{
		entities.StatusRegistered: {
			entities.StatusPendingVerification,
			entities.StatusBlocked,
		},
		entities.StatusPendingVerification: {
			entities.StatusVerified,
			entities.StatusRejected,
			entities.StatusRegistered,
			entities.StatusBlocked,
		},
		entities.StatusVerified: {
			entities.StatusAvailable,
			entities.StatusSuspended,
			entities.StatusBlocked,
		},
		entities.StatusRejected: {
			entities.StatusPendingVerification,
			entities.StatusBlocked,
		},
		entities.StatusAvailable: {
			entities.StatusOnShift,
			entities.StatusInactive,
			entities.StatusSuspended,
			entities.StatusBlocked,
		},
		entities.StatusOnShift: {
			entities.StatusBusy,
			entities.StatusAvailable,
			entities.StatusInactive,
			entities.StatusSuspended,
		},
		entities.StatusBusy: {
			entities.StatusOnShift,
			entities.StatusAvailable,
			entities.StatusInactive,
		},
		entities.StatusInactive: {
			entities.StatusAvailable,
			entities.StatusSuspended,
			entities.StatusBlocked,
		},
		entities.StatusSuspended: {
			entities.StatusAvailable,
			entities.StatusBlocked,
		},
	}

	allowedStatuses, exists := allowedTransitions[from]
	if !exists {
		return fmt.Errorf("no transitions allowed from status: %s", from)
	}

	for _, allowedStatus := range allowedStatuses {
		if allowedStatus == to {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", from, to)
}