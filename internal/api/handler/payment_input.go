package handler

import (
	"errors"
	"math"
	"strings"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

var paymentDateKeys = []string{"due_date", "paid_date", "reconciled_at"}

var (
	// The web form offers bank_transfer/cheque/credit_card/online next to the Go constants.
	validPaymentMethods = map[domain.PaymentMethod]bool{
		"transfer": true, "bank_transfer": true, "check": true, "cheque": true, "cash": true,
		"card": true, "credit_card": true, "online": true, "other": true,
	}
	validPaymentStatuses = map[domain.PaymentStatus]bool{
		domain.PaymentPending: true, domain.PaymentReceived: true, domain.PaymentOverdue: true,
		domain.PaymentFailed: true, domain.PaymentRefunded: true,
	}
)

// validatePayment applies defaults and validates the client-editable fields of a payment in place.
// currency may be empty; the caller defaults it from the lease.
func validatePayment(p *domain.Payment) error {
	if p.Amount <= 0 || badMoney(p.Amount) || math.IsNaN(p.Amount) {
		return errors.New("amount must be greater than 0")
	}
	if p.DueDate.IsZero() {
		return errors.New("due_date is required")
	}
	p.PaymentMethod = domain.PaymentMethod(strings.ToLower(strings.TrimSpace(string(p.PaymentMethod))))
	if !validPaymentMethods[p.PaymentMethod] {
		return errors.New("payment_method must be one of: transfer, bank_transfer, check, cheque, cash, card, credit_card, online, other")
	}
	if p.Status == "" {
		p.Status = domain.PaymentPending
	}
	if !validPaymentStatuses[p.Status] {
		return errors.New("status must be one of: pending, received, overdue, failed, refunded")
	}
	if p.Currency != "" {
		cur, ok := normalizeCurrency(p.Currency)
		if !ok {
			return errors.New("currency must be a 3-letter code such as AED")
		}
		p.Currency = cur
	}
	if p.LateFeeAmount < 0 || badMoney(p.LateFeeAmount) {
		return errors.New("late_fee_amount must not be negative")
	}
	if tooLong(p.PaymentReference, 255) || tooLong(p.Notes, 10000) {
		return errors.New("a field is too long")
	}
	// A payment that has been received has a date it was received on; one that has not (any more)
	// must not keep a paid date, or it reads as both unpaid and paid.
	if p.Status == domain.PaymentReceived && p.PaidDate == nil {
		now := time.Now()
		p.PaidDate = &now
	}
	if p.Status == domain.PaymentPending || p.Status == domain.PaymentOverdue {
		p.PaidDate = nil
	}
	return nil
}
