package resolver

import (
	"context"

	"driver-service/internal/domain/services"
	"driver-service/internal/interfaces/graphql/model"
	"driver-service/internal/repositories"

	"go.uber.org/zap"
)

// Resolver основной resolver для GraphQL
type Resolver struct {
	driverService   services.DriverService
	locationService LocationService
	driverRepo      repositories.DriverRepository
	locationRepo    LocationRepository
	ratingRepo      RatingRepository
	shiftRepo       ShiftRepository
	documentRepo    repositories.DocumentRepository
	logger          *zap.Logger
}

// NewResolver создает новый resolver
func NewResolver(
	driverService services.DriverService,
	locationService LocationService,
	driverRepo repositories.DriverRepository,
	locationRepo LocationRepository,
	ratingRepo RatingRepository,
	shiftRepo ShiftRepository,
	documentRepo repositories.DocumentRepository,
	logger *zap.Logger,
) *Resolver {
	return &Resolver{
		driverService:   driverService,
		locationService: locationService,
		driverRepo:      driverRepo,
		locationRepo:    locationRepo,
		ratingRepo:      ratingRepo,
		shiftRepo:       shiftRepo,
		documentRepo:    documentRepo,
		logger:          logger,
	}
}

// Query возвращает QueryResolver
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

// Mutation возвращает MutationResolver
func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}

// Subscription возвращает SubscriptionResolver
func (r *Resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}

// Driver resolver
func (r *Resolver) Driver() DriverResolver {
	return &driverResolver{r}
}

// DriverLocation resolver
func (r *Resolver) DriverLocation() DriverLocationResolver {
	return &driverLocationResolver{r}
}

// DriverRating resolver
func (r *Resolver) DriverRating() DriverRatingResolver {
	return &driverRatingResolver{r}
}

// DriverShift resolver
func (r *Resolver) DriverShift() DriverShiftResolver {
	return &driverShiftResolver{r}
}

// DriverDocument resolver
func (r *Resolver) DriverDocument() DriverDocumentResolver {
	return &driverDocumentResolver{r}
}

// Интерфейсы для resolvers (будут сгенерированы gqlgen)
type QueryResolver interface {
	Driver(ctx context.Context, id model.UUID) (*model.Driver, error)
	Drivers(ctx context.Context, filters *model.DriverFilters, limit *int, offset *int) (*model.DriversConnection, error)
	ActiveDrivers(ctx context.Context) ([]*model.Driver, error)
	DriverByPhone(ctx context.Context, phone string) (*model.Driver, error)
	DriverByEmail(ctx context.Context, email string) (*model.Driver, error)
	DriverLocation(ctx context.Context, id model.UUID) (*model.DriverLocation, error)
	DriverLocations(ctx context.Context, filters model.LocationFilters, limit *int, offset *int) (*model.LocationsConnection, error)
	NearbyDrivers(ctx context.Context, latitude float64, longitude float64, radiusKM *float64, limit *int) ([]*model.Driver, error)
	DriverRating(ctx context.Context, id model.UUID) (*model.DriverRating, error)
	DriverRatings(ctx context.Context, filters model.RatingFilters, limit *int, offset *int) (*model.RatingsConnection, error)
	DriverRatingStats(ctx context.Context, driverID model.UUID) (*model.RatingStats, error)
	DriverShift(ctx context.Context, id model.UUID) (*model.DriverShift, error)
	DriverShifts(ctx context.Context, filters model.ShiftFilters, limit *int, offset *int) (*model.ShiftsConnection, error)
	ActiveShifts(ctx context.Context) ([]*model.DriverShift, error)
	DriverDocument(ctx context.Context, id model.UUID) (*model.DriverDocument, error)
	DriverDocuments(ctx context.Context, filters model.DocumentFilters, limit *int, offset *int) ([]*model.DriverDocument, error)
	LocationStats(ctx context.Context, driverID model.UUID, from *model.Time, to *model.Time) (*model.LocationStats, error)
	ShiftStats(ctx context.Context, driverID *model.UUID, from *model.Time, to *model.Time) (*model.ShiftStats, error)
}

