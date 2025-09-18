package entities

import (
	"time"

	"github.com/google/uuid"
)

// RatingType тип оценки
type RatingType string

const (
	RatingTypeCustomer     RatingType = "customer"
	RatingTypeSystem       RatingType = "system"
	RatingTypeAdmin        RatingType = "admin"
	RatingTypePeer         RatingType = "peer"
	RatingTypeAutomatic    RatingType = "automatic"
)

// DriverRating представляет оценку водителя
type DriverRating struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	DriverID       uuid.UUID      `json:"driver_id" db:"driver_id"`
	OrderID        *uuid.UUID     `json:"order_id,omitempty" db:"order_id"`
	CustomerID     *uuid.UUID     `json:"customer_id,omitempty" db:"customer_id"`
	Rating         int            `json:"rating" db:"rating"`
	Comment        *string        `json:"comment,omitempty" db:"comment"`
	RatingType     RatingType     `json:"rating_type" db:"rating_type"`
	CriteriaScores map[string]int `json:"criteria_scores" db:"criteria_scores"`
	IsVerified     bool           `json:"is_verified" db:"is_verified"`
	IsAnonymous    bool           `json:"is_anonymous" db:"is_anonymous"`
	Metadata       Metadata       `json:"metadata" db:"metadata"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
}

// IsValid проверяет валидность оценки
func (r *DriverRating) IsValid() bool {
	return r.Rating >= 1 && r.Rating <= 5
}

// GetOverallScore вычисляет общий балл на основе критериев
func (r *DriverRating) GetOverallScore() float64 {
	if len(r.CriteriaScores) == 0 {
		return float64(r.Rating)
	}

	total := 0
	count := 0
	for _, score := range r.CriteriaScores {
		total += score
		count++
	}

	if count == 0 {
		return float64(r.Rating)
	}

	return float64(total) / float64(count)
}

// Verify верифицирует оценку
func (r *DriverRating) Verify() {
	r.IsVerified = true
	r.UpdatedAt = time.Now()
}

// Validate проверяет валидность данных оценки
func (r *DriverRating) Validate() error {
	if r.DriverID == uuid.Nil {
		return ErrInvalidDriverID
	}

	if !r.IsValid() {
		return ErrInvalidRating
	}

	// Проверяем критерии оценки
	for _, score := range r.CriteriaScores {
		if score < 1 || score > 5 {
			return ErrInvalidCriteriaScore
		}
	}

	return nil
}

// NewDriverRating создает новую оценку водителя
func NewDriverRating(driverID uuid.UUID, rating int, ratingType RatingType) *DriverRating {
	now := time.Now()
	return &DriverRating{
		ID:             uuid.New(),
		DriverID:       driverID,
		Rating:         rating,
		RatingType:     ratingType,
		CriteriaScores: make(map[string]int),
		IsVerified:     false,
		IsAnonymous:    false,
		Metadata:       make(Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// RatingStats статистика рейтинга водителя
type RatingStats struct {
	DriverID          uuid.UUID          `json:"driver_id" db:"driver_id"`
	AverageRating     float64            `json:"average_rating" db:"average_rating"`
	TotalRatings      int                `json:"total_ratings" db:"total_ratings"`
	RatingDistribution map[int]int       `json:"rating_distribution" db:"rating_distribution"`
	CriteriaAverages  map[string]float64 `json:"criteria_averages" db:"criteria_averages"`
	LastRatingDate    *time.Time         `json:"last_rating_date,omitempty" db:"last_rating_date"`
	LastUpdated       time.Time          `json:"last_updated" db:"last_updated"`
}

// Calculate пересчитывает статистику на основе массива оценок
func (rs *RatingStats) Calculate(ratings []*DriverRating) {
	if len(ratings) == 0 {
		rs.AverageRating = 0
		rs.TotalRatings = 0
		rs.RatingDistribution = make(map[int]int)
		rs.CriteriaAverages = make(map[string]float64)
		rs.LastRatingDate = nil
		rs.LastUpdated = time.Now()
		return
	}

	// Инициализация
	rs.TotalRatings = len(ratings)
	rs.RatingDistribution = make(map[int]int)
	rs.CriteriaAverages = make(map[string]float64)
	
	criteriaSum := make(map[string]int)
	criteriaCount := make(map[string]int)
	
	totalRating := 0
	var latestDate *time.Time

	// Обработка всех оценок
	for _, rating := range ratings {
		totalRating += rating.Rating
		rs.RatingDistribution[rating.Rating]++

		// Обработка критериев
		for criteria, score := range rating.CriteriaScores {
			criteriaSum[criteria] += score
			criteriaCount[criteria]++
		}

		// Поиск последней даты оценки
		if latestDate == nil || rating.CreatedAt.After(*latestDate) {
			latestDate = &rating.CreatedAt
		}
	}

	// Вычисление средних значений
	rs.AverageRating = float64(totalRating) / float64(rs.TotalRatings)
	rs.LastRatingDate = latestDate

	// Вычисление средних по критериям
	for criteria, sum := range criteriaSum {
		if count := criteriaCount[criteria]; count > 0 {
			rs.CriteriaAverages[criteria] = float64(sum) / float64(count)
		}
	}

	rs.LastUpdated = time.Now()
}

// UpdateAverage обновляет средний рейтинг при добавлении новой оценки
func (rs *RatingStats) UpdateAverage(newRating int) {
	if rs.TotalRatings == 0 {
		rs.AverageRating = float64(newRating)
		rs.TotalRatings = 1
		rs.RatingDistribution = map[int]int{newRating: 1}
	} else {
		totalScore := rs.AverageRating * float64(rs.TotalRatings)
		totalScore += float64(newRating)
		rs.TotalRatings++
		rs.AverageRating = totalScore / float64(rs.TotalRatings)
		
		if rs.RatingDistribution == nil {
			rs.RatingDistribution = make(map[int]int)
		}
		rs.RatingDistribution[newRating]++
	}
	
	rs.LastUpdated = time.Now()
}

// GetPercentile возвращает процентиль оценок (например, 95-й процентиль)
func (rs *RatingStats) GetPercentile(percentile float64) float64 {
	if rs.TotalRatings == 0 {
		return 0
	}

	targetCount := int(float64(rs.TotalRatings) * percentile / 100)
	currentCount := 0

	// Идем от высоких оценок к низким
	for rating := 5; rating >= 1; rating-- {
		if count, exists := rs.RatingDistribution[rating]; exists {
			currentCount += count
			if currentCount >= targetCount {
				return float64(rating)
			}
		}
	}

	return 1.0
}

// NewRatingStats создает новую статистику рейтингов
func NewRatingStats(driverID uuid.UUID) *RatingStats {
	return &RatingStats{
		DriverID:          driverID,
		AverageRating:     0,
		TotalRatings:      0,
		RatingDistribution: make(map[int]int),
		CriteriaAverages:  make(map[string]float64),
		LastUpdated:       time.Now(),
	}
}

// RatingFilters фильтры для поиска оценок
type RatingFilters struct {
	DriverID   *uuid.UUID   `json:"driver_id,omitempty"`
	CustomerID *uuid.UUID   `json:"customer_id,omitempty"`
	OrderID    *uuid.UUID   `json:"order_id,omitempty"`
	RatingType []RatingType `json:"rating_type,omitempty"`
	MinRating  *int         `json:"min_rating,omitempty"`
	MaxRating  *int         `json:"max_rating,omitempty"`
	IsVerified *bool        `json:"is_verified,omitempty"`
	From       *time.Time   `json:"from,omitempty"`
	To         *time.Time   `json:"to,omitempty"`
	Limit      int          `json:"limit,omitempty"`
	Offset     int          `json:"offset,omitempty"`
	SortBy     string       `json:"sort_by,omitempty"`
	SortDirection string    `json:"sort_direction,omitempty"`
}

// RatingRequest запрос на добавление оценки
type RatingRequest struct {
	Rating         int            `json:"rating" binding:"required,min=1,max=5"`
	Comment        *string        `json:"comment,omitempty"`
	CriteriaScores map[string]int `json:"criteria_scores,omitempty"`
	IsAnonymous    *bool          `json:"is_anonymous,omitempty"`
}

// RatingResponse ответ с информацией об оценке
type RatingResponse struct {
	ID             uuid.UUID      `json:"id"`
	DriverID       uuid.UUID      `json:"driver_id"`
	OrderID        *uuid.UUID     `json:"order_id,omitempty"`
	CustomerID     *uuid.UUID     `json:"customer_id,omitempty"`
	Rating         int            `json:"rating"`
	Comment        *string        `json:"comment,omitempty"`
	RatingType     RatingType     `json:"rating_type"`
	CriteriaScores map[string]int `json:"criteria_scores,omitempty"`
	IsVerified     bool           `json:"is_verified"`
	IsAnonymous    bool           `json:"is_anonymous"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ToResponse конвертирует в ответ
