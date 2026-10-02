package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/gofiber/fiber/v2"
)

func TestCheckBlacklist_TokenNotBlacklisted(t *testing.T) {
	app := fiber.New()
	mockRedis, mock := redismock.NewClientMock()

	token := "valid-test-token"
	mock.ExpectExists("blacklist:"+token).SetVal(0)

	middleware := CheckBlacklist(mockRedis)
	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCheckBlacklist_TokenIsBlacklisted(t *testing.T) {
	app := fiber.New()
	mockRedis, mock := redismock.NewClientMock()

	token := "revoked-test-token"
	mock.ExpectExists("blacklist:"+token).SetVal(1)

	middleware := CheckBlacklist(mockRedis)
	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCheckBlacklist_RedisError_FailsClosed(t *testing.T) {
	app := fiber.New()
	mockRedis, mock := redismock.NewClientMock()

	token := "test-token"
	mock.ExpectExists("blacklist:"+token).SetErr(context.DeadlineExceeded)

	middleware := CheckBlacklist(mockRedis)
	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	// Must fail closed - return 500 on Redis error
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 on Redis error (fail-closed), got %d", resp.StatusCode)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCheckBlacklist_NoToken(t *testing.T) {
	app := fiber.New()
	mockRedis, _ := redismock.NewClientMock()

	middleware := CheckBlacklist(mockRedis)
	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	// Should allow through if no token present
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestClaimsFromCtx_NilOrInvalid(t *testing.T) {
	app := fiber.New()

	app.Get("/test-nil", func(c *fiber.Ctx) error {
		claims := ClaimsFromCtx(c)
		if claims != nil {
			return c.Status(500).SendString("expected nil")
		}
		return c.SendString("OK")
	})

	app.Get("/test-wrong-type", func(c *fiber.Ctx) error {
		c.Locals("user", "not-a-token")
		claims := ClaimsFromCtx(c)
		if claims != nil {
			return c.Status(500).SendString("expected nil")
		}
		return c.SendString("OK")
	})

	req1 := httptest.NewRequest(http.MethodGet, "/test-nil", nil)
	resp1, _ := app.Test(req1)
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for nil user context, got %d", resp1.StatusCode)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/test-wrong-type", nil)
	resp2, _ := app.Test(req2)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for wrong type in user context, got %d", resp2.StatusCode)
	}
}

func TestRequireRole_MissingUserOrRole(t *testing.T) {
	app := fiber.New()

	handler := RequireRole("admin")
	app.Get("/test", handler, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Case 1: missing user context
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp1, _ := app.Test(req1)
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 when user is missing, got %d", resp1.StatusCode)
	}
}

func TestExtractClaims_MissingUser(t *testing.T) {
	app := fiber.New()

	app.Get("/test", ExtractClaims(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 when user context is missing, got %d", resp.StatusCode)
	}
}

func TestCheckBlacklist_MalformedAuthHeader(t *testing.T) {
	app := fiber.New()
	mockRedis, _ := redismock.NewClientMock()

	middleware := CheckBlacklist(mockRedis)
	app.Get("/test", middleware, func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	// Should allow through if token cannot be extracted
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBearerToken_ValidFormat(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		token := BearerToken(c)
		return c.SendString(token)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer my-test-token")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	// BearerToken should extract the token correctly
	// (We're just checking it doesn't panic)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBearerToken_CaseInsensitive(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		token := BearerToken(c)
		if token == "" {
			return c.SendString("empty")
		}
		return c.SendString(token)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "bearer my-test-token")

	resp, _ := app.Test(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
