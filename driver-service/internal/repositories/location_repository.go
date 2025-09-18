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
	"go.uber.org/zap"
)

// LocationRepository интерфейс для работы с местоположениями водителей
type LocationRepository interface {
	Create(ctx context.Context, location *entities.DriverLocation) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverLocation, error)
	GetLatestByDriverID(ctx context.Context, driverID uuid.UUID) (*entities.DriverLocation, error)
	GetByDriverIDInTimeRange(ctx context.Context, driverID uuid.UUID, from, to time.Time) ([]*entities.DriverLocation, error)
	List(ctx context.Context, filters *entities.LocationFilters) ([]*entities.DriverLocation, error)
	CreateBatch(ctx context.Context, locations []*entities.DriverLocation) error
	DeleteOld(ctx context.Context, olderThan time.Time) error
	GetNearby(ctx context.Context, lat, lon, radiusKm float64, limit int) ([]*entities.DriverLocation, error)
}

type locationRepository struct {
	db     *database.DB
	logger *zap.Logger
}

func NewLocationRepository(db *database.DB, logger *zap.Logger) LocationRepository {
	return &locationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *locationRepository) Create(ctx context.Context, location *entities.DriverLocation) error {
	query := `
		INSERT INTO driver_locations (
			id, driver_id, latitude, longitude, altitude, accuracy,
			speed, bearing, address, metadata, recorded_at, created_at
		) VALUES (
			:id, :driver_id, :latitude, :longitude, :altitude, :accuracy,
			:speed, :bearing, :address, :metadata, :recorded_at, :created_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, location)
	return err
}

func (r *locationRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverLocation, error) {
	var location entities.DriverLocation
	query := `SELECT * FROM driver_locations WHERE id = $1`

	err := r.db.GetContext(ctx, &location, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrLocationNotFound
		}
		return nil, err
	}
	return &location, nil
}

func (r *locationRepository) GetLatestByDriverID(ctx context.Context, driverID uuid.UUID) (*entities.DriverLocation, error) {
	var location entities.DriverLocation
	query := `
		SELECT * FROM driver_locations 
		WHERE driver_id = $1 
		ORDER BY recorded_at DESC 
		LIMIT 1`

	err := r.db.GetContext(ctx, &location, query, driverID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrLocationNotFound
		}
		return nil, err
	}
	return &location, nil
}

func (r *locationRepository) GetByDriverIDInTimeRange(ctx context.Context, driverID uuid.UUID, from, to time.Time) ([]*entities.DriverLocation, error) {
	var locations []*entities.DriverLocation
	query := `
		SELECT * FROM driver_locations 
		WHERE driver_id = $1 AND recorded_at BETWEEN $2 AND $3
		ORDER BY recorded_at ASC`

	err := r.db.SelectContext(ctx, &locations, query, driverID, from, to)
	return locations, err
}

func (r *locationRepository) List(ctx context.Context, filters *entities.LocationFilters) ([]*entities.DriverLocation, error) {
	query, args := r.buildListQuery(filters)
	
	var locations []*entities.DriverLocation
	err := r.db.SelectContext(ctx, &locations, query, args...)
	return locations, err
}

func (r *locationRepository) CreateBatch(ctx context.Context, locations []*entities.DriverLocation) error {
	if len(locations) == 0 {
		return nil
	}

	query := `
		INSERT INTO driver_locations (
			id, driver_id, latitude, longitude, altitude, accuracy,
			speed, bearing, address, metadata, recorded_at, created_at
		) VALUES (
			:id, :driver_id, :latitude, :longitude, :altitude, :accuracy,
			:speed, :bearing, :address, :metadata, :recorded_at, :created_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, locations)
	return err
}

func (r *locationRepository) DeleteOld(ctx context.Context, olderThan time.Time) error {
	query := `DELETE FROM driver_locations WHERE recorded_at < $1`
	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Info("Deleted old locations", zap.Int64("rows_affected", rowsAffected))
	return nil
}

func (r *locationRepository) GetNearby(ctx context.Context, lat, lon, radiusKm float64, limit int) ([]*entities.DriverLocation, error) {
	query := `
		SELECT DISTINCT ON (driver_id) *
		FROM driver_locations 
		WHERE point(longitude, latitude) <@> point($1, $2) <= $3
		ORDER BY driver_id, recorded_at DESC
		LIMIT $4`

	var locations []*entities.DriverLocation
	err := r.db.SelectContext(ctx, &locations, query, lon, lat, radiusKm, limit)
	return locations, err
}

func (r *locationRepository) buildListQuery(filters *entities.LocationFilters) (string, []interface{}) {
	query := "SELECT * FROM driver_locations WHERE 1=1"
	var args []interface{}
	argCount := 0

	if filters != nil {
		if filters.DriverID != nil {
			argCount++
			query += fmt.Sprintf(" AND driver_id = $%d", argCount)
			args = append(args, *filters.DriverID)
		}

		if filters.From != nil {
			argCount++
			query += fmt.Sprintf(" AND recorded_at >= $%d", argCount)
			args = append(args, *filters.From)
		}

		if filters.To != nil {
			argCount++
			query += fmt.Sprintf(" AND recorded_at <= $%d", argCount)
			args = append(args, *filters.To)
		}

		query += " ORDER BY recorded_at DESC"

		if filters.Limit > 0 {
			argCount++
			query += fmt.Sprintf(" LIMIT $%d", argCount)
			args = append(args, filters.Limit)
		}
	}

	return query, args
}