package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/sms"
	"github.com/maidulcu/masaar-crm/internal/testdb"
)

type authEnv struct {
	app   *fiber.App
	pool  *pgxpool.Pool
	rdb   *redis.Client
	users *repo.UserRepo
	tag   string // unique per test so parallel packages never collide
}

// newAuthEnv needs Postgres (TEST_DATABASE_URL) and Redis (TEST_REDIS_ADDR).
func newAuthEnv(t *testing.T) *authEnv {
	t.Helper()
	url, addr := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_REDIS_ADDR")
	if url == "" || addr == "" {
		t.Skip("TEST_DATABASE_URL / TEST_REDIS_ADDR not set")
	}
	testdb.Migrate(t, url)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	e := &authEnv{pool: pool, rdb: rdb, users: repo.NewUserRepo(pool), tag: strings.ReplaceAll(uuid.NewString()[:8], "-", "")}

	cfg := &config.Config{
		JWTSecret: "test-secret-test-secret-test-secret", JWTAccessExpiryMin: 15, JWTRefreshExpiryDays: 7,
		AllowRegistration: true, TrialDurationDays: 14, MagicLinkExpiryMin: 15, MagicLinkRateLimit: 3,
		AppURL: "http://localhost:3000",
	}
	h := NewAuthHandler(e.users, repo.NewCompanyRepo(pool), rdb, cfg, repo.NewAuditLogRepo(pool), nil, sms.NewClient("k", "t", "s"))
	app := fiber.New()
	app.Post("/login", h.Login)
	app.Post("/register", h.Register)
	app.Post("/refresh", h.Refresh)
	app.Post("/forgot", h.ForgotPassword)
	app.Post("/reset", h.ResetPassword)
	app.Post("/magic/verify", h.VerifyMagicLink)
	app.Post("/sms/request", h.RequestSMSOTP)
	e.app = app

	t.Cleanup(func() {
		c := context.Background()
		like := "%" + e.tag + "%"
		_, _ = pool.Exec(c, `DELETE FROM password_reset_tokens WHERE user_id IN (SELECT id FROM users WHERE email LIKE $1)`, like)
		_, _ = pool.Exec(c, `DELETE FROM audit_logs WHERE user_id IN (SELECT id FROM users WHERE email LIKE $1)`, like)
		_, _ = pool.Exec(c, `DELETE FROM users WHERE email LIKE $1`, like)
		_, _ = pool.Exec(c, `DELETE FROM company_settings WHERE company_id IN (SELECT id FROM companies WHERE subdomain LIKE $1)`, like)
		_, _ = pool.Exec(c, `DELETE FROM companies WHERE subdomain LIKE $1`, like)
		keys, _ := rdb.Keys(c, "*"+e.tag+"*").Result()
		if len(keys) > 0 {
			rdb.Del(c, keys...)
		}
		rdb.Close()
		pool.Close()
	})
	return e
}

func (e *authEnv) email(name string) string { return fmt.Sprintf("%s-%s@example.com", name, e.tag) }
func (e *authEnv) sub(name string) string   { return fmt.Sprintf("%s-%s", name, e.tag) }

