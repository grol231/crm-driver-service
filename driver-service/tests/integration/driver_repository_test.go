//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/repositories"
	"driver-service/tests/fixtures"
	"driver-service/tests/helpers"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DriverRepositoryTestSuite тестовый suite для DriverRepository
type DriverRepositoryTestSuite struct {
	suite.Suite
	testDB     *helpers.TestDB
	repository repositories.DriverRepository
	ctx        context.Context
}

// SetupSuite выполняется один раз перед всеми тестами
func (suite *DriverRepositoryTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDB(suite.T())
	suite.repository = repositories.NewDriverRepository(suite.testDB.DB, helpers.CreateTestLogger(suite.T()))
	suite.ctx = context.Background()
}

// TearDownSuite выполняется один раз после всех тестов
func (suite *DriverRepositoryTestSuite) TearDownSuite() {
	suite.testDB.TeardownTestDB(suite.T())
}

// SetupTest выполняется перед каждым тестом
func (suite *DriverRepositoryTestSuite) SetupTest() {
	suite.testDB.CleanupTables(suite.T())
}

// TestCreateDriver тестирует создание водителя
func (suite *DriverRepositoryTestSuite) TestCreateDriver() {
	// Arrange
	driver := fixtures.CreateTestDriver()

	// Act
	err := suite.repository.Create(suite.ctx, driver)

	// Assert
	require.NoError(suite.T(), err)

	// Проверяем, что водитель создан
	createdDriver, err := suite.repository.GetByID(suite.ctx, driver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), driver.Phone, createdDriver.Phone)
	assert.Equal(suite.T(), driver.Email, createdDriver.Email)
	assert.Equal(suite.T(), driver.Status, createdDriver.Status)
}

// TestCreateDriverDuplicate тестирует создание дублирующего водителя
func (suite *DriverRepositoryTestSuite) TestCreateDriverDuplicate() {
	// Arrange
	driver1 := fixtures.CreateTestDriver()
	driver2 := fixtures.CreateTestDriver()
	driver2.Phone = driver1.Phone // Дублируем телефон

	// Act
	err1 := suite.repository.Create(suite.ctx, driver1)
	err2 := suite.repository.Create(suite.ctx, driver2)

	// Assert
	require.NoError(suite.T(), err1)
	require.Error(suite.T(), err2) // Должна быть ошибка уникальности
}

// TestGetDriverByPhone тестирует поиск водителя по телефону
func (suite *DriverRepositoryTestSuite) TestGetDriverByPhone() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	err := suite.repository.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	foundDriver, err := suite.repository.GetByPhone(suite.ctx, driver.Phone)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), driver.ID, foundDriver.ID)
	assert.Equal(suite.T(), driver.Phone, foundDriver.Phone)
}

// TestGetDriverByEmail тестирует поиск водителя по email
func (suite *DriverRepositoryTestSuite) TestGetDriverByEmail() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	err := suite.repository.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	foundDriver, err := suite.repository.GetByEmail(suite.ctx, driver.Email)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), driver.ID, foundDriver.ID)
	assert.Equal(suite.T(), driver.Email, foundDriver.Email)
}

// TestGetDriverNotFound тестирует поиск несуществующего водителя
func (suite *DriverRepositoryTestSuite) TestGetDriverNotFound() {
	// Act
	_, err := suite.repository.GetByID(suite.ctx, uuid.New())

	// Assert
	assert.Equal(suite.T(), entities.ErrDriverNotFound, err)
}

// TestUpdateDriver тестирует обновление водителя
func (suite *DriverRepositoryTestSuite) TestUpdateDriver() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	err := suite.repository.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	driver.FirstName = "Обновленное Имя"
	driver.Email = "updated@example.com"
	driver.CurrentRating = 4.5

	err = suite.repository.Update(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Assert
	updatedDriver, err := suite.repository.GetByID(suite.ctx, driver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Обновленное Имя", updatedDriver.FirstName)
	assert.Equal(suite.T(), "updated@example.com", updatedDriver.Email)
	assert.Equal(suite.T(), 4.5, updatedDriver.CurrentRating)
	assert.True(suite.T(), updatedDriver.UpdatedAt.After(updatedDriver.CreatedAt))
}

// TestUpdateDriverStatus тестирует обновление статуса водителя
func (suite *DriverRepositoryTestSuite) TestUpdateDriverStatus() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	err := suite.repository.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	newStatus := entities.StatusAvailable
	err = suite.repository.UpdateStatus(suite.ctx, driver.ID, newStatus)
	require.NoError(suite.T(), err)

	// Assert
	updatedDriver, err := suite.repository.GetByID(suite.ctx, driver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), newStatus, updatedDriver.Status)
}

