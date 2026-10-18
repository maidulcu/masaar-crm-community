package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/jurisdiction"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type SettingsHandler struct {
	apiSettingsRepo     *repo.SettingsRepo
	companySettingsRepo *repo.CompanySettingsRepo
}

func NewSettingsHandler(apiSettingsRepo *repo.SettingsRepo, companySettingsRepo *repo.CompanySettingsRepo) *SettingsHandler {
	return &SettingsHandler{
		apiSettingsRepo:     apiSettingsRepo,
		companySettingsRepo: companySettingsRepo,
	}
}

// GetBOS24Settings retrieves the BOS24 API token configuration
// @Summary Get Waqov data API settings
// @Description Retrieve Waqov data API token (admin only). Used to check if integration is configured.
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
// @Summary Update Waqov data API token
// @Description Update Waqov data API token (admin only). Token is required to enable real estate features.
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

	// Validate required fields. The VAT number (TRN) is optional: a business below the
	// registration threshold has none.
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.BusinessAddress) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name and business_address are required",
		})
	}

	current, err := h.companySettingsRepo.Get(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve company settings",
		})
	}
	// The row is the caller's company's; never trust an id from the body.
	req.ID = current.ID
	if msg := mergeLocale(&req, current); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
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

// mergeLocale keeps the stored country, region, currency and timezone for fields the
// request leaves empty (older clients do not send them), then validates the result
// against internal/jurisdiction. It returns a client-facing message, or "".
func mergeLocale(req, current *domain.CompanySettings) string {
	if req.Country == "" {
		req.Country = current.Country
	}
	if req.Currency == "" {
		req.Currency = current.Currency
	}
	if req.Timezone == "" {
		req.Timezone = current.Timezone
	}
	countryChanged := !strings.EqualFold(req.Country, current.Country)
	if req.Region == "" && !countryChanged {
		req.Region = current.Region
	}
	req.Country = strings.ToUpper(strings.TrimSpace(req.Country))
	req.Region = strings.ToUpper(strings.TrimSpace(req.Region))
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.Timezone = strings.TrimSpace(req.Timezone)

	rules, ok := jurisdiction.For(req.Country)
	if !ok {
		return "unsupported country"
	}
	req.VATNumber = jurisdiction.NormalizeTaxID(req.VATNumber)
	if !rules.ValidTaxID(req.VATNumber) {
		return fmt.Sprintf("vat_number must be %d digits", rules.TaxIDDigits)
	}
	if !rules.HasRegion(req.Region) {
		return "region does not belong to the selected country"
	}
	if !jurisdiction.IsCurrency(req.Currency) {
		return "unsupported currency"
	}
	if !jurisdiction.ValidTimezone(req.Timezone) {
		return "invalid timezone"
	}
	return ""
}

// GetLocale returns the company's country, region, currency and timezone together with
// the country's rules (VAT rates, weekend, regions, currency decimals). Every role may
// read it: the web app formats money and dates with it.
// @Summary Company locale and country rules
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/settings/locale [get]
// @Security Bearer
func (h *SettingsHandler) GetLocale(c *fiber.Ctx) error {
	loc, err := h.companySettingsRepo.Locale(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	decimals, ok := jurisdiction.CurrencyDecimals(loc.Currency)
	if !ok {
		decimals = 2
	}
	return c.JSON(fiber.Map{
		"country":           loc.Country,
		"region":            loc.Region,
		"currency":          loc.Currency,
		"currency_decimals": decimals,
		"timezone":          loc.Timezone,
		"show_hijri":        loc.ShowHijri,
		"rules":             jurisdiction.MustFor(loc.Country),
	})
}

// ListJurisdictions returns every supported country's rules, for the settings form.
// @Summary Supported countries
// @Tags Settings
// @Produce json
// @Success 200 {array} jurisdiction.Rules
// @Router /api/v1/settings/jurisdictions [get]
// @Security Bearer
func (h *SettingsHandler) ListJurisdictions(c *fiber.Ctx) error {
	return c.JSON(jurisdiction.Supported())
}

// Request types
type UpdateBOS24Request struct {
	Token string `json:"token"`
}