func (e *authEnv) post(t *testing.T, path string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func (e *authEnv) register(t *testing.T, name, email, password, sub string) (int, map[string]any) {
	return e.post(t, "/register", fiber.Map{"name": name, "email": email, "password": password, "company_name": "Co " + name, "subdomain": sub})
}

func (e *authEnv) count(t *testing.T, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRegister_NormalisesEmailAndSubdomain(t *testing.T) {
	e := newAuthEnv(t)
	mixed := strings.ToUpper(e.email("owner")[:1]) + e.email("owner")[1:] // capitalised local part
	code, out := e.register(t, "Owner", "  "+mixed+" ", "Password1", strings.ToUpper(e.sub("acme")))
	if code != 201 {
		t.Fatalf("register = %d %v", code, out)
	}
	if got := out["user"].(map[string]any)["email"]; got != strings.ToLower(mixed) {
		t.Errorf("stored email = %v, want lower-cased", got)
	}
	if got := out["company"].(map[string]any)["subdomain"]; got != e.sub("acme") {
		t.Errorf("stored subdomain = %v, want lower-cased %s", got, e.sub("acme"))
	}
	// and the account can actually sign in with any capitalisation
	if code, _ := e.post(t, "/login", fiber.Map{"email": mixed, "password": "Password1"}); code != 200 {
		t.Fatalf("login with registered email = %d", code)
	}
}

func TestRegister_RejectsBadInputWithoutServerError(t *testing.T) {
	e := newAuthEnv(t)
	long := "A1" + strings.Repeat("x", 80) // 82 bytes: bcrypt refuses > 72
	cases := []struct {
		name, email, pass, sub string
	}{
		{"not an email", "nope", "Password1", e.sub("a")},
		{"display-name email", "Joe <joe@example.com>", "Password1", e.sub("b")},
		{"password too long", e.email("long"), long, e.sub("c")},
		{"subdomain with spaces", e.email("sp"), "Password1", "my company"},
		{"subdomain too short", e.email("sh"), "Password1", "ab"},
		{"subdomain leading hyphen", e.email("hy"), "Password1", "-" + e.sub("d")},
	}
	for _, tc := range cases {
		if code, out := e.register(t, "N", tc.email, tc.pass, tc.sub); code != 400 {
			t.Errorf("%s: status %d (%v), want 400", tc.name, code, out)
		}
	}
	if code, out := e.register(t, "N", e.email("www"), "Password1", "www"); code != 409 {
		t.Errorf("reserved subdomain: status %d (%v), want 409", code, out)
	}
	if n := e.count(t, `SELECT count(*) FROM companies WHERE subdomain LIKE $1`, "%"+e.tag+"%"); n != 0 {
		t.Errorf("%d companies created by rejected sign-ups", n)
	}
}

// Taken email / subdomain must be a 409 and leave nothing behind (previously the company was
// created first, then the user insert failed with a 500 and an orphan company remained).
func TestRegister_ConflictsAreCleanAndAtomic(t *testing.T) {
	e := newAuthEnv(t)
	if code, out := e.register(t, "First", e.email("dup"), "Password1", e.sub("one")); code != 201 {
		t.Fatalf("first register = %d %v", code, out)
	}
	upper := strings.ToUpper(e.email("dup"))
	code, out := e.register(t, "Second", upper, "Password1", e.sub("two"))
	if code != 409 || out["error"] != "email_taken" {
		t.Fatalf("duplicate email (other case) = %d %v, want 409 email_taken", code, out)
	}
	if n := e.count(t, `SELECT count(*) FROM companies WHERE subdomain = $1`, e.sub("two")); n != 0 {
		t.Errorf("orphan company left after failed sign-up: %d", n)
	}
	code, out = e.register(t, "Third", e.email("other"), "Password1", strings.ToUpper(e.sub("one")))
	if code != 409 || out["error"] != "subdomain_taken" {
		t.Fatalf("duplicate subdomain = %d %v, want 409 subdomain_taken", code, out)
	}
}

func TestRegister_ConcurrentSameEmailCreatesOneAccount(t *testing.T) {
	e := newAuthEnv(t)
	var wg sync.WaitGroup
	codes := make([]int, 6)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], _ = e.register(t, "Racer", e.email("race"), "Password1", e.sub(fmt.Sprintf("race%d", i)))
		}(i)
	}
	wg.Wait()
	created := 0
	for _, c := range codes {
		switch c {
		case 201:
			created++
		case 409:
		default:
			t.Errorf("unexpected status %d in %v", c, codes)
		}
	}
	if created != 1 {
		t.Fatalf("%d concurrent sign-ups succeeded, want 1 (%v)", created, codes)
	}
	if n := e.count(t, `SELECT count(*) FROM companies WHERE subdomain LIKE $1`, "race%"+e.tag); n != 1 {
		t.Errorf("%d companies for one account (orphans from the losing requests)", n)
	}
}

