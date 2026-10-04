package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/ai"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

type WhatsAppHandler struct {
	wa             *repo.WhatsAppRepo
	contacts       *repo.ContactRepo
	outbound       *repo.WhatsAppOutboundRepo
	comms          *repo.CommunicationHistoryRepo
	media          *WAMediaService
	taggingService *ai.TaggingService
	hub            *ws.Hub
	config         *config.Config
}

func NewWhatsAppHandler(wa *repo.WhatsAppRepo, contacts *repo.ContactRepo, outbound *repo.WhatsAppOutboundRepo, comms *repo.CommunicationHistoryRepo, media *WAMediaService, taggingService *ai.TaggingService, hub *ws.Hub, cfg *config.Config) *WhatsAppHandler {
	return &WhatsAppHandler{wa: wa, contacts: contacts, outbound: outbound, comms: comms, media: media, taggingService: taggingService, hub: hub, config: cfg}
}

// GET /webhooks/whatsapp — Meta webhook verification
func (h *WhatsAppHandler) Verify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && subtle.ConstantTimeCompare([]byte(token), []byte(h.config.WAVerifyToken)) == 1 {
		return c.SendString(challenge)
	}
	return c.SendStatus(fiber.StatusForbidden)
}

// ListThreads godoc
// @Summary      List WhatsApp threads
// @Description  Returns paginated WhatsApp conversation threads with contact details.
// @Tags         WhatsApp
// @Produce      json
// @Param        status      query     string  false  "Filter by status: open|pending|closed"
// @Param        contact_id  query     string  false  "Filter by contact UUID"
// @Param        page        query     int     false  "Page number (default 1)"
// @Param        limit       query     int     false  "Page size (default 20)"
// @Success      200     {array}   domain.WhatsAppThread
// @Security     BearerAuth
// @Router       /threads [get]
func (h *WhatsAppHandler) ListThreads(c *fiber.Ctx) error {
	status := c.Query("status", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var contactID *uuid.UUID
	if s := c.Query("contact_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			contactID = &id
		}
	}

	threads, err := h.wa.ListThreads(c.Context(), status, contactID, page, limit)
	if err != nil {
		log.Printf("ListThreads error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load threads"})
	}
	return c.JSON(threads)
}

// GetMessages godoc
// @Summary      Get thread messages
// @Description  Returns messages for a WhatsApp thread, ordered by sent_at ascending.
// @Tags         WhatsApp
// @Produce      json
// @Param        id     path      string  true   "Thread UUID"
// @Param        limit  query     int     false  "Max messages to return (default 100)"
// @Success      200    {array}   domain.WhatsAppMessage
// @Failure      400    {object}  object{error=string}
// @Security     BearerAuth
// @Router       /threads/{id}/messages [get]
func (h *WhatsAppHandler) GetMessages(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	_, limit := pageParams(c, 100, 500)

	msgs, err := h.wa.GetMessages(c.Context(), id, limit)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(msgs)
}

