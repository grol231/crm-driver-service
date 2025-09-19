package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Status статус водителя
type Status string

const (
	StatusRegistered          Status = "registered"
	StatusPendingVerification Status = "pending_verification"
	StatusVerified            Status = "verified"
	StatusRejected            Status = "rejected"
	StatusAvailable           Status = "available"
	StatusOnShift             Status = "on_shift"
	StatusBusy                Status = "busy"
	StatusInactive            Status = "inactive"
	StatusSuspended           Status = "suspended"
	StatusBlocked             Status = "blocked"
)

// Metadata дополнительные данные в формате JSON
type Metadata map[string]interface{}

// Value реализует интерфейс driver.Valuer для сериализации в БД
func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// Scan реализует интерфейс sql.Scanner для десериализации из БД
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(Metadata)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into Metadata", value)
	}

	return json.Unmarshal(bytes, m)
}

// Driver представляет водителя в системе
type Driver struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	Phone          string     `json:"phone" db:"phone"`
	Email          string     `json:"email" db:"email"`
	FirstName      string     `json:"first_name" db:"first_name"`
	LastName       string     `json:"last_name" db:"last_name"`
	MiddleName     *string    `json:"middle_name,omitempty" db:"middle_name"`
	BirthDate      time.Time  `json:"birth_date" db:"birth_date"`
	PassportSeries string     `json:"passport_series" db:"passport_series"`
	PassportNumber string     `json:"passport_number" db:"passport_number"`
	LicenseNumber  string     `json:"license_number" db:"license_number"`
	LicenseExpiry  time.Time  `json:"license_expiry" db:"license_expiry"`
	Status         Status     `json:"status" db:"status"`
	CurrentRating  float64    `json:"current_rating" db:"current_rating"`
	TotalTrips     int        `json:"total_trips" db:"total_trips"`
	Metadata       Metadata   `json:"metadata" db:"metadata"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsActive проверяет, активен ли водитель
func (d *Driver) IsActive() bool {
	return d.Status == StatusAvailable || d.Status == StatusOnShift || d.Status == StatusBusy
}

// CanReceiveOrders проверяет, может ли водитель получать заказы
func (d *Driver) CanReceiveOrders() bool {
	return d.Status == StatusAvailable && d.DeletedAt == nil
}

// UpdateRating обновляет рейтинг водителя
func (d *Driver) UpdateRating(newRating float64) {
	d.CurrentRating = newRating
	d.UpdatedAt = time.Now()
}

// IncrementTripCount увеличивает счетчик поездок
func (d *Driver) IncrementTripCount() {
	d.TotalTrips++
	d.UpdatedAt = time.Now()
}

// ChangeStatus изменяет статус водителя
func (d *Driver) ChangeStatus(newStatus Status) {
	d.Status = newStatus
	d.UpdatedAt = time.Now()
}

// IsLicenseExpired проверяет, не истекло ли водительское удостоверение
func (d *Driver) IsLicenseExpired() bool {
	return time.Now().After(d.LicenseExpiry)
}

// GetFullName возвращает полное имя водителя
func (d *Driver) GetFullName() string {
	if d.MiddleName != nil && *d.MiddleName != "" {
		return d.LastName + " " + d.FirstName + " " + *d.MiddleName
	}
	return d.LastName + " " + d.FirstName
}

// Validate проверяет валидность данных водителя
func (d *Driver) Validate() error {
	if d.Phone == "" {
		return ErrInvalidPhone
	}

	if d.Email == "" {
		return ErrInvalidEmail
	}

	if d.FirstName == "" || d.LastName == "" {
		return ErrInvalidName
	}

	if d.LicenseNumber == "" {
		return ErrInvalidLicense
	}

	if d.PassportSeries == "" || d.PassportNumber == "" {
		return ErrInvalidPassport
	}

	return nil
}

// NewDriver создает нового водителя
func NewDriver(phone, email, firstName, lastName, licenseNumber string) *Driver {
	now := time.Now()
	return &Driver{
		ID:            uuid.New(),
		Phone:         phone,
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		LicenseNumber: licenseNumber,
		Status:        StatusRegistered,
		CurrentRating: 0.0,
		TotalTrips:    0,
		Metadata:      make(Metadata),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// DriverFilters фильтры для поиска водителей
type DriverFilters struct {
	Status        []Status   `json:"status,omitempty"`
	MinRating     *float64   `json:"min_rating,omitempty"`
	MaxRating     *float64   `json:"max_rating,omitempty"`
	City          *string    `json:"city,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	Offset        int        `json:"offset,omitempty"`
	SortBy        string     `json:"sort_by,omitempty"`
	SortDirection string     `json:"sort_direction,omitempty"`
}

// DriverSummary краткая информация о водителе
type DriverSummary struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Phone         string    `json:"phone"`
	Status        Status    `json:"status"`
	CurrentRating float64   `json:"current_rating"`
	TotalTrips    int       `json:"total_trips"`
}

// ToSummary возвращает краткую информацию о водителе
func (d *Driver) ToSummary() *DriverSummary {
	return &DriverSummary{
		ID:            d.ID,
		Name:          d.GetFullName(),
		Phone:         d.Phone,
		Status:        d.Status,
		CurrentRating: d.CurrentRating,
		TotalTrips:    d.TotalTrips,
	}
}
