package handlers

import (
	"net/http"
	"strconv"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/domain/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DriverHandler обработчик HTTP запросов для водителей
type DriverHandler struct {
	driverService services.DriverService
	logger        *zap.Logger
}

// NewDriverHandler создает новый DriverHandler
func NewDriverHandler(driverService services.DriverService, logger *zap.Logger) *DriverHandler {
	return &DriverHandler{
		driverService: driverService,
		logger:        logger,
	}
}

// CreateDriverRequest запрос на создание водителя
type CreateDriverRequest struct {
	Phone          string    `json:"phone" binding:"required"`
	Email          string    `json:"email" binding:"required,email"`
	FirstName      string    `json:"first_name" binding:"required"`
	LastName       string    `json:"last_name" binding:"required"`
	MiddleName     *string   `json:"middle_name,omitempty"`
	BirthDate      time.Time `json:"birth_date" binding:"required"`
	PassportSeries string    `json:"passport_series" binding:"required"`
	PassportNumber string    `json:"passport_number" binding:"required"`
	LicenseNumber  string    `json:"license_number" binding:"required"`
	LicenseExpiry  time.Time `json:"license_expiry" binding:"required"`
}

// UpdateDriverRequest запрос на обновление водителя
type UpdateDriverRequest struct {
	Email          *string    `json:"email,omitempty"`
	FirstName      *string    `json:"first_name,omitempty"`
	LastName       *string    `json:"last_name,omitempty"`
	MiddleName     *string    `json:"middle_name,omitempty"`
	BirthDate      *time.Time `json:"birth_date,omitempty"`
	PassportSeries *string    `json:"passport_series,omitempty"`
	PassportNumber *string    `json:"passport_number,omitempty"`
	LicenseExpiry  *time.Time `json:"license_expiry,omitempty"`
}

// ChangeStatusRequest запрос на изменение статуса
type ChangeStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// DriverResponse ответ с информацией о водителе
type DriverResponse struct {
	ID              uuid.UUID         `json:"id"`
	Phone           string            `json:"phone"`
	Email           string            `json:"email"`
	FirstName       string            `json:"first_name"`
	LastName        string            `json:"last_name"`
	MiddleName      *string           `json:"middle_name,omitempty"`
	BirthDate       time.Time         `json:"birth_date"`
	PassportSeries  string            `json:"passport_series"`
	PassportNumber  string            `json:"passport_number"`
	LicenseNumber   string            `json:"license_number"`
	LicenseExpiry   time.Time         `json:"license_expiry"`
	Status          entities.Status   `json:"status"`
	CurrentRating   float64           `json:"current_rating"`
	TotalTrips      int               `json:"total_trips"`
	Metadata        entities.Metadata `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// ListDriversResponse ответ со списком водителей
type ListDriversResponse struct {
	Drivers    []*DriverResponse `json:"drivers"`
	Total      int               `json:"total"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
	HasMore    bool              `json:"has_more"`
}

// ErrorResponse стандартный ответ с ошибкой
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// CreateDriver создает нового водителя
func (h *DriverHandler) CreateDriver(c *gin.Context) {
	var req CreateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid create driver request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	// Создаем объект водителя
	driver := &entities.Driver{
		Phone:          req.Phone,
		Email:          req.Email,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		MiddleName:     req.MiddleName,
		BirthDate:      req.BirthDate,
		PassportSeries: req.PassportSeries,
		PassportNumber: req.PassportNumber,
		LicenseNumber:  req.LicenseNumber,
		LicenseExpiry:  req.LicenseExpiry,
	}

	// Создаем водителя через сервис
	createdDriver, err := h.driverService.CreateDriver(c.Request.Context(), driver)
	if err != nil {
		h.handleServiceError(c, err, "Failed to create driver")
		return
	}

	response := h.toDriverResponse(createdDriver)
	c.JSON(http.StatusCreated, response)
}

// GetDriver получает водителя по ID
func (h *DriverHandler) GetDriver(c *gin.Context) {
	driverIDStr := c.Param("id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver ID format",
		})
		return
	}

	driver, err := h.driverService.GetDriverByID(c.Request.Context(), driverID)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get driver")
		return
	}

	response := h.toDriverResponse(driver)
	c.JSON(http.StatusOK, response)
}

// UpdateDriver обновляет данные водителя
func (h *DriverHandler) UpdateDriver(c *gin.Context) {
	driverIDStr := c.Param("id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver ID format",
		})
		return
	}

	var req UpdateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid update driver request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	// Получаем текущего водителя
	driver, err := h.driverService.GetDriverByID(c.Request.Context(), driverID)
	if err != nil {
		h.handleServiceError(c, err, "Failed to get driver for update")
		return
	}

	// Обновляем только переданные поля
	if req.Email != nil {
		driver.Email = *req.Email
	}
	if req.FirstName != nil {
		driver.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		driver.LastName = *req.LastName
	}
	if req.MiddleName != nil {
		driver.MiddleName = req.MiddleName
	}
	if req.BirthDate != nil {
		driver.BirthDate = *req.BirthDate
	}
	if req.PassportSeries != nil {
		driver.PassportSeries = *req.PassportSeries
	}
	if req.PassportNumber != nil {
		driver.PassportNumber = *req.PassportNumber
	}
	if req.LicenseExpiry != nil {
		driver.LicenseExpiry = *req.LicenseExpiry
	}

	// Обновляем водителя через сервис
	updatedDriver, err := h.driverService.UpdateDriver(c.Request.Context(), driver)
	if err != nil {
		h.handleServiceError(c, err, "Failed to update driver")
		return
	}

	response := h.toDriverResponse(updatedDriver)
	c.JSON(http.StatusOK, response)
}

