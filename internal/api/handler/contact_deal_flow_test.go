package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/repo"
)

// cdEnv reuses the lead test fixtures (two companies, users with each role, pipeline stages)
// and mounts the contact, deal and invoice handlers.
type cdEnv struct {
	*leadEnv
	app2 *fiber.App
}

func newCDEnv(t *testing.T) *cdEnv {
	t.Helper()
	le := newLeadEnv(t)
	ctx := context.Background()
	t.Cleanup(func() { // run before leadEnv's cleanup (LIFO), which does not know about these tables
		for _, co := range []uuid.UUID{le.a, le.b} {
			_, _ = le.pool.Exec(ctx, `DELETE FROM vat_invoices WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM deals WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM whatsapp_threads WHERE company_id = $1`, co)
		}
	})
	audit := repo.NewAuditLogRepo(le.pool)
	ch := NewContactHandler(repo.NewContactRepo(le.pool), audit)
	dh := NewDealHandler(repo.NewDealRepo(le.pool), repo.NewInvoiceRepo(le.pool), audit)
	ih := NewInvoiceHandler(repo.NewInvoiceRepo(le.pool), repo.NewDealRepo(le.pool), repo.NewCompanySettingsRepo(le.pool))

	app := fiber.New()
	for prefix, co := range map[string]uuid.UUID{"/a": le.a, "/b": le.b} {
		co := co
		actor := le.admin
		if co == le.b {
			actor = le.adminB
		}
		g := app.Group(prefix, func(c *fiber.Ctx) error {
			c.Locals("company_id", co.String())
			c.Locals("user_id", actor)
			return c.Next()
		})
		g.Post("/contacts", ch.Create)
		g.Patch("/contacts/:id", ch.Update)
		g.Delete("/contacts/:id", ch.Delete)
		g.Get("/deals", dh.List)
		g.Post("/deals", dh.Create)
		g.Patch("/deals/:id", dh.Update)
		g.Patch("/deals/:id/stage", dh.UpdateStage)
		g.Delete("/deals/:id", dh.Delete)
		g.Get("/deals/:id/invoices", dh.ListInvoices)
		g.Post("/invoices", ih.Create)
		g.Get("/invoices/:id", ih.Get)
		g.Get("/invoices", ih.List)
		g.Post("/invoices/:id/send", ih.Send)
		g.Patch("/invoices/:id/status", ih.UpdateStatus)
	}
	return &cdEnv{leadEnv: le, app2: app}
}

