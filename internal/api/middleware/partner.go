package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/maidulcu/masaar-crm/internal/partner"
)

// PartnerTokenPrefix starts every partner access token (OAuth 2.0 client credentials).
const PartnerTokenPrefix = "mpt_"

// PartnerTokenValidator returns the client, space-separated scopes and secret fingerprint ("" if
// not recorded) of a live token hash.
type PartnerTokenValidator interface {
	Validate(ctx context.Context, tokenHash string) (clientID, scopes, secretFingerprint string, err error)
}

// PartnerAuth authenticates Partner API requests ("Bearer mpt_…") and stores partner_client_id
// and partner_scopes in locals. Tokens are checked against the current configuration too: a
// client removed from PARTNER_CLIENTS, or a scope taken away, stops working at once.
func PartnerAuth(tokens PartnerTokenValidator, clients partner.Clients) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tok := BearerToken(c)
		if !strings.HasPrefix(tok, PartnerTokenPrefix) {
			c.Set(fiber.HeaderWWWAuthenticate, `Bearer error="invalid_token"`)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "partner access token required"})
		}
		sum := sha256.Sum256([]byte(tok))
		clientID, scopes, fingerprint, err := tokens.Validate(c.Context(), hex.EncodeToString(sum[:]))
		if err != nil {
			c.Set(fiber.HeaderWWWAuthenticate, `Bearer error="invalid_token"`)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}
		client, ok := clients[clientID]
		// A token issued under a secret that has since been rotated is refused.
		if ok && fingerprint != "" && fingerprint != client.Fingerprint() {
			ok = false
		}
		if !ok {
			c.Set(fiber.HeaderWWWAuthenticate, `Bearer error="invalid_token"`)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}
		held := map[string]bool{}
		for _, s := range client.Scopes {
			held[s] = true
		}
		var live []string
		for _, s := range strings.Fields(scopes) {
			if held[s] {
				live = append(live, s)
			}
		}
		c.Locals("partner_client_id", clientID)
		c.Locals("partner_scopes", strings.Join(live, " "))
		return c.Next()
	}
}

// RequirePartnerScope refuses partner tokens without scope.
func RequirePartnerScope(scope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scopes, _ := c.Locals("partner_scopes").(string)
		for _, s := range strings.Fields(scopes) {
			if s == scope {
				return c.Next()
			}
		}
		c.Set(fiber.HeaderWWWAuthenticate, fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q`, scope))
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient_scope", "scope": scope})
	}
}
