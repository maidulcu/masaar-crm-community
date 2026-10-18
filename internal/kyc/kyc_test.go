package kyc

import (
	"reflect"
	"testing"
	"time"
)

func day(s string) *time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return &t
}

func TestCheck(t *testing.T) {
	today, _ := time.Parse("2006-01-02", "2026-10-14")
	for name, c := range map[string]struct {
		docs      []Doc
		corporate bool
		want      []string
	}{
		"nothing":              {nil, false, []string{MissingID}},
		"visa only":            {[]Doc{{ResidenceVisa, day("2027-01-01")}}, false, []string{MissingID}},
		"valid passport":       {[]Doc{{Passport, day("2030-01-01")}}, false, nil},
		"expires today":        {[]Doc{{EmiratesID, day("2026-10-14")}}, false, nil},
		"expired":              {[]Doc{{EmiratesID, day("2026-10-13")}}, false, []string{ExpiredID}},
		"expired and undated":  {[]Doc{{EmiratesID, day("2020-01-01")}, {Passport, nil}}, false, []string{NoExpiry}},
		"expired, one valid":   {[]Doc{{EmiratesID, day("2020-01-01")}, {Passport, day("2031-01-01")}}, false, nil},
		"company no licence":   {[]Doc{{Passport, day("2031-01-01")}}, true, []string{MissingCorp}},
		"company with licence": {[]Doc{{TradeLicence, day("2027-01-01")}}, true, nil},
	} {
		if got := Check(c.docs, c.corporate, today); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
}

func TestFlagReasons(t *testing.T) {
	s := AMLSettings{CashThreshold: 55000, FlagVirtualAsset: true}
	for name, c := range map[string]struct {
		method       string
		amount, cash float64
		want         []string
	}{
		"transfer":            {PayTransfer, 2_000_000, 0, nil},
		"cash at threshold":   {PayCash, 55000, 0, []string{ReasonCash}},
		"cash below":          {PayCash, 54999.99, 0, nil},
		"mixed, cash part":    {PayMixed, 1_000_000, 60000, []string{ReasonCash}},
		"mixed, small cash":   {PayMixed, 1_000_000, 10000, nil},
		"virtual asset":       {PayVirtualAsset, 900_000, 0, []string{ReasonVirtualAsset}},
		"unknown method cash": {"", 900_000, 70000, []string{ReasonCash}},
	} {
		if got := FlagReasons(c.method, c.amount, c.cash, s); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v, want %v", name, got, c.want)
		}
	}
	if got := FlagReasons(PayVirtualAsset, 1, 0, AMLSettings{CashThreshold: 55000}); got != nil {
		t.Errorf("virtual assets flagged with the setting off: %v", got)
	}
}
