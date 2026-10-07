package handler

import (
	"context"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/repo"
)

func TestBankIntegration_ReadableValidatedAndGuarded(t *testing.T) {
	e := newTLEnv(t)
	for name, body := range map[string]fiber.Map{
		"bad type":      {"bank_name": "ENBD", "account_number": "1", "integration_type": "carrier-pigeon"},
		"http endpoint": {"bank_name": "ENBD", "account_number": "1", "integration_type": "api", "api_endpoint": "http://10.0.0.1/x"},
		"bad iban":      {"bank_name": "ENBD", "account_number": "1", "integration_type": "manual", "iban": "12"},
		"bad interval":  {"bank_name": "ENBD", "account_number": "1", "integration_type": "manual", "sync_interval_hours": 100000},
	} {
		if code, out := e.req(t, "POST", "/a/bank-integrations", body); code != 400 {
			t.Errorf("%s = %d %v, want 400", name, code, out)
		}
	}
	code, out := e.req(t, "POST", "/a/bank-integrations", fiber.Map{
		"bank_name": "ENBD", "account_number": "1", "integration_type": "manual",
		"is_connected": true, "sync_error_count": 9,
	})
	if code != 201 {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)
	if v := e.val(t, `SELECT status FROM bank_integrations WHERE id=$1`, id); v != "active" {
		t.Fatalf("status = %q, want the default active", v)
	}
	if v := e.val(t, `SELECT is_connected::text FROM bank_integrations WHERE id=$1`, id); v != "false" {
		t.Fatalf("is_connected taken from the client: %s", v)
	}
	// readable straight after creation (NULL last_sync_error used to break every read)
	if code, out := e.req(t, "GET", "/a/bank-integrations/"+id, nil); code != 200 {
		t.Fatalf("get = %d %v", code, out)
	}
	if code, out := e.req(t, "GET", "/a/bank-integrations", nil); code != 200 {
		t.Fatalf("list = %d %v", code, out)
	}
	// an id in the body cannot redirect the update to another integration
	other := func() string {
		_, o := e.req(t, "POST", "/a/bank-integrations", fiber.Map{"bank_name": "Other", "account_number": "2", "integration_type": "manual"})
		return o["id"].(string)
	}()
	if code, out := e.req(t, "PATCH", "/a/bank-integrations/"+id, fiber.Map{"id": other, "bank_name": "Renamed", "account_number": "1", "integration_type": "manual", "is_connected": true}); code != 200 {
		t.Fatalf("update = %d %v", code, out)
	}
	if v := e.val(t, `SELECT bank_name FROM bank_integrations WHERE id=$1`, other); v != "Other" {
		t.Fatalf("another integration was overwritten: %s", v)
	}
	if v := e.val(t, `SELECT bank_name FROM bank_integrations WHERE id=$1`, id); v != "Renamed" {
		t.Fatalf("update not applied: %s", v)
	}
	if v := e.val(t, `SELECT is_connected::text FROM bank_integrations WHERE id=$1`, id); v != "false" {
		t.Fatalf("is_connected taken from the client on update: %s", v)
	}
	if code, _ := e.req(t, "DELETE", "/b/bank-integrations/"+id, nil); code != 404 {
		t.Fatalf("cross-company delete = %d, want 404", code)
	}
	if code, _ := e.req(t, "DELETE", "/a/bank-integrations/"+uuid.NewString(), nil); code != 404 {
		t.Fatalf("missing delete = %d, want 404", code)
	}
	// a statement references it
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO bank_statements (id, company_id, bank_integration_id, file_name, file_size_bytes, file_url, file_format, uploaded_by, upload_date, processing_status, data_classification) VALUES ($1,$2,$3,'s.csv',1,'','csv',$4,NOW(),'pending','confidential')`,
		uuid.New(), e.a, id, e.admin); err != nil {
		t.Fatal(err)
	}
	if code, _ := e.req(t, "DELETE", "/a/bank-integrations/"+id, nil); code != 409 {
		t.Fatalf("delete with statements = %d, want 409", code)
	}
}

func TestPayment_ReconciledLockPaidDateAndConfirmation(t *testing.T) {
	e := newTLEnv(t)
	prop, ten := e.property(t), e.tenant(t, "PC-1")
	_, out := e.lease(t, prop, ten, nil)
	lease := out["id"].(string)
	pid := e.mk(t, "/a/payments", fiber.Map{"lease_id": lease, "amount": 1000, "due_date": "2026-02-01", "payment_method": "transfer", "status": "received", "paid_date": "2026-02-01"})

	// a pending payment never keeps a paid date
	if code, out := e.req(t, "PATCH", "/a/payments/"+pid, fiber.Map{"amount": 1000, "payment_method": "transfer", "status": "pending", "paid_date": "2026-02-02"}); code != 200 {
		t.Fatalf("update = %d %v", code, out)
	}
	if v := e.val(t, `SELECT paid_date::text FROM payments WHERE id=$1`, pid); v != "" {
		t.Fatalf("paid_date kept on a pending payment: %s", v)
	}
	// confirmations: only for received payments, and honest when unavailable
	if code, _ := e.req(t, "POST", "/a/payments/"+pid+"/send-confirmation", nil); code != 409 {
		t.Fatalf("confirmation for a pending payment = %d, want 409", code)
	}
	if code, _ := e.req(t, "POST", "/a/payments/"+uuid.NewString()+"/send-confirmation", nil); code != 404 {
		t.Fatalf("confirmation for a missing payment = %d, want 404", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/payments/"+pid, fiber.Map{"amount": 1000, "payment_method": "transfer", "status": "received"}); code != 200 {
		t.Fatal("could not mark received")
	}
	if code, out := e.req(t, "POST", "/a/payments/"+pid+"/send-confirmation", nil); code != 501 {
		t.Fatalf("confirmation in this edition = %d %v, want 501 (was 201 with null)", code, out)
	}
	if code, _ := e.req(t, "POST", "/b/payments/"+pid+"/send-confirmation", nil); code != 404 {
		t.Fatalf("cross-company confirmation = %d, want 404", code)
	}

	// reconcile it with a bank transaction: amount/status/delete are then locked
	bt := uuid.New()
	if _, err := e.pool.Exec(context.Background(), `INSERT INTO bank_transactions (id, company_id, external_id, transaction_date, amount, currency, status) VALUES ($1,$2,'x1',NOW(),1000,'AED','completed')`, bt, e.a); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(context.Background(), `UPDATE payments SET bank_transaction_id=$1 WHERE id=$2`, bt, pid); err != nil {
		t.Fatal(err)
	}
	if code, _ := e.req(t, "PATCH", "/a/payments/"+pid, fiber.Map{"amount": 999, "payment_method": "transfer", "status": "received"}); code != 409 {
		t.Fatalf("amount change on a reconciled payment = %d, want 409", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/payments/"+pid, fiber.Map{"amount": 1000, "payment_method": "transfer", "status": "received", "notes": "ok"}); code != 200 {
		t.Fatalf("note on a reconciled payment = %d, want 200", code)
	}
	if code, _ := e.req(t, "DELETE", "/a/payments/"+pid, nil); code != 409 {
		t.Fatalf("delete of a reconciled payment = %d, want 409", code)
	}
}

func TestAnalytics_PastDuePendingCountsAsOverdue(t *testing.T) {
	e := newTLEnv(t)
	prop, ten := e.property(t), e.tenant(t, "AN-1")
	_, out := e.lease(t, prop, ten, nil)
	lease := out["id"].(string)
	old := time.Now().AddDate(0, 0, -20).Format("2006-01-02")
	future := time.Now().AddDate(0, 0, 20).Format("2006-01-02")
	e.mk(t, "/a/payments", fiber.Map{"lease_id": lease, "amount": 700, "due_date": old, "payment_method": "cash"})
	e.mk(t, "/a/payments", fiber.Map{"lease_id": lease, "amount": 300, "due_date": future, "payment_method": "cash"})

	var cat uuid.UUID
	if err := e.pool.QueryRow(context.Background(), `INSERT INTO expense_categories (company_id, category_name, category_type) VALUES ($1, 'Power', 'utilities') RETURNING id`, e.a).Scan(&cat); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM expenses WHERE company_id = $1`, e.a)
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM expense_categories WHERE company_id = $1`, e.a)
	})
	for _, amt := range []float64{50, 999} {
		del := "NULL"
		if amt == 999 {
			del = "NOW()"
		}
		if _, err := e.pool.Exec(context.Background(), `INSERT INTO expenses (company_id, category_id, amount, expense_date, created_by, deleted_at) VALUES ($1,$2,$3,CURRENT_DATE,$4,`+del+`)`, e.a, cat, amt, e.admin); err != nil {
			t.Fatal(err)
		}
	}

	ar := repo.NewAnalyticsRepository(e.pool, nil)
	ta, err := ar.GetTenantAnalytics(context.Background(), e.a)
	if err != nil {
		t.Fatal(err)
	}
	if ta.OverduePayments != 1 || ta.OverdueDuesAmount != 700 {
		t.Fatalf("overdue = %d / %.0f, want 1 / 700 (nothing ever set status=overdue)", ta.OverduePayments, ta.OverdueDuesAmount)
	}
	fa, err := ar.GetFinancialAnalytics(context.Background(), e.a, time.Now().AddDate(0, -1, 0), time.Now().AddDate(0, 1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if fa.UtilitiesExpense != 50 || fa.TotalExpenses != 50 {
		t.Fatalf("expenses = %.0f / %.0f, want 50 / 50 (the endpoint used to fail outright; deleted expenses are excluded)", fa.UtilitiesExpense, fa.TotalExpenses)
	}
	if fa.RentOverdue != 700 || fa.RentPending != 300 {
		t.Fatalf("financial overdue/pending = %.0f / %.0f, want 700 / 300", fa.RentOverdue, fa.RentPending)
	}
}

func TestInvoice_IssuedCannotRevertToDraftAndGetIs404(t *testing.T) {
	e := newCDEnv(t)
	cid, leadID := e.newLeadFor(t)
	_ = cid
	deal := e.newDeal(t, leadID, nil)
	code, out := e.req(t, "POST", "/a/invoices", fiber.Map{"deal_id": deal, "subtotal": 100})
	if code != 201 {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)
	if code, _ := e.req(t, "PATCH", "/a/invoices/"+id+"/status", fiber.Map{"status": "draft"}); code != 200 {
		t.Fatalf("draft -> draft = %d, want 200", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/invoices/"+id+"/status", fiber.Map{"status": "paid"}); code != 200 {
		t.Fatalf("draft -> paid = %d, want 200", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/invoices/"+id+"/status", fiber.Map{"status": "draft"}); code != 409 {
		t.Fatalf("paid -> draft = %d, want 409", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/invoices/"+id+"/status", fiber.Map{"status": "sent"}); code != 200 {
		t.Fatalf("paid -> sent (payment reversal) = %d, want 200", code)
	}
	// a freshly created invoice is readable (NULL qr_payload used to make every GET a 500)
	if code, out := e.req(t, "GET", "/a/invoices/"+id, nil); code != 200 {
		t.Fatalf("get = %d %v", code, out)
	}
	// the list filters by status on the server
	if code, out := e.req(t, "GET", "/a/invoices?status=draft", nil); code != 200 || out["total"].(float64) != 0 {
		t.Fatalf("draft filter = %d %v, want no drafts", code, out)
	}
	if code, out := e.req(t, "GET", "/a/invoices?status=sent", nil); code != 200 || out["total"].(float64) != 1 {
		t.Fatalf("sent filter = %d %v, want 1", code, out)
	}
	if code, _ := e.req(t, "GET", "/a/invoices?status=bogus", nil); code != 400 {
		t.Fatalf("bad status filter = %d, want 400", code)
	}
	if code, _ := e.req(t, "PATCH", "/a/invoices/"+uuid.NewString()+"/status", fiber.Map{"status": "sent"}); code != 404 {
		t.Fatalf("missing invoice = %d, want 404", code)
	}
	if code, _ := e.req(t, "PATCH", "/b/invoices/"+id+"/status", fiber.Map{"status": "paid"}); code != 404 {
		t.Fatalf("cross-company status change = %d, want 404", code)
	}
}