// TestSoftDeleteDriver тестирует мягкое удаление водителя
func (suite *DriverRepositoryTestSuite) TestSoftDeleteDriver() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	err := suite.repository.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	err = suite.repository.SoftDelete(suite.ctx, driver.ID)
	require.NoError(suite.T(), err)

	// Assert
	_, err = suite.repository.GetByID(suite.ctx, driver.ID)
	assert.Equal(suite.T(), entities.ErrDriverNotFound, err)
}

// TestListDrivers тестирует получение списка водителей
func (suite *DriverRepositoryTestSuite) TestListDrivers() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(5)
	for _, driver := range drivers {
		err := suite.repository.Create(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act
	filters := &entities.DriverFilters{
		Limit:         3,
		Offset:        0,
		SortBy:        "created_at",
		SortDirection: "desc",
	}

	result, err := suite.repository.List(suite.ctx, filters)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 3)
}

// TestListDriversWithFilters тестирует фильтрацию водителей
func (suite *DriverRepositoryTestSuite) TestListDriversWithFilters() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(3)
	drivers[0].Status = entities.StatusAvailable
	drivers[0].CurrentRating = 4.5
	drivers[1].Status = entities.StatusBusy
	drivers[1].CurrentRating = 3.8
	drivers[2].Status = entities.StatusAvailable
	drivers[2].CurrentRating = 4.8

	for _, driver := range drivers {
		err := suite.repository.Create(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act - фильтр по статусу
	filters := &entities.DriverFilters{
		Status: []entities.Status{entities.StatusAvailable},
	}

	result, err := suite.repository.List(suite.ctx, filters)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)

	for _, driver := range result {
		assert.Equal(suite.T(), entities.StatusAvailable, driver.Status)
	}
}

// TestListDriversWithRatingFilter тестирует фильтрацию по рейтингу
func (suite *DriverRepositoryTestSuite) TestListDriversWithRatingFilter() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(3)
	drivers[0].CurrentRating = 3.5
	drivers[1].CurrentRating = 4.2
	drivers[2].CurrentRating = 4.8

	for _, driver := range drivers {
		err := suite.repository.Create(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act
	minRating := 4.0
	filters := &entities.DriverFilters{
		MinRating: &minRating,
	}

	result, err := suite.repository.List(suite.ctx, filters)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)

	for _, driver := range result {
		assert.GreaterOrEqual(suite.T(), driver.CurrentRating, 4.0)
	}
}

// TestCountDrivers тестирует подсчет водителей
func (suite *DriverRepositoryTestSuite) TestCountDrivers() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(7)
	for _, driver := range drivers {
		err := suite.repository.Create(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act
	count, err := suite.repository.Count(suite.ctx, &entities.DriverFilters{})

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 7, count)
}

// TestExistsDriver тестирует проверку существования водителя
func (suite *DriverRepositoryTestSuite) TestExistsDriver() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	err := suite.repository.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	exists, err := suite.repository.Exists(suite.ctx, driver.Phone, driver.LicenseNumber)

	// Assert
	require.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	// Test non-existing
	exists, err = suite.repository.Exists(suite.ctx, "+79999999999", "NONEXISTENT")
	require.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

// TestIncrementTripCount тестирует увеличение счетчика поездок
func (suite *DriverRepositoryTestSuite) TestIncrementTripCount() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	driver.TotalTrips = 5
	err := suite.repository.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act
	err = suite.repository.IncrementTripCount(suite.ctx, driver.ID)
	require.NoError(suite.T(), err)

	// Assert
	updatedDriver, err := suite.repository.GetByID(suite.ctx, driver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 6, updatedDriver.TotalTrips)
}

// TestGetActiveDrivers тестирует получение активных водителей
func (suite *DriverRepositoryTestSuite) TestGetActiveDrivers() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(5)
	drivers[0].Status = entities.StatusAvailable
	drivers[1].Status = entities.StatusOnShift
	drivers[2].Status = entities.StatusBusy
	drivers[3].Status = entities.StatusInactive
	drivers[4].Status = entities.StatusBlocked

	for _, driver := range drivers {
		err := suite.repository.Create(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act
	activeDrivers, err := suite.repository.GetActiveDrivers(suite.ctx)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), activeDrivers, 3) // available, on_shift, busy

	for _, driver := range activeDrivers {
		assert.True(suite.T(), driver.IsActive())
	}
}

