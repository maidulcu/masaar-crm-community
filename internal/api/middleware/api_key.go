package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"github.com/maidulcu/masaar-crm/internal/apiscope"
)

// ValidateAPIKey checks the Authorization header for "Bearer sk_live_...".
// On success, stores api_key_id, company_id, api_key_scopes in Fiber locals.
func ValidateAPIKey(keys APIKeyValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, ok, err := authenticateAPIKey(c, keys); !ok {
			return err
		}
		return c.Next()
	}
}

// APIKeyRateLimit enforces per-key rate limiting using a Redis sliding window.
// maxReq requests are allowed per window duration; excess calls get HTTP 429.
// Uses atomic Lua script to prevent race condition between INCR and EXPIRE.
func APIKeyRateLimit(rdb *redis.Client, maxReq int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if rdb == nil {
			return c.Next()
		}
		keyID := c.Locals("api_key_id")
		if keyID == nil {
			return c.Next()
		}
		redisKey := fmt.Sprintf("ratelimit:apikey:%v", keyID)
		ctx := c.Context()

		// Lua script: atomically increment counter and set expiry on first request
		script := `
if redis.call('exists', KEYS[1]) == 0 then
  redis.call('setex', KEYS[1], ARGV[1], 1)
  return 1
else
  return redis.call('incr', KEYS[1])
end
`
		result, err := rdb.Eval(ctx, script, []string{redisKey}, int(window.Seconds())).Result()
		if err != nil {
			return c.Next() // fail open — don't block on Redis errors
		}

		count, ok := result.(int64)
		if !ok {
			return c.Next()
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
		scopes, ok := c.Locals("api_key_scopes").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "missing api key scopes"})
		}
		if !apiscope.Grants(scopes, requiredScope) {
			c.Set(fiber.HeaderWWWAuthenticate, fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q`, requiredScope))
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient permissions"})
		}
		return c.Next()
	}
}
