package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type BankTransactionRepo struct {
	db *pgxpool.Pool
}

func NewBankTransactionRepo(db *pgxpool.Pool) *BankTransactionRepo {
	return &BankTransactionRepo{db: db}
}

func (r *BankTransactionRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.BankTransaction], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM bank_transactions WHERE company_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count transactions: %w", err)
	}

	const q = `
		SELECT id, company_id, bank_integration_id, external_id, transaction_date, amount, currency,
		       from_account, to_account, from_name, to_name, reference, transaction_type, status,
		       matched_payment_id, match_confidence, matched_at, imported_at, last_checked, sync_error
		FROM bank_transactions
		WHERE company_id = $1
		ORDER BY transaction_date DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []domain.BankTransaction
	for rows.Next() {
		var bt domain.BankTransaction
		if err := rows.Scan(
			&bt.ID, &bt.CompanyID, &bt.BankIntegrationID, &bt.ExternalID, &bt.TransactionDate, &bt.Amount, &bt.Currency,
			&bt.FromAccount, &bt.ToAccount, &bt.FromName, &bt.ToName, &bt.Reference, &bt.TransactionType, &bt.Status,
			&bt.MatchedPaymentID, &bt.MatchConfidence, &bt.MatchedAt, &bt.ImportedAt, &bt.LastChecked, &bt.SyncError,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, bt)
	}

	return &domain.PaginatedResult[domain.BankTransaction]{
		Data:  transactions,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *BankTransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.BankTransaction, error) {
	const q = `
		SELECT id, company_id, bank_integration_id, external_id, transaction_date, amount, currency,
		       from_account, to_account, from_name, to_name, reference, transaction_type, status,
		       matched_payment_id, match_confidence, matched_at, imported_at, last_checked, sync_error
		FROM bank_transactions WHERE id = $1
	`
	bt := &domain.BankTransaction{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&bt.ID, &bt.CompanyID, &bt.BankIntegrationID, &bt.ExternalID, &bt.TransactionDate, &bt.Amount, &bt.Currency,
		&bt.FromAccount, &bt.ToAccount, &bt.FromName, &bt.ToName, &bt.Reference, &bt.TransactionType, &bt.Status,
		&bt.MatchedPaymentID, &bt.MatchConfidence, &bt.MatchedAt, &bt.ImportedAt, &bt.LastChecked, &bt.SyncError,
	)
	if err != nil {
		return nil, fmt.Errorf("get transaction by id: %w", err)
	}
	return bt, nil
}

func (r *BankTransactionRepo) Create(ctx context.Context, bt *domain.BankTransaction) error {
	const q = `
		INSERT INTO bank_transactions (
			id, company_id, bank_integration_id, external_id, transaction_date, amount, currency,
			from_account, to_account, from_name, to_name, reference, transaction_type, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		RETURNING imported_at
	`
	bt.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		bt.ID, bt.CompanyID, bt.BankIntegrationID, bt.ExternalID, bt.TransactionDate, bt.Amount, bt.Currency,
		bt.FromAccount, bt.ToAccount, bt.FromName, bt.ToName, bt.Reference, bt.TransactionType, bt.Status,
	).Scan(&bt.ImportedAt)
}

func (r *BankTransactionRepo) Update(ctx context.Context, bt *domain.BankTransaction) error {
	const q = `
		UPDATE bank_transactions
		SET matched_payment_id=$1, match_confidence=$2, matched_at=$3, last_checked=$4
		WHERE id=$5
	`
	_, err := r.db.Exec(ctx, q, bt.MatchedPaymentID, bt.MatchConfidence, bt.MatchedAt, bt.LastChecked, bt.ID)
	return err
}

func (r *BankTransactionRepo) GetUnmatchedByAmount(ctx context.Context, companyID uuid.UUID, amount float64, tolerance float64) ([]domain.BankTransaction, error) {
	const q = `
		SELECT id, company_id, bank_integration_id, external_id, transaction_date, amount, currency,
		       from_account, to_account, from_name, to_name, reference, transaction_type, status,
		       matched_payment_id, match_confidence, matched_at, imported_at, last_checked, sync_error
		FROM bank_transactions
		WHERE company_id = $1 AND matched_payment_id IS NULL
		  AND amount BETWEEN $2 - $3 AND $2 + $3
		ORDER BY transaction_date DESC
	`
	rows, err := r.db.Query(ctx, q, companyID, amount, tolerance)
	if err != nil {
		return nil, fmt.Errorf("get unmatched transactions: %w", err)
	}
	defer rows.Close()

	var transactions []domain.BankTransaction
	for rows.Next() {
		var bt domain.BankTransaction
		if err := rows.Scan(
			&bt.ID, &bt.CompanyID, &bt.BankIntegrationID, &bt.ExternalID, &bt.TransactionDate, &bt.Amount, &bt.Currency,
			&bt.FromAccount, &bt.ToAccount, &bt.FromName, &bt.ToName, &bt.Reference, &bt.TransactionType, &bt.Status,
			&bt.MatchedPaymentID, &bt.MatchConfidence, &bt.MatchedAt, &bt.ImportedAt, &bt.LastChecked, &bt.SyncError,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, bt)
	}
	return transactions, nil
}

func (r *BankTransactionRepo) GetSinceDateByIntegration(ctx context.Context, integrationID uuid.UUID, since time.Time) ([]domain.BankTransaction, error) {
	const q = `
		SELECT id, company_id, bank_integration_id, external_id, transaction_date, amount, currency,
		       from_account, to_account, from_name, to_name, reference, transaction_type, status,
		       matched_payment_id, match_confidence, matched_at, imported_at, last_checked, sync_error
		FROM bank_transactions
		WHERE bank_integration_id = $1 AND transaction_date >= $2
		ORDER BY transaction_date DESC
	`
	rows, err := r.db.Query(ctx, q, integrationID, since)
	if err != nil {
		return nil, fmt.Errorf("get transactions since date: %w", err)
	}
	defer rows.Close()

	var transactions []domain.BankTransaction
	for rows.Next() {
		var bt domain.BankTransaction
		if err := rows.Scan(
			&bt.ID, &bt.CompanyID, &bt.BankIntegrationID, &bt.ExternalID, &bt.TransactionDate, &bt.Amount, &bt.Currency,
			&bt.FromAccount, &bt.ToAccount, &bt.FromName, &bt.ToName, &bt.Reference, &bt.TransactionType, &bt.Status,
			&bt.MatchedPaymentID, &bt.MatchConfidence, &bt.MatchedAt, &bt.ImportedAt, &bt.LastChecked, &bt.SyncError,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, bt)
	}
	return transactions, nil
}
