// Package invoice computes and validates tax invoices and credit notes: per-line VAT by
// category, rounding to the currency's minor units, and the totals stored on the invoice.
//
// VAT is calculated per line and rounded per line; the invoice VAT is the sum of the line
// amounts, so the printed lines always add up to the printed total.
package invoice

import (
	"errors"
	"fmt"
	"strings"

	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
)

// MaxLines bounds the size of one document.
const MaxLines = 200

// Line is one invoice line as entered.
type Line struct {
	Description string                   `json:"description"`
	Quantity    float64                  `json:"quantity"`
	UnitPrice   float64                  `json:"unit_price"`
	VATCategory jurisdiction.VATCategory `json:"vat_category"`
}

// ComputedLine is a line with its rate and rounded amounts.
type ComputedLine struct {
	Line
	VATRate   float64 `json:"vat_rate"`
	NetAmount float64 `json:"net_amount"`
	VATAmount float64 `json:"vat_amount"`
}

// Totals are the document's amounts.
type Totals struct {
	Subtotal  float64        `json:"subtotal"`
	VATAmount float64        `json:"vat_amount"`
	Total     float64        `json:"total"`
	Lines     []ComputedLine `json:"lines"`
}

// ValidationError is a problem with the caller's input (answer 422).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) error { return &ValidationError{Msg: fmt.Sprintf(format, a...)} }

// IsValidation reports whether err is a ValidationError.
func IsValidation(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// Compute validates lines and calculates their VAT under rules, rounding to currency.
// supplierRegistered is false when the company has no TRN: it may then only issue
// documents without VAT (zero-rated, exempt and standard-rated lines are refused, and
// out-of-scope is the only category allowed).
func Compute(lines []Line, rules jurisdiction.Rules, currency string, supplierRegistered bool) (*Totals, error) {
	if len(lines) == 0 {
		return nil, invalid("at least one line is required")
	}
	if len(lines) > MaxLines {
		return nil, invalid("at most %d lines are allowed", MaxLines)
	}
	if !jurisdiction.IsCurrency(currency) {
		return nil, invalid("unsupported currency %q", currency)
	}
	t := &Totals{Lines: make([]ComputedLine, 0, len(lines))}
	for i, l := range lines {
		n := i + 1
		l.Description = strings.TrimSpace(l.Description)
		if l.Description == "" {
			return nil, invalid("line %d: description is required", n)
		}
		if len(l.Description) > 1000 {
			return nil, invalid("line %d: description is too long", n)
		}
		if l.Quantity <= 0 {
			return nil, invalid("line %d: quantity must be positive", n)
		}
		if l.UnitPrice < 0 {
			return nil, invalid("line %d: unit price cannot be negative", n)
		}
		if l.VATCategory == "" {
			l.VATCategory = jurisdiction.VATStandard
			if !supplierRegistered {
				l.VATCategory = jurisdiction.VATOutOfScope
			}
		}
		if !supplierRegistered && l.VATCategory != jurisdiction.VATOutOfScope {
			return nil, invalid("line %d: set the company's TRN (Settings → Company) before issuing a tax invoice", n)
		}
		rate, err := rules.VATRate(l.VATCategory)
		if err != nil {
			return nil, invalid("line %d: unknown VAT category %q", n, l.VATCategory)
		}
		net := jurisdiction.RoundMoney(l.Quantity*l.UnitPrice, currency)
		vat := jurisdiction.RoundMoney(net*rate, currency)
		t.Lines = append(t.Lines, ComputedLine{Line: l, VATRate: rate, NetAmount: net, VATAmount: vat})
		t.Subtotal += net
		t.VATAmount += vat
	}
	t.Subtotal = jurisdiction.RoundMoney(t.Subtotal, currency)
	t.VATAmount = jurisdiction.RoundMoney(t.VATAmount, currency)
	t.Total = jurisdiction.RoundMoney(t.Subtotal+t.VATAmount, currency)
	if t.Total <= 0 {
		return nil, invalid("the total must be greater than zero")
	}
	return t, nil
}

// CheckCredit reports whether a credit note of creditTotal fits on an invoice of
// invoiceTotal that already carries creditedSoFar in credit notes.
func CheckCredit(invoiceTotal, creditedSoFar, creditTotal float64, currency string) error {
	remaining := jurisdiction.RoundMoney(invoiceTotal-creditedSoFar, currency)
	if jurisdiction.RoundMoney(creditTotal, currency) > remaining {
		return invalid("the credit note (%.3f) exceeds the amount still creditable (%.3f)", creditTotal, remaining)
	}
	return nil
}

// Document number prefixes.
const (
	PrefixInvoice    = "INV"
	PrefixCreditNote = "CN"
)

// Number formats a document number such as INV-2026-0007.
func Number(prefix string, year, seq int) string {
	return fmt.Sprintf("%s-%04d-%04d", prefix, year, seq)
}
