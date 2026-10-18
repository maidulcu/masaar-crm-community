package handler

import (
	"context"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
)

// companyLocale returns the locale of the company bound to ctx (set by UseCompanyLocale).
var companyLocale func(ctx context.Context) (*domain.CompanyLocale, error)

// UseCompanyLocale lets handlers default amounts to the company's own currency instead
// of assuming AED.
func UseCompanyLocale(f func(ctx context.Context) (*domain.CompanyLocale, error)) {
	companyLocale = f
}

// defaultCurrency is the currency of the company bound to ctx.
func defaultCurrency(ctx context.Context) string {
	if companyLocale != nil {
		if l, err := companyLocale(ctx); err == nil && l.Currency != "" {
			return l.Currency
		}
	}
	return jurisdiction.MustFor(jurisdiction.DefaultCountry).Currency
}

// defaultCountry is the country of the company bound to ctx.
func defaultCountry(ctx context.Context) string {
	if companyLocale != nil {
		if l, err := companyLocale(ctx); err == nil && l.Country != "" {
			return l.Country
		}
	}
	return jurisdiction.DefaultCountry
}
