package phone

import "testing"

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
