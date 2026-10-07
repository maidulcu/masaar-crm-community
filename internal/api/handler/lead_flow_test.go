package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/testdb"
	"github.com/maidulcu/masaar-crm/internal/webhook"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

type fakeDispatcher struct {
	mu     sync.Mutex
	events []string
}

func (f *fakeDispatcher) Dispatch(_ uuid.UUID, event string, _ interface{}) {
	f.mu.Lock()
	f.events = append(f.events, event)
	f.mu.Unlock()
}
func (f *fakeDispatcher) reset() { f.mu.Lock(); f.events = nil; f.mu.Unlock() }
func (f *fakeDispatcher) got() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.events, ",")
}

type noAudit struct{}

func (noAudit) Log(context.Context, uuid.UUID, string, string, uuid.UUID, any) {}

type leadEnv struct {
	pool          *pgxpool.Pool
	app           *fiber.App
	disp          *fakeDispatcher
	rotation      *LeadRotationHandler
	a, b          uuid.UUID // companies
	admin, agent1 uuid.UUID
	agent2        uuid.UUID
	viewer        uuid.UUID
	adminB        uuid.UUID
}

func newLeadEnv(t *testing.T) *leadEnv {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	testdb.Migrate(t, url)
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	e := &leadEnv{pool: pool, disp: &fakeDispatcher{}, a: uuid.New(), b: uuid.New()}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	for _, co := range []uuid.UUID{e.a, e.b} {
		exec(`INSERT INTO companies (id, name, subdomain) VALUES ($1::uuid, 'Lead test', 'lt-' || substr($1::uuid::text,1,8))`, co)
		exec(`INSERT INTO company_settings (company_id, name, vat_number, business_address) VALUES ($1, 'Lead test', '', '')`, co)
	}
	user := func(co uuid.UUID, role string, active bool) uuid.UUID {
		id := uuid.New()
		exec(`INSERT INTO users (id, company_id, name, email, password_hash, role, is_active) VALUES ($1,$2,$3,$4,'x',$5,$6)`,
			id, co, role, id.String()+"@lead.test", role, active)
		return id
	}
	e.admin, e.agent1, e.agent2, e.viewer = user(e.a, "admin", true), user(e.a, "agent", true), user(e.a, "agent", true), user(e.a, "viewer", true)
	e.adminB = user(e.b, "admin", true)
	stage := func(name string, order int, won, lost, def bool) {
		exec(`INSERT INTO pipeline_stages (company_id, entity_type, name, sort_order, is_won, is_lost, is_default) VALUES ($1,'lead',$2,$3,$4,$5,$6)`,
			e.a, name, order, won, lost, def)
	}
	stage("fresh", 0, false, false, true) // the company renamed "new" to "fresh"
	stage("talking", 1, false, false, false)
	stage("signed", 2, true, false, false) // custom won stage
	stage("dead", 3, false, true, false)   // custom lost stage

	t.Cleanup(func() {
		c := context.Background()
		for _, co := range []uuid.UUID{e.a, e.b} {
			for _, q := range []string{
				`DELETE FROM lead_rotation_settings WHERE company_id = $1`,
				`DELETE FROM audit_logs WHERE company_id = $1`,
				`DELETE FROM leads WHERE company_id = $1`,
				`DELETE FROM contacts WHERE company_id = $1`,
				`DELETE FROM pipeline_stages WHERE company_id = $1`,
				`DELETE FROM users WHERE company_id = $1`,
				`DELETE FROM company_settings WHERE company_id = $1`,
				`DELETE FROM companies WHERE id = $1`,
			} {
				_, _ = pool.Exec(c, q, co)
			}
		}
		pool.Close()
	})

	hub := ws.NewHub()
	leadRepo := repo.NewLeadRepo(pool)
	contactRepo := repo.NewContactRepo(pool)
	e.rotation = NewLeadRotationHandler(repo.NewLeadRotationRepo(pool), leadRepo, repo.NewUserRepo(pool), hub)
	lh := NewLeadHandler(leadRepo, contactRepo, nil, nil, repo.NewLeadTagRepo(pool), hub, noAudit{}, e.disp, repo.NewPipelineStageRepo(pool))
	lh.SetAssigner(e.rotation)
	pub := NewPublicLeadHandler(contactRepo, leadRepo, webhook.NewDispatcher(repo.NewWebhookRepo(pool)))
	pub.SetAssigner(e.rotation)
	imp := NewImportExportHandler(contactRepo, leadRepo, repo.NewListingRepo(pool))
	imp.SetAssigner(e.rotation)

	app := fiber.New()
	for prefix, co := range map[string]uuid.UUID{"/a": e.a, "/b": e.b} {
		co := co
		actor := e.admin
		if co == e.b {
			actor = e.adminB
		}
		g := app.Group(prefix, func(c *fiber.Ctx) error {
			c.Locals("company_id", co.String())
			c.Locals("user_id", actor)
			return c.Next()
		})
		g.Post("/leads", lh.Create)
		g.Patch("/leads/:id/stage", lh.UpdateStage)
		g.Patch("/leads/:id/notes", lh.UpdateNotes)
		g.Patch("/leads/:id/assign", lh.Assign)
		g.Post("/leads/:id/auto-assign", e.rotation.AutoAssign)
		g.Patch("/settings/lead-rotation", e.rotation.UpdateSettings)
		g.Get("/settings/lead-rotation", e.rotation.GetSettings)
		g.Post("/webhooks/leads", pub.SubmitLead)
		g.Post("/import/leads", imp.ImportLeads)
	}
	e.app = app
	return e
}

