package handler

import (
	"context"
	"fmt"
	"mime"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/invoice"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
	"github.com/maidulcu/masaar-crm/internal/pdf"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// InvoiceRepository defines the interface for invoice data access.
type InvoiceRepository interface {
	Issue(ctx context.Context, inv *domain.VATInvoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.VATInvoice, error)
	ListAll(ctx context.Context, page, limit int) ([]domain.VATInvoice, int, error)
	ListCreditNotes(ctx context.Context, invoiceID uuid.UUID) ([]domain.VATInvoice, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus, voidReason string) error
}

// DealRepository defines the interface for deal data access used by the invoice handler.
type DealRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Deal, error)
}

// CompanySettingsRepository defines the interface for company settings data access.
type CompanySettingsRepository interface {
	Get(ctx context.Context) (*domain.CompanySettings, error)
}

// customerLookup finds the deal's contact, used as the default customer.
type customerLookup interface {
	CustomerForDeal(ctx context.Context, dealID uuid.UUID) (name string, err error)
}

type auditLogger interface {
	Log(ctx context.Context, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, diff any)
}

type InvoiceHandler struct {
	invoices  InvoiceRepository
	deals     DealRepository
	company   CompanySettingsRepository
	customers customerLookup
	audit     auditLogger
}

func NewInvoiceHandler(invoices InvoiceRepository, deals DealRepository, company CompanySettingsRepository,
	customers customerLookup, audit auditLogger) *InvoiceHandler {
	return &InvoiceHandler{invoices: invoices, deals: deals, company: company, customers: customers, audit: audit}
}

// List godoc
// @Summary      List all invoices and credit notes
// @Tags         Invoices
// @Produce      json
// @Param        page  query  int  false  "Page (default 1)"
// @Param        limit query  int  false  "Limit (default 50)"
// @Success      200  {object}  object{data=[]domain.VATInvoice,total=int}
// @Security     BearerAuth
// @Router       /invoices [get]
func (h *InvoiceHandler) List(c *fiber.Ctx) error {
	page, limit := pageParams(c, 50, 100)
	invoices, total, err := h.invoices.ListAll(c.Context(), page, limit)
	if err != nil {
		return serverError(c, err)
	}
	if invoices == nil {
		invoices = []domain.VATInvoice{}
	}
	return c.JSON(fiber.Map{"data": invoices, "total": total})
}

type partyInput struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	TRN     string `json:"trn"`
}

type createInvoiceRequest struct {
	DealID     uuid.UUID      `json:"deal_id"`
	Currency   string         `json:"currency"`
	SupplyDate string         `json:"supply_date"` // YYYY-MM-DD, optional
	Customer   *partyInput    `json:"customer"`
	Lines      []invoice.Line `json:"lines"`
	Notes      string         `json:"notes"`
	// Subtotal is the legacy single-amount form: one standard-rated line for the deal.
	Subtotal float64 `json:"subtotal"`
}

