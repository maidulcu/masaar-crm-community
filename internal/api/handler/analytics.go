package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/repo"
)

type AnalyticsHandler struct {
	analyticsRepo *repo.AnalyticsRepository
}

func NewAnalyticsHandler(analyticsRepo *repo.AnalyticsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsRepo: analyticsRepo,
	}
}

// GetTenantOverview returns company-wide tenant analytics
// @Summary Get tenant overview analytics
// @Description Returns company-wide KPIs: total tenants, occupancy rate, revenue, collection rate
// @Tags Analytics
// @Security Bearer
// @Produce json
// @Success 200 {object} map[string]interface{} "Tenant analytics data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/v1/analytics/tenant-overview [get]
func (h *AnalyticsHandler) GetTenantOverview(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	analytics, err := h.analyticsRepo.GetTenantAnalytics(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch tenant analytics",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": analytics,
	})
}

// ListPropertiesAnalytics returns paginated property performance metrics
// @Summary List property analytics
// @Description Returns performance metrics for all properties with pagination
// @Tags Analytics
// @Security Bearer
// @Param limit query int false "Limit results" default(10)
// @Param offset query int false "Offset results" default(0)
// @Produce json
// @Success 200 {object} map[string]interface{} "Properties analytics list"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/v1/analytics/properties [get]
func (h *AnalyticsHandler) ListPropertiesAnalytics(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	properties, total, err := h.analyticsRepo.ListPropertiesAnalytics(c.Context(), companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch properties analytics",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": properties,
		"meta": fiber.Map{
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"count":   len(properties),
		},
	})
}

// GetPropertyAnalytics returns detailed analytics for a specific property
// @Summary Get property analytics
// @Description Returns detailed performance metrics for a specific property
// @Tags Analytics
// @Security Bearer
// @Param propertyID path string true "Property ID"
// @Produce json
// @Success 200 {object} map[string]interface{} "Property analytics data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Not found"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/v1/analytics/properties/{propertyID} [get]
func (h *AnalyticsHandler) GetPropertyAnalytics(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	propertyIDStr := c.Params("propertyID")
	propertyID, err := uuid.Parse(propertyIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid property ID",
		})
	}

	analytics, err := h.analyticsRepo.GetPropertyAnalytics(c.Context(), companyID, propertyID)
	if err != nil {
		if errors.Is(err, fiber.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Property not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch property analytics",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": analytics,
	})
}

// ListTenantsPerformance returns paginated tenant performance metrics
// @Summary List tenant performance analytics
// @Description Returns performance metrics for all tenants with risk scoring and payment history
// @Tags Analytics
// @Security Bearer
// @Param limit query int false "Limit results" default(10)
// @Param offset query int false "Offset results" default(0)
// @Produce json
// @Success 200 {object} map[string]interface{} "Tenants performance list"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/v1/analytics/tenants [get]
func (h *AnalyticsHandler) ListTenantsPerformance(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	tenants, total, err := h.analyticsRepo.ListTenantsPerformance(c.Context(), companyID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch tenants analytics",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": tenants,
		"meta": fiber.Map{
			"total":   total,
			"limit":   limit,
			"offset":  offset,
			"count":   len(tenants),
		},
	})
}

// GetTenantPerformance returns detailed analytics for a specific tenant
// @Summary Get tenant performance analytics
// @Description Returns detailed performance metrics, payment history, and risk score for a specific tenant
// @Tags Analytics
// @Security Bearer
// @Param tenantID path string true "Tenant ID"
// @Produce json
// @Success 200 {object} map[string]interface{} "Tenant performance analytics"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 404 {object} map[string]interface{} "Not found"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/v1/analytics/tenants/{tenantID} [get]
func (h *AnalyticsHandler) GetTenantPerformance(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	tenantIDStr := c.Params("tenantID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid tenant ID",
		})
	}

	analytics, err := h.analyticsRepo.GetTenantPerformance(c.Context(), companyID, tenantID)
	if err != nil {
		if errors.Is(err, fiber.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Tenant not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch tenant performance analytics",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": analytics,
	})
}

// GetFinancialAnalytics returns financial analytics for a date range
// @Summary Get financial analytics
// @Description Returns revenue, expenses, and profit metrics for a specified date range
// @Tags Analytics
// @Security Bearer
// @Param startDate query string false "Start date (YYYY-MM-DD)"
// @Param endDate query string false "End date (YYYY-MM-DD)"
// @Produce json
// @Success 200 {object} map[string]interface{} "Financial analytics data"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/v1/analytics/financial [get]
func (h *AnalyticsHandler) GetFinancialAnalytics(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	startDateStr := c.Query("startDate", time.Now().AddDate(0, -1, 0).Format("2006-01-02"))
	endDateStr := c.Query("endDate", time.Now().Format("2006-01-02"))

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid start date format (use YYYY-MM-DD)",
		})
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid end date format (use YYYY-MM-DD)",
		})
	}

	analytics, err := h.analyticsRepo.GetFinancialAnalytics(c.Context(), companyID, startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch financial analytics",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": analytics,
	})
}

// GetMaintenanceAnalytics returns maintenance task analytics
// @Summary Get maintenance analytics
// @Description Returns task completion rates, average response time, and priority distribution
// @Tags Analytics
// @Security Bearer
// @Produce json
// @Success 200 {object} map[string]interface{} "Maintenance analytics data"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /api/v1/analytics/maintenance [get]
func (h *AnalyticsHandler) GetMaintenanceAnalytics(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	analytics, err := h.analyticsRepo.GetMaintenanceAnalytics(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch maintenance analytics",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": analytics,
	})
}
