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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DocumentRepositoryTestSuite тестовый suite для DocumentRepository
type DocumentRepositoryTestSuite struct {
	suite.Suite
	testDB       *helpers.TestDB
	documentRepo repositories.DocumentRepository
	driverRepo   repositories.DriverRepository
	ctx          context.Context
	testDriverID uuid.UUID
}

// SetupSuite выполняется один раз перед всеми тестами
func (suite *DocumentRepositoryTestSuite) SetupSuite() {
	suite.testDB = helpers.SetupTestDB(suite.T())
	logger := helpers.CreateTestLogger(suite.T())

	suite.documentRepo = repositories.NewDocumentRepository(suite.testDB.DB, logger)
	suite.driverRepo = repositories.NewDriverRepository(suite.testDB.DB, logger)
	suite.ctx = context.Background()
}

// TearDownSuite выполняется один раз после всех тестов
func (suite *DocumentRepositoryTestSuite) TearDownSuite() {
	suite.testDB.TeardownTestDB(suite.T())
}

// SetupTest выполняется перед каждым тестом
func (suite *DocumentRepositoryTestSuite) SetupTest() {
	suite.testDB.CleanupTables(suite.T())

	// Создаем тестового водителя
	driver := fixtures.CreateTestDriver()
	err := suite.driverRepo.Create(suite.ctx, driver)
	require.NoError(suite.T(), err)
	suite.testDriverID = driver.ID
}

// TestCreateDocument тестирует создание документа
func (suite *DocumentRepositoryTestSuite) TestCreateDocument() {
	// Arrange
	document := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)

	// Act
	err := suite.documentRepo.Create(suite.ctx, document)

	// Assert
	require.NoError(suite.T(), err)

	// Проверяем, что документ создан
	createdDoc, err := suite.documentRepo.GetByID(suite.ctx, document.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), document.DriverID, createdDoc.DriverID)
	assert.Equal(suite.T(), document.DocumentType, createdDoc.DocumentType)
	assert.Equal(suite.T(), document.DocumentNumber, createdDoc.DocumentNumber)
	assert.Equal(suite.T(), entities.VerificationStatusPending, createdDoc.Status)
}

// TestCreateDuplicateDocument тестирует создание дублирующего документа
func (suite *DocumentRepositoryTestSuite) TestCreateDuplicateDocument() {
	// Arrange
	doc1 := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)
	doc2 := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)

	// Act
	err1 := suite.documentRepo.Create(suite.ctx, doc1)
	err2 := suite.documentRepo.Create(suite.ctx, doc2)

	// Assert
	require.NoError(suite.T(), err1)
	require.Error(suite.T(), err2) // Должна быть ошибка уникальности (driver_id, document_type)
}

// TestGetDocumentsByDriverID тестирует получение всех документов водителя
func (suite *DocumentRepositoryTestSuite) TestGetDocumentsByDriverID() {
	// Arrange
	documents := []*entities.DriverDocument{
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeMedicalCert),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypePassport),
	}

	for _, doc := range documents {
		err := suite.documentRepo.Create(suite.ctx, doc)
		require.NoError(suite.T(), err)
	}

	// Act
	driverDocs, err := suite.documentRepo.GetByDriverID(suite.ctx, suite.testDriverID)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), driverDocs, 3)

	// Проверяем, что все документы принадлежат водителю
	for _, doc := range driverDocs {
		assert.Equal(suite.T(), suite.testDriverID, doc.DriverID)
	}
}

// TestGetDocumentByType тестирует получение документа определенного типа
func (suite *DocumentRepositoryTestSuite) TestGetDocumentByType() {
	// Arrange
	licenseDoc := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)
	medicalDoc := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeMedicalCert)

	err := suite.documentRepo.Create(suite.ctx, licenseDoc)
	require.NoError(suite.T(), err)
	err = suite.documentRepo.Create(suite.ctx, medicalDoc)
	require.NoError(suite.T(), err)

	// Act
	foundDoc, err := suite.documentRepo.GetByDriverIDAndType(suite.ctx, suite.testDriverID, entities.DocumentTypeDriverLicense)

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), licenseDoc.ID, foundDoc.ID)
	assert.Equal(suite.T(), entities.DocumentTypeDriverLicense, foundDoc.DocumentType)
}

