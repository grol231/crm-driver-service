package entities

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// DriverLocation представляет местоположение водителя
type DriverLocation struct {
	ID         uuid.UUID `json:"id" db:"id"`
	DriverID   uuid.UUID `json:"driver_id" db:"driver_id"`
	Latitude   float64   `json:"latitude" db:"latitude"`
	Longitude  float64   `json:"longitude" db:"longitude"`
	Altitude   *float64  `json:"altitude,omitempty" db:"altitude"`
	Accuracy   *float64  `json:"accuracy,omitempty" db:"accuracy"`
	Speed      *float64  `json:"speed,omitempty" db:"speed"`
	Bearing    *float64  `json:"bearing,omitempty" db:"bearing"`
	Address    *string   `json:"address,omitempty" db:"address"`
	Metadata   Metadata  `json:"metadata" db:"metadata"`
	RecordedAt time.Time `json:"recorded_at" db:"recorded_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Location базовая структура для координат
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
}

// DistanceTo вычисляет расстояние до другой точки в километрах (формула гаверсинуса)
func (dl *DriverLocation) DistanceTo(other *DriverLocation) float64 {
	const earthRadiusKm = 6371.0

	lat1Rad := dl.Latitude * math.Pi / 180
	lon1Rad := dl.Longitude * math.Pi / 180
	lat2Rad := other.Latitude * math.Pi / 180
	lon2Rad := other.Longitude * math.Pi / 180

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// IsValidLocation проверяет валидность координат
func (dl *DriverLocation) IsValidLocation() bool {
	return dl.Latitude >= -90 && dl.Latitude <= 90 &&
		dl.Longitude >= -180 && dl.Longitude <= 180
}

// IsInRadius проверяет, находится ли точка в радиусе от другой точки
func (dl *DriverLocation) IsInRadius(center *DriverLocation, radiusKm float64) bool {
	distance := dl.DistanceTo(center)
	return distance <= radiusKm
}

// GetSpeed возвращает скорость в км/ч
func (dl *DriverLocation) GetSpeed() float64 {
	if dl.Speed == nil {
		return 0
	}
	return *dl.Speed
}

// GetBearing возвращает направление в градусах
func (dl *DriverLocation) GetBearing() float64 {
	if dl.Bearing == nil {
		return 0
	}
	return *dl.Bearing
}

// GetAccuracy возвращает точность в метрах
func (dl *DriverLocation) GetAccuracy() float64 {
	if dl.Accuracy == nil {
		return 0
	}
	return *dl.Accuracy
}

// IsHighAccuracy проверяет высокую точность (меньше 50 метров)
func (dl *DriverLocation) IsHighAccuracy() bool {
	return dl.GetAccuracy() > 0 && dl.GetAccuracy() < 50
}

// Validate проверяет валидность данных местоположения
func (dl *DriverLocation) Validate() error {
	if dl.DriverID == uuid.Nil {
		return ErrInvalidDriverID
	}

	if !dl.IsValidLocation() {
		return ErrInvalidLocation
	}

	if dl.RecordedAt.IsZero() {
		return ErrInvalidTimestamp
	}

	return nil
}

// NewDriverLocation создает новое местоположение водителя
func NewDriverLocation(driverID uuid.UUID, lat, lon float64, recordedAt time.Time) *DriverLocation {
	now := time.Now()
	return &DriverLocation{
		ID:         uuid.New(),
		DriverID:   driverID,
		Latitude:   lat,
		Longitude:  lon,
		Metadata:   make(Metadata),
		RecordedAt: recordedAt,
		CreatedAt:  now,
	}
}

// LocationFilters фильтры для поиска местоположений
type LocationFilters struct {
	DriverID  *uuid.UUID `json:"driver_id,omitempty"`
	From      *time.Time `json:"from,omitempty"`
	To        *time.Time `json:"to,omitempty"`
	Bounds    *GeoBounds `json:"bounds,omitempty"`
	MinSpeed  *float64   `json:"min_speed,omitempty"`
	MaxSpeed  *float64   `json:"max_speed,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// GeoBounds географические границы
type GeoBounds struct {
	NorthEast Location `json:"north_east"`
	SouthWest Location `json:"south_west"`
}

// LocationUpdateRequest запрос на обновление местоположения
type LocationUpdateRequest struct {
	Latitude  float64  `json:"latitude" binding:"required"`
	Longitude float64  `json:"longitude" binding:"required"`
	Altitude  *float64 `json:"altitude,omitempty"`
	Accuracy  *float64 `json:"accuracy,omitempty"`
	Speed     *float64 `json:"speed,omitempty"`
	Bearing   *float64 `json:"bearing,omitempty"`
	Timestamp *int64   `json:"timestamp,omitempty"`
}

// LocationResponse ответ с местоположением
type LocationResponse struct {
	ID         uuid.UUID `json:"id"`
	DriverID   uuid.UUID `json:"driver_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Altitude   *float64  `json:"altitude,omitempty"`
	Accuracy   *float64  `json:"accuracy,omitempty"`
	Speed      *float64  `json:"speed,omitempty"`
	Bearing    *float64  `json:"bearing,omitempty"`
	Address    *string   `json:"address,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
}

// ToResponse конвертирует в ответ
func (dl *DriverLocation) ToResponse() *LocationResponse {
	return &LocationResponse{
		ID:         dl.ID,
		DriverID:   dl.DriverID,
		Latitude:   dl.Latitude,
		Longitude:  dl.Longitude,
		Altitude:   dl.Altitude,
		Accuracy:   dl.Accuracy,
		Speed:      dl.Speed,
		Bearing:    dl.Bearing,
		Address:    dl.Address,
		RecordedAt: dl.RecordedAt,
	}
}

// ToLocation конвертирует в базовую структуру Location
func (dl *DriverLocation) ToLocation() Location {
	address := ""
	if dl.Address != nil {
		address = *dl.Address
	}
	
	return Location{
		Latitude:  dl.Latitude,
		Longitude: dl.Longitude,
		Address:   address,
	}
}

// LocationStats статистика по местоположению
type LocationStats struct {
	TotalPoints    int     `json:"total_points"`
	DistanceTraveled float64 `json:"distance_traveled_km"`
	AverageSpeed   float64 `json:"average_speed_kmh"`
	MaxSpeed       float64 `json:"max_speed_kmh"`
	TimeSpan       int64   `json:"time_span_minutes"`
}

// CalculateLocationStats вычисляет статистику по массиву точек
func CalculateLocationStats(locations []*DriverLocation) *LocationStats {
	if len(locations) == 0 {
		return &LocationStats{}
	}

	stats := &LocationStats{
		TotalPoints: len(locations),
	}

	if len(locations) == 1 {
		return stats
	}

	var totalDistance float64
	var totalSpeed float64
	var maxSpeed float64
	var speedCount int

	for i := 1; i < len(locations); i++ {
		prev := locations[i-1]
		curr := locations[i]

		// Расчет расстояния
		distance := prev.DistanceTo(curr)
		totalDistance += distance

		// Расчет скорости
		if curr.Speed != nil {
			speed := *curr.Speed
			totalSpeed += speed
			speedCount++
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
	}

	stats.DistanceTraveled = totalDistance
	stats.MaxSpeed = maxSpeed

	if speedCount > 0 {
		stats.AverageSpeed = totalSpeed / float64(speedCount)
	}

	// Временной промежуток
	firstTime := locations[0].RecordedAt
	lastTime := locations[len(locations)-1].RecordedAt
	stats.TimeSpan = int64(lastTime.Sub(firstTime).Minutes())

	return stats
}