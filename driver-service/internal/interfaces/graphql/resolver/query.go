package resolver

import (
	"context"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/interfaces/graphql/model"

	"go.uber.org/zap"
)

type queryResolver struct{ *Resolver }

// Driver получает водителя по ID
func (r *queryResolver) Driver(ctx context.Context, id model.UUID) (*model.Driver, error) {
	r.logger.Info("GraphQL: Getting driver", zap.String("driver_id", id.String()))

	driver, err := r.driverService.GetDriverByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get driver", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(driver), nil
}

// Drivers получает список водителей с фильтрами
func (r *queryResolver) Drivers(ctx context.Context, filters *model.DriverFilters, limit *int, offset *int) (*model.DriversConnection, error) {
	r.logger.Info("GraphQL: Getting drivers list")

	// Устанавливаем значения по умолчанию
	if limit == nil {
		defaultLimit := 20
		limit = &defaultLimit
	}
	if offset == nil {
		defaultOffset := 0
		offset = &defaultOffset
	}

	// Конвертируем фильтры
	entityFilters := model.DriverFiltersToEntity(filters)
	entityFilters.Limit = *limit
	entityFilters.Offset = *offset

	// Получаем водителей
	drivers, err := r.driverService.ListDrivers(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to get drivers", zap.Error(err))
		return nil, err
	}

	// Получаем общее количество
	total, err := r.driverService.CountDrivers(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to count drivers", zap.Error(err))
		total = len(drivers)
	}

	return &model.DriversConnection{
		Drivers: model.DriversFromEntity(drivers),
		PageInfo: &model.PageInfo{
			HasMore: *offset+len(drivers) < total,
			Total:   total,
			Limit:   *limit,
			Offset:  *offset,
		},
	}, nil
}

// ActiveDrivers получает список активных водителей
func (r *queryResolver) ActiveDrivers(ctx context.Context) ([]*model.Driver, error) {
	r.logger.Info("GraphQL: Getting active drivers")

	drivers, err := r.driverService.GetActiveDrivers(ctx)
	if err != nil {
		r.logger.Error("Failed to get active drivers", zap.Error(err))
		return nil, err
	}

	return model.DriversFromEntity(drivers), nil
}

