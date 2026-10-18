package handler

import (
	"errors"
	"net/mail"
	"regexp"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/webhook"
)

var e164PublicRe = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

// externalSourceRe names the pushing system: lowercase, short, e.g. "waqov" or "zapier".
var externalSourceRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)

// externalRefRe is the pushing system's own id for the lead, e.g. "enquiry:48213".
var externalRefRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)

// publicLeadSources are the sources an API client may set. bos24 is internal only.
var publicLeadSources = map[string]bool{"web": true, "referral": true, "event": true, "whatsapp": true,
	"property_finder": true, "bayut": true, "dubizzle": true, "portal": true}

type PublicLeadHandler struct {
	contacts   *repo.ContactRepo
	leads      *repo.LeadRepo
	audit      *repo.AuditLogRepo
	dispatcher *webhook.Dispatcher
	privacy    consentStore
}

// UsePrivacy records the consent submitted with each lead.
func (h *PublicLeadHandler) UsePrivacy(p consentStore) *PublicLeadHandler {
	h.privacy = p
	return h
}

func NewPublicLeadHandler(contacts *repo.ContactRepo, leads *repo.LeadRepo, audit *repo.AuditLogRepo, dispatcher *webhook.Dispatcher) *PublicLeadHandler {
	return &PublicLeadHandler{contacts: contacts, leads: leads, audit: audit, dispatcher: dispatcher}
}

type PublicLeadRequest struct {
	Name      string  `json:"name"`
	Phone     string  `json:"phone"`
	Email     string  `json:"email"`
	Language  string  `json:"language"` // ar | en (default: ar)
	Source    string  `json:"source"`   // web|referral|event|whatsapp|property_finder|bayut|dubizzle|portal (default: web)
	Notes     string  `json:"notes"`
	DealValue float64 `json:"deal_value"`
	Currency  string  `json:"currency"` // default: AED
	// Optional metadata stored in notes
	PropertyType string `json:"property_type"` // e.g. "2BR", "villa"
	Area         string `json:"area"`          // e.g. "Marina", "Downtown"
	// Optional, sent together: the pushing system and its own id for this lead.
	// A repeat of the same pair returns the existing lead (200) instead of a new one.
	ExternalSource string `json:"external_source"` // e.g. "waqov"
	ExternalRef    string `json:"external_ref"`    // e.g. "enquiry:48213"
	// Consent the person gave on the form, per channel (whatsapp, email, sms, phone):
	// true = opted in, false = declined. Omitted channels record nothing.
	Consent map[string]bool `json:"consent"`
}

