package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/webhook"
)

type WebhookSubHandler struct {
	webhooks   *repo.WebhookRepo
	dispatcher *webhook.Dispatcher
}

func NewWebhookSubHandler(webhooks *repo.WebhookRepo, dispatcher *webhook.Dispatcher) *WebhookSubHandler {
	return &WebhookSubHandler{webhooks: webhooks, dispatcher: dispatcher}
}

type WebhookSubCreateRequest struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Events string `json:"events"` // comma-separated: "lead.created,lead.stage_changed"
}

type WebhookSubResponse struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	URL          string     `json:"url"`
	Events       []string   `json:"events"`
	Active       bool       `json:"active"`
	Secret       string     `json:"secret,omitempty"` // only on creation
	FailureCount int        `json:"failure_count"`
	CreatedAt    string     `json:"created_at"`
	LastFiredAt  *string    `json:"last_fired_at"`
}

// ListWebhooks lists webhook subscriptions for the company
// @Summary List webhook subscriptions
// @Tags Settings
// @Produce json
// @Success 200 {array} WebhookSubResponse
// @Security Bearer
// @Router /api/v1/settings/webhooks [get]
func (h *WebhookSubHandler) List(c *fiber.Ctx) error {
	companyID := c.Locals("company_id").(string)
	compID, err := uuid.Parse(companyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	subs, err := h.webhooks.List(c.Context(), compID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var out []WebhookSubResponse
	for _, s := range subs {
		r := WebhookSubResponse{
			ID:           s.ID,
			Name:         s.Name,
			URL:          s.URL,
			Events:       s.Events,
			Active:       s.Active,
			FailureCount: s.FailureCount,
			CreatedAt:    s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if s.LastFiredAt != nil {
			t := s.LastFiredAt.Format("2006-01-02T15:04:05Z")
			r.LastFiredAt = &t
		}
		out = append(out, r)
	}
	return c.JSON(out)
}

// CreateWebhook registers a new webhook subscription
// @Summary Register webhook
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body WebhookSubCreateRequest true "Webhook config"
// @Success 201 {object} WebhookSubResponse
// @Security Bearer
// @Router /api/v1/settings/webhooks [post]
func (h *WebhookSubHandler) Create(c *fiber.Ctx) error {
	if c.Locals("role").(domain.Role) != domain.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can register webhooks"})
	}

	companyID := c.Locals("company_id").(string)
	compID, err := uuid.Parse(companyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	var req WebhookSubCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	if req.URL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "url is required"})
	}
	if !strings.HasPrefix(req.URL, "https://") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "webhook url must use https"})
	}

	secret, err := webhook.GenerateSecret()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate secret"})
	}

	sub, err := h.webhooks.Create(c.Context(), compID, req.Name, req.URL, req.Events, secret)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(WebhookSubResponse{
		ID:        sub.ID,
		Name:      sub.Name,
		URL:       sub.URL,
		Events:    sub.Events,
		Active:    sub.Active,
		Secret:    secret, // shown once — verify with X-Masaar-Signature header
		CreatedAt: sub.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// DeleteWebhook removes a webhook subscription
// @Summary Delete webhook
// @Tags Settings
// @Param id path string true "Subscription ID"
// @Success 204
// @Security Bearer
// @Router /api/v1/settings/webhooks/{id} [delete]
func (h *WebhookSubHandler) Delete(c *fiber.Ctx) error {
	if c.Locals("role").(domain.Role) != domain.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can delete webhooks"})
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.webhooks.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// TestWebhook sends a test ping to verify the endpoint is reachable
// @Summary Send test webhook
// @Tags Settings
// @Param id path string true "Subscription ID"
// @Success 200 {object} map[string]string
// @Security Bearer
// @Router /api/v1/settings/webhooks/{id}/test [post]
func (h *WebhookSubHandler) Test(c *fiber.Ctx) error {
	if c.Locals("role").(domain.Role) != domain.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can test webhooks"})
	}

	companyID := c.Locals("company_id").(string)
	compID, err := uuid.Parse(companyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	h.dispatcher.Dispatch(compID, "webhook.test", fiber.Map{
		"message": "This is a test event from Masaar CRM",
	})

	return c.JSON(fiber.Map{"message": "test event dispatched"})
}
