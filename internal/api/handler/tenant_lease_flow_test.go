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

	"github.com/maidulcu/masaar-crm/internal/ai"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type tlEnv struct {
	*leadEnv
	app2 *fiber.App
}

func newTLEnv(t *testing.T) *tlEnv {
	t.Helper()
	le := newLeadEnv(t)
	ctx := context.Background()
	t.Cleanup(func() {
		for _, co := range []uuid.UUID{le.a, le.b} {
			_, _ = le.pool.Exec(ctx, `DELETE FROM maintenance_photos WHERE task_id IN (SELECT id FROM maintenance_tasks WHERE company_id = $1)`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM maintenance_tasks WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM inspections WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM inspection_templates WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM expense_approvals WHERE expense_id IN (SELECT id FROM expenses WHERE company_id = $1)`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM expenses WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM expense_categories WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `UPDATE payments SET bank_transaction_id = NULL WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM bank_transactions WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM bank_statements WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM bank_integrations WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM renewal_communication_log WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM lease_renewal_workflows WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM payments WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM leases WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM lease_templates WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM tenants WHERE company_id = $1`, co)
			_, _ = le.pool.Exec(ctx, `DELETE FROM rental_properties WHERE company_id = $1`, co)
		}
	})
	th := NewTenantHandler(repo.NewTenantRepo(le.pool))
	lh := NewLeaseHandler(repo.NewLeaseRepo(le.pool))
	ph := NewPaymentHandler(repo.NewPaymentRepo(le.pool))
	tph := NewLeaseTemplateHandler(repo.NewLeaseTemplateRepo(le.pool))
	rh := NewRentalPropertyHandler(repo.NewRentalPropertyRepo(le.pool))
	eh := NewExpenseHandler(repo.NewExpenseRepository(le.pool))
	mh := NewMaintenanceTaskHandler(repo.NewMaintenanceTaskRepo(le.pool))
	ih := NewInspectionHandler(repo.NewInspectionTemplateRepo(le.pool), repo.NewInspectionRepo(le.pool))
	bih := NewBankIntegrationHandler(repo.NewBankIntegrationRepo(le.pool))
	pch := NewPaymentConfirmationHandler(repo.NewPaymentConfirmationRepo(le.pool), repo.NewPaymentRepo(le.pool), ai.NewPaymentConfirmationService(nil, nil, nil, nil, nil, nil, nil))
	lrh := NewLeaseRenewalHandler(repo.NewLeaseRenewalRepo(le.pool), repo.NewRenewalTemplateRepo(le.pool),
		repo.NewRenewalCommunicationLogRepo(le.pool), repo.NewLeaseRepo(le.pool))

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
		g.Post("/tenants", th.Create)
		g.Patch("/tenants/:id", th.Update)
		g.Delete("/tenants/:id", th.Delete)
		g.Post("/tenants/:id/verify", th.Verify)
		g.Post("/leases", lh.Create)
		g.Patch("/leases/:id", lh.Update)
		g.Delete("/leases/:id", lh.Delete)
		g.Get("/payments", ph.List)
		g.Post("/payments", ph.Create)
		g.Patch("/payments/:id", ph.Update)
		g.Delete("/payments/:id", ph.Delete)
		g.Post("/lease-templates", tph.Create)
		g.Patch("/lease-templates/:id", tph.Update)
		g.Delete("/lease-templates/:id", tph.Delete)
		g.Post("/rental-properties", rh.Create)
		g.Post("/expense-categories", eh.CreateCategory)
		g.Post("/expenses", eh.CreateExpense)
		g.Get("/expenses/:id", eh.GetExpense)
		g.Patch("/expenses/:id", eh.UpdateExpense)
		g.Delete("/expenses/:id", eh.DeleteExpense)
		g.Post("/expenses/:id/approve", eh.ApproveExpense)
		g.Post("/maintenance-tasks", mh.Create)
		g.Get("/maintenance-tasks", mh.List)
		g.Get("/maintenance-tasks/:id", mh.Get)
		g.Patch("/maintenance-tasks/:id", mh.Update)
		g.Post("/maintenance-tasks/:id/complete", mh.Complete)
		g.Post("/maintenance-tasks/:id/photos", mh.AddPhoto)
		g.Get("/maintenance-tasks/:id/photos", mh.GetPhotos)
		g.Delete("/maintenance-tasks/:id", mh.Delete)
		g.Post("/inspection-templates", ih.CreateTemplate)
		g.Post("/inspections", ih.CreateInspection)
		g.Get("/inspections/:id", ih.GetInspection)
		g.Patch("/inspections/:id", ih.UpdateInspection)
		g.Post("/inspections/:id/complete", ih.CompleteInspection)
		g.Delete("/rental-properties/:id", rh.Delete)
		g.Get("/bank-integrations/:id", bih.Get)
		g.Get("/bank-integrations", bih.List)
		g.Post("/bank-integrations", bih.Create)
		g.Patch("/bank-integrations/:id", bih.Update)
		g.Delete("/bank-integrations/:id", bih.Delete)
		g.Post("/payments/:payment_id/send-confirmation", pch.Send)
		g.Post("/lease-renewals/:lease_id/initiate", lrh.Initiate)
		g.Put("/lease-renewals/:id/propose", lrh.Propose)
		g.Post("/lease-renewals/:id/send-offer", lrh.SendOffer)
		g.Put("/lease-renewals/:id/accept", lrh.Accept)
		g.Put("/lease-renewals/:id/reject", lrh.Reject)
	}
	return &tlEnv{leadEnv: le, app2: app}
}

func (e *tlEnv) req(t *testing.T, method, path string, body any) (int, map[string]any) {
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

func (e *tlEnv) mk(t *testing.T, path string, body any) string {
	t.Helper()
	code, out := e.req(t, "POST", path, body)
	if code != 201 {
		t.Fatalf("POST %s = %d %v", path, code, out)
	}
	return out["id"].(string)
}

func (e *tlEnv) tenant(t *testing.T, idNumber string) string {
	return e.mk(t, "/a/tenants", fiber.Map{"full_name_en": "Sara Ali", "id_type": "emirates_id", "id_number": idNumber})
}

func (e *tlEnv) property(t *testing.T) string {
	return e.mk(t, "/a/rental-properties", fiber.Map{"name": "Tower 1", "property_type": "apartment"})
}

func (e *tlEnv) lease(t *testing.T, prop, ten string, extra fiber.Map) (int, map[string]any) {
	body := fiber.Map{"property_id": prop, "tenant_id": ten, "start_date": "2026-01-01", "end_date": "2026-12-31", "monthly_rent": 5000}
	for k, v := range extra {
		body[k] = v
	}
	return e.req(t, "POST", "/a/leases", body)
}

func (e *tlEnv) val(t *testing.T, q string, args ...any) string {
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

func TestTenant_VerificationCannotBeForgedAndIdsAreCompanyScoped(t *testing.T) {
	e := newTLEnv(t)
	code, out := e.req(t, "POST", "/a/tenants", fiber.Map{
		"full_name_en": "Sara", "id_type": "emirates_id", "id_number": "784-1", "id_expiry_date": "2030-01-01",
		"is_verified": true, "verification_status": "verified",
	})
	if code != 201 {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)
	if v := e.val(t, `SELECT is_verified::text FROM tenants WHERE id=$1`, id); v != "false" {
		t.Fatalf("is_verified forged: %s", v)
	}
	// still not forgeable via update; an empty expiry date must not be a parse error
	if code, out := e.req(t, "PATCH", "/a/tenants/"+id, fiber.Map{"full_name_en": "Sara B", "id_type": "emirates_id", "id_number": "784-1", "id_expiry_date": "", "is_verified": true}); code != 200 {
		t.Fatalf("update = %d %v", code, out)
	}
	if v := e.val(t, `SELECT is_verified::text FROM tenants WHERE id=$1`, id); v != "false" {
		t.Fatalf("is_verified forged via update: %s", v)
	}
	if code, _ := e.req(t, "POST", "/a/tenants/"+id+"/verify", fiber.Map{}); code != 200 {
		t.Fatalf("verify = %d", code)
	}
	if v := e.val(t, `SELECT is_verified::text FROM tenants WHERE id=$1`, id); v != "true" {
		t.Fatalf("verify did not stick: %s", v)
	}
	// duplicate within a company -> 409, same number in another company is fine
	if code, _ := e.req(t, "POST", "/a/tenants", fiber.Map{"full_name_en": "X", "id_type": "passport", "id_number": "784-1"}); code != 409 {
		t.Fatalf("duplicate id_number = %d, want 409", code)
	}
	if code, out := e.req(t, "POST", "/b/tenants", fiber.Map{"full_name_en": "Y", "id_type": "passport", "id_number": "784-1"}); code != 201 {
		t.Fatalf("same id_number in another company = %d %v", code, out)
	}
	if code, _ := e.req(t, "POST", "/a/tenants", fiber.Map{"full_name_en": "Z", "id_type": "nonsense"}); code != 400 {
		t.Fatalf("bad id_type = %d, want 400", code)
	}
	if code, _ := e.req(t, "DELETE", "/b/tenants/"+id, nil); code != 404 {
		t.Fatalf("cross-company delete = %d, want 404", code)
	}
}

func TestLease_ValidationOverlapAndGuards(t *testing.T) {
	e := newTLEnv(t)
	prop, ten := e.property(t), e.tenant(t, "L-1")

	for name, extra := range map[string]fiber.Map{
		"end before start": {"start_date": "2026-06-01", "end_date": "2026-01-01"},
		"zero rent":        {"monthly_rent": 0},
		"bad frequency":    {"payment_frequency": "weekly-ish"},
		"late fee > 100":   {"late_fee_percent": 150},
	} {
		if code, out := e.lease(t, prop, ten, extra); code != 400 {
			t.Errorf("%s = %d %v, want 400", name, code, out)
		}
	}
	if code, _ := e.lease(t, uuid.NewString(), ten, nil); code != 404 && code != 400 && code != 409 {
		t.Errorf("unknown property = %d, want a client error", code)
	}

	code, out := e.lease(t, prop, ten, fiber.Map{"status": "terminated"})
	if code != 400 {
		t.Fatalf("creating a terminated lease = %d %v, want 400", code, out)
	}
	code, out = e.lease(t, prop, ten, nil)
	if code != 201 {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)
	if code, _ := e.lease(t, prop, ten, fiber.Map{"start_date": "2026-06-01", "end_date": "2027-05-31"}); code != 409 {
		t.Fatalf("overlapping lease on a single-unit property = %d, want 409", code)
	}
	if code, out := e.lease(t, prop, ten, fiber.Map{"start_date": "2027-01-01", "end_date": "2027-12-31"}); code != 201 {
		t.Fatalf("adjacent lease = %d %v", code, out)
	}

	// the property/tenant/dates of an existing lease are not rewritable via PATCH
	other := e.property(t)
	if code, out := e.req(t, "PATCH", "/a/leases/"+id, fiber.Map{
		"property_id": other, "start_date": "2020-01-01", "end_date": "2020-02-01",
		"monthly_rent": 6000, "payment_frequency": "monthly", "status": "active",
	}); code != 200 {
		t.Fatalf("update = %d %v", code, out)
	}
	if v := e.val(t, `SELECT property_id::text FROM leases WHERE id=$1`, id); v != prop {
		t.Fatalf("property_id rewritten to %s", v)
	}
	if v := e.val(t, `SELECT start_date::text FROM leases WHERE id=$1`, id); v != "2026-01-01" {
		t.Fatalf("start_date rewritten to %s", v)
	}
	if v := e.val(t, `SELECT monthly_rent::text FROM leases WHERE id=$1`, id); v != "6000.00" && v != "6000" {
		t.Fatalf("monthly_rent not updated: %s", v)
	}
	if code, _ := e.req(t, "DELETE", "/b/leases/"+id, nil); code != 404 {
		t.Fatalf("cross-company delete = %d, want 404", code)
	}
}

func TestPayment_FilterValidationAndLeaseCurrency(t *testing.T) {
	e := newTLEnv(t)
	prop, ten := e.property(t), e.tenant(t, "P-1")
	code, out := e.lease(t, prop, ten, fiber.Map{"currency": "SAR"})
	if code != 201 {
		t.Fatalf("lease = %d %v", code, out)
	}
	l1 := out["id"].(string)
	prop2 := e.property(t)
	_, out = e.lease(t, prop2, ten, nil)
	l2 := out["id"].(string)

	pid := e.mk(t, "/a/payments", fiber.Map{"lease_id": l1, "amount": 5000, "due_date": "2026-02-01", "payment_method": "cheque", "status": "pending", "currency": "USD"})
	e.mk(t, "/a/payments", fiber.Map{"lease_id": l2, "amount": 100, "due_date": "2026-02-01", "payment_method": "cash"})

	if v := e.val(t, `SELECT currency FROM payments WHERE id=$1`, pid); v != "SAR" {
		t.Fatalf("payment currency = %s, want the lease's SAR", v)
	}
	if code, _ := e.req(t, "POST", "/a/payments", fiber.Map{"lease_id": l1, "amount": -5, "due_date": "2026-03-01"}); code != 400 {
		t.Fatalf("negative amount = %d, want 400", code)
	}
	if code, _ := e.req(t, "POST", "/a/payments", fiber.Map{"lease_id": l1, "amount": 5, "due_date": "2026-03-01", "payment_method": "bitcoin"}); code != 400 {
		t.Fatalf("bad method = %d, want 400", code)
	}
	// payment on another company's lease
	if code, _ := e.req(t, "POST", "/b/payments", fiber.Map{"lease_id": l1, "amount": 5, "due_date": "2026-03-01"}); code == 201 {
		t.Fatal("payment against another company's lease was accepted")
	}

	code, out = e.req(t, "GET", "/a/payments?lease_id="+l1, nil)
	if code != 200 {
		t.Fatalf("filtered list = %d %v", code, out)
	}
	if data, _ := out["data"].([]any); len(data) != 1 {
		t.Fatalf("lease filter returned %d payments, want 1", len(data))
	}
	if code, _ := e.req(t, "GET", "/a/payments?lease_id=nope", nil); code != 400 {
		t.Fatalf("bad lease_id = %d, want 400", code)
	}

	// received sets paid_date; reconciliation fields are not client-writable
	if code, out := e.req(t, "PATCH", "/a/payments/"+pid, fiber.Map{
		"amount": 5000, "status": "received", "payment_method": "cheque", "lease_id": l2, "due_date": "2026-02-01",
		"bank_transaction_id": uuid.NewString(),
	}); code != 200 {
		t.Fatalf("update = %d %v", code, out)
	}
	if v := e.val(t, `SELECT lease_id::text FROM payments WHERE id=$1`, pid); v != l1 {
		t.Fatalf("lease_id rewritten to %s", v)
	}
	if v := e.val(t, `SELECT paid_date::text FROM payments WHERE id=$1`, pid); v == "" {
		t.Fatal("paid_date not set for a received payment")
	}
	if v := e.val(t, `SELECT bank_transaction_id::text FROM payments WHERE id=$1`, pid); v != "" {
		t.Fatalf("bank_transaction_id forged: %s", v)
	}
	// a lease with payments cannot be deleted
	if code, _ := e.req(t, "DELETE", "/a/leases/"+l1, nil); code != 409 {
		t.Fatalf("deleting a lease with payments = %d, want 409", code)
	}
	if code, _ := e.req(t, "DELETE", "/a/payments/"+uuid.NewString(), nil); code != 404 {
		t.Fatalf("deleting a missing payment = %d, want 404", code)
	}
}

func TestLeaseTemplate_ValidationAndSingleDefault(t *testing.T) {
	e := newTLEnv(t)
	if code, _ := e.req(t, "POST", "/a/lease-templates", fiber.Map{"name": "x", "payment_frequency": "daily"}); code != 400 {
		t.Fatalf("bad frequency = %d, want 400", code)
	}
	t1 := e.mk(t, "/a/lease-templates", fiber.Map{"name": "Standard", "payment_frequency": "monthly", "is_default": true})
	t2 := e.mk(t, "/a/lease-templates", fiber.Map{"name": "Premium", "payment_frequency": "quarterly", "is_default": true})
	if n := e.val(t, `SELECT COUNT(*)::text FROM lease_templates WHERE company_id=$1 AND is_default`, e.a); n != "1" {
		t.Fatalf("%s default templates, want exactly 1", n)
	}
	if v := e.val(t, `SELECT is_default::text FROM lease_templates WHERE id=$1`, t2); v != "true" {
		t.Fatalf("newest default not kept, t1=%s t2 default=%s", t1, v)
	}
	if code, _ := e.req(t, "DELETE", "/b/lease-templates/"+t1, nil); code != 404 {
		t.Fatalf("cross-company delete = %d, want 404", code)
	}
	if code, _ := e.req(t, "DELETE", "/a/lease-templates/"+uuid.NewString(), nil); code != 404 {
		t.Fatalf("missing delete = %d, want 404", code)
	}
}

func TestLeaseRenewal_UsesLeaseEndAndGuardsState(t *testing.T) {
	e := newTLEnv(t)
	prop, ten := e.property(t), e.tenant(t, "R-1")
	_, out := e.lease(t, prop, ten, nil)
	lease := out["id"].(string)

	if code, _ := e.req(t, "POST", "/a/lease-renewals/"+uuid.NewString()+"/initiate", nil); code != 404 {
		t.Fatalf("initiate for a missing lease = %d, want 404", code)
	}
	if code, _ := e.req(t, "POST", "/b/lease-renewals/"+lease+"/initiate", nil); code != 404 {
		t.Fatalf("initiate for another company's lease = %d, want 404", code)
	}
	code, out := e.req(t, "POST", "/a/lease-renewals/"+lease+"/initiate", nil)
	if code != 201 {
		t.Fatalf("initiate = %d %v", code, out)
	}
	if code, _ := e.req(t, "POST", "/a/lease-renewals/"+lease+"/initiate", nil); code != 409 {
		t.Fatalf("second initiate = %d, want 409", code)
	}
	rid := e.val(t, `SELECT id::text FROM lease_renewal_workflows WHERE lease_id=$1`, lease)
	if d := e.val(t, `SELECT renewal_date::text FROM lease_renewal_workflows WHERE id=$1`, rid); d != "2026-12-31" {
		t.Fatalf("renewal_date = %s, want the lease end 2026-12-31", d)
	}
	if code, _ := e.req(t, "PUT", "/a/lease-renewals/"+rid+"/propose", fiber.Map{"proposed_rent_amount": -1}); code != 400 {
		t.Fatalf("negative proposal = %d, want 400", code)
	}
	// no template: the offer is still recorded, without a dangling zero template id
	if code, out := e.req(t, "POST", "/a/lease-renewals/"+rid+"/send-offer", fiber.Map{"communication_type": "email"}); code != 200 {
		t.Fatalf("send-offer without template = %d %v", code, out)
	}
	if code, _ := e.req(t, "PUT", "/a/lease-renewals/"+rid+"/accept", nil); code != 200 {
		t.Fatalf("accept = %d", code)
	}
	if code, _ := e.req(t, "PUT", "/a/lease-renewals/"+rid+"/reject", nil); code != 409 {
		t.Fatalf("reject after accept = %d, want 409", code)
	}
}

// mk2 creates a record whose response is wrapped as {"data": {...}} and returns its id.
func (e *tlEnv) mk2(t *testing.T, path string, body any) string {
	t.Helper()
	code, out := e.req(t, "POST", path, body)
	if code != 201 {
		t.Fatalf("POST %s = %d %v", path, code, out)
	}
	return out["data"].(map[string]any)["id"].(string)
}

func ctxBG() context.Context { return context.Background() }
