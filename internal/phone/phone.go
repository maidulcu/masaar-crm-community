// Package phone normalises phone numbers to the one canonical form the CRM stores:
// E.164 with a leading "+" ("+971501234567").
//
// WhatsApp reports numbers as digits only ("971501234567") while people type "+971 50 123 4567",
// "00971501234567" or "(971) 50-123-4567". Storing them as typed made the same person two
// contacts, so every write and lookup goes through Normalize.
package phone

import "strings"

// Normalize returns the E.164 form of raw, or ok=false when raw cannot be a full international
// number. Numbers in national format ("0501234567") are rejected rather than guessed: the CRM
// has no way to know the country.
func Normalize(raw string) (string, bool) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')' || r == '.' || r == ' ':
			// formatting characters are ignored
		default:
			return "", false
		}
	}
	digits := b.String()
	// "00" is the international call prefix in most countries ("00971…" == "+971…").
	if strings.HasPrefix(digits, "00") {
		digits = digits[2:]
	}
	// E.164: country code (no leading zero) + subscriber number, at most 15 digits.
	if len(digits) < 7 || len(digits) > 15 || digits[0] == '0' {
		return "", false
	}
	return "+" + digits, true
}
