package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/infrastructure/database"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// DriverRepository интерфейс для работы с водителями
type DriverRepository interface {
	Create(ctx context.Context, driver *entities.Driver) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Driver, error)
	GetByPhone(ctx context.Context, phone string) (*entities.Driver, error)
	GetByEmail(ctx context.Context, email string) (*entities.Driver, error)
	GetByLicenseNumber(ctx context.Context, licenseNumber string) (*entities.Driver, error)
	Update(ctx context.Context, driver *entities.Driver) error
	Delete(ctx context.Context, id uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filters *entities.DriverFilters) ([]*entities.Driver, error)
	Count(ctx context.Context, filters *entities.DriverFilters) (int, error)
	Exists(ctx context.Context, phone, licenseNumber string) (bool, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.Status) error
	UpdateRating(ctx context.Context, id uuid.UUID, rating float64) error
	IncrementTripCount(ctx context.Context, id uuid.UUID) error
	GetActiveDrivers(ctx context.Context) ([]*entities.Driver, error)
}

// driverRepository реализация DriverRepository
type driverRepository struct {
	db     *database.DB
	logger *zap.Logger
}

// NewDriverRepository создает новый репозиторий водителей
func NewDriverRepository(db *database.DB, logger *zap.Logger) DriverRepository {
	return &driverRepository{
		db:     db,
		logger: logger,
	}
}

// Create создает нового водителя
func (r *driverRepository) Create(ctx context.Context, driver *entities.Driver) error {
	query := `
		INSERT INTO drivers (
			id, phone, email, first_name, last_name, middle_name,
			birth_date, passport_series, passport_number, license_number,
			license_expiry, status, current_rating, total_trips, metadata,
			created_at, updated_at
		) VALUES (
			:id, :phone, :email, :first_name, :last_name, :middle_name,
			:birth_date, :passport_series, :passport_number, :license_number,
			:license_expiry, :status, :current_rating, :total_trips, :metadata,
			:created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, driver)
	if err != nil {
		r.logger.Error("Failed to create driver",
			zap.Error(err),
			zap.String("driver_id", driver.ID.String()),
			zap.String("phone", driver.Phone),
		)
		return fmt.Errorf("failed to create driver: %w", err)
	}

	r.logger.Info("Driver created successfully",
		zap.String("driver_id", driver.ID.String()),
		zap.String("phone", driver.Phone),
	)

	return nil
}

// GetByID получает водителя по ID
func (r *driverRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Driver, error) {
	var driver entities.Driver
	query := `
		SELECT * FROM drivers 
		WHERE id = $1 AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &driver, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrDriverNotFound
		}
		r.logger.Error("Failed to get driver by ID",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get driver by ID: %w", err)
	}

	return &driver, nil
}

// GetByPhone получает водителя по номеру телефона
func (r *driverRepository) GetByPhone(ctx context.Context, phone string) (*entities.Driver, error) {
	var driver entities.Driver
	query := `
		SELECT * FROM drivers 
		WHERE phone = $1 AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &driver, query, phone)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrDriverNotFound
		}
		r.logger.Error("Failed to get driver by phone",
			zap.Error(err),
			zap.String("phone", phone),
		)
		return nil, fmt.Errorf("failed to get driver by phone: %w", err)
	}

	return &driver, nil
}

// GetByEmail получает водителя по email
func (r *driverRepository) GetByEmail(ctx context.Context, email string) (*entities.Driver, error) {
	var driver entities.Driver
	query := `
		SELECT * FROM drivers 
		WHERE email = $1 AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &driver, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrDriverNotFound
		}
		r.logger.Error("Failed to get driver by email",
			zap.Error(err),
			zap.String("email", email),
		)
		return nil, fmt.Errorf("failed to get driver by email: %w", err)
	}

	return &driver, nil
}

