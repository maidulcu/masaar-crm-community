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
func Normalize(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	// E.164 allows at most 15 digits. Raw inputs starting with "00" prefix
	// can have up to 17 digits total before "00" prefix stripping.
	var buf [20]byte
	n := 0

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
		} else if c >= 0x80 {
			r, w := utf8.DecodeRuneInString(trimmed[i:])
			if r == ' ' { // Non-breaking space U+00A0
				i += w
			} else {
				return "", false
			}
		} else {
			return "", false
		}
	}

	digits := buf[:n]
	if n >= 2 && digits[0] == '0' && digits[1] == '0' {
		digits = digits[2:]
	}

	if len(digits) < 7 || len(digits) > 15 || digits[0] == '0' {
		return "", false
	}

	// Performance optimization: construct "+"+digits with 1 allocation
	out := make([]byte, 1+len(digits))
	out[0] = '+'
	copy(out[1:], digits)
	return string(out), true
}
