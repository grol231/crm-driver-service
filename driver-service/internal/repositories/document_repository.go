package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"driver-service/internal/domain/entities"
	"driver-service/internal/infrastructure/database"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DocumentRepository интерфейс для работы с документами водителей
type DocumentRepository interface {
	Create(ctx context.Context, document *entities.DriverDocument) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverDocument, error)
	GetByDriverID(ctx context.Context, driverID uuid.UUID) ([]*entities.DriverDocument, error)
	GetByDriverIDAndType(ctx context.Context, driverID uuid.UUID, docType entities.DocumentType) (*entities.DriverDocument, error)
	Update(ctx context.Context, document *entities.DriverDocument) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filters *entities.DocumentFilters) ([]*entities.DriverDocument, error)
	Count(ctx context.Context, filters *entities.DocumentFilters) (int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.VerificationStatus, verifierID, reason *string) error
	GetExpiring(ctx context.Context, days int) ([]*entities.DriverDocument, error)
	GetExpired(ctx context.Context) ([]*entities.DriverDocument, error)
	MarkExpired(ctx context.Context, documentIDs []uuid.UUID) error
}

// documentRepository реализация DocumentRepository
type documentRepository struct {
	db     *database.DB
	logger *zap.Logger
}

// NewDocumentRepository создает новый репозиторий документов
func NewDocumentRepository(db *database.DB, logger *zap.Logger) DocumentRepository {
	return &documentRepository{
		db:     db,
		logger: logger,
	}
}

// Create создает новый документ водителя
func (r *documentRepository) Create(ctx context.Context, document *entities.DriverDocument) error {
	query := `
		INSERT INTO driver_documents (
			id, driver_id, document_type, document_number, issue_date,
			expiry_date, file_url, status, metadata, created_at, updated_at
		) VALUES (
			:id, :driver_id, :document_type, :document_number, :issue_date,
			:expiry_date, :file_url, :status, :metadata, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, document)
	if err != nil {
		r.logger.Error("Failed to create document",
			zap.Error(err),
			zap.String("document_id", document.ID.String()),
			zap.String("driver_id", document.DriverID.String()),
			zap.String("document_type", string(document.DocumentType)),
		)
		return fmt.Errorf("failed to create document: %w", err)
	}

	r.logger.Info("Document created successfully",
		zap.String("document_id", document.ID.String()),
		zap.String("driver_id", document.DriverID.String()),
		zap.String("document_type", string(document.DocumentType)),
	)

	return nil
}

// GetByID получает документ по ID
func (r *documentRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.DriverDocument, error) {
	var document entities.DriverDocument
	query := `SELECT * FROM driver_documents WHERE id = $1`

	err := r.db.GetContext(ctx, &document, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrDocumentNotFound
		}
		r.logger.Error("Failed to get document by ID",
			zap.Error(err),
			zap.String("document_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get document by ID: %w", err)
	}

	return &document, nil
}

// GetByDriverID получает все документы водителя
func (r *documentRepository) GetByDriverID(ctx context.Context, driverID uuid.UUID) ([]*entities.DriverDocument, error) {
	var documents []*entities.DriverDocument
	query := `
		SELECT * FROM driver_documents 
		WHERE driver_id = $1 
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &documents, query, driverID)
	if err != nil {
		r.logger.Error("Failed to get documents by driver ID",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
		)
		return nil, fmt.Errorf("failed to get documents by driver ID: %w", err)
	}

	return documents, nil
}

// GetByDriverIDAndType получает документ водителя определенного типа
func (r *documentRepository) GetByDriverIDAndType(ctx context.Context, driverID uuid.UUID, docType entities.DocumentType) (*entities.DriverDocument, error) {
	var document entities.DriverDocument
	query := `
		SELECT * FROM driver_documents 
		WHERE driver_id = $1 AND document_type = $2
		ORDER BY created_at DESC
		LIMIT 1`

	err := r.db.GetContext(ctx, &document, query, driverID, docType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entities.ErrDocumentNotFound
		}
		r.logger.Error("Failed to get document by driver ID and type",
			zap.Error(err),
			zap.String("driver_id", driverID.String()),
			zap.String("document_type", string(docType)),
		)
		return nil, fmt.Errorf("failed to get document by driver ID and type: %w", err)
	}

	return &document, nil
}

