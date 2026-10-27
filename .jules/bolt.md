## 2025-05-10 - Zero-Allocation Route Lookup in Go Middleware
**Learning:** Fiber/Go HTTP middleware matching target URL paths against candidate route tables can suffer from heap slice allocations inside nested matching loops if route paths are split dynamically (`strings.Split`). Pre-compiling route path segments during initialization and using a stack-allocated buffer array (`[16]string`) for request path segment splitting eliminates heap allocations on hot path API key scope lookups (0 B/op, 0 allocs/op).
**Action:** Always pre-segment static route tables at middleware initialization time, and use stack buffers for splitting incoming HTTP path segments when path depth is bounded.

## 2026-10-26 - Non-Regex Byte Inspection for Hot-Path Middleware Path Parameters
**Learning:** Go's `regexp` engine adds significant CPU overhead per match (~500+ ns) even for compiled regular expressions due to DFA state machine traversal. In hot-path HTTP middleware route lookups, replacing regex URL segment matching (e.g., UUID validation) with direct byte scanning reduces route scope resolution latency by over 75% (721 ns -> 169 ns).
**Action:** Avoid `regexp.MatchString` inside hot-path middleware or routing functions; use direct length and byte inspection helpers instead.

## 2026-10-27 - Fast Non-Regex E.164 Validation in API Handlers
**Learning:** Validating input phone numbers against `regexp.MustCompile(`^\+[1-9]\d{6,14}$`)` in high-throughput handlers (e.g. public lead submission and contact creation) takes ~285 ns/op. Replacing regex match with a direct byte-inspection function (`phone.IsE164`) reduces validation latency to ~10.7 ns/op (~96% speedup, 0 B/op, 0 allocs/op).
**Action:** Replace `regexp.MatchString` for common canonical string shapes (e.g., E.164, UUIDs) with fast byte/length inspection helpers.
