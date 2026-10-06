package handler

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSafeFilenamePart(t *testing.T) {
	cases := []struct{ in, want string }{
		{"REF-123", "REF-123"},
		{`x".pdf"; filename="evil.exe`, "x_.pdf___filename__evil.exe"},
		{"a\r\nSet-Cookie: x=1", "a__Set-Cookie__x_1"},
		{"../../etc/passwd", "etc_passwd"},
		{"", "fallback"},
		{"...", "fallback"},
	}
	for _, tc := range cases {
		got := safeFilenamePart(tc.in, "fallback")
		if got != tc.want {
			t.Errorf("safeFilenamePart(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if strings.ContainsAny(got, "\"\r\n/;") || strings.Contains(got, `\`) {
			t.Errorf("safeFilenamePart(%q) = %q still contains unsafe characters", tc.in, got)
		}
	}
}

func TestCampaignHTMLHelpersEscape(t *testing.T) {
	if got := coverImg(`https://cdn.example.com/a.jpg" onerror="alert(1)`); strings.Contains(got, `" onerror="`) {
		t.Errorf("coverImg did not escape attribute: %s", got)
	}
	if got := coverImg("javascript:alert(1)"); got != "" {
		t.Errorf("coverImg accepted non-http(s) URL: %s", got)
	}
	if got := coverImg(""); got != "" {
		t.Errorf("coverImg(\"\") = %q", got)
	}
	if got := refLink(`<script>alert(1)</script>`); strings.Contains(got, "<script>") {
		t.Errorf("refLink did not escape: %s", got)
	}
}

func TestPublicListingURLIgnoresRequestHost(t *testing.T) {
	h := NewMarketingHandler(nil, nil, nil, nil, nil, "https://crm.example.ae/")
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	if got, want := h.publicListingURL(id), "https://crm.example.ae/l/11111111-2222-3333-4444-555555555555"; got != want {
		t.Fatalf("publicListingURL = %q, want %q", got, want)
	}
}
