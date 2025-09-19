package model

import (
	"driver-service/internal/domain/entities"
)

// Конверторы из domain entities в GraphQL модели

// Driver converters
func DriverFromEntity(driver *entities.Driver) *Driver {
	if driver == nil {
		return nil
	}

	var metadata *Metadata
	if driver.Metadata != nil {
		m := Metadata(driver.Metadata)
		metadata = &m
	}

	return &Driver{
		ID:              driver.ID,
		Phone:           driver.Phone,
		Email:           driver.Email,
		FirstName:       driver.FirstName,
		LastName:        driver.LastName,
		MiddleName:      driver.MiddleName,
		BirthDate:       driver.BirthDate,
		PassportSeries:  driver.PassportSeries,
		PassportNumber:  driver.PassportNumber,
		LicenseNumber:   driver.LicenseNumber,
		LicenseExpiry:   driver.LicenseExpiry,
		Status:          StatusFromEntity(driver.Status),
		CurrentRating:   driver.CurrentRating,
		TotalTrips:      driver.TotalTrips,
		Metadata:        metadata,
		CreatedAt:       driver.CreatedAt,
		UpdatedAt:       driver.UpdatedAt,
	}
}

func DriversFromEntity(drivers []*entities.Driver) []*Driver {
	result := make([]*Driver, len(drivers))
	for i, driver := range drivers {
		result[i] = DriverFromEntity(driver)
	}
	return result
}

func DriverToEntity(driver *Driver) *entities.Driver {
	if driver == nil {
		return nil
	}

	var metadata entities.Metadata
	if driver.Metadata != nil {
		metadata = entities.Metadata(*driver.Metadata)
	}

	return &entities.Driver{
		ID:              driver.ID,
		Phone:           driver.Phone,
		Email:           driver.Email,
		FirstName:       driver.FirstName,
		LastName:        driver.LastName,
		MiddleName:      driver.MiddleName,
		BirthDate:       driver.BirthDate,
		PassportSeries:  driver.PassportSeries,
		PassportNumber:  driver.PassportNumber,
		LicenseNumber:   driver.LicenseNumber,
		LicenseExpiry:   driver.LicenseExpiry,
		Status:          StatusToEntity(driver.Status),
		CurrentRating:   driver.CurrentRating,
		TotalTrips:      driver.TotalTrips,
		Metadata:        metadata,
		CreatedAt:       driver.CreatedAt,
		UpdatedAt:       driver.UpdatedAt,
	}
}

// DriverLocation converters
func DriverLocationFromEntity(location *entities.DriverLocation) *DriverLocation {
	if location == nil {
		return nil
	}

	return &DriverLocation{
		ID:         location.ID,
		DriverID:   location.DriverID,
		Latitude:   location.Latitude,
		Longitude:  location.Longitude,
		Altitude:   location.Altitude,
		Accuracy:   location.Accuracy,
		Speed:      location.Speed,
		Bearing:    location.Bearing,
		Address:    location.Address,
		RecordedAt: location.RecordedAt,
		CreatedAt:  location.CreatedAt,
	}
}

func DriverLocationsFromEntity(locations []*entities.DriverLocation) []*DriverLocation {
	result := make([]*DriverLocation, len(locations))
	for i, location := range locations {
		result[i] = DriverLocationFromEntity(location)
	}
	return result
}

func DriverLocationToEntity(location *DriverLocation) *entities.DriverLocation {
	if location == nil {
		return nil
	}

	return &entities.DriverLocation{
		ID:         location.ID,
		DriverID:   location.DriverID,
		Latitude:   location.Latitude,
		Longitude:  location.Longitude,
		Altitude:   location.Altitude,
		Accuracy:   location.Accuracy,
		Speed:      location.Speed,
		Bearing:    location.Bearing,
		Address:    location.Address,
		RecordedAt: location.RecordedAt,
		CreatedAt:  location.CreatedAt,
	}
}

