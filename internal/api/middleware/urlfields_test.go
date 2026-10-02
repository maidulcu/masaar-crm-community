package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSafeURL(t *testing.T) {
	ok := []string{"", "https://example.com/a.pdf", "http://cdn.example.com/x?y=1", "/uploads/receipt.pdf"}
	bad := []string{
		"javascript:alert(1)", "JaVaScRiPt:alert(1)", " javascript:alert(1)", "java\tscript:alert(1)",
		"data:text/html,<script>alert(1)</script>", "vbscript:x", "file:///etc/passwd", "ftp://x/y",
		"//evil.example.com/x", `/\evil.example.com`, "https://", "mailto:a@b.c", "blob:https://x/y",
	}
	for _, s := range ok {
		if !SafeURL(s) {
			t.Errorf("%q should be allowed", s)
		}
	}
	for _, s := range bad {
		if SafeURL(s) {
			t.Errorf("%q should be rejected", s)
		}
	}
}

func TestValidateURLFieldsMiddleware(t *testing.T) {
	app := fiber.New()
	app.Use(ValidateURLFields())
	app.All("/x", func(c *fiber.Ctx) error { return c.SendString("OK") })

	do := func(method, ct, body string) int {
		req := httptest.NewRequest(method, "/x", strings.NewReader(body))
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		resp, _ := app.Test(req)
		resp.Body.Close()
		return resp.StatusCode
	}
	j := "application/json"
	cases := []struct {
		name         string
		method, body string
		want         int
	}{
		{"clean body", "POST", `{"title":"x","receipt_url":"https://a.test/r.pdf"}`, 200},
		{"empty url allowed", "POST", `{"receipt_url":""}`, 200},
		{"javascript in *_url", "POST", `{"receipt_url":"javascript:alert(1)"}`, 422},
		{"javascript in plain url", "PATCH", `{"url":"javascript:alert(1)"}`, 422},
		{"nested object", "PUT", `{"a":{"b":{"file_url":"data:text/html,x"}}}`, 422},
		{"array of urls", "POST", `{"image_urls":["https://ok.test/1.png","javascript:alert(1)"]}`, 422},
		{"non-url key is not inspected", "POST", `{"notes":"javascript:alert(1) is just text"}`, 200},
		{"GET untouched", "GET", `{"receipt_url":"javascript:alert(1)"}`, 200},
		{"invalid json passes through", "POST", `{not json`, 200},
	}
	for _, tc := range cases {
		if got := do(tc.method, j, tc.body); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
	if got := do("POST", "text/plain", `{"receipt_url":"javascript:x"}`); got != http.StatusOK {
		t.Errorf("non-JSON content type is not inspected, got %d", got)
	}
}
