package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDriver(t *testing.T) {
	// Arrange
	phone := "+79001234567"
	email := "test@example.com"
	firstName := "Иван"
	lastName := "Тестовый"
	licenseNumber := "TEST123456"

	// Act
	driver := NewDriver(phone, email, firstName, lastName, licenseNumber)

	// Assert
	assert.NotEqual(t, uuid.Nil, driver.ID)
	assert.Equal(t, phone, driver.Phone)
	assert.Equal(t, email, driver.Email)
	assert.Equal(t, firstName, driver.FirstName)
	assert.Equal(t, lastName, driver.LastName)
	assert.Equal(t, licenseNumber, driver.LicenseNumber)
	assert.Equal(t, StatusRegistered, driver.Status)
	assert.Equal(t, 0.0, driver.CurrentRating)
	assert.Equal(t, 0, driver.TotalTrips)
	assert.NotNil(t, driver.Metadata)
}

func TestDriver_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"Available driver", StatusAvailable, true},
		{"On shift driver", StatusOnShift, true},
		{"Busy driver", StatusBusy, true},
		{"Registered driver", StatusRegistered, false},
		{"Inactive driver", StatusInactive, false},
		{"Blocked driver", StatusBlocked, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &Driver{Status: tt.status}
			assert.Equal(t, tt.expected, driver.IsActive())
		})
	}
}

func TestDriver_CanReceiveOrders(t *testing.T) {
	tests := []struct {
		name      string
		status    Status
		deletedAt *time.Time
		expected  bool
	}{
		{"Available driver", StatusAvailable, nil, true},
		{"Busy driver", StatusBusy, nil, false},
		{"Available but deleted", StatusAvailable, timePtr(time.Now()), false},
		{"Inactive driver", StatusInactive, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &Driver{
				Status:    tt.status,
				DeletedAt: tt.deletedAt,
			}
			assert.Equal(t, tt.expected, driver.CanReceiveOrders())
		})
	}
}

func TestDriver_IsLicenseExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		licenseExpiry time.Time
		expected      bool
	}{
		{"Valid license", now.Add(24 * time.Hour), false},
		{"Expired license", now.Add(-24 * time.Hour), true},
		{"Expiring today", now, true}, // time.Now().After(now) = true, так как время прошло
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &Driver{LicenseExpiry: tt.licenseExpiry}
			assert.Equal(t, tt.expected, driver.IsLicenseExpired())
		})
	}
}

func TestDriver_GetFullName(t *testing.T) {
	tests := []struct {
		name       string
		firstName  string
		lastName   string
		middleName *string
		expected   string
	}{
		{
			name:       "With middle name",
			firstName:  "Иван",
			lastName:   "Иванов",
			middleName: stringPtr("Иванович"),
			expected:   "Иванов Иван Иванович",
		},
		{
			name:       "Without middle name",
			firstName:  "Иван",
			lastName:   "Иванов",
			middleName: nil,
			expected:   "Иванов Иван",
		},
		{
			name:       "Empty middle name",
			firstName:  "Иван",
			lastName:   "Иванов",
			middleName: stringPtr(""),
			expected:   "Иванов Иван",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driver := &Driver{
				FirstName:  tt.firstName,
				LastName:   tt.lastName,
				MiddleName: tt.middleName,
			}
			assert.Equal(t, tt.expected, driver.GetFullName())
		})
	}
}

func TestDriver_UpdateRating(t *testing.T) {
	// Arrange
	driver := &Driver{
		CurrentRating: 3.5,
		UpdatedAt:     time.Now().Add(-1 * time.Hour),
	}

	oldUpdatedAt := driver.UpdatedAt
	newRating := 4.2

	// Act
	driver.UpdateRating(newRating)

	// Assert
	assert.Equal(t, newRating, driver.CurrentRating)
	assert.True(t, driver.UpdatedAt.After(oldUpdatedAt))
}

func TestDriver_IncrementTripCount(t *testing.T) {
	// Arrange
	driver := &Driver{
		TotalTrips: 5,
		UpdatedAt:  time.Now().Add(-1 * time.Hour),
	}

	oldUpdatedAt := driver.UpdatedAt
	oldTripCount := driver.TotalTrips

	// Act
	driver.IncrementTripCount()

	// Assert
	assert.Equal(t, oldTripCount+1, driver.TotalTrips)
	assert.True(t, driver.UpdatedAt.After(oldUpdatedAt))
}

func TestDriver_ChangeStatus(t *testing.T) {
	// Arrange
	driver := &Driver{
		Status:    StatusRegistered,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	oldUpdatedAt := driver.UpdatedAt
	newStatus := StatusAvailable

	// Act
	driver.ChangeStatus(newStatus)

	// Assert
	assert.Equal(t, newStatus, driver.Status)
	assert.True(t, driver.UpdatedAt.After(oldUpdatedAt))
}

func TestDriver_Validate(t *testing.T) {
	tests := []struct {
		name        string
		driver      *Driver
		expectError bool
		expectedErr error
	}{
		{
			name: "Valid driver",
			driver: &Driver{
				Phone:          "+79001234567",
				Email:          "test@example.com",
				FirstName:      "Иван",
				LastName:       "Иванов",
				LicenseNumber:  "TEST123456",
				PassportSeries: "1234",
				PassportNumber: "567890",
			},
			expectError: false,
		},
		{
			name: "Empty phone",
			driver: &Driver{
				Phone:          "",
				Email:          "test@example.com",
				FirstName:      "Иван",
				LastName:       "Иванов",
				LicenseNumber:  "TEST123456",
				PassportSeries: "1234",
				PassportNumber: "567890",
			},
			expectError: true,
			expectedErr: ErrInvalidPhone,
		},
		{
			name: "Empty email",
			driver: &Driver{
				Phone:          "+79001234567",
				Email:          "",
				FirstName:      "Иван",
				LastName:       "Иванов",
				LicenseNumber:  "TEST123456",
				PassportSeries: "1234",
				PassportNumber: "567890",
			},
			expectError: true,
			expectedErr: ErrInvalidEmail,
		},
		{
			name: "Empty first name",
			driver: &Driver{
				Phone:          "+79001234567",
				Email:          "test@example.com",
				FirstName:      "",
				LastName:       "Иванов",
				LicenseNumber:  "TEST123456",
				PassportSeries: "1234",
				PassportNumber: "567890",
			},
			expectError: true,
			expectedErr: ErrInvalidName,
		},
		{
			name: "Empty license number",
			driver: &Driver{
				Phone:          "+79001234567",
				Email:          "test@example.com",
				FirstName:      "Иван",
				LastName:       "Иванов",
				LicenseNumber:  "",
				PassportSeries: "1234",
				PassportNumber: "567890",
			},
			expectError: true,
			expectedErr: ErrInvalidLicense,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.driver.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDriver_ToSummary(t *testing.T) {
	// Arrange
	driver := &Driver{
		ID:            uuid.New(),
		Phone:         "+79001234567",
		FirstName:     "Иван",
		LastName:      "Иванов",
		MiddleName:    stringPtr("Иванович"),
		Status:        StatusAvailable,
		CurrentRating: 4.5,
		TotalTrips:    25,
	}

	// Act
	summary := driver.ToSummary()

	// Assert
	assert.Equal(t, driver.ID, summary.ID)
	assert.Equal(t, "Иванов Иван Иванович", summary.Name)
	assert.Equal(t, driver.Phone, summary.Phone)
	assert.Equal(t, driver.Status, summary.Status)
	assert.Equal(t, driver.CurrentRating, summary.CurrentRating)
	assert.Equal(t, driver.TotalTrips, summary.TotalTrips)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