// DriverByPhone получает водителя по номеру телефона
func (r *queryResolver) DriverByPhone(ctx context.Context, phone string) (*model.Driver, error) {
	r.logger.Info("GraphQL: Getting driver by phone", zap.String("phone", phone))

	driver, err := r.driverService.GetDriverByPhone(ctx, phone)
	if err != nil {
		r.logger.Error("Failed to get driver by phone", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(driver), nil
}

// DriverByEmail получает водителя по email
func (r *queryResolver) DriverByEmail(ctx context.Context, email string) (*model.Driver, error) {
	r.logger.Info("GraphQL: Getting driver by email", zap.String("email", email))

	driver, err := r.driverService.GetDriverByEmail(ctx, email)
	if err != nil {
		r.logger.Error("Failed to get driver by email", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(driver), nil
}

// DriverLocation получает местоположение по ID
func (r *queryResolver) DriverLocation(ctx context.Context, id model.UUID) (*model.DriverLocation, error) {
	r.logger.Info("GraphQL: Getting driver location", zap.String("location_id", id.String()))

	location, err := r.locationRepo.GetByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get location", zap.Error(err))
		return nil, err
	}

	return model.DriverLocationFromEntity(location), nil
}

// DriverLocations получает список местоположений с фильтрами
func (r *queryResolver) DriverLocations(ctx context.Context, filters model.LocationFilters, limit *int, offset *int) (*model.LocationsConnection, error) {
	r.logger.Info("GraphQL: Getting driver locations")

	// Устанавливаем значения по умолчанию
	if limit == nil {
		defaultLimit := 100
		limit = &defaultLimit
	}
	if offset == nil {
		defaultOffset := 0
		offset = &defaultOffset
	}

	// Конвертируем фильтры
	entityFilters := model.LocationFiltersToEntity(&filters)
	entityFilters.Limit = *limit
	entityFilters.Offset = *offset

	// Получаем местоположения
	locations, err := r.locationRepo.List(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to get locations", zap.Error(err))
		return nil, err
	}

	// Получаем общее количество
	total, err := r.locationRepo.Count(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to count locations", zap.Error(err))
		total = len(locations)
	}

	return &model.LocationsConnection{
		Locations: model.DriverLocationsFromEntity(locations),
		PageInfo: &model.PageInfo{
			HasMore: *offset+len(locations) < total,
			Total:   total,
			Limit:   *limit,
			Offset:  *offset,
		},
	}, nil
}

// NearbyDrivers получает ближайших водителей
func (r *queryResolver) NearbyDrivers(ctx context.Context, latitude, longitude float64, radiusKM *float64, limit *int) ([]*model.Driver, error) {
	r.logger.Info("GraphQL: Getting nearby drivers",
		zap.Float64("lat", latitude),
		zap.Float64("lon", longitude),
	)

	// Устанавливаем значения по умолчанию
	if radiusKM == nil {
		defaultRadius := 5.0
		radiusKM = &defaultRadius
	}
	if limit == nil {
		defaultLimit := 10
		limit = &defaultLimit
	}

	// Здесь предполагается, что у нас есть метод в locationService для поиска ближайших водителей
	drivers, err := r.locationService.GetNearbyDrivers(ctx, latitude, longitude, *radiusKM, *limit)
	if err != nil {
		r.logger.Error("Failed to get nearby drivers", zap.Error(err))
		return nil, err
	}

	return model.DriversFromEntity(drivers), nil
}

// DriverRating получает рейтинг по ID
func (r *queryResolver) DriverRating(ctx context.Context, id model.UUID) (*model.DriverRating, error) {
	r.logger.Info("GraphQL: Getting driver rating", zap.String("rating_id", id.String()))

	rating, err := r.ratingRepo.GetByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get rating", zap.Error(err))
		return nil, err
	}

	return model.DriverRatingFromEntity(rating), nil
}

// DriverRatings получает список рейтингов с фильтрами
func (r *queryResolver) DriverRatings(ctx context.Context, filters model.RatingFilters, limit *int, offset *int) (*model.RatingsConnection, error) {
	r.logger.Info("GraphQL: Getting driver ratings")

	// Устанавливаем значения по умолчанию
	if limit == nil {
		defaultLimit := 50
		limit = &defaultLimit
	}
	if offset == nil {
		defaultOffset := 0
		offset = &defaultOffset
	}

	// Конвертируем фильтры
	entityFilters := model.RatingFiltersToEntity(&filters)
	entityFilters.Limit = *limit
	entityFilters.Offset = *offset

	// Получаем рейтинги
	ratings, err := r.ratingRepo.List(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to get ratings", zap.Error(err))
		return nil, err
	}

	// Получаем общее количество
	total, err := r.ratingRepo.Count(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to count ratings", zap.Error(err))
		total = len(ratings)
	}

	return &model.RatingsConnection{
		Ratings: model.DriverRatingsFromEntity(ratings),
		PageInfo: &model.PageInfo{
			HasMore: *offset+len(ratings) < total,
			Total:   total,
			Limit:   *limit,
			Offset:  *offset,
		},
	}, nil
}

// DriverRatingStats получает статистику рейтингов водителя
func (r *queryResolver) DriverRatingStats(ctx context.Context, driverID model.UUID) (*model.RatingStats, error) {
	r.logger.Info("GraphQL: Getting driver rating stats", zap.String("driver_id", driverID.String()))

	stats, err := r.ratingRepo.GetDriverStats(ctx, driverID)
	if err != nil {
		r.logger.Error("Failed to get rating stats", zap.Error(err))
		return nil, err
	}

	return model.RatingStatsFromEntity(stats), nil
}

// DriverShift получает смену по ID
func (r *queryResolver) DriverShift(ctx context.Context, id model.UUID) (*model.DriverShift, error) {
	r.logger.Info("GraphQL: Getting driver shift", zap.String("shift_id", id.String()))

	shift, err := r.shiftRepo.GetByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get shift", zap.Error(err))
		return nil, err
	}

	return model.DriverShiftFromEntity(shift), nil
}

// DriverShifts получает список смен с фильтрами
func (r *queryResolver) DriverShifts(ctx context.Context, filters model.ShiftFilters, limit *int, offset *int) (*model.ShiftsConnection, error) {
	r.logger.Info("GraphQL: Getting driver shifts")

	// Устанавливаем значения по умолчанию
	if limit == nil {
		defaultLimit := 50
		limit = &defaultLimit
	}
	if offset == nil {
		defaultOffset := 0
		offset = &defaultOffset
	}

	// Конвертируем фильтры
	entityFilters := model.ShiftFiltersToEntity(&filters)
	entityFilters.Limit = *limit
	entityFilters.Offset = *offset

	// Получаем смены
	shifts, err := r.shiftRepo.List(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to get shifts", zap.Error(err))
		return nil, err
	}

	// Получаем общее количество
	total, err := r.shiftRepo.Count(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to count shifts", zap.Error(err))
		total = len(shifts)
	}

	return &model.ShiftsConnection{
		Shifts: model.DriverShiftsFromEntity(shifts),
		PageInfo: &model.PageInfo{
			HasMore: *offset+len(shifts) < total,
			Total:   total,
			Limit:   *limit,
			Offset:  *offset,
		},
	}, nil
}

// ActiveShifts получает список активных смен
func (r *queryResolver) ActiveShifts(ctx context.Context) ([]*model.DriverShift, error) {
	r.logger.Info("GraphQL: Getting active shifts")

	shifts, err := r.shiftRepo.GetActiveShifts(ctx)
	if err != nil {
		r.logger.Error("Failed to get active shifts", zap.Error(err))
		return nil, err
	}

	return model.DriverShiftsFromEntity(shifts), nil
}

// DriverDocument получает документ по ID
func (r *queryResolver) DriverDocument(ctx context.Context, id model.UUID) (*model.DriverDocument, error) {
	r.logger.Info("GraphQL: Getting driver document", zap.String("document_id", id.String()))

	document, err := r.documentRepo.GetByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get document", zap.Error(err))
		return nil, err
	}

	return model.DriverDocumentFromEntity(document), nil
}

