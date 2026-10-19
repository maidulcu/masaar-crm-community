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

type fakePartnerTokens map[string][2]string // hash -> client, scopes

func (f fakePartnerTokens) Validate(_ context.Context, h string) (string, string, error) {
	if v, ok := f[h]; ok {
		return v[0], v[1], nil
	}
	return "", "", errors.New("not found")
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
