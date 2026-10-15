package handler

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/kyc"
	"github.com/maidulcu/masaar-crm/internal/mediastore"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// KYCHandler serves customer due diligence (identity documents, risk, screening,
// beneficial owners) and the AML review of deals. Every read of identity documents and
// of the AML queue is audit-logged.
type KYCHandler struct {
	kyc      *repo.KYCRepo
	audit    *repo.AuditLogRepo
	store    *mediastore.Store // private document store; may be nil
	maxBytes int64
	screener kyc.Screener // optional
}

func NewKYCHandler(k *repo.KYCRepo, audit *repo.AuditLogRepo, store *mediastore.Store, maxMB int64) *KYCHandler {
	if maxMB <= 0 {
		maxMB = 10
	}
	return &KYCHandler{kyc: k, audit: audit, store: store, maxBytes: maxMB << 20}
}

// UseScreener plugs in a sanctions/PEP screening provider.
func (h *KYCHandler) UseScreener(s kyc.Screener) *KYCHandler {
	h.screener = s
	return h
}

func (h *KYCHandler) log(c *fiber.Ctx, action, entity string, id uuid.UUID, diff fiber.Map) {
	if actor := localsUserPtr(c); actor != nil && h.audit != nil {
		h.audit.Log(c.Context(), *actor, action, entity, id, diff)
	}
}

func idParam(c *fiber.Ctx, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Params(name))
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
		return uuid.Nil, false
	}
	return id, true
}

// Profile godoc
// @Summary      A contact's KYC record
// @Description  Identity documents, risk rating, PEP flag, screening result, and what is missing. Audit-logged.
// @Tags         KYC
// @Produce      json
// @Param        id  path  string  true  "Contact ID"
// @Success      200  {object}  repo.KYCProfile
// @Security     BearerAuth
// @Router       /contacts/{id}/kyc [get]
func (h *KYCHandler) Profile(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	p, err := h.kyc.Profile(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	h.log(c, "kyc_view", "contact", id, nil)
	return c.JSON(p)
}

// UpdateProfile godoc
// @Summary      Record a contact's risk rating, PEP flag and screening result
// @Description  Admins and compliance officers. The screening result is entered by hand unless a provider is configured.
// @Tags         KYC
// @Accept       json
// @Param        id    path  string  true  "Contact ID"
// @Param        body  body  object{date_of_birth=string,risk=string,risk_reason=string,is_pep=bool,screening=string,screening_source=string,notes=string}  true  "Record"
// @Success      200  {object}  repo.KYCProfile
// @Security     BearerAuth
// @Router       /contacts/{id}/kyc [put]
func (h *KYCHandler) UpdateProfile(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	var body struct {
		DateOfBirth     string `json:"date_of_birth"`
		Risk            string `json:"risk"`
		RiskReason      string `json:"risk_reason"`
		IsPEP           bool   `json:"is_pep"`
		Screening       string `json:"screening"`
		ScreeningSource string `json:"screening_source"`
		Notes           string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if body.Screening == "" {
		body.Screening = kyc.NotScreened
	}
	switch {
	case body.Risk != "" && body.Risk != "low" && body.Risk != "medium" && body.Risk != "high":
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "risk must be low, medium or high"})
	case !kyc.ValidScreening(body.Screening):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "unknown screening result"})
	case (body.Risk == "high" || body.IsPEP) && strings.TrimSpace(body.RiskReason) == "":
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "give a reason for a high risk rating or a PEP"})
	}
	dob, err := parseDay(body.DateOfBirth)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "date_of_birth must be YYYY-MM-DD"})
	}
	actor := localsUserPtr(c)
	if actor == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	p := &repo.KYCProfile{ContactID: id, DateOfBirth: dob, Risk: body.Risk, RiskReason: strings.TrimSpace(body.RiskReason),
		IsPEP: body.IsPEP, Screening: body.Screening, ScreeningSource: strings.TrimSpace(body.ScreeningSource), Notes: body.Notes}
	if err := h.kyc.UpdateProfile(c.Context(), p, *actor); err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditUpdate, "contact_kyc", id, fiber.Map{"risk": body.Risk, "is_pep": body.IsPEP, "screening": body.Screening})
	got, err := h.kyc.Profile(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(got)
}

