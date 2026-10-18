package handler

import (
	"testing"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestMergeLocale(t *testing.T) {
	current := &domain.CompanySettings{Country: "AE", Region: "AE-DU", Currency: "AED", Timezone: "Asia/Dubai"}

	// An older client sends no locale fields: everything is kept.
	req := &domain.CompanySettings{}
	if msg := mergeLocale(req, current); msg != "" {
		t.Fatalf("unexpected error %q", msg)
	}
	if req.Country != "AE" || req.Region != "AE-DU" || req.Currency != "AED" || req.Timezone != "Asia/Dubai" {
		t.Fatalf("not kept: %+v", req)
	}

	// Values are normalised.
	req = &domain.CompanySettings{Region: "ae-az", Currency: "usd"}
	if msg := mergeLocale(req, current); msg != "" || req.Region != "AE-AZ" || req.Currency != "USD" {
		t.Fatalf("normalise: %q %+v", msg, req)
	}

	for name, bad := range map[string]*domain.CompanySettings{
		"country":  {Country: "ZZ"},
		"region":   {Region: "SA-01"},
		"currency": {Currency: "XYZ"},
		"timezone": {Timezone: "Mars/Base"},
		"trn":      {VATNumber: "12345"},
	} {
		if msg := mergeLocale(bad, current); msg == "" {
			t.Errorf("invalid %s accepted", name)
		}
	}
}
