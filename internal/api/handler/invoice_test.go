package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

type fakeInvoices struct {
	issued []*domain.VATInvoice
	byID   map[uuid.UUID]*domain.VATInvoice
}

func (f *fakeInvoices) Issue(_ context.Context, inv *domain.VATInvoice) error {
	inv.ID = uuid.New()
	inv.InvoiceNo = "INV-2026-0001"
	f.issued = append(f.issued, inv)
	return nil
}
func (f *fakeInvoices) GetByID(_ context.Context, id uuid.UUID) (*domain.VATInvoice, error) {
	if inv, ok := f.byID[id]; ok {
		return inv, nil
	}
	return nil, io.EOF
}
func (f *fakeInvoices) ListAll(context.Context, int, int) ([]domain.VATInvoice, int, error) {
	return nil, 0, nil
}
func (f *fakeInvoices) ListCreditNotes(context.Context, uuid.UUID) ([]domain.VATInvoice, error) {
	return nil, nil
}
func (f *fakeInvoices) UpdateStatus(context.Context, uuid.UUID, domain.InvoiceStatus, string) error {
	return nil
}

type fakeDeals struct{ deal *domain.Deal }

func (f fakeDeals) GetByID(context.Context, uuid.UUID) (*domain.Deal, error) { return f.deal, nil }
func (f fakeDeals) CustomerForDeal(context.Context, uuid.UUID) (string, error) {
	return "Fatima Al Mansoori", nil
}

type fakeCompany struct{ s domain.CompanySettings }

func (f fakeCompany) Get(context.Context) (*domain.CompanySettings, error) { return &f.s, nil }

func invoiceApp(trn string) (*fiber.App, *fakeInvoices, *domain.Deal) {
	deal := &domain.Deal{ID: uuid.New(), Title: "Marina apartment", Currency: "AED"}
	inv := &fakeInvoices{byID: map[uuid.UUID]*domain.VATInvoice{}}
	h := NewInvoiceHandler(inv, fakeDeals{deal}, fakeCompany{domain.CompanySettings{
		Name: "Masaar Realty", BusinessAddress: "Dubai", VATNumber: trn, Country: "AE", Currency: "AED",
	}}, fakeDeals{deal}, nil)
	app := fiber.New()
	app.Post("/invoices", h.Create)
	app.Post("/invoices/:id/credit-notes", h.CreateCreditNote)
	return app, inv, deal
}

func post(t *testing.T, app *fiber.App, path string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestCreateInvoiceLegacyBodyAndDefaults(t *testing.T) {
	app, inv, deal := invoiceApp("100123456700003")
	status, _ := post(t, app, "/invoices", map[string]any{"deal_id": deal.ID, "subtotal": 1000})
	if status != fiber.StatusCreated {
		t.Fatalf("status = %d", status)
	}
	got := inv.issued[0]
	if len(got.Lines) != 1 || got.Lines[0].Description != "Marina apartment" || got.Total != 1050 || got.VATAmount != 50 {
		t.Fatalf("issued = %+v", got)
	}
	if got.Customer.Name != "Fatima Al Mansoori" || got.Supplier.TRN != "100123456700003" || got.Currency != "AED" {
		t.Fatalf("parties/currency = %+v %+v %s", got.Supplier, got.Customer, got.Currency)
	}
}

func TestCreateInvoiceValidation(t *testing.T) {
	app, _, deal := invoiceApp("")
	// No company TRN: VAT cannot be charged.
	status, out := post(t, app, "/invoices", map[string]any{"deal_id": deal.ID,
		"lines": []map[string]any{{"description": "Commission", "quantity": 1, "unit_price": 100, "vat_category": "standard"}}})
	if status != fiber.StatusUnprocessableEntity {
		t.Fatalf("standard VAT without TRN: %d %v", status, out)
	}
	// Out-of-scope is fine.
	status, _ = post(t, app, "/invoices", map[string]any{"deal_id": deal.ID,
		"lines": []map[string]any{{"description": "Commission", "quantity": 1, "unit_price": 100, "vat_category": "out_of_scope"}}})
	if status != fiber.StatusCreated {
		t.Fatalf("out-of-scope invoice: %d", status)
	}
	// Bad customer TRN.
	app, _, deal = invoiceApp("100123456700003")
	status, _ = post(t, app, "/invoices", map[string]any{"deal_id": deal.ID, "subtotal": 10,
		"customer": map[string]any{"name": "B2B LLC", "trn": "123"}})
	if status != fiber.StatusUnprocessableEntity {
		t.Fatalf("bad customer TRN: %d", status)
	}
}

func TestCreditNoteDefaultsToFullCredit(t *testing.T) {
	app, inv, deal := invoiceApp("100123456700003")
	orig := &domain.VATInvoice{ID: uuid.New(), Kind: domain.KindInvoice, DealID: deal.ID, Currency: "AED",
		Supplier: domain.InvoiceParty{Name: "S", TRN: "100123456700003"}, Customer: domain.InvoiceParty{Name: "C"},
		Lines: []domain.InvoiceLine{{Description: "Fee", Quantity: 2, UnitPrice: 50, VATCategory: "standard"}}}
	inv.byID[orig.ID] = orig

	if status, _ := post(t, app, "/invoices/"+orig.ID.String()+"/credit-notes", map[string]any{}); status != fiber.StatusUnprocessableEntity {
		t.Fatalf("credit note without reason: %d", status)
	}
	status, _ := post(t, app, "/invoices/"+orig.ID.String()+"/credit-notes", map[string]any{"reason": "cancelled"})
	if status != fiber.StatusCreated {
		t.Fatalf("status = %d", status)
	}
	cn := inv.issued[0]
	if cn.Kind != domain.KindCreditNote || *cn.OriginalInvoiceID != orig.ID || cn.Total != 105 || cn.Customer.Name != "C" {
		t.Fatalf("credit note = %+v", cn)
	}
}

// A crafted invoice number cannot inject response headers through the PDF download.
func TestDownloadPDF_HeaderSafety(t *testing.T) {
	deal := &domain.Deal{ID: uuid.New(), Title: "Test Deal"}
	inv := &domain.VATInvoice{ID: uuid.New(), DealID: deal.ID, InvoiceNo: "2025-001\r\nX-Injected-Header: injected_value"}
	h := NewInvoiceHandler(&fakeInvoices{byID: map[uuid.UUID]*domain.VATInvoice{inv.ID: inv}}, fakeDeals{deal},
		fakeCompany{domain.CompanySettings{Name: "Test Company"}}, fakeDeals{deal}, nil)
	app := fiber.New()
	app.Get("/invoices/:id/pdf", h.DownloadPDF)

	resp, err := app.Test(httptest.NewRequest("GET", "/invoices/"+inv.ID.String()+"/pdf", nil))
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	formatted := mime.FormatMediaType("attachment", map[string]string{"filename": safeFilename("invoice-" + inv.InvoiceNo + ".pdf")})
	if strings.ContainsAny(formatted, "\r\n") {
		t.Errorf("Content-Disposition header contains newline characters: %q", formatted)
	}
	if resp.Header.Get("X-Injected-Header") != "" {
		t.Errorf("header injection succeeded: X-Injected-Header found in response")
	}
}