// TestUpdateDocumentStatus тестирует обновление статуса документа
func (suite *DocumentRepositoryTestSuite) TestUpdateDocumentStatus() {
	// Arrange
	document := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)
	err := suite.documentRepo.Create(suite.ctx, document)
	require.NoError(suite.T(), err)

	// Act - верификация документа
	verifierID := "admin-user-123"
	err = suite.documentRepo.UpdateStatus(suite.ctx, document.ID, entities.VerificationStatusVerified, &verifierID, nil)
	require.NoError(suite.T(), err)

	// Assert
	updatedDoc, err := suite.documentRepo.GetByID(suite.ctx, document.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.VerificationStatusVerified, updatedDoc.Status)
	assert.Equal(suite.T(), verifierID, *updatedDoc.VerifiedBy)
	assert.NotNil(suite.T(), updatedDoc.VerifiedAt)
	assert.Nil(suite.T(), updatedDoc.RejectionReason)
}

// TestRejectDocument тестирует отклонение документа
func (suite *DocumentRepositoryTestSuite) TestRejectDocument() {
	// Arrange
	document := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)
	err := suite.documentRepo.Create(suite.ctx, document)
	require.NoError(suite.T(), err)

	// Act
	verifierID := "admin-user-123"
	rejectionReason := "Документ нечитаемый"
	err = suite.documentRepo.UpdateStatus(suite.ctx, document.ID, entities.VerificationStatusRejected, &verifierID, &rejectionReason)
	require.NoError(suite.T(), err)

	// Assert
	updatedDoc, err := suite.documentRepo.GetByID(suite.ctx, document.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.VerificationStatusRejected, updatedDoc.Status)
	assert.Equal(suite.T(), verifierID, *updatedDoc.VerifiedBy)
	assert.Equal(suite.T(), rejectionReason, *updatedDoc.RejectionReason)
}

// TestGetExpiringDocuments тестирует получение истекающих документов
func (suite *DocumentRepositoryTestSuite) TestGetExpiringDocuments() {
	// Arrange
	now := time.Now()

	documents := []*entities.DriverDocument{
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeMedicalCert),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypePassport),
	}

	// Устанавливаем разные даты истечения
	documents[0].ExpiryDate = now.Add(5 * 24 * time.Hour)  // Истекает через 5 дней
	documents[1].ExpiryDate = now.Add(15 * 24 * time.Hour) // Истекает через 15 дней
	documents[2].ExpiryDate = now.Add(40 * 24 * time.Hour) // Истекает через 40 дней

	// Верифицируем документы
	for _, doc := range documents {
		doc.Status = entities.VerificationStatusVerified
		err := suite.documentRepo.Create(suite.ctx, doc)
		require.NoError(suite.T(), err)
	}

	// Act - ищем документы, истекающие в течение 30 дней
	expiringDocs, err := suite.documentRepo.GetExpiring(suite.ctx, 30)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), expiringDocs, 2) // Первые два документа

	// Проверяем сортировку по дате истечения
	assert.True(suite.T(), expiringDocs[0].ExpiryDate.Before(expiringDocs[1].ExpiryDate))
}

// TestGetExpiredDocuments тестирует получение истекших документов
func (suite *DocumentRepositoryTestSuite) TestGetExpiredDocuments() {
	// Arrange
	now := time.Now()

	documents := []*entities.DriverDocument{
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeMedicalCert),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypePassport),
	}

	// Устанавливаем даты истечения
	documents[0].ExpiryDate = now.Add(-5 * 24 * time.Hour) // Истек 5 дней назад
	documents[1].ExpiryDate = now.Add(-1 * 24 * time.Hour) // Истек вчера
	documents[2].ExpiryDate = now.Add(10 * 24 * time.Hour) // Еще действует

	// Устанавливаем статусы
	documents[0].Status = entities.VerificationStatusVerified
	documents[1].Status = entities.VerificationStatusPending
	documents[2].Status = entities.VerificationStatusVerified

	for _, doc := range documents {
		err := suite.documentRepo.Create(suite.ctx, doc)
		require.NoError(suite.T(), err)
	}

	// Act
	expiredDocs, err := suite.documentRepo.GetExpired(suite.ctx)

	// Assert
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), expiredDocs, 2) // Первые два документа
}