// GetByLicenseNumber получает водителя по номеру водительского удостоверения
func (r *driverRepository) GetByLicenseNumber(ctx context.Context, licenseNumber string) (*entities.Driver, error) {
	var driver entities.Driver
	query := `
		SELECT * FROM drivers 
		WHERE license_number = $1 AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &driver, query, licenseNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrDriverNotFound
		}
		r.logger.Error("Failed to get driver by license number",
			zap.Error(err),
			zap.String("license_number", licenseNumber),
		)
		return nil, fmt.Errorf("failed to get driver by license number: %w", err)
	}

	return &driver, nil
}

// Update обновляет данные водителя
func (r *driverRepository) Update(ctx context.Context, driver *entities.Driver) error {
	driver.UpdatedAt = time.Now()
	
	query := `
		UPDATE drivers SET
			phone = :phone, email = :email, first_name = :first_name,
			last_name = :last_name, middle_name = :middle_name,
			birth_date = :birth_date, passport_series = :passport_series,
			passport_number = :passport_number, license_number = :license_number,
			license_expiry = :license_expiry, status = :status,
			current_rating = :current_rating, total_trips = :total_trips,
			metadata = :metadata, updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL`

	result, err := r.db.NamedExecContext(ctx, query, driver)
	if err != nil {
		r.logger.Error("Failed to update driver",
			zap.Error(err),
			zap.String("driver_id", driver.ID.String()),
		)
		return fmt.Errorf("failed to update driver: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDriverNotFound
	}

	r.logger.Info("Driver updated successfully",
		zap.String("driver_id", driver.ID.String()),
	)

	return nil
}

// Delete удаляет водителя (жесткое удаление)
func (r *driverRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM drivers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete driver",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
		return fmt.Errorf("failed to delete driver: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDriverNotFound
	}

	r.logger.Info("Driver deleted successfully",
		zap.String("driver_id", id.String()),
	)

	return nil
}

// SoftDelete мягкое удаление водителя
func (r *driverRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE drivers 
		SET deleted_at = $1, updated_at = $1 
		WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		r.logger.Error("Failed to soft delete driver",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
		return fmt.Errorf("failed to soft delete driver: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDriverNotFound
	}

	r.logger.Info("Driver soft deleted successfully",
		zap.String("driver_id", id.String()),
	)

	return nil
}

// List получает список водителей с фильтрами
func (r *driverRepository) List(ctx context.Context, filters *entities.DriverFilters) ([]*entities.Driver, error) {
	query, args, err := r.buildListQuery(filters, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build list query: %w", err)
	}

	var drivers []*entities.Driver
	err = r.db.SelectContext(ctx, &drivers, query, args...)
	if err != nil {
		r.logger.Error("Failed to list drivers",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to list drivers: %w", err)
	}

	return drivers, nil
}

// Count возвращает количество водителей с фильтрами
func (r *driverRepository) Count(ctx context.Context, filters *entities.DriverFilters) (int, error) {
	query, args, err := r.buildListQuery(filters, true)
	if err != nil {
		return 0, fmt.Errorf("failed to build count query: %w", err)
	}

	var count int
	err = r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		r.logger.Error("Failed to count drivers",
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to count drivers: %w", err)
	}

	return count, nil
}

// Exists проверяет существование водителя по телефону или номеру лицензии
func (r *driverRepository) Exists(ctx context.Context, phone, licenseNumber string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM drivers 
			WHERE (phone = $1 OR license_number = $2) 
			AND deleted_at IS NULL
		)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, phone, licenseNumber)
	if err != nil {
		r.logger.Error("Failed to check driver existence",
			zap.Error(err),
			zap.String("phone", phone),
			zap.String("license_number", licenseNumber),
		)
		return false, fmt.Errorf("failed to check driver existence: %w", err)
	}

	return exists, nil
}

// UpdateStatus обновляет статус водителя
func (r *driverRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.Status) error {
	query := `
		UPDATE drivers 
		SET status = $1, updated_at = $2 
		WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		r.logger.Error("Failed to update driver status",
			zap.Error(err),
			zap.String("driver_id", id.String()),
			zap.String("status", string(status)),
		)
		return fmt.Errorf("failed to update driver status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDriverNotFound
	}

	r.logger.Info("Driver status updated successfully",
		zap.String("driver_id", id.String()),
		zap.String("status", string(status)),
	)

	return nil
}