// DriverRating converters
func DriverRatingFromEntity(rating *entities.DriverRating) *DriverRating {
	if rating == nil {
		return nil
	}

	var criteriaScores *CriteriaScores
	if rating.CriteriaScores != nil {
		cs := CriteriaScores(rating.CriteriaScores)
		criteriaScores = &cs
	}

	return &DriverRating{
		ID:             rating.ID,
		DriverID:       rating.DriverID,
		OrderID:        rating.OrderID,
		CustomerID:     rating.CustomerID,
		Rating:         rating.Rating,
		Comment:        rating.Comment,
		RatingType:     RatingTypeFromEntity(rating.RatingType),
		CriteriaScores: criteriaScores,
		IsVerified:     rating.IsVerified,
		IsAnonymous:    rating.IsAnonymous,
		CreatedAt:      rating.CreatedAt,
		UpdatedAt:      rating.UpdatedAt,
	}
}

func DriverRatingsFromEntity(ratings []*entities.DriverRating) []*DriverRating {
	result := make([]*DriverRating, len(ratings))
	for i, rating := range ratings {
		result[i] = DriverRatingFromEntity(rating)
	}
	return result
}

// DriverShift converters
func DriverShiftFromEntity(shift *entities.DriverShift) *DriverShift {
	if shift == nil {
		return nil
	}

	var startLocation, endLocation *Location
	if shift.StartLatitude != nil && shift.StartLongitude != nil {
		startLocation = &Location{
			Latitude:  *shift.StartLatitude,
			Longitude: *shift.StartLongitude,
		}
	}
	if shift.EndLatitude != nil && shift.EndLongitude != nil {
		endLocation = &Location{
			Latitude:  *shift.EndLatitude,
			Longitude: *shift.EndLongitude,
		}
	}

	return &DriverShift{
		ID:              shift.ID,
		DriverID:        shift.DriverID,
		VehicleID:       shift.VehicleID,
		Status:          ShiftStatusFromEntity(shift.Status),
		StartTime:       shift.StartTime,
		EndTime:         shift.EndTime,
		StartLocation:   startLocation,
		EndLocation:     endLocation,
		TotalTrips:      shift.TotalTrips,
		TotalDistance:   shift.TotalDistance,
		TotalEarnings:   shift.TotalEarnings,
		FuelConsumed:    shift.FuelConsumed,
		CreatedAt:       shift.CreatedAt,
		UpdatedAt:       shift.UpdatedAt,
	}
}

func DriverShiftsFromEntity(shifts []*entities.DriverShift) []*DriverShift {
	result := make([]*DriverShift, len(shifts))
	for i, shift := range shifts {
		result[i] = DriverShiftFromEntity(shift)
	}
	return result
}

// DriverDocument converters
func DriverDocumentFromEntity(document *entities.DriverDocument) *DriverDocument {
	if document == nil {
		return nil
	}

	return &DriverDocument{
		ID:               document.ID,
		DriverID:         document.DriverID,
		DocumentType:     DocumentTypeFromEntity(document.DocumentType),
		DocumentNumber:   document.DocumentNumber,
		IssueDate:        document.IssueDate,
		ExpiryDate:       document.ExpiryDate,
		FileURL:          document.FileURL,
		Status:           VerificationStatusFromEntity(document.Status),
		VerifiedBy:       document.VerifiedBy,
		VerifiedAt:       document.VerifiedAt,
		RejectionReason:  document.RejectionReason,
		CreatedAt:        document.CreatedAt,
		UpdatedAt:        document.UpdatedAt,
	}
}

func DriverDocumentsFromEntity(documents []*entities.DriverDocument) []*DriverDocument {
	result := make([]*DriverDocument, len(documents))
	for i, document := range documents {
		result[i] = DriverDocumentFromEntity(document)
	}
	return result
}

// RatingStats converters
func RatingStatsFromEntity(stats *entities.RatingStats) *RatingStats {
	if stats == nil {
		return nil
	}

	var distribution *RatingDistribution
	if stats.RatingDistribution != nil {
		distribution = &RatingDistribution{
			One:   stats.RatingDistribution[1],
			Two:   stats.RatingDistribution[2],
			Three: stats.RatingDistribution[3],
			Four:  stats.RatingDistribution[4],
			Five:  stats.RatingDistribution[5],
		}
	}

	var criteriaAverages *CriteriaAverages
	if stats.CriteriaAverages != nil {
		ca := CriteriaAverages(stats.CriteriaAverages)
		criteriaAverages = &ca
	}

	return &RatingStats{
		DriverID:           stats.DriverID,
		AverageRating:      stats.AverageRating,
		TotalRatings:       stats.TotalRatings,
		RatingDistribution: distribution,
		CriteriaAverages:   criteriaAverages,
		LastRatingDate:     stats.LastRatingDate,
		Percentile95:       stats.GetPercentile(95),
		Percentile90:       stats.GetPercentile(90),
		LastUpdated:        stats.LastUpdated,
	}
}

