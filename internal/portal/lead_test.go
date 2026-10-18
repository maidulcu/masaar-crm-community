package portal

import (
	"errors"
	"strings"
	"testing"
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
