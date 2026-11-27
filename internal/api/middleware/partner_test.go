package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/maidulcu/masaar-crm/internal/partner"
)

type fakePartnerTokens map[string][3]string // hash -> client, scopes, secret fingerprint

func (f fakePartnerTokens) Validate(_ context.Context, h string) (string, string, string, error) {
	if v, ok := f[h]; ok {
		return v[0], v[1], v[2], nil
	}
	return "", "", "", errors.New("not found")
}

func hashTok(t string) string { s := sha256.Sum256([]byte(t)); return hex.EncodeToString(s[:]) }

// Tokens follow the current configuration: a removed client or a withdrawn scope stops working.
func TestPartnerAuthFollowsConfig(t *testing.T) {
	sum := sha256.Sum256([]byte("x"))
	clients, err := partner.Parse("kept:" + hex.EncodeToString(sum[:]) + ":partner:login")
	if err != nil {
		t.Fatal(err)
	}
	tokens := fakePartnerTokens{
		hashTok("mpt_kept"):    {"kept", "partner:login partner:provision"}, // provision since withdrawn
		hashTok("mpt_removed"): {"gone", "partner:login"},
	}
	app := fiber.New()
	app.Use(PartnerAuth(tokens, clients))
	app.Get("/login", RequirePartnerScope("partner:login"), func(c *fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/provision", RequirePartnerScope("partner:provision"), func(c *fiber.Ctx) error { return c.SendString("ok") })

	cases := []struct {
		tok, path string
		want      int
	}{
		{"mpt_kept", "/login", 200},
		{"mpt_kept", "/provision", 403},
		{"mpt_removed", "/login", 401},
		{"mpt_unknown", "/login", 401},
		{"sk_live_abc", "/login", 401},
		{"", "/login", 401},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", c.path, nil)
		if c.tok != "" {
			req.Header.Set("Authorization", "Bearer "+c.tok)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != c.want {
			t.Errorf("%s %s = %d, want %d", c.tok, c.path, resp.StatusCode, c.want)
		}
	}
}

// A token issued under a secret that has since been rotated stops working at once; tokens
// without a recorded fingerprint (issued before fingerprints existed) still work until expiry.
func TestPartnerAuthRefusesTokensOfARotatedSecret(t *testing.T) {
	cfg := func(secret string) partner.Clients {
		sum := sha256.Sum256([]byte(secret))
		cs, err := partner.Parse("acme:" + hex.EncodeToString(sum[:]))
		if err != nil {
			t.Fatal(err)
		}
		return cs
	}
	before, after := cfg("old-secret"), cfg("new-secret")
	tokens := fakePartnerTokens{
		hashTok("mpt_old"):    {"acme", "partner:login", before["acme"].Fingerprint()},
		hashTok("mpt_new"):    {"acme", "partner:login", after["acme"].Fingerprint()},
		hashTok("mpt_legacy"): {"acme", "partner:login", ""},
	}
	app := fiber.New()
	app.Use(PartnerAuth(tokens, after)) // the secret has been rotated
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
	for tok, want := range map[string]int{"mpt_old": 401, "mpt_new": 200, "mpt_legacy": 200} {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != want {
			t.Errorf("%s = %d, want %d", tok, resp.StatusCode, want)
		}
	}
	if before["acme"].Fingerprint() == after["acme"].Fingerprint() {
		t.Fatal("fingerprint does not change with the secret")
	}
}
