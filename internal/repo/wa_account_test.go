package repo

import (
	"testing"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestWAAccountTenantIsolation(t *testing.T) {
	e := setup(t)
	accounts := NewWAAccountRepo(e.pool)

	a := &domain.WAAccount{
		Provider:      domain.WAProviderCloudAPI,
		Status:        domain.WAAccountConnected,
		PhoneE164:     "+971500000001",
		PhoneNumberID: "pn-" + uuid.NewString(),
		DisplayName:   "Sales line",
	}
	if err := accounts.Create(e.a.ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.CompanyID != e.a.id {
		t.Fatalf("company = %s, want %s", a.CompanyID, e.a.id)
	}

	// Company B cannot see or read A's account.
	listB, err := accounts.List(e.b.ctx)
	if err != nil {
		t.Fatalf("list B: %v", err)
	}
	if len(listB) != 0 {
		t.Fatalf("B sees %d accounts, want 0", len(listB))
	}
	if _, err := accounts.Get(e.b.ctx, a.ID); err == nil {
		t.Fatal("B read A's account")
	}

	// B cannot change its status either.
	if err := accounts.UpdateStatus(e.b.ctx, a.ID, domain.WAAccountDegraded, "test"); err == nil {
		t.Fatal("B changed A's account status")
	}

	listA, err := accounts.List(e.a.ctx)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if len(listA) != 1 || listA[0].ID != a.ID || listA[0].DisplayName != "Sales line" {
		t.Fatalf("A accounts = %+v", listA)
	}
}

func TestEnsureDeploymentAccountIsIdempotent(t *testing.T) {
	e := setup(t)
	accounts := NewWAAccountRepo(e.pool)
	phoneNumberID := "pn-" + uuid.NewString()

	first, err := accounts.EnsureDeploymentAccount(e.a.ctx, e.a.id, phoneNumberID, "")
	if err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	second, err := accounts.EnsureDeploymentAccount(e.a.ctx, e.a.id, phoneNumberID, "Main number")
	if err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("second ensure created %s, want %s", second.ID, first.ID)
	}
	if second.DisplayName != "Main number" {
		t.Fatalf("display name = %q, want %q", second.DisplayName, "Main number")
	}

	list, err := accounts.List(e.a.ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("accounts = %d, want 1", len(list))
	}

	// The same legacy number may exist under another company; the rows stay separate.
	other, err := accounts.EnsureDeploymentAccount(e.b.ctx, e.b.id, phoneNumberID, "")
	if err != nil {
		t.Fatalf("ensure for B: %v", err)
	}
	if other.ID == first.ID {
		t.Fatal("B resolved A's account")
	}
}

func TestResolveWAAccountByPhoneNumberID(t *testing.T) {
	e := setup(t)
	accounts := NewWAAccountRepo(e.pool)
	phoneNumberID := "pn-" + uuid.NewString()

	created, err := accounts.EnsureDeploymentAccount(e.a.ctx, e.a.id, phoneNumberID, "Inbox")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}

	// Webhooks carry no tenant; the caller supplies the deployment company and the row decides
	// the rest.
	got, err := accounts.ResolveByPhoneNumberID(t.Context(), e.a.id, phoneNumberID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != created.ID || got.CompanyID != e.a.id {
		t.Fatalf("resolved %s/%s, want %s/%s", got.ID, got.CompanyID, created.ID, e.a.id)
	}

	if _, err := accounts.ResolveByPhoneNumberID(t.Context(), e.a.id, "pn-"+uuid.NewString()); err == nil {
		t.Fatal("resolved a phone number that does not exist")
	}
	if _, err := accounts.ResolveByPhoneNumberID(t.Context(), e.b.id, phoneNumberID); err == nil {
		t.Fatal("B resolved A's phone number")
	}
}

func TestUpsertThreadLinksAccount(t *testing.T) {
	e := setup(t)
	phoneNumberID := "pn-" + uuid.NewString()
	account, err := NewWAAccountRepo(e.pool).EnsureDeploymentAccount(e.a.ctx, e.a.id, phoneNumberID, "")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}

	contact := e.contact(t, e.a, testPhone("97154"))
	thread, err := NewWhatsAppRepo(e.pool).UpsertThread(e.a.ctx, contact.ID, phoneNumberID)
	if err != nil {
		t.Fatalf("upsert thread: %v", err)
	}

	var accountID *uuid.UUID
	if err := e.pool.QueryRow(e.a.ctx, `SELECT account_id FROM whatsapp_threads WHERE id = $1`, thread.ID).Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	if accountID == nil || *accountID != account.ID {
		t.Fatalf("thread account = %v, want %s", accountID, account.ID)
	}
}

