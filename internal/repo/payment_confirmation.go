package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type PaymentConfirmationRepo struct {
	db *pgxpool.Pool
}

func NewPaymentConfirmationRepo(db *pgxpool.Pool) *PaymentConfirmationRepo {
	return &PaymentConfirmationRepo{db: db}
}

func (r *PaymentConfirmationRepo) Create(ctx context.Context, pc *domain.PaymentConfirmation) error {
	const q = `
		INSERT INTO payment_confirmations (
			id, company_id, payment_id, confirmation_number, tenant_email, tenant_phone,
			delivery_method, data_classification, retention_until
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`
	pc.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		pc.ID, pc.CompanyID, pc.PaymentID, pc.ConfirmationNumber, pc.TenantEmail, pc.TenantPhone,
		pc.DeliveryMethod, pc.DataClassification, pc.RetentionUntil,
	).Scan(&pc.CreatedAt, &pc.UpdatedAt)
}

func (r *PaymentConfirmationRepo) GetByPaymentID(ctx context.Context, paymentID uuid.UUID) (*domain.PaymentConfirmation, error) {
	const q = `
		SELECT id, company_id, payment_id, confirmation_number, tenant_email, tenant_phone,
		       sent_at, delivery_status, delivery_method, pdf_url, data_classification,
		       retention_until, created_at, updated_at, deleted_at
		FROM payment_confirmations
		WHERE payment_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`
	pc := &domain.PaymentConfirmation{}
	err := r.db.QueryRow(ctx, q, paymentID).Scan(
		&pc.ID, &pc.CompanyID, &pc.PaymentID, &pc.ConfirmationNumber, &pc.TenantEmail, &pc.TenantPhone,
		&pc.SentAt, &pc.DeliveryStatus, &pc.DeliveryMethod, &pc.PDFURL, &pc.DataClassification,
		&pc.RetentionUntil, &pc.CreatedAt, &pc.UpdatedAt, &pc.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get payment confirmation: %w", err)
	}
	return pc, nil
}

func (r *PaymentConfirmationRepo) GetPending(ctx context.Context, companyID uuid.UUID) ([]domain.PaymentConfirmation, error) {
	const q = `
		SELECT id, company_id, payment_id, confirmation_number, tenant_email, tenant_phone,
		       sent_at, delivery_status, delivery_method, pdf_url, data_classification,
		       retention_until, created_at, updated_at, deleted_at
		FROM payment_confirmations
		WHERE company_id = $1 AND delivery_status = 'pending' AND deleted_at IS NULL
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, q, companyID)
	if err != nil {
		return nil, fmt.Errorf("get pending confirmations: %w", err)
	}
	defer rows.Close()

	var confirmations []domain.PaymentConfirmation
	for rows.Next() {
		var pc domain.PaymentConfirmation
		if err := rows.Scan(
			&pc.ID, &pc.CompanyID, &pc.PaymentID, &pc.ConfirmationNumber, &pc.TenantEmail, &pc.TenantPhone,
			&pc.SentAt, &pc.DeliveryStatus, &pc.DeliveryMethod, &pc.PDFURL, &pc.DataClassification,
			&pc.RetentionUntil, &pc.CreatedAt, &pc.UpdatedAt, &pc.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan confirmation: %w", err)
		}
		confirmations = append(confirmations, pc)
	}
	return confirmations, nil
}

func (r *PaymentConfirmationRepo) MarkSent(ctx context.Context, id uuid.UUID, pdfURL *string) error {
	const q = `
		UPDATE payment_confirmations
		SET delivery_status = 'sent', sent_at = NOW(), pdf_url = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING updated_at
	`
	var updated time.Time
	return r.db.QueryRow(ctx, q, pdfURL, id).Scan(&updated)
}

func (r *PaymentConfirmationRepo) MarkFailed(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE payment_confirmations
		SET delivery_status = 'failed', updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	var updated time.Time
	return r.db.QueryRow(ctx, q, id).Scan(&updated)
}