// Update обновляет документ
func (r *documentRepository) Update(ctx context.Context, document *entities.DriverDocument) error {
	document.UpdatedAt = time.Now()

	query := `
		UPDATE driver_documents SET
			document_number = :document_number, issue_date = :issue_date,
			expiry_date = :expiry_date, file_url = :file_url, status = :status,
			verified_by = :verified_by, verified_at = :verified_at,
			rejection_reason = :rejection_reason, metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, document)
	if err != nil {
		r.logger.Error("Failed to update document",
			zap.Error(err),
			zap.String("document_id", document.ID.String()),
		)
		return fmt.Errorf("failed to update document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDocumentNotFound
	}

	r.logger.Info("Document updated successfully",
		zap.String("document_id", document.ID.String()),
	)

	return nil
}

// Delete удаляет документ
func (r *documentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM driver_documents WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete document",
			zap.Error(err),
			zap.String("document_id", id.String()),
		)
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDocumentNotFound
	}

	r.logger.Info("Document deleted successfully",
		zap.String("document_id", id.String()),
	)

	return nil
}

// List получает список документов с фильтрами
func (r *documentRepository) List(ctx context.Context, filters *entities.DocumentFilters) ([]*entities.DriverDocument, error) {
	query, args, err := r.buildListQuery(filters, false)
	if err != nil {
		return nil, fmt.Errorf("failed to build list query: %w", err)
	}

	var documents []*entities.DriverDocument
	err = r.db.SelectContext(ctx, &documents, query, args...)
	if err != nil {
		r.logger.Error("Failed to list documents",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return documents, nil
}

// Count возвращает количество документов с фильтрами
func (r *documentRepository) Count(ctx context.Context, filters *entities.DocumentFilters) (int, error) {
	query, args, err := r.buildListQuery(filters, true)
	if err != nil {
		return 0, fmt.Errorf("failed to build count query: %w", err)
	}

	var count int
	err = r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		r.logger.Error("Failed to count documents",
			zap.Error(err),
		)
		return 0, fmt.Errorf("failed to count documents: %w", err)
	}

	return count, nil
}

// UpdateStatus обновляет статус документа
func (r *documentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.VerificationStatus, verifierID, reason *string) error {
	now := time.Now()
	query := `
		UPDATE driver_documents SET
			status = $1, verified_by = $2, verified_at = $3,
			rejection_reason = $4, updated_at = $5
		WHERE id = $6`

	result, err := r.db.ExecContext(ctx, query, status, verifierID, &now, reason, now, id)
	if err != nil {
		r.logger.Error("Failed to update document status",
			zap.Error(err),
			zap.String("document_id", id.String()),
			zap.String("status", string(status)),
		)
		return fmt.Errorf("failed to update document status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return entities.ErrDocumentNotFound
	}

	r.logger.Info("Document status updated successfully",
		zap.String("document_id", id.String()),
		zap.String("status", string(status)),
	)

	return nil
}

// GetExpiring получает документы, истекающие через указанное количество дней
func (r *documentRepository) GetExpiring(ctx context.Context, days int) ([]*entities.DriverDocument, error) {
	query := `
		SELECT * FROM driver_documents
		WHERE expiry_date <= $1
		AND expiry_date > NOW()
		AND status = 'verified'
		ORDER BY expiry_date ASC`

	expiryDate := time.Now().AddDate(0, 0, days)
	var documents []*entities.DriverDocument
	
	err := r.db.SelectContext(ctx, &documents, query, expiryDate)
	if err != nil {
		r.logger.Error("Failed to get expiring documents",
			zap.Error(err),
			zap.Int("days", days),
		)
		return nil, fmt.Errorf("failed to get expiring documents: %w", err)
	}

	return documents, nil
}

// GetExpired получает истекшие документы
func (r *documentRepository) GetExpired(ctx context.Context) ([]*entities.DriverDocument, error) {
	query := `
		SELECT * FROM driver_documents
		WHERE expiry_date < NOW()
		AND status IN ('verified', 'pending')
		ORDER BY expiry_date DESC`

	var documents []*entities.DriverDocument
	err := r.db.SelectContext(ctx, &documents, query)
	if err != nil {
		r.logger.Error("Failed to get expired documents",
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get expired documents: %w", err)
	}

	return documents, nil
}

// MarkExpired помечает документы как истекшие
func (r *documentRepository) MarkExpired(ctx context.Context, documentIDs []uuid.UUID) error {
	if len(documentIDs) == 0 {
		return nil
	}

	// Создаем плейсхолдеры для IN запроса
	placeholders := make([]string, len(documentIDs))
	args := make([]interface{}, len(documentIDs)+1)
	args[0] = time.Now() // updated_at

	for i, id := range documentIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		UPDATE driver_documents 
		SET status = 'expired', updated_at = $1
		WHERE id IN (%s)`, strings.Join(placeholders, ","))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to mark documents as expired",
			zap.Error(err),
			zap.Int("document_count", len(documentIDs)),
		)
		return fmt.Errorf("failed to mark documents as expired: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info("Documents marked as expired",
		zap.Int64("rows_affected", rowsAffected),
		zap.Int("document_count", len(documentIDs)),
	)

	return nil
}

