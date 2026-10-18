package invoice

import (
	"testing"

	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
)

var ae = jurisdiction.MustFor("AE")

func TestComputeMixedCategories(t *testing.T) {
	tot, err := Compute([]Line{
		{Description: "Agency commission", Quantity: 1, UnitPrice: 20000, VATCategory: jurisdiction.VATStandard},
		{Description: "Residential rent (exempt)", Quantity: 1, UnitPrice: 85000, VATCategory: jurisdiction.VATExempt},
		{Description: "Admin fee", Quantity: 3, UnitPrice: 33.335, VATCategory: jurisdiction.VATStandard},
	}, ae, "AED", true)
	if err != nil {
		t.Fatal(err)
	}
	// 3 × 33.335 = 100.005 → 100.01; VAT 5.0005 → 5.00
	want := []struct{ net, vat float64 }{{20000, 1000}, {85000, 0}, {100.01, 5}}
	for i, w := range want {
		if tot.Lines[i].NetAmount != w.net || tot.Lines[i].VATAmount != w.vat {
			t.Errorf("line %d = %v/%v, want %v/%v", i+1, tot.Lines[i].NetAmount, tot.Lines[i].VATAmount, w.net, w.vat)
		}
	}
	if tot.Subtotal != 105100.01 || tot.VATAmount != 1005 || tot.Total != 106105.01 {
		t.Fatalf("totals = %v / %v / %v", tot.Subtotal, tot.VATAmount, tot.Total)
	}
}

func TestComputeThreeDecimalCurrency(t *testing.T) {
	tot, err := Compute([]Line{{Description: "Fee", Quantity: 1, UnitPrice: 10.1234, VATCategory: jurisdiction.VATStandard}}, ae, "BHD", true)
	if err != nil {
		t.Fatal(err)
	}
	if tot.Subtotal != 10.123 || tot.VATAmount != 0.506 || tot.Total != 10.629 {
		t.Fatalf("BHD totals = %v / %v / %v", tot.Subtotal, tot.VATAmount, tot.Total)
	}
}

func TestComputeUnregisteredSupplier(t *testing.T) {
	if _, err := Compute([]Line{{Description: "Fee", Quantity: 1, UnitPrice: 10, VATCategory: jurisdiction.VATStandard}}, ae, "AED", false); !IsValidation(err) {
		t.Fatalf("standard-rated line without TRN accepted: %v", err)
	}
	tot, err := Compute([]Line{{Description: "Fee", Quantity: 1, UnitPrice: 10}}, ae, "AED", false)
	if err != nil || tot.VATAmount != 0 || tot.Lines[0].VATCategory != jurisdiction.VATOutOfScope {
		t.Fatalf("default for unregistered supplier: %+v %v", tot, err)
	}
}

func TestComputeRejectsBadInput(t *testing.T) {
	cases := map[string][]Line{
		"no lines":    nil,
		"description": {{Quantity: 1, UnitPrice: 1}},
		"quantity":    {{Description: "x", Quantity: 0, UnitPrice: 1}},
		"price":       {{Description: "x", Quantity: 1, UnitPrice: -1}},
		"category":    {{Description: "x", Quantity: 1, UnitPrice: 1, VATCategory: "luxury"}},
		"zero total":  {{Description: "x", Quantity: 1, UnitPrice: 0}},
	}
	for name, lines := range cases {
		if _, err := Compute(lines, ae, "AED", true); !IsValidation(err) {
			t.Errorf("%s: got %v, want a validation error", name, err)
		}
	}
	if _, err := Compute([]Line{{Description: "x", Quantity: 1, UnitPrice: 1}}, ae, "XYZ", true); !IsValidation(err) {
		t.Error("unknown currency accepted")
	}
}

func TestCheckCredit(t *testing.T) {
	if err := CheckCredit(1050, 0, 1050, "AED"); err != nil {
		t.Fatalf("full credit refused: %v", err)
	}
	if err := CheckCredit(1050, 500, 550, "AED"); err != nil {
		t.Fatalf("remaining credit refused: %v", err)
	}
	if err := CheckCredit(1050, 500, 550.01, "AED"); !IsValidation(err) {
		t.Fatal("over-credit accepted")
	}
}

func TestNumber(t *testing.T) {
	if got := Number(PrefixCreditNote, 2026, 7); got != "CN-2026-0007" {
		t.Fatalf("Number = %s", got)
	}
}