// DriverDocuments получает список документов с фильтрами
func (r *queryResolver) DriverDocuments(ctx context.Context, filters model.DocumentFilters, limit *int, offset *int) ([]*model.DriverDocument, error) {
	r.logger.Info("GraphQL: Getting driver documents")

	// Устанавливаем значения по умолчанию
	if limit == nil {
		defaultLimit := 100
		limit = &defaultLimit
	}
	if offset == nil {
		defaultOffset := 0
		offset = &defaultOffset
	}

	// Конвертируем фильтры
	entityFilters := model.DocumentFiltersToEntity(&filters)
	entityFilters.Limit = *limit
	entityFilters.Offset = *offset

	// Получаем документы
	documents, err := r.documentRepo.List(ctx, entityFilters)
	if err != nil {
		r.logger.Error("Failed to get documents", zap.Error(err))
		return nil, err
	}

	return model.DriverDocumentsFromEntity(documents), nil
}

// LocationStats получает статистику местоположений
func (r *queryResolver) LocationStats(ctx context.Context, driverID model.UUID, from *model.Time, to *model.Time) (*model.LocationStats, error) {
	r.logger.Info("GraphQL: Getting location stats", zap.String("driver_id", driverID.String()))

	// Устанавливаем период по умолчанию (последние 7 дней)
	if from == nil {
		defaultFrom := time.Now().AddDate(0, 0, -7)
		from = &defaultFrom
	}
	if to == nil {
		now := time.Now()
		to = &now
	}

	// Получаем местоположения за период
	filters := &entities.LocationFilters{
		DriverID: &driverID,
		From:     from,
		To:       to,
	}

	locations, err := r.locationRepo.List(ctx, filters)
	if err != nil {
		r.logger.Error("Failed to get locations for stats", zap.Error(err))
		return nil, err
	}

	// Вычисляем статистику
	stats := entities.CalculateLocationStats(locations)
	return model.LocationStatsFromEntity(stats), nil
}

// ShiftStats получает статистику смен
func (r *queryResolver) ShiftStats(ctx context.Context, driverID *model.UUID, from *model.Time, to *model.Time) (*model.ShiftStats, error) {
	r.logger.Info("GraphQL: Getting shift stats")

	// Устанавливаем период по умолчанию (последние 30 дней)
	if from == nil {
		defaultFrom := time.Now().AddDate(0, 0, -30)
		from = &defaultFrom
	}
	if to == nil {
		now := time.Now()
		to = &now
	}

	// Получаем смены за период
	filters := &entities.ShiftFilters{
		From: from,
		To:   to,
	}
	if driverID != nil {
		filters.DriverID = driverID
	}

	shifts, err := r.shiftRepo.List(ctx, filters)
	if err != nil {
		r.logger.Error("Failed to get shifts for stats", zap.Error(err))
		return nil, err
	}

	// Вычисляем статистику
	stats := calculateShiftStats(shifts)
	return model.ShiftStatsFromEntity(stats), nil
}

// calculateShiftStats вычисляет статистику по сменам
func calculateShiftStats(shifts []*entities.DriverShift) *entities.ShiftStats {
	if len(shifts) == 0 {
		return &entities.ShiftStats{}
	}

	stats := &entities.ShiftStats{
		TotalShifts: len(shifts),
	}

	var totalHours float64
	var totalEarnings float64
	var totalTrips int
	var totalDistance float64

	for _, shift := range shifts {
		switch shift.Status {
		case entities.ShiftStatusActive:
			stats.ActiveShifts++
		case entities.ShiftStatusCompleted:
			stats.CompletedShifts++
		}

		duration := shift.GetDuration()
		totalHours += float64(duration) / 60.0
		totalEarnings += shift.TotalEarnings
		totalTrips += shift.TotalTrips
		totalDistance += shift.TotalDistance
	}

	stats.TotalHours = totalHours
	stats.TotalEarnings = totalEarnings
	stats.TotalTrips = totalTrips
	stats.TotalDistance = totalDistance

	if stats.TotalShifts > 0 {
		stats.AvgShiftDuration = totalHours / float64(stats.TotalShifts)
		stats.AvgShiftEarnings = totalEarnings / float64(stats.TotalShifts)
	}

	if totalHours > 0 {
		stats.AvgHourlyRate = totalEarnings / totalHours
	}

	return stats
}