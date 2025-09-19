package resolver

import (
	"context"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/interfaces/graphql/model"

	"go.uber.org/zap"
)

// Driver field resolvers
type driverResolver struct{ *Resolver }

// FullName возвращает полное имя водителя
func (r *driverResolver) FullName(ctx context.Context, obj *model.Driver) (string, error) {
	name := obj.LastName + " " + obj.FirstName
	if obj.MiddleName != nil && *obj.MiddleName != "" {
		name += " " + *obj.MiddleName
	}
	return name, nil
}

// Documents возвращает документы водителя
func (r *driverResolver) Documents(ctx context.Context, obj *model.Driver) ([]*model.DriverDocument, error) {
	filters := &entities.DocumentFilters{
		DriverID: &obj.ID,
	}

	documents, err := r.documentRepo.List(ctx, filters)
	if err != nil {
		r.logger.Error("Failed to get driver documents", zap.Error(err))
		return nil, err
	}

	return model.DriverDocumentsFromEntity(documents), nil
}

// CurrentLocation возвращает текущее местоположение водителя
func (r *driverResolver) CurrentLocation(ctx context.Context, obj *model.Driver) (*model.DriverLocation, error) {
	location, err := r.locationRepo.GetLatestByDriverID(ctx, obj.ID)
	if err != nil {
		if err == entities.ErrLocationNotFound {
			return nil, nil // Возвращаем nil, а не ошибку, если местоположение не найдено
		}
		r.logger.Error("Failed to get current location", zap.Error(err))
		return nil, err
	}

	return model.DriverLocationFromEntity(location), nil
}

// LocationHistory возвращает историю местоположений водителя
func (r *driverResolver) LocationHistory(ctx context.Context, obj *model.Driver, limit *int, offset *int) ([]*model.DriverLocation, error) {
	// Устанавливаем значения по умолчанию
	if limit == nil {
		defaultLimit := 50
		limit = &defaultLimit
	}
	if offset == nil {
		defaultOffset := 0
		offset = &defaultOffset
	}

	filters := &entities.LocationFilters{
		DriverID: &obj.ID,
		Limit:    *limit,
		Offset:   *offset,
	}

	locations, err := r.locationRepo.List(ctx, filters)
	if err != nil {
		r.logger.Error("Failed to get location history", zap.Error(err))
		return nil, err
	}

	return model.DriverLocationsFromEntity(locations), nil
}

// Ratings возвращает рейтинги водителя
func (r *driverResolver) Ratings(ctx context.Context, obj *model.Driver, limit *int, offset *int) ([]*model.DriverRating, error) {
	// Устанавливаем значения по умолчанию
	if limit == nil {
		defaultLimit := 20
		limit = &defaultLimit
	}
	if offset == nil {
		defaultOffset := 0
		offset = &defaultOffset
	}

	filters := &entities.RatingFilters{
		DriverID: &obj.ID,
		Limit:    *limit,
		Offset:   *offset,
	}

	ratings, err := r.ratingRepo.List(ctx, filters)
	if err != nil {
		r.logger.Error("Failed to get driver ratings", zap.Error(err))
		return nil, err
	}

	return model.DriverRatingsFromEntity(ratings), nil
}

// RatingStats возвращает статистику рейтингов водителя
func (r *driverResolver) RatingStats(ctx context.Context, obj *model.Driver) (*model.RatingStats, error) {
	stats, err := r.ratingRepo.GetDriverStats(ctx, obj.ID)
	if err != nil {
		r.logger.Error("Failed to get rating stats", zap.Error(err))
		return nil, err
	}

	return model.RatingStatsFromEntity(stats), nil
}

// ActiveShift возвращает активную смену водителя
func (r *driverResolver) ActiveShift(ctx context.Context, obj *model.Driver) (*model.DriverShift, error) {
	shift, err := r.shiftRepo.GetActiveByDriverID(ctx, obj.ID)
	if err != nil {
		if err == entities.ErrShiftNotFound {
			return nil, nil // Возвращаем nil, а не ошибку, если активной смены нет
		}
		r.logger.Error("Failed to get active shift", zap.Error(err))
		return nil, err
	}

	return model.DriverShiftFromEntity(shift), nil
}

// Shifts возвращает смены водителя
func (r *driverResolver) Shifts(ctx context.Context, obj *model.Driver, limit *int, offset *int) ([]*model.DriverShift, error) {
	// Устанавливаем значения по умолчанию
	if limit == nil {
		defaultLimit := 20
		limit = &defaultLimit
	}
	if offset == nil {
		defaultOffset := 0
		offset = &defaultOffset
	}

	filters := &entities.ShiftFilters{
		DriverID: &obj.ID,
		Limit:    *limit,
		Offset:   *offset,
	}

	shifts, err := r.shiftRepo.List(ctx, filters)
	if err != nil {
		r.logger.Error("Failed to get driver shifts", zap.Error(err))
		return nil, err
	}

	return model.DriverShiftsFromEntity(shifts), nil
}

// IsActive проверяет, активен ли водитель
func (r *driverResolver) IsActive(ctx context.Context, obj *model.Driver) (bool, error) {
	entityStatus := model.StatusToEntity(obj.Status)
	return entityStatus == entities.StatusAvailable || 
		   entityStatus == entities.StatusOnShift || 
		   entityStatus == entities.StatusBusy, nil
}

// CanReceiveOrders проверяет, может ли водитель получать заказы
func (r *driverResolver) CanReceiveOrders(ctx context.Context, obj *model.Driver) (bool, error) {
	entityStatus := model.StatusToEntity(obj.Status)
	return entityStatus == entities.StatusAvailable, nil
}

