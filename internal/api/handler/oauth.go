package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/apiscope"
	"github.com/maidulcu/masaar-crm/internal/partner"
)

// OAuth 2.0 authorization server, client credentials grant (RFC 6749 §4.4) for partner clients,
// with token revocation (RFC 7009) and server metadata (RFC 8414).

const partnerTokenTTL = time.Hour

type partnerTokenStore interface {
	Issue(ctx context.Context, tokenHash, clientID, scopes, secretFingerprint string, expiresAt time.Time) error
	Revoke(ctx context.Context, tokenHash, clientID string) error
}

type OAuthHandler struct {
	clients partner.Clients
	tokens  partnerTokenStore
	issuer  string // PUBLIC_API_URL; empty: derived from the request
}

func NewOAuthHandler(clients partner.Clients, tokens partnerTokenStore, publicAPIURL string) *OAuthHandler {
	return &OAuthHandler{clients: clients, tokens: tokens, issuer: strings.TrimRight(publicAPIURL, "/")}
}

// Clients returns the configured partner clients (nil-safe: no handler, no clients).
func (h *OAuthHandler) Clients() partner.Clients {
	if h == nil {
		return nil
	}
	return h.clients
}

// oauthError answers per RFC 6749 §5.2.
func oauthError(c *fiber.Ctx, status int, code, desc string) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	if status == fiber.StatusUnauthorized {
		c.Set(fiber.HeaderWWWAuthenticate, `Basic realm="oauth"`)
	}
	body := fiber.Map{"error": code}
	if desc != "" {
		body["error_description"] = desc
	}
	return c.Status(status).JSON(body)
}

// clientCredentials reads client authentication from HTTP Basic (client_secret_basic, with
// form-urlencoded id and secret) or from the form (client_secret_post), not both.
func clientCredentials(c *fiber.Ctx) (id, secret string, ok bool) {
	if auth := c.Get(fiber.HeaderAuthorization); len(auth) > 6 && strings.EqualFold(auth[:6], "basic ") {
		if c.FormValue("client_secret") != "" {
			return "", "", false
		}
		raw, err := decodeBasic(auth[6:])
		if err != nil {
			return "", "", false
		}
		u, p, found := strings.Cut(raw, ":")
		if !found {
			return "", "", false
		}
		id, err1 := url.QueryUnescape(u)
		secret, err2 := url.QueryUnescape(p)
		return id, secret, err1 == nil && err2 == nil && id != ""
	}
	id, secret = c.FormValue("client_id"), c.FormValue("client_secret")
	return id, secret, id != "" && secret != ""
}

// Token godoc
// @Summary      Get an access token (OAuth 2.0)
// @Description  Client credentials grant for partner clients. Authenticate with HTTP Basic (client_id:client_secret) or client_id and client_secret in the form. Returns a Bearer token for the Partner API, valid one hour.
// @Tags         OAuth
// @Accept       x-www-form-urlencoded
// @Produce      json
// @Param        grant_type     formData  string  true   "client_credentials"
// @Param        scope          formData  string  false  "Space-separated scopes (default: all the client holds)"
// @Param        client_id      formData  string  false  "With client_secret, when not using HTTP Basic"
// @Param        client_secret  formData  string  false  "With client_id, when not using HTTP Basic"
// @Success      200  {object}  object{access_token=string,token_type=string,expires_in=int,scope=string}
// @Failure      400  {object}  object{error=string,error_description=string}
// @Failure      401  {object}  object{error=string}
// @Router       /oauth/token [post]
func (h *OAuthHandler) Token(c *fiber.Ctx) error {
	id, secret, ok := clientCredentials(c)
	if !ok {
		return oauthError(c, fiber.StatusUnauthorized, "invalid_client", "client authentication required")
	}
	client, ok := h.clients.Authenticate(id, secret)
	if !ok {
		return oauthError(c, fiber.StatusUnauthorized, "invalid_client", "")
	}
	if gt := c.FormValue("grant_type"); gt != "client_credentials" {
		if gt == "" {
			return oauthError(c, fiber.StatusBadRequest, "invalid_request", "grant_type is required")
		}
		return oauthError(c, fiber.StatusBadRequest, "unsupported_grant_type", "")
	}
	scopes, ok := client.Grant(c.FormValue("scope"))
	if !ok {
		return oauthError(c, fiber.StatusBadRequest, "invalid_scope", "")
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return serverError(c, err)
	}
	token := middleware.PartnerTokenPrefix + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	scope := strings.Join(scopes, " ")
	if err := h.tokens.Issue(c.Context(), hex.EncodeToString(sum[:]), client.ID, scope, client.Fingerprint(), time.Now().Add(partnerTokenTTL)); err != nil {
		return serverError(c, err)
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderPragma, "no-cache")
	return c.JSON(fiber.Map{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   int(partnerTokenTTL.Seconds()),
		"scope":        scope,
	})
}

// Revoke godoc
// @Summary      Revoke an access token (RFC 7009)
// @Description  Revokes a token issued to the authenticating client. Answers 200 whether or not the token was known.
// @Tags         OAuth
// @Accept       x-www-form-urlencoded
// @Param        token  formData  string  true  "The access token"
// @Success      200
// @Failure      401  {object}  object{error=string}
// @Router       /oauth/revoke [post]
func (h *OAuthHandler) Revoke(c *fiber.Ctx) error {
	id, secret, ok := clientCredentials(c)
	if !ok {
		return oauthError(c, fiber.StatusUnauthorized, "invalid_client", "client authentication required")
	}
	client, ok := h.clients.Authenticate(id, secret)
	if !ok {
		return oauthError(c, fiber.StatusUnauthorized, "invalid_client", "")
	}
	token := c.FormValue("token")
	if token == "" {
		return oauthError(c, fiber.StatusBadRequest, "invalid_request", "token is required")
	}
	sum := sha256.Sum256([]byte(token))
	if err := h.tokens.Revoke(c.Context(), hex.EncodeToString(sum[:]), client.ID); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusOK)
}

// Metadata godoc
// @Summary      OAuth 2.0 authorization server metadata (RFC 8414)
// @Tags         OAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /.well-known/oauth-authorization-server [get]
func (h *OAuthHandler) Metadata(c *fiber.Ctx) error {
	issuer := h.issuer
	if issuer == "" {
		issuer = c.BaseURL()
	}
	scopes := make([]string, 0, len(apiscope.PartnerScopes))
	for s := range apiscope.PartnerScopes {
		scopes = append(scopes, s)
	}
	sort.Strings(scopes)
	auth := []string{"client_secret_basic", "client_secret_post"}
	c.Set(fiber.HeaderCacheControl, "public, max-age=3600")
	return c.JSON(fiber.Map{
		"issuer":                                     issuer,
		"token_endpoint":                             issuer + "/oauth/token",
		"revocation_endpoint":                        issuer + "/oauth/revoke",
		"grant_types_supported":                      []string{"client_credentials"},
		"token_endpoint_auth_methods_supported":      auth,
		"revocation_endpoint_auth_methods_supported": auth,
		"scopes_supported":                           scopes,
		"service_documentation":                      issuer + "/openapi.json",
		"response_types_supported":                   []string{},
	})
}

func decodeBasic(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	return string(b), err
}