// Screen godoc
// @Summary      Screen a contact with the configured provider
// @Description  Admins and compliance officers. Returns 501 when no provider is configured (none is bundled).
// @Tags         KYC
// @Param        id  path  string  true  "Contact ID"
// @Success      200  {object}  repo.KYCProfile
// @Failure      501  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /contacts/{id}/kyc/screen [post]
func (h *KYCHandler) Screen(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	if h.screener == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": kyc.ErrNoScreener.Error()})
	}
	p, err := h.kyc.Profile(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	subj := kyc.Subject{Name: p.FullName, DateOfBirth: p.DateOfBirth, Nationality: p.Nationality}
	for _, d := range p.Documents {
		if d.DocType == kyc.EmiratesID || d.DocType == kyc.Passport {
			subj.IDNumber = d.Number
			break
		}
	}
	res, err := h.screener.Screen(c.Context(), subj)
	if err != nil {
		log.Printf("kyc: screening %s: %v", id, err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "the screening provider did not answer"})
	}
	if !kyc.ValidScreening(res.Status) || res.Status == kyc.NotScreened {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "the screening provider gave an unknown result"})
	}
	actor := localsUserPtr(c)
	if actor == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if err := h.kyc.RecordScreening(c.Context(), id, res, h.screener.Name(), *actor); err != nil {
		return serverError(c, err)
	}
	h.log(c, "kyc_screen", "contact", id, fiber.Map{"provider": h.screener.Name(), "result": res.Status, "reference": res.Reference})
	got, err := h.kyc.Profile(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(got)
}

var kycFileTypes = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "application/pdf": ".pdf"}

// AddDocument godoc
// @Summary      Add an identity document
// @Description  Multipart form: doc_type (emirates_id, passport, residence_visa, trade_licence, other), number, issuing_country (ISO alpha-2), issue_date, expiry_date (YYYY-MM-DD), and an optional file (JPEG, PNG or PDF), stored privately.
// @Tags         KYC
// @Accept       multipart/form-data
// @Produce      json
// @Param        id  path  string  true  "Contact ID"
// @Success      201  {object}  repo.KYCDocument
// @Security     BearerAuth
// @Router       /contacts/{id}/kyc/documents [post]
func (h *KYCHandler) AddDocument(c *fiber.Ctx) error {
	contactID, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	d := &repo.KYCDocument{ContactID: contactID, DocType: c.FormValue("doc_type"), Number: strings.TrimSpace(c.FormValue("number")),
		IssuingCountry: strings.ToUpper(strings.TrimSpace(c.FormValue("issuing_country")))}
	if !kyc.ValidDocType(d.DocType) || d.Number == "" || len(d.Number) > 64 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "doc_type and number are required"})
	}
	if d.IssuingCountry != "" && len(d.IssuingCountry) != 2 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "issuing_country must be a two-letter ISO code"})
	}
	var err error
	if d.IssueDate, err = parseDay(c.FormValue("issue_date")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "issue_date must be YYYY-MM-DD"})
	}
	if d.ExpiryDate, err = parseDay(c.FormValue("expiry_date")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "expiry_date must be YYYY-MM-DD"})
	}
	key := ""
	if file, ferr := c.FormFile("file"); ferr == nil {
		if h.store == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "file storage is not available"})
		}
		if file.Size > h.maxBytes {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file is too large"})
		}
		f, err := file.Open()
		if err != nil {
			return serverError(c, err)
		}
		defer f.Close()
		head := make([]byte, 512)
		n, _ := io.ReadFull(f, head)
		head = head[:n]
		if _, ok := kycFileTypes[http.DetectContentType(head)]; !ok {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "the copy must be a JPEG, PNG or PDF file"})
		}
		if key, _, err = h.store.Save(io.MultiReader(bytes.NewReader(head), f), h.maxBytes); err != nil {
			if errors.Is(err, mediastore.ErrTooLarge) {
				return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file is too large"})
			}
			return serverError(c, err)
		}
		d.FileName = safeFilename(filepath.Base(file.Filename))
	}
	if err := h.kyc.AddDocument(c.Context(), d, key, localsUserPtr(c)); err != nil {
		if key != "" {
			_ = h.store.Remove(key)
		}
		return serverError(c, err)
	}
	h.log(c, repo.AuditCreate, "kyc_document", d.ID, fiber.Map{"contact_id": contactID, "doc_type": d.DocType})
	return c.Status(fiber.StatusCreated).JSON(d)
}

