package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/pdf"
)

// InvoiceRepository defines the interface for invoice data access.
type InvoiceRepository interface {
	Create(ctx context.Context, inv *domain.VATInvoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.VATInvoice, error)
	ListAll(ctx context.Context, page, limit int) ([]domain.VATInvoice, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus) error
	NextInvoiceNo(ctx context.Context) (string, error)
}

// DealRepository defines the interface for deal data access used by the invoice handler.
type DealRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Deal, error)
}

// CompanySettingsRepository defines the interface for company settings data access.
type CompanySettingsRepository interface {
	Get(ctx context.Context) (*domain.CompanySettings, error)
}

type InvoiceHandler struct {
	invoices InvoiceRepository
	deals    DealRepository
	company  CompanySettingsRepository
}

func NewInvoiceHandler(invoices InvoiceRepository, deals DealRepository, company CompanySettingsRepository) *InvoiceHandler {
	return &InvoiceHandler{
		invoices: invoices,
		deals:    deals,
		company:  company,
	}
}

// List godoc
// @Summary      List all invoices
// @Tags         Invoices
// @Produce      json
// @Param        page  query  int  false  "Page (default 1)"
// @Param        limit query  int  false  "Limit (default 50)"
// @Success      200  {object}  object{data=[]domain.VATInvoice,total=int}
// @Security     BearerAuth
// @Router       /invoices [get]
func (h *InvoiceHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	invoices, total, err := h.invoices.ListAll(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if invoices == nil {
		invoices = []domain.VATInvoice{}
	}
	return c.JSON(fiber.Map{"data": invoices, "total": total})
}

// Create godoc
// @Summary      Create VAT invoice
// @Description  Generates a VAT invoice (5% UAE VAT) for a deal. Invoice number is auto-assigned as INV-YYYY-NNNN.
// @Tags         Invoices
// @Accept       json
// @Produce      json
// @Param        body  body      object{deal_id=string,subtotal=number}  true  "Invoice payload"
// @Success      201   {object}  domain.VATInvoice
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices [post]
func (h *InvoiceHandler) Create(c *fiber.Ctx) error {
	var body struct {
		DealID   uuid.UUID `json:"deal_id"`
		Subtotal float64   `json:"subtotal"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if body.DealID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "deal_id is required"})
	}
	if _, err := h.deals.GetByID(c.Context(), body.DealID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "deal not found"})
	}
	if body.Subtotal <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "subtotal must be positive"})
	}

	invoiceNo, err := h.invoices.NextInvoiceNo(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	inv := domain.VATInvoice{
		DealID:    body.DealID,
		InvoiceNo: invoiceNo,
		Subtotal:  body.Subtotal,
		VATRate:   0.05,
		Status:    domain.InvoiceDraft,
	}

	if err := h.invoices.Create(c.Context(), &inv); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(inv)
}

// Get godoc
// @Summary      Get invoice
// @Tags         Invoices
// @Produce      json
// @Param        id  path      string  true  "Invoice UUID"
// @Success      200  {object}  domain.VATInvoice
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices/{id} [get]
func (h *InvoiceHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	inv, err := h.invoices.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "invoice not found"})
	}
	return c.JSON(inv)
}

// Send godoc
// @Summary      Mark invoice as sent
// @Description  Transitions the invoice status from draft → sent.
// @Tags         Invoices
// @Produce      json
// @Param        id  path      string  true  "Invoice UUID"
// @Success      200  {object}  object{id=string,status=string}
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices/{id}/send [post]
func (h *InvoiceHandler) Send(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.invoices.UpdateStatus(c.Context(), id, domain.InvoiceSent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"id": id, "status": domain.InvoiceSent})
}

// UpdateStatus godoc
// @Summary      Update invoice status
// @Description  Manually set invoice status to draft|sent|paid.
// @Tags         Invoices
// @Accept       json
// @Produce      json
// @Param        id    path      string                   true  "Invoice UUID"
// @Param        body  body      object{status=string}    true  "New status"
// @Success      200   {object}  object{id=string,status=string}
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices/{id}/status [patch]
func (h *InvoiceHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Status domain.InvoiceStatus `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status required"})
	}

	validStatuses := map[domain.InvoiceStatus]bool{
		domain.InvoiceDraft: true,
		domain.InvoiceSent:  true,
		domain.InvoicePaid:  true,
	}
	if !validStatuses[body.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}

	if err := h.invoices.UpdateStatus(c.Context(), id, body.Status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"id": id, "status": body.Status})
}

// DownloadPDF godoc
// @Summary      Download invoice as PDF
// @Description  Generates and downloads a PDF version of the invoice.
// @Tags         Invoices
// @Produce      pdf
// @Param        id  path  string  true  "Invoice UUID"
// @Success      200  {file}  application/pdf
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices/{id}/pdf [get]
func (h *InvoiceHandler) DownloadPDF(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	inv, err := h.invoices.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "invoice not found"})
	}

	deal, err := h.deals.GetByID(c.Context(), inv.DealID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "deal not found"})
	}

	// Load company settings from database
	company, err := h.company.Get(c.Context())
	if err != nil {
		// Fallback if company settings not configured
		company = &domain.CompanySettings{
			Name:            "Your Company Name",
			VATNumber:       "Not configured",
			BusinessAddress: "Dubai, United Arab Emirates",
		}
	}

	pdfData := pdf.InvoiceData{
		InvoiceNo:   inv.InvoiceNo,
		IssuedAt:    inv.IssuedAt,
		Subtotal:    inv.Subtotal,
		VATRate:     inv.VATRate,
		VATAmount:   inv.VATAmount,
		Total:       inv.Total,
		DealTitle:   deal.Title,
		CompanyName: company.Name,
		CompanyAddr: company.BusinessAddress,
		CompanyVAT:  company.VATNumber,
	}

	pdfBytes, err := pdf.GenerateInvoice(pdfData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate PDF"})
	}

	filename := fmt.Sprintf("invoice-%s.pdf", inv.InvoiceNo)
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename="+filename)
	return c.Send(pdfBytes)
}