// UpdateRating обновляет рейтинг водителя
func (r *driverRepository) UpdateRating(ctx context.Context, id uuid.UUID, rating float64) error {
	query := `
		UPDATE drivers 
		SET current_rating = $1, updated_at = $2 
		WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, rating, time.Now(), id)
	if err != nil {
		r.logger.Error("Failed to update driver rating",
			zap.Error(err),
			zap.String("driver_id", id.String()),
			zap.Float64("rating", rating),
		)
		return fmt.Errorf("failed to update driver rating: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDriverNotFound
	}

	return nil
}

// IncrementTripCount увеличивает счетчик поездок
func (r *driverRepository) IncrementTripCount(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE drivers 
		SET total_trips = total_trips + 1, updated_at = $1 
		WHERE id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		r.logger.Error("Failed to increment trip count",
			zap.Error(err),
			zap.String("driver_id", id.String()),
		)
		return fmt.Errorf("failed to increment trip count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDriverNotFound
	}

	return nil
}

// GetActiveDrivers получает список активных водителей
func (r *driverRepository) GetActiveDrivers(ctx context.Context) ([]*entities.Driver, error) {
	query := `
		SELECT * FROM drivers 
		WHERE status IN ('available', 'on_shift', 'busy') 
		AND deleted_at IS NULL
		ORDER BY current_rating DESC`

	var drivers []*entities.Driver
	err := r.db.SelectContext(ctx, &drivers, query)
	if err != nil {
		r.logger.Error("Failed to get active drivers",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get active drivers: %w", err)
	}

	return drivers, nil
}

// buildListQuery строит SQL запрос для получения списка водителей
func (r *driverRepository) buildListQuery(filters *entities.DriverFilters, isCount bool) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}
	argCount := 0

	// Базовый запрос
	baseQuery := "FROM drivers WHERE deleted_at IS NULL"
	
	var selectClause string
	if isCount {
		selectClause = "SELECT COUNT(*) "
	} else {
		selectClause = "SELECT * "
	}

	// Добавляем условия фильтрации
	if filters != nil {
		if len(filters.Status) > 0 {
			placeholders := make([]string, len(filters.Status))
			for i, status := range filters.Status {
				argCount++
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, status)
			}
			conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
		}

		if filters.MinRating != nil {
			argCount++
			conditions = append(conditions, fmt.Sprintf("current_rating >= $%d", argCount))
			args = append(args, *filters.MinRating)
		}

		if filters.MaxRating != nil {
			argCount++
			conditions = append(conditions, fmt.Sprintf("current_rating <= $%d", argCount))
			args = append(args, *filters.MaxRating)
		}

		if filters.CreatedAfter != nil {
			argCount++
			conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argCount))
			args = append(args, *filters.CreatedAfter)
		}

		if filters.CreatedBefore != nil {
			argCount++
			conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argCount))
			args = append(args, *filters.CreatedBefore)
		}
	}

	// Собираем WHERE условия
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	query := selectClause + baseQuery + whereClause

	// Для запросов списка добавляем сортировку и пагинацию
	if !isCount && filters != nil {
		// Сортировка
		orderBy := "ORDER BY created_at DESC"
		if filters.SortBy != "" {
			direction := "ASC"
			if filters.SortDirection == "desc" {
				direction = "DESC"
			}
			orderBy = fmt.Sprintf("ORDER BY %s %s", filters.SortBy, direction)
		}
		query += " " + orderBy

		// Пагинация
		if filters.Limit > 0 {
			argCount++
			query += fmt.Sprintf(" LIMIT $%d", argCount)
			args = append(args, filters.Limit)
		}

		if filters.Offset > 0 {
			argCount++
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, filters.Offset)
		}
	}

	return query, args, nil
}