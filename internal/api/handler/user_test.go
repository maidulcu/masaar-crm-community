package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestUserEmailValidation(t *testing.T) {
	app := fiber.New()
	h := &UserHandler{}

	app.Post("/users", func(c *fiber.Ctx) error {
		// Mock claims local for ClaimsFromCtx
		c.Locals("user", jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":        uuid.New().String(),
			"company_id": uuid.New().String(),
			"role":       "admin",
		}))
		return h.CreateUser(c)
	})

	app.Post("/users/invite", func(c *fiber.Ctx) error {
		c.Locals("user", jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":        uuid.New().String(),
			"company_id": uuid.New().String(),
			"role":       "admin",
		}))
		return h.InviteUser(c)
	})

	invalidEmails := []string{
		"plainaddress",
		"@missingusername.com",
		"username@.com",
		"username@domain..com",
		"invalid email@domain.com",
		strings.Repeat("a", 250) + "@domain.com", // > 254 chars
	}

	endpoints := []string{"/users", "/users/invite"}

	for _, ep := range endpoints {
		for _, email := range invalidEmails {
			var body map[string]interface{}
			if ep == "/users" {
				body = map[string]interface{}{
					"name":     "Test User",
					"email":    email,
					"password": "Password1",
					"role":     "agent",
				}
			} else {
				body = map[string]interface{}{
					"name":  "Test User",
					"email": email,
					"role":  "agent",
				}
			}

			jsonBody, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, ep, bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed on %s for %s: %v", ep, email, err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("endpoint %s with invalid email %q gave status %d, want %d", ep, email, resp.StatusCode, http.StatusBadRequest)
			}
		}
	}
}
