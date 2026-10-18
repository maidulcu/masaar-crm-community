package bizhours

import (
	"testing"
	"time"
)

func TestOpenAt(t *testing.T) {
	dubai, _ := time.LoadLocation("Asia/Dubai")
	s := Default([]time.Weekday{time.Saturday, time.Sunday})
	s.Specials = []Special{
		{Label: "Ramadan", From: "2027-02-08", To: "2027-03-09", Span: Span{Open: "09:00", Close: "15:00"}},
		{Label: "Eid", From: "2027-03-10", To: "2027-03-12"},
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	at := func(v string) time.Time {
		tm, err := time.ParseInLocation("2006-01-02 15:04", v, dubai)
		if err != nil {
			t.Fatal(err)
		}
		return tm.UTC()
	}
	for v, want := range map[string]bool{
		"2026-10-14 10:00": true,  // Wednesday
		"2026-10-14 08:59": false, // before opening
		"2026-10-14 18:00": false, // closing time is closed
		"2026-10-17 11:00": false, // Saturday
		"2027-02-10 14:30": true,  // Ramadan hours
		"2027-02-10 16:00": false, // after Ramadan closing
		"2027-03-11 11:00": false, // Eid
	} {
		if got := s.OpenAt(at(v), dubai); got != want {
			t.Errorf("%s: open=%v, want %v", v, got, want)
		}
	}
}

func TestValidate(t *testing.T) {
	bad := []Schedule{
		{Week: [7]Span{{Open: "18:00", Close: "09:00"}}},
		{Week: [7]Span{{Open: "9:00", Close: "17:00"}}},
		{Week: [7]Span{{Open: "09:00", Close: "24:30"}}},
		{Specials: []Special{{Label: "x", From: "2027-03-02", To: "2027-03-01"}}},
	}
	for i, s := range bad {
		if s.Validate() == nil {
			t.Errorf("schedule %d accepted", i)
		}
	}
	ok := Schedule{Week: [7]Span{{Open: "00:00", Close: "24:00"}}}
	if err := ok.Validate(); err != nil {
		t.Error(err)
	}
}
