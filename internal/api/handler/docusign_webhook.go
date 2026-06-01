package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// DocusignWebhookHandler receives DocuSign Connect events.
type DocusignWebhookHandler struct {
	docs          *repo.DocumentRepo
	webhookSecret string
}

func NewDocusignWebhookHandler(docs *repo.DocumentRepo, webhookSecret string) *DocusignWebhookHandler {
	return &DocusignWebhookHandler{docs: docs, webhookSecret: webhookSecret}
}

// Handle processes incoming DocuSign Connect webhook notifications.
// POST /webhooks/docusign
func (h *DocusignWebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()

	// Verify HMAC if a secret is configured
	if h.webhookSecret != "" {
		sig := c.Get("X-DocuSign-Signature-1")
		if sig == "" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		mac := hmac.New(sha256.New, []byte(h.webhookSecret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(sig), []byte(expected)) {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
	}

	var event struct {
		Event       string `json:"event"`
		Data        struct {
			EnvelopeID string `json:"envelopeId"`
			Status     string `json:"status"`
			Recipients []struct {
				Status string `json:"status"`
			} `json:"recipients"`
		} `json:"data"`
	}
	json.Unmarshal(body, &event)

	envelopeID := event.Data.EnvelopeID
	if envelopeID == "" {
		return c.JSON(fiber.Map{"ok": true})
	}

	if event.Data.Status == "completed" || event.Data.Status == "signed" {
		_ = h.docs.MarkSignedByEnvelope(c.Context(), envelopeID)
	}

	return c.JSON(fiber.Map{"ok": true})
}
