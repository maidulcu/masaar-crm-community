package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/phone"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type ContactHandler struct {
	contacts *repo.ContactRepo
	audit    *repo.AuditLogRepo
	privacy  consentStore
	profiles *repo.ContactProfileRepo
}

// UseProfiles returns and accepts the contact profile (secondary phones, nationality,
// organisation, …) on GET and PATCH /contacts/:id.
func (h *ContactHandler) UseProfiles(p *repo.ContactProfileRepo) *ContactHandler {
	h.profiles = p
	return h
}

// UsePrivacy records consent given when a contact is created.
func (h *ContactHandler) UsePrivacy(p consentStore) *ContactHandler {
	h.privacy = p
	return h
}

func NewContactHandler(contacts *repo.ContactRepo, audit *repo.AuditLogRepo) *ContactHandler {
	return &ContactHandler{contacts: contacts, audit: audit}
}

// List godoc
// @Summary      List contacts
// @Description  Returns a paginated list of contacts. Supports keyword search on name, phone, and email.
// @Tags         Contacts
// @Produce      json
// @Param        search  query     string  false  "Keyword search"
// @Param        page    query     int     false  "Page number (default 1)"
// @Param        limit   query     int     false  "Page size 1-100 (default 20)"
// @Success      200     {object}  domain.PaginatedResult[domain.Contact]
// @Security     BearerAuth
// @Router       /contacts [get]
func (h *ContactHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	search := c.Query("search", "")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	result, err := h.contacts.List(c.Context(), search, page, limit)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get contact
// @Description  Returns a single contact by UUID.
// @Tags         Contacts
// @Produce      json
// @Param        id  path      string  true  "Contact UUID"
// @Success      200  {object}  domain.Contact
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /contacts/{id} [get]
func (h *ContactHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	contact, err := h.contacts.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "contact not found"})
	}
	if h.profiles != nil {
		if p, err := h.profiles.GetProfile(c.Context(), id); err == nil {
			contact.Profile = p
		}
	}
	return c.JSON(contact)
}

// Create godoc
// @Summary      Create contact
// @Description  Creates a new contact. phone_wa and full_name are required.
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        body  body      domain.Contact  true  "Contact payload"
// @Success      201   {object}  domain.Contact
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /contacts [post]
func (h *ContactHandler) Create(c *fiber.Ctx) error {
	var contact domain.Contact
	if err := c.BodyParser(&contact); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	// Optional {"consent": {"whatsapp": true, "email": false}} alongside the contact fields.
	var extra struct {
		Consent map[string]bool `json:"consent"`
	}
	_ = c.BodyParser(&extra)
	// Server-owned fields are never taken from the client.
	contact.ID, contact.CreatedAt, contact.UpdatedAt = uuid.Nil, time.Time{}, time.Time{}
	contact.FullName = strings.TrimSpace(contact.FullName)
	if contact.PhoneWA == "" || contact.FullName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "phone_wa and full_name are required"})
	}
	// Accept the formats people type ("971 50 123 4567", "00971…") and store one canonical form.
	if normalized, ok := phone.Normalize(contact.PhoneWA); ok {
		contact.PhoneWA = normalized
	}
	// Fast byte inspection replaces regex matching for E.164 phone validation (~11ns vs ~200ns)
	if !phone.IsE164(contact.PhoneWA) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "phone_wa must be in E.164 format (e.g. +971501234567)"})
	}
	if contact.Language == "" {
		contact.Language = "ar"
	}
	if msg := validateContactFields(&contact.FullName, &contact.Email, &contact.Language, &contact.LeadScore); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
	}

	if err := h.contacts.Create(c.Context(), &contact); err != nil {
		return serverError(c, err)
	}
	consentSource := "manual"
	if actorUserID(c) == nil {
		consentSource = "api"
	}
	recordConsentChoices(c.Context(), h.privacy, contact.ID, extra.Consent, consentSource, actorUserID(c), actorAPIKeyID(c))
	auditWrite(c, h.audit, repo.AuditCreate, repo.AuditContact, contact.ID, contact)
	return c.Status(fiber.StatusCreated).JSON(contact)
}

type contactUpdateRequest struct {
	FullName   *string                `json:"full_name"`
	Email      *string                `json:"email"` // "" clears it
	Language   *string                `json:"language"`
	LeadScore  *int                   `json:"lead_score"`
	AssignedTo *uuid.UUID             `json:"assigned_to"` // see assigned_to handling in Update: null clears it
	Profile    *domain.ContactProfile `json:"profile"`     // replaces the whole profile when given
}

// cleanProfile validates and normalises a profile; it returns a client message or "".
func cleanProfile(p *domain.ContactProfile) string {
	if len(p.SecondaryPhones) > 10 {
		return "at most 10 secondary phones"
	}
	phones := make([]string, 0, len(p.SecondaryPhones))
	for _, raw := range p.SecondaryPhones {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		n, ok := phone.Normalize(raw)
		if !ok {
			return "secondary phones must be valid international numbers"
		}
		phones = append(phones, n)
	}
	p.SecondaryPhones = phones
	p.Nationality = strings.ToUpper(strings.TrimSpace(p.Nationality))
	if p.Nationality != "" && len(p.Nationality) != 2 {
		return "nationality must be a 2-letter country code"
	}
	for _, f := range []*string{&p.Source, &p.PreferredContactTime, &p.JobTitle} {
		*f = strings.TrimSpace(*f)
		if len(*f) > 200 {
			return "a profile field is too long"
		}
	}
	if len(p.Address) > 1000 || len(p.Notes) > 10000 {
		return "address or notes are too long"
	}
	return ""
}

