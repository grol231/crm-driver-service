//go:build integration

package fixtures

import (
	"fmt"
	"time"

	"driver-service/internal/domain/entities"

	"github.com/google/uuid"
)

// CreateTestDriver создает тестового водителя
func CreateTestDriver() *entities.Driver {
	now := time.Now()
	return &entities.Driver{
		ID:             uuid.New(),
		Phone:          "+79001234567",
		Email:          "test.driver@example.com",
		FirstName:      "Иван",
		LastName:       "Тестовый",
		MiddleName:     stringPtr("Иванович"),
		BirthDate:      time.Date(1985, 5, 15, 0, 0, 0, 0, time.UTC),
		PassportSeries: "1234",
		PassportNumber: "567890",
		LicenseNumber:  "TEST123456",
		LicenseExpiry:  time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Status:         entities.StatusRegistered,
		CurrentRating:  0.0,
		TotalTrips:     0,
		Metadata:       make(entities.Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CreateTestDriverWithStatus создает тестового водителя с определенным статусом
func CreateTestDriverWithStatus(status entities.Status) *entities.Driver {
	driver := CreateTestDriver()
	driver.Status = status
	return driver
}

// CreateTestDriverWithRating создает тестового водителя с рейтингом
func CreateTestDriverWithRating(rating float64, trips int) *entities.Driver {
	driver := CreateTestDriver()
	driver.CurrentRating = rating
	driver.TotalTrips = trips
	return driver
}

// CreateTestLocation создает тестовое местоположение
func CreateTestLocation(driverID uuid.UUID) *entities.DriverLocation {
	now := time.Now()
	return &entities.DriverLocation{
		ID:         uuid.New(),
		DriverID:   driverID,
		Latitude:   55.7558, // Москва, Красная площадь
		Longitude:  37.6173,
		Altitude:   floatPtr(150.0),
		Accuracy:   floatPtr(10.0),
		Speed:      floatPtr(60.5),
		Bearing:    floatPtr(45.0),
		Address:    stringPtr("Красная площадь, 1, Москва"),
		Metadata:   make(entities.Metadata),
		RecordedAt: now,
		CreatedAt:  now,
	}
}

// CreateTestLocationWithCoords создает тестовое местоположение с конкретными координатами
func CreateTestLocationWithCoords(driverID uuid.UUID, lat, lon float64) *entities.DriverLocation {
	location := CreateTestLocation(driverID)
	location.Latitude = lat
	location.Longitude = lon
	return location
}

// CreateTestDocument создает тестовый документ
func CreateTestDocument(driverID uuid.UUID, docType entities.DocumentType) *entities.DriverDocument {
	now := time.Now()
	return &entities.DriverDocument{
		ID:             uuid.New(),
		DriverID:       driverID,
		DocumentType:   docType,
		DocumentNumber: "TEST-DOC-123456",
		IssueDate:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiryDate:     time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		FileURL:        "https://example.com/documents/test-doc.pdf",
		Status:         entities.VerificationStatusPending,
		Metadata:       make(entities.Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CreateTestShift создает тестовую смену
func CreateTestShift(driverID uuid.UUID) *entities.DriverShift {
	now := time.Now()
	vehicleID := uuid.New()

	return &entities.DriverShift{
		ID:             uuid.New(),
		DriverID:       driverID,
		VehicleID:      &vehicleID,
		StartTime:      now,
		Status:         entities.ShiftStatusActive,
		StartLatitude:  floatPtr(55.7558),
		StartLongitude: floatPtr(37.6173),
		TotalTrips:     0,
		TotalDistance:  0.0,
		TotalEarnings:  0.0,
		Metadata:       make(entities.Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CreateTestRating создает тестовую оценку
func CreateTestRating(driverID uuid.UUID, rating int) *entities.DriverRating {
	now := time.Now()
	orderID := uuid.New()
	customerID := uuid.New()

	return &entities.DriverRating{
		ID:         uuid.New(),
		DriverID:   driverID,
		OrderID:    &orderID,
		CustomerID: &customerID,
		Rating:     rating,
		Comment:    stringPtr("Отличный водитель!"),
		RatingType: entities.RatingTypeCustomer,
		CriteriaScores: map[string]int{
			"punctuality": rating,
			"politeness":  rating,
			"driving":     rating,
		},
		IsVerified:  false,
		IsAnonymous: false,
		Metadata:    make(entities.Metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// CreateMultipleTestDrivers создает несколько тестовых водителей
func CreateMultipleTestDrivers(count int) []*entities.Driver {
	drivers := make([]*entities.Driver, count)

	for i := 0; i < count; i++ {
		driver := CreateTestDriver()
		driver.Phone = fmt.Sprintf("+7900123456%d", i)
		driver.Email = fmt.Sprintf("driver%d@example.com", i)
		driver.LicenseNumber = fmt.Sprintf("TEST%06d", i)
		drivers[i] = driver
	}

	return drivers
}

// CreateTestLocationHistory создает историю местоположений
func CreateTestLocationHistory(driverID uuid.UUID, count int, interval time.Duration) []*entities.DriverLocation {
	locations := make([]*entities.DriverLocation, count)
	baseTime := time.Now().Add(-time.Duration(count) * interval)

	// Начальные координаты (Москва)
	baseLat := 55.7558
	baseLon := 37.6173

	for i := 0; i < count; i++ {
		location := CreateTestLocation(driverID)
		location.RecordedAt = baseTime.Add(time.Duration(i) * interval)
		location.CreatedAt = location.RecordedAt

		// Небольшие случайные отклонения координат для имитации движения
		location.Latitude = baseLat + float64(i)*0.001
		location.Longitude = baseLon + float64(i)*0.001
		location.Speed = floatPtr(float64(30 + i*2)) // Увеличиваем скорость

		locations[i] = location
	}

	return locations
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}

// TestDataSet набор тестовых данных
type TestDataSet struct {
	Drivers   []*entities.Driver
	Locations []*entities.DriverLocation
	Documents []*entities.DriverDocument
	Shifts    []*entities.DriverShift
	Ratings   []*entities.DriverRating
}

// CreateFullTestDataSet создает полный набор связанных тестовых данных
func CreateFullTestDataSet() *TestDataSet {
	// Создаем 3 водителей
	drivers := CreateMultipleTestDrivers(3)

	// Устанавливаем разные статусы
	drivers[0].Status = entities.StatusAvailable
	drivers[1].Status = entities.StatusOnShift
	drivers[2].Status = entities.StatusBusy

	var locations []*entities.DriverLocation
	var documents []*entities.DriverDocument
	var shifts []*entities.DriverShift
	var ratings []*entities.DriverRating

	// Создаем данные для каждого водителя
	for i, driver := range drivers {
		// Местоположения
		driverLocations := CreateTestLocationHistory(driver.ID, 5, 5*time.Minute)
		locations = append(locations, driverLocations...)

		// Документы
		licenseDoc := CreateTestDocument(driver.ID, entities.DocumentTypeDriverLicense)
		if i == 0 {
			licenseDoc.Status = entities.VerificationStatusVerified
		}
		documents = append(documents, licenseDoc)

		medicalDoc := CreateTestDocument(driver.ID, entities.DocumentTypeMedicalCert)
		documents = append(documents, medicalDoc)

		// Смены (только для активных водителей)
		if driver.Status != entities.StatusRegistered {
			shift := CreateTestShift(driver.ID)
			if i == 2 { // Завершенная смена для третьего водителя
				endTime := time.Now()
				shift.EndTime = &endTime
				shift.Status = entities.ShiftStatusCompleted
				shift.TotalTrips = 5
				shift.TotalDistance = 25.5
				shift.TotalEarnings = 1250.0
			}
			shifts = append(shifts, shift)
		}

		// Рейтинги
		for j := 0; j < 3; j++ {
			rating := CreateTestRating(driver.ID, 4+j%2) // Рейтинги 4 и 5
			ratings = append(ratings, rating)
		}
	}

	return &TestDataSet{
		Drivers:   drivers,
		Locations: locations,
		Documents: documents,
		Shifts:    shifts,
		Ratings:   ratings,
	}
}
