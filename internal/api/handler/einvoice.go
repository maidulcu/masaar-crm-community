package handler

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/einvoice"
	"github.com/maidulcu/masaar-crm/internal/jobs"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// EInvoiceHandler turns tax invoices and credit notes into PINT AE e-invoices and submits
// them through the company's provider (#88).
type EInvoiceHandler struct {
	einvoices *repo.EInvoiceRepo
	invoices  *repo.InvoiceRepo
	audit     *repo.AuditLogRepo
	notifier  *jobs.Notifier
}

func NewEInvoiceHandler(e *repo.EInvoiceRepo, invoices *repo.InvoiceRepo, audit *repo.AuditLogRepo, n *jobs.Notifier) *EInvoiceHandler {
	return &EInvoiceHandler{einvoices: e, invoices: invoices, audit: audit, notifier: n}
}

// Credit note reasons (BTAE-03), from the UAE VAT law (Federal Decree-Law No. 8 of 2017,
// Article 61(1)) as coded in PINT AE. VD needs no preceding invoice.
var creditReasons = map[string]bool{"DL8.61.1.A": true, "DL8.61.1.B": true, "DL8.61.1.C": true, "DL8.61.1.D": true, "DL8.61.1.E": true, "VD": true}

type einvoiceBody struct {
	Buyer        einvoice.Party `json:"buyer"`
	CreditReason string         `json:"credit_reason"`
	DueDate      string         `json:"due_date"` // YYYY-MM-DD; default 30 days after issue
}

// document builds the e-invoice for an invoice.
func (h *EInvoiceHandler) document(ctx context.Context, inv *domain.VATInvoice, b einvoiceBody) (*einvoice.Document, *time.Time, error) {
	settings, err := h.einvoices.Settings(ctx)
	if err != nil {
		return nil, nil, err
	}
	seller, iban, err := h.einvoices.Seller(ctx)
	if err != nil {
		return nil, nil, err
	}
	b.Buyer.Country = strings.ToUpper(strings.TrimSpace(b.Buyer.Country))
	if b.Buyer.Country == "" {
		b.Buyer.Country = "AE"
	}
	if b.Buyer.Name == "" {
		b.Buyer.Name = inv.Customer.Name
	}
	if b.Buyer.TRN == "" {
		b.Buyer.TRN = inv.Customer.TRN
	}
	d := &einvoice.Document{
		Kind: einvoice.KindInvoice, Number: inv.InvoiceNo, UUID: inv.ID.String(), IssueDate: inv.IssuedAt, Currency: strings.TrimSpace(inv.Currency),
		Seller: seller, Buyer: b.Buyer, Note: inv.Notes, IBAN: iban, ServiceSAC: settings.ServiceCode, ExemptionCode: settings.ExemptionCode,
	}
	for _, l := range inv.Lines {
		d.Lines = append(d.Lines, einvoice.Line{Description: l.Description, Quantity: l.Quantity, UnitPrice: l.UnitPrice,
			Net: l.NetAmount, Category: l.VATCategory, Rate: l.VATRate, VAT: l.VATAmount})
	}
	var due *time.Time
	if inv.Kind == domain.KindCreditNote {
		d.Kind, d.CreditReason = einvoice.KindCreditNote, b.CreditReason
		if inv.OriginalInvoiceID != nil {
			orig, err := h.invoices.GetByID(ctx, *inv.OriginalInvoiceID)
			if err != nil {
				return nil, nil, err
			}
			d.PrecedingNumber, d.PrecedingDate = orig.InvoiceNo, orig.IssuedAt
		}
	} else {
		dd := inv.IssuedAt.AddDate(0, 0, 30)
		if b.DueDate != "" {
			parsed, err := time.Parse("2006-01-02", b.DueDate)
			if err != nil {
				return nil, nil, &einvoice.MissingError{Fields: []string{"a due date as YYYY-MM-DD"}}
			}
			dd = parsed
		}
		due, d.DueDate = &dd, &dd
	}
	return d, due, nil
}

func (h *EInvoiceHandler) load(c *fiber.Ctx) (*domain.VATInvoice, *einvoiceBody, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return nil, nil, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var b einvoiceBody
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&b); err != nil {
			return nil, nil, badRequest(c, err)
		}
	}
	inv, err := h.invoices.GetByID(c.Context(), id)
	if err != nil {
		return nil, nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "invoice not found"})
	}
	if inv.Status == domain.InvoiceVoid {
		return nil, nil, c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "a void invoice cannot be e-invoiced"})
	}
	if inv.Kind == domain.KindCreditNote && b.CreditReason != "" && !creditReasons[b.CreditReason] {
		return nil, nil, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "unknown credit note reason code"})
	}
	return inv, &b, nil
}