func (r *DriverRating) ToResponse() *RatingResponse {
	return &RatingResponse{
		ID:             r.ID,
		DriverID:       r.DriverID,
		OrderID:        r.OrderID,
		CustomerID:     r.CustomerID,
		Rating:         r.Rating,
		Comment:        r.Comment,
		RatingType:     r.RatingType,
		CriteriaScores: r.CriteriaScores,
		IsVerified:     r.IsVerified,
		IsAnonymous:    r.IsAnonymous,
		CreatedAt:      r.CreatedAt,
	}
}

// RatingStatsResponse ответ со статистикой рейтингов
type RatingStatsResponse struct {
	DriverID           uuid.UUID          `json:"driver_id"`
	AverageRating      float64            `json:"average_rating"`
	TotalRatings       int                `json:"total_ratings"`
	RatingDistribution map[int]int        `json:"rating_distribution"`
	CriteriaAverages   map[string]float64 `json:"criteria_averages"`
	LastRatingDate     *time.Time         `json:"last_rating_date,omitempty"`
	Percentile95       float64            `json:"percentile_95"`
	Percentile90       float64            `json:"percentile_90"`
	LastUpdated        time.Time          `json:"last_updated"`
}

// ToResponse конвертирует статистику в ответ
func (rs *RatingStats) ToResponse() *RatingStatsResponse {
	return &RatingStatsResponse{
		DriverID:           rs.DriverID,
		AverageRating:      rs.AverageRating,
		TotalRatings:       rs.TotalRatings,
		RatingDistribution: rs.RatingDistribution,
		CriteriaAverages:   rs.CriteriaAverages,
		LastRatingDate:     rs.LastRatingDate,
		Percentile95:       rs.GetPercentile(95),
		Percentile90:       rs.GetPercentile(90),
		LastUpdated:        rs.LastUpdated,
	}
}