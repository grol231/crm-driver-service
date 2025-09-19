package resolver

import (
	"context"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/interfaces/graphql/model"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mutationResolver struct{ *Resolver }

// CreateDriver создает нового водителя
func (r *mutationResolver) CreateDriver(ctx context.Context, input model.CreateDriverInput) (*model.Driver, error) {
	r.logger.Info("GraphQL: Creating driver", zap.String("phone", input.Phone))

	// Конвертируем входные данные в доменную сущность
	driver := &entities.Driver{
		Phone:          input.Phone,
		Email:          input.Email,
		FirstName:      input.FirstName,
		LastName:       input.LastName,
		MiddleName:     input.MiddleName,
		BirthDate:      input.BirthDate,
		PassportSeries: input.PassportSeries,
		PassportNumber: input.PassportNumber,
		LicenseNumber:  input.LicenseNumber,
		LicenseExpiry:  input.LicenseExpiry,
	}

	// Создаем водителя через сервис
	createdDriver, err := r.driverService.CreateDriver(ctx, driver)
	if err != nil {
		r.logger.Error("Failed to create driver", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(createdDriver), nil
}

// UpdateDriver обновляет данные водителя
func (r *mutationResolver) UpdateDriver(ctx context.Context, id model.UUID, input model.UpdateDriverInput) (*model.Driver, error) {
	r.logger.Info("GraphQL: Updating driver", zap.String("driver_id", id.String()))

	// Получаем текущего водителя
	existingDriver, err := r.driverService.GetDriverByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get existing driver", zap.Error(err))
		return nil, err
	}

	// Обновляем только переданные поля
	if input.Email != nil {
		existingDriver.Email = *input.Email
	}
	if input.FirstName != nil {
		existingDriver.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		existingDriver.LastName = *input.LastName
	}
	if input.MiddleName != nil {
		existingDriver.MiddleName = input.MiddleName
	}
	if input.BirthDate != nil {
		existingDriver.BirthDate = *input.BirthDate
	}
	if input.PassportSeries != nil {
		existingDriver.PassportSeries = *input.PassportSeries
	}
	if input.PassportNumber != nil {
		existingDriver.PassportNumber = *input.PassportNumber
	}
	if input.LicenseExpiry != nil {
		existingDriver.LicenseExpiry = *input.LicenseExpiry
	}

	// Обновляем водителя через сервис
	updatedDriver, err := r.driverService.UpdateDriver(ctx, existingDriver)
	if err != nil {
		r.logger.Error("Failed to update driver", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(updatedDriver), nil
}

// DeleteDriver удаляет водителя
func (r *mutationResolver) DeleteDriver(ctx context.Context, id model.UUID) (bool, error) {
	r.logger.Info("GraphQL: Deleting driver", zap.String("driver_id", id.String()))

	err := r.driverService.DeleteDriver(ctx, id)
	if err != nil {
		r.logger.Error("Failed to delete driver", zap.Error(err))
		return false, err
	}

	return true, nil
}

// ChangeDriverStatus изменяет статус водителя
func (r *mutationResolver) ChangeDriverStatus(ctx context.Context, id model.UUID, status model.Status) (*model.Driver, error) {
	r.logger.Info("GraphQL: Changing driver status",
		zap.String("driver_id", id.String()),
		zap.String("status", string(status)),
	)

	entityStatus := model.StatusToEntity(status)
	err := r.driverService.ChangeDriverStatus(ctx, id, entityStatus)
	if err != nil {
		r.logger.Error("Failed to change driver status", zap.Error(err))
		return nil, err
	}

	// Получаем обновленного водителя
	driver, err := r.driverService.GetDriverByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get updated driver", zap.Error(err))
		return nil, err
	}

	return model.DriverFromEntity(driver), nil
}

// UpdateDriverLocation обновляет местоположение водителя
func (r *mutationResolver) UpdateDriverLocation(ctx context.Context, driverID model.UUID, input model.LocationUpdateInput) (*model.DriverLocation, error) {
	r.logger.Info("GraphQL: Updating driver location", zap.String("driver_id", driverID.String()))

	// Определяем время записи
	recordedAt := time.Now()
	if input.Timestamp != nil {
		recordedAt = *input.Timestamp
	}

	// Создаем объект местоположения
	location := entities.NewDriverLocation(driverID, input.Latitude, input.Longitude, recordedAt)
	location.Altitude = input.Altitude
	location.Accuracy = input.Accuracy
	location.Speed = input.Speed
	location.Bearing = input.Bearing

	// Сохраняем через сервис
	err := r.locationService.UpdateLocation(ctx, location)
	if err != nil {
		r.logger.Error("Failed to update location", zap.Error(err))
		return nil, err
	}

	return model.DriverLocationFromEntity(location), nil
}

// BatchUpdateDriverLocations обновляет несколько местоположений водителя
func (r *mutationResolver) BatchUpdateDriverLocations(ctx context.Context, driverID model.UUID, locations []model.LocationUpdateInput) ([]*model.DriverLocation, error) {
	r.logger.Info("GraphQL: Batch updating driver locations",
		zap.String("driver_id", driverID.String()),
		zap.Int("count", len(locations)),
	)

	var entityLocations []*entities.DriverLocation
	for _, input := range locations {
		recordedAt := time.Now()
		if input.Timestamp != nil {
			recordedAt = *input.Timestamp
		}

		location := entities.NewDriverLocation(driverID, input.Latitude, input.Longitude, recordedAt)
		location.Altitude = input.Altitude
		location.Accuracy = input.Accuracy
		location.Speed = input.Speed
		location.Bearing = input.Bearing

		entityLocations = append(entityLocations, location)
	}

	// Сохраняем через сервис
	err := r.locationService.BatchUpdateLocations(ctx, entityLocations)
	if err != nil {
		r.logger.Error("Failed to batch update locations", zap.Error(err))
		return nil, err
	}

	return model.DriverLocationsFromEntity(entityLocations), nil
}

// AddDriverRating добавляет рейтинг водителю
func (r *mutationResolver) AddDriverRating(ctx context.Context, driverID model.UUID, orderID *model.UUID, customerID *model.UUID, input model.RatingInput) (*model.DriverRating, error) {
	r.logger.Info("GraphQL: Adding driver rating",
		zap.String("driver_id", driverID.String()),
		zap.Int("rating", input.Rating),
	)

	// Создаем объект рейтинга
	rating := entities.NewDriverRating(driverID, input.Rating, entities.RatingTypeCustomer)
	rating.OrderID = orderID
	rating.CustomerID = customerID
	rating.Comment = input.Comment

	if input.IsAnonymous != nil {
		rating.IsAnonymous = *input.IsAnonymous
	}

	// Конвертируем критерии оценки
	if input.CriteriaScores != nil {
		criteriaScores := make(map[string]int)
		if input.CriteriaScores.Cleanliness != nil {
			criteriaScores["cleanliness"] = *input.CriteriaScores.Cleanliness
		}
		if input.CriteriaScores.Driving != nil {
			criteriaScores["driving"] = *input.CriteriaScores.Driving
		}
		if input.CriteriaScores.Punctuality != nil {
			criteriaScores["punctuality"] = *input.CriteriaScores.Punctuality
		}
		if input.CriteriaScores.Politeness != nil {
			criteriaScores["politeness"] = *input.CriteriaScores.Politeness
		}
		if input.CriteriaScores.Navigation != nil {
			criteriaScores["navigation"] = *input.CriteriaScores.Navigation
		}
		rating.CriteriaScores = criteriaScores
	}

	// Сохраняем через репозиторий
	err := r.ratingRepo.Create(ctx, rating)
	if err != nil {
		r.logger.Error("Failed to add rating", zap.Error(err))
		return nil, err
	}

	// Обновляем рейтинг водителя
	err = r.driverService.UpdateDriverRating(ctx, driverID, rating.GetOverallScore())
	if err != nil {
		r.logger.Error("Failed to update driver rating", zap.Error(err))
	}

	return model.DriverRatingFromEntity(rating), nil
}

// VerifyRating верифицирует рейтинг
func (r *mutationResolver) VerifyRating(ctx context.Context, id model.UUID) (*model.DriverRating, error) {
	r.logger.Info("GraphQL: Verifying rating", zap.String("rating_id", id.String()))

	// Получаем рейтинг
	rating, err := r.ratingRepo.GetByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get rating", zap.Error(err))
		return nil, err
	}

	// Верифицируем
	rating.Verify()

	// Обновляем
	err = r.ratingRepo.Update(ctx, rating)
	if err != nil {
		r.logger.Error("Failed to update rating", zap.Error(err))
		return nil, err
	}

	return model.DriverRatingFromEntity(rating), nil
}

// StartShift начинает смену водителя
func (r *mutationResolver) StartShift(ctx context.Context, driverID model.UUID, input *model.ShiftStartInput) (*model.DriverShift, error) {
	r.logger.Info("GraphQL: Starting shift", zap.String("driver_id", driverID.String()))

	var startLocation *entities.DriverLocation
	if input != nil && input.Latitude != nil && input.Longitude != nil {
		startLocation = entities.NewDriverLocation(driverID, *input.Latitude, *input.Longitude, time.Now())
	}

	var vehicleID *uuid.UUID
	if input != nil && input.VehicleID != nil {
		vehicleID = input.VehicleID
	}

	// Создаем новую смену
	shift := entities.NewDriverShift(driverID, vehicleID, startLocation)

	// Добавляем заметки в метаданные
	if input != nil && input.Notes != nil {
		if shift.Metadata == nil {
			shift.Metadata = make(entities.Metadata)
		}
		shift.Metadata["start_notes"] = *input.Notes
	}

	// Сохраняем смену
	err := r.shiftRepo.Create(ctx, shift)
	if err != nil {
		r.logger.Error("Failed to start shift", zap.Error(err))
		return nil, err
	}

	// Обновляем статус водителя
	err = r.driverService.ChangeDriverStatus(ctx, driverID, entities.StatusOnShift)
	if err != nil {
		r.logger.Error("Failed to update driver status to on_shift", zap.Error(err))
	}

	return model.DriverShiftFromEntity(shift), nil
}

// EndShift завершает смену водителя
func (r *mutationResolver) EndShift(ctx context.Context, driverID model.UUID, input *model.ShiftEndInput) (*model.DriverShift, error) {
	r.logger.Info("GraphQL: Ending shift", zap.String("driver_id", driverID.String()))

	// Получаем активную смену
	shift, err := r.shiftRepo.GetActiveByDriverID(ctx, driverID)
	if err != nil {
		r.logger.Error("Failed to get active shift", zap.Error(err))
		return nil, err
	}

	var endLocation *entities.DriverLocation
	if input != nil && input.Latitude != nil && input.Longitude != nil {
		endLocation = entities.NewDriverLocation(driverID, *input.Latitude, *input.Longitude, time.Now())
	}

	// Завершаем смену
	shift.End(endLocation)

	// Добавляем заметки в метаданные
	if input != nil && input.Notes != nil {
		if shift.Metadata == nil {
			shift.Metadata = make(entities.Metadata)
		}
		shift.Metadata["end_notes"] = *input.Notes
	}

	// Обновляем смену
	err = r.shiftRepo.Update(ctx, shift)
	if err != nil {
		r.logger.Error("Failed to end shift", zap.Error(err))
		return nil, err
	}

	// Обновляем статус водителя
	err = r.driverService.ChangeDriverStatus(ctx, driverID, entities.StatusAvailable)
	if err != nil {
		r.logger.Error("Failed to update driver status to available", zap.Error(err))
	}

	return model.DriverShiftFromEntity(shift), nil
}

// SuspendShift приостанавливает смену водителя
func (r *mutationResolver) SuspendShift(ctx context.Context, driverID model.UUID) (*model.DriverShift, error) {
	r.logger.Info("GraphQL: Suspending shift", zap.String("driver_id", driverID.String()))

	// Получаем активную смену
	shift, err := r.shiftRepo.GetActiveByDriverID(ctx, driverID)
	if err != nil {
		r.logger.Error("Failed to get active shift", zap.Error(err))
		return nil, err
	}

	// Приостанавливаем смену
	shift.Suspend()

	// Обновляем смену
	err = r.shiftRepo.Update(ctx, shift)
	if err != nil {
		r.logger.Error("Failed to suspend shift", zap.Error(err))
		return nil, err
	}

	return model.DriverShiftFromEntity(shift), nil
}

// ResumeShift возобновляет смену водителя
func (r *mutationResolver) ResumeShift(ctx context.Context, driverID model.UUID) (*model.DriverShift, error) {
	r.logger.Info("GraphQL: Resuming shift", zap.String("driver_id", driverID.String()))

	// Получаем приостановленную смену
	filters := &entities.ShiftFilters{
		DriverID: &driverID,
		Status:   []entities.ShiftStatus{entities.ShiftStatusSuspended},
		Limit:    1,
	}

	shifts, err := r.shiftRepo.List(ctx, filters)
	if err != nil || len(shifts) == 0 {
		r.logger.Error("Failed to get suspended shift", zap.Error(err))
		return nil, entities.ErrShiftNotFound
	}

	shift := shifts[0]

	// Возобновляем смену
	shift.Resume()

	// Обновляем смену
	err = r.shiftRepo.Update(ctx, shift)
	if err != nil {
		r.logger.Error("Failed to resume shift", zap.Error(err))
		return nil, err
	}

	return model.DriverShiftFromEntity(shift), nil
}

// CancelShift отменяет смену водителя
func (r *mutationResolver) CancelShift(ctx context.Context, driverID model.UUID) (*model.DriverShift, error) {
	r.logger.Info("GraphQL: Cancelling shift", zap.String("driver_id", driverID.String()))

	// Получаем активную смену
	shift, err := r.shiftRepo.GetActiveByDriverID(ctx, driverID)
	if err != nil {
		r.logger.Error("Failed to get active shift", zap.Error(err))
		return nil, err
	}

	// Отменяем смену
	shift.Cancel()

	// Обновляем смену
	err = r.shiftRepo.Update(ctx, shift)
	if err != nil {
		r.logger.Error("Failed to cancel shift", zap.Error(err))
		return nil, err
	}

	// Обновляем статус водителя
	err = r.driverService.ChangeDriverStatus(ctx, driverID, entities.StatusAvailable)
	if err != nil {
		r.logger.Error("Failed to update driver status to available", zap.Error(err))
	}

	return model.DriverShiftFromEntity(shift), nil
}

// UploadDocument загружает документ водителя
func (r *mutationResolver) UploadDocument(ctx context.Context, driverID model.UUID, input model.DocumentUploadInput, file string) (*model.DriverDocument, error) {
	r.logger.Info("GraphQL: Uploading document",
		zap.String("driver_id", driverID.String()),
		zap.String("doc_type", string(input.DocumentType)),
	)

	// Создаем объект документа
	docType := model.DocumentTypeToEntity(input.DocumentType)
	document := entities.NewDriverDocument(
		driverID,
		docType,
		input.DocumentNumber,
		input.IssueDate,
		input.ExpiryDate,
		file, // В реальном приложении здесь должна быть ссылка на файл в storage
	)

	// Сохраняем документ
	err := r.documentRepo.Create(ctx, document)
	if err != nil {
		r.logger.Error("Failed to upload document", zap.Error(err))
		return nil, err
	}

	return model.DriverDocumentFromEntity(document), nil
}

// VerifyDocument верифицирует документ
func (r *mutationResolver) VerifyDocument(ctx context.Context, id model.UUID, input model.DocumentVerificationInput) (*model.DriverDocument, error) {
	r.logger.Info("GraphQL: Verifying document", zap.String("document_id", id.String()))

	// Получаем документ
	document, err := r.documentRepo.GetByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get document", zap.Error(err))
		return nil, err
	}

	// Верифицируем или отклоняем
	verifierID := "system" // В реальном приложении здесь должен быть ID верификатора из контекста
	status := model.VerificationStatusToEntity(input.Status)

	switch status {
	case entities.VerificationStatusVerified:
		document.Verify(verifierID)
	case entities.VerificationStatusRejected:
		reason := ""
		if input.RejectionReason != nil {
			reason = *input.RejectionReason
		}
		document.Reject(verifierID, reason)
	default:
		document.Status = status
		document.UpdatedAt = time.Now()
	}

	// Добавляем заметки в метаданные
	if input.Notes != nil {
		if document.Metadata == nil {
			document.Metadata = make(entities.Metadata)
		}
		document.Metadata["verification_notes"] = *input.Notes
	}

	// Обновляем документ
	err = r.documentRepo.Update(ctx, document)
	if err != nil {
		r.logger.Error("Failed to update document", zap.Error(err))
		return nil, err
	}

	return model.DriverDocumentFromEntity(document), nil
}