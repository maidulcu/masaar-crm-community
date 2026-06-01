package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// demoReadOnlyPaths lists path prefixes that demo accounts ARE allowed to write to.
// Everything else is blocked for POST / PATCH / PUT / DELETE.
var demoAllowedWritePrefixes = []string{
	"/api/v1/auth/logout",    // must be able to log out
	"/api/v1/auth/refresh",   // token refresh
	"/api/v1/notifications",  // mark-read is a PATCH but harmless
}

// DemoGuard blocks mutating HTTP methods (POST/PATCH/PUT/DELETE) for demo
// company accounts, except for a small allowlist of safe operations.
// It reads the is_demo claim from the JWT (set in ExtractClaims).
// Always call this AFTER ExtractClaims in the middleware chain.
func DemoGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Only applies to mutating methods
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		// Read is_demo from JWT claims (set by generateTokenPair → ExtractClaims passes it through)
		claims := ClaimsFromCtx(c)
		isDemo, _ := claims["is_demo"].(bool)
		if !isDemo {
			return c.Next()
		}

		// Allow the explicit write paths even for demo accounts
		path := c.Path()
		for _, prefix := range demoAllowedWritePrefixes {
			if strings.HasPrefix(path, prefix) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "demo_account_read_only",
			"message": "This is a demo account. Write operations are disabled. Please sign up for a free trial to use this feature.",
		})
	}
}
