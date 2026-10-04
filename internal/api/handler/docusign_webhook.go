package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

	// DocuSign Connect sends HMAC-SHA256 signatures base64-encoded in X-DocuSign-Signature-1.
	// Webhook secret must be configured to process incoming webhooks securely.
	if h.webhookSecret == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	sig := c.Get("X-DocuSign-Signature-1")
	if sig == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
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

	if h.docs != nil && (event.Data.Status == "completed" || event.Data.Status == "signed") {
		_ = h.docs.MarkSignedByEnvelope(c.Context(), envelopeID)
	}

	return c.JSON(fiber.Map{"ok": true})
}