// DocumentFile godoc
// @Summary      Download the copy of an identity document
// @Description  Audit-logged.
// @Tags         KYC
// @Param        id  path  string  true  "Document ID"
// @Success      200
// @Security     BearerAuth
// @Router       /kyc-documents/{id}/file [get]
func (h *KYCHandler) DocumentFile(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	key, name, contactID, err := h.kyc.DocumentFile(c.Context(), id)
	if err != nil || key == "" || h.store == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}
	f, err := h.store.Open(key)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		return serverError(c, err)
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return serverError(c, err)
	}
	ctype := http.DetectContentType(head[:n])
	if _, ok := kycFileTypes[ctype]; !ok {
		ctype = fiber.MIMEOctetStream
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return serverError(c, err)
	}
	h.log(c, "kyc_document_download", "kyc_document", id, fiber.Map{"contact_id": contactID})
	if name == "" {
		name = "document" + kycFileTypes[ctype]
	}
	c.Set(fiber.HeaderContentType, ctype)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderContentDisposition, mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	return c.SendStream(f, int(st.Size()))
}

// DeleteDocument godoc
// @Summary      Delete an identity document and its copy
// @Tags         KYC
// @Param        id  path  string  true  "Document ID"
// @Success      204
// @Security     BearerAuth
// @Router       /kyc-documents/{id} [delete]
func (h *KYCHandler) DeleteDocument(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	key, contactID, err := h.kyc.DeleteDocument(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if key != "" && h.store != nil {
		if err := h.store.Remove(key); err != nil {
			log.Printf("kyc: remove file: %v", err)
		}
	}
	h.log(c, repo.AuditDelete, "kyc_document", id, fiber.Map{"contact_id": contactID})
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Beneficial owners ─────────────────────────────────────────────────────────

// Owners godoc
// @Summary      A corporate client's beneficial owners
// @Tags         KYC
// @Produce      json
// @Param        id  path  string  true  "Organisation ID"
// @Success      200  {array}  repo.BeneficialOwner
// @Security     BearerAuth
// @Router       /organisations/{id}/owners [get]
func (h *KYCHandler) Owners(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	list, err := h.kyc.Owners(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(list)
}

// SetOwner godoc
// @Summary      Add or update a beneficial owner
// @Tags         KYC
// @Accept       json
// @Param        id    path  string  true  "Organisation ID"
// @Param        body  body  object{contact_id=string,ownership_pct=number,role=string}  true  "Owner (role: owner, controller, signatory)"
// @Success      204
// @Security     BearerAuth
// @Router       /organisations/{id}/owners [post]
func (h *KYCHandler) SetOwner(c *fiber.Ctx) error {
	orgID, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	var body struct {
		ContactID    uuid.UUID `json:"contact_id"`
		OwnershipPct *float64  `json:"ownership_pct"`
		Role         string    `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if body.Role == "" {
		body.Role = "owner"
	}
	if body.Role != "owner" && body.Role != "controller" && body.Role != "signatory" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "role must be owner, controller or signatory"})
	}
	if p := body.OwnershipPct; p != nil && (*p <= 0 || *p > 100) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "ownership_pct must be between 0 and 100"})
	}
	if err := h.kyc.SetOwner(c.Context(), orgID, body.ContactID, body.OwnershipPct, body.Role); err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditUpdate, "organisation", orgID, fiber.Map{"beneficial_owner": body.ContactID, "role": body.Role, "pct": body.OwnershipPct})
	return c.SendStatus(fiber.StatusNoContent)
}

// RemoveOwner godoc
// @Summary      Remove a beneficial owner
// @Tags         KYC
// @Param        id          path  string  true  "Organisation ID"
// @Param        contact_id  path  string  true  "Contact ID"
// @Success      204
// @Security     BearerAuth
// @Router       /organisations/{id}/owners/{contact_id} [delete]
func (h *KYCHandler) RemoveOwner(c *fiber.Ctx) error {
	orgID, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	contactID, ok := idParam(c, "contact_id")
	if !ok {
		return nil
	}
	if err := h.kyc.RemoveOwner(c.Context(), orgID, contactID); err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditUpdate, "organisation", orgID, fiber.Map{"beneficial_owner_removed": contactID})
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Deals ─────────────────────────────────────────────────────────────────────

// DealKYC godoc
// @Summary      KYC status of a deal's buyer
// @Tags         KYC
// @Produce      json
// @Param        id  path  string  true  "Deal ID"
// @Success      200  {object}  repo.DealKYC
// @Security     BearerAuth
// @Router       /deals/{id}/kyc [get]
func (h *KYCHandler) DealKYC(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	k, err := h.kyc.DealKYC(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(k)
}

// OverrideDealKYC godoc
// @Summary      Allow a deal to be won without complete KYC
// @Description  Admin only; the reason is required and audit-logged.
// @Tags         KYC
// @Accept       json
// @Param        id    path  string  true  "Deal ID"
// @Param        body  body  object{reason=string}  true  "Reason"
// @Success      204
// @Security     BearerAuth
// @Router       /deals/{id}/kyc-override [post]
func (h *KYCHandler) OverrideDealKYC(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if len(body.Reason) < 10 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "give a reason (at least 10 characters)"})
	}
	actor := localsUserPtr(c)
	if actor == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if err := h.kyc.OverrideDealKYC(c.Context(), id, body.Reason, *actor); err != nil {
		return serverError(c, err)
	}
	h.log(c, "kyc_override", "deal", id, fiber.Map{"reason": body.Reason})
	return c.SendStatus(fiber.StatusNoContent)
}

// SetDealPayment godoc
// @Summary      Record how a deal is paid
// @Description  payment_method: cash, cheque, transfer, virtual_asset, mixed (or empty); cash_amount: the cash part. A deal that meets the company's review criteria (cash at or above the threshold, virtual assets) goes to the AML review queue.
// @Tags         KYC
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "Deal ID"
// @Param        body  body  object{payment_method=string,cash_amount=number}  true  "Payment"
// @Success      200  {object}  object{aml_status=string,aml_reasons=[]string}
// @Security     BearerAuth
// @Router       /deals/{id}/payment [put]
func (h *KYCHandler) SetDealPayment(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	var body struct {
		PaymentMethod string  `json:"payment_method"`
		CashAmount    float64 `json:"cash_amount"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if !kyc.ValidPaymentMethod(body.PaymentMethod) || body.CashAmount < 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid payment method or cash amount"})
	}
	status, reasons, err := h.kyc.SetDealPayment(c.Context(), id, body.PaymentMethod, body.CashAmount)
	if err != nil {
		return serverError(c, err)
	}
	h.log(c, repo.AuditUpdate, "deal", id, fiber.Map{"payment_method": body.PaymentMethod, "cash_amount": body.CashAmount, "aml_status": status})
	return c.JSON(fiber.Map{"aml_status": status, "aml_reasons": reasons})
}

// ── AML queue ─────────────────────────────────────────────────────────────────

// AMLQueue godoc
// @Summary      AML review queue
// @Description  Admins and compliance officers; audit-logged. status: flagged (default), no_report, reported, or all.
// @Tags         Compliance
// @Produce      json
// @Success      200  {array}  repo.AMLCase
// @Security     BearerAuth
// @Router       /compliance/aml [get]
func (h *KYCHandler) AMLQueue(c *fiber.Ctx) error {
	status := c.Query("status", "flagged")
	if status == "all" {
		status = ""
	}
	if status != "" && status != "flagged" && status != "no_report" && status != "reported" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unknown status"})
	}
	list, err := h.kyc.AMLCases(c.Context(), status)
	if err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, "aml_queue_view", "compliance", *actor, fiber.Map{"status": status, "count": len(list)})
	}
	return c.JSON(list)
}

