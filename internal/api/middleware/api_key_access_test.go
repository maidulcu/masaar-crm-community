package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type fakeKeys map[string]*repo.ApiKey // plaintext -> key

func (f fakeKeys) ValidateKey(_ context.Context, hash string) (*repo.ApiKey, error) {
	for plain, k := range f {
		sum := sha256.Sum256([]byte(plain))
		if hex.EncodeToString(sum[:]) == hash {
			return k, nil
		}
	}
	return nil, errors.New("not found")
}

var testRoutes = []APIKeyRoute{
	{Method: "GET", Path: "/api/v1/leads/:id", Scope: "leads:read"},
	{Method: "PATCH", Path: "/api/v1/leads/:id/stage", Scope: "leads:write"},
	{Method: "GET", Path: "/api/v1/contacts/:id", Scope: "contacts:read"},
	{Method: "POST", Path: "/api/v1/contacts", Scope: "contacts:write"},
}

func TestScopeFor(t *testing.T) {
	table := newAPIKeyRouteTable(testRoutes)
	id := uuid.NewString()
	cases := []struct {
		method, path, scope string
		open                bool
	}{
		{"GET", "/api/v1/leads/" + id, "leads:read", true},
		{"HEAD", "/api/v1/leads/" + id, "leads:read", true},
		{"GET", "/api/v1/leads/" + id + "/", "leads:read", true},
		{"GET", "/API/V1/Leads/" + strings.ToUpper(id), "leads:read", true},
		{"PATCH", "/api/v1/leads/" + id + "/stage", "leads:write", true},
		{"DELETE", "/api/v1/leads/" + id, "", false},
		{"GET", "/api/v1/contacts/duplicates", "", false}, // :id only matches a UUID
		{"GET", "/api/v1/contacts/%64uplicates", "", false},
		{"GET", "/api/v1/leads", "", false},
		{"GET", "/api/v1//leads/" + id, "", false},
		{"POST", "/api/v1/contacts", "contacts:write", true},
	}
	for _, c := range cases {
		scope, open := table.scopeFor(c.method, c.path)
		if scope != c.scope || open != c.open {
			t.Errorf("%s %s = (%q, %v), want (%q, %v)", c.method, c.path, scope, open, c.scope, c.open)
		}
	}
}

// newGatedApp mirrors the /api/v1 chain: API keys are checked, session middleware is skipped
// for them, and a route behind RequireRole answers.
func newGatedApp(keys fakeKeys) *fiber.App {
	app := fiber.New()
	sessionRan := func(c *fiber.Ctx) error {
		c.Set("X-Session-Chain", "ran")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	v1 := app.Group("/api/v1", ProblemDetails(), APIKeyAccess(keys, testRoutes), UnlessAPIKey(sessionRan))
	ok := func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"company": c.Locals("company_id")})
	}
	v1.Get("/leads/:id", ok)
	v1.Delete("/leads/:id", RequireRole(domain.RoleAdmin), ok)
	v1.Get("/contacts/duplicates", RequireRole(domain.RoleAdmin), ok)
	v1.Post("/contacts", RequireRole(domain.RoleAdmin, domain.RoleAgent), ok)
	v1.Get("/contacts/:id", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "contact not found", "hint": "x"})
	})
	return app
}

func call(t *testing.T, app *fiber.App, method, path, key string) (*httptestResult, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	return &httptestResult{status: resp.StatusCode, ctype: resp.Header.Get("Content-Type"),
		wwwAuth: resp.Header.Get("WWW-Authenticate"), session: resp.Header.Get("X-Session-Chain")}, body
}

type httptestResult struct {
	status                  int
	ctype, wwwAuth, session string
}

