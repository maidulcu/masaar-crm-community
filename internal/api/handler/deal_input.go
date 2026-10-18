package handler

import (
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

const maxDealTitleRunes = 200

// parseCloseDate accepts the plain date the forms use ("2026-10-31") as well as the RFC 3339
// timestamp the API itself returns for close_date. The deal page used to send back the value it
// had just been given, which the date-only parser rejected, so a deal with a close date could not
// be edited. An empty string means "no close date" (nil).
func parseCloseDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return &d, nil
	}
	return nil, errors.New("invalid close_date format, expected YYYY-MM-DD")
}

// validateDealTitle trims a title and checks its length ("" and over-long titles are refused).
func validateDealTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", errors.New("title is required")
	}
	if utf8.RuneCountInString(title) > maxDealTitleRunes {
		return "", errors.New("title is too long")
	}
	return title, nil
}

// validateDealAmount checks an amount fits the numeric(12,2) column and rounds it to whole fils.
func validateDealAmount(a float64) (float64, error) {
	if math.IsNaN(a) || math.IsInf(a, 0) || a < 0 || a > maxDealValue {
		return 0, errors.New("amount must be between 0 and 9999999999.99")
	}
	return math.Round(a*100) / 100, nil
}
