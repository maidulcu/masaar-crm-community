package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type PaymentRepo struct {
	db *pgxpool.Pool
}

func NewPaymentRepo(db *pgxpool.Pool) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.Payment], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM payments WHERE company_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count payments: %w", err)
	}

	const q = `
		SELECT id, company_id, lease_id, amount, currency, due_date, paid_date, payment_method,
		       payment_reference, status, bank_transaction_id, reconciled_at, reconciled_by,
		       notes, receipt_url, late_fee_applied, late_fee_amount,
		       created_at, updated_at, created_by, updated_by
		FROM payments
		WHERE company_id = $1
		ORDER BY due_date DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.LeaseID, &p.Amount, &p.Currency, &p.DueDate, &p.PaidDate, &p.PaymentMethod,
			&p.PaymentReference, &p.Status, &p.BankTransactionID, &p.ReconciledAt, &p.ReconciledBy,
			&p.Notes, &p.ReceiptURL, &p.LateFeesApplied, &p.LateFeeAmount,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}

	return &domain.PaginatedResult[domain.Payment]{
		Data:  payments,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *PaymentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	const q = `
		SELECT id, company_id, lease_id, amount, currency, due_date, paid_date, payment_method,
		       payment_reference, status, bank_transaction_id, reconciled_at, reconciled_by,
		       notes, receipt_url, late_fee_applied, late_fee_amount,
		       created_at, updated_at, created_by, updated_by
		FROM payments WHERE id = $1
	`
	p := &domain.Payment{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&p.ID, &p.CompanyID, &p.LeaseID, &p.Amount, &p.Currency, &p.DueDate, &p.PaidDate, &p.PaymentMethod,
		&p.PaymentReference, &p.Status, &p.BankTransactionID, &p.ReconciledAt, &p.ReconciledBy,
		&p.Notes, &p.ReceiptURL, &p.LateFeesApplied, &p.LateFeeAmount,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get payment by id: %w", err)
	}
	return p, nil
}

func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	const q = `
		INSERT INTO payments (
			id, company_id, lease_id, amount, currency, due_date, paid_date, payment_method,
			payment_reference, status, notes, receipt_url, late_fee_applied, late_fee_amount,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		RETURNING created_at, updated_at
	`
	p.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		p.ID, p.CompanyID, p.LeaseID, p.Amount, p.Currency, p.DueDate, p.PaidDate, p.PaymentMethod,
		p.PaymentReference, p.Status, p.Notes, p.ReceiptURL, p.LateFeesApplied, p.LateFeeAmount,
		p.CreatedBy, p.UpdatedBy,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
}

func (r *PaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	const q = `
		UPDATE payments
		SET amount=$1, paid_date=$2, payment_method=$3, payment_reference=$4, status=$5,
		    bank_transaction_id=$6, reconciled_at=$7, reconciled_by=$8, notes=$9, receipt_url=$10,
		    late_fee_applied=$11, late_fee_amount=$12, updated_by=$13, updated_at=NOW()
		WHERE id=$14
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, q,
		p.Amount, p.PaidDate, p.PaymentMethod, p.PaymentReference, p.Status,
		p.BankTransactionID, p.ReconciledAt, p.ReconciledBy, p.Notes, p.ReceiptURL,
		p.LateFeesApplied, p.LateFeeAmount, p.UpdatedBy, p.ID,
	).Scan(&p.UpdatedAt)
}