func (e *leadEnv) do(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
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

func (e *leadEnv) contact(t *testing.T, co uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	phone := fmt.Sprintf("+9715%08d", id.ID()%100000000)
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO contacts (id, company_id, phone_wa, full_name, email) VALUES ($1,$2,$3,'C','')`, id, co, phone); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *leadEnv) newLead(t *testing.T) string {
	t.Helper()
	code, out := e.do(t, "POST", "/a/leads", fiber.Map{"contact_id": e.contact(t, e.a)})
	if code != 201 {
		t.Fatalf("create lead = %d %v", code, out)
	}
	return out["id"].(string)
}

func (e *leadEnv) col(t *testing.T, id, col string) string {
	t.Helper()
	var v *string
	if err := e.pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT %s::text FROM leads WHERE id = $1`, col), id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v == nil {
		return ""
	}
	return *v
}

func TestLeadCreate_ValidatesAndAppliesDefaults(t *testing.T) {
	e := newLeadEnv(t)
	ct := e.contact(t, e.a)

	// Defaults: stage is the company's default stage ("fresh", not a hard-coded "new"), source
	// "web" (an empty source used to hit the CHECK constraint), currency upper-cased.
	code, out := e.do(t, "POST", "/a/leads", fiber.Map{"contact_id": ct, "currency": "usd", "deal_value": 1500.555})
	if code != 201 {
		t.Fatalf("create = %d %v", code, out)
	}
	if out["stage"] != "fresh" || out["source"] != "web" || out["currency"] != "USD" || out["deal_value"] != 1500.56 {
		t.Errorf("defaults not applied: %v", out)
	}

	bad := []struct {
		name string
		body fiber.Map
	}{
		{"currency too long", fiber.Map{"currency": "AEDX"}},
		{"currency digits", fiber.Map{"currency": "12$"}},
		{"negative value", fiber.Map{"deal_value": -5}},
		{"huge value", fiber.Map{"deal_value": 1e15}},
		{"bad source", fiber.Map{"source": "carrier-pigeon"}},
		{"stage not in pipeline", fiber.Map{"stage": "won"}}, // this company has no stage called "won"
		{"notes too long", fiber.Map{"notes": strings.Repeat("x", maxLeadNotesRunes+1)}},
	}
	for _, tc := range bad {
		tc.body["contact_id"] = ct
		if code, out := e.do(t, "POST", "/a/leads", tc.body); code != 422 {
			t.Errorf("%s: status %d (%v), want 422", tc.name, code, out)
		}
	}

	// another company's contact is not found, never a 500
	if code, _ := e.do(t, "POST", "/a/leads", fiber.Map{"contact_id": e.contact(t, e.b)}); code != 404 {
		t.Errorf("foreign contact = %d, want 404", code)
	}
}