func TestAPIKeyAccess(t *testing.T) {
	company := uuid.New()
	keys := fakeKeys{
		"sk_live_reader":    {ID: uuid.New(), CompanyID: company, Scopes: "leads:read,contacts:read", CompanyActive: true},
		"sk_live_writer":    {ID: uuid.New(), CompanyID: company, Scopes: "contacts:write", CompanyActive: true},
		"sk_live_suspended": {ID: uuid.New(), CompanyID: company, Scopes: "leads:read", CompanyActive: false},
		"sk_live_demo":      {ID: uuid.New(), CompanyID: company, Scopes: "contacts:read,contacts:write", CompanyActive: true, CompanyDemo: true},
	}
	app := newGatedApp(keys)
	lead := "/api/v1/leads/" + uuid.NewString()

	r, body := call(t, app, "GET", lead, "sk_live_reader")
	if r.status != 200 || body["company"] != company.String() || r.session != "" {
		t.Fatalf("reader GET lead: %+v %v", r, body)
	}

	// Scope missing: 403 with an RFC 6750 challenge naming the scope.
	r, body = call(t, app, "POST", "/api/v1/contacts", "sk_live_reader")
	if r.status != 403 || body["error"] != "insufficient_scope" || !strings.Contains(r.wwwAuth, `scope="contacts:write"`) {
		t.Fatalf("reader POST contacts: %+v %v", r, body)
	}

	// Scope held on a role-gated route: RequireRole lets the key through.
	if r, _ = call(t, app, "POST", "/api/v1/contacts", "sk_live_writer"); r.status != 200 {
		t.Fatalf("writer POST contacts: %+v", r)
	}

	// Routes not listed stay closed, even with a matching resource scope.
	for _, path := range []string{"/api/v1/contacts/duplicates"} {
		if r, _ = call(t, app, "GET", path, "sk_live_reader"); r.status != 403 {
			t.Fatalf("GET %s: %+v", path, r)
		}
	}
	if r, _ = call(t, app, "DELETE", lead, "sk_live_reader"); r.status != 403 {
		t.Fatalf("DELETE lead: %+v", r)
	}

	// Unknown key and suspended company.
	if r, _ = call(t, app, "GET", lead, "sk_live_nope"); r.status != 401 || !strings.Contains(r.wwwAuth, "invalid_token") {
		t.Fatalf("unknown key: %+v", r)
	}
	if r, _ = call(t, app, "GET", lead, "sk_live_suspended"); r.status != 403 {
		t.Fatalf("suspended: %+v", r)
	}

	// The demo company's keys are read-only.
	if r, body = call(t, app, "POST", "/api/v1/contacts", "sk_live_demo"); r.status != 403 || body["error"] != "demo_account_read_only" {
		t.Fatalf("demo write: %+v %v", r, body)
	}

	// No API key: the session chain handles it, error body unchanged.
	r, body = call(t, app, "GET", lead, "")
	if r.session != "ran" || r.status != 401 || body["type"] != nil {
		t.Fatalf("no key: %+v %v", r, body)
	}
}

func TestProblemDetails(t *testing.T) {
	keys := fakeKeys{"sk_live_reader": {ID: uuid.New(), CompanyID: uuid.New(), Scopes: "contacts:read", CompanyActive: true}}
	app := newGatedApp(keys)
	r, body := call(t, app, "GET", "/api/v1/contacts/"+uuid.NewString(), "sk_live_reader")
	if r.status != 404 || r.ctype != "application/problem+json" {
		t.Fatalf("got %+v", r)
	}
	want := map[string]any{"type": "about:blank", "title": "Not Found", "status": float64(404),
		"detail": "contact not found", "error": "contact not found", "hint": "x"}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("%s = %v, want %v", k, body[k], v)
		}
	}
	// Errors written by the gate itself are converted too.
	if r, body = call(t, app, "GET", "/api/v1/leads/"+uuid.NewString(), "sk_live_bad"); r.ctype != "application/problem+json" || body["title"] != "Unauthorized" {
		t.Fatalf("gate error: %+v %v", r, body)
	}
}

type memIdem struct {
	mu   sync.Mutex
	recs map[string][]byte
}

func (m *memIdem) Reserve(_ context.Context, key, fp string, _ time.Duration) (*IdempotentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if raw, ok := m.recs[key]; ok {
		return decideIdempotent(raw, fp)
	}
	m.recs[key], _ = json.Marshal(IdempotentResponse{Fingerprint: fp})
	return nil, nil
}

func (m *memIdem) Complete(_ context.Context, key string, r IdempotentResponse, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recs[key], _ = json.Marshal(r)
	return nil
}

func (m *memIdem) Release(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.recs, key)
	return nil
}

