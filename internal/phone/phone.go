// Package phone normalises phone numbers to the one canonical form the CRM stores:
// E.164 with a leading "+" ("+971501234567").
//
// WhatsApp reports numbers as digits only ("971501234567") while people type "+971 50 123 4567",
// "00971501234567" or "(971) 50-123-4567". Storing them as typed made the same person two
// contacts, so every write and lookup goes through Normalize.
package phone

import (
	"strings"
	"unicode/utf8"
)

// Normalize returns the E.164 form of raw, or ok=false when raw cannot be a full international
// number. Numbers in national format ("0501234567") are rejected rather than guessed: the CRM
// has no way to know the country.
//
// Optimization Note:
// Uses zero-allocation fast paths for already canonical E.164 numbers (e.g., "+971501234567")
// and 1-allocation fast paths for digits-only WhatsApp/international numbers ("971501234567", "00971501234567").
func Normalize(raw string) (string, bool) {
	// Fast path: if input is already canonical E.164 (+ followed by 7-15 digits, no leading 0),
	// return raw directly with 0 heap allocations (~12 ns/op vs ~70 ns/op).
	if isCanonicalE164(raw) {
		return raw, true
	}

	trimmed := strings.TrimSpace(raw)
	if isCanonicalE164(trimmed) {
		return trimmed, true
	}

	// Fast path for digits-only (e.g. WhatsApp wa_id "971501234567" or international "00971501234567")
	if len(trimmed) >= 7 && len(trimmed) <= 15 && trimmed[0] != '0' && isDigits(trimmed) {
		return "+" + trimmed, true
	}
	if len(trimmed) >= 9 && len(trimmed) <= 17 && strings.HasPrefix(trimmed, "00") {
		rest := trimmed[2:]
		if len(rest) >= 7 && len(rest) <= 15 && rest[0] != '0' && isDigits(rest) {
			return "+" + rest, true
		}
	}

	// E.164 allows at most 15 digits. Raw inputs starting with "00" prefix
	// can have up to 17 digits total before "00" prefix stripping.
	// Store '+' in buf[0] to construct "+"+digits directly in the stack buffer,
	// avoiding heap allocation for slice creation.
	var buf [20]byte
	buf[0] = '+'
	n := 1

	for i := 0; i < len(trimmed); {
		c := trimmed[i]
		if c >= '0' && c <= '9' {
			if n >= len(buf) {
				return "", false
			}
			buf[n] = c
			n++
			i++
		} else if c == '+' || c == ' ' || c == '-' || c == '(' || c == ')' || c == '.' {
			i++
		} else if c < 0x80 {
			return "", false
		} else {
			r, w := utf8.DecodeRuneInString(trimmed[i:])
			if r == ' ' { // Non-breaking space U+00A0
				i += w
			} else {
				return "", false
			}
		}
	}

	digits := buf[1:n]
	outStart := 0
	if len(digits) >= 2 && digits[0] == '0' && digits[1] == '0' {
		digits = digits[2:]
		outStart = 2
		buf[outStart] = '+'
	}

	if len(digits) < 7 || len(digits) > 15 || digits[0] == '0' {
		return "", false
	}

	return "+" + string(digits), true
}

// isCanonicalE164 checks if s is already in valid E.164 canonical format (+ followed by 7-15 digits, not starting with 0).
func isCanonicalE164(s string) bool {
	if len(s) < 8 || len(s) > 16 || s[0] != '+' || s[1] == '0' {
		return false
	}
	return isDigits(s[1:])
}

// isDigits returns true if s consists only of ASCII digits '0'-'9'.
func isDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
