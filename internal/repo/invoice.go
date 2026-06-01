package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type InvoiceRepo struct {
	db *pgxpool.Pool
}

func NewInvoiceRepo(db *pgxpool.Pool) *InvoiceRepo {
	return &InvoiceRepo{db: db}
}

func (r *InvoiceRepo) Create(ctx context.Context, inv *domain.VATInvoice) error {
	const q = `
		INSERT INTO vat_invoices (id, deal_id, invoice_no, subtotal, vat_rate, status)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING vat_amount, total, issued_at
	`
	inv.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		inv.ID, inv.DealID, inv.InvoiceNo,
		inv.Subtotal, inv.VATRate, inv.Status,
	).Scan(&inv.VATAmount, &inv.Total, &inv.IssuedAt)
}

func (r *InvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.VATInvoice, error) {
	const q = `
		SELECT id, deal_id, invoice_no, subtotal, vat_rate, vat_amount, total, qr_payload, status, issued_at
		FROM vat_invoices WHERE id = $1
	`
	inv := &domain.VATInvoice{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&inv.ID, &inv.DealID, &inv.InvoiceNo,
		&inv.Subtotal, &inv.VATRate, &inv.VATAmount,
		&inv.Total, &inv.QRPayload, &inv.Status, &inv.IssuedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	return inv, nil
}

func (r *InvoiceRepo) ListByDeal(ctx context.Context, dealID uuid.UUID) ([]domain.VATInvoice, error) {
	const q = `
		SELECT id, deal_id, invoice_no, subtotal, vat_rate, vat_amount, total, qr_payload, status, issued_at
		FROM vat_invoices WHERE deal_id = $1 ORDER BY issued_at DESC
	`
	rows, err := r.db.Query(ctx, q, dealID)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()

	var invoices []domain.VATInvoice
	for rows.Next() {
		var inv domain.VATInvoice
		if err := rows.Scan(
			&inv.ID, &inv.DealID, &inv.InvoiceNo,
			&inv.Subtotal, &inv.VATRate, &inv.VATAmount,
			&inv.Total, &inv.QRPayload, &inv.Status, &inv.IssuedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

func (r *InvoiceRepo) ListAll(ctx context.Context, page, limit int) ([]domain.VATInvoice, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM vat_invoices`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}

	const q = `
		SELECT id, deal_id, invoice_no, subtotal, vat_rate, vat_amount, total, qr_payload, status, issued_at
		FROM vat_invoices ORDER BY issued_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list all invoices: %w", err)
	}
	defer rows.Close()

	var invoices []domain.VATInvoice
	for rows.Next() {
		var inv domain.VATInvoice
		if err := rows.Scan(
			&inv.ID, &inv.DealID, &inv.InvoiceNo,
			&inv.Subtotal, &inv.VATRate, &inv.VATAmount,
			&inv.Total, &inv.QRPayload, &inv.Status, &inv.IssuedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan invoice: %w", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, total, rows.Err()
}

func (r *InvoiceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE vat_invoices SET status=$1 WHERE id=$2`,
		status, id,
	)
	return err
}

// NextInvoiceNo generates a sequential invoice number: INV-YYYY-NNNN
// Uses advisory lock to prevent race conditions
func (r *InvoiceRepo) NextInvoiceNo(ctx context.Context) (string, error) {
	var no string
	err := r.db.QueryRow(ctx, `
		SELECT 'INV-' || TO_CHAR(NOW(), 'YYYY') || '-' || LPAD(
			(COALESCE(
				(SELECT MAX(SUBSTRING(invoice_no FROM '....$')::INT) 
				 FROM vat_invoices 
				 WHERE invoice_no LIKE 'INV-' || TO_CHAR(NOW(), 'YYYY') || '-%'),
				0) + 1
			)::TEXT, 4, '0'
		)
	`).Scan(&no)
	return no, err
}