func TestLeadUpdateStage_NoPhantomEventsAndClosedReason(t *testing.T) {
	e := newLeadEnv(t)
	id := e.newLead(t)
	e.disp.reset()

	// unknown / foreign / deleted leads are 404 and fire nothing
	if code, _ := e.do(t, "PATCH", "/a/leads/"+uuid.NewString()+"/stage", fiber.Map{"stage": "signed"}); code != 404 {
		t.Errorf("missing lead = %d, want 404", code)
	}
	if code, _ := e.do(t, "PATCH", "/b/leads/"+id+"/stage", fiber.Map{"stage": "signed"}); code != 404 && code != 400 {
		t.Errorf("foreign lead = %d, want 404/400", code)
	}
	if got := e.disp.got(); got != "" {
		t.Fatalf("events fired for leads that were not changed: %q", got)
	}
	if got := e.col(t, id, "stage"); got != "fresh" {
		t.Fatalf("stage changed to %q by a rejected request", got)
	}

	// a custom won stage (is_won) fires lead.won even though it is not named "won"
	if code, out := e.do(t, "PATCH", "/a/leads/"+id+"/stage", fiber.Map{"stage": "signed", "closed_reason": " paid in full "}); code != 200 {
		t.Fatalf("move = %d %v", code, out)
	}
	if got := e.disp.got(); got != "lead.stage_changed,lead.won" {
		t.Errorf("events = %q, want lead.stage_changed,lead.won", got)
	}
	if got := e.col(t, id, "closed_reason"); got != "paid in full" {
		t.Errorf("closed_reason = %q", got)
	}

	// repeating the same move is a no-op: no duplicate events, reason untouched
	e.disp.reset()
	if code, _ := e.do(t, "PATCH", "/a/leads/"+id+"/stage", fiber.Map{"stage": "signed", "closed_reason": "overwritten?"}); code != 200 {
		t.Fatal("repeat move failed")
	}
	if got := e.disp.got(); got != "" {
		t.Errorf("repeat move fired %q", got)
	}
	if got := e.col(t, id, "closed_reason"); got != "paid in full" {
		t.Errorf("repeat move rewrote closed_reason to %q", got)
	}

	// reopening clears the stale reason; a reason sent for an open stage is ignored
	if code, _ := e.do(t, "PATCH", "/a/leads/"+id+"/stage", fiber.Map{"stage": "talking", "closed_reason": "ignored"}); code != 200 {
		t.Fatal("reopen failed")
	}
	if got := e.col(t, id, "closed_reason"); got != "" {
		t.Errorf("closed_reason after reopening = %q, want empty", got)
	}
	if code, _ := e.do(t, "PATCH", "/a/leads/"+id+"/stage", fiber.Map{"stage": "nonsense"}); code != 400 {
		t.Errorf("unknown stage = %d, want 400", code)
	}
}

