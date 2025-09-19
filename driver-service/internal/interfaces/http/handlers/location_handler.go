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

// LocationHandler обработчик HTTP запросов для местоположений
type LocationHandler struct {
	locationService services.LocationService
	logger          *zap.Logger
}

// NewLocationHandler создает новый LocationHandler
func NewLocationHandler(locationService services.LocationService, logger *zap.Logger) *LocationHandler {
	return &LocationHandler{
		locationService: locationService,
		logger:          logger,
	}
}

// UpdateLocationRequest запрос на обновление местоположения
type UpdateLocationRequest struct {
	Latitude  float64  `json:"latitude" binding:"required"`
	Longitude float64  `json:"longitude" binding:"required"`
	Altitude  *float64 `json:"altitude,omitempty"`
	Accuracy  *float64 `json:"accuracy,omitempty"`
	Speed     *float64 `json:"speed,omitempty"`
	Bearing   *float64 `json:"bearing,omitempty"`
	Timestamp *int64   `json:"timestamp,omitempty"`
}

// BatchLocationRequest запрос на пакетное обновление местоположений
type BatchLocationRequest struct {
	Locations []UpdateLocationRequest `json:"locations" binding:"required,min=1"`
}

// LocationResponse ответ с местоположением
type LocationResponse struct {
	ID         uuid.UUID `json:"id"`
	DriverID   uuid.UUID `json:"driver_id"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Altitude   *float64  `json:"altitude,omitempty"`
	Accuracy   *float64  `json:"accuracy,omitempty"`
	Speed      *float64  `json:"speed,omitempty"`
	Bearing    *float64  `json:"bearing,omitempty"`
	Address    *string   `json:"address,omitempty"`
	RecordedAt time.Time `json:"recorded_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// LocationHistoryResponse ответ с историей местоположений
type LocationHistoryResponse struct {
	Locations []*LocationResponse    `json:"locations"`
	Stats     *entities.LocationStats `json:"stats"`
	Count     int                    `json:"count"`
}

// NearbyDriversResponse ответ с водителями поблизости
type NearbyDriversResponse struct {
	Drivers []*NearbyDriverInfo `json:"drivers"`
	Count   int                 `json:"count"`
}

// NearbyDriverInfo информация о водителе поблизости
type NearbyDriverInfo struct {
	DriverID  uuid.UUID `json:"driver_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Distance  float64   `json:"distance_km,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpdateLocation обновляет местоположение водителя
func (h *LocationHandler) UpdateLocation(c *gin.Context) {
	driverIDStr := c.Param("id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver ID format",
		})
		return
	}

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid update location request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	// Создаем объект местоположения
	recordedAt := time.Now()
	if req.Timestamp != nil {
		recordedAt = time.Unix(*req.Timestamp, 0)
	}

	location := entities.NewDriverLocation(driverID, req.Latitude, req.Longitude, recordedAt)
	location.Altitude = req.Altitude
	location.Accuracy = req.Accuracy
	location.Speed = req.Speed
	location.Bearing = req.Bearing

	// Обновляем местоположение через сервис
	err = h.locationService.UpdateLocation(c.Request.Context(), location)
	if err != nil {
		h.handleLocationServiceError(c, err, "Failed to update location")
		return
	}

	response := h.toLocationResponse(location)
	c.JSON(http.StatusOK, response)
}

// BatchUpdateLocations пакетное обновление местоположений
func (h *LocationHandler) BatchUpdateLocations(c *gin.Context) {
	driverIDStr := c.Param("id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver ID format",
		})
		return
	}

	var req BatchLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid batch update request",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request data",
			Details: err.Error(),
		})
		return
	}

	// Создаем объекты местоположений
	locations := make([]*entities.DriverLocation, len(req.Locations))
	for i, locReq := range req.Locations {
		recordedAt := time.Now()
		if locReq.Timestamp != nil {
			recordedAt = time.Unix(*locReq.Timestamp, 0)
		}

		location := entities.NewDriverLocation(driverID, locReq.Latitude, locReq.Longitude, recordedAt)
		location.Altitude = locReq.Altitude
		location.Accuracy = locReq.Accuracy
		location.Speed = locReq.Speed
		location.Bearing = locReq.Bearing

		locations[i] = location
	}

	// Пакетно обновляем местоположения
	err = h.locationService.BatchUpdateLocations(c.Request.Context(), locations)
	if err != nil {
		h.handleLocationServiceError(c, err, "Failed to batch update locations")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Locations updated successfully",
		"count":   len(locations),
	})
}

// GetCurrentLocation получает текущее местоположение водителя
func (h *LocationHandler) GetCurrentLocation(c *gin.Context) {
	driverIDStr := c.Param("id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver ID format",
		})
		return
	}

	location, err := h.locationService.GetCurrentLocation(c.Request.Context(), driverID)
	if err != nil {
		h.handleLocationServiceError(c, err, "Failed to get current location")
		return
	}

	response := h.toLocationResponse(location)
	c.JSON(http.StatusOK, response)
}