// LocationStats converters
func LocationStatsFromEntity(stats *entities.LocationStats) *LocationStats {
	if stats == nil {
		return nil
	}

	return &LocationStats{
		TotalPoints:       stats.TotalPoints,
		DistanceTraveled:  stats.DistanceTraveled,
		AverageSpeed:      stats.AverageSpeed,
		MaxSpeed:          stats.MaxSpeed,
		TimeSpan:          int(stats.TimeSpan),
	}
}

// ShiftStats converters
func ShiftStatsFromEntity(stats *entities.ShiftStats) *ShiftStats {
	if stats == nil {
		return nil
	}

	return &ShiftStats{
		TotalShifts:      stats.TotalShifts,
		ActiveShifts:     stats.ActiveShifts,
		CompletedShifts:  stats.CompletedShifts,
		TotalHours:       stats.TotalHours,
		TotalEarnings:    stats.TotalEarnings,
		TotalTrips:       stats.TotalTrips,
		TotalDistance:    stats.TotalDistance,
		AvgShiftDuration: stats.AvgShiftDuration,
		AvgShiftEarnings: stats.AvgShiftEarnings,
		AvgHourlyRate:    stats.AvgHourlyRate,
	}
}

// Enum converters
func StatusFromEntity(status entities.Status) Status {
	switch status {
	case entities.StatusRegistered:
		return StatusRegistered
	case entities.StatusPendingVerification:
		return StatusPendingVerification
	case entities.StatusVerified:
		return StatusVerified
	case entities.StatusRejected:
		return StatusRejected
	case entities.StatusAvailable:
		return StatusAvailable
	case entities.StatusOnShift:
		return StatusOnShift
	case entities.StatusBusy:
		return StatusBusy
	case entities.StatusInactive:
		return StatusInactive
	case entities.StatusSuspended:
		return StatusSuspended
	case entities.StatusBlocked:
		return StatusBlocked
	default:
		return StatusRegistered
	}
}

func StatusToEntity(status Status) entities.Status {
	switch status {
	case StatusRegistered:
		return entities.StatusRegistered
	case StatusPendingVerification:
		return entities.StatusPendingVerification
	case StatusVerified:
		return entities.StatusVerified
	case StatusRejected:
		return entities.StatusRejected
	case StatusAvailable:
		return entities.StatusAvailable
	case StatusOnShift:
		return entities.StatusOnShift
	case StatusBusy:
		return entities.StatusBusy
	case StatusInactive:
		return entities.StatusInactive
	case StatusSuspended:
		return entities.StatusSuspended
	case StatusBlocked:
		return entities.StatusBlocked
	default:
		return entities.StatusRegistered
	}
}

func ShiftStatusFromEntity(status entities.ShiftStatus) ShiftStatus {
	switch status {
	case entities.ShiftStatusActive:
		return ShiftStatusActive
	case entities.ShiftStatusCompleted:
		return ShiftStatusCompleted
	case entities.ShiftStatusSuspended:
		return ShiftStatusSuspended
	case entities.ShiftStatusCancelled:
		return ShiftStatusCancelled
	default:
		return ShiftStatusActive
	}
}

func ShiftStatusToEntity(status ShiftStatus) entities.ShiftStatus {
	switch status {
	case ShiftStatusActive:
		return entities.ShiftStatusActive
	case ShiftStatusCompleted:
		return entities.ShiftStatusCompleted
	case ShiftStatusSuspended:
		return entities.ShiftStatusSuspended
	case ShiftStatusCancelled:
		return entities.ShiftStatusCancelled
	default:
		return entities.ShiftStatusActive
	}
}

