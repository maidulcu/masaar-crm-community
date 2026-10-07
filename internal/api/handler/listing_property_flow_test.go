package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/repo"
)

type lpEnv struct {
	*leadEnv
	app2 *fiber.App
}

func newLPEnv(t *testing.T) *lpEnv {
	t.Helper()
	le := newLeadEnv(t)
	ctx := context.Background()
	t.Cleanup(func() {
		for _, co := range []uuid.UUID{le.a, le.b} {
			_, _ = le.pool.Exec(ctx, `DELETE FROM approval_requests WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM approval_configs WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM offers WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM listings WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM expenses WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM leases WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM rental_properties WHERE company_id = $1`, co)
		}
	})
	approvals := repo.NewApprovalRepo(le.pool)
	lh := NewListingHandler(repo.NewListingRepo(le.pool), approvals)
	ah := NewApprovalHandler(approvals)
	rh := NewRentalPropertyHandler(repo.NewRentalPropertyRepo(le.pool))

	app := fiber.New()
	for prefix, co := range map[string]uuid.UUID{"/a": le.a, "/b": le.b} {
		co := co
		actor := le.admin
		if co == le.b {
			actor = le.adminB
		}
		g := app.Group(prefix, func(c *fiber.Ctx) error {
			c.Locals("company_id", co.String())
			c.Locals("user_id", actor) // a uuid.UUID, exactly as the auth middleware sets it
			return c.Next()
		})
		g.Post("/listings", lh.Create)
		g.Patch("/listings/:id", lh.Update)
		g.Patch("/listings/:id/status", lh.UpdateStatus)
		g.Delete("/listings/:id", lh.Delete)
		g.Patch("/approval-config", ah.SaveConfig)
		g.Post("/approval-requests/:id/review", ah.ReviewRequest)
		g.Post("/rental-properties", rh.Create)
		g.Patch("/rental-properties/:id", rh.Update)
		g.Delete("/rental-properties/:id", rh.Delete)
	}
	return &lpEnv{leadEnv: le, app2: app}
}

