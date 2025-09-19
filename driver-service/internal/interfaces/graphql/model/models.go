package model

import (
	"time"

	"github.com/google/uuid"
)

// Скалярные типы
type UUID = uuid.UUID
type Time = time.Time
type Metadata = map[string]interface{}
type CriteriaScores = map[string]int
type CriteriaAverages = map[string]float64

// Enums
type Status string

const (
	StatusRegistered          Status = "REGISTERED"
	StatusPendingVerification Status = "PENDING_VERIFICATION"
	StatusVerified            Status = "VERIFIED"
	StatusRejected            Status = "REJECTED"
	StatusAvailable           Status = "AVAILABLE"
	StatusOnShift             Status = "ON_SHIFT"
	StatusBusy                Status = "BUSY"
	StatusInactive            Status = "INACTIVE"
	StatusSuspended           Status = "SUSPENDED"
	StatusBlocked             Status = "BLOCKED"
)

type ShiftStatus string

const (
	ShiftStatusActive    ShiftStatus = "ACTIVE"
	ShiftStatusCompleted ShiftStatus = "COMPLETED"
	ShiftStatusSuspended ShiftStatus = "SUSPENDED"
	ShiftStatusCancelled ShiftStatus = "CANCELLED"
)

type DocumentType string

const (
	DocumentTypeDriverLicense     DocumentType = "DRIVER_LICENSE"
	DocumentTypeMedicalCert      DocumentType = "MEDICAL_CERTIFICATE"
	DocumentTypeVehicleReg       DocumentType = "VEHICLE_REGISTRATION"
	DocumentTypeInsurance        DocumentType = "INSURANCE"
	DocumentTypePassport         DocumentType = "PASSPORT"
	DocumentTypeTaxiPermit       DocumentType = "TAXI_PERMIT"
	DocumentTypeWorkPermit       DocumentType = "WORK_PERMIT"
)

type VerificationStatus string

const (
	VerificationStatusPending    VerificationStatus = "PENDING"
	VerificationStatusVerified   VerificationStatus = "VERIFIED"
	VerificationStatusRejected   VerificationStatus = "REJECTED"
	VerificationStatusExpired    VerificationStatus = "EXPIRED"
	VerificationStatusProcessing VerificationStatus = "PROCESSING"
)

type RatingType string

const (
	RatingTypeCustomer  RatingType = "CUSTOMER"
	RatingTypeSystem    RatingType = "SYSTEM"
	RatingTypeAdmin     RatingType = "ADMIN"
	RatingTypePeer      RatingType = "PEER"
	RatingTypeAutomatic RatingType = "AUTOMATIC"
)

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "ASC"
	SortDirectionDesc SortDirection = "DESC"
)

// Основные типы
type Driver struct {
	ID              UUID      `json:"id"`
	Phone           string    `json:"phone"`
	Email           string    `json:"email"`
	FirstName       string    `json:"firstName"`
	LastName        string    `json:"lastName"`
	MiddleName      *string   `json:"middleName"`
	BirthDate       Time      `json:"birthDate"`
	PassportSeries  string    `json:"passportSeries"`
	PassportNumber  string    `json:"passportNumber"`
	LicenseNumber   string    `json:"licenseNumber"`
	LicenseExpiry   Time      `json:"licenseExpiry"`
	Status          Status    `json:"status"`
	CurrentRating   float64   `json:"currentRating"`
	TotalTrips      int       `json:"totalTrips"`
	Metadata        *Metadata `json:"metadata"`
	CreatedAt       Time      `json:"createdAt"`
	UpdatedAt       Time      `json:"updatedAt"`
}

