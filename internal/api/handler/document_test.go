package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestSendForSignatureInputValidation(t *testing.T) {
	app := fiber.New()
	h := &DocumentHandler{}
	app.Post("/documents/:id/request-signature", h.SendForSignature)

	docID := uuid.New().String()

	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantError  string
	}{
		{
			name:       "Missing signer_name",
			body:       map[string]any{"signer_email": "user@example.com"},
			wantStatus: fiber.StatusBadRequest,
			wantError:  "signer_name and signer_email required",
		},
		{
			name:       "Whitespace-only signer_name",
			body:       map[string]any{"signer_name": "   ", "signer_email": "user@example.com"},
			wantStatus: fiber.StatusBadRequest,
			wantError:  "signer_name and signer_email required",
		},
		{
			name:       "SignerName too long",
			body:       map[string]any{"signer_name": strings.Repeat("a", 201), "signer_email": "user@example.com"},
			wantStatus: fiber.StatusUnprocessableEntity,
			wantError:  "signer_name is too long",
		},
		{
			name:       "SignerEmail too long",
			body:       map[string]any{"signer_name": "Valid Name", "signer_email": strings.Repeat("e", 250) + "@example.com"},
			wantStatus: fiber.StatusUnprocessableEntity,
			wantError:  "signer_email is too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/documents/"+docID+"/request-signature", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, tt.wantStatus)
			}

			var res map[string]string
			_ = json.NewDecoder(resp.Body).Decode(&res)
			if !strings.Contains(res["error"], tt.wantError) {
				t.Errorf("error = %q, want containing %q", res["error"], tt.wantError)
			}
		})
	}
}
