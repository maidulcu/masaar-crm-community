package handler

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/maidulcu/masaar-crm/internal/partner"
)

type memTokens struct {
	mu      sync.Mutex
	issued  map[string]string // hash -> client
	revoked map[string]bool
}

func (m *memTokens) Issue(_ context.Context, h, client, _ string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.issued[h] = client
	return nil
}

func (m *memTokens) Revoke(_ context.Context, h, client string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.issued[h] == client {
		m.revoked[h] = true
	}
	return nil
}

func oauthApp(t *testing.T) (*fiber.App, *memTokens) {
	t.Helper()
	sum := sha256.Sum256([]byte("s3cret"))
	clients, err := partner.Parse("portal:" + hex.EncodeToString(sum[:]) + ",reader:" + hex.EncodeToString(sum[:]) + ":partner:login")
	if err != nil {
		t.Fatal(err)
	}
	store := &memTokens{issued: map[string]string{}, revoked: map[string]bool{}}
	h := NewOAuthHandler(clients, store, "https://crm.example.test")
	app := fiber.New()
	app.Post("/oauth/token", h.Token)
	app.Post("/oauth/revoke", h.Revoke)
	app.Get("/.well-known/oauth-authorization-server", h.Metadata)
	return app, store
}

func form(t *testing.T, app *fiber.App, path string, vals url.Values, basic string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(vals.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if basic != "" {
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(basic)))
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	return resp.StatusCode, body, resp.Header.Get("Cache-Control")
}

func TestOAuthToken(t *testing.T) {
	app, store := oauthApp(t)
	cc := url.Values{"grant_type": {"client_credentials"}}

	code, body, cache := form(t, app, "/oauth/token", cc, "portal:s3cret")
	tok, _ := body["access_token"].(string)
	if code != 200 || !strings.HasPrefix(tok, "mpt_") || body["token_type"] != "Bearer" || body["expires_in"] != float64(3600) ||
		body["scope"] != "partner:login partner:provision" || cache != "no-store" {
		t.Fatalf("basic auth: %d %v %q", code, body, cache)
	}
	sum := sha256.Sum256([]byte(tok))
	if store.issued[hex.EncodeToString(sum[:])] != "portal" {
		t.Error("token hash not stored for the client")
	}

	// client_secret_post, narrowed scope.
	post := url.Values{"grant_type": {"client_credentials"}, "client_id": {"portal"}, "client_secret": {"s3cret"}, "scope": {"partner:login"}}
	if code, body, _ = form(t, app, "/oauth/token", post, ""); code != 200 || body["scope"] != "partner:login" {
		t.Fatalf("post auth: %d %v", code, body)
	}

	cases := []struct {
		name   string
		vals   url.Values
		basic  string
		status int
		err    string
	}{
		{"no client auth", cc, "", 401, "invalid_client"},
		{"wrong secret", cc, "portal:nope", 401, "invalid_client"},
		{"unknown client", cc, "ghost:s3cret", 401, "invalid_client"},
		{"two auth methods", url.Values{"grant_type": {"client_credentials"}, "client_secret": {"s3cret"}}, "portal:s3cret", 401, "invalid_client"},
		{"missing grant", url.Values{}, "portal:s3cret", 400, "invalid_request"},
		{"other grant", url.Values{"grant_type": {"password"}}, "portal:s3cret", 400, "unsupported_grant_type"},
		{"scope not held", url.Values{"grant_type": {"client_credentials"}, "scope": {"partner:provision"}}, "reader:s3cret", 400, "invalid_scope"},
		{"unknown scope", url.Values{"grant_type": {"client_credentials"}, "scope": {"leads:read"}}, "portal:s3cret", 400, "invalid_scope"},
	}
	for _, c := range cases {
		if code, body, _ := form(t, app, "/oauth/token", c.vals, c.basic); code != c.status || body["error"] != c.err {
			t.Errorf("%s: %d %v, want %d %s", c.name, code, body, c.status, c.err)
		}
	}

	// Revocation: only the owning client, always 200.
	if code, _, _ := form(t, app, "/oauth/revoke", url.Values{"token": {tok}}, "reader:s3cret"); code != 200 || store.revoked[hex.EncodeToString(sum[:])] {
		t.Errorf("another client revoked the token (%d)", code)
	}
	if code, _, _ := form(t, app, "/oauth/revoke", url.Values{"token": {tok}}, "portal:s3cret"); code != 200 || !store.revoked[hex.EncodeToString(sum[:])] {
		t.Errorf("owner could not revoke (%d)", code)
	}
	if code, _, _ := form(t, app, "/oauth/revoke", url.Values{"token": {"mpt_unknown"}}, "portal:s3cret"); code != 200 {
		t.Errorf("unknown token: %d, want 200", code)
	}
}

func TestOAuthMetadata(t *testing.T) {
	app, _ := oauthApp(t)
	resp, err := app.Test(httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&m)
	if m["issuer"] != "https://crm.example.test" || m["token_endpoint"] != "https://crm.example.test/oauth/token" ||
		m["revocation_endpoint"] != "https://crm.example.test/oauth/revoke" {
		t.Fatalf("metadata = %v", m)
	}
}
