package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/apiscope"
	"github.com/maidulcu/masaar-crm/internal/partner"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/testdb"
)

// TestPartnerFlow drives the Partner API end to end against the database: a client-credentials
// token, company and user provisioning with seats, isolation between partners, suspension and
// the audit trail.
func TestPartnerFlow(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	testdb.Migrate(t, dbURL)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	run := strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	pa, pb := "pa-"+run, "pb-"+run
	t.Cleanup(func() {
		c := context.Background()
		for _, q := range []string{
			`DELETE FROM audit_logs WHERE company_id IN (SELECT id FROM companies WHERE external_source = ANY($1))`,
			`DELETE FROM api_keys WHERE company_id IN (SELECT id FROM companies WHERE external_source = ANY($1))`,
			`DELETE FROM users WHERE company_id IN (SELECT id FROM companies WHERE external_source = ANY($1))`,
			`DELETE FROM company_settings WHERE company_id IN (SELECT id FROM companies WHERE external_source = ANY($1))`,
			`DELETE FROM companies WHERE external_source = ANY($1)`,
			`DELETE FROM partner_tokens WHERE client_id = ANY($1)`,
		} {
			if _, err := pool.Exec(c, q, []string{pa, pb}); err != nil {
				t.Logf("cleanup: %v", err)
			}
		}
		pool.Close()
	})

	sum := sha256.Sum256([]byte("secret"))
	clients, err := partner.Parse(pa + ":" + hex.EncodeToString(sum[:]) + "," + pb + ":" + hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatal(err)
	}
	tokens, keys, audit := repo.NewPartnerTokenRepo(pool), repo.NewApiKeyRepo(pool), repo.NewAuditLogRepo(pool)
	oh := NewOAuthHandler(clients, tokens, "")
	ph := NewPartnerHandler(repo.NewPartnerRepo(pool), keys, audit, nil, "https://crm.example.test")

	app := fiber.New()
	app.Post("/oauth/token", oh.Token)
	pg := app.Group("/partner/v1", middleware.ProblemDetails(), middleware.PartnerAuth(tokens, clients))
	prov := middleware.RequirePartnerScope(apiscope.PartnerProvision)
	pg.Get("/companies/:ref", prov, ph.GetCompany)
	pg.Put("/companies/:ref", prov, ph.PutCompany)
	pg.Put("/companies/:ref/users/:user_ref", prov, ph.PutUser)
	pg.Post("/companies/:ref/api-keys", prov, ph.CreateAPIKey)
	pg.Post("/companies/:ref/users/:user_ref/login-link", middleware.RequirePartnerScope(apiscope.PartnerLogin), ph.LoginLink)

	token := func(client string) string {
		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(url.Values{
			"grant_type": {"client_credentials"}, "client_id": {client}, "client_secret": {"secret"}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		tok, _ := out["access_token"].(string)
		if tok == "" {
			t.Fatalf("token for %s: %v", client, out)
		}
		return tok
	}
	ta, tb := token(pa), token(pb)
	do := func(tok, method, path string, body any) (int, map[string]any) {
		t.Helper()
		var rdr io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		}
		req := httptest.NewRequest(method, path, rdr)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := io.ReadAll(resp.Body)
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		return resp.StatusCode, out
	}

	adminEmail := "admin-" + run + "@partner.test"
	code, out := do(ta, "PUT", "/partner/v1/companies/agency-1", fiber.Map{
		"name": "Agency One", "plan": "pro", "max_users": 2,
		"admin": fiber.Map{"ref": "u-admin", "name": "Admin", "email": adminEmail},
	})
	if code != 201 {
		t.Fatalf("create company = %d %v", code, out)
	}
	leadKey, _ := out["lead_api_key"].(string)
	company := out["company"].(map[string]any)
	if !strings.HasPrefix(leadKey, "sk_live_") || company["plan"] != "pro" || company["active_users"] != float64(1) {
		t.Fatalf("created = %v", out)
	}
	var onTrial bool
	if err := pool.QueryRow(ctx, `SELECT on_trial FROM companies WHERE external_source = $1 AND external_ref = 'agency-1'`, pa).Scan(&onTrial); err != nil || onTrial {
		t.Errorf("partner company on trial = %v, %v", onTrial, err)
	}

	// Repeating the call updates instead of creating, and returns no new key.
	if code, out = do(ta, "PUT", "/partner/v1/companies/agency-1", fiber.Map{"name": "Agency One LLC"}); code != 200 || out["lead_api_key"] != nil ||
		out["company"].(map[string]any)["name"] != "Agency One LLC" {
		t.Fatalf("update company = %d %v", code, out)
	}

	// Seats: admin + one agent fill max_users 2.
	if code, out = do(ta, "PUT", "/partner/v1/companies/agency-1/users/u-1", fiber.Map{"name": "Agent 1", "email": "a1-" + run + "@partner.test"}); code != 201 {
		t.Fatalf("user 1 = %d %v", code, out)
	}
	if code, out = do(ta, "PUT", "/partner/v1/companies/agency-1/users/u-2", fiber.Map{"name": "Agent 2", "email": "a2-" + run + "@partner.test"}); code != 409 || out["error"] != "seat_limit" {
		t.Fatalf("user over seats = %d %v", code, out)
	}
	if code, _ = do(ta, "PUT", "/partner/v1/companies/agency-1/users/u-1", fiber.Map{"name": "Agent 1", "active": false}); code != 200 {
		t.Fatalf("deactivate user 1 = %d", code)
	}
	if code, out = do(ta, "PUT", "/partner/v1/companies/agency-1/users/u-2", fiber.Map{"name": "Agent 2", "email": "a2-" + run + "@partner.test", "role": "viewer"}); code != 201 || out["user"].(map[string]any)["role"] != "viewer" {
		t.Fatalf("user 2 after freeing a seat = %d %v", code, out)
	}
	if code, out = do(ta, "PUT", "/partner/v1/companies/agency-1/users/u-1", fiber.Map{"name": "Agent 1", "active": true}); code != 409 {
		t.Fatalf("reactivating over seats = %d %v", code, out)
	}
	var companyID uuid.UUID
	_ = pool.QueryRow(ctx, `SELECT id FROM companies WHERE external_source = $1 AND external_ref = 'agency-1'`, pa).Scan(&companyID)
	if ok, err := repo.NewUserRepo(pool).SeatAvailable(ctx, companyID, uuid.Nil); err != nil || ok {
		t.Errorf("SeatAvailable on a full company = %v, %v", ok, err)
	}
	// The trigger holds the limit on any path, e.g. the admin screens inserting directly.
	_, err = pool.Exec(ctx, `INSERT INTO users (id, company_id, name, email, password_hash, role, is_active)
		VALUES ($1, $2, 'Direct', $3, 'x', 'agent', true)`, uuid.New(), companyID, "direct-"+run+"@partner.test")
	if !repo.IsSeatLimit(err) {
		t.Fatalf("direct insert over seats: %v, want seat_limit", err)
	}

	// An email that already has an account is refused, for companies and users.
	if code, out = do(ta, "PUT", "/partner/v1/companies/agency-2", fiber.Map{"name": "Two",
		"admin": fiber.Map{"ref": "x", "name": "X", "email": adminEmail}}); code != 409 || out["error"] != "email_taken" {
		t.Fatalf("duplicate admin email = %d %v", code, out)
	}

	// Another partner cannot see or change this company.
	if code, _ = do(tb, "GET", "/partner/v1/companies/agency-1", nil); code != 404 {
		t.Fatalf("other partner GET = %d, want 404", code)
	}
	if code, _ = do(tb, "PUT", "/partner/v1/companies/agency-1/users/u-9", fiber.Map{"name": "Z", "email": "z-" + run + "@partner.test"}); code != 404 {
		t.Fatalf("other partner PUT user = %d, want 404", code)
	}

	// Suspension: the company's lead key stops working.
	keyHash := sha256.Sum256([]byte(leadKey))
	if k, err := keys.ValidateKey(ctx, hex.EncodeToString(keyHash[:])); err != nil || !k.CompanyActive || k.Scopes != apiscope.LeadCreate {
		t.Fatalf("lead key before suspension: %+v %v", k, err)
	}
	if code, _ = do(ta, "PUT", "/partner/v1/companies/agency-1", fiber.Map{"active": false}); code != 200 {
		t.Fatalf("suspend = %d", code)
	}
	if k, err := keys.ValidateKey(ctx, hex.EncodeToString(keyHash[:])); err != nil || k.CompanyActive {
		t.Fatalf("lead key after suspension: %+v %v", k, err)
	}
	if code, out = do(ta, "POST", "/partner/v1/companies/agency-1/users/u-admin/login-link", nil); code != 403 {
		t.Fatalf("login link for a suspended company = %d %v", code, out)
	}
	if code, _ = do(ta, "PUT", "/partner/v1/companies/agency-1", fiber.Map{"active": true}); code != 200 {
		t.Fatalf("reactivate = %d", code)
	}
	if code, _ = do(ta, "POST", "/partner/v1/companies/agency-1/users/u-admin/login-link", nil); code != 503 {
		t.Fatalf("login link without Redis = %d, want 503", code)
	}

	// Problem details for partner errors, and a bad token.
	if code, out = do(ta, "GET", "/partner/v1/companies/nope", nil); code != 404 || out["title"] != "Not Found" {
		t.Fatalf("missing company = %d %v", code, out)
	}
	if code, _ = do("mpt_bogus", "GET", "/partner/v1/companies/agency-1", nil); code != 401 {
		t.Fatalf("bogus token = %d", code)
	}

	var audited int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE company_id = $1 AND partner_client_id = $2`, companyID, pa).Scan(&audited); err != nil || audited < 5 {
		t.Fatalf("partner audit rows = %d, %v", audited, err)
	}
}
