package jurisdiction

import (
	"testing"
	"time"
)

func TestForAndFallback(t *testing.T) {
	ae, ok := For("ae")
	if !ok || ae.Currency != "AED" || ae.CurrencyDecimals != 2 || ae.Timezone != "Asia/Dubai" {
		t.Fatalf("AE rules wrong: %+v ok=%v", ae, ok)
	}
	if _, ok := For("ZZ"); ok {
		t.Fatal("unknown country should not resolve")
	}
	if got := MustFor("").Country; got != DefaultCountry {
		t.Fatalf("MustFor(\"\") = %s, want %s", got, DefaultCountry)
	}
}

func TestVATRate(t *testing.T) {
	ae := MustFor("AE")
	if r, err := ae.VATRate(VATStandard); err != nil || r != 0.05 {
		t.Fatalf("standard rate = %v, %v", r, err)
	}
	if r, err := ae.VATRate(VATExempt); err != nil || r != 0 {
		t.Fatalf("exempt rate = %v, %v", r, err)
	}
	if _, err := ae.VATRate("luxury"); err == nil {
		t.Fatal("unknown category should error")
	}
}

func TestRegions(t *testing.T) {
	ae := MustFor("AE")
	for _, code := range []string{"", "AE-DU", "ae-az"} {
		if !ae.HasRegion(code) {
			t.Errorf("HasRegion(%q) = false", code)
		}
	}
	if ae.HasRegion("SA-01") {
		t.Error("Saudi region accepted for AE")
	}
}

func TestWeekend(t *testing.T) {
	ae := MustFor("AE")
	// 2026-10-17 is a Saturday; 2026-10-16 a Friday (a working day in the UAE since 2022).
	sat := time.Date(2026, 10, 17, 10, 0, 0, 0, ae.Location())
	fri := time.Date(2026, 10, 16, 10, 0, 0, 0, ae.Location())
	if !ae.IsWeekend(sat) || ae.IsWeekend(fri) {
		t.Fatalf("weekend wrong: sat=%v fri=%v", ae.IsWeekend(sat), ae.IsWeekend(fri))
	}
}

func TestRoundMoney(t *testing.T) {
	cases := []struct {
		amount   float64
		currency string
		want     float64
	}{
		{1.005, "AED", 1.01},
		{2.675, "AED", 2.68},
		{-1.005, "AED", -1.01},
		{12.3456, "BHD", 12.346},
		{12.3454, "KWD", 12.345},
		{0.125, "OMR", 0.125},
		{10.555, "XXX", 10.56}, // unknown currency: 2 decimals
	}
	for _, c := range cases {
		if got := RoundMoney(c.amount, c.currency); got != c.want {
			t.Errorf("RoundMoney(%v, %s) = %v, want %v", c.amount, c.currency, got, c.want)
		}
	}
}

func TestCurrency(t *testing.T) {
	if d, ok := CurrencyDecimals("bhd"); !ok || d != 3 {
		t.Fatalf("BHD decimals = %d, %v", d, ok)
	}
	if IsCurrency("ABC") {
		t.Fatal("ABC is not a supported currency")
	}
	if !ValidTimezone("Asia/Riyadh") || ValidTimezone("Mars/Base") || ValidTimezone("") {
		t.Fatal("ValidTimezone wrong")
	}
}

func TestValidTaxID(t *testing.T) {
	ae := MustFor("AE")
	for id, want := range map[string]bool{
		"":                   true,
		"100123456700003":    true,
		"100-1234-5670-0003": true,
		"10012345670000":     false,
		"10012345670000A":    false,
	} {
		if got := ae.ValidTaxID(id); got != want {
			t.Errorf("ValidTaxID(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestFindRegion(t *testing.T) {
	ae := MustFor("AE")
	for _, text := range []string{"AE-DU", "dubai", "Dubai", "دبي"} {
		reg, ok := ae.FindRegion(text)
		if !ok || reg.Code != "AE-DU" || reg.ListingPermit != PermitTrakheesi {
			t.Errorf("FindRegion(%q) = %+v, %v", text, reg, ok)
		}
	}
	if reg, ok := ae.FindRegion("Sharjah"); !ok || reg.ListingPermit != "" {
		t.Errorf("Sharjah = %+v, %v", reg, ok)
	}
	if _, ok := ae.FindRegion("Narnia"); ok {
		t.Error("unknown region resolved")
	}
}
