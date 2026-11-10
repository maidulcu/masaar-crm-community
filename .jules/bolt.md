## 2025-05-10 - Zero-Allocation Route Lookup in Go Middleware
**Learning:** Fiber/Go HTTP middleware matching target URL paths against candidate route tables can suffer from heap slice allocations inside nested matching loops if route paths are split dynamically (`strings.Split`). Pre-compiling route path segments during initialization and using a stack-allocated buffer array (`[16]string`) for request path segment splitting eliminates heap allocations on hot path API key scope lookups (0 B/op, 0 allocs/op).
**Action:** Always pre-segment static route tables at middleware initialization time, and use stack buffers for splitting incoming HTTP path segments when path depth is bounded.

## 2026-10-26 - Non-Regex Byte Inspection for Hot-Path Middleware Path Parameters
**Learning:** Go's `regexp` engine adds significant CPU overhead per match (~500+ ns) even for compiled regular expressions due to DFA state machine traversal. In hot-path HTTP middleware route lookups, replacing regex URL segment matching (e.g., UUID validation) with direct byte scanning reduces route scope resolution latency by over 75% (721 ns -> 169 ns).
**Action:** Avoid `regexp.MatchString` inside hot-path middleware or routing functions; use direct length and byte inspection helpers instead.

## 2026-10-30 - Zero-Allocation Scope Verification in Middleware
**Learning:** Replacing `strings.Fields` in hot-path scope authorization middleware (`RequirePartnerScope`) with space-delimited substring scanning eliminates heap slice allocations (1 alloc/op -> 0 allocs/op) and reduces scope verification latency by over 80% (159 ns/op -> 28.5 ns/op).
**Action:** Avoid `strings.Fields` or `strings.Split` in HTTP middleware; use zero-allocation substring indexing loops (`strings.IndexByte`) and string slicing instead.

## 2026-11-04 - Zero-Allocation Bearer Token Extraction in Fiber Middleware
**Learning:** In hot-path middleware executed on every authenticated request (e.g., JWT blacklist checking, API key checks, token extraction), parsing the `Authorization` header using `strings.SplitN` and `strings.ToLower` creates heap slice allocations and allocates lowercase string copies (40 B/op, 2 allocs/op). Checking `strings.EqualFold` on the fixed prefix slice and returning the token slice directly reduces per-request latency by over 71% (210 ns -> 60 ns) and achieves 0 B/op, 0 allocs/op.
**Action:** Use direct bounds checking and `strings.EqualFold` on the prefix of header values instead of `strings.SplitN` + `strings.ToLower` on hot request authentication paths.
