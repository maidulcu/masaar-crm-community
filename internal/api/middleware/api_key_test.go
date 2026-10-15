package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestAPIKeyRateLimit_NilRedis(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("api_key_id", uuid.New())
		return c.Next()
	})
	app.Use(APIKeyRateLimit(nil, 10, time.Minute))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK when rdb is nil, got %d", resp.StatusCode)
	}
}

func TestAPIKeyRateLimit_NoKeyID(t *testing.T) {
	app := fiber.New()
	app.Use(APIKeyRateLimit(nil, 10, time.Minute))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK when keyID is missing, got %d", resp.StatusCode)
	}
}

func TestRequireAPIKeyScope(t *testing.T) {
	tests := []struct {
		name           string
		scopesLocal    interface{}
		requiredScope  string
		expectedStatus int
	}{
		{
			name:           "missing scopes local",
			scopesLocal:    nil,
			requiredScope:  "lead:create",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "valid single scope",
			scopesLocal:    "lead:create",
			requiredScope:  "lead:create",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid multiple scopes",
			scopesLocal:    "lead:read, lead:create, lead:delete",
			requiredScope:  "lead:create",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "insufficient scope",
			scopesLocal:    "lead:read",
			requiredScope:  "lead:create",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Use(func(c *fiber.Ctx) error {
				if tt.scopesLocal != nil {
					c.Locals("api_key_scopes", tt.scopesLocal)
				}
				return c.Next()
			})
			app.Use(RequireAPIKeyScope(tt.requiredScope))
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("got status %d, want %d", resp.StatusCode, tt.expectedStatus)
			}
		})
	}
}
