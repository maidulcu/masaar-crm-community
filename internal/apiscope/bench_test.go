package apiscope

import "testing"

func BenchmarkGrantsSingle(b *testing.B) {
	granted := "leads:read"
	required := "leads:read"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Grants(granted, required)
	}
}

func BenchmarkGrantsMultiple(b *testing.B) {
	granted := "contacts:read, leads:write, listings:read, webhooks:manage"
	required := "lead:create"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Grants(granted, required)
	}
}

func BenchmarkGrantsResourceWildcard(b *testing.B) {
	granted := "leads:*"
	required := "leads:write"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Grants(granted, required)
	}
}

func BenchmarkGrantsWildcard(b *testing.B) {
	granted := "*"
	required := "webhooks:manage"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Grants(granted, required)
	}
}