func (e *lpEnv) req(t *testing.T, method, path string, body any) (int, map[string]any) {
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

func (e *lpEnv) val(t *testing.T, q string, args ...any) string {
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

func (e *lpEnv) newListing(t *testing.T, extra fiber.Map) string {
	t.Helper()
	body := fiber.Map{"title": "Marina 2BR", "property_type": "apartment", "listing_type": "rent", "price": 90000}
	for k, v := range extra {
		body[k] = v
	}
	code, out := e.req(t, "POST", "/a/listings", body)
	if code != 201 {
		t.Fatalf("create listing = %d %v", code, out)
	}
	return out["id"].(string)
}

func TestListingStatus_WorksAndStampsPublishedAt(t *testing.T) {
	e := newLPEnv(t)
	id := e.newListing(t, nil)

	// This returned 500 for everyone: the handler asserted user_id as a string and panicked.
	code, out := e.req(t, "PATCH", "/a/listings/"+id+"/status", fiber.Map{"status": "published"})
	if code != 200 || out["status"] != "published" {
		t.Fatalf("publish = %d %v", code, out)
	}
	if e.val(t, `SELECT status FROM listings WHERE id = $1`, id) != "published" {
		t.Fatal("status not stored")
	}
	if e.val(t, `SELECT published_at::text FROM listings WHERE id = $1`, id) == "" {
		t.Error("published_at was never set")
	}
	if code, _ := e.req(t, "PATCH", "/a/listings/"+id+"/status", fiber.Map{"status": "sold"}); code != 200 {
		t.Errorf("sold = %d", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/listings/"+id+"/status", fiber.Map{"status": "bogus"}); code != 400 {
		t.Errorf("bogus status = %d, want 400", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/listings/"+uuid.NewString()+"/status", fiber.Map{"status": "sold"}); code != 404 {
		t.Errorf("missing listing = %d, want 404", code)
	}
	if code, _ := e.req(t, "PATCH", "/b/listings/"+id+"/status", fiber.Map{"status": "draft"}); code != 404 {
		t.Errorf("other company's listing = %d, want 404", code)
	}

	// published straight from create
	pub := e.newListing(t, fiber.Map{"status": "published"})
	if e.val(t, `SELECT status FROM listings WHERE id = $1`, pub) != "published" || e.val(t, `SELECT published_at::text FROM listings WHERE id = $1`, pub) == "" {
		t.Error("create with status=published did not publish / stamp published_at")
	}
}

func TestListingApproval_FullWorkflow(t *testing.T) {
	e := newLPEnv(t)
	if code, out := e.req(t, "PATCH", "/a/approval-config", fiber.Map{"listing_approval": true}); code != 200 {
		t.Fatalf("config = %d %v", code, out)
	}
	id := e.newListing(t, nil)

	// Publishing needs approval: the listing stays a draft and one request is filed, however often it is asked.
	code, out := e.req(t, "PATCH", "/a/listings/"+id+"/status", fiber.Map{"status": "published"})
	if code != 200 || out["status"] != "pending_approval" {
		t.Fatalf("publish = %d %v", code, out)
	}
	first := out["approval_id"]
	_, again := e.req(t, "PATCH", "/a/listings/"+id+"/status", fiber.Map{"status": "published"})
	if again["approval_id"] != first {
		t.Errorf("a second publish request opened another approval: %v vs %v", again["approval_id"], first)
	}
	if n := e.val(t, `SELECT count(*)::text FROM approval_requests WHERE entity_id = $1`, id); n != "1" {
		t.Errorf("%s approval requests, want 1", n)
	}

	// The loophole: PATCH /listings/:id used to accept status and skip approval entirely.
	e.req(t, "PATCH", "/a/listings/"+id, fiber.Map{"title": "Renamed", "status": "published"})
	if got := e.val(t, `SELECT status FROM listings WHERE id = $1`, id); got != "draft" {
		t.Fatalf("PATCH /listings/:id changed the status to %q, bypassing approval", got)
	}
	// ...and neither can create
	sneaky := e.newListing(t, fiber.Map{"status": "published"})
	if got := e.val(t, `SELECT status FROM listings WHERE id = $1`, sneaky); got != "draft" {
		t.Errorf("create with status=published under approval = %q, want draft", got)
	}

	// Approving publishes the listing (it used to only flip the request).
	rid := first.(string)
	if code, out := e.req(t, "POST", "/a/approval-requests/"+rid+"/review", fiber.Map{"status": "approved", "note": "ok"}); code != 200 || out["status"] != "approved" {
		t.Fatalf("review = %d %v", code, out)
	}
	if got := e.val(t, `SELECT status FROM listings WHERE id = $1`, id); got != "published" {
		t.Errorf("approved listing status = %q, want published", got)
	}
	if e.val(t, `SELECT published_at::text FROM listings WHERE id = $1`, id) == "" {
		t.Error("published_at not set on approval")
	}
	// a request can be reviewed once
	if code, _ := e.req(t, "POST", "/a/approval-requests/"+rid+"/review", fiber.Map{"status": "rejected"}); code != 409 {
		t.Errorf("second review = %d, want 409", code)
	}
	if code, _ := e.req(t, "POST", "/a/approval-requests/"+uuid.NewString()+"/review", fiber.Map{"status": "approved"}); code != 404 {
		t.Errorf("review of an unknown request = %d, want 404", code)
	}

	// rejecting leaves the listing alone
	id2 := e.newListing(t, nil)
	_, o2 := e.req(t, "PATCH", "/a/listings/"+id2+"/status", fiber.Map{"status": "published"})
	e.req(t, "POST", "/a/approval-requests/"+o2["approval_id"].(string)+"/review", fiber.Map{"status": "rejected"})
	if got := e.val(t, `SELECT status FROM listings WHERE id = $1`, id2); got != "draft" {
		t.Errorf("rejected listing status = %q, want draft", got)
	}
}

func TestListingUpdateCreateDelete_ValidationAndGuards(t *testing.T) {
	e := newLPEnv(t)
	id := e.newListing(t, nil)
	other := e.newListing(t, fiber.Map{"title": "Other"})

	// The edit form sends available_from:"" for an empty date input; that used to fail to parse.
	code, out := e.req(t, "PATCH", "/a/listings/"+id, fiber.Map{"title": "Edited", "available_from": "", "price": 95000, "currency": "usd"})
	if code != 200 || out["title"] != "Edited" || out["currency"] != "USD" {
		t.Fatalf("edit with blank available_from = %d %v", code, out)
	}
	// ...and an id in the body cannot redirect the update to another listing
	e.req(t, "PATCH", "/a/listings/"+id, fiber.Map{"id": other, "title": "Hijack"})
	if got := e.val(t, `SELECT title FROM listings WHERE id = $1`, other); got != "Other" {
		t.Errorf("a body id redirected the update: other listing is now %q", got)
	}
	if got := e.val(t, `SELECT title FROM listings WHERE id = $1`, id); got != "Hijack" {
		t.Errorf("the addressed listing was not updated: %q", got)
	}

	for name, body := range map[string]fiber.Map{
		"negative price": {"title": "x", "property_type": "villa", "price": -5},
		"zero price":     {"title": "x", "property_type": "villa", "price": 0},
		"bad type":       {"title": "x", "property_type": "castle", "price": 5},
		"bad listing":    {"title": "x", "property_type": "villa", "price": 5, "listing_type": "lease"},
		"bad currency":   {"title": "x", "property_type": "villa", "price": 5, "currency": "DIRHAM"},
		"bad latitude":   {"title": "x", "property_type": "villa", "price": 5, "latitude": 123},
		"bad status":     {"title": "x", "property_type": "villa", "price": 5, "status": "live"},
		"long ref":       {"title": "x", "property_type": "villa", "price": 5, "reference_number": string(bytes.Repeat([]byte("r"), 101))},
	} {
		if c, o := e.req(t, "POST", "/a/listings", body); c != 400 {
			t.Errorf("%s: %d %v, want 400", name, c, o)
		}
	}
	if c, _ := e.req(t, "PATCH", "/a/listings/"+id, fiber.Map{"price": -1}); c != 400 {
		t.Errorf("negative price on update = %d, want 400", c)
	}

	// delete: 404, then guarded by offers
	if c, _ := e.req(t, "DELETE", "/a/listings/"+uuid.NewString(), nil); c != 404 {
		t.Errorf("delete missing = %d, want 404", c)
	}
	if c, _ := e.req(t, "DELETE", "/b/listings/"+id, nil); c != 404 {
		t.Errorf("delete other company's = %d, want 404", c)
	}
	ct := e.contact(t, e.a)
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO offers (listing_id, contact_id, company_id, offer_amount) VALUES ($1,$2,$3,100)`, id, ct, e.a); err != nil {
		t.Fatal(err)
	}
	c, o := e.req(t, "DELETE", "/a/listings/"+id, nil)
	if c != 409 || o["offers"] != float64(1) {
		t.Fatalf("delete with offers = %d %v, want 409 with offers=1", c, o)
	}
	if c, _ := e.req(t, "DELETE", "/a/listings/"+id+"?force=true", nil); c != 204 {
		t.Errorf("force delete = %d, want 204", c)
	}
	if e.val(t, `SELECT count(*)::text FROM offers WHERE company_id = $1`, e.a) != "0" {
		t.Error("offers survived a forced listing delete")
	}
}

func TestRentalProperty_DefaultsValidationAndGuards(t *testing.T) {
	e := newLPEnv(t)

	// Zero values used to be stored literally: status '', currency '', 0 units.
	code, out := e.req(t, "POST", "/a/rental-properties", fiber.Map{"name": "Tower A", "property_type": "Apartment"})
	if code != 201 || out["status"] != "active" || out["occupancy_status"] != "vacant" || out["currency"] != "AED" || out["units_count"] != float64(1) || out["property_type"] != "apartment" {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)

	for name, body := range map[string]fiber.Map{
		"bad type":       {"name": "x", "property_type": "igloo"},
		"occupied>units": {"name": "x", "property_type": "villa", "units_count": 2, "total_occupied_units": 3},
		"bad status":     {"name": "x", "property_type": "villa", "status": "gone"},
		"negative value": {"name": "x", "property_type": "villa", "market_value": -1},
		"long field":     {"name": "x", "property_type": "villa", "postal_code": string(bytes.Repeat([]byte("9"), 21))},
	} {
		if c, o := e.req(t, "POST", "/a/rental-properties", body); c != 400 {
			t.Errorf("%s: %d %v, want 400", name, c, o)
		}
	}

	// blank purchase_date from the form; id in the body is ignored; currency / purchase price now persist
	c2, o2 := e.req(t, "POST", "/a/rental-properties", fiber.Map{"name": "Tower B", "property_type": "villa"})
	if c2 != 201 {
		t.Fatal("setup")
	}
	idB := o2["id"].(string)
	code, out = e.req(t, "PATCH", "/a/rental-properties/"+id, fiber.Map{"id": idB, "name": "Tower A2", "purchase_date": "", "currency": "usd", "purchase_price": 1500000})
	if code != 200 {
		t.Fatalf("patch = %d %v", code, out)
	}
	if got := e.val(t, `SELECT name FROM rental_properties WHERE id = $1`, idB); got != "Tower B" {
		t.Errorf("a body id redirected the update onto another property: %q", got)
	}
	if e.val(t, `SELECT currency FROM rental_properties WHERE id = $1`, id) != "USD" || e.val(t, `SELECT purchase_price::text FROM rental_properties WHERE id = $1`, id) != "1500000.00" {
		t.Error("currency / purchase_price were not saved (the response said they were)")
	}
	if c, _ := e.req(t, "PATCH", "/b/rental-properties/"+id, fiber.Map{"name": "Stolen"}); c != 404 {
		t.Errorf("other company's patch = %d, want 404", c)
	}

	// delete: 404, and leases/expenses block it with a clear 409
	if c, _ := e.req(t, "DELETE", "/a/rental-properties/"+uuid.NewString(), nil); c != 404 {
		t.Errorf("delete missing = %d, want 404", c)
	}
	var cat uuid.UUID
	if err := e.pool.QueryRow(context.Background(), `INSERT INTO expense_categories (company_id, category_name, category_type) VALUES ($1, 'Repairs', 'maintenance') RETURNING id`, e.a).Scan(&cat); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM expense_categories WHERE company_id = $1`, e.a)
	})
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO expenses (company_id, category_id, property_id, expense_date, amount, description, created_by)
		VALUES ($1, $2, $3, CURRENT_DATE, 10, 'x', $4)`, e.a, cat, id, e.admin); err != nil {
		t.Fatal(err)
	}
	if c, _ := e.req(t, "DELETE", "/a/rental-properties/"+id, nil); c != 409 {
		t.Errorf("delete with an expense = %d, want 409 (was a confusing 422)", c)
	}
	if c, _ := e.req(t, "DELETE", "/a/rental-properties/"+idB, nil); c != 204 {
		t.Errorf("delete = %d, want 204", c)
	}
}
