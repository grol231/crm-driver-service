package entities

import "errors"

// Domain errors for Driver entity
var (
	// Driver errors
	ErrDriverNotFound    = errors.New("driver not found")
	ErrDriverExists      = errors.New("driver already exists")
	ErrInvalidPhone      = errors.New("invalid phone number")
	ErrInvalidEmail      = errors.New("invalid email address")
	ErrInvalidName       = errors.New("invalid name")
	ErrInvalidLicense    = errors.New("invalid license number")
	ErrInvalidPassport   = errors.New("invalid passport data")
	ErrInvalidStatus     = errors.New("invalid driver status")
	ErrInvalidDriverID   = errors.New("invalid driver ID")

	// Document errors
	ErrDocumentNotFound      = errors.New("document not found")
	ErrDocumentExists        = errors.New("document already exists")
	ErrInvalidDocumentType   = errors.New("invalid document type")
	ErrInvalidDocumentNumber = errors.New("invalid document number")
	ErrInvalidFileURL        = errors.New("invalid file URL")
	ErrInvalidExpiryDate     = errors.New("invalid expiry date")
	ErrDocumentExpired       = errors.New("document expired")
	ErrDocumentNotVerified   = errors.New("document not verified")

	// Location errors
	ErrLocationNotFound  = errors.New("location not found")
	ErrInvalidLocation   = errors.New("invalid location coordinates")
	ErrInvalidTimestamp  = errors.New("invalid timestamp")
	ErrLocationTooOld    = errors.New("location data is too old")

	// Shift errors
	ErrShiftNotFound     = errors.New("shift not found")
	ErrShiftExists       = errors.New("active shift already exists")
	ErrInvalidStartTime  = errors.New("invalid start time")
	ErrInvalidEndTime    = errors.New("invalid end time")
	ErrShiftNotActive    = errors.New("shift is not active")
	ErrShiftAlreadyEnded = errors.New("shift already ended")

	// Rating errors
	ErrRatingNotFound       = errors.New("rating not found")
	ErrInvalidRating        = errors.New("invalid rating value")
	ErrInvalidCriteriaScore = errors.New("invalid criteria score")
	ErrRatingExists         = errors.New("rating already exists")

	// Business logic errors
	ErrDriverNotAvailable     = errors.New("driver is not available")
	ErrDriverBlocked          = errors.New("driver is blocked")
	ErrDriverSuspended        = errors.New("driver is suspended")
	ErrLicenseExpired         = errors.New("driver license expired")
	ErrUnauthorized           = errors.New("unauthorized access")
	ErrPermissionDenied       = errors.New("permission denied")
	ErrInvalidOperation       = errors.New("invalid operation")
	ErrConcurrentModification = errors.New("concurrent modification detected")
)