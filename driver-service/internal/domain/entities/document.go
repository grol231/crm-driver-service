package entities

import (
	"time"

	"github.com/google/uuid"
)

// DocumentType тип документа
type DocumentType string

const (
	DocumentTypeDriverLicense     DocumentType = "driver_license"
	DocumentTypeMedicalCert      DocumentType = "medical_certificate"
	DocumentTypeVehicleReg       DocumentType = "vehicle_registration"
	DocumentTypeInsurance        DocumentType = "insurance"
	DocumentTypePassport         DocumentType = "passport"
	DocumentTypeTaxiPermit       DocumentType = "taxi_permit"
	DocumentTypeWorkPermit       DocumentType = "work_permit"
)

// VerificationStatus статус верификации документа
type VerificationStatus string

const (
	VerificationStatusPending   VerificationStatus = "pending"
	VerificationStatusVerified  VerificationStatus = "verified"
	VerificationStatusRejected  VerificationStatus = "rejected"
	VerificationStatusExpired   VerificationStatus = "expired"
	VerificationStatusProcessing VerificationStatus = "processing"
)

// DriverDocument представляет документ водителя
type DriverDocument struct {
	ID             uuid.UUID          `json:"id" db:"id"`
	DriverID       uuid.UUID          `json:"driver_id" db:"driver_id"`
	DocumentType   DocumentType       `json:"document_type" db:"document_type"`
	DocumentNumber string             `json:"document_number" db:"document_number"`
	IssueDate      time.Time          `json:"issue_date" db:"issue_date"`
	ExpiryDate     time.Time          `json:"expiry_date" db:"expiry_date"`
	FileURL        string             `json:"file_url" db:"file_url"`
	Status         VerificationStatus `json:"status" db:"status"`
	VerifiedBy     *string            `json:"verified_by,omitempty" db:"verified_by"`
	VerifiedAt     *time.Time         `json:"verified_at,omitempty" db:"verified_at"`
	RejectionReason *string           `json:"rejection_reason,omitempty" db:"rejection_reason"`
	Metadata       Metadata           `json:"metadata" db:"metadata"`
	CreatedAt      time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" db:"updated_at"`
}

// IsExpired проверяет, не истек ли документ
func (d *DriverDocument) IsExpired() bool {
	return time.Now().After(d.ExpiryDate)
}

// IsVerified проверяет, верифицирован ли документ
func (d *DriverDocument) IsVerified() bool {
	return d.Status == VerificationStatusVerified && !d.IsExpired()
}

// DaysUntilExpiry возвращает количество дней до истечения документа
func (d *DriverDocument) DaysUntilExpiry() int {
	diff := d.ExpiryDate.Sub(time.Now())
	return int(diff.Hours() / 24)
}

// Verify верифицирует документ
func (d *DriverDocument) Verify(verifierID string) {
	now := time.Now()
	d.Status = VerificationStatusVerified
	d.VerifiedBy = &verifierID
	d.VerifiedAt = &now
	d.RejectionReason = nil
	d.UpdatedAt = now
}

// Reject отклоняет документ
func (d *DriverDocument) Reject(verifierID, reason string) {
	now := time.Now()
	d.Status = VerificationStatusRejected
	d.VerifiedBy = &verifierID
	d.VerifiedAt = &now
	d.RejectionReason = &reason
	d.UpdatedAt = now
}

// MarkExpired помечает документ как истекший
func (d *DriverDocument) MarkExpired() {
	d.Status = VerificationStatusExpired
	d.UpdatedAt = time.Now()
}

// Validate проверяет валидность данных документа
func (d *DriverDocument) Validate() error {
	if d.DriverID == uuid.Nil {
		return ErrInvalidDriverID
	}
	
	if d.DocumentType == "" {
		return ErrInvalidDocumentType
	}
	
	if d.DocumentNumber == "" {
		return ErrInvalidDocumentNumber
	}
	
	if d.FileURL == "" {
		return ErrInvalidFileURL
	}
	
	if d.ExpiryDate.Before(d.IssueDate) {
		return ErrInvalidExpiryDate
	}
	
	return nil
}

// NewDriverDocument создает новый документ водителя
func NewDriverDocument(driverID uuid.UUID, docType DocumentType, docNumber string, issueDate, expiryDate time.Time, fileURL string) *DriverDocument {
	now := time.Now()
	return &DriverDocument{
		ID:             uuid.New(),
		DriverID:       driverID,
		DocumentType:   docType,
		DocumentNumber: docNumber,
		IssueDate:      issueDate,
		ExpiryDate:     expiryDate,
		FileURL:        fileURL,
		Status:         VerificationStatusPending,
		Metadata:       make(Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// DocumentFilters фильтры для поиска документов
type DocumentFilters struct {
	DriverID     *uuid.UUID           `json:"driver_id,omitempty"`
	DocumentType []DocumentType       `json:"document_type,omitempty"`
	Status       []VerificationStatus `json:"status,omitempty"`
	ExpiringIn   *int                 `json:"expiring_in_days,omitempty"`
	Expired      *bool                `json:"expired,omitempty"`
	Limit        int                  `json:"limit,omitempty"`
	Offset       int                  `json:"offset,omitempty"`
}

// DocumentUploadRequest запрос на загрузку документа
type DocumentUploadRequest struct {
	DocumentType   DocumentType `json:"document_type" binding:"required"`
	DocumentNumber string       `json:"document_number" binding:"required"`
	IssueDate      time.Time    `json:"issue_date" binding:"required"`
	ExpiryDate     time.Time    `json:"expiry_date" binding:"required"`
}

// DocumentVerificationRequest запрос на верификацию документа
type DocumentVerificationRequest struct {
	Status          VerificationStatus `json:"status" binding:"required"`
	RejectionReason *string            `json:"rejection_reason,omitempty"`
	Notes           *string            `json:"notes,omitempty"`
}

// DocumentSummary краткая информация о документе
type DocumentSummary struct {
	ID             uuid.UUID          `json:"id"`
	DocumentType   DocumentType       `json:"document_type"`
	DocumentNumber string             `json:"document_number"`
	Status         VerificationStatus `json:"status"`
	ExpiryDate     time.Time          `json:"expiry_date"`
	IsExpired      bool               `json:"is_expired"`
	DaysToExpiry   int                `json:"days_to_expiry"`
}

// ToSummary возвращает краткую информацию о документе
func (d *DriverDocument) ToSummary() *DocumentSummary {
	return &DocumentSummary{
		ID:             d.ID,
		DocumentType:   d.DocumentType,
		DocumentNumber: d.DocumentNumber,
		Status:         d.Status,
		ExpiryDate:     d.ExpiryDate,
		IsExpired:      d.IsExpired(),
		DaysToExpiry:   d.DaysUntilExpiry(),
	}
}