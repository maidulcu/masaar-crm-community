package phone

import "testing"

func TestNormalize(t *testing.T) {
	good := map[string]string{
		"+971501234567":         "+971501234567",
		"971501234567":          "+971501234567", // WhatsApp wa_id form
		"00971501234567":        "+971501234567",
		"0044791112345678":      "+44791112345678", // 16 digits raw (00 + 14 digits)
		"+971 50 123 4567":      "+971501234567",
		"(971) 50-123-4567":     "+971501234567",
		"  +971.50.123.4567 ":   "+971501234567",
		"+44 7911 123456":       "+447911123456",
		"+971\u00A050\u00A0123": "+97150123", // Non-breaking space
	}
	for in, want := range good {
		if got, ok := Normalize(in); !ok || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{
		"", "abc", "0501234567", // national format: country unknown
		"+0501234567", "12345", "+1234567890123456", // too short / too long
		"+971-50-abc", "971 50 123 4567 ext 2",
		"+971\t501234567", "+971\n501234567", // tabs and newlines rejected
	} {
		if got, ok := Normalize(in); ok {
			t.Errorf("Normalize(%q) = %q, want rejection", in, got)
		}
	}
}

func TestIsCanonicalE164(t *testing.T) {
	valid := []string{
		"+971501234567",
		"+1234567",
		"+123456789012345",
	}
	invalid := []string{
		"",
		"971501234567",
		"+0501234567",
		"+123456",           // 6 digits - too short
		"+1234567890123456", // 16 digits - too long
		"+971 501234567",
		"+971a501234567",
	}

	for _, s := range valid {
		if !IsCanonicalE164(s) {
			t.Errorf("IsCanonicalE164(%q) = false, want true", s)
		}
	}

	for _, s := range invalid {
		if IsCanonicalE164(s) {
			t.Errorf("IsCanonicalE164(%q) = true, want false", s)
		}
	}
}

func TestNormalizeIsIdempotent(t *testing.T) {
	once, _ := Normalize("00971 50 123 4567")
	twice, ok := Normalize(once)
	if !ok || once != twice {
		t.Errorf("not idempotent: %q -> %q", once, twice)
	}
}
