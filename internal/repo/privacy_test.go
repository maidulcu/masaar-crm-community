package repo

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestConsentLatestWinsAndIsScoped(t *testing.T) {
	e := setup(t)
	p := NewPrivacyRepo(e.pool)
	ct := e.contact(t, e.a, testPhone("97150"))

	if ok, err := p.MayContact(e.a.ctx, ct.ID, "whatsapp"); err != nil || !ok {
		t.Fatalf("no choice yet should allow contact: %v %v", ok, err)
	}
	for _, st := range []string{domain.ConsentGranted, domain.ConsentWithdrawn} {
		if err := p.RecordConsent(e.a.ctx, &domain.Consent{ContactID: ct.ID, Channel: "whatsapp", Status: st, Source: "manual"}); err != nil {
			t.Fatal(err)
		}
	}
	if ok, _ := p.MayContact(e.a.ctx, ct.ID, "whatsapp"); ok {
		t.Fatal("withdrawn consent still allows contact")
	}
	if ok, _ := p.MayContact(e.a.ctx, ct.ID, "email"); !ok {
		t.Fatal("withdrawing WhatsApp affected email")
	}
	cur, _ := p.CurrentConsent(e.a.ctx, ct.ID)
	if cur["whatsapp"].Status != domain.ConsentWithdrawn || len(cur) != 1 {
		t.Fatalf("current = %+v", cur)
	}
	hist, _ := p.ConsentHistory(e.a.ctx, ct.ID)
	if len(hist) != 2 {
		t.Fatalf("history has %d rows, want 2", len(hist))
	}
	// B can neither record consent for A's contact nor read it.
	if err := p.RecordConsent(e.b.ctx, &domain.Consent{ContactID: ct.ID, Channel: "email", Status: "granted", Source: "manual"}); !isNotFound(err) {
		t.Fatalf("B recorded consent on A's contact: %v", err)
	}
	if hist, _ := p.ConsentHistory(e.b.ctx, ct.ID); len(hist) != 0 {
		t.Fatal("B read A's consent history")
	}
}

func TestEmailOptedOut(t *testing.T) {
	e := setup(t)
	p := NewPrivacyRepo(e.pool)
	ct := e.contact(t, e.a, testPhone("97150"))
	ct.Email = "Client-" + ct.ID.String()[:6] + "@Example.test"
	if err := NewContactRepo(e.pool).Update(e.a.ctx, ct); err != nil {
		t.Fatal(err)
	}
	if out, _ := p.EmailOptedOut(e.a.ctx, strings.ToLower(ct.Email)); out {
		t.Fatal("opted out without a choice")
	}
	_ = p.RecordConsent(e.a.ctx, &domain.Consent{ContactID: ct.ID, Channel: "email", Status: "withdrawn", Source: "manual"})
	if out, _ := p.EmailOptedOut(e.a.ctx, strings.ToLower(ct.Email)); !out {
		t.Fatal("withdrawn email consent not detected (case-insensitive)")
	}
	if out, _ := p.EmailOptedOut(e.b.ctx, ct.Email); out {
		t.Fatal("A's opt-out leaked into company B")
	}
}