// TestConcurrentOperations тестирует конкурентные операции
func (suite *DriverRepositoryTestSuite) TestConcurrentOperations() {
	// Arrange
	driver := fixtures.CreateTestDriver()
	err := suite.repository.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)

	// Act - одновременное обновление рейтинга и счетчика поездок
	done := make(chan error, 2)

	go func() {
		done <- suite.repository.UpdateRating(suite.ctx, driver.ID, 4.5)
	}()

	go func() {
		done <- suite.repository.IncrementTripCount(suite.ctx, driver.ID)
	}()

	// Wait for both operations
	for i := 0; i < 2; i++ {
		err := <-done
		assert.NoError(suite.T(), err)
	}

	// Assert
	updatedDriver, err := suite.repository.GetByID(suite.ctx, driver.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 4.5, updatedDriver.CurrentRating)
	assert.Equal(suite.T(), 1, updatedDriver.TotalTrips)
}

// TestPaginationAndSorting тестирует пагинацию и сортировку
func (suite *DriverRepositoryTestSuite) TestPaginationAndSorting() {
	// Arrange
	drivers := fixtures.CreateMultipleTestDrivers(10)

	// Устанавливаем разные рейтинги для тестирования сортировки
	for i, driver := range drivers {
		driver.CurrentRating = float64(i+1) * 0.5 // 0.5, 1.0, 1.5, ...
		err := suite.repository.Create(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act - первая страница, сортировка по рейтингу (убывание)
	filters := &entities.DriverFilters{
		Limit:         3,
		Offset:        0,
		SortBy:        "current_rating",
		SortDirection: "desc",
	}

	page1, err := suite.repository.List(suite.ctx, filters)
	require.NoError(suite.T(), err)

	// Act - вторая страница
	filters.Offset = 3
	page2, err := suite.repository.List(suite.ctx, filters)
	require.NoError(suite.T(), err)

	// Assert
	assert.Len(suite.T(), page1, 3)
	assert.Len(suite.T(), page2, 3)

	// Проверяем сортировку по убыванию рейтинга
	assert.GreaterOrEqual(suite.T(), page1[0].CurrentRating, page1[1].CurrentRating)
	assert.GreaterOrEqual(suite.T(), page1[1].CurrentRating, page1[2].CurrentRating)

	// Проверяем пагинацию
	assert.Greater(suite.T(), page1[2].CurrentRating, page2[0].CurrentRating)
}

// TestTransactionRollback тестирует откат транзакции
func (suite *DriverRepositoryTestSuite) TestTransactionRollback() {
	// Arrange
	driver := fixtures.CreateTestDriver()

	// Act - создаем водителя в транзакции, которая откатывается
	err := suite.testDB.DB.Transaction(func(tx *sqlx.Tx) error {
		// Эмулируем создание в транзакции
		query := `
			INSERT INTO drivers (
				id, phone, email, first_name, last_name, license_number,
				license_expiry, status, current_rating, total_trips, 
				metadata, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			)`

		_, err := tx.Exec(query,
			driver.ID, driver.Phone, driver.Email, driver.FirstName,
			driver.LastName, driver.LicenseNumber, driver.LicenseExpiry,
			driver.Status, driver.CurrentRating, driver.TotalTrips,
			"{}", driver.CreatedAt, driver.UpdatedAt,
		)
		if err != nil {
			return err
		}

		// Принудительно откатываем транзакцию
		return assert.AnError
	})

	// Assert
	assert.Error(suite.T(), err)

	// Проверяем, что водитель не создан
	_, err = suite.repository.GetByID(suite.ctx, driver.ID)
	assert.Equal(suite.T(), entities.ErrDriverNotFound, err)
}

// TestDriverFiltersDateRange тестирует фильтрацию по дате
func (suite *DriverRepositoryTestSuite) TestDriverFiltersDateRange() {
	// Arrange
	now := time.Now()
	drivers := fixtures.CreateMultipleTestDrivers(3)

	// Устанавливаем разные даты создания
	drivers[0].CreatedAt = now.Add(-2 * time.Hour)
	drivers[1].CreatedAt = now.Add(-1 * time.Hour)
	drivers[2].CreatedAt = now

	for _, driver := range drivers {
		err := suite.repository.Create(suite.ctx, driver)
		require.NoError(suite.T(), err)
	}

	// Act - фильтр по дате (последний час)
	oneHourAgo := now.Add(-1 * time.Hour)
	filters := &entities.DriverFilters{
		CreatedAfter: &oneHourAgo,
	}

	result, err := suite.repository.List(suite.ctx, filters)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2) // drivers[1] и drivers[2]
}

// Запуск тестового suite
func TestDriverRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(DriverRepositoryTestSuite))
}
