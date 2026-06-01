package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/redis/go-redis/v9"
)

// ValidateAPIKey checks Authorization header for "Bearer sk_live_..." format.
// On success, stores api_key_id, company_id, api_key_scopes in Fiber locals.
func ValidateAPIKey(apiKeyRepo *repo.ApiKeyRepo) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}

		// Parse "Bearer sk_live_..."
		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format"})
		}

		plaintext := parts[1]
		if !strings.HasPrefix(plaintext, "sk_live_") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid api key format"})
		}

		// Hash the key
		hashSum := sha256.Sum256([]byte(plaintext))
		keyHash := hex.EncodeToString(hashSum[:])

		// Validate against database
		apiKey, err := apiKeyRepo.ValidateKey(c.Context(), keyHash)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or revoked api key"})
		}

		// Store in locals
		c.Locals("api_key_id", apiKey.ID)
		c.Locals("company_id", apiKey.CompanyID)
		c.Locals("api_key_scopes", apiKey.Scopes)
		c.Locals("api_key_name", apiKey.Name)

		return c.Next()
	}
}

// APIKeyRateLimit enforces per-key rate limiting using a Redis sliding window.
// maxReq requests are allowed per window duration; excess calls get HTTP 429.
func APIKeyRateLimit(rdb *redis.Client, maxReq int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		keyID := c.Locals("api_key_id")
		if keyID == nil {
			return c.Next()
		}
		redisKey := fmt.Sprintf("ratelimit:apikey:%v", keyID)
		ctx := context.Background()

		count, err := rdb.Incr(ctx, redisKey).Result()
		if err != nil {
			return c.Next() // fail open — don't block on Redis errors
		}
		if count == 1 {
			rdb.Expire(ctx, redisKey, window)
		}
		if count > int64(maxReq) {
			c.Set("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("rate limit exceeded: max %d requests per %s", maxReq, window),
			})
		}
		return c.Next()
	}
}

// RequireAPIKeyScope checks if the API key has a specific scope.
func RequireAPIKeyScope(requiredScope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scopes := c.Locals("api_key_scopes")
		if scopes == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "missing api key scopes"})
		}

		scopesStr := scopes.(string)
		for _, scope := range strings.Split(scopesStr, ",") {
			if strings.TrimSpace(scope) == requiredScope {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
	}
}
