package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/invoice"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type InvoiceRepo struct {
	db *pgxpool.Pool
}

func NewInvoiceRepo(db *pgxpool.Pool) *InvoiceRepo {
	return &InvoiceRepo{db: db}
}

// ErrInvalidTransition is returned for a status change the document's state forbids
// (e.g. paying a void invoice, voiding an invoice that has credit notes).
var ErrInvalidTransition = errors.New("invoice status change not allowed")

// Header columns, with the credited total of non-void credit notes as the last column.
const invoiceCols = `v.id, v.kind, v.deal_id, v.original_invoice_id, v.invoice_no, v.currency,
	v.subtotal, v.vat_rate, v.vat_amount, v.total, COALESCE(v.qr_payload,''), v.status, v.issued_at,
	v.supply_date, v.supplier_name, v.supplier_address, v.supplier_trn,
	v.customer_name, v.customer_address, v.customer_trn, v.notes, v.reason, v.void_reason,
	v.voided_at, v.created_by,
	COALESCE((SELECT SUM(cn.total) FROM vat_invoices cn
	           WHERE cn.original_invoice_id = v.id AND cn.status <> 'void'), 0)`

func scanInvoice(row pgx.Row, inv *domain.VATInvoice) error {
	err := row.Scan(&inv.ID, &inv.Kind, &inv.DealID, &inv.OriginalInvoiceID, &inv.InvoiceNo, &inv.Currency,
		&inv.Subtotal, &inv.VATRate, &inv.VATAmount, &inv.Total, &inv.QRPayload, &inv.Status, &inv.IssuedAt,
		&inv.SupplyDate, &inv.Supplier.Name, &inv.Supplier.Address, &inv.Supplier.TRN,
		&inv.Customer.Name, &inv.Customer.Address, &inv.Customer.TRN, &inv.Notes, &inv.Reason, &inv.VoidReason,
		&inv.VoidedAt, &inv.CreatedBy, &inv.CreditedTotal)
	if err != nil {
		return err
	}
	inv.IsTaxInvoice = inv.Supplier.TRN != ""
	inv.Outstanding = 0
	if inv.Kind == domain.KindInvoice && inv.Status != domain.InvoiceVoid && inv.Status != domain.InvoicePaid {
		inv.Outstanding = jurisdiction.RoundMoney(inv.Total-inv.CreditedTotal, inv.Currency)
	}
	return nil
}

// Issue numbers and stores an invoice or credit note with its lines, in one transaction.
// The amounts (Subtotal, VATAmount, Total, line amounts) must already be computed with
// package invoice. For a credit note, OriginalInvoiceID names the invoice; it is locked
// while the credit is checked, so concurrent credit notes cannot over-credit it.
func (r *InvoiceRepo) Issue(ctx context.Context, inv *domain.VATInvoice) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM deals WHERE id = $1 AND company_id = $2)`,
		inv.DealID, cid).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrForeignReference
	}

	prefix := invoice.PrefixInvoice
	if inv.Kind == domain.KindCreditNote {
		prefix = invoice.PrefixCreditNote
		if inv.OriginalInvoiceID == nil {
			return fmt.Errorf("credit note without original invoice")
		}
		var kind domain.InvoiceKind
		var status domain.InvoiceStatus
		var total float64
		var currency string
		err := tx.QueryRow(ctx, `
			SELECT kind, status, total, currency FROM vat_invoices
			 WHERE id = $1 AND company_id = $2 FOR UPDATE`, *inv.OriginalInvoiceID, cid,
		).Scan(&kind, &status, &total, &currency)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForeignReference
		}
		if err != nil {
			return err
		}
		if kind != domain.KindInvoice || status == domain.InvoiceVoid {
			return &invoice.ValidationError{Msg: "credit notes can only be issued against an invoice that is not void"}
		}
		if currency != inv.Currency {
			return &invoice.ValidationError{Msg: "a credit note must be in the invoice's currency"}
		}
		var credited float64
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(total), 0) FROM vat_invoices
			 WHERE original_invoice_id = $1 AND company_id = $2 AND status <> 'void'`,
			*inv.OriginalInvoiceID, cid).Scan(&credited); err != nil {
			return err
		}
		if err := invoice.CheckCredit(total, credited, inv.Total, currency); err != nil {
			return err
		}
	}

	year := time.Now().UTC().Year()
	var seq int
	if err := tx.QueryRow(ctx, `
		INSERT INTO invoice_sequences (company_id, kind, year, last_no) VALUES ($1, $2, $3, 1)
		ON CONFLICT (company_id, kind, year) DO UPDATE SET last_no = invoice_sequences.last_no + 1
		RETURNING last_no`, cid, string(inv.Kind), year).Scan(&seq); err != nil {
		return err
	}
	inv.InvoiceNo = invoice.Number(prefix, year, seq)
	inv.ID = uuid.New()
	if inv.Status == "" {
		inv.Status = domain.InvoiceDraft
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO vat_invoices (id, company_id, kind, deal_id, original_invoice_id, invoice_no, currency,
			subtotal, vat_rate, vat_amount, total, status, supply_date,
			supplier_name, supplier_address, supplier_trn, customer_name, customer_address, customer_trn,
			notes, reason, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		RETURNING issued_at`,
		inv.ID, cid, inv.Kind, inv.DealID, inv.OriginalInvoiceID, inv.InvoiceNo, inv.Currency,
		inv.Subtotal, inv.VATRate, inv.VATAmount, inv.Total, inv.Status, inv.SupplyDate,
		inv.Supplier.Name, inv.Supplier.Address, inv.Supplier.TRN,
		inv.Customer.Name, inv.Customer.Address, inv.Customer.TRN,
		inv.Notes, inv.Reason, inv.CreatedBy,
	).Scan(&inv.IssuedAt); err != nil {
		return err
	}

	batch := &pgx.Batch{}
	for i := range inv.Lines {
		l := &inv.Lines[i]
		l.ID = uuid.New()
		l.Position = i + 1
		batch.Queue(`
			INSERT INTO vat_invoice_lines (id, invoice_id, company_id, position, description, quantity,
				unit_price, vat_category, vat_rate, net_amount, vat_amount)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			l.ID, inv.ID, cid, l.Position, l.Description, l.Quantity, l.UnitPrice,
			l.VATCategory, l.VATRate, l.NetAmount, l.VATAmount)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return err
	}
	inv.IsTaxInvoice = inv.Supplier.TRN != ""
	if inv.Kind == domain.KindInvoice {
		inv.Outstanding = inv.Total
	}
	return tx.Commit(ctx)
}

