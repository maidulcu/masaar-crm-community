package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type BankIntegrationRepo struct {
	db *pgxpool.Pool
}

func NewBankIntegrationRepo(db *pgxpool.Pool) *BankIntegrationRepo {
	return &BankIntegrationRepo{db: db}
}

func (r *BankIntegrationRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.BankIntegration], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM bank_integrations WHERE company_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count integrations: %w", err)
	}

	const q = `
		SELECT id, company_id, bank_name, bank_code, account_number, account_name, iban,
		       integration_type, status, api_endpoint, auto_sync, last_sync_date,
		       sync_interval_hours, last_sync_error, sync_error_count, is_connected,
		       connection_test_date, created_at, updated_at, created_by, updated_by
		FROM bank_integrations
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}
	defer rows.Close()

	var integrations []domain.BankIntegration
	for rows.Next() {
		var bi domain.BankIntegration
		if err := rows.Scan(
			&bi.ID, &bi.CompanyID, &bi.BankName, &bi.BankCode, &bi.AccountNumber, &bi.AccountName, &bi.IBAN,
			&bi.IntegrationType, &bi.Status, &bi.APIEndpoint, &bi.AutoSync, &bi.LastSyncDate,
			&bi.SyncIntervalHours, &bi.LastSyncError, &bi.SyncErrorCount, &bi.IsConnected,
			&bi.ConnectionTestDate, &bi.CreatedAt, &bi.UpdatedAt, &bi.CreatedBy, &bi.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan integration: %w", err)
		}
		integrations = append(integrations, bi)
	}

	return &domain.PaginatedResult[domain.BankIntegration]{
		Data:  integrations,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *BankIntegrationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.BankIntegration, error) {
	const q = `
		SELECT id, company_id, bank_name, bank_code, account_number, account_name, iban,
		       integration_type, status, api_key_encrypted, api_secret_encrypted, api_endpoint,
		       auto_sync, last_sync_date, sync_interval_hours, last_sync_error, sync_error_count,
		       is_connected, connection_test_date, created_at, updated_at, created_by, updated_by
		FROM bank_integrations WHERE id = $1
	`
	bi := &domain.BankIntegration{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&bi.ID, &bi.CompanyID, &bi.BankName, &bi.BankCode, &bi.AccountNumber, &bi.AccountName, &bi.IBAN,
		&bi.IntegrationType, &bi.Status, &bi.APIKeyEncrypted, &bi.APISecretEncrypted, &bi.APIEndpoint,
		&bi.AutoSync, &bi.LastSyncDate, &bi.SyncIntervalHours, &bi.LastSyncError, &bi.SyncErrorCount,
		&bi.IsConnected, &bi.ConnectionTestDate, &bi.CreatedAt, &bi.UpdatedAt, &bi.CreatedBy, &bi.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get integration by id: %w", err)
	}
	return bi, nil
}

func (r *BankIntegrationRepo) Create(ctx context.Context, bi *domain.BankIntegration) error {
	const q = `
		INSERT INTO bank_integrations (
			id, company_id, bank_name, bank_code, account_number, account_name, iban,
			integration_type, status, api_key_encrypted, api_secret_encrypted, api_endpoint,
			auto_sync, sync_interval_hours, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		RETURNING created_at, updated_at
	`
	bi.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		bi.ID, bi.CompanyID, bi.BankName, bi.BankCode, bi.AccountNumber, bi.AccountName, bi.IBAN,
		bi.IntegrationType, bi.Status, bi.APIKeyEncrypted, bi.APISecretEncrypted, bi.APIEndpoint,
		bi.AutoSync, bi.SyncIntervalHours, bi.CreatedBy, bi.UpdatedBy,
	).Scan(&bi.CreatedAt, &bi.UpdatedAt)
}

func (r *BankIntegrationRepo) Update(ctx context.Context, bi *domain.BankIntegration) error {
	const q = `
		UPDATE bank_integrations
		SET bank_name=$1, bank_code=$2, account_number=$3, account_name=$4, iban=$5,
		    integration_type=$6, status=$7, api_key_encrypted=$8, api_secret_encrypted=$9,
		    api_endpoint=$10, auto_sync=$11, sync_interval_hours=$12, is_connected=$13,
		    connection_test_date=$14, updated_by=$15, updated_at=NOW()
		WHERE id=$16
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, q,
		bi.BankName, bi.BankCode, bi.AccountNumber, bi.AccountName, bi.IBAN,
		bi.IntegrationType, bi.Status, bi.APIKeyEncrypted, bi.APISecretEncrypted,
		bi.APIEndpoint, bi.AutoSync, bi.SyncIntervalHours, bi.IsConnected,
		bi.ConnectionTestDate, bi.UpdatedBy, bi.ID,
	).Scan(&bi.UpdatedAt)
}

func (r *BankIntegrationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM bank_integrations WHERE id=$1`, id)
	return err
}

func (r *BankIntegrationRepo) UpdateSyncStatus(ctx context.Context, id uuid.UUID, syncDate time.Time, isError bool, errorMsg string) error {
	query := `
		UPDATE bank_integrations
		SET last_sync_date=$1, sync_error_count=CASE WHEN $3 THEN sync_error_count+1 ELSE 0 END,
		    last_sync_error=$4, updated_at=NOW()
		WHERE id=$2
	`
	_, err := r.db.Exec(ctx, query, syncDate, id, isError, errorMsg)
	return err
}

func (r *BankIntegrationRepo) GetAutoSyncEnabled(ctx context.Context, companyID uuid.UUID) ([]domain.BankIntegration, error) {
	const q = `
		SELECT id, company_id, bank_name, bank_code, account_number, account_name, iban,
		       integration_type, status, api_key_encrypted, api_secret_encrypted, api_endpoint,
		       auto_sync, last_sync_date, sync_interval_hours, last_sync_error, sync_error_count,
		       is_connected, connection_test_date, created_at, updated_at, created_by, updated_by
		FROM bank_integrations
		WHERE company_id = $1 AND auto_sync = TRUE AND status = 'active'
		ORDER BY last_sync_date ASC
	`
	rows, err := r.db.Query(ctx, q, companyID)
	if err != nil {
		return nil, fmt.Errorf("get auto sync enabled: %w", err)
	}
	defer rows.Close()

	var integrations []domain.BankIntegration
	for rows.Next() {
		var bi domain.BankIntegration
		if err := rows.Scan(
			&bi.ID, &bi.CompanyID, &bi.BankName, &bi.BankCode, &bi.AccountNumber, &bi.AccountName, &bi.IBAN,
			&bi.IntegrationType, &bi.Status, &bi.APIKeyEncrypted, &bi.APISecretEncrypted, &bi.APIEndpoint,
			&bi.AutoSync, &bi.LastSyncDate, &bi.SyncIntervalHours, &bi.LastSyncError, &bi.SyncErrorCount,
			&bi.IsConnected, &bi.ConnectionTestDate, &bi.CreatedAt, &bi.UpdatedAt, &bi.CreatedBy, &bi.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan integration: %w", err)
		}
		integrations = append(integrations, bi)
	}
	return integrations, nil
}
