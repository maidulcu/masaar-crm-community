package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func BenchmarkRequirePartnerScope(b *testing.B) {
	app := fiber.New()
	handler := RequirePartnerScope("partner:provision")
	app.Get("/test", func(c *fiber.Ctx) error {
		c.Locals("partner_scopes", "partner:login partner:provision partner:admin")
		return handler(c)
	}, func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := app.Test(req, -1)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
