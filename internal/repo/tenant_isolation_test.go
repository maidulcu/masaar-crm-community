package repo

// Multi-company isolation tests. They run against a real PostgreSQL database and are
// skipped unless TEST_DATABASE_URL is set, e.g.
//
//	TEST_DATABASE_URL=postgres://masaar:masaar@127.0.0.1:5432/masaar_test?sslmode=disable go test ./internal/repo/
//
// Use a dedicated database: the test applies all migrations and creates (and removes) two
// throwaway companies. Each test creates data as company A and then proves company B can
// neither see nor change it, and cannot attach its own records to A's.

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type company struct {
	id   uuid.UUID
	user uuid.UUID
	ctx  context.Context
}

type testEnv struct {
	pool *pgxpool.Pool
	a, b company
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping database isolation tests")
	}

	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(sqlDB, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB.Close()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}

	mk := func(name string) company {
		c := company{id: uuid.New(), user: uuid.New()}
		must(t, pool, `INSERT INTO companies (id, name, subdomain) VALUES ($1, $2, $3)`, c.id, name, "t-"+c.id.String()[:8])
		must(t, pool, `INSERT INTO company_settings (company_id, name, vat_number, business_address) VALUES ($1, $2, '', '')`, c.id, name)
		must(t, pool, `INSERT INTO users (id, company_id, name, email, password_hash, role) VALUES ($1, $2, $3, $4, 'x', 'admin')`,
			c.user, c.id, name+" admin", c.id.String()+"@test.local")
		c.ctx = tenant.With(ctx, c.id)
		return c
	}
	e := &testEnv{pool: pool, a: mk("Company A"), b: mk("Company B")}

	t.Cleanup(func() {
		for _, id := range []uuid.UUID{e.a.id, e.b.id} {
			// children first; ignore errors — this is best-effort cleanup of throwaway data
			for _, q := range []string{
				`DELETE FROM whatsapp_messages WHERE thread_id IN (SELECT id FROM whatsapp_threads WHERE company_id = $1)`,
				`DELETE FROM lead_tags WHERE lead_id IN (SELECT id FROM leads WHERE company_id = $1)`,
				`DELETE FROM payment_confirmations WHERE company_id = $1`,
				`DELETE FROM payments WHERE company_id = $1`,
				`DELETE FROM leases WHERE company_id = $1`,
				`DELETE FROM vat_invoices WHERE company_id = $1`,
				`DELETE FROM deals WHERE company_id = $1`,
				`DELETE FROM offers WHERE company_id = $1`,
				`DELETE FROM viewings WHERE company_id = $1`,
				`DELETE FROM listings WHERE company_id = $1`,
				`DELETE FROM users WHERE company_id = $1`,
				`DELETE FROM companies WHERE id = $1`,
			} {
				_, _ = pool.Exec(ctx, q, id)
			}
		}
		pool.Close()
	})
	return e
}

func must(t *testing.T, pool *pgxpool.Pool, q string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

func isNotFound(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) || errors.Is(err, ErrForeignReference) || errors.Is(err, ErrUserNotFound))
}

// ── fixtures ──────────────────────────────────────────────────────────────────

func (e *testEnv) contact(t *testing.T, c company, phone string) *domain.Contact {
	t.Helper()
	ct := &domain.Contact{PhoneWA: phone, FullName: "Contact " + phone, Language: "en"}
	if err := NewContactRepo(e.pool).Create(c.ctx, ct); err != nil {
		t.Fatalf("create contact: %v", err)
	}
	return ct
}

func (e *testEnv) lead(t *testing.T, c company, contactID uuid.UUID) *domain.Lead {
	t.Helper()
	l := &domain.Lead{ContactID: contactID, Stage: domain.StageNew, Source: "web", Currency: "AED"}
	if err := NewLeadRepo(e.pool).Create(c.ctx, l); err != nil {
		t.Fatalf("create lead: %v", err)
	}
	return l
}

