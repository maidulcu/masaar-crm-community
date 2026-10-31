package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

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
// maxDocusignSignatures is how many X-DocuSign-Signature-N headers are checked (DocuSign Connect
// allows up to 100 keys; accounts use one or two).
const maxDocusignSignatures = 10

func (h *DocusignWebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()

	// DocuSign Connect sends HMAC-SHA256 signatures base64-encoded in X-DocuSign-Signature-1.
	// Webhook secret must be configured to process incoming webhooks securely.
	if h.webhookSecret == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expected := mac.Sum(nil)
	// With several Connect keys (e.g. while rotating), DocuSign sends one signature per key in
	// X-DocuSign-Signature-1, -2, …: any of them made with our secret is enough.
	signed := false
	for i := 1; i <= maxDocusignSignatures; i++ {
		sigBytes, err := base64.StdEncoding.DecodeString(c.Get(fmt.Sprintf("X-DocuSign-Signature-%d", i)))
		// Security: Compare raw HMAC digest bytes in constant time to prevent timing side-channel attacks.
		if err == nil && len(sigBytes) > 0 && hmac.Equal(sigBytes, expected) {
			signed = true
			break
		}
	}
	if !signed {
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
	if err := json.Unmarshal(body, &event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json"})
	}

	envelopeID := event.Data.EnvelopeID
	if envelopeID == "" {
		return c.JSON(fiber.Map{"ok": true})
	}

	if h.docs != nil && (event.Data.Status == "completed" || event.Data.Status == "signed") {
		_ = h.docs.MarkSignedByEnvelope(c.Context(), envelopeID)
	}

	return c.JSON(fiber.Map{"ok": true})
}