func (r *PaymentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM payments WHERE id=$1`, id)
	return err
}

func (r *PaymentRepo) GetByLeaseID(ctx context.Context, leaseID uuid.UUID) ([]domain.Payment, error) {
	const q = `
		SELECT id, company_id, lease_id, amount, currency, due_date, paid_date, payment_method,
		       payment_reference, status, bank_transaction_id, reconciled_at, reconciled_by,
		       notes, receipt_url, late_fee_applied, late_fee_amount,
		       created_at, updated_at, created_by, updated_by
		FROM payments WHERE lease_id = $1
		ORDER BY due_date ASC
	`
	rows, err := r.db.Query(ctx, q, leaseID)
	if err != nil {
		return nil, fmt.Errorf("get payments by lease: %w", err)
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.LeaseID, &p.Amount, &p.Currency, &p.DueDate, &p.PaidDate, &p.PaymentMethod,
			&p.PaymentReference, &p.Status, &p.BankTransactionID, &p.ReconciledAt, &p.ReconciledBy,
			&p.Notes, &p.ReceiptURL, &p.LateFeesApplied, &p.LateFeeAmount,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *PaymentRepo) GetPendingByDueDate(ctx context.Context, companyID uuid.UUID, beforeDate time.Time) ([]domain.Payment, error) {
	const q = `
		SELECT id, company_id, lease_id, amount, currency, due_date, paid_date, payment_method,
		       payment_reference, status, bank_transaction_id, reconciled_at, reconciled_by,
		       notes, receipt_url, late_fee_applied, late_fee_amount,
		       created_at, updated_at, created_by, updated_by
		FROM payments
		WHERE company_id = $1 AND status = 'pending' AND due_date <= $2
		ORDER BY due_date ASC
	`
	rows, err := r.db.Query(ctx, q, companyID, beforeDate)
	if err != nil {
		return nil, fmt.Errorf("get pending payments: %w", err)
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.LeaseID, &p.Amount, &p.Currency, &p.DueDate, &p.PaidDate, &p.PaymentMethod,
			&p.PaymentReference, &p.Status, &p.BankTransactionID, &p.ReconciledAt, &p.ReconciledBy,
			&p.Notes, &p.ReceiptURL, &p.LateFeesApplied, &p.LateFeeAmount,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *PaymentRepo) GetByStatus(ctx context.Context, companyID uuid.UUID, status domain.PaymentStatus) ([]domain.Payment, error) {
	const q = `
		SELECT id, company_id, lease_id, amount, currency, due_date, paid_date, payment_method,
		       payment_reference, status, bank_transaction_id, reconciled_at, reconciled_by,
		       notes, receipt_url, late_fee_applied, late_fee_amount,
		       created_at, updated_at, created_by, updated_by
		FROM payments WHERE company_id = $1 AND status = $2
		ORDER BY due_date DESC
	`
	rows, err := r.db.Query(ctx, q, companyID, status)
	if err != nil {
		return nil, fmt.Errorf("get payments by status: %w", err)
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.LeaseID, &p.Amount, &p.Currency, &p.DueDate, &p.PaidDate, &p.PaymentMethod,
			&p.PaymentReference, &p.Status, &p.BankTransactionID, &p.ReconciledAt, &p.ReconciledBy,
			&p.Notes, &p.ReceiptURL, &p.LateFeesApplied, &p.LateFeeAmount,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *PaymentRepo) GetByDueDate(ctx context.Context, companyID uuid.UUID, dueDate time.Time) ([]domain.Payment, error) {
	const q = `
		SELECT id, company_id, lease_id, amount, currency, due_date, paid_date, payment_method,
		       payment_reference, status, bank_transaction_id, reconciled_at, reconciled_by,
		       notes, receipt_url, late_fee_applied, late_fee_amount,
		       created_at, updated_at, created_by, updated_by
		FROM payments
		WHERE company_id = $1 AND DATE(due_date) = $2 AND status IN ('pending', 'overdue')
		ORDER BY due_date ASC
	`
	rows, err := r.db.Query(ctx, q, companyID, dueDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("get payments by due date: %w", err)
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.LeaseID, &p.Amount, &p.Currency, &p.DueDate, &p.PaidDate, &p.PaymentMethod,
			&p.PaymentReference, &p.Status, &p.BankTransactionID, &p.ReconciledAt, &p.ReconciledBy,
			&p.Notes, &p.ReceiptURL, &p.LateFeesApplied, &p.LateFeeAmount,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *PaymentRepo) GetOverdueByDate(ctx context.Context, companyID uuid.UUID, sinceDateInclusive time.Time) ([]domain.Payment, error) {
	const q = `
		SELECT id, company_id, lease_id, amount, currency, due_date, paid_date, payment_method,
		       payment_reference, status, bank_transaction_id, reconciled_at, reconciled_by,
		       notes, receipt_url, late_fee_applied, late_fee_amount,
		       created_at, updated_at, created_by, updated_by
		FROM payments
		WHERE company_id = $1 AND status IN ('pending', 'overdue') AND due_date <= $2
		ORDER BY due_date ASC
	`
	rows, err := r.db.Query(ctx, q, companyID, sinceDateInclusive)
	if err != nil {
		return nil, fmt.Errorf("get overdue payments: %w", err)
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(
			&p.ID, &p.CompanyID, &p.LeaseID, &p.Amount, &p.Currency, &p.DueDate, &p.PaidDate, &p.PaymentMethod,
			&p.PaymentReference, &p.Status, &p.BankTransactionID, &p.ReconciledAt, &p.ReconciledBy,
			&p.Notes, &p.ReceiptURL, &p.LateFeesApplied, &p.LateFeeAmount,
			&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, nil
}
