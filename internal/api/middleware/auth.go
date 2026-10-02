package middleware

import (
	"strings"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/redis/go-redis/v9"
)

// AccessTokenAudience is the `aud` claim stamped on every access token and
// required by ExtractClaims. It prevents other JWTs signed with the same
// secret (e.g. refresh tokens) from being accepted as access tokens.
const AccessTokenAudience = "masaar-crm"

func JWT(secret string) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey:   jwtware.SigningKey{Key: []byte(secret), JWTAlg: "HS256"},
		ErrorHandler: jwtError,
	})
}

// JWTFromQuery authenticates from a `token` query parameter. It exists only for
// the WebSocket upgrade, because browsers cannot set an Authorization header on
// WebSocket connections. Do not use it for regular HTTP routes.
func JWTFromQuery(secret string) fiber.Handler {
	return jwtware.New(jwtware.Config{
		SigningKey:   jwtware.SigningKey{Key: []byte(secret), JWTAlg: "HS256"},
		TokenLookup:  "query:token",
		ErrorHandler: jwtError,
	})
}

// hasAudience reports whether the claims' `aud` matches want. The claim may be
// a single string or a list of strings.
func hasAudience(claims jwt.MapClaims, want string) bool {
	switch aud := claims["aud"].(type) {
	case string:
		return aud == want
	case []interface{}:
		for _, a := range aud {
			if s, ok := a.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

func jwtError(c *fiber.Ctx, err error) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "unauthorized",
	})
}

// RequireRole returns 403 if the authenticated user doesn't have one of the given roles.
func RequireRole(roles ...domain.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := ClaimsFromCtx(c)
		if claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}
		roleVal, ok := claims["role"]
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "role claim missing",
			})
		}
		roleStr, ok := roleVal.(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "invalid role claim format",
			})
		}
		role := domain.Role(roleStr)
		for _, r := range roles {
			if r == role {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "forbidden",
		})
	}
}

// ClaimsFromCtx extracts JWT claims from Fiber context safely.
// Returns nil if user token is missing or malformed.
func ClaimsFromCtx(c *fiber.Ctx) jwt.MapClaims {
	u := c.Locals("user")
	if u == nil {
		return nil
	}
	token, ok := u.(*jwt.Token)
	if !ok || token == nil {
		return nil
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil
	}
	return claims
}

// BearerToken extracts the raw token string from Authorization header.
func BearerToken(c *fiber.Ctx) string {
	auth := c.Get("Authorization")
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}
	return ""
}

// ExtractClaims reads JWT claims and sets user_id (uuid.UUID), company_id (string),
// and role (domain.Role) in Fiber locals so handlers can access them without
// repeating assertion boilerplate.
func ExtractClaims() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := ClaimsFromCtx(c)
		if claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		if !hasAudience(claims, AccessTokenAudience) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		sub, _ := claims["sub"].(string)
		userID, err := uuid.Parse(sub)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token subject"})
		}
		c.Locals("user_id", userID)

		// Read company_id from JWT claims (added during login/register)
		if cidStr, ok := claims["company_id"].(string); ok {
			c.Locals("company_id", cidStr)
		} else {
			// Fallback for pre-migration tokens — use the env var
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "session expired, please login again",
			})
		}
		if roleStr, ok := claims["role"].(string); ok {
			c.Locals("role", domain.Role(roleStr))
		}
		return c.Next()
	}
}

// CheckBlacklist verifies that the current access token has not been revoked.
// If Redis fails, it fails closed returning 500.
func CheckBlacklist(rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := BearerToken(c)
		if token == "" {
			// WebSocket upgrades carry the token in the query string (see JWTFromQuery);
			// the JWT middleware has already verified it, so only revocation remains.
			token = c.Query("token")
		}
		if token != "" {
			exists, err := rdb.Exists(c.Context(), "blacklist:"+token).Result()
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "internal server error",
				})
			}
			if exists > 0 {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "token revoked",
				})
			}
		}
		return c.Next()
	}
}
