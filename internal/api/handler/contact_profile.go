package handler

import (
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// ContactProfileHandler serves organisations, custom fields and duplicate merging.
type ContactProfileHandler struct {
	profiles *repo.ContactProfileRepo
	audit    auditLogger
}

func NewContactProfileHandler(profiles *repo.ContactProfileRepo, audit auditLogger) *ContactProfileHandler {
	return &ContactProfileHandler{profiles: profiles, audit: audit}
}

// ── Organisations ────────────────────────────────────────────────────────────

// ListOrganisations godoc
// @Summary      List organisations
// @Tags         Organisations
// @Produce      json
// @Param        search  query  string  false  "Name contains"
// @Success      200  {object}  object{data=[]domain.Organisation,total=int}
// @Security     BearerAuth
// @Router       /organisations [get]
func (h *ContactProfileHandler) ListOrganisations(c *fiber.Ctx) error {
	page, limit := pageParams(c, 50, 200)
	orgs, total, err := h.profiles.ListOrganisations(c.Context(), strings.TrimSpace(c.Query("search")), limit, (page-1)*limit)
	if err != nil {
		return serverError(c, err)
	}
	if orgs == nil {
		orgs = []domain.Organisation{}
	}
	return c.JSON(fiber.Map{"data": orgs, "total": total})
}

// GetOrganisation godoc
// @Summary      Get an organisation and its contacts
// @Tags         Organisations
// @Produce      json
// @Param        id  path  string  true  "Organisation UUID"
// @Success      200  {object}  object{organisation=domain.Organisation,contacts=[]domain.Contact}
// @Security     BearerAuth
// @Router       /organisations/{id} [get]
func (h *ContactProfileHandler) GetOrganisation(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	org, err := h.profiles.GetOrganisation(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	contacts, err := h.profiles.OrganisationContacts(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if contacts == nil {
		contacts = []domain.Contact{}
	}
	return c.JSON(fiber.Map{"organisation": org, "contacts": contacts})
}

func cleanOrg(o *domain.Organisation) string {
	o.Name = strings.TrimSpace(o.Name)
	if o.Name == "" || len(o.Name) > 300 {
		return "name is required (up to 300 characters)"
	}
	for _, f := range []*string{&o.TradeLicenseNo, &o.TRN, &o.Phone, &o.Email, &o.Website} {
		*f = strings.TrimSpace(*f)
		if len(*f) > 300 {
			return "a field is too long"
		}
	}
	if len(o.Address) > 1000 || len(o.Notes) > 10000 {
		return "address or notes are too long"
	}
	return ""
}

// SaveOrganisation godoc
// @Summary      Create (POST /organisations) or update (PATCH /organisations/{id}) an organisation
// @Tags         Organisations
// @Accept       json
// @Produce      json
// @Param        body  body  domain.Organisation  true  "Organisation"
// @Success      200  {object}  domain.Organisation
// @Security     BearerAuth
// @Router       /organisations [post]
func (h *ContactProfileHandler) SaveOrganisation(c *fiber.Ctx) error {
	var o domain.Organisation
	if err := c.BodyParser(&o); err != nil {
		return badRequest(c, err)
	}
	o.ID = uuid.Nil
	if idStr := c.Params("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
		}
		o.ID = id
	}
	if msg := cleanOrg(&o); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
	}
	created := o.ID == uuid.Nil
	if err := h.profiles.SaveOrganisation(c.Context(), &o); err != nil {
		return serverError(c, err)
	}
	saved, err := h.profiles.GetOrganisation(c.Context(), o.ID)
	if err != nil {
		return serverError(c, err)
	}
	if created {
		return c.Status(fiber.StatusCreated).JSON(saved)
	}
	return c.JSON(saved)
}

// DeleteOrganisation godoc
// @Summary      Delete an organisation (its contacts are kept)
// @Tags         Organisations
// @Param        id  path  string  true  "Organisation UUID"
// @Success      204
// @Security     BearerAuth
// @Router       /organisations/{id} [delete]
func (h *ContactProfileHandler) DeleteOrganisation(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.profiles.DeleteOrganisation(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Custom fields ────────────────────────────────────────────────────────────

var customFieldName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)

// ListCustomFields godoc
// @Summary      List custom field definitions
// @Tags         Custom fields
// @Produce      json
// @Param        entity_type  query  string  false  "contact | lead | deal | listing"
// @Success      200  {array}  domain.CustomField
// @Security     BearerAuth
// @Router       /custom-fields [get]
func (h *ContactProfileHandler) ListCustomFields(c *fiber.Ctx) error {
	fields, err := h.profiles.ListCustomFields(c.Context(), c.Query("entity_type"))
	if err != nil {
		return serverError(c, err)
	}
	if fields == nil {
		fields = []domain.CustomField{}
	}
	return c.JSON(fields)
}

// SaveCustomField godoc
// @Summary      Define (POST) or change (PATCH /custom-fields/{id}) a custom field
// @Description  Admin only. name is a machine name ([a-z][a-z0-9_]*) and, like entity_type, cannot change later.
// @Tags         Custom fields
// @Accept       json
// @Produce      json
// @Param        body  body  domain.CustomField  true  "Field"
// @Success      200  {object}  domain.CustomField
// @Security     BearerAuth
// @Router       /custom-fields [post]
func (h *ContactProfileHandler) SaveCustomField(c *fiber.Ctx) error {
	var f domain.CustomField
	if err := c.BodyParser(&f); err != nil {
		return badRequest(c, err)
	}
	f.ID = uuid.Nil
	if idStr := c.Params("id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
		}
		f.ID = id
	}
	f.Label = strings.TrimSpace(f.Label)
	if f.Type == "" {
		f.Type = "text"
	}
	switch {
	case f.ID == uuid.Nil && !customFieldName.MatchString(f.Name):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "name must start with a letter and use a-z, 0-9 and _ (max 50)"})
	case f.ID == uuid.Nil && f.EntityType != "contact" && f.EntityType != "lead" && f.EntityType != "deal" && f.EntityType != "listing":
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "entity_type must be contact, lead, deal or listing"})
	case f.Label == "" || len(f.Label) > 150:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "label is required (up to 150 characters)"})
	case f.Type != "text" && f.Type != "number" && f.Type != "date" && f.Type != "select" && f.Type != "boolean":
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "type must be text, number, date, select or boolean"})
	case f.Type == "select" && (len(f.Options) == 0 || len(f.Options) > 100):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "a select field needs 1–100 options"})
	}
	created := f.ID == uuid.Nil
	if err := h.profiles.SaveCustomField(c.Context(), &f); err != nil {
		return serverError(c, err)
	}
	if created {
		return c.Status(fiber.StatusCreated).JSON(f)
	}
	return c.JSON(f)
}