// ReopenThread manually reopens a closed thread.
// POST /api/v1/threads/:id/reopen
func (h *WhatsAppHandler) ReopenThread(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.wa.ReopenThread(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// CloseThread godoc
// @Summary      Close thread
// @Description  Sets thread_status to closed.
// @Tags         WhatsApp
// @Param        id  path  string  true  "Thread UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /threads/{id}/close [post]
func (h *WhatsAppHandler) CloseThread(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.wa.CloseThread(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// GetThread godoc
// @Summary      Get thread
// @Description  Returns a single WhatsApp thread with contact details.
// @Tags         WhatsApp
// @Produce      json
// @Param        id  path      string  true  "Thread UUID"
// @Success      200  {object}  domain.WhatsAppThread
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /threads/{id} [get]
func (h *WhatsAppHandler) GetThread(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	thread, err := h.wa.GetThread(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
	}
	return c.JSON(thread)
}

// POST /webhooks/whatsapp — receive inbound messages
func (h *WhatsAppHandler) Receive(c *fiber.Ctx) error {
	// Validate Meta's HMAC-SHA256 signature. Without WA_APP_SECRET anyone who finds
	// this URL could inject messages, so production refuses to process unsigned
	// payloads; development may skip verification for local testing.
	if h.config.WAAppSecret == "" && h.config.IsProduction() {
		log.Printf("whatsapp: rejecting webhook — WA_APP_SECRET is not set")
		return c.SendStatus(fiber.StatusServiceUnavailable)
	}
	if h.config.WAAppSecret != "" {
		sig := c.Get("X-Hub-Signature-256")
		if !strings.HasPrefix(sig, "sha256=") {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		mac := hmac.New(sha256.New, []byte(h.config.WAAppSecret))
		mac.Write(c.Body())
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(sig), []byte(expected)) {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
	}
	// Inbound webhooks carry no user session; they belong to this deployment's company.
	companyID, err := uuid.Parse(h.config.AppCompanyID)
	if err != nil {
		log.Printf("whatsapp: invalid APP_COMPANY_ID: %v", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	ctx := tenant.With(c.Context(), companyID)

	var payload waPayload
	if err := c.BodyParser(&payload); err != nil {
		log.Printf("whatsapp webhook: body parse error: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("invalid body")
	}

	// Meta redelivers a webhook until it gets a 2xx, so processing is idempotent and one bad
	// item never stops the rest of the batch. Only a genuine failure (e.g. database down) makes
	// the whole request answer 5xx so Meta retries it; items already stored are then skipped.
	var failed error
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			for _, st := range change.Value.Statuses {
				if err := h.handleStatus(ctx, st); err != nil {
					log.Printf("whatsapp: status %s for %s: %v", st.Status, st.ID, err)
					failed = err
				}
			}
			for _, msg := range change.Value.Messages {
				if err := h.handleInbound(ctx, companyID, change.Value, msg); err != nil {
					log.Printf("whatsapp: message %s: %v", msg.ID, err)
					failed = err
				}
			}
		}
	}
	if failed != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	return c.SendStatus(fiber.StatusOK)
}

// handleInbound stores one customer message. It returns nil for messages that are intentionally
// not stored (duplicates, reactions, unusable sender numbers).
func (h *WhatsAppHandler) handleInbound(ctx context.Context, companyID uuid.UUID, val waChangeValue, msg waMessage) error {
	content := parseInbound(msg)
	if content.Skip || msg.ID == "" {
		return nil
	}

	// Cheap duplicate check first: a redelivery must not touch the contact or reopen a thread
	// that an agent has closed since.
	if exists, err := h.wa.MessageExists(ctx, msg.ID); err != nil {
		return err
	} else if exists {
		return nil
	}

	profileName := ""
	for _, wc := range val.Contacts {
		if wc.WAID == msg.From {
			profileName = wc.Profile.Name
			break
		}
	}

	contact, err := h.contacts.Upsert(ctx, msg.From, profileName)
	if errors.Is(err, repo.ErrInvalidPhone) {
		log.Printf("whatsapp: ignoring message %s from unusable number %q", msg.ID, msg.From)
		return nil
	}
	if err != nil {
		return err
	}

	thread, err := h.wa.UpsertThread(ctx, contact.ID, val.Metadata.PhoneNumberID)
	if err != nil {
		return err
	}

	waMsg := &domain.WhatsAppMessage{
		ThreadID:    thread.ID,
		Direction:   domain.DirectionInbound,
		Body:        content.Body,
		WAMessageID: msg.ID,
		MediaID:     content.MediaID,
		MediaMime:   content.MediaMime,
	}
	created, err := h.wa.SaveMessage(ctx, waMsg)
	if err != nil {
		return err
	}
	if !created { // lost a race with a concurrent redelivery
		return nil
	}

	if err := h.wa.UpdateThreadMeta(ctx, thread.ID); err != nil {
		log.Printf("whatsapp: thread meta for %s: %v", thread.ID, err)
	}
	if content.MediaID != "" {
		h.media.Prefetch(companyID, thread.ID, waMsg.ID)
	}
	if _, err := h.comms.LogWhatsApp(ctx, contact.ID, &domain.CommunicationHistory{
		CommunicationType: domain.CommWhatsAppInbound,
		Direction:         "inbound",
		Body:              content.Body,
		FromIdentifier:    contact.PhoneWA,
		ExternalID:        msg.ID,
		Status:            "received",
	}); err != nil {
		log.Printf("whatsapp: timeline entry for %s: %v", msg.ID, err)
	}

	// Auto-tag leads based on message content. Runs detached from the request, so it gets its
	// own context that still carries the company.
	if h.taggingService != nil && msg.Type == "text" {
		body := msg.Text.Body
		go func() {
			if err := h.taggingService.AutoTagFromMessage(tenant.With(context.Background(), companyID), contact.ID, body); err != nil {
				log.Printf("whatsapp: auto-tagging error: %v", err)
			}
		}()
	}

	h.hub.BroadcastToCompany(h.config.AppCompanyID, ws.Event{
		Type: "whatsapp.message",
		Payload: fiber.Map{
			"thread_id": thread.ID,
			"contact":   contact,
			"message":   waMsg,
		},
	})
	return nil
}

// handleStatus applies a delivery receipt (sent / delivered / read / failed) to the outbound
// message it refers to.
func (h *WhatsAppHandler) handleStatus(ctx context.Context, st waStatus) error {
	status, ok := statusFromMeta(st.Status)
	if !ok || st.ID == "" {
		return nil
	}
	errMsg := ""
	if status == "failed" {
		errMsg = describeStatusErrors(st)
	}
	updated, err := h.outbound.ApplyStatus(ctx, st.ID, domain.OutboundStatus(status), errMsg)
	if err != nil {
		return err
	}
	if updated {
		if err := h.comms.UpdateStatusByExternalID(ctx, st.ID, status); err != nil {
			log.Printf("whatsapp: timeline status for %s: %v", st.ID, err)
		}
	}
	return nil
}
