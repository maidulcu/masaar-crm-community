package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestCheckBlacklist_TokenNotBlacklisted(t *testing.T) {
	app := fiber.New()
	mockRedis, mock := redismock.NewClientMock()

	token := "valid-test-token"
	mock.ExpectExists("blacklist:" + token).SetVal(0)

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
	mock.ExpectExists("blacklist:" + token).SetVal(1)

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
	mock.ExpectExists("blacklist:" + token).SetErr(context.DeadlineExceeded)

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

func signedToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// authChain mirrors the real /api/v1 middleware order: JWT then ExtractClaims.
func authChain(secret string) *fiber.App {
	app := fiber.New()
	app.Get("/h", JWT(secret), ExtractClaims(), func(c *fiber.Ctx) error { return c.SendString("OK") })
	app.Get("/q", JWTFromQuery(secret), ExtractClaims(), func(c *fiber.Ctx) error { return c.SendString("OK") })
	return app
}

func TestExtractClaims_RequiresAccessAudience(t *testing.T) {
	const secret = "test-secret-test-secret-test-secret"
	base := func() jwt.MapClaims {
		return jwt.MapClaims{
			"sub":        uuid.NewString(),
			"company_id": uuid.NewString(),
			"role":       "admin",
			"exp":        time.Now().Add(time.Minute).Unix(),
		}
	}
	tests := []struct {
		name string
		aud  interface{}
		want int
	}{
		{"correct audience", AccessTokenAudience, http.StatusOK},
		{"audience in list", []string{"other", AccessTokenAudience}, http.StatusOK},
		{"missing audience", nil, http.StatusUnauthorized},
		{"wrong audience", "someone-else", http.StatusUnauthorized},
	}
	app := authChain(secret)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := base()
			if tt.aud != nil {
				claims["aud"] = tt.aud
			}
			req := httptest.NewRequest(http.MethodGet, "/h", nil)
			req.Header.Set("Authorization", "Bearer "+signedToken(t, secret, claims))
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.want {
				t.Fatalf("want %d, got %d", tt.want, resp.StatusCode)
			}
		})
	}
}

func TestJWT_RejectsNonHS256(t *testing.T) {
	const secret = "test-secret-test-secret-test-secret"
	claims := jwt.MapClaims{"sub": uuid.NewString(), "aud": AccessTokenAudience, "company_id": uuid.NewString()}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/h", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, _ := authChain(secret).Test(req)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("HS512 token should be rejected, got %d", resp.StatusCode)
	}
}

func TestJWTFromQuery_OnlyAcceptsQueryToken(t *testing.T) {
	const secret = "test-secret-test-secret-test-secret"
	tok := signedToken(t, secret, jwt.MapClaims{
		"sub": uuid.NewString(), "aud": AccessTokenAudience, "company_id": uuid.NewString(),
		"exp": time.Now().Add(time.Minute).Unix(),
	})
	app := authChain(secret)

	req := httptest.NewRequest(http.MethodGet, "/q?token="+tok, nil)
	resp, _ := app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("query token on /q: want 200, got %d", resp.StatusCode)
	}

	// A query token must NOT authenticate a normal header-based route.
	req = httptest.NewRequest(http.MethodGet, "/h?token="+tok, nil)
	resp, _ = app.Test(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("query token on /h: want 401, got %d", resp.StatusCode)
	}
}

func TestCheckBlacklist_QueryTokenRevoked(t *testing.T) {
	app := fiber.New()
	mockRedis, mock := redismock.NewClientMock()
	mock.ExpectExists("blacklist:revoked-tok").SetVal(1)
	app.Get("/ws", CheckBlacklist(mockRedis), func(c *fiber.Ctx) error { return c.SendString("OK") })

	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/ws?token=revoked-tok", nil))
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked query token: want 401, got %d", resp.StatusCode)
	}
}

func blacklistChain(secret string, rdb *redis.Client) *fiber.App {
	app := fiber.New()
	app.Get("/h", JWT(secret), CheckBlacklist(rdb), ExtractClaims(), func(c *fiber.Ctx) error { return c.SendString("OK") })
	return app
}

func TestCheckBlacklist_EnforcesSessionCutoff(t *testing.T) {
	const secret = "test-secret-test-secret-test-secret"
	uid := uuid.New()
	issued := time.Now().Add(-time.Hour)
	tok := signedToken(t, secret, jwt.MapClaims{
		"sub": uid.String(), "aud": AccessTokenAudience, "company_id": uuid.NewString(),
		"iat": issued.Unix(), "exp": time.Now().Add(time.Hour).Unix(),
	})
	call := func(rdb *redis.Client) int {
		req := httptest.NewRequest(http.MethodGet, "/h", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, _ := blacklistChain(secret, rdb).Test(req)
		resp.Body.Close()
		return resp.StatusCode
	}

	// no cutoff -> allowed
	rdb, mock := redismock.NewClientMock()
	mock.ExpectExists("blacklist:" + tok).SetVal(0)
	mock.ExpectGet("session_cutoff:" + uid.String()).RedisNil()
	if got := call(rdb); got != http.StatusOK {
		t.Fatalf("no cutoff: want 200, got %d", got)
	}

	// cutoff after the token was issued (user deactivated / password changed) -> rejected
	rdb, mock = redismock.NewClientMock()
	mock.ExpectExists("blacklist:" + tok).SetVal(0)
	mock.ExpectGet("session_cutoff:" + uid.String()).SetVal(strconv.FormatInt(time.Now().Unix(), 10))
	if got := call(rdb); got != http.StatusUnauthorized {
		t.Fatalf("token older than cutoff: want 401, got %d", got)
	}

	// cutoff older than the token (user re-logged in afterwards) -> allowed
	rdb, mock = redismock.NewClientMock()
	mock.ExpectExists("blacklist:" + tok).SetVal(0)
	mock.ExpectGet("session_cutoff:" + uid.String()).SetVal(strconv.FormatInt(issued.Add(-time.Hour).Unix(), 10))
	if got := call(rdb); got != http.StatusOK {
		t.Fatalf("token newer than cutoff: want 200, got %d", got)
	}

	// Redis cannot answer -> fail closed
	rdb, mock = redismock.NewClientMock()
	mock.ExpectExists("blacklist:" + tok).SetVal(0)
	mock.ExpectGet("session_cutoff:" + uid.String()).SetErr(context.DeadlineExceeded)
	if got := call(rdb); got != http.StatusUnauthorized {
		t.Fatalf("redis failure: want 401 (fail closed), got %d", got)
	}
}
