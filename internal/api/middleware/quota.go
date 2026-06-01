package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/billing"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/redis/go-redis/v9"
)

// CheckQuota enforces per-company monthly quotas for a given resource.
// It checks a Redis counter first (fast path); on miss it falls back to the DB.
// On success it increments the counter. Returns 402 when quota is exhausted.
func CheckQuota(billingRepo *repo.BillingRepo, rdb *redis.Client, resource string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyIDStr, ok := c.Locals("company_id").(string)
		if !ok || companyIDStr == "" {
			return c.Next() // no company context — skip
		}

		companyID, err := uuid.Parse(companyIDStr)
		if err != nil {
			return c.Next()
		}

		ctx := context.Background()
		period := time.Now().UTC().Format("2006-01")

		// Fast path: read current count from Redis
		redisKey := fmt.Sprintf("usage:%s:%s:%s", resource, companyID, period)
		planKey := fmt.Sprintf("plan:%s", companyID)

		// Load plan (cached in Redis for 5 min)
		planID, _ := rdb.Get(ctx, planKey).Result()
		if planID == "" {
			cp, err := billingRepo.GetPlan(ctx, companyID)
			if err == nil {
				planID = cp.Plan
				rdb.Set(ctx, planKey, planID, 5*time.Minute)
			}
		}
		if planID == "" {
			planID = billing.PlanCommunity
		}

		plan := billing.Get(planID)
		allowed, limit := billing.CheckQuota(plan, resource, 0) // check if feature enabled at all
		if !allowed && limit == 0 {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error":    fmt.Sprintf("%s access requires a paid plan", resource),
				"plan":     planID,
				"upgrade":  "/settings/billing",
			})
		}

		if limit != -1 {
			// Get current count from Redis or DB
			countStr, err := rdb.Get(ctx, redisKey).Result()
			currentCount := 0
			if err == nil {
				fmt.Sscanf(countStr, "%d", &currentCount)
			} else {
				// Redis miss — load from DB
				usage, dbErr := billingRepo.GetUsage(ctx, companyID)
				if dbErr == nil {
					currentCount = usage[resource]
					rdb.Set(ctx, redisKey, fmt.Sprintf("%d", currentCount), 2*time.Minute)
				}
			}

			if currentCount >= limit {
				return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
					"error":   fmt.Sprintf("monthly %s quota exceeded (%d/%d)", resource, currentCount, limit),
					"plan":    planID,
					"used":    currentCount,
					"limit":   limit,
					"reset":   nextMonthReset(),
					"upgrade": "/settings/billing",
				})
			}
		}

		// Store resource and companyID for post-handler increment
		c.Locals("quota_resource", resource)
		c.Locals("quota_company_id", companyID)

		err = c.Next()

		// Increment only on success (2xx)
		if c.Response().StatusCode() < 300 {
			newCount, _ := billingRepo.IncrUsage(ctx, companyID, resource)
			rdb.Set(ctx, redisKey, fmt.Sprintf("%d", newCount), 2*time.Minute)
		}

		return err
	}
}

// CheckUserAIQuota adds a per-user daily AI cap on top of the company quota.
func CheckUserAIQuota(billingRepo *repo.BillingRepo, rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		companyIDStr, _ := c.Locals("company_id").(string)
		companyID, _ := uuid.Parse(companyIDStr)

		// Get plan to know per-user daily limit
		planKey := fmt.Sprintf("plan:%s", companyID)
		planID, _ := rdb.Get(context.Background(), planKey).Result()
		if planID == "" {
			planID = billing.PlanCommunity
		}
		plan := billing.Get(planID)
		dailyLimit := plan.Quotas.AIUserDaily
		if dailyLimit == 0 {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error":   "AI access requires a paid plan",
				"upgrade": "/settings/billing",
			})
		}
		if dailyLimit == -1 {
			return c.Next() // unlimited
		}

		claims := ClaimsFromCtx(c)
		userID, _ := claims["sub"].(string)
		if userID == "" {
			return c.Next()
		}

		day := time.Now().UTC().Format("2006-01-02")
		key := fmt.Sprintf("quota:ai:user:%s:%s", userID, day)
		ctx := context.Background()

		count, _ := rdb.Incr(ctx, key).Result()
		if count == 1 {
			rdb.Expire(ctx, key, 25*time.Hour) // slightly over 24h to cover timezone drift
		}
		if int(count) > dailyLimit {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error":      fmt.Sprintf("daily AI limit reached (%d requests)", dailyLimit),
				"used_today": count,
				"limit":      dailyLimit,
				"reset":      "midnight UTC",
			})
		}

		return c.Next()
	}
}

func nextMonthReset() string {
	now := time.Now().UTC()
	first := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return first.Format("2006-01-02")
}
