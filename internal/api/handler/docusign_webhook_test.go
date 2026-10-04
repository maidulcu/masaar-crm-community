package handler

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestDocusignWebhookHandler_HMAC(t *testing.T) {
	secret := "test-secret-12345"
	payload := []byte(`{"event":"envelope-completed","data":{"envelopeId":"env-123","status":"completed"}}`)

	// Generate valid Base64 HMAC signature as sent by DocuSign Connect
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name           string
		secret         string
		signature      string
		body           []byte
		expectedStatus int
	}{
		{
			name:           "valid base64 hmac signature",
			secret:         secret,
			signature:      validSig,
			body:           payload,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid signature",
			secret:         secret,
			signature:      "invalid-base64-signature",
			body:           payload,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing signature header",
			secret:         secret,
			signature:      "",
			body:           payload,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "unconfigured webhook secret",
			secret:         "",
			signature:      validSig,
			body:           payload,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDocusignWebhookHandler(nil, tt.secret)
			app := fiber.New()
			app.Post("/webhooks/docusign", h.Handle)

			req := httptest.NewRequest(http.MethodPost, "/webhooks/docusign", bytes.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.signature != "" {
				req.Header.Set("X-DocuSign-Signature-1", tt.signature)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, tt.expectedStatus)
			}
		})
	}
}