// GetByID returns a document with its lines.
func (r *InvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.VATInvoice, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	inv := &domain.VATInvoice{}
	if err := scanInvoice(r.db.QueryRow(ctx,
		`SELECT `+invoiceCols+` FROM vat_invoices v WHERE v.id = $1 AND v.company_id = $2`, id, cid), inv); err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, position, description, quantity, unit_price, vat_category, vat_rate, net_amount, vat_amount
		  FROM vat_invoice_lines WHERE invoice_id = $1 AND company_id = $2 ORDER BY position`, id, cid)
	if err != nil {
		return nil, err
	}
	inv.Lines, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.InvoiceLine, error) {
		var l domain.InvoiceLine
		err := row.Scan(&l.ID, &l.Position, &l.Description, &l.Quantity, &l.UnitPrice,
			&l.VATCategory, &l.VATRate, &l.NetAmount, &l.VATAmount)
		return l, err
	})
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *InvoiceRepo) list(ctx context.Context, where string, args ...any) ([]domain.VATInvoice, error) {
	rows, err := r.db.Query(ctx, `SELECT `+invoiceCols+` FROM vat_invoices v WHERE `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()
	var out []domain.VATInvoice
	for rows.Next() {
		var inv domain.VATInvoice
		if err := scanInvoice(rows, &inv); err != nil {
			return nil, fmt.Errorf("scan invoice: %w", err)
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// ListByDeal returns a deal's invoices and credit notes, newest first (without lines).
func (r *InvoiceRepo) ListByDeal(ctx context.Context, dealID uuid.UUID) ([]domain.VATInvoice, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	return r.list(ctx, `v.deal_id = $1 AND v.company_id = $2 ORDER BY v.issued_at DESC, v.invoice_no DESC`, dealID, cid)
}

// ListCreditNotes returns the credit notes issued against an invoice.
func (r *InvoiceRepo) ListCreditNotes(ctx context.Context, invoiceID uuid.UUID) ([]domain.VATInvoice, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	return r.list(ctx, `v.original_invoice_id = $1 AND v.company_id = $2 ORDER BY v.issued_at, v.invoice_no`, invoiceID, cid)
}

// ListAll pages through the company's documents, newest first (without lines).
func (r *InvoiceRepo) ListAll(ctx context.Context, page, limit int) ([]domain.VATInvoice, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM vat_invoices WHERE company_id = $1`, cid).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}
	out, err := r.list(ctx, `v.company_id = $1 ORDER BY v.issued_at DESC, v.invoice_no DESC LIMIT $2 OFFSET $3`,
		cid, limit, (page-1)*limit)
	return out, total, err
}

// UpdateStatus changes a document's status. Allowed:
//   - draft ↔ sent, and sent/draft → paid, for invoices;
//   - draft ↔ sent for credit notes;
//   - any status except paid → void (with a reason); an invoice with credit notes that are
//     not void cannot be voided.
//
// A void document never changes again.
func (r *InvoiceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus, voidReason string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	var tag interface{ RowsAffected() int64 }
	switch status {
	case domain.InvoiceDraft, domain.InvoiceSent:
		tag, err = r.db.Exec(ctx, `UPDATE vat_invoices SET status = $1
			WHERE id = $2 AND company_id = $3 AND status IN ('draft','sent')`, status, id, cid)
	case domain.InvoicePaid:
		tag, err = r.db.Exec(ctx, `UPDATE vat_invoices SET status = 'paid'
			WHERE id = $1 AND company_id = $2 AND kind = 'invoice' AND status IN ('draft','sent')`, id, cid)
	case domain.InvoiceVoid:
		tag, err = r.db.Exec(ctx, `UPDATE vat_invoices v SET status = 'void', voided_at = NOW(), void_reason = $3
			WHERE v.id = $1 AND v.company_id = $2 AND v.status IN ('draft','sent')
			  AND NOT EXISTS (SELECT 1 FROM vat_invoices cn
			                   WHERE cn.original_invoice_id = v.id AND cn.status <> 'void')`, id, cid, voidReason)
	default:
		return ErrInvalidTransition
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vat_invoices WHERE id = $1 AND company_id = $2)`,
			id, cid).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return pgx.ErrNoRows
		}
		return ErrInvalidTransition
	}
	return nil
}