func missing(c *fiber.Ctx, err error) error {
	var me *einvoice.MissingError
	if errors.As(err, &me) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": me.Error(), "missing": me.Fields})
	}
	return serverError(c, err)
}

// Status godoc
// @Summary      An invoice's e-invoice status
// @Tags         E-invoicing
// @Produce      json
// @Param        id  path  string  true  "Invoice ID"
// @Success      200  {object}  object{provider=string,einvoice=repo.EInvoice,credit_reasons=[]string}
// @Security     BearerAuth
// @Router       /invoices/{id}/einvoice [get]
func (h *EInvoiceHandler) Status(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	settings, err := h.einvoices.Settings(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	resp := fiber.Map{"provider": settings.Provider, "einvoice": nil,
		"credit_reasons": []string{"DL8.61.1.A", "DL8.61.1.B", "DL8.61.1.C", "DL8.61.1.D", "DL8.61.1.E", "VD"}}
	e, err := h.einvoices.Get(c.Context(), id)
	switch {
	case err == nil:
		resp["einvoice"] = e
	case !errors.Is(err, pgx.ErrNoRows):
		return serverError(c, err)
	}
	return c.JSON(resp)
}

// ExportXML godoc
// @Summary      Download an invoice as PINT AE XML
// @Description  Builds the e-invoice without submitting it (also when e-invoicing is off). Answers 422 with the list of missing data.
// @Tags         E-invoicing
// @Accept       json
// @Produce      xml
// @Param        id    path  string        true  "Invoice ID"
// @Param        body  body  einvoiceBody  false  "Buyer details, credit note reason, due date"
// @Success      200
// @Failure      422  {object}  object{error=string,missing=[]string}
// @Security     BearerAuth
// @Router       /invoices/{id}/einvoice/xml [post]
func (h *EInvoiceHandler) ExportXML(c *fiber.Ctx) error {
	inv, b, err := h.load(c)
	if inv == nil {
		return err
	}
	d, _, err := h.document(c.Context(), inv, *b)
	if err != nil {
		return missing(c, err)
	}
	xml, err := d.XML()
	if err != nil {
		return missing(c, err)
	}
	c.Set(fiber.HeaderContentType, "application/xml; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+safeFilename(inv.InvoiceNo)+`.xml"`)
	return c.Send(xml)
}

// Submit godoc
// @Summary      Submit an invoice as an e-invoice
// @Description  Builds the PINT AE e-invoice and hands it to the company's provider. A rejected invoice can be submitted again after the data is corrected.
// @Tags         E-invoicing
// @Accept       json
// @Produce      json
// @Param        id    path  string        true  "Invoice ID"
// @Param        body  body  einvoiceBody  true  "Buyer details, credit note reason, due date"
// @Success      201  {object}  repo.EInvoice
// @Failure      409  {object}  object{error=string}
// @Failure      422  {object}  object{error=string,missing=[]string}
// @Security     BearerAuth
// @Router       /invoices/{id}/einvoice [post]
func (h *EInvoiceHandler) Submit(c *fiber.Ctx) error {
	settings, err := h.einvoices.Settings(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	if settings.Provider == "" || settings.Provider == "off" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": einvoice.ErrNotEnabled.Error()})
	}
	conn, ok := einvoice.Lookup(settings.Provider)
	if !ok {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": einvoice.ErrUnknownProvider.Error()})
	}
	inv, b, err := h.load(c)
	if inv == nil {
		return err
	}
	d, due, err := h.document(c.Context(), inv, *b)
	if err != nil {
		return missing(c, err)
	}
	xml, err := d.XML()
	if err != nil {
		return missing(c, err)
	}
	rec := &repo.EInvoice{InvoiceID: inv.ID, Provider: conn.Name(), Status: einvoice.StatusSubmitted, Buyer: d.Buyer,
		CreditReason: d.CreditReason, DueDate: due, XML: string(xml), SubmittedBy: localsUserPtr(c)}
	ref, subErr := conn.Submit(c.Context(), einvoice.Submission{Kind: d.Kind, Number: d.Number, XML: xml})
	if subErr != nil {
		log.Printf("einvoice: submit %s: %v", inv.InvoiceNo, subErr)
		rec.Status, rec.Reason = einvoice.StatusError, "the provider did not accept the submission: "+subErr.Error()
	}
	rec.ProviderRef = ref
	if err := h.einvoices.Save(c.Context(), rec); err != nil {
		if errors.Is(err, repo.ErrAlreadySubmitted) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, "einvoice_submit", repo.AuditInvoice, inv.ID, fiber.Map{"provider": conn.Name(), "status": rec.Status, "ref": ref})
	}
	if subErr == nil {
		// Ask for the decision straight away; the e-invoice status job keeps asking until there is one.
		if res, err := conn.Status(c.Context(), ref); err == nil {
			jobs.ApplyEInvoiceDecision(c.Context(), h.einvoices, h.notifier,
				repo.PendingEInvoice{InvoiceID: inv.ID, InvoiceNo: inv.InvoiceNo, Issuer: inv.CreatedBy, SubmittedBy: rec.SubmittedBy}, res)
		}
	}
	got, err := h.einvoices.Get(c.Context(), inv.ID)
	if err != nil {
		return serverError(c, err)
	}
	code := fiber.StatusCreated
	if got.Status == einvoice.StatusError {
		code = fiber.StatusBadGateway
	}
	return c.Status(code).JSON(got)
}

// StoredXML godoc
// @Summary      The XML that was submitted for an invoice
// @Tags         E-invoicing
// @Produce      xml
// @Param        id  path  string  true  "Invoice ID"
// @Success      200
// @Security     BearerAuth
// @Router       /invoices/{id}/einvoice.xml [get]
func (h *EInvoiceHandler) StoredXML(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	e, err := h.einvoices.Get(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	c.Set(fiber.HeaderContentType, "application/xml; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="einvoice.xml"`)
	return c.SendString(e.XML)
}

// GetSettings godoc
// @Summary      E-invoicing settings
// @Tags         Settings
// @Produce      json
// @Success      200  {object}  object{settings=repo.EInvoiceSettings,providers=[]string}
// @Security     BearerAuth
// @Router       /settings/einvoicing [get]
func (h *EInvoiceHandler) GetSettings(c *fiber.Ctx) error {
	s, err := h.einvoices.Settings(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(fiber.Map{"settings": s, "providers": append([]string{"off"}, einvoice.Providers()...)})
}

// UpdateSettings godoc
// @Summary      Update the e-invoicing settings
// @Description  provider: off (PDF only, the default) or a registered connector ("mock" for trying the workflow). tin: the 10-digit tax identification number used as the e-invoicing participant ID.
// @Tags         Settings
// @Accept       json
// @Param        body  body  repo.EInvoiceSettings  true  "Settings"
// @Success      200  {object}  repo.EInvoiceSettings
// @Security     BearerAuth
// @Router       /settings/einvoicing [put]
func (h *EInvoiceHandler) UpdateSettings(c *fiber.Ctx) error {
	var s repo.EInvoiceSettings
	if err := c.BodyParser(&s); err != nil {
		return badRequest(c, err)
	}
	s.TIN = strings.TrimSpace(s.TIN)
	if s.Provider == "" {
		s.Provider = "off"
	}
	if _, ok := einvoice.Lookup(s.Provider); !ok && s.Provider != "off" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "unknown provider"})
	}
	if s.TIN != "" && (len(s.TIN) != 10 || strings.Trim(s.TIN, "0123456789") != "") {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "the TIN has 10 digits"})
	}
	switch s.LegalIDType {
	case "":
		s.LegalIDType = einvoice.LegalTradeLicence
	case einvoice.LegalTradeLicence, einvoice.LegalEmiratesID, einvoice.LegalPassport, einvoice.LegalCabinet:
	default:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "legal_id_type must be TL, EID, PAS or CD"})
	}
	for _, v := range []string{s.TradeLicenseAuthority, s.City, s.ServiceCode, s.ExemptionCode} {
		if len(v) > 200 {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "a value is too long"})
		}
	}
	if err := h.einvoices.UpdateSettings(c.Context(), &s); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		if cid, err := uuid.Parse(fmtLocal(c, "company_id")); err == nil {
			h.audit.Log(c.Context(), *actor, repo.AuditUpdate, "company_settings", cid, fiber.Map{"einvoicing": s})
		}
	}
	return c.JSON(s)
}