func TestLeadAssign_RejectsBadAssigneeAndMissingLead(t *testing.T) {
	e := newLeadEnv(t)
	id := e.newLead(t)

	if code, _ := e.do(t, "PATCH", "/a/leads/"+uuid.NewString()+"/assign", fiber.Map{"assigned_to": e.agent1}); code != 404 {
		t.Errorf("missing lead = %d, want 404", code)
	}
	for name, uid := range map[string]uuid.UUID{"viewer": e.viewer, "other company": e.adminB, "no such user": uuid.New()} {
		if code, out := e.do(t, "PATCH", "/a/leads/"+id+"/assign", fiber.Map{"assigned_to": uid}); code != 422 {
			t.Errorf("assign to %s = %d %v, want 422 (was a silent 200)", name, code, out)
		}
	}
	if got := e.col(t, id, "assigned_to"); got != "" {
		t.Fatalf("a rejected assignment changed assigned_to to %q", got)
	}
	// inactive users cannot take leads either
	_, _ = e.pool.Exec(context.Background(), `UPDATE users SET is_active = FALSE WHERE id = $1`, e.agent2)
	if code, _ := e.do(t, "PATCH", "/a/leads/"+id+"/assign", fiber.Map{"assigned_to": e.agent2}); code != 422 {
		t.Errorf("inactive assignee = %d, want 422", code)
	}
	_, _ = e.pool.Exec(context.Background(), `UPDATE users SET is_active = TRUE WHERE id = $1`, e.agent2)

	if code, _ := e.do(t, "PATCH", "/a/leads/"+id+"/assign", fiber.Map{"assigned_to": e.agent1}); code != 200 {
		t.Fatal("valid assignment failed")
	}
	if got := e.col(t, id, "assigned_to"); got != e.agent1.String() {
		t.Errorf("assigned_to = %q", got)
	}
	if code, _ := e.do(t, "PATCH", "/a/leads/"+id+"/assign", fiber.Map{"assigned_to": nil}); code != 200 || e.col(t, id, "assigned_to") != "" {
		t.Error("unassign failed")
	}
	if code, _ := e.do(t, "PATCH", "/a/leads/"+uuid.NewString()+"/notes", fiber.Map{"notes": "x"}); code != 404 {
		t.Errorf("notes on a missing lead = %d, want 404", code)
	}
}

func TestLeadRotation_PatchKeepsSettingsAndNewLeadsAreAssigned(t *testing.T) {
	e := newLeadEnv(t)
	// make agent1/agent2 the only assignable people so the rotation order is deterministic
	_, _ = e.pool.Exec(context.Background(), `UPDATE users SET role = 'viewer' WHERE id = $1`, e.admin)

	if code, out := e.do(t, "PATCH", "/a/settings/lead-rotation", fiber.Map{"mode": "round_robin", "enabled": true, "max_per_agent": 7}); code != 200 {
		t.Fatalf("enable = %d %v", code, out)
	}
	// A PATCH that only changes the mode must not disable rotation or zero the cap.
	if code, _ := e.do(t, "PATCH", "/a/settings/lead-rotation", fiber.Map{"mode": "capacity"}); code != 200 {
		t.Fatal("patch failed")
	}
	_, got := e.do(t, "GET", "/a/settings/lead-rotation", nil)
	if got["enabled"] != true || got["mode"] != "capacity" || got["max_per_agent"] != float64(7) {
		t.Fatalf("settings after partial PATCH = %v, want enabled/capacity/7", got)
	}
	if code, _ := e.do(t, "PATCH", "/a/settings/lead-rotation", fiber.Map{"max_per_agent": -1}); code != 422 {
		t.Errorf("negative cap = %d, want 422", code)
	}

	// Leads created with rotation on are assigned (AssignNewLead used to be dead code).
	e.do(t, "PATCH", "/a/settings/lead-rotation", fiber.Map{"mode": "round_robin"})
	seen := map[string]int{}
	for i := 0; i < 4; i++ {
		id := e.newLead(t)
		a := e.col(t, id, "assigned_to")
		if a == "" {
			t.Fatalf("lead %d was not auto-assigned", i)
		}
		seen[a]++
	}
	if len(seen) != 2 || seen[e.agent1.String()] != 2 || seen[e.agent2.String()] != 2 {
		t.Errorf("round robin distribution = %v, want 2 each for the two agents", seen)
	}

	// auto-assign for a lead that does not exist must not burn a rotation turn
	var before, after int
	_ = e.pool.QueryRow(context.Background(), `SELECT rotation_index FROM lead_rotation_settings WHERE company_id = $1`, e.a).Scan(&before)
	if code, _ := e.do(t, "POST", "/a/leads/"+uuid.NewString()+"/auto-assign", nil); code != 404 {
		t.Errorf("auto-assign missing lead = %d, want 404", code)
	}
	_ = e.pool.QueryRow(context.Background(), `SELECT rotation_index FROM lead_rotation_settings WHERE company_id = $1`, e.a).Scan(&after)
	if before != after {
		t.Errorf("rotation index moved %d -> %d for a missing lead", before, after)
	}

	// disabled rotation is reported as such, not as "upstream service unavailable"
	e.do(t, "PATCH", "/a/settings/lead-rotation", fiber.Map{"enabled": false})
	id := e.newLead(t)
	code, out := e.do(t, "POST", "/a/leads/"+id+"/auto-assign", nil)
	if code != 400 || fmt.Sprint(out["error"]) != "lead rotation is not enabled" {
		t.Errorf("auto-assign with rotation off = %d %v, want 400 'lead rotation is not enabled'", code, out)
	}
}

