package entities

import (
	"time"

	"github.com/google/uuid"
)

// ShiftStatus статус смены
type ShiftStatus string

const (
	ShiftStatusActive    ShiftStatus = "active"
	ShiftStatusCompleted ShiftStatus = "completed"
	ShiftStatusSuspended ShiftStatus = "suspended"
	ShiftStatusCancelled ShiftStatus = "cancelled"
)

// DriverShift представляет рабочую смену водителя
type DriverShift struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	DriverID        uuid.UUID  `json:"driver_id" db:"driver_id"`
	VehicleID       *uuid.UUID `json:"vehicle_id,omitempty" db:"vehicle_id"`
	StartTime       time.Time  `json:"start_time" db:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty" db:"end_time"`
	Status          ShiftStatus `json:"status" db:"status"`
	StartLatitude   *float64   `json:"start_latitude,omitempty" db:"start_latitude"`
	StartLongitude  *float64   `json:"start_longitude,omitempty" db:"start_longitude"`
	EndLatitude     *float64   `json:"end_latitude,omitempty" db:"end_latitude"`
	EndLongitude    *float64   `json:"end_longitude,omitempty" db:"end_longitude"`
	TotalTrips      int        `json:"total_trips" db:"total_trips"`
	TotalDistance   float64    `json:"total_distance" db:"total_distance"`
	TotalEarnings   float64    `json:"total_earnings" db:"total_earnings"`
	FuelConsumed    *float64   `json:"fuel_consumed,omitempty" db:"fuel_consumed"`
	Metadata        Metadata   `json:"metadata" db:"metadata"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// GetDuration возвращает продолжительность смены в минутах
func (s *DriverShift) GetDuration() int64 {
	if s.EndTime == nil {
		return int64(time.Since(s.StartTime).Minutes())
	}
	return int64(s.EndTime.Sub(s.StartTime).Minutes())
}

// IsActive проверяет, активна ли смена
func (s *DriverShift) IsActive() bool {
	return s.Status == ShiftStatusActive && s.EndTime == nil
}

// End завершает смену
func (s *DriverShift) End(location *DriverLocation) {
	now := time.Now()
	s.EndTime = &now
	s.Status = ShiftStatusCompleted
	s.UpdatedAt = now
	
	if location != nil {
		s.EndLatitude = &location.Latitude
		s.EndLongitude = &location.Longitude
	}
}

// Suspend приостанавливает смену
func (s *DriverShift) Suspend() {
	s.Status = ShiftStatusSuspended
	s.UpdatedAt = time.Now()
}

// Resume возобновляет смену
func (s *DriverShift) Resume() {
	if s.Status == ShiftStatusSuspended {
		s.Status = ShiftStatusActive
		s.UpdatedAt = time.Now()
	}
}

// Cancel отменяет смену
func (s *DriverShift) Cancel() {
	now := time.Now()
	s.Status = ShiftStatusCancelled
	if s.EndTime == nil {
		s.EndTime = &now
	}
	s.UpdatedAt = now
}

// AddTrip добавляет поездку в смену
func (s *DriverShift) AddTrip(distance float64, earnings float64) {
	s.TotalTrips++
	s.TotalDistance += distance
	s.TotalEarnings += earnings
	s.UpdatedAt = time.Now()
}

// GetAverageEarningsPerTrip возвращает средний заработок за поездку
func (s *DriverShift) GetAverageEarningsPerTrip() float64 {
	if s.TotalTrips == 0 {
		return 0
	}
	return s.TotalEarnings / float64(s.TotalTrips)
}

// GetAverageDistancePerTrip возвращает среднее расстояние за поездку
func (s *DriverShift) GetAverageDistancePerTrip() float64 {
	if s.TotalTrips == 0 {
		return 0
	}
	return s.TotalDistance / float64(s.TotalTrips)
}

// GetEarningsPerHour возвращает заработок в час
func (s *DriverShift) GetEarningsPerHour() float64 {
	duration := s.GetDuration()
	if duration == 0 {
		return 0
	}
	hours := float64(duration) / 60.0
	return s.TotalEarnings / hours
}

// GetStartLocation возвращает начальную точку смены
func (s *DriverShift) GetStartLocation() *Location {
	if s.StartLatitude == nil || s.StartLongitude == nil {
		return nil
	}
	return &Location{
		Latitude:  *s.StartLatitude,
		Longitude: *s.StartLongitude,
	}
}

// GetEndLocation возвращает конечную точку смены
func (s *DriverShift) GetEndLocation() *Location {
	if s.EndLatitude == nil || s.EndLongitude == nil {
		return nil
	}
	return &Location{
		Latitude:  *s.EndLatitude,
		Longitude: *s.EndLongitude,
	}
}

// Validate проверяет валидность данных смены
func (s *DriverShift) Validate() error {
	if s.DriverID == uuid.Nil {
		return ErrInvalidDriverID
	}

	if s.StartTime.IsZero() {
		return ErrInvalidStartTime
	}

	if s.EndTime != nil && s.EndTime.Before(s.StartTime) {
		return ErrInvalidEndTime
	}

	return nil
}

// NewDriverShift создает новую смену
func NewDriverShift(driverID uuid.UUID, vehicleID *uuid.UUID, startLocation *DriverLocation) *DriverShift {
	now := time.Now()
	shift := &DriverShift{
		ID:            uuid.New(),
		DriverID:      driverID,
		VehicleID:     vehicleID,
		StartTime:     now,
		Status:        ShiftStatusActive,
		TotalTrips:    0,
		TotalDistance: 0,
		TotalEarnings: 0,
		Metadata:      make(Metadata),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if startLocation != nil {
		shift.StartLatitude = &startLocation.Latitude
		shift.StartLongitude = &startLocation.Longitude
	}

	return shift
}

// ShiftFilters фильтры для поиска смен
type ShiftFilters struct {
	DriverID   *uuid.UUID    `json:"driver_id,omitempty"`
	VehicleID  *uuid.UUID    `json:"vehicle_id,omitempty"`
	Status     []ShiftStatus `json:"status,omitempty"`
	From       *time.Time    `json:"from,omitempty"`
	To         *time.Time    `json:"to,omitempty"`
	MinEarnings *float64     `json:"min_earnings,omitempty"`
	MaxEarnings *float64     `json:"max_earnings,omitempty"`
	MinTrips   *int          `json:"min_trips,omitempty"`
	MaxTrips   *int          `json:"max_trips,omitempty"`
	Limit      int           `json:"limit,omitempty"`
	Offset     int           `json:"offset,omitempty"`
	SortBy     string        `json:"sort_by,omitempty"`
	SortDirection string     `json:"sort_direction,omitempty"`
}

// ShiftStartRequest запрос на начало смены
type ShiftStartRequest struct {
	VehicleID *uuid.UUID `json:"vehicle_id,omitempty"`
	Latitude  *float64   `json:"latitude,omitempty"`
	Longitude *float64   `json:"longitude,omitempty"`
	Notes     *string    `json:"notes,omitempty"`
}

// ShiftEndRequest запрос на завершение смены
type ShiftEndRequest struct {
	Latitude *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Notes    *string  `json:"notes,omitempty"`
}

// ShiftResponse ответ с информацией о смене
type ShiftResponse struct {
	ID              uuid.UUID    `json:"id"`
	DriverID        uuid.UUID    `json:"driver_id"`
	VehicleID       *uuid.UUID   `json:"vehicle_id,omitempty"`
	Status          ShiftStatus  `json:"status"`
	StartTime       time.Time    `json:"start_time"`
	EndTime         *time.Time   `json:"end_time,omitempty"`
	Duration        int64        `json:"duration_minutes"`
	StartLocation   *Location    `json:"start_location,omitempty"`
	EndLocation     *Location    `json:"end_location,omitempty"`
	TotalTrips      int          `json:"total_trips"`
	TotalDistance   float64      `json:"total_distance"`
	TotalEarnings   float64      `json:"total_earnings"`
	FuelConsumed    *float64     `json:"fuel_consumed,omitempty"`
	EarningsPerHour float64      `json:"earnings_per_hour"`
	AvgTripDistance float64      `json:"avg_trip_distance"`
	AvgTripEarnings float64      `json:"avg_trip_earnings"`
}

// ToResponse конвертирует в ответ
func (s *DriverShift) ToResponse() *ShiftResponse {
	return &ShiftResponse{
		ID:              s.ID,
		DriverID:        s.DriverID,
		VehicleID:       s.VehicleID,
		Status:          s.Status,
		StartTime:       s.StartTime,
		EndTime:         s.EndTime,
		Duration:        s.GetDuration(),
		StartLocation:   s.GetStartLocation(),
		EndLocation:     s.GetEndLocation(),
		TotalTrips:      s.TotalTrips,
		TotalDistance:   s.TotalDistance,
		TotalEarnings:   s.TotalEarnings,
		FuelConsumed:    s.FuelConsumed,
		EarningsPerHour: s.GetEarningsPerHour(),
		AvgTripDistance: s.GetAverageDistancePerTrip(),
		AvgTripEarnings: s.GetAverageEarningsPerTrip(),
	}
}

// ShiftSummary краткая информация о смене
type ShiftSummary struct {
	ID            uuid.UUID   `json:"id"`
	Status        ShiftStatus `json:"status"`
	StartTime     time.Time   `json:"start_time"`
	Duration      int64       `json:"duration_minutes"`
	TotalTrips    int         `json:"total_trips"`
	TotalEarnings float64     `json:"total_earnings"`
}

// ToSummary возвращает краткую информацию о смене
func (s *DriverShift) ToSummary() *ShiftSummary {
	return &ShiftSummary{
		ID:            s.ID,
		Status:        s.Status,
		StartTime:     s.StartTime,
		Duration:      s.GetDuration(),
		TotalTrips:    s.TotalTrips,
		TotalEarnings: s.TotalEarnings,
	}
}

// ShiftStats статистика по сменам
type ShiftStats struct {
	TotalShifts     int     `json:"total_shifts"`
	ActiveShifts    int     `json:"active_shifts"`
	CompletedShifts int     `json:"completed_shifts"`
	TotalHours      float64 `json:"total_hours"`
	TotalEarnings   float64 `json:"total_earnings"`
	TotalTrips      int     `json:"total_trips"`
	TotalDistance   float64 `json:"total_distance"`
	AvgShiftDuration float64 `json:"avg_shift_duration_hours"`
	AvgShiftEarnings float64 `json:"avg_shift_earnings"`
	AvgHourlyRate   float64 `json:"avg_hourly_rate"`
}