type DriverLocation struct {
	ID         UUID     `json:"id"`
	DriverID   UUID     `json:"driverId"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Altitude   *float64 `json:"altitude"`
	Accuracy   *float64 `json:"accuracy"`
	Speed      *float64 `json:"speed"`
	Bearing    *float64 `json:"bearing"`
	Address    *string  `json:"address"`
	RecordedAt Time     `json:"recordedAt"`
	CreatedAt  Time     `json:"createdAt"`
}

type DriverRating struct {
	ID             UUID             `json:"id"`
	DriverID       UUID             `json:"driverId"`
	OrderID        *UUID            `json:"orderId"`
	CustomerID     *UUID            `json:"customerId"`
	Rating         int              `json:"rating"`
	Comment        *string          `json:"comment"`
	RatingType     RatingType       `json:"ratingType"`
	CriteriaScores *CriteriaScores  `json:"criteriaScores"`
	IsVerified     bool             `json:"isVerified"`
	IsAnonymous    bool             `json:"isAnonymous"`
	CreatedAt      Time             `json:"createdAt"`
	UpdatedAt      Time             `json:"updatedAt"`
}

type DriverShift struct {
	ID              UUID         `json:"id"`
	DriverID        UUID         `json:"driverId"`
	VehicleID       *UUID        `json:"vehicleId"`
	Status          ShiftStatus  `json:"status"`
	StartTime       Time         `json:"startTime"`
	EndTime         *Time        `json:"endTime"`
	StartLocation   *Location    `json:"startLocation"`
	EndLocation     *Location    `json:"endLocation"`
	TotalTrips      int          `json:"totalTrips"`
	TotalDistance   float64      `json:"totalDistance"`
	TotalEarnings   float64      `json:"totalEarnings"`
	FuelConsumed    *float64     `json:"fuelConsumed"`
	CreatedAt       Time         `json:"createdAt"`
	UpdatedAt       Time         `json:"updatedAt"`
}

type DriverDocument struct {
	ID               UUID               `json:"id"`
	DriverID         UUID               `json:"driverId"`
	DocumentType     DocumentType       `json:"documentType"`
	DocumentNumber   string             `json:"documentNumber"`
	IssueDate        Time               `json:"issueDate"`
	ExpiryDate       Time               `json:"expiryDate"`
	FileURL          string             `json:"fileUrl"`
	Status           VerificationStatus `json:"status"`
	VerifiedBy       *string            `json:"verifiedBy"`
	VerifiedAt       *Time              `json:"verifiedAt"`
	RejectionReason  *string            `json:"rejectionReason"`
	CreatedAt        Time               `json:"createdAt"`
	UpdatedAt        Time               `json:"updatedAt"`
}

// Статистика
type RatingStats struct {
	DriverID           UUID                `json:"driverId"`
	AverageRating      float64             `json:"averageRating"`
	TotalRatings       int                 `json:"totalRatings"`
	RatingDistribution *RatingDistribution `json:"ratingDistribution"`
	CriteriaAverages   *CriteriaAverages   `json:"criteriaAverages"`
	LastRatingDate     *Time               `json:"lastRatingDate"`
	Percentile95       float64             `json:"percentile95"`
	Percentile90       float64             `json:"percentile90"`
	LastUpdated        Time                `json:"lastUpdated"`
}

type RatingDistribution struct {
	One   int `json:"one"`
	Two   int `json:"two"`
	Three int `json:"three"`
	Four  int `json:"four"`
	Five  int `json:"five"`
}

type LocationStats struct {
	TotalPoints       int     `json:"totalPoints"`
	DistanceTraveled  float64 `json:"distanceTraveled"`
	AverageSpeed      float64 `json:"averageSpeed"`
	MaxSpeed          float64 `json:"maxSpeed"`
	TimeSpan          int     `json:"timeSpan"`
}

type ShiftStats struct {
	TotalShifts      int     `json:"totalShifts"`
	ActiveShifts     int     `json:"activeShifts"`
	CompletedShifts  int     `json:"completedShifts"`
	TotalHours       float64 `json:"totalHours"`
	TotalEarnings    float64 `json:"totalEarnings"`
	TotalTrips       int     `json:"totalTrips"`
	TotalDistance    float64 `json:"totalDistance"`
	AvgShiftDuration float64 `json:"avgShiftDuration"`
	AvgShiftEarnings float64 `json:"avgShiftEarnings"`
	AvgHourlyRate    float64 `json:"avgHourlyRate"`
}

// Вспомогательные типы
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   *string `json:"address"`
}

type GeoBounds struct {
	NorthEast Location `json:"northEast"`
	SouthWest Location `json:"southWest"`
}

type PageInfo struct {
	HasMore bool `json:"hasMore"`
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
}

// Соединения с пагинацией
type DriversConnection struct {
	Drivers  []*Driver `json:"drivers"`
	PageInfo *PageInfo `json:"pageInfo"`
}

type LocationsConnection struct {
	Locations []*DriverLocation `json:"locations"`
	PageInfo  *PageInfo         `json:"pageInfo"`
}

type RatingsConnection struct {
	Ratings  []*DriverRating `json:"ratings"`
	PageInfo *PageInfo       `json:"pageInfo"`
}

type ShiftsConnection struct {
	Shifts   []*DriverShift `json:"shifts"`
	PageInfo *PageInfo      `json:"pageInfo"`
}

// Входные типы для фильтров
type DriverFilters struct {
	Status        []*Status   `json:"status"`
	MinRating     *float64    `json:"minRating"`
	MaxRating     *float64    `json:"maxRating"`
	City          *string     `json:"city"`
	CreatedAfter  *Time       `json:"createdAfter"`
	CreatedBefore *Time       `json:"createdBefore"`
	SortBy        *string     `json:"sortBy"`
	SortDirection *SortDirection `json:"sortDirection"`
}

type LocationFilters struct {
	DriverID  *UUID       `json:"driverId"`
	From      *Time       `json:"from"`
	To        *Time       `json:"to"`
	Bounds    *GeoBoundsInput `json:"bounds"`
	MinSpeed  *float64    `json:"minSpeed"`
	MaxSpeed  *float64    `json:"maxSpeed"`
}

type RatingFilters struct {
	DriverID      *UUID         `json:"driverId"`
	CustomerID    *UUID         `json:"customerId"`
	OrderID       *UUID         `json:"orderId"`
	RatingType    []*RatingType `json:"ratingType"`
	MinRating     *int          `json:"minRating"`
	MaxRating     *int          `json:"maxRating"`
	IsVerified    *bool         `json:"isVerified"`
	From          *Time         `json:"from"`
	To            *Time         `json:"to"`
	SortBy        *string       `json:"sortBy"`
	SortDirection *SortDirection `json:"sortDirection"`
}

type ShiftFilters struct {
	DriverID      *UUID          `json:"driverId"`
	VehicleID     *UUID          `json:"vehicleId"`
	Status        []*ShiftStatus `json:"status"`
	From          *Time          `json:"from"`
	To            *Time          `json:"to"`
	MinEarnings   *float64       `json:"minEarnings"`
	MaxEarnings   *float64       `json:"maxEarnings"`
	MinTrips      *int           `json:"minTrips"`
	MaxTrips      *int           `json:"maxTrips"`
	SortBy        *string        `json:"sortBy"`
	SortDirection *SortDirection `json:"sortDirection"`
}

type DocumentFilters struct {
	DriverID        *UUID                  `json:"driverId"`
	DocumentType    []*DocumentType        `json:"documentType"`
	Status          []*VerificationStatus  `json:"status"`
	ExpiringInDays  *int                   `json:"expiringInDays"`
	Expired         *bool                  `json:"expired"`
}

// Входные типы для мутаций
type CreateDriverInput struct {
	Phone          string  `json:"phone"`
	Email          string  `json:"email"`
	FirstName      string  `json:"firstName"`
	LastName       string  `json:"lastName"`
	MiddleName     *string `json:"middleName"`
	BirthDate      Time    `json:"birthDate"`
	PassportSeries string  `json:"passportSeries"`
	PassportNumber string  `json:"passportNumber"`
	LicenseNumber  string  `json:"licenseNumber"`
	LicenseExpiry  Time    `json:"licenseExpiry"`
}

type UpdateDriverInput struct {
	Email          *string `json:"email"`
	FirstName      *string `json:"firstName"`
	LastName       *string `json:"lastName"`
	MiddleName     *string `json:"middleName"`
	BirthDate      *Time   `json:"birthDate"`
	PassportSeries *string `json:"passportSeries"`
	PassportNumber *string `json:"passportNumber"`
	LicenseExpiry  *Time   `json:"licenseExpiry"`
}

type LocationUpdateInput struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Altitude  *float64 `json:"altitude"`
	Accuracy  *float64 `json:"accuracy"`
	Speed     *float64 `json:"speed"`
	Bearing   *float64 `json:"bearing"`
	Timestamp *Time    `json:"timestamp"`
}

type RatingInput struct {
	Rating         int                    `json:"rating"`
	Comment        *string                `json:"comment"`
	CriteriaScores *CriteriaScoresInput   `json:"criteriaScores"`
	IsAnonymous    *bool                  `json:"isAnonymous"`
}

type CriteriaScoresInput struct {
	Cleanliness  *int `json:"cleanliness"`
	Driving      *int `json:"driving"`
	Punctuality  *int `json:"punctuality"`
	Politeness   *int `json:"politeness"`
	Navigation   *int `json:"navigation"`
}

type GeoBoundsInput struct {
	NorthEast LocationInput `json:"northEast"`
	SouthWest LocationInput `json:"southWest"`
}

type LocationInput struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   *string `json:"address"`
}

type ShiftStartInput struct {
	VehicleID *UUID    `json:"vehicleId"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Notes     *string  `json:"notes"`
}

type ShiftEndInput struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Notes     *string  `json:"notes"`
}

type DocumentUploadInput struct {
	DocumentType   DocumentType `json:"documentType"`
	DocumentNumber string       `json:"documentNumber"`
	IssueDate      Time         `json:"issueDate"`
	ExpiryDate     Time         `json:"expiryDate"`
}

type DocumentVerificationInput struct {
	Status          VerificationStatus `json:"status"`
	RejectionReason *string            `json:"rejectionReason"`
	Notes           *string            `json:"notes"`
}