func TestRegister_ClosedRegistrationRefusesWhenUsersExist(t *testing.T) {
	e := newAuthEnv(t)
	if code, _ := e.register(t, "A", e.email("closed"), "Password1", e.sub("closed")); code != 201 {
		t.Fatal("setup register failed")
	}
	_, _, err := repo.NewCompanyRepo(e.pool).RegisterTenant(context.Background(), repo.RegistrationInput{
		CompanyName: "X", Subdomain: e.sub("late"), TrialDurationDays: 1, AdminName: "X",
		AdminEmail: e.email("late"), AdminPasswordHash: "x", OpenRegistration: false,
	})
	if err != repo.ErrRegistrationClosed {
		t.Fatalf("closed registration with existing users: %v, want ErrRegistrationClosed", err)
	}
	if n := e.count(t, `SELECT count(*) FROM companies WHERE subdomain = $1`, e.sub("late")); n != 0 {
		t.Error("company created although registration is closed")
	}
}

// The "account locked" answer used to appear only for registered addresses, so five bad logins
// told an attacker whether an email had an account.
func TestLogin_LockoutIsIndistinguishableForUnknownEmail(t *testing.T) {
	e := newAuthEnv(t)
	if code, _ := e.register(t, "Known", e.email("known"), "Password1", e.sub("lock")); code != 201 {
		t.Fatal("setup failed")
	}
	seq := func(email string) []int {
		var out []int
		for i := 0; i < 7; i++ {
			code, _ := e.post(t, "/login", fiber.Map{"email": email, "password": "Wrong-pass1"})
			out = append(out, code)
		}
		return out
	}
	known, unknown := seq(e.email("known")), seq(e.email("nobody"))
	if fmt.Sprint(known) != fmt.Sprint(unknown) {
		t.Fatalf("status sequence differs: registered %v vs unregistered %v", known, unknown)
	}
	if known[4] != 429 {
		t.Fatalf("5th failure should lock the account, got %v", known)
	}
}