// Update godoc
// @Summary      Update contact
// @Description  Partial update — only provided fields are changed. An empty email clears it and `"assigned_to": null` unassigns the contact.
// @Tags         Contacts
// @Accept       json
// @Produce      json
// @Param        id    path      string          true  "Contact UUID"
// @Param        body  body      domain.Contact  true  "Fields to update"
// @Success      200   {object}  domain.Contact
// @Failure      400   {object}  object{error=string}
// @Failure      404   {object}  object{error=string}
// @Failure      422   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /contacts/{id} [patch]
func (h *ContactHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	existing, err := h.contacts.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "contact not found"})
	}

	// Merge only provided fields
	var patch contactUpdateRequest
	if err := c.BodyParser(&patch); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	// A pointer cannot tell `"assigned_to": null` (unassign) from an absent key, so look at the keys.
	var keys map[string]json.RawMessage
	_ = json.Unmarshal(c.Body(), &keys)

	if patch.FullName != nil {
		existing.FullName = *patch.FullName
	}
	if patch.Email != nil {
		existing.Email = *patch.Email
	}
	if patch.Language != nil {
		existing.Language = *patch.Language
	}
	if patch.LeadScore != nil {
		existing.LeadScore = *patch.LeadScore
	}
	if _, present := keys["assigned_to"]; present {
		existing.AssignedTo = patch.AssignedTo // nil when null
	}
	// Validate only what this request changes: a legacy value in a field the caller did not touch
	// (say an email stored before validation existed) must not block an unrelated edit.
	name, email, lang, score := existing.FullName, existing.Email, existing.Language, existing.LeadScore
	if patch.FullName == nil {
		name = "-"
	}
	if patch.Email == nil {
		email = ""
	}
	if patch.Language == nil {
		lang = "en"
	}
	if patch.LeadScore == nil {
		score = 0
	}
	if msg := validateContactFields(&name, &email, &lang, &score); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
	}
	if patch.FullName != nil {
		existing.FullName = name
	}
	if patch.Email != nil {
		existing.Email = email
	}
	if patch.Language != nil {
		existing.Language = lang
	}
	if patch.LeadScore != nil {
		existing.LeadScore = score
	}

	if patch.Profile != nil {
		if msg := cleanProfile(patch.Profile); msg != "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
		}
	}

	if err := h.contacts.Update(c.Context(), existing); err != nil {
		if errors.Is(err, pgx.ErrNoRows) && existing.AssignedTo != nil {
			// The contact was just read, so the only way for the update to match nothing is an
			// assignee who is not an active admin/agent of this company.
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "assignee must be an active admin or agent of this company"})
		}
		return serverError(c, err)
	}
	if patch.Profile != nil && h.profiles != nil {
		if err := h.profiles.UpdateProfile(c.Context(), existing.ID, patch.Profile); err != nil {
			return serverError(c, err)
		}
		existing.Profile, _ = h.profiles.GetProfile(c.Context(), existing.ID)
	}
	auditWrite(c, h.audit, repo.AuditUpdate, repo.AuditContact, existing.ID, existing)
	return c.JSON(existing)
}

// Delete godoc
// @Summary      Delete contact
// @Description  Permanently deletes a contact. Requires admin role. Deleting also removes the contact's leads, deals, offers, viewings and WhatsApp threads, so when any exist the request is refused with 409 and the counts unless `?force=true` is given. A contact whose deals have invoices can never be deleted.
// @Tags         Contacts
// @Param        id     path   string  true   "Contact UUID"
// @Param        force  query  bool    false  "Also delete the contact's linked records"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Failure      409  {object}  object{error=string,linked=object}
// @Security     BearerAuth
// @Router       /contacts/{id} [delete]
func (h *ContactHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if _, err := h.contacts.GetByID(c.Context(), id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "contact not found"})
	}

	linked, err := h.contacts.LinkedRecords(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if linked.Invoices > 0 {
		// vat_invoices is ON DELETE RESTRICT: these are accounting records.
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":  "this contact has invoices and cannot be deleted",
			"linked": linked,
		})
	}
	if linked.Any() && !c.QueryBool("force", false) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":  "this contact has linked leads, deals or conversations that would be deleted too; confirm with force=true",
			"linked": linked,
		})
	}

	if err := h.contacts.Delete(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	actorID := c.Locals("user_id").(uuid.UUID)
	h.audit.Log(c.Context(), actorID, repo.AuditDelete, repo.AuditContact, id, fiber.Map{"linked_deleted": linked})
	return c.SendStatus(fiber.StatusNoContent)
}