// DeleteCustomField godoc
// @Summary      Delete a custom field and all its values
// @Tags         Custom fields
// @Param        id  path  string  true  "Field UUID"
// @Success      204
// @Security     BearerAuth
// @Router       /custom-fields/{id} [delete]
func (h *ContactProfileHandler) DeleteCustomField(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.profiles.DeleteCustomField(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// GetCustomValues godoc
// @Summary      A record's custom field values
// @Tags         Custom fields
// @Produce      json
// @Param        entity_type  path  string  true  "contact | lead | deal | listing"
// @Param        id           path  string  true  "Record UUID"
// @Success      200  {object}  map[string]string
// @Security     BearerAuth
// @Router       /custom-fields/values/{entity_type}/{id} [get]
func (h *ContactProfileHandler) GetCustomValues(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	values, err := h.profiles.CustomValues(c.Context(), c.Params("entity_type"), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(values)
}

// SetCustomValues godoc
// @Summary      Set a record's custom field values
// @Description  Body is {field_name: value}; an empty value clears the field.
// @Tags         Custom fields
// @Accept       json
// @Param        entity_type  path  string             true  "contact | lead | deal | listing"
// @Param        id           path  string             true  "Record UUID"
// @Param        body         body  map[string]string  true  "Values"
// @Success      204
// @Security     BearerAuth
// @Router       /custom-fields/values/{entity_type}/{id} [put]
func (h *ContactProfileHandler) SetCustomValues(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var values map[string]string
	if err := c.BodyParser(&values); err != nil {
		return badRequest(c, err)
	}
	if len(values) > 200 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "too many values"})
	}
	if err := h.profiles.SetCustomValues(c.Context(), c.Params("entity_type"), id, values); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Duplicates ───────────────────────────────────────────────────────────────

// Duplicates godoc
// @Summary      Contacts that look like duplicates
// @Description  Groups of contacts sharing an email, a full name, or a phone number (one's WhatsApp number is the other's secondary phone).
// @Tags         Contacts
// @Produce      json
// @Success      200  {array}  domain.DuplicateGroup
// @Security     BearerAuth
// @Router       /contacts/duplicates [get]
func (h *ContactProfileHandler) Duplicates(c *fiber.Ctx) error {
	groups, err := h.profiles.FindDuplicates(c.Context(), 200)
	if err != nil {
		return serverError(c, err)
	}
	if groups == nil {
		groups = []domain.DuplicateGroup{}
	}
	return c.JSON(groups)
}

// Merge godoc
// @Summary      Merge duplicate contacts
// @Description  Admin only. Moves everything from merge_ids into keep_id (leads, offers, viewings, conversations, consent, tasks, custom values), fills keep_id's empty fields and deletes the others. Irreversible; audit-logged.
// @Tags         Contacts
// @Accept       json
// @Param        body  body  object{keep_id=string,merge_ids=[]string}  true  "Contacts"
// @Success      204
// @Security     BearerAuth
// @Router       /contacts/merge [post]
func (h *ContactProfileHandler) Merge(c *fiber.Ctx) error {
	var body struct {
		KeepID   uuid.UUID   `json:"keep_id"`
		MergeIDs []uuid.UUID `json:"merge_ids"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if body.KeepID == uuid.Nil || len(body.MergeIDs) == 0 || len(body.MergeIDs) > 20 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "keep_id and 1–20 merge_ids are required"})
	}
	seen := map[uuid.UUID]bool{body.KeepID: true}
	for _, id := range body.MergeIDs {
		if seen[id] {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "each contact may appear once"})
		}
		seen[id] = true
	}
	if err := h.profiles.Merge(c.Context(), body.KeepID, body.MergeIDs); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil && h.audit != nil {
		h.audit.Log(c.Context(), *actor, "merge", repo.AuditContact, body.KeepID, fiber.Map{"merged": body.MergeIDs})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
