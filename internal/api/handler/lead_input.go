package handler

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// maxDealValue is the largest amount the leads.deal_value column (numeric(12,2)) can hold.
const maxDealValue = 9_999_999_999.99

// leadSources are the sources a new lead may be created with. "bos24" is still allowed by the
// leads_source_check constraint for leads the retired BOS24 integration imported, but no new
// lead gets it.
var leadSources = map[string]bool{"whatsapp": true, "web": true, "referral": true, "event": true,
	"property_finder": true, "bayut": true, "dubizzle": true, "portal": true}

// isCurrencyCode performs fast byte inspection to verify s is exactly a 3-letter ASCII currency code.
func isCurrencyCode(s string) bool {
	if len(s) != 3 {
		return false
	}
	return ((s[0] >= 'A' && s[0] <= 'Z') || (s[0] >= 'a' && s[0] <= 'z')) &&
		((s[1] >= 'A' && s[1] <= 'Z') || (s[1] >= 'a' && s[1] <= 'z')) &&
		((s[2] >= 'A' && s[2] <= 'Z') || (s[2] >= 'a' && s[2] <= 'z'))
}

// normalizeCurrency upper-cases a 3-letter currency code ("" means the AED default). The column
// is char(3): anything longer used to be a 500, and lower-case codes were stored as-is.
func normalizeCurrency(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "AED", true
	}
	if !isCurrencyCode(s) {
		return "", false
	}
	return strings.ToUpper(s), true
}

// applyLeadInput validates the client-controlled fields of a new lead and writes the normalised
// values into l. It returns a message for the client, or "" when the input is acceptable.
// An empty stage is left empty so the caller can apply the company's default stage.
func applyLeadInput(l *domain.Lead, stage, source, currency string, dealValue float64, notes string) string {
	l.Stage = domain.LeadStage(strings.TrimSpace(stage))

	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		source = "web" // the column is NOT NULL with a CHECK: an empty source used to be rejected
	}
	if !leadSources[source] {
		return "source must be one of: whatsapp, web, referral, event, property_finder, bayut, dubizzle, portal"
	}
	l.Source = domain.LeadSource(source)

	cur, ok := normalizeCurrency(currency)
	if !ok {
		return "currency must be a 3-letter code such as AED"
	}
	l.Currency = cur

	if math.IsNaN(dealValue) || math.IsInf(dealValue, 0) || dealValue < 0 || dealValue > maxDealValue {
		return "deal_value must be between 0 and 9999999999.99"
	}
	l.DealValue = math.Round(dealValue*100) / 100

	if utf8.RuneCountInString(notes) > maxLeadNotesRunes {
		return "notes are too long"
	}
	l.Notes = notes
	return ""
}
