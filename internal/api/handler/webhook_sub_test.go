package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/api/handler"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestWebhookSub_Test_NilRole(t *testing.T) {
	h := handler.NewWebhookSubHandler(nil, nil)

	app := fiber.New()
	app.Post("/api/v1/settings/webhooks/:id/test", func(c *fiber.Ctx) error {
		// Simulating request where role is not set in locals (e.g., API key request or missing role)
		return h.Test(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/webhooks/123/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, resp.StatusCode)
	}
}

func TestWebhookSub_Test_AgentRole(t *testing.T) {
	h := handler.NewWebhookSubHandler(nil, nil)

	app := fiber.New()
	app.Post("/api/v1/settings/webhooks/:id/test", func(c *fiber.Ctx) error {
		c.Locals("role", domain.RoleAgent)
		return h.Test(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings/webhooks/123/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, resp.StatusCode)
	}
}
