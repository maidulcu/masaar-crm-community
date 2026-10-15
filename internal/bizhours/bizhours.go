// Package bizhours decides whether a business is open at a given moment, from weekly
// opening hours and dated special periods (Ramadan hours, public holidays).
package bizhours

import (
	"errors"
	"fmt"
	"time"
)

// Span is one day's opening hours, "HH:MM" in the business's timezone. Close may be
// "24:00". An empty Open means closed all day.
type Span struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

// Special overrides the weekly hours between two dates (inclusive), such as Ramadan
// hours or a holiday (empty Open: closed).
type Special struct {
	Label string `json:"label"`
	From  string `json:"from"` // YYYY-MM-DD
	To    string `json:"to"`   // YYYY-MM-DD
	Span
}

// Schedule is a business's opening hours. Week is indexed by time.Weekday (0 = Sunday).
type Schedule struct {
	Week     [7]Span   `json:"week"`
	Specials []Special `json:"specials"`
}

// Default returns 09:00–18:00 on working days and closed on the given weekend days.
func Default(weekend []time.Weekday) Schedule {
	var s Schedule
	for d := range s.Week {
		s.Week[d] = Span{Open: "09:00", Close: "18:00"}
	}
	for _, d := range weekend {
		s.Week[d] = Span{}
	}
	return s
}

func minutes(hhmm string) (int, error) {
	var h, m int
	if _, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m); err != nil || len(hhmm) != 5 || h < 0 || m < 0 || m > 59 || h > 24 || (h == 24 && m != 0) {
		return 0, fmt.Errorf("bad time %q", hhmm)
	}
	return h*60 + m, nil
}

func (sp Span) validate() error {
	if sp.Open == "" && sp.Close == "" {
		return nil
	}
	o, err := minutes(sp.Open)
	if err != nil {
		return err
	}
	c, err := minutes(sp.Close)
	if err != nil {
		return err
	}
	if c <= o {
		return errors.New("closing time must be after opening time")
	}
	return nil
}

// Validate checks every time and date in the schedule.
func (s Schedule) Validate() error {
	for d, sp := range s.Week {
		if err := sp.validate(); err != nil {
			return fmt.Errorf("%s: %w", time.Weekday(d), err)
		}
	}
	if len(s.Specials) > 50 {
		return errors.New("too many special periods")
	}
	for _, sp := range s.Specials {
		from, err1 := time.Parse("2006-01-02", sp.From)
		to, err2 := time.Parse("2006-01-02", sp.To)
		if err1 != nil || err2 != nil || to.Before(from) {
			return fmt.Errorf("%s: dates must be YYYY-MM-DD, from before to", sp.Label)
		}
		if err := sp.Span.validate(); err != nil {
			return fmt.Errorf("%s: %w", sp.Label, err)
		}
	}
	return nil
}

// SpanAt returns the hours that apply on the local date of t: a special period's, else
// the weekday's.
func (s Schedule) SpanAt(t time.Time) Span {
	date := t.Format("2006-01-02")
	for _, sp := range s.Specials {
		if date >= sp.From && date <= sp.To {
			return sp.Span
		}
	}
	return s.Week[t.Weekday()]
}

// OpenAt reports whether the business is open at t, read in loc.
func (s Schedule) OpenAt(t time.Time, loc *time.Location) bool {
	t = t.In(loc)
	sp := s.SpanAt(t)
	if sp.Open == "" {
		return false
	}
	o, err1 := minutes(sp.Open)
	c, err2 := minutes(sp.Close)
	if err1 != nil || err2 != nil {
		return true // a broken schedule never silences the inbox
	}
	now := t.Hour()*60 + t.Minute()
	return now >= o && now < c
}
