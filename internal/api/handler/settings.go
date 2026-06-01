package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type SettingsHandler struct {
	apiSettingsRepo    *repo.SettingsRepo
	companySettingsRepo *repo.CompanySettingsRepo
}

func NewSettingsHandler(apiSettingsRepo *repo.SettingsRepo, companySettingsRepo *repo.CompanySettingsRepo) *SettingsHandler {
	return &SettingsHandler{
		apiSettingsRepo: apiSettingsRepo,
		companySettingsRepo: companySettingsRepo,
	}
}

// GetBOS24Settings retrieves the BOS24 API token configuration
// @Summary Get BuyOrSell24 API settings
// @Description Retrieve BuyOrSell24 API token (admin only). Used to check if integration is configured.
// @Tags Settings
// @Produce json
// @Success 200 {object} domain.APISetting
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/settings/bos24 [get]
// @Security Bearer
func (h *SettingsHandler) GetBOS24Settings(c *fiber.Ctx) error {
	setting, err := h.apiSettingsRepo.Get(c.Context(), "bos24_api_token")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve settings",
		})
	}

	return c.JSON(setting)
}

// UpdateBOS24Settings updates the BOS24 API token
// @Summary Update BuyOrSell24 API token
// @Description Update BuyOrSell24 API token (admin only). Token is required to enable real estate features.
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body UpdateBOS24Request true "New API token"
// @Success 200 {object} domain.APISetting
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/settings/bos24 [patch]
// @Security Bearer
func (h *SettingsHandler) UpdateBOS24Settings(c *fiber.Ctx) error {
	var req UpdateBOS24Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	if err := h.apiSettingsRepo.UpdateBOS24Token(c.Context(), req.Token, &userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update settings",
		})
	}

	updated, err := h.apiSettingsRepo.Get(c.Context(), "bos24_api_token")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve updated settings",
		})
	}

	return c.JSON(updated)
}

// GetCompanySettings retrieves company information for invoices
// @Summary Get company settings
// @Description Retrieve company name, VAT number, address, and bank details (admin only).
// @Tags Settings
// @Produce json
// @Success 200 {object} domain.CompanySettings
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/settings/company [get]
// @Security Bearer
func (h *SettingsHandler) GetCompanySettings(c *fiber.Ctx) error {
	settings, err := h.companySettingsRepo.Get(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve company settings",
		})
	}

	return c.JSON(settings)
}

// UpdateCompanySettings updates company information for invoices
// @Summary Update company settings
// @Description Update company name, VAT number, address, and bank details (admin only). Required for valid UAE invoices.
// @Tags Settings
// @Accept json
// @Produce json
// @Param request body domain.CompanySettings true "Company details"
// @Success 200 {object} domain.CompanySettings
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/settings/company [patch]
// @Security Bearer
func (h *SettingsHandler) UpdateCompanySettings(c *fiber.Ctx) error {
	var req domain.CompanySettings
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	// Validate required fields
	if req.Name == "" || req.VATNumber == "" || req.BusinessAddress == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name, vat_number, and business_address are required",
		})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	if err := h.companySettingsRepo.Update(c.Context(), &req, &userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update company settings",
		})
	}

	updated, err := h.companySettingsRepo.Get(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve updated settings",
		})
	}

	return c.JSON(updated)
}

// Request types
type UpdateBOS24Request struct {
	Token string `json:"token"`
}