// SubmitLead creates a lead from an external source using API key auth.
// @Summary Submit a lead (public)
// @Description Create a lead from an external source (website, Zapier, etc.)
// @Description Requires API key with scope lead:create in Authorization header.
// @Tags Public
// @Accept json
// @Produce json
// @Param request body PublicLeadRequest true "Lead data"
// @Success 201 {object} map[string]interface{}
// @Success 200 {object} map[string]interface{} "Duplicate external_ref: the existing lead"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 422 {object} map[string]string
// @Router /webhooks/leads [post]
func (h *PublicLeadHandler) SubmitLead(c *fiber.Ctx) error {
	var req PublicLeadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	// Required fields
	if req.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "name is required"})
	}
	if len(req.Name) > 200 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "name is too long (max 200 characters)"})
	}
	if req.Phone == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "phone is required"})
	}
	if !e164PublicRe.MatchString(req.Phone) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "phone must be in E.164 format (e.g. +971501234567)",
		})
	}

	// Validate input lengths to prevent DoS / payload bloat
	if len(req.Email) > 254 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "email is too long (max 254 characters)"})
	}
	if req.Email != "" {
		if _, err := mail.ParseAddress(req.Email); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid email format"})
		}
	}
	if len(req.Notes) > 2000 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "notes are too long (max 2000 characters)"})
	}
	if len(req.PropertyType) > 100 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "property_type is too long (max 100 characters)"})
	}
	if len(req.Area) > 100 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "area is too long (max 100 characters)"})
	}

	// Defaults
	if req.Language == "" {
		req.Language = "ar"
	}
	if req.Source == "" {
		req.Source = "web"
	}
	if req.Currency == "" {
		req.Currency = defaultCurrency(c.Context())
	}

	if !publicLeadSources[req.Source] {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "source must be one of: web, referral, event, whatsapp, property_finder, bayut, dubizzle, portal",
		})
	}
	if (req.ExternalSource == "") != (req.ExternalRef == "") {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "external_source and external_ref must be sent together"})
	}
	if req.ExternalSource != "" && !externalSourceRe.MatchString(req.ExternalSource) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "external_source must be 1-32 lowercase letters, digits, '.', '_' or '-'",
		})
	}
	if req.ExternalRef != "" && !externalRefRe.MatchString(req.ExternalRef) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "external_ref must be 1-128 letters, digits, '.', '_', ':', '/' or '-'",
		})
	}
	if req.Source == "portal" && req.ExternalSource == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "source portal requires external_source and external_ref"})
	}

	// Enrich notes with property metadata if provided
	notes := req.Notes
	if req.PropertyType != "" || req.Area != "" {
		meta := ""
		if req.Area != "" {
			meta += "Area: " + req.Area
		}
		if req.PropertyType != "" {
			if meta != "" {
				meta += " | "
			}
			meta += "Type: " + req.PropertyType
		}
		if notes != "" {
			notes = meta + " | " + notes
		} else {
			notes = meta
		}
	}

	ctx := c.Context()

	// A retried push returns the lead it already made, before touching the contact.
	if req.ExternalRef != "" {
		if existing, err := h.leads.FindByExternalRef(ctx, req.ExternalSource, req.ExternalRef); err == nil {
			return duplicateLeadResponse(c, existing)
		}
	}

	// Upsert contact — creates if not found; an existing contact keeps the name it already has
	contact, err := h.contacts.Upsert(ctx, req.Phone, req.Name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create contact",
		})
	}

	// Patch email/language onto the contact if provided
	if req.Email != "" || req.Language != "" {
		if req.Email != "" {
			contact.Email = req.Email
		}
		if req.Language != "" {
			contact.Language = req.Language
		}
		_ = h.contacts.Update(ctx, contact)
	}

	if len(req.Consent) > 0 {
		var keyID *uuid.UUID
		if id, ok := c.Locals("api_key_id").(uuid.UUID); ok {
			keyID = &id
		}
		recordConsentChoices(ctx, h.privacy, contact.ID, req.Consent, "public_form", nil, keyID)
	}

	// Create lead
	lead := &domain.Lead{
		ContactID: contact.ID,
		Stage:     domain.StageNew,
		Source:    domain.LeadSource(req.Source),
		DealValue: req.DealValue,
		Currency:  req.Currency,
		Notes:     notes,

		ExternalSource: req.ExternalSource,
		ExternalRef:    req.ExternalRef,
	}

	if err := h.leads.Create(ctx, lead); err != nil {
		// Two pushes of the same lead raced; the other one won.
		if errors.Is(err, repo.ErrDuplicateExternalRef) {
			if existing, findErr := h.leads.FindByExternalRef(ctx, req.ExternalSource, req.ExternalRef); findErr == nil {
				return duplicateLeadResponse(c, existing)
			}
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create lead",
		})
	}

	if keyID, ok := c.Locals("api_key_id").(uuid.UUID); ok {
		h.audit.LogAPIKey(ctx, keyID, repo.AuditCreate, repo.AuditLead, lead.ID, fiber.Map{
			"source":          lead.Source,
			"external_source": lead.ExternalSource,
			"external_ref":    lead.ExternalRef,
		})
	}

	// Fire outbound webhook asynchronously
	var companyID uuid.UUID
	if cidStr, ok := c.Locals("company_id").(string); ok {
		companyID, _ = uuid.Parse(cidStr)
	}
	h.dispatcher.Dispatch(companyID, webhook.EventLeadCreated, fiber.Map{
		"lead_id":    lead.ID,
		"contact_id": contact.ID,
		"contact":    fiber.Map{"name": contact.FullName, "phone": contact.PhoneWA},
		"stage":      lead.Stage,
		"source":     lead.Source,
		"deal_value": lead.DealValue,

		"external_source": lead.ExternalSource,
		"external_ref":    lead.ExternalRef,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"lead_id":    lead.ID,
		"contact_id": contact.ID,
		"stage":      lead.Stage,
		"source":     lead.Source,
		"message":    "Lead created successfully",

		"external_source": lead.ExternalSource,
		"external_ref":    lead.ExternalRef,
		"duplicate":       false,
	})
}

// duplicateLeadResponse answers a repeated push with the lead the first push made.
func duplicateLeadResponse(c *fiber.Ctx, l *domain.Lead) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"lead_id":         l.ID,
		"contact_id":      l.ContactID,
		"stage":           l.Stage,
		"source":          l.Source,
		"message":         "Lead already exists for this external_ref",
		"external_source": l.ExternalSource,
		"external_ref":    l.ExternalRef,
		"duplicate":       true,
	})
}
