package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"

	"github.com/maidulcu/masaar-crm/internal/apiscope"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

const apiKeyPrefix = "sk_live_"

// APIKeyValidator looks up an active key by the SHA-256 hash of its plaintext.
type APIKeyValidator interface {
	ValidateKey(ctx context.Context, keyHash string) (*repo.ApiKey, error)
}

// APIKeyRoute opens one /api/v1 route to API keys that hold Scope. Path uses Fiber syntax; a
// ":param" segment only matches a UUID, so "/contacts/:id" never matches "/contacts/duplicates".
type APIKeyRoute struct {
	Method, Path, Scope string
}

var uuidSegment = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type apiKeyRouteTable []APIKeyRoute

// scopeFor returns the scope a request needs, or false when the route is closed to API keys.
// Matching mirrors the router's defaults: case-insensitive, optional trailing slash.
func (t apiKeyRouteTable) scopeFor(method, path string) (string, bool) {
	if method == fiber.MethodHead {
		method = fiber.MethodGet
	}
	segs := strings.Split(strings.TrimSuffix(path, "/"), "/")
	for _, r := range t {
		if r.Method != method {
			continue
		}
		want := strings.Split(r.Path, "/")
		if len(want) != len(segs) {
			continue
		}
		ok := true
		for i, w := range want {
			if strings.HasPrefix(w, ":") {
				ok = uuidSegment.MatchString(segs[i])
			} else {
				ok = strings.EqualFold(w, segs[i])
			}
			if !ok {
				break
			}
		}
		if ok {
			return r.Scope, true
		}
	}
	return "", false
}

// IsAPIKey reports whether the request was authenticated with an API key.
func IsAPIKey(c *fiber.Ctx) bool { return c.Locals("api_key_id") != nil }

func hasAPIKeyBearer(c *fiber.Ctx) bool { return strings.HasPrefix(BearerToken(c), apiKeyPrefix) }

// hasMachineBearer reports a bearer that is an API key or a partner token, the requests that
// get RFC 9457 error bodies.
func hasMachineBearer(c *fiber.Ctx) bool {
	t := BearerToken(c)
	return strings.HasPrefix(t, apiKeyPrefix) || strings.HasPrefix(t, PartnerTokenPrefix)
}

// authenticateAPIKey validates the bearer key and stores api_key_id, company_id, api_key_scopes
// and api_key_name in locals, and reports whether the key belongs to the demo company. On failure
// it has already written the response.
func authenticateAPIKey(c *fiber.Ctx, keys APIKeyValidator) (demo, ok bool, err error) {
	plaintext := BearerToken(c)
	if plaintext == "" {
		return false, false, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
	}
	if !strings.HasPrefix(plaintext, apiKeyPrefix) {
		return false, false, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid api key format"})
	}
	sum := sha256.Sum256([]byte(plaintext))
	apiKey, verr := keys.ValidateKey(c.Context(), hex.EncodeToString(sum[:]))
	if verr != nil {
		c.Set(fiber.HeaderWWWAuthenticate, `Bearer error="invalid_token"`)
		return false, false, c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or revoked api key"})
	}
	if !apiKey.CompanyActive {
		return false, false, c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account_suspended"})
	}
	c.Locals("api_key_id", apiKey.ID)
	// String form, consistent with ExtractClaims: handlers and tenant.From read it as a string.
	c.Locals("company_id", apiKey.CompanyID.String())
	c.Locals("api_key_scopes", apiKey.Scopes)
	c.Locals("api_key_name", apiKey.Name)
	return apiKey.CompanyDemo, true, nil
}

// APIKeyAccess lets API keys call the listed /api/v1 routes. Requests without an "sk_live_"
// bearer pass through untouched to the session (JWT) chain. Every other route is closed to keys:
// RequireRole lets a key through only because this check already ran.
func APIKeyAccess(keys APIKeyValidator, routes []APIKeyRoute) fiber.Handler {
	table := apiKeyRouteTable(routes)
	return func(c *fiber.Ctx) error {
		if !hasAPIKeyBearer(c) {
			return c.Next()
		}
		demo, ok, err := authenticateAPIKey(c, keys)
		if !ok {
			return err
		}
		if m := c.Method(); demo && m != fiber.MethodGet && m != fiber.MethodHead {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "demo_account_read_only"})
		}
		scope, open := table.scopeFor(c.Method(), c.Path())
		if !open {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "this endpoint is not available to API keys"})
		}
		if !apiscope.Grants(c.Locals("api_key_scopes").(string), scope) {
			// RFC 6750 §3.1
			c.Set(fiber.HeaderWWWAuthenticate, fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q`, scope))
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient_scope", "scope": scope})
		}
		return c.Next()
	}
}

// UnlessAPIKey skips h for requests authenticated by APIKeyAccess (session-only middleware such
// as JWT parsing, the token blacklist and the demo guard).
func UnlessAPIKey(h fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if IsAPIKey(c) {
			return c.Next()
		}
		return h(c)
	}
}

// OnlyAPIKey runs h only for API-key requests (the per-key rate limit).
func OnlyAPIKey(h fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !IsAPIKey(c) {
			return c.Next()
		}
		return h(c)
	}
}

// ProblemDetails rewrites error responses to API-key requests as RFC 9457 problem details:
// {"type":"about:blank","title":"Not Found","status":404,"detail":"lead not found"}. The original
// "error" field and any other fields are kept, so existing clients keep working.
func ProblemDetails() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := c.Next(); err != nil || !hasMachineBearer(c) {
			return err
		}
		toProblem(c)
		return nil
	}
}

func toProblem(c *fiber.Ctx) {
	res := c.Response()
	status := res.StatusCode()
	if status < 400 || !strings.HasPrefix(string(res.Header.ContentType()), fiber.MIMEApplicationJSON) {
		return
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body(), &body); err != nil {
		return
	}
	msg, ok := body["error"].(string)
	if !ok {
		return
	}
	body["type"] = "about:blank"
	body["title"] = utils.StatusMessage(status)
	body["status"] = status
	body["detail"] = msg
	out, err := json.Marshal(body)
	if err != nil {
		return
	}
	res.SetBody(out)
	res.Header.SetContentType("application/problem+json")
}