// ReviewAML godoc
// @Summary      Record the review of a flagged deal
// @Description  decision: no_report (with a note) or reported (with the goAML reference). Audit-logged.
// @Tags         Compliance
// @Accept       json
// @Param        id    path  string  true  "Deal ID"
// @Param        body  body  object{decision=string,notes=string,goaml_reference=string}  true  "Decision"
// @Success      204
// @Security     BearerAuth
// @Router       /compliance/aml/{id}/review [post]
func (h *KYCHandler) ReviewAML(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	var body struct {
		Decision       string `json:"decision"`
		Notes          string `json:"notes"`
		GoAMLReference string `json:"goaml_reference"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	body.Notes, body.GoAMLReference = strings.TrimSpace(body.Notes), strings.TrimSpace(body.GoAMLReference)
	switch {
	case body.Decision != "no_report" && body.Decision != "reported":
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "decision must be no_report or reported"})
	case body.Decision == "no_report" && body.Notes == "":
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "explain why no report is needed"})
	case body.Decision == "reported" && body.GoAMLReference == "":
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "the goAML reference is required"})
	}
	actor := localsUserPtr(c)
	if actor == nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if err := h.kyc.ReviewAML(c.Context(), id, body.Decision, body.Notes, body.GoAMLReference, *actor); err != nil {
		return serverError(c, err)
	}
	h.log(c, "aml_review", "deal", id, fiber.Map{"decision": body.Decision, "goaml_reference": body.GoAMLReference})
	return c.SendStatus(fiber.StatusNoContent)
}

func dayOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// AMLExport godoc
// @Summary      Export AML cases as CSV (goAML preparation)
// @Description  Admins and compliance officers; audit-logged. The columns hold what a goAML report needs about the transaction, the client and their identity document, the property and the agent. The file is input for preparing the report in the goAML portal; it is not a goAML XML submission.
// @Tags         Compliance
// @Produce      text/csv
// @Param        from    query  string  false  "YYYY-MM-DD (default: 1 year ago)"
// @Param        to      query  string  false  "YYYY-MM-DD, exclusive (default: tomorrow)"
// @Param        status  query  string  false  "flagged | no_report | reported (default: all)"
// @Success      200
// @Security     BearerAuth
// @Router       /compliance/aml/export.csv [get]
func (h *KYCHandler) AMLExport(c *fiber.Ctx) error {
	to := time.Now().UTC().AddDate(0, 0, 1).Truncate(24 * time.Hour)
	from := to.AddDate(-1, 0, 0)
	if d, err := parseDay(c.Query("from")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "from must be YYYY-MM-DD"})
	} else if d != nil {
		from = *d
	}
	if d, err := parseDay(c.Query("to")); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "to must be YYYY-MM-DD"})
	} else if d != nil {
		to = *d
	}
	status := c.Query("status")
	if status != "" && status != "flagged" && status != "no_report" && status != "reported" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unknown status"})
	}
	rows, err := h.kyc.AMLExport(c.Context(), status, from, to)
	if err != nil {
		return serverError(c, err)
	}
	var buf bytes.Buffer
	buf.WriteString("\ufeff") // so spreadsheet apps read UTF-8 (Arabic names)
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"deal_id", "deal_title", "deal_stage", "transaction_date", "flagged_at", "amount", "currency", "payment_method",
		"cash_amount", "review_status", "flag_reasons", "goaml_reference",
		"client_name", "client_nationality", "client_date_of_birth", "client_email", "client_phone", "client_address",
		"client_risk", "client_pep", "client_screening", "id_type", "id_number", "id_issuing_country", "id_expiry",
		"property_title", "property_reference", "property_emirate", "property_community", "agent_name", "agent_brn"})
	for _, r := range rows {
		_ = w.Write(csvSafeRow([]string{r.DealID, r.Title, r.Stage, dayOrEmpty(r.DealDate), dayOrEmpty(r.FlaggedAt),
			fmt.Sprintf("%.3f", r.Amount), r.Currency, r.PaymentMethod, fmt.Sprintf("%.3f", r.CashAmount), r.Status, r.Reasons, r.GoAMLRef,
			r.ClientName, r.ClientNationality, dayOrEmpty(r.ClientDOB), r.ClientEmail, r.ClientPhone, r.ClientAddress,
			r.ClientRisk, fmt.Sprint(r.ClientPEP), r.ClientScreening, r.IDType, r.IDNumber, r.IDCountry, dayOrEmpty(r.IDExpiry),
			r.PropertyTitle, r.PropertyReference, r.PropertyEmirate, r.PropertyCommunity, r.AgentName, r.AgentBRN}))
	}
	w.Flush()
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, "aml_export", "compliance", *actor, fiber.Map{"from": from.Format("2006-01-02"),
			"to": to.Format("2006-01-02"), "status": status, "rows": len(rows)})
	}
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="aml-cases-`+from.Format("20060102")+`.csv"`)
	return c.Send(buf.Bytes())
}