func TestPublicLead_DedupesAndDoesNotClobberContact(t *testing.T) {
	e := newLeadEnv(t)
	phone := "+971501110099"
	post := func(body fiber.Map) (int, map[string]any) {
		body["phone"] = phone
		return e.do(t, "POST", "/a/webhooks/leads", body)
	}

	code, first := post(fiber.Map{"name": "Sara", "email": "sara@example.com", "language": "en"})
	if code != 201 {
		t.Fatalf("first = %d %v", code, first)
	}
	// An agent later fixed the contact's details...
	_, _ = e.pool.Exec(context.Background(), `UPDATE contacts SET email = 'agent-set@example.com', language = 'en' WHERE company_id = $1 AND phone_wa = $2`, e.a, phone)

	// ...a retry of the same form returns the existing lead instead of creating another
	code, retry := post(fiber.Map{"name": "Sara", "email": "other@example.com"})
	if code != 200 || retry["duplicate"] != true || retry["lead_id"] != first["lead_id"] {
		t.Fatalf("retry = %d %v, want 200 duplicate of %v", code, retry, first["lead_id"])
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM leads WHERE company_id = $1`, e.a).Scan(&n)
	if n != 1 {
		t.Errorf("%d leads after a retry, want 1", n)
	}
	var email, lang string
	_ = e.pool.QueryRow(context.Background(), `SELECT email, language FROM contacts WHERE company_id = $1 AND phone_wa = $2`, e.a, phone).Scan(&email, &lang)
	if email != "agent-set@example.com" || lang != "en" {
		t.Errorf("contact overwritten by the integration: email=%q language=%q", email, lang)
	}

	for name, body := range map[string]fiber.Map{
		"bad currency": {"name": "X", "currency": "EURO"},
		"bad language": {"name": "X", "language": "klingon"},
		"bad email":    {"name": "X", "email": "nope"},
		"neg value":    {"name": "X", "deal_value": -1},
	} {
		if code, out := post(body); code != 422 {
			t.Errorf("%s = %d %v, want 422", name, code, out)
		}
	}
}

func TestImportLeads_RejectedRowsLeaveNoContact(t *testing.T) {
	e := newLeadEnv(t)
	csv := "contact_phone,stage,currency,deal_value\n" +
		"+971502220001,new,AEDX,100\n" + // bad currency
		"+971502220002,new,aed,abc\n" + // bad number
		"+971502220003,bogus,AED,1\n" + // bad stage
		"+971502220004,new,usd,250.5\n" // good
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "leads.csv")
	_, _ = fw.Write([]byte(csv))
	_ = mw.Close()
	req := httptest.NewRequest("POST", "/a/import/leads", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := e.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out["imported"] != float64(1) || out["skipped"] != float64(3) {
		t.Fatalf("import result = %v", out)
	}
	var n int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM contacts WHERE company_id = $1`, e.a).Scan(&n)
	if n != 1 {
		t.Errorf("%d contacts created for 3 rejected rows and 1 good row, want 1", n)
	}
	var cur string
	_ = e.pool.QueryRow(context.Background(), `SELECT currency FROM leads WHERE company_id = $1`, e.a).Scan(&cur)
	if cur != "USD" {
		t.Errorf("currency = %q, want USD", cur)
	}
}
