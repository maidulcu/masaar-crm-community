package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

type ExpenseRepository struct {
	conn *pgxpool.Pool
}

func NewExpenseRepository(conn *pgxpool.Pool) *ExpenseRepository {
	return &ExpenseRepository{conn: conn}
}

// Categories
func (r *ExpenseRepository) CreateCategory(ctx context.Context, cat *domain.ExpenseCategory) error {
	return r.conn.QueryRow(ctx, `
		INSERT INTO expense_categories (id, company_id, category_name, category_type, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, cat.ID, cat.CompanyID, cat.CategoryName, cat.CategoryType, cat.Description).Scan(&cat.ID, &cat.CreatedAt)
}

func (r *ExpenseRepository) ListCategories(ctx context.Context, companyID uuid.UUID) ([]domain.ExpenseCategory, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, category_name, category_type, description, created_at
		FROM expense_categories
		WHERE company_id = $1
		ORDER BY category_name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []domain.ExpenseCategory
	for rows.Next() {
		var cat domain.ExpenseCategory
		if err := rows.Scan(&cat.ID, &cat.CompanyID, &cat.CategoryName, &cat.CategoryType, &cat.Description, &cat.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, cat)
	}
	return cats, rows.Err()
}

// Expenses
func (r *ExpenseRepository) CreateExpense(ctx context.Context, exp *domain.Expense) error {
	return r.conn.QueryRow(ctx, `
		INSERT INTO expenses (id, company_id, category_id, property_id, tenant_id, amount, currency, expense_date, description, vendor_name, vendor_contact, payment_method, payment_status, receipt_url, notes, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at, updated_at
	`, exp.ID, exp.CompanyID, exp.CategoryID, exp.PropertyID, exp.TenantID, exp.Amount, exp.Currency, exp.ExpenseDate, exp.Description, exp.VendorName, exp.VendorContact, exp.PaymentMethod, exp.PaymentStatus, exp.ReceiptURL, exp.Notes, exp.CreatedBy).Scan(&exp.ID, &exp.CreatedAt, &exp.UpdatedAt)
}

func (r *ExpenseRepository) GetExpense(ctx context.Context, expenseID uuid.UUID) (*domain.Expense, error) {
	var exp domain.Expense
	err := r.conn.QueryRow(ctx, `
		SELECT id, company_id, category_id, property_id, tenant_id, amount, currency, expense_date, description, vendor_name, vendor_contact, payment_method, payment_status, receipt_url, notes, created_by, created_at, updated_at, deleted_at
		FROM expenses
		WHERE id = $1 AND deleted_at IS NULL
	`, expenseID).Scan(
		&exp.ID, &exp.CompanyID, &exp.CategoryID, &exp.PropertyID, &exp.TenantID, &exp.Amount, &exp.Currency, &exp.ExpenseDate, &exp.Description, &exp.VendorName, &exp.VendorContact, &exp.PaymentMethod, &exp.PaymentStatus, &exp.ReceiptURL, &exp.Notes, &exp.CreatedBy, &exp.CreatedAt, &exp.UpdatedAt, &exp.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &exp, nil
}

func (r *ExpenseRepository) ListExpenses(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.Expense, int, error) {
	var total int
	err := r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM expenses WHERE company_id = $1 AND deleted_at IS NULL`, companyID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, category_id, property_id, tenant_id, amount, currency, expense_date, description, vendor_name, vendor_contact, payment_method, payment_status, receipt_url, notes, created_by, created_at, updated_at, deleted_at
		FROM expenses
		WHERE company_id = $1 AND deleted_at IS NULL
		ORDER BY expense_date DESC
		LIMIT $2 OFFSET $3
	`, companyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var exps []domain.Expense
	for rows.Next() {
		var exp domain.Expense
		if err := rows.Scan(&exp.ID, &exp.CompanyID, &exp.CategoryID, &exp.PropertyID, &exp.TenantID, &exp.Amount, &exp.Currency, &exp.ExpenseDate, &exp.Description, &exp.VendorName, &exp.VendorContact, &exp.PaymentMethod, &exp.PaymentStatus, &exp.ReceiptURL, &exp.Notes, &exp.CreatedBy, &exp.CreatedAt, &exp.UpdatedAt, &exp.DeletedAt); err != nil {
			return nil, 0, err
		}
		exps = append(exps, exp)
	}
	return exps, total, rows.Err()
}

func (r *ExpenseRepository) UpdateExpense(ctx context.Context, exp *domain.Expense) error {
	exp.UpdatedAt = time.Now()
	return r.conn.QueryRow(ctx, `
		UPDATE expenses
		SET category_id = $2, property_id = $3, tenant_id = $4, amount = $5, currency = $6, expense_date = $7, description = $8, vendor_name = $9, vendor_contact = $10, payment_method = $11, payment_status = $12, receipt_url = $13, notes = $14, updated_at = $15
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at
	`, exp.ID, exp.CategoryID, exp.PropertyID, exp.TenantID, exp.Amount, exp.Currency, exp.ExpenseDate, exp.Description, exp.VendorName, exp.VendorContact, exp.PaymentMethod, exp.PaymentStatus, exp.ReceiptURL, exp.Notes, exp.UpdatedAt).Scan(&exp.UpdatedAt)
}

func (r *ExpenseRepository) DeleteExpense(ctx context.Context, expenseID uuid.UUID) error {
	_, err := r.conn.Exec(ctx, `
		UPDATE expenses
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND deleted_at IS NULL
	`, expenseID)
	return err
}

// Approvals
func (r *ExpenseRepository) CreateApproval(ctx context.Context, approval *domain.ExpenseApproval) error {
	return r.conn.QueryRow(ctx, `
		INSERT INTO expense_approvals (id, expense_id, approval_status, approved_by, approval_comments, approval_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, approval.ID, approval.ExpenseID, approval.ApprovalStatus, approval.ApprovedBy, approval.ApprovalComments, approval.ApprovalDate).Scan(&approval.ID, &approval.CreatedAt)
}

func (r *ExpenseRepository) GetApprovalByExpense(ctx context.Context, expenseID uuid.UUID) (*domain.ExpenseApproval, error) {
	var approval domain.ExpenseApproval
	err := r.conn.QueryRow(ctx, `
		SELECT id, expense_id, approval_status, approved_by, approval_comments, approval_date, created_at
		FROM expense_approvals
		WHERE expense_id = $1
	`, expenseID).Scan(
		&approval.ID, &approval.ExpenseID, &approval.ApprovalStatus, &approval.ApprovedBy, &approval.ApprovalComments, &approval.ApprovalDate, &approval.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &approval, nil
}

func (r *ExpenseRepository) UpdateApproval(ctx context.Context, approval *domain.ExpenseApproval) error {
	_, err := r.conn.Exec(ctx, `
		UPDATE expense_approvals
		SET approval_status = $2, approved_by = $3, approval_comments = $4, approval_date = $5
		WHERE id = $1
	`, approval.ID, approval.ApprovalStatus, approval.ApprovedBy, approval.ApprovalComments, approval.ApprovalDate)
	return err
}
