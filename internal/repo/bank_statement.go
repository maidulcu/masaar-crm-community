package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type BankStatementRepo struct {
	db *pgxpool.Pool
}

func NewBankStatementRepo(db *pgxpool.Pool) *BankStatementRepo {
	return &BankStatementRepo{db: db}
}

func (r *BankStatementRepo) Create(ctx context.Context, bs *domain.BankStatement) error {
	const q = `
		INSERT INTO bank_statements (
			id, company_id, bank_integration_id, file_name, file_size_bytes, file_url,
			file_format, uploaded_by, upload_date, processing_status, data_classification, retention_until
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`
	bs.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		bs.ID, bs.CompanyID, bs.BankIntegrationID, bs.FileName, bs.FileSizeBytes, bs.FileURL,
		bs.FileFormat, bs.UploadedBy, bs.UploadDate, bs.ProcessingStatus, bs.DataClassification, bs.RetentionUntil,
	).Scan(&bs.CreatedAt, &bs.UpdatedAt)
}

func (r *BankStatementRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.BankStatement, error) {
	const q = `
		SELECT id, company_id, bank_integration_id, file_name, file_size_bytes, file_url,
		       file_format, uploaded_by, upload_date, processing_status, transactions_imported,
		       import_error, data_classification, retention_until, created_at, updated_at, deleted_at
		FROM bank_statements WHERE id = $1 AND deleted_at IS NULL
	`
	bs := &domain.BankStatement{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&bs.ID, &bs.CompanyID, &bs.BankIntegrationID, &bs.FileName, &bs.FileSizeBytes, &bs.FileURL,
		&bs.FileFormat, &bs.UploadedBy, &bs.UploadDate, &bs.ProcessingStatus, &bs.TransactionsImported,
		&bs.ImportError, &bs.DataClassification, &bs.RetentionUntil, &bs.CreatedAt, &bs.UpdatedAt, &bs.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get bank statement: %w", err)
	}
	return bs, nil
}

func (r *BankStatementRepo) ListByCompany(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.BankStatement], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM bank_statements WHERE company_id = $1 AND deleted_at IS NULL`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count bank statements: %w", err)
	}

	const q = `
		SELECT id, company_id, bank_integration_id, file_name, file_size_bytes, file_url,
		       file_format, uploaded_by, upload_date, processing_status, transactions_imported,
		       import_error, data_classification, retention_until, created_at, updated_at, deleted_at
		FROM bank_statements
		WHERE company_id = $1 AND deleted_at IS NULL
		ORDER BY upload_date DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list bank statements: %w", err)
	}
	defer rows.Close()

	var statements []domain.BankStatement
	for rows.Next() {
		var bs domain.BankStatement
		if err := rows.Scan(
			&bs.ID, &bs.CompanyID, &bs.BankIntegrationID, &bs.FileName, &bs.FileSizeBytes, &bs.FileURL,
			&bs.FileFormat, &bs.UploadedBy, &bs.UploadDate, &bs.ProcessingStatus, &bs.TransactionsImported,
			&bs.ImportError, &bs.DataClassification, &bs.RetentionUntil, &bs.CreatedAt, &bs.UpdatedAt, &bs.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan bank statement: %w", err)
		}
		statements = append(statements, bs)
	}

	return &domain.PaginatedResult[domain.BankStatement]{
		Data:  statements,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *BankStatementRepo) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status domain.ProcessingStatus, transactionCount int, errMsg string) error {
	const q = `
		UPDATE bank_statements
		SET processing_status = $1, transactions_imported = $2, import_error = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`
	var updated time.Time
	return r.db.QueryRow(ctx, q, status, transactionCount, errMsg, id).Scan(&updated)
}

func (r *BankStatementRepo) GetPendingForProcessing(ctx context.Context, companyID uuid.UUID) ([]domain.BankStatement, error) {
	const q = `
		SELECT id, company_id, bank_integration_id, file_name, file_size_bytes, file_url,
		       file_format, uploaded_by, upload_date, processing_status, transactions_imported,
		       import_error, data_classification, retention_until, created_at, updated_at, deleted_at
		FROM bank_statements
		WHERE company_id = $1 AND processing_status = 'pending' AND deleted_at IS NULL
		ORDER BY upload_date ASC
	`
	rows, err := r.db.Query(ctx, q, companyID)
	if err != nil {
		return nil, fmt.Errorf("get pending statements: %w", err)
	}
	defer rows.Close()

	var statements []domain.BankStatement
	for rows.Next() {
		var bs domain.BankStatement
		if err := rows.Scan(
			&bs.ID, &bs.CompanyID, &bs.BankIntegrationID, &bs.FileName, &bs.FileSizeBytes, &bs.FileURL,
			&bs.FileFormat, &bs.UploadedBy, &bs.UploadDate, &bs.ProcessingStatus, &bs.TransactionsImported,
			&bs.ImportError, &bs.DataClassification, &bs.RetentionUntil, &bs.CreatedAt, &bs.UpdatedAt, &bs.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan bank statement: %w", err)
		}
		statements = append(statements, bs)
	}
	return statements, nil
}

func (r *BankStatementRepo) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE bank_statements SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}
