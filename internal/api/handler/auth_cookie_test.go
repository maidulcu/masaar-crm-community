package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/config"
)

func cookieTestHandler(env, origins, sameSite string) *AuthHandler {
	return &AuthHandler{config: &config.Config{
		AppEnv: env, AllowedOrigins: origins, AuthCookieSameSite: sameSite, JWTRefreshExpiryDays: 7,
	}}
}

func TestAttachRefreshBodyVsCookie(t *testing.T) {
	h := cookieTestHandler("production", "https://crm.example.com", "")
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		return c.JSON(h.attachRefresh(c, fiber.Map{"access_token": "a"}, "RT"))
	})

	// Default (API / mobile) mode: token stays in the body, no cookie.
	resp, _ := app.Test(httptest.NewRequest("POST", "/", nil))
	body := readAll(t, resp.Body)
	if !strings.Contains(body, `"refresh_token":"RT"`) {
		t.Fatalf("body mode should include refresh_token: %s", body)
	}
	if len(resp.Cookies()) != 0 {
		t.Fatalf("body mode must not set cookies")
	}

	// Browser mode: token only in an HttpOnly, Secure, path-scoped cookie.
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("X-Auth-Mode", "cookie")
	resp, _ = app.Test(req)
	body = readAll(t, resp.Body)
	if strings.Contains(body, "RT") || strings.Contains(body, "refresh_token") {
		t.Fatalf("cookie mode leaked refresh token into body: %s", body)
	}
	cs := resp.Cookies()
	if len(cs) != 1 {
		t.Fatalf("want 1 cookie, got %d", len(cs))
	}
	ck := cs[0]
	if ck.Name != "masaar_rt" || ck.Value != "RT" || !ck.HttpOnly || !ck.Secure || ck.Path != "/api/v1/auth" {
		t.Fatalf("bad cookie attributes: %+v", ck)
	}
	if ck.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", ck.SameSite)
	}
}

func TestSameSiteNoneForcesSecureEvenInDev(t *testing.T) {
	h := cookieTestHandler("development", "http://localhost:3000", "none")
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error { h.setRefreshCookie(c, "RT"); return nil })
	resp, _ := app.Test(httptest.NewRequest("POST", "/", nil))
	if ck := resp.Cookies()[0]; !ck.Secure {
		t.Fatalf("SameSite=None cookie must be Secure: %+v", ck)
	}
}

func TestOriginAllowed(t *testing.T) {
	cases := []struct {
		allowed, origin string
		want            bool
	}{
		{"https://crm.example.com", "", true}, // non-browser client
		{"https://crm.example.com", "https://crm.example.com", true},
		{"https://crm.example.com/, https://b.example.com", "https://b.example.com", true},
		{"https://crm.example.com", "https://evil.example", false},
		{"https://crm.example.com", "http://crm.example.com", false}, // scheme matters
		{"https://crm.example.com", "https://crm.example.com.evil.io", false},
		{"https://crm.example.com", "null", false},
		{"*", "https://anything.example", true},
	}
	for _, tc := range cases {
		h := cookieTestHandler("production", tc.allowed, "")
		app := fiber.New()
		var got bool
		app.Post("/", func(c *fiber.Ctx) error { got = h.originAllowed(c); return nil })
		req := httptest.NewRequest("POST", "/", nil)
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		_, _ = app.Test(req)
		if got != tc.want {
			t.Errorf("allowed=%q origin=%q: got %v want %v", tc.allowed, tc.origin, got, tc.want)
		}
	}
}

func readAll(t *testing.T, r interface{ Read([]byte) (int, error) }) string {
	t.Helper()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return sb.String()
}