// DeleteDriver удаляет водителя
func (h *DriverHandler) DeleteDriver(c *gin.Context) {
	driverIDStr := c.Param("id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver ID format",
		})
		return
	}

	err = h.driverService.DeleteDriver(c.Request.Context(), driverID)
	if err != nil {
		h.handleServiceError(c, err, "Failed to delete driver")
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListDrivers получает список водителей с фильтрами
func (h *DriverHandler) ListDrivers(c *gin.Context) {
	filters := &entities.DriverFilters{}

	// Парсим параметры запроса
	if statusStr := c.Query("status"); statusStr != "" {
		filters.Status = []entities.Status{entities.Status(statusStr)}
	}

	if minRatingStr := c.Query("min_rating"); minRatingStr != "" {
		if minRating, err := strconv.ParseFloat(minRatingStr, 64); err == nil {
			filters.MinRating = &minRating
		}
	}

	if maxRatingStr := c.Query("max_rating"); maxRatingStr != "" {
		if maxRating, err := strconv.ParseFloat(maxRatingStr, 64); err == nil {
			filters.MaxRating = &maxRating
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		} else {
			filters.Limit = 20 // значение по умолчанию
		}
	} else {
		filters.Limit = 20
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	if sortBy := c.Query("sort_by"); sortBy != "" {
		filters.SortBy = sortBy
	}

	if sortDirection := c.Query("sort_direction"); sortDirection != "" {
		filters.SortDirection = sortDirection
	}

	// Получаем список водителей
	drivers, err := h.driverService.ListDrivers(c.Request.Context(), filters)
	if err != nil {
		h.handleServiceError(c, err, "Failed to list drivers")
		return
	}

	// Получаем общее количество
	total, err := h.driverService.CountDrivers(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to count drivers",
			zap.Error(err),
		)
		// Не прерываем выполнение, просто логируем ошибку
		total = len(drivers)
	}

	// Преобразуем в ответ
	driverResponses := make([]*DriverResponse, len(drivers))
	for i, driver := range drivers {
		driverResponses[i] = h.toDriverResponse(driver)
	}

	response := &ListDriversResponse{
		Drivers: driverResponses,
		Total:   total,
		Limit:   filters.Limit,
		Offset:  filters.Offset,
		HasMore: filters.Offset+len(drivers) < total,
	}

	c.JSON(http.StatusOK, response)
}

// ChangeStatus изменяет статус водителя
func (h *DriverHandler) ChangeStatus(c *gin.Context) {
	driverIDStr := c.Param("id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver ID format",
		})
		return
	}

	var req ChangeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid change status request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	status := entities.Status(req.Status)

	// Изменяем статус через сервис
	err = h.driverService.ChangeDriverStatus(c.Request.Context(), driverID, status)
	if err != nil {
		h.handleServiceError(c, err, "Failed to change driver status")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Driver status changed successfully",
		"status":  status,
	})
}

// GetActiveDrivers получает список активных водителей
func (h *DriverHandler) GetActiveDrivers(c *gin.Context) {
	drivers, err := h.driverService.GetActiveDrivers(c.Request.Context())
	if err != nil {
		h.handleServiceError(c, err, "Failed to get active drivers")
		return
	}

	// Преобразуем в ответ
	driverResponses := make([]*DriverResponse, len(drivers))
	for i, driver := range drivers {
		driverResponses[i] = h.toDriverResponse(driver)
	}

	c.JSON(http.StatusOK, gin.H{
		"drivers": driverResponses,
		"count":   len(driverResponses),
	})
}

// toDriverResponse преобразует Driver entity в DriverResponse
func (h *DriverHandler) toDriverResponse(driver *entities.Driver) *DriverResponse {
	return &DriverResponse{
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
		Status:          driver.Status,
		CurrentRating:   driver.CurrentRating,
		TotalTrips:      driver.TotalTrips,
		Metadata:        driver.Metadata,
		CreatedAt:       driver.CreatedAt,
		UpdatedAt:       driver.UpdatedAt,
	}
}

// handleServiceError обрабатывает ошибки из сервисного слоя
func (h *DriverHandler) handleServiceError(c *gin.Context, err error, message string) {
	h.logger.Error(message, zap.Error(err))

	switch err {
	case entities.ErrDriverNotFound:
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Driver not found",
			Code:  "DRIVER_NOT_FOUND",
		})
	case entities.ErrDriverExists:
		c.JSON(http.StatusConflict, ErrorResponse{
			Error: "Driver already exists",
			Code:  "DRIVER_EXISTS",
		})
	case entities.ErrInvalidPhone, entities.ErrInvalidEmail, entities.ErrInvalidName,
		 entities.ErrInvalidLicense, entities.ErrInvalidPassport:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver data",
			Code:  "INVALID_DATA",
			Details: err.Error(),
		})
	case entities.ErrDriverNotAvailable:
		c.JSON(http.StatusConflict, ErrorResponse{
			Error: "Driver is not available",
			Code:  "DRIVER_NOT_AVAILABLE",
		})
	case entities.ErrDriverBlocked, entities.ErrDriverSuspended:
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error: "Driver is blocked or suspended",
			Code:  "DRIVER_BLOCKED",
		})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}