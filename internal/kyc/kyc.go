// Package kyc holds the customer-due-diligence and anti-money-laundering rules Masaar
// applies as workflow support: when a client's identity record is complete, and when a
// deal should be put in front of a compliance reviewer. It does not decide whether to
// report anything; people do. The thresholds are settings (check the current Ministry of
// Economy and UAE FIU guidance), and nothing here is legal advice.
package kyc

import (
	"context"
	"errors"
	"time"
)

// Identity document types.
const (
	EmiratesID    = "emirates_id"
	Passport      = "passport"
	ResidenceVisa = "residence_visa"
	TradeLicence  = "trade_licence"
	Other         = "other"
)

// ValidDocType reports whether t is a known document type.
func ValidDocType(t string) bool {
	switch t {
	case EmiratesID, Passport, ResidenceVisa, TradeLicence, Other:
		return true
	}
	return false
}

// Doc is what the completeness check needs to know about a document.
type Doc struct {
	Type   string
	Expiry *time.Time
}

// Problems with a client's identity record.
const (
	MissingID   = "id_missing"      // no identity document recorded
	ExpiredID   = "id_expired"      // the identity documents have all expired
	NoExpiry    = "id_no_expiry"    // the identity document has no expiry date recorded
	MissingCorp = "licence_missing" // a corporate client without a valid trade licence
)

// identity reports whether a document identifies a person (or a company, for corporate
// clients).
func identity(t string, corporate bool) bool {
	if corporate {
		return t == TradeLicence
	}
	return t == EmiratesID || t == Passport
}

// Check returns what is missing from a client's identity record on a given day: a valid
// (unexpired, dated) Emirates ID or passport, or, for a corporate client, trade licence.
// An empty result means complete.
func Check(docs []Doc, corporate bool, today time.Time) []string {
	y, m, d := today.Date()
	today = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	found, undated := false, false
	for _, doc := range docs {
		if !identity(doc.Type, corporate) {
			continue
		}
		found = true
		if doc.Expiry == nil {
			undated = true
			continue
		}
		if !doc.Expiry.Before(today) {
			return nil
		}
	}
	switch {
	case !found && corporate:
		return []string{MissingCorp}
	case !found:
		return []string{MissingID}
	case undated:
		return []string{NoExpiry}
	default:
		return []string{ExpiredID}
	}
}

// Deal payment methods.
const (
	PayCash         = "cash"
	PayCheque       = "cheque"
	PayTransfer     = "transfer"
	PayVirtualAsset = "virtual_asset"
	PayMixed        = "mixed"
)

// ValidPaymentMethod reports whether m is a known payment method ("" means unknown).
func ValidPaymentMethod(m string) bool {
	switch m {
	case "", PayCash, PayCheque, PayTransfer, PayVirtualAsset, PayMixed:
		return true
	}
	return false
}

// Reasons a deal goes to the review queue.
const (
	ReasonCash         = "cash_at_or_above_threshold"
	ReasonVirtualAsset = "virtual_asset"
)

// AMLSettings are a company's review criteria.
type AMLSettings struct {
	CashThreshold    float64 `json:"cash_threshold"`
	FlagVirtualAsset bool    `json:"flag_virtual_asset"`
}

// FlagReasons returns why a deal should be reviewed, if at all. Cash counts the cash
// part of the price: all of it when paid in cash, cashAmount when the payment is mixed.
func FlagReasons(method string, amount, cashAmount float64, s AMLSettings) []string {
	var out []string
	cash := cashAmount
	if method == PayCash && amount > cash {
		cash = amount
	}
	if s.CashThreshold > 0 && cash >= s.CashThreshold {
		out = append(out, ReasonCash)
	}
	if s.FlagVirtualAsset && method == PayVirtualAsset {
		out = append(out, ReasonVirtualAsset)
	}
	return out
}

// ── Screening ────────────────────────────────────────────────────────────────

// Screening results.
const (
	NotScreened   = "not_screened"
	Clear         = "clear"
	PossibleMatch = "possible_match"
	Match         = "match"
)

// ValidScreening reports whether s is a known screening result.
func ValidScreening(s string) bool {
	switch s {
	case NotScreened, Clear, PossibleMatch, Match:
		return true
	}
	return false
}

// Subject is who is screened.
type Subject struct {
	Name        string
	DateOfBirth *time.Time
	Nationality string // ISO 3166-1 alpha-2
	IDNumber    string
}

// ScreeningResult is a provider's answer.
type ScreeningResult struct {
	Status    string // clear | possible_match | match
	Reference string // the provider's case or search id
	Details   string // a short summary for the record
}

// Screener checks a subject against sanctions and PEP lists. No provider is bundled;
// an integration implements this interface and is passed to the KYC handler. Results
// are recorded for a person to review: a match does not block anything by itself.
type Screener interface {
	Name() string
	Screen(ctx context.Context, s Subject) (ScreeningResult, error)
}

// ErrNoScreener is returned when no screening provider is configured.
var ErrNoScreener = errors.New("no screening provider is configured; record the result by hand")
