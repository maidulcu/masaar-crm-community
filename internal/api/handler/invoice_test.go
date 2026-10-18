package handler

import (
	"context"
	"mime"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type mockInvoiceRepo struct {
	inv *domain.VATInvoice
	err error
}

func (m *mockInvoiceRepo) Create(ctx context.Context, inv *domain.VATInvoice) error {
	return nil
}

func (m *mockInvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.VATInvoice, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.inv, nil
}

func (m *mockInvoiceRepo) ListAll(ctx context.Context, page, limit int) ([]domain.VATInvoice, int, error) {
	return nil, 0, nil
}

func (m *mockInvoiceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus) error {
	return nil
}

func (m *mockInvoiceRepo) NextInvoiceNo(ctx context.Context) (string, error) {
	return "INV-001", nil
}

type mockDealRepo struct {
	deal *domain.Deal
	err  error
}

func (m *mockDealRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deal, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.deal, nil
}

type mockCompanySettingsRepo struct {
	settings *domain.CompanySettings
	err      error
}

func (m *mockCompanySettingsRepo) Get(ctx context.Context) (*domain.CompanySettings, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.settings, nil
}

func TestDownloadPDF_HeaderSafety(t *testing.T) {
	app := fiber.New()

	dealID := uuid.New()
	invID := uuid.New()

	invRepo := &mockInvoiceRepo{
		inv: &domain.VATInvoice{
			ID:        invID,
			DealID:    dealID,
			InvoiceNo: "2025-001\r\nX-Injected-Header: injected_value",
		},
	}
	dealRepo := &mockDealRepo{
		deal: &domain.Deal{
			ID:    dealID,
			Title: "Test Deal",
		},
	}
	compRepo := &mockCompanySettingsRepo{
		settings: &domain.CompanySettings{
			Name: "Test Company",
		},
	}

	h := NewInvoiceHandler(invRepo, dealRepo, compRepo)

	// Route that tests DownloadPDF handler (it returns 500 because pdf.GenerateInvoice returns errProOnly in Community edition, but headers are set prior)
	app.Get("/invoices/:id/pdf", h.DownloadPDF)

	req := httptest.NewRequest("GET", "/invoices/"+invID.String()+"/pdf", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	// Verify safeFilename directly on malicious invoice numbers
	filename := "invoice-" + invRepo.inv.InvoiceNo + ".pdf"
	formatted := mime.FormatMediaType("attachment", map[string]string{"filename": safeFilename(filename)})

	if strings.ContainsAny(formatted, "\r\n") {
		t.Errorf("Formatted Content-Disposition header contains newline characters: %q", formatted)
	}

	if resp.Header.Get("X-Injected-Header") != "" {
		t.Errorf("Header injection succeeded! X-Injected-Header found in response")
	}
}
