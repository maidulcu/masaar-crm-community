package handler

import (
	"regexp"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/webhook"
)

var e164PublicRe = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

type PublicLeadHandler struct {
	contacts   *repo.ContactRepo
	leads      *repo.LeadRepo
	dispatcher *webhook.Dispatcher
}

func NewPublicLeadHandler(contacts *repo.ContactRepo, leads *repo.LeadRepo, dispatcher *webhook.Dispatcher) *PublicLeadHandler {
	return &PublicLeadHandler{contacts: contacts, leads: leads, dispatcher: dispatcher}
}

type PublicLeadRequest struct {
	Name        string  `json:"name"`
	Phone       string  `json:"phone"`
	Email       string  `json:"email"`
	Language    string  `json:"language"`    // ar | en (default: ar)
	Source      string  `json:"source"`      // web|referral|event (default: web)
	Notes       string  `json:"notes"`
	DealValue   float64 `json:"deal_value"`
	Currency    string  `json:"currency"`    // default: AED
	// Optional metadata stored in notes
	PropertyType string `json:"property_type"` // e.g. "2BR", "villa"
	Area         string `json:"area"`          // e.g. "Marina", "Downtown"
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
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
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
	if req.Phone == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "phone is required"})
	}
	if !e164PublicRe.MatchString(req.Phone) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "phone must be in E.164 format (e.g. +971501234567)",
		})
	}

	// Defaults
	if req.Language == "" {
		req.Language = "ar"
	}
	if req.Source == "" {
		req.Source = "web"
	}
	if req.Currency == "" {
		req.Currency = "AED"
	}

	// Validate source
	validSources := map[string]bool{"web": true, "referral": true, "event": true, "whatsapp": true}
	if !validSources[req.Source] {
		req.Source = "web"
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

	// Upsert contact — creates if not found, updates name if phone exists
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

	// Create lead
	lead := &domain.Lead{
		ContactID: contact.ID,
		Stage:     domain.StageNew,
		Source:    domain.LeadSource(req.Source),
		DealValue: req.DealValue,
		Currency:  req.Currency,
		Notes:     notes,
	}

	if err := h.leads.Create(ctx, lead); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create lead",
		})
	}

	// Fire outbound webhook asynchronously
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))
	h.dispatcher.Dispatch(companyID, webhook.EventLeadCreated, fiber.Map{
		"lead_id":    lead.ID,
		"contact_id": contact.ID,
		"contact":    fiber.Map{"name": contact.FullName, "phone": contact.PhoneWA},
		"stage":      lead.Stage,
		"source":     lead.Source,
		"deal_value": lead.DealValue,
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"lead_id":    lead.ID,
		"contact_id": contact.ID,
		"stage":      lead.Stage,
		"source":     lead.Source,
		"message":    "Lead created successfully",
	})
}
