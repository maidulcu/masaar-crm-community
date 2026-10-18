package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// consentStore is the part of repo.PrivacyRepo that send and capture paths need.
type consentStore interface {
	RecordConsent(ctx context.Context, c *domain.Consent) error
	MayContact(ctx context.Context, contactID uuid.UUID, channel string) (bool, error)
	EmailOptedOut(ctx context.Context, email string) (bool, error)
}

// recordConsentChoices stores one consent row per channel the caller expressed a choice
// for (true = granted, false = withdrawn). Unknown channels are ignored.
func recordConsentChoices(ctx context.Context, store consentStore, contactID uuid.UUID, choices map[string]bool,
	source string, by, apiKey *uuid.UUID) {
	if store == nil {
		return
	}
	for ch, granted := range choices {
		if !domain.ValidConsentChannel(ch) {
			continue
		}
		status := domain.ConsentWithdrawn
		if granted {
			status = domain.ConsentGranted
		}
		if err := store.RecordConsent(ctx, &domain.Consent{
			ContactID: contactID, Channel: ch, Status: status, Source: source, CapturedBy: by, APIKeyID: apiKey,
		}); err != nil {
			log.Printf("consent: record %s for contact %s: %v", ch, contactID, err)
		}
	}
}

// optedOut answers 403 and returns true when the contact may not be messaged on channel.
func optedOut(c *fiber.Ctx, store consentStore, contactID uuid.UUID, channel string) (bool, error) {
	if store == nil {
		return false, nil
	}
	ok, err := store.MayContact(c.Context(), contactID, channel)
	if err != nil {
		return true, serverError(c, err)
	}
	if !ok {
		return true, c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "this contact has opted out of " + channel + " messages",
		})
	}
	return false, nil
}

// PrivacyHandler serves consent records and data-subject requests (PDPL).
type PrivacyHandler struct {
	privacy *repo.PrivacyRepo
	audit   auditLogger
	// removeMedia deletes a stored WhatsApp attachment by key (may be nil).
	removeMedia func(key string) error
}

func NewPrivacyHandler(privacy *repo.PrivacyRepo, audit auditLogger, removeMedia func(string) error) *PrivacyHandler {
	return &PrivacyHandler{privacy: privacy, audit: audit, removeMedia: removeMedia}
}

// ListConsents godoc
// @Summary      A contact's consent
// @Description  Current choice per channel and the full history, newest first.
// @Tags         Privacy
// @Produce      json
// @Param        id  path  string  true  "Contact UUID"
// @Success      200  {object}  object{current=map[string]domain.Consent,history=[]domain.Consent}
// @Security     BearerAuth
// @Router       /contacts/{id}/consents [get]
func (h *PrivacyHandler) ListConsents(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	current, err := h.privacy.CurrentConsent(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	history, err := h.privacy.ConsentHistory(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if history == nil {
		history = []domain.Consent{}
	}
	return c.JSON(fiber.Map{"current": current, "history": history})
}

// RecordConsent godoc
// @Summary      Record a consent decision
// @Tags         Privacy
// @Accept       json
// @Produce      json
// @Param        id    path  string                                         true  "Contact UUID"
// @Param        body  body  object{channel=string,status=string,note=string}  true  "channel: whatsapp|email|sms|phone; status: granted|withdrawn"
// @Success      201  {object}  domain.Consent
// @Security     BearerAuth
// @Router       /contacts/{id}/consents [post]
func (h *PrivacyHandler) RecordConsent(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Channel string `json:"channel"`
		Status  string `json:"status"`
		Note    string `json:"note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if !domain.ValidConsentChannel(body.Channel) || (body.Status != domain.ConsentGranted && body.Status != domain.ConsentWithdrawn) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "channel must be whatsapp|email|sms|phone and status granted|withdrawn"})
	}
	if len(body.Note) > 500 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "note is too long"})
	}
	consent := &domain.Consent{ContactID: id, Channel: body.Channel, Status: body.Status, Source: "manual",
		CapturedBy: localsUserPtr(c), Note: strings.TrimSpace(body.Note)}
	if err := h.privacy.RecordConsent(c.Context(), consent); err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(consent)
}

// Export godoc
// @Summary      Export everything held about a contact
// @Description  Admin only. Data-subject access request: the contact, consent, leads, deals, offers, viewings, WhatsApp conversations, communications, emails and invoices, as one JSON file. The export is audit-logged.
// @Tags         Privacy
// @Produce      json
// @Param        id  path  string  true  "Contact UUID"
// @Success      200  {object}  map[string]interface{}
// @Security     BearerAuth
// @Router       /contacts/{id}/export [get]
func (h *PrivacyHandler) Export(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	data, err := h.privacy.Export(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil && h.audit != nil {
		h.audit.Log(c.Context(), *actor, "export", repo.AuditContact, id, fiber.Map{"reason": "data-subject access request"})
	}
	out, err := json.MarshalIndent(fiber.Map{
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"contact_id":  id,
		"data":        data,
	}, "", "  ")
	if err != nil {
		return serverError(c, err)
	}
	c.Set("Content-Type", "application/json; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="contact-%s.json"`, id))
	c.Set("Cache-Control", "no-store")
	return c.Send(out)
}

// Erase godoc
// @Summary      Erase a contact's personal data
// @Description  Admin only, irreversible. Anonymises the contact and removes personal data from its conversations, notes, messages and attachments; consent is recorded as withdrawn on every channel. Issued invoices keep the customer details printed on them, as tax records must be retained. Requires {"confirm": "ERASE"} and a reason; the erasure is audit-logged.
// @Tags         Privacy
// @Accept       json
// @Param        id    path  string                               true  "Contact UUID"
// @Param        body  body  object{confirm=string,reason=string}  true  "Confirmation"
// @Success      204
// @Security     BearerAuth
// @Router       /contacts/{id}/erase [post]
func (h *PrivacyHandler) Erase(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Confirm string `json:"confirm"`
		Reason  string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if body.Confirm != "ERASE" || body.Reason == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": `send {"confirm": "ERASE", "reason": "…"} to erase this contact`})
	}
	actor := localsUserPtr(c)
	if actor == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	paths, err := h.privacy.Erase(c.Context(), id, *actor)
	if errors.Is(err, repo.ErrContactErased) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "contact already erased"})
	}
	if err != nil {
		return serverError(c, err)
	}
	if h.removeMedia != nil {
		for _, p := range paths {
			if err := h.removeMedia(p); err != nil {
				log.Printf("erase: remove media for contact %s: %v", id, err)
			}
		}
	}
	if h.audit != nil {
		h.audit.Log(c.Context(), *actor, "erase", repo.AuditContact, id, fiber.Map{"reason": body.Reason, "files_removed": len(paths)})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