func (e *cdEnv) req(t *testing.T, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	r := httptest.NewRequest(method, path, rdr)
	r.Header.Set("Content-Type", "application/json")
	resp, err := e.app2.Test(r, -1)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func (e *cdEnv) q(t *testing.T, q string, args ...any) string {
	t.Helper()
	var v *string
	if err := e.pool.QueryRow(context.Background(), q, args...).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v == nil {
		return ""
	}
	return *v
}

// newLeadFor creates a lead (and contact) in company A and returns both ids.
func (e *cdEnv) newLeadFor(t *testing.T) (contactID uuid.UUID, leadID string) {
	t.Helper()
	contactID = e.contact(t, e.a)
	code, out := e.do(t, "POST", "/a/leads", fiber.Map{"contact_id": contactID})
	if code != 201 {
		t.Fatalf("create lead = %d %v", code, out)
	}
	return contactID, out["id"].(string)
}

func (e *cdEnv) newDeal(t *testing.T, leadID string, extra fiber.Map) string {
	t.Helper()
	body := fiber.Map{"lead_id": leadID, "title": "Deal"}
	for k, v := range extra {
		body[k] = v
	}
	code, out := e.req(t, "POST", "/a/deals", body)
	if code != 201 {
		t.Fatalf("create deal = %d %v", code, out)
	}
	return out["id"].(string)
}

func TestContactCreateUpdate_ValidationAndClearing(t *testing.T) {
	e := newCDEnv(t)

	code, out := e.req(t, "POST", "/a/contacts", fiber.Map{"id": uuid.NewString(), "phone_wa": "+971507770001", "full_name": "  Sara  ", "email": "sara@example.com", "language": "en"})
	if code != 201 || out["full_name"] != "Sara" {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)

	for name, body := range map[string]fiber.Map{
		"bad language": {"phone_wa": "+971507770002", "full_name": "X", "language": "english"},
		"bad email":    {"phone_wa": "+971507770003", "full_name": "X", "email": "not-an-email"},
		"score range":  {"phone_wa": "+971507770004", "full_name": "X", "lead_score": 101},
		"blank name":   {"phone_wa": "+971507770005", "full_name": "   "},
	} {
		if c, o := e.req(t, "POST", "/a/contacts", body); c != 422 && c != 400 {
			t.Errorf("%s: status %d (%v), want 4xx not 500", name, c, o)
		}
	}
	if c, _ := e.req(t, "POST", "/a/contacts", fiber.Map{"phone_wa": "+971507770001", "full_name": "Dup"}); c != 409 {
		t.Errorf("duplicate phone = %d, want 409", c)
	}

	// PATCH can clear the email and unassign the contact (both were silently ignored before)
	e.req(t, "PATCH", "/a/contacts/"+id, fiber.Map{"assigned_to": e.agent1})
	if got := e.q(t, `SELECT assigned_to::text FROM contacts WHERE id = $1`, id); got != e.agent1.String() {
		t.Fatalf("assigned_to = %q", got)
	}
	if c, o := e.req(t, "PATCH", "/a/contacts/"+id, fiber.Map{"email": "", "assigned_to": nil}); c != 200 {
		t.Fatalf("clear = %d %v", c, o)
	}
	if e.q(t, `SELECT COALESCE(email,'') FROM contacts WHERE id = $1`, id) != "" || e.q(t, `SELECT assigned_to::text FROM contacts WHERE id = $1`, id) != "" {
		t.Error("email / assignee were not cleared")
	}
	// an absent assigned_to leaves it alone
	e.req(t, "PATCH", "/a/contacts/"+id, fiber.Map{"assigned_to": e.agent2})
	e.req(t, "PATCH", "/a/contacts/"+id, fiber.Map{"full_name": "Sara K"})
	if got := e.q(t, `SELECT assigned_to::text FROM contacts WHERE id = $1`, id); got != e.agent2.String() {
		t.Errorf("an unrelated PATCH changed assigned_to to %q", got)
	}
	if c, _ := e.req(t, "PATCH", "/a/contacts/"+id, fiber.Map{"assigned_to": e.viewer}); c != 422 {
		t.Errorf("assign to a viewer = %d, want 422", c)
	}
	if c, _ := e.req(t, "PATCH", "/a/contacts/"+id, fiber.Map{"language": "fr"}); c != 422 {
		t.Errorf("language fr = %d, want 422", c)
	}
	// a legacy invalid email must not block an unrelated edit
	_, _ = e.pool.Exec(context.Background(), `UPDATE contacts SET email = 'legacy-bad' WHERE id = $1`, id)
	if c, o := e.req(t, "PATCH", "/a/contacts/"+id, fiber.Map{"full_name": "Sara Z"}); c != 200 {
		t.Errorf("unrelated edit blocked by legacy email: %d %v", c, o)
	}
}

func TestContactDelete_RefusesToCascadeSilently(t *testing.T) {
	e := newCDEnv(t)
	contactID, leadID := e.newLeadFor(t)
	dealID := e.newDeal(t, leadID, nil)

	if c, _ := e.req(t, "DELETE", "/a/contacts/"+uuid.NewString(), nil); c != 404 {
		t.Errorf("missing contact = %d, want 404", c)
	}
	if c, _ := e.req(t, "DELETE", "/b/contacts/"+contactID.String(), nil); c != 404 {
		t.Errorf("other company's contact = %d, want 404", c)
	}

	c, out := e.req(t, "DELETE", "/a/contacts/"+contactID.String(), nil)
	if c != 409 {
		t.Fatalf("delete with linked records = %d %v, want 409", c, out)
	}
	linked, _ := out["linked"].(map[string]any)
	if linked["leads"] != float64(1) || linked["deals"] != float64(1) {
		t.Errorf("linked = %v, want 1 lead and 1 deal", linked)
	}
	if e.q(t, `SELECT count(*)::text FROM leads WHERE contact_id = $1`, contactID) != "1" {
		t.Fatal("the refused delete removed the lead anyway")
	}

	// invoices make it undeletable even with force
	if c, o := e.req(t, "POST", "/a/invoices", fiber.Map{"deal_id": dealID, "subtotal": 100}); c != 201 {
		t.Fatalf("invoice = %d %v", c, o)
	}
	if c, _ := e.req(t, "DELETE", "/a/contacts/"+contactID.String()+"?force=true", nil); c != 409 {
		t.Errorf("force delete with invoices = %d, want 409", c)
	}
	_, _ = e.pool.Exec(context.Background(), `DELETE FROM vat_invoices WHERE company_id = $1`, e.a)

	if c, o := e.req(t, "DELETE", "/a/contacts/"+contactID.String()+"?force=true", nil); c != 204 {
		t.Fatalf("force delete = %d %v", c, o)
	}
	if e.q(t, `SELECT ((SELECT count(*) FROM leads WHERE id = $1) + (SELECT count(*) FROM deals WHERE id = $2))::text`, leadID, dealID) != "0" {
		t.Error("force delete left the lead/deal")
	}

	// a contact with nothing attached deletes without ceremony
	lone := e.contact(t, e.a)
	if c, _ := e.req(t, "DELETE", "/a/contacts/"+lone.String(), nil); c != 204 {
		t.Errorf("delete of a lone contact = %d, want 204", c)
	}
}

func TestDeal_CreateUpdateStageDelete(t *testing.T) {
	e := newCDEnv(t)
	_, leadID := e.newLeadFor(t)

	// 0% is honoured (it used to become 50), date-only close_date accepted, currency normalised
	code, out := e.req(t, "POST", "/a/deals", fiber.Map{"lead_id": leadID, "title": " Villa ", "probability": 0, "currency": "usd", "close_date": "2026-12-31", "amount": 1000.499})
	if code != 201 || out["probability"] != float64(0) || out["currency"] != "USD" || out["title"] != "Villa" || out["amount"] != 1000.5 {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)

	// a deal created as won is 100% likely
	_, won := e.req(t, "POST", "/a/deals", fiber.Map{"lead_id": leadID, "title": "Done", "stage": "won", "probability": 10})
	if won["probability"] != float64(100) {
		t.Errorf("won-on-create probability = %v, want 100", won["probability"])
	}

	for name, body := range map[string]fiber.Map{
		"blank title":  {"lead_id": leadID, "title": "  "},
		"bad currency": {"lead_id": leadID, "title": "x", "currency": "EURO"},
		"bad stage":    {"lead_id": leadID, "title": "x", "stage": "pending"},
		"bad prob":     {"lead_id": leadID, "title": "x", "probability": 101},
		"huge amount":  {"lead_id": leadID, "title": "x", "amount": 1e15},
		"bad date":     {"lead_id": leadID, "title": "x", "close_date": "31/12/2026"},
		"negative":     {"lead_id": leadID, "title": "x", "amount": -1},
	} {
		if c, o := e.req(t, "POST", "/a/deals", body); c != 400 {
			t.Errorf("%s: %d %v, want 400", name, c, o)
		}
	}
	// a deleted lead cannot get new deals
	_, _ = e.pool.Exec(context.Background(), `UPDATE leads SET deleted_at = NOW() WHERE id = $1`, leadID)
	if c, _ := e.req(t, "POST", "/a/deals", fiber.Map{"lead_id": leadID, "title": "x"}); c != 404 {
		t.Errorf("deal on a deleted lead = %d, want 404", c)
	}
	_, _ = e.pool.Exec(context.Background(), `UPDATE leads SET deleted_at = NULL WHERE id = $1`, leadID)

	// The deal page sends back the timestamp the API gave it; that must round-trip.
	_, got := e.req(t, "PATCH", "/a/deals/"+id, fiber.Map{"close_date": "2026-12-31T00:00:00Z", "title": "Villa 2"})
	if got["title"] != "Villa 2" {
		t.Fatalf("PATCH with an RFC3339 close_date failed: %v", got)
	}
	if c, o := e.req(t, "PATCH", "/a/deals/"+id, fiber.Map{"close_date": ""}); c != 200 || o["close_date"] != nil {
		t.Errorf("clearing close_date = %d %v", c, o)
	}
	if c, _ := e.req(t, "PATCH", "/a/deals/"+id, fiber.Map{"title": "   "}); c != 400 {
		t.Errorf("blank title on update = %d, want 400", c)
	}

	// stage: 404 for missing, won => 100% + close date, lost => 0%
	if c, _ := e.req(t, "PATCH", "/a/deals/"+uuid.NewString()+"/stage", fiber.Map{"stage": "won"}); c != 404 {
		t.Errorf("stage of a missing deal = %d, want 404", c)
	}
	if c, _ := e.req(t, "PATCH", "/b/deals/"+id+"/stage", fiber.Map{"stage": "won"}); c != 404 {
		t.Errorf("stage of another company's deal = %d, want 404", c)
	}
	e.req(t, "PATCH", "/a/deals/"+id+"/stage", fiber.Map{"stage": "won"})
	if p := e.q(t, `SELECT probability::text FROM deals WHERE id = $1`, id); p != "100" {
		t.Errorf("won probability = %s", p)
	}
	if e.q(t, `SELECT close_date::text FROM deals WHERE id = $1`, id) == "" {
		t.Error("won deal has no close date")
	}
	e.req(t, "PATCH", "/a/deals/"+id+"/stage", fiber.Map{"stage": "lost"})
	if p := e.q(t, `SELECT probability::text FROM deals WHERE id = $1`, id); p != "0" {
		t.Errorf("lost probability = %s", p)
	}

	// listing
	if c, _ := e.req(t, "GET", "/a/deals?owner_id=nope", nil); c != 400 {
		t.Errorf("bad owner_id = %d, want 400", c)
	}
	if c, _ := e.req(t, "GET", "/a/deals?stage=pending", nil); c != 400 {
		t.Errorf("bad stage filter = %d, want 400", c)
	}

	// invoices of a deal: [] not null; 404 for a missing deal; deals with invoices cannot be deleted
	r := httptest.NewRequest("GET", "/a/deals/"+id+"/invoices", nil)
	resp, _ := e.app2.Test(r, -1)
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "[]" {
		t.Errorf("invoices of an invoice-less deal = %s, want []", body)
	}
	if c, _ := e.req(t, "GET", "/a/deals/"+uuid.NewString()+"/invoices", nil); c != 404 {
		t.Errorf("invoices of a missing deal = %d, want 404", c)
	}
	e.req(t, "POST", "/a/invoices", fiber.Map{"deal_id": id, "subtotal": 50})
	if c, _ := e.req(t, "DELETE", "/a/deals/"+id, nil); c != 409 {
		t.Errorf("deleting a deal with invoices = %d, want 409 (was a confusing 422)", c)
	}
	if c, _ := e.req(t, "DELETE", "/a/deals/"+uuid.NewString(), nil); c != 404 {
		t.Errorf("deleting a missing deal = %d, want 404 (was 204)", c)
	}
}

func TestInvoice_NumberingIsRaceFreeAndSendIsGuarded(t *testing.T) {
	e := newCDEnv(t)
	_, leadID := e.newLeadFor(t)
	dealID := e.newDeal(t, leadID, nil)
	year := time.Now().Format("2006")

	// numbering continues past 9999 (the last-four-characters parse wrapped and duplicated) and
	// skips numbers it cannot parse instead of failing
	for _, no := range []string{"INV-" + year + "-9999", "INV-" + year + "-ABC"} {
		_, _ = e.pool.Exec(context.Background(),
			`INSERT INTO vat_invoices (deal_id, company_id, invoice_no, subtotal) VALUES ($1,$2,$3,1)`, dealID, e.a, no)
	}
	c, out := e.req(t, "POST", "/a/invoices", fiber.Map{"deal_id": dealID, "subtotal": 10})
	if c != 201 || out["invoice_no"] != "INV-"+year+"-10000" {
		t.Fatalf("first invoice after 9999 = %d %v, want INV-%s-10000", c, out, year)
	}

	// concurrent creation: every request succeeds with a distinct number
	const n = 12
	var wg sync.WaitGroup
	nums := make([]string, n)
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var o map[string]any
			codes[i], o = e.req(t, "POST", "/a/invoices", fiber.Map{"deal_id": dealID, "subtotal": 10 + i})
			if s, ok := o["invoice_no"].(string); ok {
				nums[i] = s
			}
		}(i)
	}
	wg.Wait()
	seen := map[string]bool{}
	for i := range codes {
		if codes[i] != 201 {
			t.Errorf("concurrent create %d = %d", i, codes[i])
		}
		if seen[nums[i]] {
			t.Errorf("duplicate invoice number %s", nums[i])
		}
		seen[nums[i]] = true
	}
	keys := make([]string, 0, n)
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) != n {
		t.Errorf("%d distinct numbers for %d invoices: %v", len(keys), n, keys)
	}

	// subtotal is rounded to fils, sent only from draft, paid never regresses, unknown ids are 404
	_, inv := e.req(t, "POST", "/a/invoices", fiber.Map{"deal_id": dealID, "subtotal": 10.456})
	if inv["subtotal"] != 10.46 {
		t.Errorf("subtotal = %v, want it rounded to 10.46", inv["subtotal"])
	}
	id := inv["id"].(string)
	if c, _ := e.req(t, "POST", "/a/invoices/"+id+"/send", nil); c != 200 {
		t.Fatalf("send draft = %d", c)
	}
	if c, _ := e.req(t, "POST", "/a/invoices/"+id+"/send", nil); c != 409 {
		t.Errorf("re-send = %d, want 409", c)
	}
	e.req(t, "PATCH", "/a/invoices/"+id+"/status", fiber.Map{"status": "paid"})
	if c, _ := e.req(t, "POST", "/a/invoices/"+id+"/send", nil); c != 409 {
		t.Errorf("sending a paid invoice = %d, want 409", c)
	}
	if got := e.q(t, `SELECT status FROM vat_invoices WHERE id = $1`, id); got != "paid" {
		t.Errorf("a paid invoice regressed to %q", got)
	}
	if c, _ := e.req(t, "POST", "/a/invoices/"+uuid.NewString()+"/send", nil); c != 404 {
		t.Errorf("send of a missing invoice = %d, want 404", c)
	}
	if c, _ := e.req(t, "PATCH", "/a/invoices/"+uuid.NewString()+"/status", fiber.Map{"status": "paid"}); c != 404 {
		t.Errorf("status of a missing invoice = %d, want 404", c)
	}
	if c, _ := e.req(t, "POST", "/a/invoices", fiber.Map{"deal_id": dealID, "subtotal": 1e12}); c != 400 {
		t.Errorf("huge subtotal = %d, want 400", c)
	}
}
