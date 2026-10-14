// Package compliance holds UAE regulatory checks that do not belong to a single
// handler: listing advertising permits, licence expiry.
//
// These checks support the brokerage's own compliance process; they are not legal
// advice and do not verify anything with the regulator.
package compliance

import (
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
)

// ListingPermit evaluates a listing's advertising permit under its company's country
// rules, on the day `now` falls on in the country's timezone.
func ListingPermit(l *domain.Listing, now time.Time) *domain.PermitStatus {
	rules := jurisdiction.MustFor(l.Country)
	st := &domain.PermitStatus{Valid: true, Overridden: l.PermitOverrideBy != nil}
	reg, ok := rules.FindRegion(l.Emirate)
	if !ok || reg.ListingPermit == "" {
		return st
	}
	st.Required, st.System = true, reg.ListingPermit
	switch {
	case l.PermitNumber == "":
		st.Valid, st.Problem = false, "missing"
	case l.PermitExpiry != nil && dateOnly(*l.PermitExpiry).Before(dateOnly(now.In(rules.Location()))):
		st.Valid, st.Problem = false, "expired"
	}
	return st
}

// Decorate sets l.Permit for API responses.
func Decorate(l *domain.Listing, now time.Time) {
	l.Permit = ListingPermit(l, now)
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