type MutationResolver interface {
	CreateDriver(ctx context.Context, input model.CreateDriverInput) (*model.Driver, error)
	UpdateDriver(ctx context.Context, id model.UUID, input model.UpdateDriverInput) (*model.Driver, error)
	DeleteDriver(ctx context.Context, id model.UUID) (bool, error)
	ChangeDriverStatus(ctx context.Context, id model.UUID, status model.Status) (*model.Driver, error)
	UpdateDriverLocation(ctx context.Context, driverID model.UUID, input model.LocationUpdateInput) (*model.DriverLocation, error)
	BatchUpdateDriverLocations(ctx context.Context, driverID model.UUID, locations []model.LocationUpdateInput) ([]*model.DriverLocation, error)
	AddDriverRating(ctx context.Context, driverID model.UUID, orderID *model.UUID, customerID *model.UUID, input model.RatingInput) (*model.DriverRating, error)
	VerifyRating(ctx context.Context, id model.UUID) (*model.DriverRating, error)
	StartShift(ctx context.Context, driverID model.UUID, input *model.ShiftStartInput) (*model.DriverShift, error)
	EndShift(ctx context.Context, driverID model.UUID, input *model.ShiftEndInput) (*model.DriverShift, error)
	SuspendShift(ctx context.Context, driverID model.UUID) (*model.DriverShift, error)
	ResumeShift(ctx context.Context, driverID model.UUID) (*model.DriverShift, error)
	CancelShift(ctx context.Context, driverID model.UUID) (*model.DriverShift, error)
	UploadDocument(ctx context.Context, driverID model.UUID, input model.DocumentUploadInput, file string) (*model.DriverDocument, error)
	VerifyDocument(ctx context.Context, id model.UUID, input model.DocumentVerificationInput) (*model.DriverDocument, error)
}

type SubscriptionResolver interface {
	DriverLocationUpdated(ctx context.Context, driverID model.UUID) (<-chan *model.DriverLocation, error)
	DriverStatusChanged(ctx context.Context, driverID model.UUID) (<-chan *model.Driver, error)
	ShiftUpdated(ctx context.Context, driverID model.UUID) (<-chan *model.DriverShift, error)
	NewRating(ctx context.Context, driverID model.UUID) (<-chan *model.DriverRating, error)
}

type DriverResolver interface {
	FullName(ctx context.Context, obj *model.Driver) (string, error)
	Documents(ctx context.Context, obj *model.Driver) ([]*model.DriverDocument, error)
	CurrentLocation(ctx context.Context, obj *model.Driver) (*model.DriverLocation, error)
	LocationHistory(ctx context.Context, obj *model.Driver, limit *int, offset *int) ([]*model.DriverLocation, error)
	Ratings(ctx context.Context, obj *model.Driver, limit *int, offset *int) ([]*model.DriverRating, error)
	RatingStats(ctx context.Context, obj *model.Driver) (*model.RatingStats, error)
	ActiveShift(ctx context.Context, obj *model.Driver) (*model.DriverShift, error)
	Shifts(ctx context.Context, obj *model.Driver, limit *int, offset *int) ([]*model.DriverShift, error)
	IsActive(ctx context.Context, obj *model.Driver) (bool, error)
	CanReceiveOrders(ctx context.Context, obj *model.Driver) (bool, error)
	IsLicenseExpired(ctx context.Context, obj *model.Driver) (bool, error)
}

type DriverLocationResolver interface {
	Driver(ctx context.Context, obj *model.DriverLocation) (*model.Driver, error)
	IsHighAccuracy(ctx context.Context, obj *model.DriverLocation) (bool, error)
	IsValidLocation(ctx context.Context, obj *model.DriverLocation) (bool, error)
}

type DriverRatingResolver interface {
	Driver(ctx context.Context, obj *model.DriverRating) (*model.Driver, error)
	IsValid(ctx context.Context, obj *model.DriverRating) (bool, error)
	OverallScore(ctx context.Context, obj *model.DriverRating) (float64, error)
}

type DriverShiftResolver interface {
	Driver(ctx context.Context, obj *model.DriverShift) (*model.Driver, error)
	Duration(ctx context.Context, obj *model.DriverShift) (int, error)
	IsActive(ctx context.Context, obj *model.DriverShift) (bool, error)
	AverageEarningsPerTrip(ctx context.Context, obj *model.DriverShift) (float64, error)
	AverageDistancePerTrip(ctx context.Context, obj *model.DriverShift) (float64, error)
	EarningsPerHour(ctx context.Context, obj *model.DriverShift) (float64, error)
}

type DriverDocumentResolver interface {
	Driver(ctx context.Context, obj *model.DriverDocument) (*model.Driver, error)
	IsExpired(ctx context.Context, obj *model.DriverDocument) (bool, error)
	IsVerified(ctx context.Context, obj *model.DriverDocument) (bool, error)
	DaysUntilExpiry(ctx context.Context, obj *model.DriverDocument) (int, error)
}