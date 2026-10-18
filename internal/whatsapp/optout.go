package whatsapp

import (
	"strings"
	"unicode"
)

// Replies that opt a contact out of, or back into, WhatsApp messages. Matching is on the
// whole message, so "please stop calling me" is a message for an agent, not an opt-out.
var (
	optOutWords = map[string]bool{
		"stop": true, "stop all": true, "unsubscribe": true, "opt out": true, "optout": true,
		"ايقاف": true, "توقف": true, "الغاء الاشتراك": true,
	}
	optInWords = map[string]bool{
		"start": true, "subscribe": true, "opt in": true, "unstop": true,
		"اشتراك": true, "ابدا": true,
	}
)

// normalizeReply lowercases, folds Arabic letter variants (إ/أ/آ → ا, ى → ي, ة → ه),
// drops diacritics, punctuation and emoji, and collapses spaces.
func normalizeReply(s string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch r {
		case 'إ', 'أ', 'آ', 'ٱ':
			r = 'ا'
		case 'ى':
			r = 'ي'
		case 'ة':
			r = 'ه'
		}
		switch {
		case unicode.Is(unicode.Mn, r): // Arabic diacritics (tashkeel)
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
		default:
			space = true
		}
	}
	return b.String()
}

// OptKeyword reports whether a message is an opt-out ("STOP", "إيقاف") or opt-in
// ("START", "اشتراك") reply.
func OptKeyword(body string) (optOut, optIn bool) {
	n := normalizeReply(body)
	return optOutWords[n], optInWords[n]
}
