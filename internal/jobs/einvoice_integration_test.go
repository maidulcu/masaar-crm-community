package jobs

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/einvoice"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/testdb"
)

// With the mock provider, a submitted e-invoice is accepted; one whose XML carries the
// reject marker is rejected, with the reason recorded and the issuer and submitter told once.
func TestEInvoiceStatusWithMockProvider(t *testing.T) {
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
	defer pool.Close()
	cid, issuer, submitter, contact, lead := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`INSERT INTO companies (id, name, subdomain) VALUES ($1, 'EI Co', $2)`, cid, "ei-"+cid.String()[:8])
	exec(`INSERT INTO company_settings (company_id, name, vat_number, business_address, einvoice_provider) VALUES ($1, 'EI Co', '', '', 'mock')`, cid)
	for _, u := range []uuid.UUID{issuer, submitter} {
		exec(`INSERT INTO users (id, company_id, name, email, password_hash, role) VALUES ($1, $2, 'U', $3, 'x', 'admin')`, u, cid, u.String()+"@test.local")
	}
	exec(`INSERT INTO contacts (id, company_id, phone_wa, full_name, language) VALUES ($1, $2, $3, 'Buyer', 'en')`, contact, cid, "+9715"+cid.String()[:8])
	exec(`INSERT INTO leads (id, company_id, contact_id, stage, source, currency) VALUES ($1, $2, $3, 'new', 'web', 'AED')`, lead, cid, contact)
	t.Cleanup(func() {
		for _, q := range []string{`DELETE FROM notifications WHERE company_id = $1`, `DELETE FROM einvoices WHERE company_id = $1`,
			`DELETE FROM vat_invoices WHERE company_id = $1`, `DELETE FROM deals WHERE company_id = $1`, `DELETE FROM leads WHERE company_id = $1`,
			`DELETE FROM contacts WHERE company_id = $1`, `DELETE FROM users WHERE company_id = $1`,
			`DELETE FROM company_settings WHERE company_id = $1`, `DELETE FROM companies WHERE id = $1`} {
			_, _ = pool.Exec(ctx, q, cid)
		}
	})

	tctx := tenant.With(ctx, cid)
	deal := &domain.Deal{LeadID: lead, Title: "Deal", Stage: domain.DealStageOpen, Amount: 100, Currency: "AED", OwnerID: issuer}
	if err := repo.NewDealRepo(pool).Create(tctx, deal); err != nil {
		t.Fatal(err)
	}
	invoices, store := repo.NewInvoiceRepo(pool), repo.NewEInvoiceRepo(pool)
	mock, _ := einvoice.Lookup("mock")
	submit := func(xml string) uuid.UUID {
		t.Helper()
		inv := &domain.VATInvoice{Kind: domain.KindInvoice, DealID: deal.ID, Currency: "AED", Subtotal: 100, VATRate: 0.05, VATAmount: 5, Total: 105,
			Supplier: domain.InvoiceParty{Name: "EI Co"}, Customer: domain.InvoiceParty{Name: "Buyer"}, CreatedBy: &issuer,
			Lines: []domain.InvoiceLine{{Description: "Fee", Quantity: 1, UnitPrice: 100, VATCategory: "standard", VATRate: 0.05, NetAmount: 100, VATAmount: 5}}}
		if err := invoices.Issue(tctx, inv); err != nil {
			t.Fatal(err)
		}
		ref, err := mock.Submit(tctx, einvoice.Submission{Kind: einvoice.KindInvoice, Number: inv.InvoiceNo, XML: []byte(xml)})
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Save(tctx, &repo.EInvoice{InvoiceID: inv.ID, Provider: "mock", Status: einvoice.StatusSubmitted, ProviderRef: ref,
			XML: xml, SubmittedBy: &submitter}); err != nil {
			t.Fatal(err)
		}
		return inv.ID
	}
	ok := submit("<Invoice><Note>thanks</Note></Invoice>")
	bad := submit("<Invoice><Note>" + einvoice.RejectMarker + "</Note></Invoice>")

	job := EInvoiceStatus(store, NewNotifier(repo.NewNotificationRepo(pool), repo.NewUserRepo(pool), nil, nil))
	co := domain.CompanyLocale{CompanyID: cid, Country: "AE", Currency: "AED", Timezone: "Asia/Dubai"}
	for round, want := range []int{2, 0} {
		if n, err := job.PerCompany(tctx, co); err != nil || n != want {
			t.Fatalf("round %d: decided %d (%v), want %d", round, n, err, want)
		}
	}
	if e, _ := store.Get(tctx, ok); e.Status != einvoice.StatusAccepted || e.DecidedAt == nil {
		t.Errorf("accepted: %+v", e)
	}
	if e, _ := store.Get(tctx, bad); e.Status != einvoice.StatusRejected || e.Reason == "" {
		t.Errorf("rejected: %+v", e)
	}
	for _, u := range []uuid.UUID{issuer, submitter} {
		var c int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND type = 'einvoice_rejected' AND data::text LIKE '%' || $2 || '%'`,
			u, bad.String()).Scan(&c); err != nil || c != 1 {
			t.Errorf("rejection notifications for %s: %d (%v), want 1", u, c, err)
		}
	}
}