// TestMarkDocumentsExpired тестирует массовое помечание документов как истекших
func (suite *DocumentRepositoryTestSuite) TestMarkDocumentsExpired() {
	// Arrange
	documents := []*entities.DriverDocument{
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeMedicalCert),
	}

	for _, doc := range documents {
		doc.Status = entities.VerificationStatusVerified
		err := suite.documentRepo.Create(suite.ctx, doc)
		require.NoError(suite.T(), err)
	}

	documentIDs := []uuid.UUID{documents[0].ID, documents[1].ID}

	// Act
	err := suite.documentRepo.MarkExpired(suite.ctx, documentIDs)

	// Assert
	require.NoError(suite.T(), err)

	// Проверяем, что статус изменился
	for _, docID := range documentIDs {
		doc, err := suite.documentRepo.GetByID(suite.ctx, docID)
		require.NoError(suite.T(), err)
		assert.Equal(suite.T(), entities.VerificationStatusExpired, doc.Status)
	}
}

// TestDocumentFilters тестирует фильтрацию документов
func (suite *DocumentRepositoryTestSuite) TestDocumentFilters() {
	// Arrange
	// Создаем второго водителя
	driver2 := fixtures.CreateTestDriver()
	driver2.Phone = "+79001234568"
	driver2.Email = "driver2@example.com"
	driver2.LicenseNumber = "TEST123457"
	err := suite.driverRepo.Create(suite.ctx, driver2)
	require.NoError(suite.T(), err)

	// Создаем документы для разных водителей и типов
	documents := []*entities.DriverDocument{
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeMedicalCert),
		fixtures.CreateTestDocument(driver2.ID, entities.DocumentTypeDriverLicense),
		fixtures.CreateTestDocument(driver2.ID, entities.DocumentTypePassport),
	}

	// Устанавливаем разные статусы
	statuses := []entities.VerificationStatus{
		entities.VerificationStatusVerified,
		entities.VerificationStatusPending,
		entities.VerificationStatusVerified,
		entities.VerificationStatusRejected,
	}

	for i, doc := range documents {
		doc.Status = statuses[i]
		err := suite.documentRepo.Create(suite.ctx, doc)
		require.NoError(suite.T(), err)
	}

	// Test 1: Фильтр по водителю
	filters := &entities.DocumentFilters{
		DriverID: &suite.testDriverID,
	}

	result, err := suite.documentRepo.List(suite.ctx, filters)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)

	// Test 2: Фильтр по типу документа
	filters = &entities.DocumentFilters{
		DocumentType: []entities.DocumentType{entities.DocumentTypeDriverLicense},
	}

	result, err = suite.documentRepo.List(suite.ctx, filters)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2) // Два водительских удостоверения

	// Test 3: Фильтр по статусу
	filters = &entities.DocumentFilters{
		Status: []entities.VerificationStatus{entities.VerificationStatusVerified},
	}

	result, err = suite.documentRepo.List(suite.ctx, filters)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2) // Два верифицированных документа
}

// TestDocumentExpiration тестирует логику истечения документов
func (suite *DocumentRepositoryTestSuite) TestDocumentExpiration() {
	// Arrange
	now := time.Now()
	document := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)
	document.ExpiryDate = now.Add(-1 * time.Hour) // Уже истек
	document.Status = entities.VerificationStatusVerified

	err := suite.documentRepo.Create(suite.ctx, document)
	require.NoError(suite.T(), err)

	// Act & Assert - проверяем методы entity
	retrievedDoc, err := suite.documentRepo.GetByID(suite.ctx, document.ID)
	require.NoError(suite.T(), err)

	assert.True(suite.T(), retrievedDoc.IsExpired())
	assert.False(suite.T(), retrievedDoc.IsVerified()) // Не валиден, так как истек
	assert.Negative(suite.T(), retrievedDoc.DaysUntilExpiry())
}

// TestDocumentUpdateWorkflow тестирует workflow обновления документа
func (suite *DocumentRepositoryTestSuite) TestDocumentUpdateWorkflow() {
	// Arrange
	document := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)
	err := suite.documentRepo.Create(suite.ctx, document)
	require.NoError(suite.T(), err)

	// Act 1: Обновляем основные данные
	document.DocumentNumber = "UPDATED-123456"
	document.FileURL = "https://example.com/updated-doc.pdf"

	err = suite.documentRepo.Update(suite.ctx, document)
	require.NoError(suite.T(), err)

	// Assert 1
	updatedDoc, err := suite.documentRepo.GetByID(suite.ctx, document.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "UPDATED-123456", updatedDoc.DocumentNumber)
	assert.Equal(suite.T(), "https://example.com/updated-doc.pdf", updatedDoc.FileURL)

	// Act 2: Верификация
	verifierID := "admin-123"
	err = suite.documentRepo.UpdateStatus(suite.ctx, document.ID, entities.VerificationStatusVerified, &verifierID, nil)
	require.NoError(suite.T(), err)

	// Assert 2
	verifiedDoc, err := suite.documentRepo.GetByID(suite.ctx, document.ID)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), entities.VerificationStatusVerified, verifiedDoc.Status)
	assert.Equal(suite.T(), verifierID, *verifiedDoc.VerifiedBy)
	assert.NotNil(suite.T(), verifiedDoc.VerifiedAt)
}

