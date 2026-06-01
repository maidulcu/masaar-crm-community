package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// TrialCheck reads company_id from JWT claims (not Locals) and verifies the
// company is active. If the trial has expired, it auto-downgrades to community.
// Sets plan and on_trial in locals for downstream handlers.
func TrialCheck(companyRepo *repo.CompanyRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := ClaimsFromCtx(c)
		cidStr, ok := claims["company_id"].(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
		}
		companyID, err := uuid.Parse(cidStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
		}

		company, err := companyRepo.GetByID(c.Context(), companyID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "company not found"})
		}

		if !company.IsActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account_suspended"})
		}

		// Auto-downgrade expired trials
		if company.OnTrial && company.TrialEndsAt != nil && company.TrialEndsAt.Before(time.Now()) {
			_ = companyRepo.EndTrial(c.Context(), companyID)
			company.Plan = "community"
			company.OnTrial = false
		}

		c.Locals("plan", company.Plan)
		c.Locals("on_trial", company.OnTrial)
		return c.Next()
	}
}
