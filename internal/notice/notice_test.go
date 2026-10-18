package notice

import (
	"strings"
	"testing"
	"time"
)

func TestRentChangeDraft(t *testing.T) {
	in := Input{Kind: RentChange, Today: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), TenantName: "Omar", PropertyName: "Marina Tower",
		UnitNumber: "1204", EjariNumber: "EJ-1", CompanyName: "Masaar Realty", EndDate: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Currency: "AED", CurrentAnnual: 120000, NewAnnual: 126000, NoticeDays: 90}
	en, ar := Draft(in)
	for _, want := range []string{"AED 120,000.00", "AED 126,000.00", "1 January 2027", "31 December 2026", "unit 1204", "EJ-1", "90 days", "Masaar Realty"} {
		if !strings.Contains(en, want) {
			t.Errorf("English draft lacks %q:\n%s", want, en)
		}
	}
	for _, want := range []string{"AED 126,000.00", "2027-01-01", "وحدة 1204", "90"} {
		if !strings.Contains(ar, want) {
			t.Errorf("Arabic draft lacks %q", want)
		}
	}
	if money("KWD", 1234567.5) != "KWD 1,234,567.500" {
		t.Error(money("KWD", 1234567.5))
	}
	if !ValidKind(NonRenewal) || ValidKind("eviction") {
		t.Error("ValidKind")
	}
}