func DocumentTypeFromEntity(docType entities.DocumentType) DocumentType {
	switch docType {
	case entities.DocumentTypeDriverLicense:
		return DocumentTypeDriverLicense
	case entities.DocumentTypeMedicalCert:
		return DocumentTypeMedicalCert
	case entities.DocumentTypeVehicleReg:
		return DocumentTypeVehicleReg
	case entities.DocumentTypeInsurance:
		return DocumentTypeInsurance
	case entities.DocumentTypePassport:
		return DocumentTypePassport
	case entities.DocumentTypeTaxiPermit:
		return DocumentTypeTaxiPermit
	case entities.DocumentTypeWorkPermit:
		return DocumentTypeWorkPermit
	default:
		return DocumentTypeDriverLicense
	}
}

func DocumentTypeToEntity(docType DocumentType) entities.DocumentType {
	switch docType {
	case DocumentTypeDriverLicense:
		return entities.DocumentTypeDriverLicense
	case DocumentTypeMedicalCert:
		return entities.DocumentTypeMedicalCert
	case DocumentTypeVehicleReg:
		return entities.DocumentTypeVehicleReg
	case DocumentTypeInsurance:
		return entities.DocumentTypeInsurance
	case DocumentTypePassport:
		return entities.DocumentTypePassport
	case DocumentTypeTaxiPermit:
		return entities.DocumentTypeTaxiPermit
	case DocumentTypeWorkPermit:
		return entities.DocumentTypeWorkPermit
	default:
		return entities.DocumentTypeDriverLicense
	}
}

func VerificationStatusFromEntity(status entities.VerificationStatus) VerificationStatus {
	switch status {
	case entities.VerificationStatusPending:
		return VerificationStatusPending
	case entities.VerificationStatusVerified:
		return VerificationStatusVerified
	case entities.VerificationStatusRejected:
		return VerificationStatusRejected
	case entities.VerificationStatusExpired:
		return VerificationStatusExpired
	case entities.VerificationStatusProcessing:
		return VerificationStatusProcessing
	default:
		return VerificationStatusPending
	}
}

func VerificationStatusToEntity(status VerificationStatus) entities.VerificationStatus {
	switch status {
	case VerificationStatusPending:
		return entities.VerificationStatusPending
	case VerificationStatusVerified:
		return entities.VerificationStatusVerified
	case VerificationStatusRejected:
		return entities.VerificationStatusRejected
	case VerificationStatusExpired:
		return entities.VerificationStatusExpired
	case VerificationStatusProcessing:
		return entities.VerificationStatusProcessing
	default:
		return entities.VerificationStatusPending
	}
}

func RatingTypeFromEntity(ratingType entities.RatingType) RatingType {
	switch ratingType {
	case entities.RatingTypeCustomer:
		return RatingTypeCustomer
	case entities.RatingTypeSystem:
		return RatingTypeSystem
	case entities.RatingTypeAdmin:
		return RatingTypeAdmin
	case entities.RatingTypePeer:
		return RatingTypePeer
	case entities.RatingTypeAutomatic:
		return RatingTypeAutomatic
	default:
		return RatingTypeCustomer
	}
}

func RatingTypeToEntity(ratingType RatingType) entities.RatingType {
	switch ratingType {
	case RatingTypeCustomer:
		return entities.RatingTypeCustomer
	case RatingTypeSystem:
		return entities.RatingTypeSystem
	case RatingTypeAdmin:
		return entities.RatingTypeAdmin
	case RatingTypePeer:
		return entities.RatingTypePeer
	case RatingTypeAutomatic:
		return entities.RatingTypeAutomatic
	default:
		return entities.RatingTypeCustomer
	}
}

// Filter converters
func DriverFiltersToEntity(filters *DriverFilters) *entities.DriverFilters {
	if filters == nil {
		return &entities.DriverFilters{}
	}

	entityFilters := &entities.DriverFilters{
		MinRating:     filters.MinRating,
		MaxRating:     filters.MaxRating,
		City:          filters.City,
		CreatedAfter:  filters.CreatedAfter,
		CreatedBefore: filters.CreatedBefore,
	}

	if filters.SortBy != nil {
		entityFilters.SortBy = *filters.SortBy
	}
	if filters.SortDirection != nil {
		if *filters.SortDirection == SortDirectionDesc {
			entityFilters.SortDirection = "desc"
		} else {
			entityFilters.SortDirection = "asc"
		}
	}

	if len(filters.Status) > 0 {
		entityFilters.Status = make([]entities.Status, len(filters.Status))
		for i, status := range filters.Status {
			if status != nil {
				entityFilters.Status[i] = StatusToEntity(*status)
			}
		}
	}

	return entityFilters
}

