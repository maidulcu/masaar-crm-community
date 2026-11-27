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

	invalidJSON := []byte(`{invalid-json`)
	mac2 := hmac.New(sha256.New, []byte(secret))
	mac2.Write(invalidJSON)
	invalidJSONSig := base64.StdEncoding.EncodeToString(mac2.Sum(nil))

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
		{
			name:           "malformed json payload",
			secret:         secret,
			signature:      invalidJSONSig,
			body:           invalidJSON,
			expectedStatus: http.StatusBadRequest,
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

// While Connect keys are rotated, DocuSign sends one signature per key; ours may not be the first.
func TestDocusignWebhookHandler_AcceptsAnySignatureHeader(t *testing.T) {
	secret := "test-secret-12345"
	payload := []byte(`{"event":"envelope-sent","data":{"envelopeId":"env-1","status":"sent"}}`)
	sign := func(key string) string {
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write(payload)
		return base64.StdEncoding.EncodeToString(mac.Sum(nil))
	}
	h := NewDocusignWebhookHandler(nil, secret)
	app := fiber.New()
	app.Post("/webhooks/docusign", h.Handle)
	for name, tc := range map[string]struct {
		headers map[string]string
		want    int
	}{
		"ours second":     {map[string]string{"X-DocuSign-Signature-1": sign("other-key"), "X-DocuSign-Signature-2": sign(secret)}, http.StatusOK},
		"only foreign":    {map[string]string{"X-DocuSign-Signature-1": sign("other-key"), "X-DocuSign-Signature-2": sign("third-key")}, http.StatusUnauthorized},
		"gap before ours": {map[string]string{"X-DocuSign-Signature-3": sign(secret)}, http.StatusOK},
	} {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/docusign", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range tc.headers {
			req.Header.Set(k, v)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != tc.want {
			t.Errorf("%s: status %d, want %d", name, resp.StatusCode, tc.want)
		}
	}
}