func TestLinkThreadsToAccount(t *testing.T) {
	e := setup(t)
	accounts := NewWAAccountRepo(e.pool)
	phoneNumberID := "pn-" + uuid.NewString()

	account, err := accounts.EnsureDeploymentAccount(e.a.ctx, e.a.id, phoneNumberID, "")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}

	// Two threads with the legacy key and one already attached to another account.
	contact := uuid.New()
	must(t, e.pool, `INSERT INTO contacts (id, company_id, phone_wa, full_name) VALUES ($1, $2, $3, 'Customer')`,
		contact, e.a.id, "+971500000002")
	for i := 0; i < 2; i++ {
		must(t, e.pool, `INSERT INTO whatsapp_threads (id, company_id, contact_id, wa_account_id)
			VALUES ($1, $2, $3, $4)`, uuid.New(), e.a.id, contact, phoneNumberID)
	}

	linked, err := accounts.LinkThreads(e.a.ctx, account.ID)
	if err != nil {
		t.Fatalf("link threads: %v", err)
	}
	if linked != 2 {
		t.Fatalf("linked %d threads, want 2", linked)
	}

	var unlinked int
	if err := e.pool.QueryRow(e.a.ctx, `SELECT COUNT(*) FROM whatsapp_threads
		WHERE company_id = $1 AND wa_account_id = $2 AND account_id IS DISTINCT FROM $3`,
		e.a.id, phoneNumberID, account.ID).Scan(&unlinked); err != nil {
		t.Fatal(err)
	}
	if unlinked != 0 {
		t.Fatalf("%d threads still unlinked", unlinked)
	}

	// Running it again is a no-op.
	again, err := accounts.LinkThreads(e.a.ctx, account.ID)
	if err != nil || again != 0 {
		t.Fatalf("second link = %d, %v; want 0, nil", again, err)
	}

	// B cannot link A's account.
	if _, err := accounts.LinkThreads(e.b.ctx, account.ID); err == nil {
		t.Fatal("B linked threads to A's account")
	}
}

func TestWAAccountDefaultsAndStatus(t *testing.T) {
	e := setup(t)
	accounts := NewWAAccountRepo(e.pool)

	a := &domain.WAAccount{Provider: domain.WAProviderBSP, BSPSlug: "partner"}
	if err := accounts.Create(e.a.ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.Status != domain.WAAccountPending {
		t.Fatalf("status = %q, want pending", a.Status)
	}

	if err := accounts.UpdateStatus(e.a.ctx, a.ID, domain.WAAccountConnected, ""); err != nil {
		t.Fatalf("update status: %v", err)
	}
	got, err := accounts.Get(e.a.ctx, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.WAAccountConnected || got.LastSeenAt == nil {
		t.Fatalf("account = %+v", got)
	}

	if err := accounts.UpdateStatus(e.a.ctx, uuid.New(), domain.WAAccountConnected, ""); err == nil {
		t.Fatal("updated an account that does not exist")
	}
}

func TestWAAccountPhoneIsUniquePerCompany(t *testing.T) {
	e := setup(t)
	accounts := NewWAAccountRepo(e.pool)

	for i := 0; i < 2; i++ {
		err := accounts.Create(e.a.ctx, &domain.WAAccount{
			Provider:  domain.WAProviderBSP,
			Status:    domain.WAAccountPending,
			PhoneE164: "+971500000003",
		})
		if i == 0 && err != nil {
			t.Fatalf("first create: %v", err)
		}
		if i == 1 && err == nil {
			t.Fatal("created two accounts with the same phone in one company")
		}
	}

	// A different company may use the same display phone.
	if err := accounts.Create(e.b.ctx, &domain.WAAccount{
		Provider:  domain.WAProviderBSP,
		Status:    domain.WAAccountPending,
		PhoneE164: "+971500000003",
	}); err != nil {
		t.Fatalf("B create: %v", err)
	}
}
