package handler

import (
	"github.com/google/uuid"
	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type ApiKeyHandler struct {
	apiKeyRepo *repo.ApiKeyRepo
}

func NewApiKeyHandler(apiKeyRepo *repo.ApiKeyRepo) *ApiKeyHandler {
	return &ApiKeyHandler{apiKeyRepo: apiKeyRepo}
}

type ApiKeyCreateRequest struct {
	Name   string `json:"name"`
	Scopes string `json:"scopes"` // Comma-separated: "lead:create,lead:read"
}

type ApiKeyResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	KeyPrefix string    `json:"key_prefix"` // Safe to display
	Scopes    string    `json:"scopes"`
	CreatedAt string    `json:"created_at"`
	LastUsedAt *string  `json:"last_used_at"`
	Plaintext string    `json:"plaintext,omitempty"` // Only on creation, never again
}

// ListApiKeys lists all API keys for the authenticated user's company
// @Summary List API keys
// @Description Get all active API keys for your company
// @Tags Settings
// @Produce json
// @Success 200 {array} ApiKeyResponse
// @Failure 401 {object} map[string]string
// @Security Bearer
// @Router /api/v1/settings/api-keys [get]
func (h *ApiKeyHandler) ListApiKeys(c *fiber.Ctx) error {
	companyID := c.Locals("company_id").(string)

	compID, err := uuid.Parse(companyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	keys, err := h.apiKeyRepo.List(c.Context(), compID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var responses []ApiKeyResponse
	for _, key := range keys {
		resp := ApiKeyResponse{
			ID:        key.ID,
			Name:      key.Name,
			KeyPrefix: key.KeyPrefix,
			Scopes:    key.Scopes,
			CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if key.LastUsedAt != nil {
			lastUsed := key.LastUsedAt.Format("2006-01-02T15:04:05Z")
			resp.LastUsedAt = &lastUsed
		}
		responses = append(responses, resp)
	}

	return c.JSON(responses)
}

// CreateApiKey generates a new API key
// @Summary Create API key
// @Description Generate a new API key for external integrations
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body ApiKeyCreateRequest true "Key details"
// @Success 201 {object} ApiKeyResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Security Bearer
// @Router /api/v1/settings/api-keys [post]
func (h *ApiKeyHandler) CreateApiKey(c *fiber.Ctx) error {
	// Only Admin can create keys
	role := c.Locals("role").(domain.Role)
	if role != domain.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can create api keys"})
	}

	companyID := c.Locals("company_id").(string)
	compID, err := uuid.Parse(companyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	var req ApiKeyCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	// Generate the key
	plaintext, keyHash, keyPrefix, err := h.apiKeyRepo.GenerateKey()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Store in database
	id, err := h.apiKeyRepo.Create(c.Context(), compID, req.Name, keyHash, keyPrefix, req.Scopes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Return with plaintext (only shown once)
	return c.Status(fiber.StatusCreated).JSON(ApiKeyResponse{
		ID:        id,
		Name:      req.Name,
		KeyPrefix: keyPrefix,
		Scopes:    req.Scopes,
		Plaintext: plaintext, // Only returned at creation time
	})
}

// RevokeApiKey revokes an API key
// @Summary Revoke API key
// @Description Disable an API key immediately
// @Tags Settings
// @Param id path string true "API Key ID"
// @Success 204
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security Bearer
// @Router /api/v1/settings/api-keys/{id} [delete]
func (h *ApiKeyHandler) RevokeApiKey(c *fiber.Ctx) error {
	// Only Admin can revoke
	role := c.Locals("role").(domain.Role)
	if role != domain.RoleAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "only admins can revoke api keys"})
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.apiKeyRepo.Revoke(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
