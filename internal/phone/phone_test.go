package phone

import "testing"

func TestNormalize(t *testing.T) {
	good := map[string]string{
		"+971501234567":       "+971501234567",
		"971501234567":        "+971501234567", // WhatsApp wa_id form
		"00971501234567":      "+971501234567",
		"+971 50 123 4567":    "+971501234567",
		"(971) 50-123-4567":   "+971501234567",
		"  +971.50.123.4567 ": "+971501234567",
		"+44 7911 123456":     "+447911123456",
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
	} {
		if got, ok := Normalize(in); ok {
			t.Errorf("Normalize(%q) = %q, want rejection", in, got)
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