func LocationFiltersToEntity(filters *LocationFilters) *entities.LocationFilters {
	if filters == nil {
		return &entities.LocationFilters{}
	}

	return &entities.LocationFilters{
		DriverID:  filters.DriverID,
		From:      filters.From,
		To:        filters.To,
		MinSpeed:  filters.MinSpeed,
		MaxSpeed:  filters.MaxSpeed,
	}
}

func RatingFiltersToEntity(filters *RatingFilters) *entities.RatingFilters {
	if filters == nil {
		return &entities.RatingFilters{}
	}

	entityFilters := &entities.RatingFilters{
		DriverID:   filters.DriverID,
		CustomerID: filters.CustomerID,
		OrderID:    filters.OrderID,
		MinRating:  filters.MinRating,
		MaxRating:  filters.MaxRating,
		IsVerified: filters.IsVerified,
		From:       filters.From,
		To:         filters.To,
	}

	if filters.SortBy != nil {
		entityFilters.SortBy = *filters.SortBy
	}
	if filters.SortDirection != nil {
		if *filters.SortDirection == SortDirectionDesc {
			entityFilters.SortDirection = "desc"
		} else {
			entityFilters.SortDirection = "asc"
		}
	}

	if len(filters.RatingType) > 0 {
		entityFilters.RatingType = make([]entities.RatingType, len(filters.RatingType))
		for i, ratingType := range filters.RatingType {
			if ratingType != nil {
				entityFilters.RatingType[i] = RatingTypeToEntity(*ratingType)
			}
		}
	}

	return entityFilters
}

func ShiftFiltersToEntity(filters *ShiftFilters) *entities.ShiftFilters {
	if filters == nil {
		return &entities.ShiftFilters{}
	}

	entityFilters := &entities.ShiftFilters{
		DriverID:    filters.DriverID,
		VehicleID:   filters.VehicleID,
		From:        filters.From,
		To:          filters.To,
		MinEarnings: filters.MinEarnings,
		MaxEarnings: filters.MaxEarnings,
		MinTrips:    filters.MinTrips,
		MaxTrips:    filters.MaxTrips,
	}

	if filters.SortBy != nil {
		entityFilters.SortBy = *filters.SortBy
	}
	if filters.SortDirection != nil {
		if *filters.SortDirection == SortDirectionDesc {
			entityFilters.SortDirection = "desc"
		} else {
			entityFilters.SortDirection = "asc"
		}
	}

	if len(filters.Status) > 0 {
		entityFilters.Status = make([]entities.ShiftStatus, len(filters.Status))
		for i, status := range filters.Status {
			if status != nil {
				entityFilters.Status[i] = ShiftStatusToEntity(*status)
			}
		}
	}

	return entityFilters
}

func DocumentFiltersToEntity(filters *DocumentFilters) *entities.DocumentFilters {
	if filters == nil {
		return &entities.DocumentFilters{}
	}

	entityFilters := &entities.DocumentFilters{
		DriverID:       filters.DriverID,
		ExpiringIn:     filters.ExpiringInDays,
		Expired:        filters.Expired,
	}

	if len(filters.DocumentType) > 0 {
		entityFilters.DocumentType = make([]entities.DocumentType, len(filters.DocumentType))
		for i, docType := range filters.DocumentType {
			if docType != nil {
				entityFilters.DocumentType[i] = DocumentTypeToEntity(*docType)
			}
		}
	}

	if len(filters.Status) > 0 {
		entityFilters.Status = make([]entities.VerificationStatus, len(filters.Status))
		for i, status := range filters.Status {
			if status != nil {
				entityFilters.Status[i] = VerificationStatusToEntity(*status)
			}
		}
	}

	return entityFilters
}