// GetAMLSettings godoc
// @Summary      AML review criteria
// @Tags         Compliance
// @Produce      json
// @Success      200  {object}  kyc.AMLSettings
// @Security     BearerAuth
// @Router       /settings/aml [get]
func (h *KYCHandler) GetAMLSettings(c *fiber.Ctx) error {
	s, err := h.kyc.AMLSettings(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(s)
}

// UpdateAMLSettings godoc
// @Summary      Update the AML review criteria
// @Description  Admin only. cash_threshold: cash at or above this sends a deal to review (AED 55,000 by default; check the current figure).
// @Tags         Compliance
// @Accept       json
// @Param        body  body  kyc.AMLSettings  true  "Criteria"
// @Success      200  {object}  kyc.AMLSettings
// @Security     BearerAuth
// @Router       /settings/aml [put]
func (h *KYCHandler) UpdateAMLSettings(c *fiber.Ctx) error {
	var s kyc.AMLSettings
	if err := c.BodyParser(&s); err != nil {
		return badRequest(c, err)
	}
	if s.CashThreshold < 0 || s.CashThreshold > 1e12 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid threshold"})
	}
	if err := h.kyc.UpdateAMLSettings(c.Context(), s); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		if cid, err := uuid.Parse(fmtLocal(c, "company_id")); err == nil {
			h.audit.Log(c.Context(), *actor, repo.AuditUpdate, "company_settings", cid, fiber.Map{"aml": s})
		}
	}
	return c.JSON(s)
}
