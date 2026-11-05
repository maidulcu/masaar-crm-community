package phone

import (
	"regexp"
	"testing"
)

var benchE164Re = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

func BenchmarkE164Regex(b *testing.B) {
	in := "+971501234567"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchE164Re.MatchString(in)
	}
}

func BenchmarkIsCanonicalE164(b *testing.B) {
	in := "+971501234567"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsCanonicalE164(in)
	}
}

func BenchmarkNormalize(b *testing.B) {
	inputs := []string{
		"+971501234567",
		"971501234567",
		"00971501234567",
		"+971 50 123 4567",
		"(971) 50-123-4567",
		"  +971.50.123.4567 ",
		"+44 7911 123456",
		"invalid_phone_number_here",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, in := range inputs {
			Normalize(in)
		}
	}
}

func BenchmarkNormalizeCanonical(b *testing.B) {
	in := "+971501234567"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Normalize(in)
	}
}

func BenchmarkNormalizeWhatsAppDigits(b *testing.B) {
	in := "971501234567"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Normalize(in)
	}
}

func BenchmarkNormalizeFormatted(b *testing.B) {
	in := "+971 50 123 4567"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Normalize(in)
	}
}
