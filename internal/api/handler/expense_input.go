package handler

import (
	"errors"
	"net/url"
	"strings"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

const maxExpenseAmount = 999_999_999_999.99 // numeric(15,2) with headroom for sums

var (
	// The web form offers bank_transfer/credit_card/check; the shorter forms are accepted too.
	validExpenseMethods = map[domain.ExpensePaymentMethod]bool{
		"cash": true, "bank_transfer": true, "transfer": true, "credit_card": true, "card": true,
		"check": true, "cheque": true, "other": true,
	}
	validExpenseStatuses = map[domain.ExpensePaymentStatus]bool{
		domain.ExpensePaymentPending: true, domain.ExpensePaymentPaid: true, domain.ExpensePaymentRefunded: true,
	}
	// Category types: the declared ones plus the maintenance trades the financial analytics group on.
	validCategoryTypes = map[domain.ExpenseCategoryType]bool{
		"property_maintenance": true, "maintenance": true, "utilities": true, "insurance": true, "cleaning": true,
		"repairs": true, "staff": true, "other": true, "plumbing": true, "electrical": true, "hvac": true,
		"flooring": true, "painting": true, "structural": true,
	}
)

// safeLink reports whether s is empty, an absolute http(s) URL or a site-relative path. Anything else
// (javascript:, data:, file:, ...) would be a stored-XSS vector wherever the link is rendered.
func safeLink(s string) bool {
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "/") {
		return !strings.HasPrefix(s, "//") && !strings.Contains(s, `\`)
	}
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}

// validateExpense applies defaults and validates the client-editable fields in place.
func validateExpense(e *domain.Expense) error {
	e.Description = strings.TrimSpace(e.Description)
	if e.Amount <= 0 || e.Amount > maxExpenseAmount || e.Amount != e.Amount {
		return errors.New("amount must be greater than 0")
	}
	if e.ExpenseDate.IsZero() {
		return errors.New("expense_date is required")
	}
	if e.Description == "" {
		return errors.New("description is required")
	}
	e.PaymentMethod = domain.ExpensePaymentMethod(strings.ToLower(strings.TrimSpace(string(e.PaymentMethod))))
	if e.PaymentMethod == "" {
		e.PaymentMethod = domain.ExpensePaymentOther
	}
	if !validExpenseMethods[e.PaymentMethod] {
		return errors.New("payment_method must be one of: cash, bank_transfer, credit_card, check, other")
	}
	if e.PaymentStatus == "" {
		e.PaymentStatus = domain.ExpensePaymentPending
	}
	if !validExpenseStatuses[e.PaymentStatus] {
		return errors.New("payment_status must be one of: pending, paid, refunded")
	}
	if tooLong(e.Description, 5000) || tooLong(e.VendorName, 150) || tooLong(e.VendorContact, 255) || tooLong(e.Notes, 10000) {
		return errors.New("a field is too long")
	}
	if tooLong(e.ReceiptURL, 500) || !safeLink(e.ReceiptURL) {
		return errors.New("receipt_url must be an http(s) URL")
	}
	return nil
}

// validateExpenseCategory validates a new category in place.
func validateExpenseCategory(c *domain.ExpenseCategory) error {
	c.CategoryName = strings.TrimSpace(c.CategoryName)
	c.CategoryType = domain.ExpenseCategoryType(strings.ToLower(strings.TrimSpace(string(c.CategoryType))))
	if c.CategoryName == "" {
		return errors.New("category_name is required")
	}
	if c.CategoryType == "" {
		c.CategoryType = domain.ExpenseCategoryOther
	}
	if !validCategoryTypes[c.CategoryType] {
		return errors.New("category_type is not a known type")
	}
	if tooLong(c.CategoryName, 100) || tooLong(c.Description, 2000) {
		return errors.New("a field is too long")
	}
	return nil
}
