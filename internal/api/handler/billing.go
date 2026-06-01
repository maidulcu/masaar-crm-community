package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/billing"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type BillingHandler struct {
	billingRepo  *repo.BillingRepo
	settingsRepo *repo.CompanySettingsRepo
	stripeConfig *billing.StripeConfig
}

func NewBillingHandler(
	billingRepo *repo.BillingRepo,
	settingsRepo *repo.CompanySettingsRepo,
	stripeCfg *billing.StripeConfig,
) *BillingHandler {
	return &BillingHandler{
		billingRepo:  billingRepo,
		settingsRepo: settingsRepo,
		stripeConfig: stripeCfg,
	}
}

// GetBilling returns the company's active plan and usage counters.
func (h *BillingHandler) GetBilling(c *fiber.Ctx) error {
	ctx := c.Context()

	companyIDStr, _ := c.Locals("company_id").(string)
	companyID, _ := uuid.Parse(companyIDStr)

	companyPlan, err := h.billingRepo.GetPlan(ctx, companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load plan"})
	}

	usage, err := h.billingRepo.GetUsage(ctx, companyID)
	if err != nil {
		usage = map[string]int{"bos24": 0, "ai": 0, "pdf": 0}
	}

	plan := billing.Get(companyPlan.Plan)

	quotaDisplay := func(used, limit int) fiber.Map {
		if limit == 0 {
			return fiber.Map{"used": 0, "limit": 0, "available": false}
		}
		if limit == -1 {
			return fiber.Map{"used": used, "limit": -1, "available": true, "unlimited": true}
		}
		return fiber.Map{"used": used, "limit": limit, "available": true, "pct": used * 100 / limit}
	}

	planList := make([]fiber.Map, 0, len(billing.Plans))
	for _, p := range billing.Plans {
		planList = append(planList, fiber.Map{
			"id":       p.ID,
			"name":     p.Name,
			"price":    p.PriceUSDMonth,
			"current":  p.ID == plan.ID,
			"features": p.Features,
			"quotas": fiber.Map{
				"bos24_monthly": p.Quotas.BOS24Monthly,
				"ai_monthly":    p.Quotas.AIMonthly,
				"pdf_monthly":   p.Quotas.PDFMonthly,
			},
		})
	}

	return c.JSON(fiber.Map{
		"plan": fiber.Map{
			"id":             plan.ID,
			"name":           plan.Name,
			"price":          plan.PriceUSDMonth,
			"started_at":     companyPlan.PlanStartedAt,
			"expires_at":     companyPlan.PlanExpiresAt,
			"stripe_enabled": false,
			"has_sub":        false,
		},
		"usage": fiber.Map{
			"bos24": quotaDisplay(usage["bos24"], plan.Quotas.BOS24Monthly),
			"ai":    quotaDisplay(usage["ai"], plan.Quotas.AIMonthly),
			"pdf":   quotaDisplay(usage["pdf"], plan.Quotas.PDFMonthly),
			"reset": nextMonthReset(),
		},
		"plans": planList,
	})
}

// CreateCheckout is unavailable in the community edition.
func (h *BillingHandler) CreateCheckout(c *fiber.Ctx) error {
	return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
		"error":       "Upgrade via https://masaar.io/pricing — self-hosted billing is a Pro feature",
		"upgrade_url": "https://masaar.io/pricing",
	})
}

// CreatePortal is unavailable in the community edition.
func (h *BillingHandler) CreatePortal(c *fiber.Ctx) error {
	return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
		"error":       "Billing portal requires a Pro plan",
		"upgrade_url": "https://masaar.io/pricing",
	})
}

// StripeWebhook is a no-op in the community edition.
func (h *BillingHandler) StripeWebhook(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}

func nextMonthReset() string {
	now := time.Now().UTC()
	first := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return first.Format("2006-01-02")
}

// GetUsage returns current month usage counters.
func (h *BillingHandler) GetUsage(c *fiber.Ctx) error {
	companyIDStr, _ := c.Locals("company_id").(string)
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company"})
	}

	companyPlan, _ := h.billingRepo.GetPlan(c.Context(), companyID)
	plan := billing.Get(companyPlan.Plan)

	usage, err := h.billingRepo.GetUsage(c.Context(), companyID)
	if err != nil {
		usage = map[string]int{"bos24": 0, "ai": 0, "pdf": 0}
	}

	claims := middleware.ClaimsFromCtx(c)
	userID, _ := claims["sub"].(string)

	return c.JSON(fiber.Map{
		"plan":  plan.ID,
		"reset": nextMonthReset(),
		"user":  userID,
		"bos24": fiber.Map{"used": usage["bos24"], "limit": plan.Quotas.BOS24Monthly},
		"ai":    fiber.Map{"used": usage["ai"], "limit": plan.Quotas.AIMonthly},
		"pdf":   fiber.Map{"used": usage["pdf"], "limit": plan.Quotas.PDFMonthly},
	})
}
