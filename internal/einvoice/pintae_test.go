package einvoice

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden PINT AE files in testdata")

func sampleSeller() Party {
	return Party{Name: "Example Realty LLC", TIN: "1000000001", TRN: "100000000100003", LegalID: "CN-1234567",
		LegalIDType: LegalTradeLicence, LegalAuthority: "Dubai Department of Economy and Tourism",
		Street: "Office 1201, Marina Plaza", City: "Dubai", Emirate: "DXB", Country: "AE", Email: "billing@example.ae"}
}

func sampleBuyer() Party {
	return Party{Name: "Buyer Holdings LLC", TIN: "1000000002", TRN: "100000000200003", LegalID: "CN-7654321",
		LegalIDType: LegalTradeLicence, LegalAuthority: "Abu Dhabi Department of Economic Development",
		Street: "Tower 2, Al Maryah Island", City: "Abu Dhabi", Emirate: "AUH", Country: "AE"}
}

func sampleInvoice() *Document {
	due := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	return &Document{
		Kind: KindInvoice, Number: "INV-2026-0042", UUID: "6f1c0b8e-2d1f-4b43-9a3e-7c0f9d2b5a11",
		IssueDate: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), DueDate: &due, Currency: "AED",
		Seller: sampleSeller(), Buyer: sampleBuyer(), IBAN: "AE070331234567890123456", ServiceSAC: "997212",
		Lines: []Line{
			{Description: "Agency commission: sale of Marina Vista 2BR, 2% of AED 2,400,000", Quantity: 1, UnitPrice: 48000, Net: 48000, Category: "standard", Rate: 0.05, VAT: 2400},
			{Description: "Tenancy contract registration fee (disbursement)", Quantity: 1, UnitPrice: 220, Net: 220, Category: "zero_rated", Rate: 0, VAT: 0},
		},
	}
}

func sampleCreditNote() *Document {
	return &Document{
		Kind: KindCreditNote, Number: "CN-2026-0007", UUID: "0b9a7e52-8a1c-4d5f-b0f4-2e6d8c9a1b33",
		IssueDate: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), Currency: "AED",
		Seller: sampleSeller(), Buyer: sampleBuyer(), ServiceSAC: "997212",
		PrecedingNumber: "INV-2026-0042", PrecedingDate: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), CreditReason: "DL8.61.1.C",
		Lines: []Line{{Description: "Discount on agency commission", Quantity: 1, UnitPrice: 4000, Net: 4000, Category: "standard", Rate: 0.05, VAT: 200}},
	}
}

// The golden files are what CI validates against the official PINT AE rules
// (scripts/pint-ae/validate.sh); this test keeps them in step with the generator.
func TestGolden(t *testing.T) {
	mixed := sampleInvoice()
	mixed.Number, mixed.UUID, mixed.ExemptionCode = "INV-2026-0043", "9d2c4e61-7b3a-4f0e-8c51-1a2b3c4d5e6f", "VATEX-AE-EXAMPLE"
	mixed.Lines = []Line{
		{Description: "Property management fee, October", Quantity: 1, UnitPrice: 1500, Net: 1500, Category: "standard", Rate: 0.05, VAT: 75},
		{Description: "Residential rent collected for the landlord", Quantity: 2, UnitPrice: 7500, Net: 15000, Category: "exempt", Rate: 0, VAT: 0},
		{Description: "Government fee passed through", Quantity: 1, UnitPrice: 580, Net: 580, Category: "out_of_scope", Rate: 0, VAT: 0},
	}
	oos := sampleInvoice()
	oos.Number, oos.UUID = "INV-2026-0044", "3e8f1a27-5c6d-4b9e-a0f1-2b3c4d5e6f70"
	oos.Lines = []Line{{Description: "Government fees paid on the client's behalf", Quantity: 1, UnitPrice: 4200, Net: 4200, Category: "out_of_scope"}}
	for name, d := range map[string]*Document{"invoice.xml": sampleInvoice(), "credit-note.xml": sampleCreditNote(),
		"invoice-mixed.xml": mixed, "invoice-out-of-scope.xml": oos} {
		got, err := d.XML()
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("testdata", name)
		if *update {
			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run go test ./internal/einvoice -update)", err)
		}
		if string(got) != string(want) {
			t.Errorf("%s is out of date: run go test ./internal/einvoice -update and validate it", name)
		}
	}
}

func TestCheckListsMissingData(t *testing.T) {
	d := sampleInvoice()
	d.Seller.TIN, d.Buyer.Emirate, d.IBAN, d.DueDate = "", "", "", nil
	err := d.Check()
	me, ok := err.(*MissingError)
	if !ok || len(me.Fields) != 4 {
		t.Fatalf("missing = %v", err)
	}
}
