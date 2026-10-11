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
	// Fast path 1: Already canonical E.164 ("+971501234567").
	// E.164 length is 1 '+' plus 7..15 digits = 8..16 bytes. Zero allocations.
	if len(raw) >= 8 && len(raw) <= 16 && raw[0] == '+' && raw[1] >= '1' && raw[1] <= '9' {
		isCanonical := true
		for i := 2; i < len(raw); i++ {
			c := raw[i]
			if c < '0' || c > '9' {
				isCanonical = false
				break
			}
		}
		if isCanonical {
			return raw, true
		}
	}

	// Fast path 2: Pure digits without leading zero ("971501234567").
	// E.164 allows 7..15 digits. Exactly 1 allocation ("+" + raw).
	if len(raw) >= 7 && len(raw) <= 15 && raw[0] >= '1' && raw[0] <= '9' {
		isDigits := true
		for i := 1; i < len(raw); i++ {
			c := raw[i]
			if c < '0' || c > '9' {
				isDigits = false
				break
			}
		}
		if isDigits {
			return "+" + raw, true
		}
	}

	// Fast path 3: Pure digits starting with "00" prefix ("00971501234567").
	// "00" prefix + 7..15 digits = 9..17 bytes. Exactly 1 allocation ("+" + raw[2:]).
	if len(raw) >= 9 && len(raw) <= 17 && raw[0] == '0' && raw[1] == '0' && raw[2] >= '1' && raw[2] <= '9' {
		isDigits := true
		for i := 3; i < len(raw); i++ {
			c := raw[i]
			if c < '0' || c > '9' {
				isDigits = false
				break
			}
		}
		if isDigits {
			return "+" + raw[2:], true
		}
	}

	// General path: parse formatted phone number using stack-allocated buffer.
	trimmed := strings.TrimSpace(raw)
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

	digits := buf[1:n]
	if len(digits) >= 2 && digits[0] == '0' && digits[1] == '0' {
		digits = digits[2:]
		buf[2] = '+'
		if len(digits) < 7 || len(digits) > 15 || digits[0] == '0' {
			return "", false
		}
		return string(buf[2:n]), true
	}

	if len(digits) < 7 || len(digits) > 15 || digits[0] == '0' {
		return "", false
	}

	return string(buf[:n]), true
}
