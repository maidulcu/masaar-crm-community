package repo

import (
	"errors"
	"testing"

	"github.com/maidulcu/masaar-crm/internal/einvoice"
)

// An invoice is submitted once; it can be submitted again only after a rejection or a
// failed delivery. Another company can neither read nor submit it.
func TestEInvoiceSubmissionRules(t *testing.T) {
	e := setup(t)
	invoices, store := NewInvoiceRepo(e.pool), NewEInvoiceRepo(e.pool)
	inv := testInvoice(e.deal(t, e.a), 100)
	if err := invoices.Issue(e.a.ctx, inv); err != nil {
		t.Fatal(err)
	}
	rec := func(status string) *EInvoice {
		return &EInvoice{InvoiceID: inv.ID, Provider: "mock", Status: status, ProviderRef: "ref-" + status, XML: "<Invoice/>",
			Buyer: einvoice.Party{Name: "Buyer"}, SubmittedBy: &e.a.user}
	}

	if err := store.Save(e.b.ctx, rec(einvoice.StatusSubmitted)); !isNotFound(err) {
		t.Fatalf("B submitted A's invoice: %v", err)
	}
	if err := store.Save(e.a.ctx, rec(einvoice.StatusSubmitted)); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(e.a.ctx, rec(einvoice.StatusSubmitted)); !errors.Is(err, ErrAlreadySubmitted) {
		t.Fatalf("second submission while in progress: %v", err)
	}
	if _, err := store.Get(e.b.ctx, inv.ID); !isNotFound(err) {
		t.Fatalf("B read A's e-invoice: %v", err)
	}
	if p, _ := store.Pending(e.b.ctx); len(p) != 0 {
		t.Fatal("B saw A's pending e-invoice")
	}
	if ok, _ := store.Decide(e.b.ctx, inv.ID, einvoice.StatusAccepted, ""); ok {
		t.Fatal("B decided A's e-invoice")
	}

	p, err := store.Pending(e.a.ctx)
	if err != nil || len(p) != 1 || p[0].InvoiceNo != inv.InvoiceNo || p[0].ProviderRef != "ref-submitted" {
		t.Fatalf("pending: %+v %v", p, err)
	}
	if ok, err := store.Decide(e.a.ctx, inv.ID, einvoice.StatusRejected, "bad buyer TIN"); !ok || err != nil {
		t.Fatalf("decide: %v %v", ok, err)
	}
	if ok, _ := store.Decide(e.a.ctx, inv.ID, einvoice.StatusAccepted, ""); ok {
		t.Fatal("a decided e-invoice was decided again")
	}
	got, err := store.Get(e.a.ctx, inv.ID)
	if err != nil || got.Status != einvoice.StatusRejected || got.Reason != "bad buyer TIN" || got.DecidedAt == nil || got.Buyer.Name != "Buyer" {
		t.Fatalf("after rejection: %+v %v", got, err)
	}

	// Resubmission after the rejection clears the decision.
	if err := store.Save(e.a.ctx, rec(einvoice.StatusSubmitted)); err != nil {
		t.Fatalf("resubmit after rejection: %v", err)
	}
	if got, _ := store.Get(e.a.ctx, inv.ID); got.Status != einvoice.StatusSubmitted || got.Reason != "" || got.DecidedAt != nil {
		t.Fatalf("after resubmission: %+v", got)
	}
	if ok, _ := store.Decide(e.a.ctx, inv.ID, einvoice.StatusAccepted, ""); !ok {
		t.Fatal("accept")
	}
	if err := store.Save(e.a.ctx, rec(einvoice.StatusSubmitted)); !errors.Is(err, ErrAlreadySubmitted) {
		t.Fatalf("resubmitting an accepted e-invoice: %v", err)
	}
}

func TestEInvoiceSettingsAndSeller(t *testing.T) {
	e := setup(t)
	store := NewEInvoiceRepo(e.pool)
	if s, err := store.Settings(e.a.ctx); err != nil || s.Provider != "off" || s.LegalIDType != "TL" {
		t.Fatalf("defaults: %+v %v", s, err)
	}
	must(t, e.pool, `UPDATE company_settings SET vat_number = '100000000000003', trade_license_no = 'TL-1', region = 'AE-DU',
		business_address = 'Street 1', bank_iban = 'AE07 0331 2345 6789 0123 456' WHERE company_id = $1`, e.a.id)
	in := &EInvoiceSettings{Provider: "mock", TIN: "1000000000", LegalIDType: "TL", TradeLicenseAuthority: "DET", City: "Dubai"}
	if err := store.UpdateSettings(e.a.ctx, in); err != nil {
		t.Fatal(err)
	}
	if s, _ := store.Settings(e.b.ctx); s.Provider != "off" {
		t.Fatal("A's settings changed B's")
	}
	seller, iban, err := store.Seller(e.a.ctx)
	if err != nil || seller.TIN != "1000000000" || seller.Emirate != "DXB" || seller.LegalAuthority != "DET" || seller.City != "Dubai" ||
		iban != "AE070331234567890123456" {
		t.Fatalf("seller: %+v %q %v", seller, iban, err)
	}
}