func (e *testEnv) listing(t *testing.T, c company) uuid.UUID {
	t.Helper()
	id := uuid.New()
	must(t, e.pool, `INSERT INTO listings (id, company_id, title, property_type, price) VALUES ($1, $2, 'L', 'apartment', 1)`, id, c.id)
	return id
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestFailClosedWithoutTenant(t *testing.T) {
	e := setup(t)
	bare := context.Background()
	if _, err := NewContactRepo(e.pool).List(bare, "", 1, 10); !errors.Is(err, tenant.ErrMissing) {
		t.Fatalf("List without a company must fail closed, got %v", err)
	}
	if _, err := NewLeadRepo(e.pool).KanbanBoard(bare); !errors.Is(err, tenant.ErrMissing) {
		t.Fatalf("KanbanBoard without a company must fail closed, got %v", err)
	}
	if _, err := NewStatsRepo(e.pool).Overview(bare); !errors.Is(err, tenant.ErrMissing) {
		t.Fatalf("Overview without a company must fail closed, got %v", err)
	}
	if _, err := NewCompanySettingsRepo(e.pool).Get(bare); !errors.Is(err, tenant.ErrMissing) {
		t.Fatalf("company settings without a company must fail closed, got %v", err)
	}
}

func TestContactsIsolated(t *testing.T) {
	e := setup(t)
	repo := NewContactRepo(e.pool)
	phone := "+97150" + uuid.NewString()[:7]
	ct := e.contact(t, e.a, phone)

	if _, err := repo.GetByID(e.b.ctx, ct.ID); !isNotFound(err) {
		t.Fatalf("B read A's contact: %v", err)
	}
	if _, err := repo.GetByPhone(e.b.ctx, phone); !isNotFound(err) {
		t.Fatalf("B found A's contact by phone: %v", err)
	}
	res, err := repo.List(e.b.ctx, "", 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range res.Data {
		if c.ID == ct.ID {
			t.Fatal("A's contact appears in B's list")
		}
	}

	hijack := *ct
	hijack.FullName = "hijacked"
	if err := repo.Update(e.b.ctx, &hijack); !isNotFound(err) {
		t.Fatalf("B updated A's contact: %v", err)
	}
	_ = repo.Delete(e.b.ctx, ct.ID)
	_ = repo.UpdateScore(e.b.ctx, ct.ID, 99)
	got, err := repo.GetByID(e.a.ctx, ct.ID)
	if err != nil || got.FullName != ct.FullName || got.LeadScore == 99 {
		t.Fatalf("A's contact was changed or deleted by B: %+v %v", got, err)
	}

	// The same phone number is a different contact in another company.
	other, err := repo.Upsert(e.b.ctx, phone, "B's view")
	if err != nil || other.ID == ct.ID {
		t.Fatalf("per-company phone uniqueness: %v %v", other, err)
	}
}

func TestLeadsIsolatedAndNoCrossReferences(t *testing.T) {
	e := setup(t)
	leads := NewLeadRepo(e.pool)
	ct := e.contact(t, e.a, "+97151"+uuid.NewString()[:7])
	ld := e.lead(t, e.a, ct.ID)

	if _, err := leads.GetByID(e.b.ctx, ld.ID); !isNotFound(err) {
		t.Fatalf("B read A's lead: %v", err)
	}
	board, err := leads.KanbanBoard(e.b.ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range board {
		for _, l := range col {
			if l.ID == ld.ID {
				t.Fatal("A's lead on B's board")
			}
		}
	}

	_ = leads.UpdateStage(e.b.ctx, ld.ID, domain.StageWon, "stolen")
	_ = leads.UpdateNotes(e.b.ctx, ld.ID, "pwned")
	_ = leads.Assign(e.b.ctx, ld.ID, &e.b.user)
	if err := leads.Delete(e.b.ctx, ld.ID); err == nil {
		t.Fatal("B deleted A's lead")
	}
	got, err := leads.GetByID(e.a.ctx, ld.ID)
	if err != nil || got.Stage != domain.StageNew || got.Notes == "pwned" || got.AssignedTo != nil {
		t.Fatalf("A's lead was modified by B: %+v %v", got, err)
	}

	// B cannot create a lead that points at A's contact, nor assign A's agent.
	bad := &domain.Lead{ContactID: ct.ID, Stage: domain.StageNew, Source: "web", Currency: "AED"}
	if err := leads.Create(e.b.ctx, bad); !isNotFound(err) {
		t.Fatalf("B created a lead on A's contact: %v", err)
	}
	bc := e.contact(t, e.b, "+97152"+uuid.NewString()[:7])
	withForeignAgent := &domain.Lead{ContactID: bc.ID, Stage: domain.StageNew, Source: "web", Currency: "AED", AssignedTo: &e.a.user}
	if err := leads.Create(e.b.ctx, withForeignAgent); !isNotFound(err) {
		t.Fatalf("B assigned a lead to A's agent: %v", err)
	}
}

func TestDealsAndInvoicesIsolated(t *testing.T) {
	e := setup(t)
	ct := e.contact(t, e.a, "+97153"+uuid.NewString()[:7])
	ld := e.lead(t, e.a, ct.ID)
	deals := NewDealRepo(e.pool)
	invoices := NewInvoiceRepo(e.pool)

	d := &domain.Deal{LeadID: ld.ID, Title: "A deal", Stage: domain.DealStageOpen, Amount: 100, Currency: "AED", OwnerID: e.a.user}
	if err := deals.Create(e.a.ctx, d); err != nil {
		t.Fatal(err)
	}
	if _, err := deals.GetByID(e.b.ctx, d.ID); !isNotFound(err) {
		t.Fatalf("B read A's deal: %v", err)
	}
	_ = deals.UpdateStage(e.b.ctx, d.ID, domain.DealStageLost)
	_ = deals.Delete(e.b.ctx, d.ID)
	if got, err := deals.GetByID(e.a.ctx, d.ID); err != nil || got.Stage != domain.DealStageOpen {
		t.Fatalf("A's deal modified by B: %+v %v", got, err)
	}
	// B cannot attach a deal to A's lead.
	if err := deals.Create(e.b.ctx, &domain.Deal{LeadID: ld.ID, Title: "x", Stage: domain.DealStageOpen, OwnerID: e.b.user}); !isNotFound(err) {
		t.Fatalf("B created a deal on A's lead: %v", err)
	}

	no, err := invoices.NextInvoiceNo(e.a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	inv := &domain.VATInvoice{DealID: d.ID, InvoiceNo: no, Subtotal: 100, VATRate: 5, Status: domain.InvoiceDraft}
	if err := invoices.Create(e.a.ctx, inv); err != nil {
		t.Fatal(err)
	}
	if _, err := invoices.GetByID(e.b.ctx, inv.ID); !isNotFound(err) {
		t.Fatalf("B read A's invoice: %v", err)
	}
	if list, _ := invoices.ListByDeal(e.b.ctx, d.ID); len(list) != 0 {
		t.Fatal("B listed A's invoices")
	}
	if err := invoices.Create(e.b.ctx, &domain.VATInvoice{DealID: d.ID, InvoiceNo: "X", Subtotal: 1, VATRate: 5, Status: domain.InvoiceDraft}); !isNotFound(err) {
		t.Fatalf("B invoiced A's deal: %v", err)
	}
	// Invoice numbering is per company: B's first number must not depend on A's.
	bNo, err := invoices.NextInvoiceNo(e.b.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if bNo[len(bNo)-4:] != "0001" {
		t.Fatalf("B's first invoice number should restart at 0001, got %s", bNo)
	}
}

func TestWhatsAppAndNotificationsIsolated(t *testing.T) {
	e := setup(t)
	wa := NewWhatsAppRepo(e.pool)
	ct := e.contact(t, e.a, "+97154"+uuid.NewString()[:7])

	thread, err := wa.UpsertThread(e.a.ctx, ct.ID, "acct")
	if err != nil {
		t.Fatal(err)
	}
	msg := &domain.WhatsAppMessage{ThreadID: thread.ID, Direction: domain.DirectionInbound, Body: "secret", WAMessageID: "wamid-" + uuid.NewString()}
	if err := wa.SaveMessage(e.a.ctx, msg); err != nil {
		t.Fatal(err)
	}

	if _, err := wa.GetThread(e.b.ctx, thread.ID); !isNotFound(err) {
		t.Fatalf("B read A's thread: %v", err)
	}
	if msgs, _ := wa.GetMessages(e.b.ctx, thread.ID, 50); len(msgs) != 0 {
		t.Fatal("B read A's WhatsApp messages")
	}
	if threads, _ := wa.ListThreads(e.b.ctx, "", nil, 1, 100); len(threads) != 0 {
		t.Fatal("A's thread in B's inbox")
	}
	// B cannot open a thread for A's contact or inject a message into A's thread.
	if _, err := wa.UpsertThread(e.b.ctx, ct.ID, "acct"); !isNotFound(err) {
		t.Fatalf("B opened a thread on A's contact: %v", err)
	}
	inject := &domain.WhatsAppMessage{ThreadID: thread.ID, Direction: domain.DirectionInbound, Body: "injected", WAMessageID: "wamid-" + uuid.NewString()}
	if err := wa.SaveMessage(e.b.ctx, inject); !isNotFound(err) {
		t.Fatalf("B injected a message into A's thread: %v", err)
	}

	// Notifications: scoped to company and user; B cannot notify A's user.
	notes := NewNotificationRepo(e.pool)
	n := &domain.Notification{ID: uuid.New(), UserID: e.a.user, Type: "t", Title: "for A", Body: "b"}
	if err := notes.Create(e.a.ctx, n); err != nil {
		t.Fatal(err)
	}
	if res, _ := notes.ListByUser(e.b.ctx, e.a.user, 1, 50); res != nil && len(res.Data) != 0 {
		t.Fatal("B read A's user's notifications")
	}
	if err := notes.Create(e.b.ctx, &domain.Notification{ID: uuid.New(), UserID: e.a.user, Type: "t", Title: "spam", Body: "b"}); err != nil {
		t.Fatal(err) // the insert is a silent no-op: it must not create a row for A's user
	}
	res, err := notes.ListByUser(e.a.ctx, e.a.user, 1, 50)
	if err != nil || len(res.Data) != 1 {
		t.Fatalf("A should have exactly its own notification, got %v %v", res, err)
	}
}

func TestViewingsAndOffersIsolated(t *testing.T) {
	e := setup(t)
	ct := e.contact(t, e.a, "+97155"+uuid.NewString()[:7])
	listing := e.listing(t, e.a)
	viewings := NewViewingRepo(e.pool)
	offers := NewOfferRepo(e.pool)

	v := &domain.Viewing{ContactID: ct.ID, ListingID: &listing, AgentID: &e.a.user, ScheduledAt: timeNow()}
	if err := viewings.Create(e.a.ctx, v); err != nil {
		t.Fatal(err)
	}
	if _, err := viewings.GetByID(e.b.ctx, v.ID); !isNotFound(err) {
		t.Fatalf("B read A's viewing: %v", err)
	}
	_ = viewings.UpdateStatus(e.b.ctx, v.ID, domain.ViewingCancelled)
	_ = viewings.Delete(e.b.ctx, v.ID)
	if got, err := viewings.GetByID(e.a.ctx, v.ID); err != nil || got.Status == domain.ViewingCancelled {
		t.Fatalf("A's viewing modified by B: %+v %v", got, err)
	}
	bad := &domain.Viewing{ContactID: ct.ID, ScheduledAt: timeNow()}
	if err := viewings.Create(e.b.ctx, bad); !isNotFound(err) {
		t.Fatalf("B booked a viewing for A's contact: %v", err)
	}

	o := &domain.Offer{ListingID: listing, ContactID: ct.ID, OfferAmount: 10}
	if err := offers.Create(e.a.ctx, o); err != nil {
		t.Fatal(err)
	}
	if _, err := offers.GetByID(e.b.ctx, o.ID); !isNotFound(err) {
		t.Fatalf("B read A's offer: %v", err)
	}
	_ = offers.UpdateStatus(e.b.ctx, o.ID, domain.OfferRejected)
	_ = offers.Delete(e.b.ctx, o.ID)
	if got, err := offers.GetByID(e.a.ctx, o.ID); err != nil || got.Status == domain.OfferRejected {
		t.Fatalf("A's offer modified by B: %+v %v", got, err)
	}
	if err := offers.Create(e.b.ctx, &domain.Offer{ListingID: listing, ContactID: ct.ID, OfferAmount: 1}); !isNotFound(err) {
		t.Fatalf("B made an offer on A's listing: %v", err)
	}
}

func TestUsersAdminScopedToCompany(t *testing.T) {
	e := setup(t)
	users := NewUserRepo(e.pool)
	if err := users.UpdateUser(e.b.ctx, e.b.id, e.a.user, "pwned", domain.RoleViewer); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("B admin changed A's user: %v", err)
	}
	if err := users.SetActive(e.b.ctx, e.b.id, e.a.user, false); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("B admin deactivated A's user: %v", err)
	}
	if err := users.Delete(e.b.ctx, e.b.id, e.a.user); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("B admin deleted A's user: %v", err)
	}
	u, err := users.FindByID(e.a.ctx, e.a.user)
	if err != nil || u.Role != domain.RoleAdmin || !u.IsActive {
		t.Fatalf("A's user was altered: %+v %v", u, err)
	}
}

func TestSettingsIsolated(t *testing.T) {
	e := setup(t)
	cs := NewCompanySettingsRepo(e.pool)
	a, err := cs.Get(e.a.ctx)
	if err != nil || a.Name != "Company A" {
		t.Fatalf("A should see its own settings, got %+v %v", a, err)
	}
	b, err := cs.Get(e.b.ctx)
	if err != nil || b.Name != "Company B" {
		t.Fatalf("B should see its own settings, got %+v %v", b, err)
	}
	// B cannot overwrite A's settings by id.
	hijack := *a
	hijack.BankIBAN = "AE-HIJACKED"
	_ = cs.Update(e.b.ctx, &hijack, &e.b.user)
	if again, _ := cs.Get(e.a.ctx); again.BankIBAN == "AE-HIJACKED" {
		t.Fatal("B overwrote A's company settings")
	}

	api := NewSettingsRepo(e.pool)
	if err := api.Update(e.a.ctx, "bos24_api_token", "secret-a", &e.a.user); err != nil {
		t.Fatal(err)
	}
	if s, err := api.Get(e.b.ctx, "bos24_api_token"); err == nil {
		t.Fatalf("B read A's integration token: %+v", s)
	}
	if err := api.Update(e.b.ctx, "bos24_api_token", "secret-b", &e.b.user); err != nil {
		t.Fatal(err)
	}
	if s, _ := api.Get(e.a.ctx, "bos24_api_token"); s == nil || s.SettingValue != "secret-a" {
		t.Fatalf("A's token was overwritten: %+v", s)
	}
}

func TestAuditLogScoped(t *testing.T) {
	e := setup(t)
	audit := NewAuditLogRepo(e.pool)
	entity := uuid.New()
	audit.Log(e.a.ctx, e.a.user, AuditCreate, AuditContact, entity, nil)

	resA, err := audit.List(e.a.ctx, AuditLogFilter{EntityID: &entity})
	if err != nil || resA.Total != 1 {
		t.Fatalf("A should see its audit entry: %+v %v", resA, err)
	}
	resB, err := audit.List(e.b.ctx, AuditLogFilter{EntityID: &entity})
	if err != nil || resB.Total != 0 {
		t.Fatalf("B must not see A's audit entry: %+v %v", resB, err)
	}
	// An unauthenticated context (login events) is attributed to the actor's own company.
	audit.Log(context.Background(), e.b.user, AuditLogin, AuditUser, e.b.user, nil)
	if res, _ := audit.List(e.b.ctx, AuditLogFilter{ActorID: &e.b.user, Action: AuditLogin}); res == nil || res.Total == 0 {
		t.Fatal("login audit entry was not attributed to the actor's company")
	}
}

func TestRentalDomainIsolated(t *testing.T) {
	e := setup(t)
	props := NewRentalPropertyRepo(e.pool)
	tenants := NewTenantRepo(e.pool)
	leases := NewLeaseRepo(e.pool)

	p := &domain.RentalProperty{Name: "Tower A", PropertyType: "apartment", Currency: "AED", Status: "active", UnitsCount: 1}
	if err := props.Create(e.a.ctx, p); err != nil {
		t.Fatal(err)
	}
	tn := &domain.Tenant{FullNameEN: "Tenant A", IDType: "emirates_id", Status: "active"}
	if err := tenants.Create(e.a.ctx, tn); err != nil {
		t.Fatal(err)
	}

	if _, err := props.GetByID(e.b.ctx, p.ID); !isNotFound(err) {
		t.Fatalf("B read A's property: %v", err)
	}
	if _, err := tenants.GetByID(e.b.ctx, tn.ID); !isNotFound(err) {
		t.Fatalf("B read A's tenant: %v", err)
	}
	_ = props.Delete(e.b.ctx, p.ID)
	_ = tenants.Delete(e.b.ctx, tn.ID)
	if _, err := props.GetByID(e.a.ctx, p.ID); err != nil {
		t.Fatalf("A's property deleted by B: %v", err)
	}
	if _, err := tenants.GetByID(e.a.ctx, tn.ID); err != nil {
		t.Fatalf("A's tenant deleted by B: %v", err)
	}

	// A client-supplied company id is ignored on create.
	spoof := &domain.RentalProperty{CompanyID: e.a.id, Name: "Spoofed", PropertyType: "villa", Currency: "AED", Status: "active", UnitsCount: 1}
	if err := props.Create(e.b.ctx, spoof); err != nil {
		t.Fatal(err)
	}
	if spoof.CompanyID != e.b.id {
		t.Fatalf("create must use the caller's company, got %s", spoof.CompanyID)
	}

	// B cannot create a lease over A's property/tenant.
	l := &domain.Lease{PropertyID: p.ID, TenantID: tn.ID, MonthlyRent: 1, Currency: "AED", Status: "draft",
		StartDate: timeNow(), EndDate: timeNow().AddDate(1, 0, 0)}
	if err := leases.Create(e.b.ctx, l); !errors.Is(err, ErrForeignReference) {
		t.Fatalf("B created a lease over A's property: %v", err)
	}
}

func TestStatsOnlyCountOwnData(t *testing.T) {
	e := setup(t)
	e.contact(t, e.a, "+97156"+uuid.NewString()[:7])
	e.contact(t, e.a, "+97157"+uuid.NewString()[:7])
	stats := NewStatsRepo(e.pool)
	a, err := stats.Overview(e.a.ctx)
	if err != nil || a.TotalContacts != 2 {
		t.Fatalf("A should count its 2 contacts, got %+v %v", a, err)
	}
	b, err := stats.Overview(e.b.ctx)
	if err != nil || b.TotalContacts != 0 {
		t.Fatalf("B should count 0 contacts, got %+v %v", b, err)
	}
}

func timeNow() time.Time { return time.Now() }
