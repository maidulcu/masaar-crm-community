package handler

// Integration test of two-factor sign-in against Postgres and Redis. Skipped unless
// TEST_DATABASE_URL and TEST_REDIS_ADDR are set.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/testdb"
	"github.com/maidulcu/masaar-crm/internal/totp"
)

func TestTwoFactorSignIn(t *testing.T) {
	url, addr := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_ADDR")
	if url == "" || addr == "" {
		t.Skip("TEST_DATABASE_URL and TEST_REDIS_ADDR not set")
	}
	testdb.Migrate(t, url)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { rdb.Close() })

	cid, alice, bob := uuid.New(), uuid.New(), uuid.New()
	aliceEmail, bobEmail := "alice-"+cid.String()[:8]+"@test.local", "bob-"+cid.String()[:8]+"@test.local"
	hash, _ := bcrypt.GenerateFromPassword([]byte("Passw0rd!x"), bcrypt.MinCost)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`INSERT INTO companies (id, name, subdomain) VALUES ($1, '2FA Co', $2)`, cid, "tf-"+cid.String()[:8])
	exec(`INSERT INTO company_settings (company_id, name, vat_number, business_address) VALUES ($1, '2FA Co', '', '')`, cid)
	exec(`INSERT INTO users (id, company_id, name, email, password_hash, role) VALUES ($1, $2, 'Alice', $3, $4, 'admin'), ($5, $2, 'Bob', $6, $4, 'agent')`,
		alice, cid, aliceEmail, string(hash), bob, bobEmail)
	t.Cleanup(func() {
		for _, q := range []string{`DELETE FROM audit_logs WHERE company_id = $1`, `DELETE FROM user_backup_codes WHERE user_id IN (SELECT id FROM users WHERE company_id = $1)`,
			`DELETE FROM users WHERE company_id = $1`, `DELETE FROM company_settings WHERE company_id = $1`, `DELETE FROM companies WHERE id = $1`} {
			_, _ = pool.Exec(ctx, q, cid)
		}
		rdb.Del(ctx, "lockout:"+aliceEmail, "lockout:"+bobEmail, "loginfail:"+aliceEmail, "loginfail:"+bobEmail)
	})

	box, _ := totp.NewBox("test key")
	cfg := &config.Config{JWTSecret: "test-secret-0123456789", JWTAccessExpiryMin: 15, JWTRefreshExpiryDays: 7}
	h := NewAuthHandler(repo.NewUserRepo(pool), repo.NewCompanyRepo(pool), rdb, cfg, repo.NewAuditLogRepo(pool), nil, nil).
		UseTwoFactor(repo.NewTwoFactorRepo(pool), box)

	app := fiber.New()
	app.Post("/login", h.Login)
	app.Post("/verify", h.VerifyTOTP)
	app.Post("/enroll", h.EnrollPending)
	app.Post("/enroll/confirm", h.ConfirmEnrollPending)
	as := func(uid uuid.UUID) fiber.Handler {
		return func(c *fiber.Ctx) error {
			c.Locals("user_id", uid)
			c.Locals("company_id", cid.String())
			return c.Next()
		}
	}
	app.Post("/me/setup", as(alice), h.TwoFactorSetup)
	app.Post("/me/enable", as(alice), h.TwoFactorEnable)
	app.Post("/me/disable", as(alice), h.TwoFactorDisable)
	app.Put("/security", as(alice), h.UpdateSecuritySettings)

	call := func(path string, body any) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", path, bytes.NewReader(raw))
		if path == "/security" {
			req.Method = "PUT"
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, 10_000)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	login := func(email string) map[string]any {
		t.Helper()
		code, out := call("/login", map[string]string{"email": email, "password": "Passw0rd!x"})
		if code != 200 {
			t.Fatalf("login %s: %d %v", email, code, out)
		}
		return out
	}

	// Without two-factor: a session straight away.
	if out := login(aliceEmail); out["access_token"] == nil || out["two_factor_required"] != nil {
		t.Fatalf("plain login = %v", out)
	}

	// Alice turns it on.
	_, setup := call("/me/setup", nil)
	secret := setup["secret"].(string)
	if !strings.HasPrefix(setup["otpauth_url"].(string), "otpauth://totp/") {
		t.Fatalf("setup = %v", setup)
	}
	if code, _ := call("/me/enable", map[string]string{"code": "000000"}); code != 401 {
		t.Fatalf("enable with a wrong code: %d", code)
	}
	now := time.Now()
	good, _ := totp.CodeAt(secret, totp.Step(now))
	code, enabled := call("/me/enable", map[string]string{"code": good})
	if code != 200 || len(enabled["backup_codes"].([]any)) != 8 {
		t.Fatalf("enable: %d %v", code, enabled)
	}
	backup := enabled["backup_codes"].([]any)[0].(string)

	// Now the password alone gives no session.
	out := login(aliceEmail)
	pending, _ := out["pending_token"].(string)
	if out["access_token"] != nil || out["two_factor_required"] != true || out["setup_required"] != false || pending == "" {
		t.Fatalf("login with 2FA = %v", out)
	}
	// The code that enabled 2FA cannot be replayed.
	if code, _ := call("/verify", map[string]string{"pending_token": pending, "code": good}); code != 401 {
		t.Fatalf("replayed code: %d", code)
	}
	if code, _ := call("/verify", map[string]string{"pending_token": "nonsense", "code": good}); code != 401 {
		t.Fatalf("bad pending token: %d", code)
	}
	code, sess := call("/verify", map[string]string{"pending_token": pending, "code": backup})
	if code != 200 || sess["access_token"] == nil || sess["backup_codes_left"] != float64(7) {
		t.Fatalf("backup code: %d %v", code, sess)
	}
	if code, _ := call("/verify", map[string]string{"pending_token": pending, "code": backup}); code != 401 {
		t.Fatalf("pending token reused: %d", code)
	}
	pending = login(aliceEmail)["pending_token"].(string)
	if code, _ := call("/verify", map[string]string{"pending_token": pending, "code": backup}); code != 401 {
		t.Fatalf("backup code reused: %d", code)
	}
	next, _ := totp.CodeAt(secret, totp.Step(now)+1)
	if code, sess := call("/verify", map[string]string{"pending_token": pending, "code": next}); code != 200 || sess["access_token"] == nil {
		t.Fatalf("app code: %d %v", code, sess)
	}

	// Disabling needs the password and a code.
	if code, _ := call("/me/disable", map[string]string{"password": "wrong", "code": next}); code != 401 {
		t.Fatalf("disable with a wrong password: %d", code)
	}

	// The company requires two-factor: Bob, without it, enrols during sign-in.
	if code, _ := call("/security", map[string]bool{"require_2fa": true}); code != 204 {
		t.Fatalf("require: %d", code)
	}
	out = login(bobEmail)
	if out["setup_required"] != true || out["access_token"] != nil {
		t.Fatalf("Bob's login = %v", out)
	}
	bobPending := out["pending_token"].(string)
	if code, _ := call("/verify", map[string]string{"pending_token": bobPending, "code": "123456"}); code != 409 {
		t.Fatalf("verify before set-up: %d", code)
	}
	_, enrol := call("/enroll", map[string]string{"pending_token": bobPending})
	bobCode, _ := totp.CodeAt(enrol["secret"].(string), totp.Step(time.Now()))
	code, bobSess := call("/enroll/confirm", map[string]string{"pending_token": bobPending, "code": bobCode})
	if code != 200 || bobSess["access_token"] == nil || len(bobSess["backup_codes"].([]any)) != 8 {
		t.Fatalf("enrol: %d %v", code, bobSess)
	}
	if code, _ := call("/me/disable", map[string]string{"password": "Passw0rd!x", "code": next}); code != 403 {
		t.Fatalf("disable while required: %d", code)
	}

	// Five wrong codes lock the account, for the password too.
	pending = login(bobEmail)["pending_token"].(string)
	var last int
	for i := 0; i < 5; i++ {
		last, _ = call("/verify", map[string]string{"pending_token": pending, "code": "000000"})
	}
	if last != 429 {
		t.Fatalf("fifth wrong code: %d", last)
	}
	if code, _ := call("/login", map[string]string{"email": bobEmail, "password": "Passw0rd!x"}); code != 429 {
		t.Fatalf("password login while locked: %d", code)
	}
}
