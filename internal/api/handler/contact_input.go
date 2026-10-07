package handler

import (
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	maxContactNameRunes = 200
	maxContactEmail     = 254
)

// validateContactFields trims and checks the editable contact fields in place and returns a
// message for the client, or "" when they are acceptable. The language column is varchar(2) and the
// score a smallint: bad values used to reach the database and come back as a 500.
func validateContactFields(name, email, language *string, score *int) string {
	*name = strings.TrimSpace(*name)
	if *name == "" {
		return "full_name cannot be empty"
	}
	if utf8.RuneCountInString(*name) > maxContactNameRunes {
		return "full_name is too long"
	}

	*email = strings.TrimSpace(*email)
	if *email != "" {
		if len(*email) > maxContactEmail {
			return "email is too long"
		}
		if addr, err := mail.ParseAddress(*email); err != nil || addr.Address != *email {
			return "email is not a valid address"
		}
	}

	*language = strings.ToLower(strings.TrimSpace(*language))
	if *language != "ar" && *language != "en" {
		return "language must be ar or en"
	}

	if *score < 0 || *score > 100 {
		return "lead_score must be between 0 and 100"
	}
	return ""
}
