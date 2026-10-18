package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/rera"
)

// ReraHandler pre-fills RERA brokerage forms (A, B, I, F) and stores them as documents
// that can be sent for e-signature.
type ReraHandler struct {
	forms *repo.ReraRepo
	audit auditLogger
}

func NewReraHandler(forms *repo.ReraRepo, audit auditLogger) *ReraHandler {
	return &ReraHandler{forms: forms, audit: audit}
}

// Definitions godoc
// @Summary      RERA form definitions
// @Description  The fields of Forms A, B, I and F with English and Arabic labels.
// @Tags         RERA forms
// @Produce      json
// @Success      200  {object}  map[string]rera.Form
// @Security     BearerAuth
// @Router       /rera-forms/definitions [get]
func (h *ReraHandler) Definitions(c *fiber.Ctx) error {
	return c.JSON(rera.Forms)
}

type createReraRequest struct {
	FormType  string            `json:"form_type"` // A | B | I | F
	ListingID *uuid.UUID        `json:"listing_id"`
	ContactID *uuid.UUID        `json:"contact_id"`
	OfferID   *uuid.UUID        `json:"offer_id"`
	Fields    map[string]string `json:"fields"` // overrides of the pre-filled values
}

// Create godoc
// @Summary      Generate a pre-filled RERA form
// @Description  Form A and I need listing_id, Form B contact_id, Form F offer_id. Values are pre-filled from the company, listing, agent, contact and offer; `fields` overrides them. The form is stored as a pending document; send it for signature with POST /documents/{document_id}/request-signature. Masaar does not submit forms to the DLD.
// @Tags         RERA forms
// @Accept       json
// @Produce      json
// @Param        body  body  createReraRequest  true  "Form"
// @Success      201  {object}  repo.ReraForm
// @Security     BearerAuth
// @Router       /rera-forms [post]
func (h *ReraHandler) Create(c *fiber.Ctx) error {
	var body createReraRequest
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	def, err := rera.Lookup(body.FormType)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "form_type must be A, B, I or F"})
	}
	actor := localsUserPtr(c)
	if actor == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var relatedType string
	var relatedID *uuid.UUID
	switch def.Type {
	case "A", "I":
		if body.ListingID == nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "listing_id is required for Form " + def.Type})
		}
		relatedType, relatedID = "listing", body.ListingID
	case "B":
		if body.ContactID == nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "contact_id is required for Form B"})
		}
		relatedType, relatedID = "contact", body.ContactID
	case "F":
		if body.OfferID == nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "offer_id is required for Form F"})
		}
		relatedType, relatedID = "offer", body.OfferID
	}
	prefill, err := h.forms.Prefill(c.Context(), body.ListingID, body.ContactID, body.OfferID, *actor)
	if err != nil {
		return serverError(c, err)
	}
	data, err := def.Merge(prefill, body.Fields)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	title := def.TitleEN
	if p := data["property_title"]; p != "" {
		title += " — " + p
	} else if b := data["buyer_name"]; b != "" {
		title += " — " + b
	}
	f := &repo.ReraForm{FormType: def.Type, ListingID: body.ListingID, ContactID: body.ContactID, OfferID: body.OfferID,
		Data: data, Title: title}
	if def.Type == "F" {
		// The offer's listing and buyer were resolved during pre-fill; keep the links.
		f.ListingID, f.ContactID = nil, nil
	}
	if err := h.forms.Create(c.Context(), f, *actor, relatedType, relatedID); err != nil {
		return serverError(c, err)
	}
	if h.audit != nil {
		h.audit.Log(c.Context(), *actor, repo.AuditCreate, repo.AuditDocument, f.DocumentID, fiber.Map{"rera_form": def.Type})
	}
	return c.Status(fiber.StatusCreated).JSON(f)
}

// List godoc
// @Summary      RERA forms for a listing or contact
// @Tags         RERA forms
// @Produce      json
// @Param        listing_id  query  string  false  "Listing UUID"
// @Param        contact_id  query  string  false  "Contact UUID"
// @Success      200  {array}  repo.ReraForm
// @Security     BearerAuth
// @Router       /rera-forms [get]
func (h *ReraHandler) List(c *fiber.Ctx) error {
	parse := func(k string) (*uuid.UUID, bool) {
		s := c.Query(k)
		if s == "" {
			return nil, true
		}
		id, err := uuid.Parse(s)
		return &id, err == nil
	}
	listingID, ok1 := parse("listing_id")
	contactID, ok2 := parse("contact_id")
	if !ok1 || !ok2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	forms, err := h.forms.List(c.Context(), listingID, contactID)
	if err != nil {
		return serverError(c, err)
	}
	if forms == nil {
		forms = []repo.ReraForm{}
	}
	return c.JSON(forms)
}

// Get godoc
// @Summary      A RERA form with its definition
// @Tags         RERA forms
// @Produce      json
// @Param        id  path  string  true  "Document UUID"
// @Success      200  {object}  object{form=repo.ReraForm,definition=rera.Form}
// @Security     BearerAuth
// @Router       /rera-forms/{id} [get]
func (h *ReraHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	f, err := h.forms.Get(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	def, _ := rera.Lookup(f.FormType)
	return c.JSON(fiber.Map{"form": f, "definition": def})
}