// buildListQuery строит SQL запрос для получения списка документов
func (r *documentRepository) buildListQuery(filters *entities.DocumentFilters, isCount bool) (string, []interface{}, error) {
	var conditions []string
	var args []interface{}
	argCount := 0

	// Базовый запрос
	baseQuery := "FROM driver_documents WHERE 1=1"
	
	var selectClause string
	if isCount {
		selectClause = "SELECT COUNT(*) "
	} else {
		selectClause = "SELECT * "
	}

	// Добавляем условия фильтрации
	if filters != nil {
		if filters.DriverID != nil {
			argCount++
			conditions = append(conditions, fmt.Sprintf("driver_id = $%d", argCount))
			args = append(args, *filters.DriverID)
		}

		if len(filters.DocumentType) > 0 {
			placeholders := make([]string, len(filters.DocumentType))
			for i, docType := range filters.DocumentType {
				argCount++
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, docType)
			}
			conditions = append(conditions, fmt.Sprintf("document_type IN (%s)", strings.Join(placeholders, ",")))
		}

		if len(filters.Status) > 0 {
			placeholders := make([]string, len(filters.Status))
			for i, status := range filters.Status {
				argCount++
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, status)
			}
			conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
		}

		if filters.ExpiringIn != nil {
			argCount++
			conditions = append(conditions, fmt.Sprintf("expiry_date <= NOW() + INTERVAL '%d days'", *filters.ExpiringIn))
			conditions = append(conditions, "expiry_date > NOW()")
		}

		if filters.Expired != nil && *filters.Expired {
			conditions = append(conditions, "expiry_date < NOW()")
		}
	}

	// Собираем WHERE условия
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	query := selectClause + baseQuery + whereClause

	// Для запросов списка добавляем сортировку и пагинацию
	if !isCount && filters != nil {
		query += " ORDER BY created_at DESC"

		if filters.Limit > 0 {
			argCount++
			query += fmt.Sprintf(" LIMIT $%d", argCount)
			args = append(args, filters.Limit)
		}

		if filters.Offset > 0 {
			argCount++
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, filters.Offset)
		}
	}

	return query, args, nil
}