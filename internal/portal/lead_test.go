package portal

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseLeadShapes(t *testing.T) {
	cases := map[string]struct {
		body string
		want Lead
	}{
		"flat": {`{"id": 991, "name": "Sara", "phone": "+971501112233", "email": "s@x.test", "message": "Is it available?", "reference": "MAS-12"}`,
			Lead{ExternalID: "991", Name: "Sara", Phone: "+971501112233", Email: "s@x.test", Message: "Is it available?", ListingReference: "MAS-12"}},
		"nested": {`{"lead": {"lead_id": "pf-77", "contact": {"first_name": "Omar", "last_name": "Ali", "mobile": "0501234567"}, "property": {"reference_number": "R-9"}}}`,
			Lead{ExternalID: "pf-77", Name: "Omar Ali", Phone: "0501234567", ListingReference: "R-9"}},
	}
	for name, c := range cases {
		got, err := ParseLead([]byte(c.body))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if *got != c.want {
			t.Errorf("%s: got %+v, want %+v", name, *got, c.want)
		}
	}
}

func TestParseLeadFallbacks(t *testing.T) {
	body := []byte(`{"data": {"phone_number": "+971500000001"}}`)
	a, err := ParseLead(body)
	if err != nil || !strings.HasPrefix(a.ExternalID, "sha256:") || a.Name != "+971500000001" {
		t.Fatalf("fallbacks = %+v, %v", a, err)
	}
	b, _ := ParseLead(body)
	if a.ExternalID != b.ExternalID {
		t.Fatal("same body gave different ids")
	}
	if _, err := ParseLead([]byte(`{"name": "No phone"}`)); !errors.Is(err, ErrNoContact) {
		t.Fatalf("missing phone: %v", err)
	}
	if _, err := ParseLead([]byte(`not json`)); err == nil {
		t.Fatal("invalid JSON accepted")
	}
}

func TestParseLeadLimitsKeepValidUTF8(t *testing.T) {
	long := strings.Repeat("م", 2500) // 2 bytes per letter
	body, _ := json.Marshal(map[string]string{
		"id": strings.Repeat("x", 3000), "name": strings.Repeat("س", 300), "phone": "+971501234567",
		"email": "not an email", "message": long, "reference": strings.Repeat("r", 500),
	})
	l, err := ParseLead(body)
	if err != nil {
		t.Fatal(err)
	}
	for field, v := range map[string]string{"name": l.Name, "message": l.Message, "reference": l.ListingReference} {
		if !utf8.ValidString(v) {
			t.Errorf("%s is not valid UTF-8 after truncation", field)
		}
	}
	if n := utf8.RuneCountInString(l.Message); n != 2000 {
		t.Errorf("message = %d characters, want 2000", n)
	}
	if n := utf8.RuneCountInString(l.Name); n != 200 {
		t.Errorf("name = %d characters, want 200", n)
	}
	if len(l.ListingReference) != 128 || len(l.ExternalID) > maxExternalIDLen || !strings.HasPrefix(l.ExternalID, "sha256:") {
		t.Errorf("reference %d bytes, external id %q", len(l.ListingReference), l.ExternalID)
	}
	if l.Email != "" {
		t.Errorf("invalid email kept: %q", l.Email)
	}
	// The same oversized id maps to the same ref, so a redelivery is still recognised.
	if again, _ := ParseLead(body); again.ExternalID != l.ExternalID {
		t.Error("oversized id does not hash stably")
	}
}