func TestResetPassword_AcceptsNewPasswordAndClearsLockout(t *testing.T) {
	e := newAuthEnv(t)
	email := e.email("reset")
	if code, _ := e.register(t, "Reset", email, "Password1", e.sub("reset")); code != 201 {
		t.Fatal("setup failed")
	}
	for i := 0; i < 5; i++ { // lock the account
		e.post(t, "/login", fiber.Map{"email": email, "password": "Wrong-pass1"})
	}
	if code, _ := e.post(t, "/login", fiber.Map{"email": email, "password": "Password1"}); code != 429 {
		t.Fatalf("expected the account to be locked, got %d", code)
	}

	u, err := e.users.FindByEmail(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	token, err := e.users.CreatePasswordResetToken(context.Background(), u.ID)
	if err != nil {
		t.Fatal(err)
	}
	// The web app sends new_password; the field it used to send ("password") must not work.
	if code, _ := e.post(t, "/reset", fiber.Map{"token": token, "password": "NewPassword1"}); code != 400 {
		t.Fatalf("reset with the wrong field name = %d, want 400", code)
	}
	if code, out := e.post(t, "/reset", fiber.Map{"token": token, "new_password": "NewPassword1"}); code != 204 && code != 200 {
		t.Fatalf("reset = %d %v", code, out)
	}
	if code, out := e.post(t, "/login", fiber.Map{"email": email, "password": "NewPassword1"}); code != 200 {
		t.Fatalf("login after reset = %d %v (lockout should be cleared)", code, out)
	}
}

func TestForgotPassword_PerEmailCap(t *testing.T) {
	e := newAuthEnv(t)
	email := e.email("bomb")
	if code, _ := e.register(t, "Bomb", email, "Password1", e.sub("bomb")); code != 201 {
		t.Fatal("setup failed")
	}
	u, _ := e.users.FindByEmail(context.Background(), email)
	var first string
	for i := 0; i < forgotPasswordPerHour+3; i++ {
		if code, _ := e.post(t, "/forgot", fiber.Map{"email": email}); code != 200 {
			t.Fatalf("forgot = %d, must always answer 200", code)
		}
		if i == 0 {
			_ = e.pool.QueryRow(context.Background(), `SELECT token_hash FROM password_reset_tokens WHERE user_id = $1`, u.ID).Scan(&first)
		}
	}
	// beyond the cap no new token is minted: the original link stays valid
	var current string
	_ = e.pool.QueryRow(context.Background(), `SELECT token_hash FROM password_reset_tokens WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`, u.ID).Scan(&current)
	if n := e.count(t, `SELECT count(*) FROM password_reset_tokens WHERE user_id = $1`, u.ID); n != 1 {
		t.Errorf("%d tokens", n)
	}
	if first == current {
		t.Error("expected the token to rotate within the cap")
	}
}

func TestRefreshAndMagicLinkReturnCompanyWithTrialDays(t *testing.T) {
	e := newAuthEnv(t)
	email := e.email("trial")
	code, reg := e.register(t, "Trial", email, "Password1", e.sub("trial"))
	if code != 201 {
		t.Fatalf("register = %d", code)
	}
	if d := reg["company"].(map[string]any)["days_remaining"].(float64); d < 13 {
		t.Fatalf("register days_remaining = %v", d)
	}

	// magic link
	tok := uuid.NewString()
	e.rdb.Set(context.Background(), "magic:"+tok, email+"|en", 0)
	code, out := e.post(t, "/magic/verify", fiber.Map{"token": tok})
	if code != 200 {
		t.Fatalf("magic verify = %d %v", code, out)
	}
	if d := out["company"].(map[string]any)["days_remaining"].(float64); d < 13 {
		t.Errorf("magic-link login reported days_remaining = %v (hard-coded 0 hid the trial banner)", d)
	}

	// refresh returns the company too
	code, out = e.post(t, "/refresh", fiber.Map{"refresh_token": out["refresh_token"]})
	if code != 200 {
		t.Fatalf("refresh = %d %v", code, out)
	}
	if c, ok := out["company"].(map[string]any); !ok || c["plan"] == nil {
		t.Errorf("refresh response has no company: %v", out)
	}

	// a suspended workspace cannot sign in through the passwordless route
	_, _ = e.pool.Exec(context.Background(), `UPDATE companies SET is_active = FALSE WHERE subdomain = $1`, e.sub("trial"))
	tok2 := uuid.NewString()
	e.rdb.Set(context.Background(), "magic:"+tok2, email+"|en", 0)
	if code, out := e.post(t, "/magic/verify", fiber.Map{"token": tok2}); code != 403 {
		t.Errorf("magic link for suspended company = %d %v, want 403", code, out)
	}
}

func TestSMSRequest_OnlyTextsRegisteredNumbers(t *testing.T) {
	e := newAuthEnv(t)
	phone := "+97150" + e.tag[:6] // not registered to anyone
	if code, out := e.post(t, "/sms/request", fiber.Map{"phone": phone}); code != 200 {
		t.Fatalf("request = %d %v, want the generic 200", code, out)
	}
	if n, _ := e.rdb.Exists(context.Background(), "sms_otp:"+phone).Result(); n != 0 {
		t.Fatal("an OTP was generated (and an SMS would be sent) for an unregistered number")
	}
	// the per-phone limit is atomic: 3 allowed, then 429
	var codes []int
	for i := 0; i < 5; i++ {
		c, _ := e.post(t, "/sms/request", fiber.Map{"phone": phone})
		codes = append(codes, c)
	}
	if fmt.Sprint(codes) != "[200 200 429 429 429]" { // one request already counted above
		t.Errorf("rate limit sequence = %v", codes)
	}
}