// TestDocumentCascadeDelete тестирует каскадное удаление документов при удалении водителя
func (suite *DocumentRepositoryTestSuite) TestDocumentCascadeDelete() {
	// Arrange
	documents := []*entities.DriverDocument{
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeMedicalCert),
	}

	for _, doc := range documents {
		err := suite.documentRepo.Create(suite.ctx, doc)
		require.NoError(suite.T(), err)
	}

	// Проверяем, что документы созданы
	driverDocs, err := suite.documentRepo.GetByDriverID(suite.ctx, suite.testDriverID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), driverDocs, 2)

	// Act - удаляем водителя (каскадное удаление документов)
	err = suite.driverRepo.Delete(suite.ctx, suite.testDriverID)
	require.NoError(suite.T(), err)

	// Assert - документы должны быть удалены
	driverDocsAfter, err := suite.documentRepo.GetByDriverID(suite.ctx, suite.testDriverID)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), driverDocsAfter, 0)
}

// TestDocumentCount тестирует подсчет документов
func (suite *DocumentRepositoryTestSuite) TestDocumentCount() {
	// Arrange
	documents := []*entities.DriverDocument{
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeMedicalCert),
		fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypePassport),
	}

	for _, doc := range documents {
		err := suite.documentRepo.Create(suite.ctx, doc)
		require.NoError(suite.T(), err)
	}

	// Act
	count, err := suite.documentRepo.Count(suite.ctx, &entities.DocumentFilters{
		DriverID: &suite.testDriverID,
	})

	// Assert
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), 3, count)
}

// TestDocumentPagination тестирует пагинацию документов
func (suite *DocumentRepositoryTestSuite) TestDocumentPagination() {
	// Arrange - создаем много документов разных типов
	docTypes := []entities.DocumentType{
		entities.DocumentTypeDriverLicense,
		entities.DocumentTypeMedicalCert,
		entities.DocumentTypePassport,
		entities.DocumentTypeInsurance,
		entities.DocumentTypeTaxiPermit,
	}

	for _, docType := range docTypes {
		doc := fixtures.CreateTestDocument(suite.testDriverID, docType)
		err := suite.documentRepo.Create(suite.ctx, doc)
		require.NoError(suite.T(), err)
	}

	// Act - первая страница
	filters := &entities.DocumentFilters{
		DriverID: &suite.testDriverID,
		Limit:    3,
		Offset:   0,
	}

	page1, err := suite.documentRepo.List(suite.ctx, filters)
	require.NoError(suite.T(), err)

	// Act - вторая страница
	filters.Offset = 3
	page2, err := suite.documentRepo.List(suite.ctx, filters)
	require.NoError(suite.T(), err)

	// Assert
	assert.Len(suite.T(), page1, 3)
	assert.Len(suite.T(), page2, 2)

	// Проверяем, что документы не дублируются между страницами
	page1IDs := make(map[uuid.UUID]bool)
	for _, doc := range page1 {
		page1IDs[doc.ID] = true
	}

	for _, doc := range page2 {
		assert.False(suite.T(), page1IDs[doc.ID], "Document should not appear on both pages")
	}
}

// TestDocumentValidation тестирует валидацию документов на уровне базы данных
func (suite *DocumentRepositoryTestSuite) TestDocumentValidation() {
	// Test 1: Невалидные даты (expiry_date <= issue_date)
	document := fixtures.CreateTestDocument(suite.testDriverID, entities.DocumentTypeDriverLicense)
	document.ExpiryDate = document.IssueDate.Add(-1 * time.Hour) // Дата истечения раньше даты выдачи

	err := suite.documentRepo.Create(suite.ctx, document)
	assert.Error(suite.T(), err) // Должна быть ошибка constraint'а

	// Test 2: Невалидный тип документа (на уровне БД это проверяется constraint'ом)
	// Этот тест зависит от того, как именно реализована валидация в БД
}

// Запуск тестового suite
func TestDocumentRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(DocumentRepositoryTestSuite))
}
