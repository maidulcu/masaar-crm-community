package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type InvoiceRepo struct {
	db *pgxpool.Pool
}

func NewInvoiceRepo(db *pgxpool.Pool) *InvoiceRepo {
	return &InvoiceRepo{db: db}
}

func (r *InvoiceRepo) Create(ctx context.Context, inv *domain.VATInvoice) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// The deal must belong to the caller's company.
	const q = `
		INSERT INTO vat_invoices (id, company_id, deal_id, invoice_no, subtotal, vat_rate, status)
		SELECT $1,$2,$3,$4,$5,$6,$7
		WHERE EXISTS (SELECT 1 FROM deals WHERE id = $3 AND company_id = $2)
		RETURNING vat_amount, total, issued_at
	`
	inv.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		inv.ID, cid, inv.DealID, inv.InvoiceNo,
		inv.Subtotal, inv.VATRate, inv.Status,
	).Scan(&inv.VATAmount, &inv.Total, &inv.IssuedAt)
}

func (r *InvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.VATInvoice, error) {
	const q = `
		SELECT id, deal_id, invoice_no, subtotal, vat_rate, vat_amount, total, COALESCE(qr_payload, ''), status, issued_at
		FROM vat_invoices WHERE id = $1 AND company_id = $2
	`
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	inv := &domain.VATInvoice{}
	err = r.db.QueryRow(ctx, q, id, cid).Scan(
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
		SELECT id, deal_id, invoice_no, subtotal, vat_rate, vat_amount, total, COALESCE(qr_payload, ''), status, issued_at
		FROM vat_invoices WHERE deal_id = $1 AND company_id = $2 ORDER BY issued_at DESC
	`
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, q, dealID, cid)
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

func (r *InvoiceRepo) ListAll(ctx context.Context, page, limit int, status domain.InvoiceStatus) ([]domain.VATInvoice, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM vat_invoices WHERE company_id = $1 AND ($2 = '' OR status = $2)`, cid, string(status)).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}

	const q = `
		SELECT id, deal_id, invoice_no, subtotal, vat_rate, vat_amount, total, COALESCE(qr_payload, ''), status, issued_at
		FROM vat_invoices WHERE company_id = $3 AND ($4 = '' OR status = $4) ORDER BY issued_at DESC, id LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, q, limit, offset, cid, string(status))
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
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// A sent or paid invoice is a tax document that has left the building; it cannot become a draft
	// again (it could then be "sent" a second time, or quietly vanish from the books).
	tag, err := r.db.Exec(ctx,
		`UPDATE vat_invoices SET status=$1 WHERE id=$2 AND company_id=$3 AND ($1 <> 'draft' OR status = 'draft')`,
		status, id, cid,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vat_invoices WHERE id = $1 AND company_id = $2)`, id, cid).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrInvoiceNotFound
		}
		return ErrInvoiceIssued
	}
	return nil
}

// ErrInvoiceNotFound is returned when an invoice does not exist in the caller's company. It wraps
// pgx.ErrNoRows so handlers answer 404.
var ErrInvoiceNotFound = fmt.Errorf("invoice not found: %w", pgx.ErrNoRows)

// ErrInvoiceIssued is returned by UpdateStatus when a sent or paid invoice would go back to draft.
var ErrInvoiceIssued = errors.New("invoice has already been issued")

// ErrInvoiceNotDraft is returned by MarkSent when the invoice has already left the draft state.
var ErrInvoiceNotDraft = errors.New("invoice is not a draft")

// nextInvoiceNoSQL computes INV-YYYY-NNNN from the highest existing number of the year. It reads
// the trailing digit run (not the last four characters, which wrapped once a year passed 9,999
// invoices and then handed out duplicates) and ignores numbers it cannot parse.
const nextInvoiceNoSQL = `
	SELECT 'INV-' || TO_CHAR(NOW(), 'YYYY') || '-' ||
	       CASE WHEN n >= 10000 THEN n::TEXT ELSE LPAD(n::TEXT, 4, '0') END -- LPAD would cut 10000 to '1000'
	FROM (
		SELECT COALESCE(
			(SELECT MAX(SUBSTRING(invoice_no FROM '(\d+)$')::BIGINT)
			 FROM vat_invoices
			 WHERE company_id = $1 AND invoice_no LIKE 'INV-' || TO_CHAR(NOW(), 'YYYY') || '-%'
			   AND invoice_no ~ '-\d+$'),
			0) + 1 AS n
	) next`

// invoiceLockKey prefixes the per-company advisory lock that serialises invoice numbering.
const invoiceLockKey = "invoice-no:"

// NextInvoiceNo returns the number the next invoice would get. It is only a preview: concurrent
// callers can see the same value. Use CreateNumbered to create an invoice.
func (r *InvoiceRepo) NextInvoiceNo(ctx context.Context) (string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", err
	}
	var no string
	err = r.db.QueryRow(ctx, nextInvoiceNoSQL, cid).Scan(&no)
	return no, err
}

// CreateNumbered assigns the next invoice number and inserts the invoice in one transaction
// holding a per-company advisory lock. Numbering and insert used to be separate steps, so two
// invoices created at the same moment got the same number and one of them failed.
func (r *InvoiceRepo) CreateNumbered(ctx context.Context, inv *domain.VATInvoice) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, invoiceLockKey+cid.String()); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, nextInvoiceNoSQL, cid).Scan(&inv.InvoiceNo); err != nil {
		return err
	}
	inv.ID = uuid.New()
	err = tx.QueryRow(ctx, `
		INSERT INTO vat_invoices (id, company_id, deal_id, invoice_no, subtotal, vat_rate, status)
		SELECT $1,$2,$3,$4,$5,$6,$7
		WHERE EXISTS (SELECT 1 FROM deals WHERE id = $3 AND company_id = $2)
		RETURNING vat_amount, total, issued_at`,
		inv.ID, cid, inv.DealID, inv.InvoiceNo, inv.Subtotal, inv.VATRate, inv.Status,
	).Scan(&inv.VATAmount, &inv.Total, &inv.IssuedAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// MarkSent moves a draft invoice to "sent". An invoice that is already sent or paid is
// ErrInvoiceNotDraft (re-sending used to flip a paid invoice back to "sent"); a missing or
// foreign one is ErrInvoiceNotFound.
func (r *InvoiceRepo) MarkSent(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx,
		`UPDATE vat_invoices SET status = 'sent' WHERE id = $1 AND company_id = $2 AND status = 'draft'`, id, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vat_invoices WHERE id = $1 AND company_id = $2)`, id, cid).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrInvoiceNotFound
		}
		return ErrInvoiceNotDraft
	}
	return nil
}
