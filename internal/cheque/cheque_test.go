package cheque

import (
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestLeaseMonths(t *testing.T) {
	for _, c := range []struct {
		start, end string
		want       int
	}{
		{"2026-01-01", "2026-12-31", 12},
		{"2026-01-15", "2027-01-14", 12},
		{"2026-03-01", "2026-08-31", 6},
		{"2026-01-31", "2027-01-30", 12},
		{"2026-01-01", "2026-12-20", 12}, // 20 days of December count as a month
		{"2026-01-01", "2026-12-10", 11},
	} {
		if got := LeaseMonths(d(c.start), d(c.end)); got != c.want {
			t.Errorf("%s–%s: %d months, want %d", c.start, c.end, got, c.want)
		}
	}
}

func TestTwelveChequeSchedule(t *testing.T) {
	s, err := Schedule(d("2026-01-31"), d("2027-01-30"), time.Time{}, 10_000, 12, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 12 {
		t.Fatalf("%d cheques", len(s))
	}
	want := []string{"2026-01-31", "2026-02-28", "2026-03-31", "2026-04-30"}
	for i, w := range want {
		if got := s[i].Date.Format("2006-01-02"); got != w {
			t.Errorf("cheque %d dated %s, want %s", i+1, got, w)
		}
	}
	sum := 0.0
	for _, p := range s {
		if p.Amount != 10_000 {
			t.Errorf("amount %v", p.Amount)
		}
		sum += p.Amount
	}
	if sum != 120_000 {
		t.Errorf("total %v", sum)
	}
}

func TestScheduleRoundingAndSpacing(t *testing.T) {
	s, err := Schedule(d("2026-01-01"), d("2026-12-31"), d("2026-01-05"), 8333.33, 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	dates := []string{"2026-01-05", "2026-04-05", "2026-07-05", "2026-10-05"}
	total := 0.0
	for i, p := range s {
		if p.Date.Format("2006-01-02") != dates[i] {
			t.Errorf("cheque %d dated %s", i+1, p.Date.Format("2006-01-02"))
		}
		total += p.Amount
	}
	if s[0].Amount != 24999.99 || s[3].Amount != 24999.99 {
		t.Errorf("amounts %+v", s)
	}
	if got := int64(total*100 + 0.5); got != 9999996 {
		t.Errorf("total %v, want 99999.96", total)
	}
	// 3-decimal currency, uneven split: the last cheque takes the remainder.
	s, _ = Schedule(d("2026-01-01"), d("2026-12-31"), time.Time{}, 100, 7, 3)
	if s[0].Amount != 171.429 || s[6].Amount != 171.426 {
		t.Errorf("BHD split %v … %v", s[0].Amount, s[6].Amount)
	}
	if _, err := Schedule(d("2026-01-01"), d("2026-12-31"), time.Time{}, 100, 0, 2); err == nil {
		t.Error("zero cheques accepted")
	}
	if _, err := Schedule(d("2026-01-01"), d("2026-01-05"), time.Time{}, 100, 1, 2); err == nil {
		t.Error("a five-day lease accepted")
	}
}

func TestTransitions(t *testing.T) {
	ok := [][2]string{{Received, Deposited}, {Deposited, Cleared}, {Deposited, Bounced}, {Bounced, Replaced}, {Bounced, Deposited}, {OnHold, Received}}
	for _, c := range ok {
		if !CanTransition(c[0], c[1]) {
			t.Errorf("%s → %s refused", c[0], c[1])
		}
	}
	bad := [][2]string{{Received, Cleared}, {Cleared, Bounced}, {Replaced, Received}, {Returned, Deposited}, {Received, Bounced}}
	for _, c := range bad {
		if CanTransition(c[0], c[1]) {
			t.Errorf("%s → %s allowed", c[0], c[1])
		}
	}
	if !Final(Cleared) || Final(Bounced) {
		t.Error("Final")
	}
}

func TestCountFor(t *testing.T) {
	for _, c := range []struct {
		freq   string
		months int
		want   int
	}{{"monthly", 12, 12}, {"quarterly", 12, 4}, {"semi_annual", 12, 2}, {"annual", 12, 1}, {"monthly", 36, 24}, {"weekly", 12, 1}, {"quarterly", 2, 1}} {
		if got := CountFor(c.freq, c.months); got != c.want {
			t.Errorf("CountFor(%s, %d) = %d, want %d", c.freq, c.months, got, c.want)
		}
	}
}