// GetLocationHistory получает историю местоположений водителя
func (h *LocationHandler) GetLocationHistory(c *gin.Context) {
	driverIDStr := c.Param("id")
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid driver ID format",
		})
		return
	}

	// Парсим параметры времени
	var from, to time.Time
	
	if fromStr := c.Query("from"); fromStr != "" {
		if fromUnix, err := strconv.ParseInt(fromStr, 10, 64); err == nil {
			from = time.Unix(fromUnix, 0)
		} else if parsedTime, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = parsedTime
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid 'from' time format",
				Details: "Use Unix timestamp or RFC3339 format",
			})
			return
		}
	} else {
		from = time.Now().Add(-24 * time.Hour) // По умолчанию последние 24 часа
	}

	if toStr := c.Query("to"); toStr != "" {
		if toUnix, err := strconv.ParseInt(toStr, 10, 64); err == nil {
			to = time.Unix(toUnix, 0)
		} else if parsedTime, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = parsedTime
		} else {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: "Invalid 'to' time format",
				Details: "Use Unix timestamp or RFC3339 format",
			})
			return
		}
	} else {
		to = time.Now()
	}

	// Получаем историю местоположений
	locations, err := h.locationService.GetLocationHistory(c.Request.Context(), driverID, from, to)
	if err != nil {
		h.handleLocationServiceError(c, err, "Failed to get location history")
		return
	}

	// Получаем статистику
	stats, err := h.locationService.GetLocationStats(c.Request.Context(), driverID, from, to)
	if err != nil {
		h.logger.Error("Failed to get location stats",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
		)
		// Не прерываем выполнение, просто не возвращаем статистику
	}

	// Преобразуем в ответ
	locationResponses := make([]*LocationResponse, len(locations))
	for i, location := range locations {
		locationResponses[i] = h.toLocationResponse(location)
	}

	response := &LocationHistoryResponse{
		Locations: locationResponses,
		Stats:     stats,
		Count:     len(locationResponses),
	}

	c.JSON(http.StatusOK, response)
}

// GetNearbyDrivers получает водителей поблизости
func (h *LocationHandler) GetNearbyDrivers(c *gin.Context) {
	// Парсим координаты
	latStr := c.Query("latitude")
	lonStr := c.Query("longitude")
	if latStr == "" || lonStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Latitude and longitude are required",
		})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid latitude format",
		})
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid longitude format",
		})
		return
	}

	// Парсим радиус
	radiusKm := 5.0 // По умолчанию 5 км
	if radiusStr := c.Query("radius_km"); radiusStr != "" {
		if parsed, err := strconv.ParseFloat(radiusStr, 64); err == nil && parsed > 0 {
			radiusKm = parsed
		}
	}

	// Парсим лимит
	limit := 20 // По умолчанию 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Получаем водителей поблизости
	locations, err := h.locationService.GetNearbyDrivers(c.Request.Context(), lat, lon, radiusKm, limit)
	if err != nil {
		h.handleLocationServiceError(c, err, "Failed to get nearby drivers")
		return
	}

	// Преобразуем в ответ
	centerLocation := &entities.DriverLocation{
		Latitude:  lat,
		Longitude: lon,
	}

	nearbyDrivers := make([]*NearbyDriverInfo, len(locations))
	for i, location := range locations {
		distance := centerLocation.DistanceTo(location)
		nearbyDrivers[i] = &NearbyDriverInfo{
			DriverID:  location.DriverID,
			Latitude:  location.Latitude,
			Longitude: location.Longitude,
			Distance:  distance,
			UpdatedAt: location.RecordedAt,
		}
	}

	response := &NearbyDriversResponse{
		Drivers: nearbyDrivers,
		Count:   len(nearbyDrivers),
	}

	c.JSON(http.StatusOK, response)
}

// toLocationResponse преобразует DriverLocation entity в LocationResponse
func (h *LocationHandler) toLocationResponse(location *entities.DriverLocation) *LocationResponse {
	return &LocationResponse{
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

// handleLocationServiceError обрабатывает ошибки из LocationService
func (h *LocationHandler) handleLocationServiceError(c *gin.Context, err error, message string) {
	h.logger.Error(message, zap.Error(err))

	switch err {
	case entities.ErrLocationNotFound:
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Location not found",
			Code:  "LOCATION_NOT_FOUND",
		})
	case entities.ErrInvalidLocation:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid location coordinates",
			Code:  "INVALID_LOCATION",
		})
	case entities.ErrInvalidTimestamp:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid timestamp",
			Code:  "INVALID_TIMESTAMP",
		})
	case entities.ErrLocationTooOld:
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Location data is too old",
			Code:  "LOCATION_TOO_OLD",
		})
	case entities.ErrDriverNotFound:
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Driver not found",
			Code:  "DRIVER_NOT_FOUND",
		})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}