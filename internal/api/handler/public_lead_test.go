package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSubmitLeadInputValidation(t *testing.T) {
	app := fiber.New()
	h := &PublicLeadHandler{}
	app.Post("/webhooks/leads", h.SubmitLead)

	tests := []struct {
		name       string
		req        PublicLeadRequest
		wantStatus int
		wantError  string
	}{
		{
			name: "Name too long",
			req: PublicLeadRequest{
				Name:  strings.Repeat("a", 201),
				Phone: "+971501234567",
			},
			wantStatus: fiber.StatusUnprocessableEntity,
			wantError:  "name is too long",
		},
		{
			name: "Email too long",
			req: PublicLeadRequest{
				Name:  "Valid Name",
				Phone: "+971501234567",
				Email: strings.Repeat("a", 250) + "@example.com",
			},
			wantStatus: fiber.StatusUnprocessableEntity,
			wantError:  "email is too long",
		},
		{
			name: "Notes too long",
			req: PublicLeadRequest{
				Name:  "Valid Name",
				Phone: "+971501234567",
				Notes: strings.Repeat("n", 2001),
			},
			wantStatus: fiber.StatusUnprocessableEntity,
			wantError:  "notes are too long",
		},
		{
			name: "PropertyType too long",
			req: PublicLeadRequest{
				Name:         "Valid Name",
				Phone:        "+971501234567",
				PropertyType: strings.Repeat("p", 101),
			},
			wantStatus: fiber.StatusUnprocessableEntity,
			wantError:  "property_type is too long",
		},
		{
			name: "Area too long",
			req: PublicLeadRequest{
				Name:  "Valid Name",
				Phone: "+971501234567",
				Area:  strings.Repeat("a", 101),
			},
			wantStatus: fiber.StatusUnprocessableEntity,
			wantError:  "area is too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.req)
			req := httptest.NewRequest("POST", "/webhooks/leads", bytes.NewReader(body))
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
