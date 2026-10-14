// Package jurisdiction holds the country-specific rules the CRM needs: VAT rates and
// categories, currency and its minor-unit precision, default timezone, weekend days and
// the regions (emirates, provinces) a company or listing can be in.
//
// Code elsewhere asks this package instead of hard-coding UAE values, so adding a
// country means adding a Rules value here, not changing handlers.
//
// The rates and rules below are configuration, not legal advice. Check them against the
// tax authority's current guidance before relying on them.
package jurisdiction

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// VATCategory classifies a supply for VAT purposes.
type VATCategory string

const (
	VATStandard   VATCategory = "standard"
	VATZeroRated  VATCategory = "zero_rated"
	VATExempt     VATCategory = "exempt"
	VATOutOfScope VATCategory = "out_of_scope"
)

// Region is a first-level subdivision (an emirate, a province) identified by its
// ISO 3166-2 code.
type Region struct {
	Code   string `json:"code"`
	NameEN string `json:"name_en"`
	NameAR string `json:"name_ar"`
}

// Rules are the defaults for one country.
type Rules struct {
	Country          string                  `json:"country"` // ISO 3166-1 alpha-2
	NameEN           string                  `json:"name_en"`
	NameAR           string                  `json:"name_ar"`
	Currency         string                  `json:"currency"` // ISO 4217
	CurrencyDecimals int                     `json:"currency_decimals"`
	Timezone         string                  `json:"timezone"` // IANA name
	Weekend          []time.Weekday          `json:"weekend"`
	VATRates         map[VATCategory]float64 `json:"vat_rates"` // fraction, e.g. 0.05
	// TaxIDDigits is the length of the VAT registration number (the UAE TRN has 15).
	TaxIDDigits int      `json:"tax_id_digits"`
	Regions     []Region `json:"regions"`
}

// DefaultCountry is used when a company has not chosen one.
const DefaultCountry = "AE"

var registry = map[string]Rules{
	"AE": {
		Country:          "AE",
		NameEN:           "United Arab Emirates",
		NameAR:           "الإمارات العربية المتحدة",
		Currency:         "AED",
		CurrencyDecimals: 2,
		Timezone:         "Asia/Dubai",
		Weekend:          []time.Weekday{time.Saturday, time.Sunday},
		TaxIDDigits:      15,
		VATRates: map[VATCategory]float64{
			VATStandard:   0.05,
			VATZeroRated:  0,
			VATExempt:     0,
			VATOutOfScope: 0,
		},
		Regions: []Region{
			{Code: "AE-AZ", NameEN: "Abu Dhabi", NameAR: "أبوظبي"},
			{Code: "AE-DU", NameEN: "Dubai", NameAR: "دبي"},
			{Code: "AE-SH", NameEN: "Sharjah", NameAR: "الشارقة"},
			{Code: "AE-AJ", NameEN: "Ajman", NameAR: "عجمان"},
			{Code: "AE-UQ", NameEN: "Umm Al Quwain", NameAR: "أم القيوين"},
			{Code: "AE-RK", NameEN: "Ras Al Khaimah", NameAR: "رأس الخيمة"},
			{Code: "AE-FU", NameEN: "Fujairah", NameAR: "الفجيرة"},
		},
	},
}

// currencyDecimals lists ISO 4217 minor units for the currencies a company may pick,
// independently of its country (a UAE company may price a listing in USD).
var currencyDecimals = map[string]int{
	"AED": 2, "SAR": 2, "QAR": 2, "USD": 2, "EUR": 2, "GBP": 2, "INR": 2, "PKR": 2,
	"BHD": 3, "KWD": 3, "OMR": 3, "JOD": 3,
}

// For returns the rules for an ISO country code (case-insensitive).
func For(country string) (Rules, bool) {
	r, ok := registry[strings.ToUpper(strings.TrimSpace(country))]
	return r, ok
}

// MustFor returns the rules for country, falling back to DefaultCountry when the code is
// empty or unknown. Use it where a value is always needed (formatting, scheduling).
func MustFor(country string) Rules {
	if r, ok := For(country); ok {
		return r
	}
	return registry[DefaultCountry]
}

// Supported lists the countries that have rules, sorted by code.
func Supported() []Rules {
	out := make([]Rules, 0, len(registry))
	for _, code := range sortedKeys() {
		out = append(out, registry[code])
	}
	return out
}

func sortedKeys() []string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// VATRate returns the rate for a category, or an error for an unknown category.
func (r Rules) VATRate(cat VATCategory) (float64, error) {
	rate, ok := r.VATRates[cat]
	if !ok {
		return 0, fmt.Errorf("jurisdiction %s: unknown VAT category %q", r.Country, cat)
	}
	return rate, nil
}

// HasRegion reports whether code is one of the country's regions. An empty code is allowed.
func (r Rules) HasRegion(code string) bool {
	if code == "" {
		return true
	}
	for _, reg := range r.Regions {
		if strings.EqualFold(reg.Code, code) {
			return true
		}
	}
	return false
}

// ValidTaxID reports whether id looks like the country's VAT registration number: the
// right number of digits, spaces and dashes ignored. It does not check that the number
// is registered. An empty id is valid (the business may not be VAT-registered).
func (r Rules) ValidTaxID(id string) bool {
	id = NormalizeTaxID(id)
	if id == "" || r.TaxIDDigits == 0 {
		return true
	}
	if len(id) != r.TaxIDDigits {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// NormalizeTaxID strips spaces and dashes from a tax registration number.
func NormalizeTaxID(id string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(id))
}

// Location loads the country's default timezone, falling back to UTC.
func (r Rules) Location() *time.Location {
	if loc, err := time.LoadLocation(r.Timezone); err == nil {
		return loc
	}
	return time.UTC
}

// IsWeekend reports whether t (in the country's timezone) falls on a weekend day.
func (r Rules) IsWeekend(t time.Time) bool {
	day := t.In(r.Location()).Weekday()
	for _, w := range r.Weekend {
		if w == day {
			return true
		}
	}
	return false
}

// CurrencyDecimals returns the number of minor-unit digits for an ISO 4217 code, and
// whether the currency is known.
func CurrencyDecimals(currency string) (int, bool) {
	d, ok := currencyDecimals[strings.ToUpper(strings.TrimSpace(currency))]
	return d, ok
}

// IsCurrency reports whether currency is a supported ISO 4217 code.
func IsCurrency(currency string) bool {
	_, ok := CurrencyDecimals(currency)
	return ok
}

// RoundMoney rounds amount to the currency's minor units (half away from zero). Unknown
// currencies round to 2 decimals.
func RoundMoney(amount float64, currency string) float64 {
	d, ok := CurrencyDecimals(currency)
	if !ok {
		d = 2
	}
	return RoundTo(amount, d)
}

// RoundTo rounds amount to the given number of decimals, half away from zero. A tiny
// epsilon corrects binary floating-point artefacts such as 1.005 being stored as
// 1.00499999….
func RoundTo(amount float64, decimals int) float64 {
	p := math.Pow10(decimals)
	v := amount * p
	if v >= 0 {
		return math.Floor(v+0.5+1e-9) / p
	}
	return math.Ceil(v-0.5-1e-9) / p
}

// ValidTimezone reports whether tz is a loadable IANA timezone name.
func ValidTimezone(tz string) bool {
	if tz == "" {
		return false
	}
	_, err := time.LoadLocation(tz)
	return err == nil
}