func TestIdempotency(t *testing.T) {
	store := &memIdem{recs: map[string][]byte{}}
	keyID := uuid.New()
	created, failNext, forbidNext := 0, false, false
	app := fiber.New()
	app.Use(recover.New()) // as in production: a panic becomes a 500
	app.Use(func(c *fiber.Ctx) error {
		if c.Get("X-Key") != "" {
			c.Locals("api_key_id", keyID)
		}
		return c.Next()
	})
	app.Use(Idempotency(store))
	app.Post("/things", func(c *fiber.Ctx) error {
		if c.Query("panic") != "" {
			panic("boom")
		}
		if failNext {
			failNext = false
			return c.Status(500).JSON(fiber.Map{"error": "boom"})
		}
		if forbidNext {
			forbidNext = false
			return c.Status(403).JSON(fiber.Map{"error": "insufficient_scope"})
		}
		created++
		return c.Status(201).JSON(fiber.Map{"n": created})
	})

	post := func(idem, body string, withKey bool) (int, string, string) {
		req := httptest.NewRequest("POST", "/things", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if idem != "" {
			req.Header.Set("Idempotency-Key", idem)
		}
		if withKey {
			req.Header.Set("X-Key", "1")
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(raw), resp.Header.Get("Idempotent-Replayed")
	}

	s1, b1, _ := post("abc", `{"a":1}`, true)
	s2, b2, replayed := post("abc", `{"a":1}`, true)
	if s1 != 201 || s2 != 201 || b1 != b2 || replayed != "true" || created != 1 {
		t.Fatalf("replay: %d %s / %d %s %q, created=%d", s1, b1, s2, b2, replayed, created)
	}
	if s, _, _ := post("abc", `{"a":2}`, true); s != 422 {
		t.Fatalf("mismatch: %d", s)
	}
	if s, _, _ := post("bad\x01key", `{}`, true); s != 400 {
		t.Fatalf("invalid key: %d", s)
	}

	// A 5xx frees the key so the retry runs.
	failNext = true
	if s, _, _ := post("retry", `{}`, true); s != 500 {
		t.Fatalf("first attempt: %d", s)
	}
	if s, _, r := post("retry", `{}`, true); s != 201 || r != "" {
		t.Fatalf("retry after 500: %d %q", s, r)
	}

	// A permission refusal is not kept: once the client has the scope, the retry runs.
	forbidNext = true
	if s, _, _ := post("scoped", `{}`, true); s != 403 {
		t.Fatalf("first attempt: %d", s)
	}
	if s, _, r := post("scoped", `{}`, true); s != 201 || r != "" {
		t.Fatalf("retry after 403: %d %q, want a fresh 201", s, r)
	}

	// A panic frees the key too.
	app2Req := httptest.NewRequest("POST", "/things?panic=1", strings.NewReader(`{}`))
	app2Req.Header.Set("Idempotency-Key", "panicky")
	app2Req.Header.Set("X-Key", "1")
	if resp, err := app.Test(app2Req); err != nil || resp.StatusCode != 500 {
		t.Fatalf("panic request: %v %v", resp, err)
	}
	if _, held := store.recs["idem:"+keyID.String()+":panicky"]; held {
		t.Fatal("key still held after a panic")
	}

	// Session requests and requests without the header are untouched.
	before := created
	post("abc", `{"a":1}`, false)
	post("", `{"a":1}`, true)
	if created != before+2 {
		t.Fatalf("untouched requests: created %d, want %d", created, before+2)
	}

	// A request still in progress gets 409.
	store.recs["idem:"+keyID.String()+":busy"], _ = json.Marshal(IdempotentResponse{Fingerprint: fingerprint("POST", "/things", `{}`)})
	if s, _, _ := post("busy", `{}`, true); s != 409 {
		t.Fatalf("in flight: %d", s)
	}
}

func fingerprint(method, path, body string) string {
	sum := sha256.Sum256([]byte(method + " " + path + "?\n" + body))
	return hex.EncodeToString(sum[:])
}

func BenchmarkScopeFor(b *testing.B) {
	table := newAPIKeyRouteTable(testRoutes)
	id := uuid.NewString()
	path := "/api/v1/leads/" + id
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		table.scopeFor("GET", path)
	}
}