// Create godoc
// @Summary      Issue a tax invoice
// @Description  Issues a numbered invoice (INV-YYYY-NNNN) for a deal. VAT is calculated per line from its category (standard, zero_rated, exempt, out_of_scope) under the company's country rules. Supplier details are copied from Settings → Company; the customer defaults to the deal's contact. The legacy body {deal_id, subtotal} still works and issues one standard-rated line.
// @Tags         Invoices
// @Accept       json
// @Produce      json
// @Param        body  body      createInvoiceRequest  true  "Invoice"
// @Success      201   {object}  domain.VATInvoice
// @Failure      400   {object}  object{error=string}
// @Failure      422   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices [post]
func (h *InvoiceHandler) Create(c *fiber.Ctx) error {
	var body createInvoiceRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if body.DealID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "deal_id is required"})
	}
	deal, err := h.deals.GetByID(c.Context(), body.DealID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "deal not found"})
	}
	company, err := h.company.Get(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	rules := jurisdiction.MustFor(company.Country)

	lines := body.Lines
	if len(lines) == 0 && body.Subtotal > 0 {
		lines = []invoice.Line{{Description: deal.Title, Quantity: 1, UnitPrice: body.Subtotal}}
	}
	currency := strings.ToUpper(strings.TrimSpace(body.Currency))
	if currency == "" {
		currency = strings.TrimSpace(deal.Currency)
	}
	if currency == "" {
		currency = company.Currency
	}

	supplierTRN := jurisdiction.NormalizeTaxID(company.VATNumber)
	totals, err := invoice.Compute(lines, rules, currency, supplierTRN != "")
	if err != nil {
		return serverError(c, err)
	}

	customer := domain.InvoiceParty{}
	if body.Customer != nil {
		customer = domain.InvoiceParty{
			Name:    strings.TrimSpace(body.Customer.Name),
			Address: strings.TrimSpace(body.Customer.Address),
			TRN:     jurisdiction.NormalizeTaxID(body.Customer.TRN),
		}
	}
	if !rules.ValidTaxID(customer.TRN) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": fmt.Sprintf("customer TRN must be %d digits", rules.TaxIDDigits)})
	}
	if customer.Name == "" && h.customers != nil {
		if name, err := h.customers.CustomerForDeal(c.Context(), deal.ID); err == nil {
			customer.Name = name
		}
	}
	if customer.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "customer name is required"})
	}

	var supplyDate *time.Time
	if body.SupplyDate != "" {
		d, err := time.Parse("2006-01-02", body.SupplyDate)
		if err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "supply_date must be YYYY-MM-DD"})
		}
		supplyDate = &d
	}

	inv := &domain.VATInvoice{
		Kind:       domain.KindInvoice,
		DealID:     deal.ID,
		Currency:   currency,
		SupplyDate: supplyDate,
		Supplier:   domain.InvoiceParty{Name: company.Name, Address: company.BusinessAddress, TRN: supplierTRN},
		Customer:   customer,
		Notes:      strings.TrimSpace(body.Notes),
		Status:     domain.InvoiceDraft,
		CreatedBy:  localsUserPtr(c),
	}
	applyTotals(inv, totals, rules)
	if err := h.invoices.Issue(c.Context(), inv); err != nil {
		return serverError(c, err)
	}
	h.logAudit(c, repo.AuditCreate, inv.ID, fiber.Map{"invoice_no": inv.InvoiceNo, "total": inv.Total, "currency": inv.Currency})
	return c.Status(fiber.StatusCreated).JSON(inv)
}

type creditNoteRequest struct {
	Reason string         `json:"reason"`
	Lines  []invoice.Line `json:"lines"`
}

// CreateCreditNote godoc
// @Summary      Issue a credit note
// @Description  Issues a credit note (CN-YYYY-NNNN) against an invoice. Without lines, the invoice's lines are credited in full (only possible while nothing has been credited yet). The total of an invoice's credit notes can never exceed the invoice total.
// @Tags         Invoices
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "Invoice UUID"
// @Param        body  body      creditNoteRequest  true  "Credit note"
// @Success      201   {object}  domain.VATInvoice
// @Failure      422   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices/{id}/credit-notes [post]
func (h *InvoiceHandler) CreateCreditNote(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body creditNoteRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if body.Reason == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "reason is required"})
	}
	orig, err := h.invoices.GetByID(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if orig.Kind != domain.KindInvoice {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "credit notes can only be issued against an invoice"})
	}
	company, err := h.company.Get(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	rules := jurisdiction.MustFor(company.Country)

	lines := body.Lines
	if len(lines) == 0 {
		for _, l := range orig.Lines {
			lines = append(lines, invoice.Line{
				Description: l.Description, Quantity: l.Quantity, UnitPrice: l.UnitPrice,
				VATCategory: jurisdiction.VATCategory(l.VATCategory),
			})
		}
	}
	// A credit note follows the invoice's VAT treatment: the supplier's registration is
	// judged by the original document, not by today's settings.
	totals, err := invoice.Compute(lines, rules, orig.Currency, orig.Supplier.TRN != "")
	if err != nil {
		return serverError(c, err)
	}
	cn := &domain.VATInvoice{
		Kind:              domain.KindCreditNote,
		DealID:            orig.DealID,
		OriginalInvoiceID: &orig.ID,
		Currency:          orig.Currency,
		Supplier:          orig.Supplier,
		Customer:          orig.Customer,
		Reason:            body.Reason,
		Status:            domain.InvoiceDraft,
		CreatedBy:         localsUserPtr(c),
	}
	applyTotals(cn, totals, rules)
	if err := h.invoices.Issue(c.Context(), cn); err != nil {
		return serverError(c, err)
	}
	h.logAudit(c, repo.AuditCreate, cn.ID, fiber.Map{"credit_note_no": cn.InvoiceNo, "invoice_no": orig.InvoiceNo, "total": cn.Total, "reason": cn.Reason})
	return c.Status(fiber.StatusCreated).JSON(cn)
}

