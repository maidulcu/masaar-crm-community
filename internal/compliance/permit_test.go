package compliance

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestListingPermit(t *testing.T) {
	today := time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	sameDay := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	admin := uuid.New()

	cases := []struct {
		name         string
		l            domain.Listing
		required     bool
		valid        bool
		problem      string
		advertisable bool
	}{
		{"sharjah needs none", domain.Listing{Emirate: "Sharjah"}, false, true, "", true},
		{"unknown emirate", domain.Listing{Emirate: ""}, false, true, "", true},
		{"dubai missing", domain.Listing{Emirate: "Dubai"}, true, false, "missing", false},
		{"dubai valid, no expiry", domain.Listing{Emirate: "AE-DU", PermitNumber: "7114567890"}, true, true, "", true},
		{"dubai expires today is still valid", domain.Listing{Emirate: "Dubai", PermitNumber: "1", PermitExpiry: &sameDay}, true, true, "", true},
		{"dubai expired", domain.Listing{Emirate: "Dubai", PermitNumber: "1", PermitExpiry: &yesterday}, true, false, "expired", false},
		{"abu dhabi missing but overridden", domain.Listing{Emirate: "أبوظبي", PermitOverrideBy: &admin}, true, false, "missing", true},
	}
	for _, c := range cases {
		st := ListingPermit(&c.l, today)
		if st.Required != c.required || st.Valid != c.valid || st.Problem != c.problem || st.Advertisable() != c.advertisable {
			t.Errorf("%s: got %+v (advertisable %v)", c.name, st, st.Advertisable())
		}
	}
}