func TestExportAndErase(t *testing.T) {
	e := setup(t)
	p := NewPrivacyRepo(e.pool)
	ct := e.contact(t, e.a, testPhone("97150"))
	ld := e.lead(t, e.a, ct.ID)
	must(t, e.pool, `UPDATE leads SET notes = 'Wants a 2BR in Marina, budget 1.2M' WHERE id = $1`, ld.ID)
	threadID := uuid.New()
	must(t, e.pool, `INSERT INTO whatsapp_threads (id, company_id, contact_id, wa_account_id) VALUES ($1, $2, $3, 'wa')`, threadID, e.a.id, ct.ID)
	must(t, e.pool, `INSERT INTO whatsapp_messages (thread_id, direction, body, media_path) VALUES ($1, 'inbound', 'My passport attached', 'abc123')`, threadID)
	d := &domain.Deal{LeadID: ld.ID, Title: "Deal", Stage: domain.DealStageOpen, Amount: 1, Currency: "AED", OwnerID: e.a.user}
	if err := NewDealRepo(e.pool).Create(e.a.ctx, d); err != nil {
		t.Fatal(err)
	}
	inv := testInvoice(d.ID, 100)
	inv.Customer.Name = ct.FullName
	if err := NewInvoiceRepo(e.pool).Issue(e.a.ctx, inv); err != nil {
		t.Fatal(err)
	}

	data, err := p.Export(e.a.ctx, ct.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"contact", "leads", "whatsapp_messages", "invoices"} {
		var rows []map[string]any
		if err := json.Unmarshal(data[key], &rows); err != nil || len(rows) != 1 {
			t.Fatalf("export %s = %s (%v)", key, data[key], err)
		}
	}
	if _, err := p.Export(e.b.ctx, ct.ID); !isNotFound(err) {
		t.Fatalf("B exported A's contact: %v", err)
	}
	if _, err := p.Erase(e.b.ctx, ct.ID, e.b.user); !isNotFound(err) {
		t.Fatalf("B erased A's contact: %v", err)
	}

	paths, err := p.Erase(e.a.ctx, ct.ID, e.a.user)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "abc123" {
		t.Fatalf("media paths = %v", paths)
	}
	got, err := NewContactRepo(e.pool).GetByID(e.a.ctx, ct.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.FullName != "Erased contact" || got.Email != "" || !strings.HasPrefix(got.PhoneWA, "erased:") {
		t.Fatalf("contact after erasure = %+v", got)
	}
	var body string
	var mediaPath *string
	_ = e.pool.QueryRow(e.a.ctx, `SELECT body, media_path FROM whatsapp_messages WHERE thread_id = $1`, threadID).Scan(&body, &mediaPath)
	var notes string
	_ = e.pool.QueryRow(e.a.ctx, `SELECT notes FROM leads WHERE id = $1`, ld.ID).Scan(&notes)
	if body != "[erased]" || mediaPath != nil || notes != "" {
		t.Fatalf("left personal data: body=%q media=%v notes=%q", body, mediaPath, notes)
	}
	if ok, _ := p.MayContact(e.a.ctx, ct.ID, "whatsapp"); ok {
		t.Fatal("erased contact may still be messaged")
	}
	// The issued invoice is a tax record and keeps the printed customer name.
	kept, err := NewInvoiceRepo(e.pool).GetByID(e.a.ctx, inv.ID)
	if err != nil || kept.Customer.Name != ct.FullName {
		t.Fatalf("invoice after erasure = %+v, %v", kept, err)
	}
	if _, err := p.Erase(e.a.ctx, ct.ID, e.a.user); err != ErrContactErased {
		t.Fatalf("second erasure: %v", err)
	}
}

func TestRetentionPurgeIsScoped(t *testing.T) {
	e := setup(t)
	p := NewPrivacyRepo(e.pool)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	mk := func(c company, until time.Time) uuid.UUID {
		id, bank := uuid.New(), uuid.New()
		must(t, e.pool, `INSERT INTO bank_integrations (id, company_id, bank_name) VALUES ($1, $2, 'Bank')`, bank, c.id)
		must(t, e.pool, `INSERT INTO bank_statements (id, company_id, bank_integration_id, file_name, file_size_bytes, file_url,
		                     file_format, uploaded_by, upload_date, retention_until, storage_key)
		                 VALUES ($1, $2, $3, 's.pdf', 1, '', 'pdf', $4, NOW(), $5, $6)`,
			id, c.id, bank, c.user, until, "key-"+id.String()[:8])
		return id
	}
	old := mk(e.a, today.AddDate(0, 0, -1))
	keep := mk(e.a, today.AddDate(1, 0, 0))
	otherCo := mk(e.b, today.AddDate(0, 0, -1))
	t.Cleanup(func() {
		_, _ = e.pool.Exec(e.a.ctx, `DELETE FROM bank_statements WHERE id = ANY($1)`, []uuid.UUID{old, keep, otherCo})
		_, _ = e.pool.Exec(e.a.ctx, `DELETE FROM bank_integrations WHERE company_id = ANY($1)`, []uuid.UUID{e.a.id, e.b.id})
	})

	files, err := p.ExpiredBankStatements(e.a.ctx, today)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].ID != old || files[0].Path == "" {
		t.Fatalf("expired = %+v, want only A's old statement", files)
	}
	if err := p.MarkBankStatementDeleted(e.a.ctx, old); err != nil {
		t.Fatal(err)
	}
	if files, _ := p.ExpiredBankStatements(e.a.ctx, today); len(files) != 0 {
		t.Fatalf("statement still listed after deletion: %+v", files)
	}
}

func TestListExpiringLicences(t *testing.T) {
	e := setup(t)
	cr := NewComplianceRepo(e.pool)
	soon := time.Now().UTC().AddDate(0, 0, 10)
	later := time.Now().UTC().AddDate(0, 3, 0)
	must(t, e.pool, `UPDATE users SET brn = 'BRN1', brn_expiry = $1 WHERE id = $2`, soon, e.a.user)
	must(t, e.pool, `UPDATE company_settings SET trade_license_expiry = $1 WHERE company_id = $2`, later, e.a.id)
	listing := e.listing(t, e.a)
	must(t, e.pool, `UPDATE listings SET status = 'published', emirate = 'Dubai', permit_number = 'P1', permit_expiry = $1, assigned_to = $2 WHERE id = $3`,
		soon, e.a.user, listing)

	items, err := cr.ListExpiring(e.a.ctx, time.Now().UTC().AddDate(0, 0, 30))
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, it := range items {
		kinds[it.EntityType] = true
	}
	if len(items) != 2 || !kinds["brn"] || !kinds["listing_permit"] {
		t.Fatalf("expiring = %+v, want the BRN and the permit (not the licence 3 months out)", items)
	}
	if items, _ := cr.ListExpiring(e.b.ctx, time.Now().UTC().AddDate(0, 0, 30)); len(items) != 0 {
		t.Fatalf("B sees A's licences: %+v", items)
	}
	expiry := soon.Truncate(24 * time.Hour)
	if ok, _ := cr.ClaimAlert(e.a.ctx, "brn", e.a.user, expiry, 30); !ok {
		t.Fatal("first claim failed")
	}
	if ok, _ := cr.ClaimAlert(e.a.ctx, "brn", e.a.user, expiry, 30); ok {
		t.Fatal("alert claimed twice")
	}
	if ok, _ := cr.ClaimAlert(e.a.ctx, "brn", e.a.user, expiry, 7); !ok {
		t.Fatal("7-day alert blocked by the 30-day one")
	}
	if admins, _ := cr.AdminIDs(e.a.ctx); len(admins) != 1 || admins[0] != e.a.user {
		t.Fatalf("admins = %v", admins)
	}
}
