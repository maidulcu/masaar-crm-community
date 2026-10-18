// Package cheque holds the rules for post-dated rent cheques: how a lease's rent is split
// into a cheque schedule, and which status changes are allowed.
package cheque

import (
	"errors"
	"math"
	"time"
)

// Statuses of a cheque.
const (
	Received  = "received"  // in the agency's hands, not yet banked
	OnHold    = "on_hold"   // held back (tenant asked to delay, dispute…)
	Deposited = "deposited" // presented to the bank
	Cleared   = "cleared"   // paid
	Bounced   = "bounced"   // returned unpaid by the bank
	Replaced  = "replaced"  // swapped for another cheque
	Returned  = "returned"  // handed back to the tenant
)

var transitions = map[string][]string{
	Received:  {Deposited, OnHold, Returned, Replaced},
	OnHold:    {Received, Deposited, Returned, Replaced},
	Deposited: {Cleared, Bounced},
	Bounced:   {Deposited, Replaced, Returned}, // re-presented, swapped, or given back
}

// ValidStatus reports whether s is a known status.
func ValidStatus(s string) bool {
	switch s {
	case Received, OnHold, Deposited, Cleared, Bounced, Replaced, Returned:
		return true
	}
	return false
}

// CanTransition reports whether a cheque may move from one status to another.
func CanTransition(from, to string) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// Final reports whether no further change is possible.
func Final(status string) bool { return len(transitions[status]) == 0 }

// Planned is one cheque of a schedule.
type Planned struct {
	Date   time.Time
	Amount float64
}

// MaxCheques bounds a schedule.
const MaxCheques = 24

// ErrBadSchedule is returned for an impossible schedule.
var ErrBadSchedule = errors.New("a schedule needs 1 to 24 cheques and a lease of at least a month")

// LeaseMonths returns the whole number of months a lease covers. The end date is the
// lease's last day, so 1 Jan – 31 Dec is 12 months. A remainder of 15 days or more counts
// as a month.
func LeaseMonths(start, end time.Time) int {
	start, endExcl := day(start), day(end).AddDate(0, 0, 1)
	months := (endExcl.Year()-start.Year())*12 + int(endExcl.Month()-start.Month())
	anchor := addMonths(start, months)
	if anchor.After(endExcl) {
		months--
		anchor = addMonths(start, months)
	}
	if endExcl.Sub(anchor) >= 15*24*time.Hour {
		months++
	}
	return months
}

// Schedule splits the rent for the lease (monthly rent × lease months) into count
// cheques, dated at equal month intervals from first (the lease start when zero). Amounts
// are rounded to the currency's decimals; the last cheque absorbs the rounding.
func Schedule(start, end, first time.Time, monthlyRent float64, count, decimals int) ([]Planned, error) {
	months := LeaseMonths(start, end)
	if count < 1 || count > MaxCheques || months < 1 || monthlyRent <= 0 {
		return nil, ErrBadSchedule
	}
	if first.IsZero() {
		first = start
	}
	first = day(first)
	scale := math.Pow10(decimals)
	total := math.Round(monthlyRent * float64(months) * scale)
	each := math.Round(total / float64(count))
	out := make([]Planned, count)
	for i := range out {
		amt := each
		if i == count-1 {
			amt = total - each*float64(count-1)
		}
		out[i] = Planned{Date: addMonths(first, i*months/count), Amount: amt / scale}
	}
	return out, nil
}

func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// addMonths adds n months, keeping the day of month where it exists and otherwise using
// the month's last day (31 Jan + 1 month = 28/29 Feb).
func addMonths(t time.Time, n int) time.Time {
	y, m, d := t.Date()
	firstOfTarget := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	last := firstOfTarget.AddDate(0, 1, -1).Day()
	if d > last {
		d = last
	}
	return time.Date(firstOfTarget.Year(), firstOfTarget.Month(), d, 0, 0, 0, 0, time.UTC)
}

// CountFor suggests the number of cheques for a lease term and payment frequency:
// monthly, quarterly, semi_annual or annual (anything else: one cheque).
func CountFor(frequency string, months int) int {
	per := map[string]int{"monthly": 1, "quarterly": 3, "semi_annual": 6, "annual": 12}[frequency]
	if per == 0 || months < per {
		return 1
	}
	n := months / per
	if n > MaxCheques {
		n = MaxCheques
	}
	return n
}