// IsLicenseExpired проверяет, истекла ли лицензия водителя
func (r *driverResolver) IsLicenseExpired(ctx context.Context, obj *model.Driver) (bool, error) {
	return time.Now().After(obj.LicenseExpiry), nil
}

// DriverLocation field resolvers
type driverLocationResolver struct{ *Resolver }

// Driver возвращает водителя для местоположения
func (r *driverLocationResolver) Driver(ctx context.Context, obj *model.DriverLocation) (*model.Driver, error) {
	driver, err := r.driverService.GetDriverByID(ctx, obj.DriverID)
	if err != nil {
		r.logger.Error("Failed to get driver for location", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(driver), nil
}

// IsHighAccuracy проверяет высокую точность местоположения
func (r *driverLocationResolver) IsHighAccuracy(ctx context.Context, obj *model.DriverLocation) (bool, error) {
	return obj.Accuracy != nil && *obj.Accuracy > 0 && *obj.Accuracy < 50, nil
}

// IsValidLocation проверяет валидность координат
func (r *driverLocationResolver) IsValidLocation(ctx context.Context, obj *model.DriverLocation) (bool, error) {
	return obj.Latitude >= -90 && obj.Latitude <= 90 &&
		   obj.Longitude >= -180 && obj.Longitude <= 180, nil
}

// DriverRating field resolvers
type driverRatingResolver struct{ *Resolver }

// Driver возвращает водителя для рейтинга
func (r *driverRatingResolver) Driver(ctx context.Context, obj *model.DriverRating) (*model.Driver, error) {
	driver, err := r.driverService.GetDriverByID(ctx, obj.DriverID)
	if err != nil {
		r.logger.Error("Failed to get driver for rating", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(driver), nil
}

// IsValid проверяет валидность рейтинга
func (r *driverRatingResolver) IsValid(ctx context.Context, obj *model.DriverRating) (bool, error) {
	return obj.Rating >= 1 && obj.Rating <= 5, nil
}

// OverallScore вычисляет общий балл рейтинга
func (r *driverRatingResolver) OverallScore(ctx context.Context, obj *model.DriverRating) (float64, error) {
	if obj.CriteriaScores == nil || len(*obj.CriteriaScores) == 0 {
		return float64(obj.Rating), nil
	}

	total := 0
	count := 0
	for _, score := range *obj.CriteriaScores {
		total += score
		count++
	}

	if count == 0 {
		return float64(obj.Rating), nil
	}

	return float64(total) / float64(count), nil
}

// DriverShift field resolvers
type driverShiftResolver struct{ *Resolver }

// Driver возвращает водителя для смены
func (r *driverShiftResolver) Driver(ctx context.Context, obj *model.DriverShift) (*model.Driver, error) {
	driver, err := r.driverService.GetDriverByID(ctx, obj.DriverID)
	if err != nil {
		r.logger.Error("Failed to get driver for shift", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(driver), nil
}

// Duration возвращает продолжительность смены в минутах
func (r *driverShiftResolver) Duration(ctx context.Context, obj *model.DriverShift) (int, error) {
	if obj.EndTime == nil {
		return int(time.Since(obj.StartTime).Minutes()), nil
	}
	return int(obj.EndTime.Sub(obj.StartTime).Minutes()), nil
}

// IsActive проверяет, активна ли смена
func (r *driverShiftResolver) IsActive(ctx context.Context, obj *model.DriverShift) (bool, error) {
	return obj.Status == model.ShiftStatusActive && obj.EndTime == nil, nil
}

// AverageEarningsPerTrip возвращает средний заработок за поездку
func (r *driverShiftResolver) AverageEarningsPerTrip(ctx context.Context, obj *model.DriverShift) (float64, error) {
	if obj.TotalTrips == 0 {
		return 0, nil
	}
	return obj.TotalEarnings / float64(obj.TotalTrips), nil
}

// AverageDistancePerTrip возвращает среднее расстояние за поездку
func (r *driverShiftResolver) AverageDistancePerTrip(ctx context.Context, obj *model.DriverShift) (float64, error) {
	if obj.TotalTrips == 0 {
		return 0, nil
	}
	return obj.TotalDistance / float64(obj.TotalTrips), nil
}

// EarningsPerHour возвращает заработок в час
func (r *driverShiftResolver) EarningsPerHour(ctx context.Context, obj *model.DriverShift) (float64, error) {
	duration, _ := r.Duration(ctx, obj)
	if duration == 0 {
		return 0, nil
	}
	hours := float64(duration) / 60.0
	return obj.TotalEarnings / hours, nil
}

// DriverDocument field resolvers
type driverDocumentResolver struct{ *Resolver }

// Driver возвращает водителя для документа
func (r *driverDocumentResolver) Driver(ctx context.Context, obj *model.DriverDocument) (*model.Driver, error) {
	driver, err := r.driverService.GetDriverByID(ctx, obj.DriverID)
	if err != nil {
		r.logger.Error("Failed to get driver for document", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(driver), nil
}

// IsExpired проверяет, истек ли документ
func (r *driverDocumentResolver) IsExpired(ctx context.Context, obj *model.DriverDocument) (bool, error) {
	return time.Now().After(obj.ExpiryDate), nil
}

// IsVerified проверяет, верифицирован ли документ
func (r *driverDocumentResolver) IsVerified(ctx context.Context, obj *model.DriverDocument) (bool, error) {
	isExpired, _ := r.IsExpired(ctx, obj)
	return obj.Status == model.VerificationStatusVerified && !isExpired, nil
}

// DaysUntilExpiry возвращает количество дней до истечения документа
func (r *driverDocumentResolver) DaysUntilExpiry(ctx context.Context, obj *model.DriverDocument) (int, error) {
	diff := obj.ExpiryDate.Sub(time.Now())
	return int(diff.Hours() / 24), nil
}