// ListCreditNotes godoc
// @Summary      List an invoice's credit notes
// @Tags         Invoices
// @Produce      json
// @Param        id  path  string  true  "Invoice UUID"
// @Success      200  {array}  domain.VATInvoice
// @Security     BearerAuth
// @Router       /invoices/{id}/credit-notes [get]
func (h *InvoiceHandler) ListCreditNotes(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	notes, err := h.invoices.ListCreditNotes(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if notes == nil {
		notes = []domain.VATInvoice{}
	}
	return c.JSON(notes)
}

func applyTotals(inv *domain.VATInvoice, t *invoice.Totals, rules jurisdiction.Rules) {
	inv.Subtotal, inv.VATAmount, inv.Total = t.Subtotal, t.VATAmount, t.Total
	inv.VATRate, _ = rules.VATRate(jurisdiction.VATStandard)
	inv.Lines = make([]domain.InvoiceLine, len(t.Lines))
	for i, l := range t.Lines {
		inv.Lines[i] = domain.InvoiceLine{
			Description: strings.TrimSpace(l.Description), Quantity: l.Quantity, UnitPrice: l.UnitPrice,
			VATCategory: string(l.VATCategory), VATRate: l.VATRate, NetAmount: l.NetAmount, VATAmount: l.VATAmount,
		}
	}
}

func localsUserPtr(c *fiber.Ctx) *uuid.UUID {
	if id, ok := c.Locals("user_id").(uuid.UUID); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

func (h *InvoiceHandler) logAudit(c *fiber.Ctx, action string, id uuid.UUID, diff any) {
	if h.audit == nil {
		return
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, action, repo.AuditInvoice, id, diff)
	}
}

// Get godoc
// @Summary      Get invoice or credit note, with lines
// @Tags         Invoices
// @Produce      json
// @Param        id  path      string  true  "Invoice UUID"
// @Success      200  {object}  domain.VATInvoice
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
// @Failure      409  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices/{id}/send [post]
func (h *InvoiceHandler) Send(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.invoices.UpdateStatus(c.Context(), id, domain.InvoiceSent, ""); err != nil {
		return serverError(c, err)
	}
	return c.JSON(fiber.Map{"id": id, "status": domain.InvoiceSent})
}

// UpdateStatus godoc
// @Summary      Update invoice status
// @Description  Set status to draft|sent|paid|void. Voiding needs a reason and is refused for paid documents and for invoices with credit notes; a void document never changes again. Credit notes cannot be marked paid.
// @Tags         Invoices
// @Accept       json
// @Produce      json
// @Param        id    path      string                                 true  "Invoice UUID"
// @Param        body  body      object{status=string,reason=string}    true  "New status"
// @Success      200   {object}  object{id=string,status=string}
// @Failure      409   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /invoices/{id}/status [patch]
func (h *InvoiceHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Status domain.InvoiceStatus `json:"status"`
		Reason string               `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status required"})
	}
	switch body.Status {
	case domain.InvoiceDraft, domain.InvoiceSent, domain.InvoicePaid:
	case domain.InvoiceVoid:
		body.Reason = strings.TrimSpace(body.Reason)
		if body.Reason == "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "a reason is required to void a document"})
		}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}
	if err := h.invoices.UpdateStatus(c.Context(), id, body.Status, body.Reason); err != nil {
		return serverError(c, err)
	}
	h.logAudit(c, repo.AuditUpdate, id, fiber.Map{"status": body.Status, "reason": body.Reason})
	return c.JSON(fiber.Map{"id": id, "status": body.Status})
}

// DownloadPDF godoc
// @Summary      Download invoice as PDF
// @Description  PDF generation is part of the Pro edition. In the community edition use the printable invoice page in the web app (Invoices → Print).
// @Tags         Invoices
// @Produce      pdf
// @Param        id  path  string  true  "Invoice UUID"
// @Success      200  {file}  application/pdf
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
	pdfBytes, err := pdf.GenerateInvoice(pdf.InvoiceData{
		InvoiceNo:   inv.InvoiceNo,
		IssuedAt:    inv.IssuedAt,
		Subtotal:    inv.Subtotal,
		VATRate:     inv.VATRate,
		VATAmount:   inv.VATAmount,
		Total:       inv.Total,
		DealTitle:   deal.Title,
		CompanyName: inv.Supplier.Name,
		CompanyAddr: inv.Supplier.Address,
		CompanyVAT:  inv.Supplier.TRN,
	})
	if err != nil {
		return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{"error": "PDF generation is not available in this edition; use the printable invoice page"})
	}
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": safeFilename(filename)}))
	return c.Send(pdfBytes)
}
