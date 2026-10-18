package apiscope

import "testing"

func TestNormalize(t *testing.T) {
	got, err := Normalize("leads:write, contacts:read leads:write")
	if err != nil || got != "contacts:read,leads:write" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", " , ", "*", "leads:*", "leads:delete", "admin"} {
		if _, err := Normalize(bad); err == nil {
			t.Errorf("Normalize(%q) accepted", bad)
		}
	}
}

func TestGrants(t *testing.T) {
	cases := []struct {
		granted, required string
		want              bool
	}{
		{"leads:read", "leads:read", true},
		{"leads:read", "leads:write", false},
		{"contacts:read,leads:write", "lead:create", true}, // implied
		{"lead:create", "leads:write", false},              // not the other way round
		{"lead:create", "leads:read", false},
		{"*", "webhooks:manage", true},   // legacy wildcard
		{"leads:*", "leads:write", true}, // legacy resource wildcard
		{"leads:*", "listings:read", false},
		{"", "leads:read", false},
		{" leads:read ", "leads:read", true},
	}
	for _, c := range cases {
		if got := Grants(c.granted, c.required); got != c.want {
			t.Errorf("Grants(%q, %q) = %v, want %v", c.granted, c.required, got, c.want)
		}
	}
}
