package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/ai"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

type WhatsAppHandler struct {
	wa             *repo.WhatsAppRepo
	contacts       *repo.ContactRepo
	taggingService *ai.TaggingService
	hub            *ws.Hub
	config         *config.Config
}

func NewWhatsAppHandler(wa *repo.WhatsAppRepo, contacts *repo.ContactRepo, taggingService *ai.TaggingService, hub *ws.Hub, cfg *config.Config) *WhatsAppHandler {
	return &WhatsAppHandler{wa: wa, contacts: contacts, taggingService: taggingService, hub: hub, config: cfg}
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
	limit, _ := strconv.Atoi(c.Query("limit", "100"))

	msgs, err := h.wa.GetMessages(c.Context(), id, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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
	// Validate Meta's HMAC-SHA256 signature when WA_APP_SECRET is configured.
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
	var payload struct {
		Object string `json:"object"`
		Entry  []struct {
			ID      string `json:"id"`
			Changes []struct {
				Value struct {
					MessagingProduct string `json:"messaging_product"`
					Metadata         struct {
						PhoneNumberID string `json:"phone_number_id"`
					} `json:"metadata"`
					Contacts []struct {
						Profile struct {
							Name string `json:"name"`
						} `json:"profile"`
						WAID string `json:"wa_id"`
					} `json:"contacts"`
					Messages []struct {
						From      string `json:"from"`
						ID        string `json:"id"`
						Timestamp string `json:"timestamp"`
						Type      string `json:"type"`
						Text      struct {
							Body string `json:"body"`
						} `json:"text"`
						Image struct {
							URL string `json:"url"`
						} `json:"image"`
						Video struct {
							URL string `json:"url"`
						} `json:"video"`
						Audio struct {
							URL string `json:"url"`
						} `json:"audio"`
						Document struct {
							Filename string `json:"filename"`
							URL      string `json:"url"`
						} `json:"document"`
					} `json:"messages"`
				} `json:"value"`
				Field string `json:"field"`
			} `json:"changes"`
		} `json:"entry"`
	}

	if err := c.BodyParser(&payload); err != nil {
		log.Printf("whatsapp webhook: body parse error: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString("invalid body")
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			val := change.Value

			for _, msg := range val.Messages {
				msgBody := ""
				mediaURL := ""

				switch msg.Type {
				case "text":
					msgBody = msg.Text.Body
				case "image":
					if msg.Image.URL != "" {
						msgBody = "[Image]"
						mediaURL = msg.Image.URL
					}
				case "video":
					if msg.Video.URL != "" {
						msgBody = "[Video]"
						mediaURL = msg.Video.URL
					}
				case "audio":
					if msg.Audio.URL != "" {
						msgBody = "[Audio]"
						mediaURL = msg.Audio.URL
					}
				case "document":
					if msg.Document.URL != "" {
						msgBody = "[Document: " + msg.Document.Filename + "]"
						mediaURL = msg.Document.URL
					}
				default:
					msgBody = "[" + msg.Type + "]"
				}

				if msgBody == "" {
					log.Printf("whatsapp: skipping message %s — empty body (unsupported type %s)", msg.ID, msg.Type)
					continue
				}

				senderName := msg.From
				for _, wc := range val.Contacts {
					if wc.WAID == msg.From {
						senderName = wc.Profile.Name
						break
					}
				}

				contact, err := h.contacts.Upsert(c.Context(), msg.From, senderName)
				if err != nil {
					log.Printf("whatsapp: upsert contact error for msg %s: %v", msg.ID, err)
					return c.Status(fiber.StatusInternalServerError).SendString("contact upsert failed")
				}

				thread, err := h.wa.UpsertThread(c.Context(), contact.ID, val.Metadata.PhoneNumberID)
				if err != nil {
					log.Printf("whatsapp: upsert thread error for msg %s: %v", msg.ID, err)
					return c.Status(fiber.StatusInternalServerError).SendString("thread upsert failed")
				}

				waMsg := &domain.WhatsAppMessage{
					ThreadID:    thread.ID,
					Direction:   domain.DirectionInbound,
					Body:        msgBody,
					MediaURL:    mediaURL,
					WAMessageID: msg.ID,
				}
				if err := h.wa.SaveMessage(c.Context(), waMsg); err != nil {
					log.Printf("whatsapp: save message error for msg %s: %v", msg.ID, err)
					return c.Status(fiber.StatusInternalServerError).SendString("message save failed")
				}

				_ = h.wa.UpdateThreadMeta(c.Context(), thread.ID)

				// Auto-tag leads based on message content
				if h.taggingService != nil && msg.Type == "text" {
					go func() {
						if err := h.taggingService.AutoTagFromMessage(context.Background(), contact.ID, msg.Text.Body); err != nil {
							log.Printf("whatsapp: auto-tagging error: %v", err)
						}
					}()
				}

				h.hub.Broadcast(ws.Event{
					Type: "whatsapp.message",
					Payload: fiber.Map{
						"thread_id": thread.ID,
						"contact":   contact,
						"message":   waMsg,
					},
				})
			}
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
