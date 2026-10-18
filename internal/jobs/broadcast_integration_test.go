package jobs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/testdb"
)

type recordingWA struct {
	mu   sync.Mutex
	sent map[string][]string
}

func (r *recordingWA) IsConfigured() bool { return true }
func (r *recordingWA) SendTemplate(_ context.Context, to, name, lang string, params []string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent[to] = params
	return "wamid.B" + to, nil
}

// A broadcast to 500 contacts reaches exactly those who granted WhatsApp consent, never one
// who opted out, including one who replied STOP after the broadcast started.
func TestBroadcastNeverMessagesOptedOut(t *testing.T) {
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
	cid := uuid.New()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`INSERT INTO companies (id, name, subdomain) VALUES ($1, 'BC Co', $2)`, cid, "bc-"+cid.String()[:8])
	exec(`INSERT INTO company_settings (company_id, name, vat_number, business_address, wa_broadcast_per_minute) VALUES ($1, 'BC Co', '', '', 1000)`, cid)
	t.Cleanup(func() {
		for _, q := range []string{`DELETE FROM whatsapp_broadcasts WHERE company_id = $1`, `DELETE FROM contact_consents WHERE company_id = $1`,
			`DELETE FROM lead_tags WHERE lead_id IN (SELECT id FROM leads WHERE company_id = $1)`, `DELETE FROM leads WHERE company_id = $1`,
			`DELETE FROM contacts WHERE company_id = $1`, `DELETE FROM company_settings WHERE company_id = $1`, `DELETE FROM companies WHERE id = $1`} {
			_, _ = pool.Exec(ctx, q, cid)
		}
	})

	// 500 contacts: 400 granted, 50 withdrew, 50 never asked. All carry the tag "vip".
	prefix := fmt.Sprintf("+9715%d", 10000+int(cid[0])%89999)
	var granted, optedOut []string
	for i := 0; i < 500; i++ {
		id, lead := uuid.New(), uuid.New()
		phone := fmt.Sprintf("%s%03d", prefix, i)
		exec(`INSERT INTO contacts (id, company_id, phone_wa, full_name, language) VALUES ($1, $2, $3, $4, 'en')`, id, cid, phone, fmt.Sprintf("Client %d Test", i))
		exec(`INSERT INTO leads (id, company_id, contact_id, stage, source, currency) VALUES ($1, $2, $3, 'new', 'web', 'AED')`, lead, cid, id)
		exec(`INSERT INTO lead_tags (lead_id, tag) VALUES ($1, 'vip')`, lead)
		switch {
		case i < 400:
			exec(`INSERT INTO contact_consents (company_id, contact_id, channel, status, source) VALUES ($1, $2, 'whatsapp', 'granted', 'manual')`, cid, id)
			granted = append(granted, phone)
		case i < 450:
			exec(`INSERT INTO contact_consents (company_id, contact_id, channel, status, source) VALUES ($1, $2, 'whatsapp', 'granted', 'manual')`, cid, id)
			exec(`INSERT INTO contact_consents (company_id, contact_id, channel, status, source, created_at) VALUES ($1, $2, 'whatsapp', 'withdrawn', 'inbound_message', NOW() + INTERVAL '1 second')`, cid, id)
			optedOut = append(optedOut, phone)
		}
	}

	tctx := tenant.With(ctx, cid)
	br := repo.NewBroadcastRepo(pool)
	if a, err := br.Preview(tctx, repo.Segment{Tag: "vip"}); err != nil || a.Total != 500 || a.Eligible != 400 || a.OptedOut != 50 || a.NoConsent != 50 {
		t.Fatalf("preview %+v %v", a, err)
	}
	b := &repo.Broadcast{Name: "Launch", TemplateName: "new_launch", Language: "en", Params: []string{"{{first_name}}", "Marina Vista"}, Segment: repo.Segment{Tag: "vip"}}
	if err := br.Create(tctx, b, nil); err != nil {
		t.Fatal(err)
	}
	if err := br.Start(tctx, b.ID); err != nil {
		t.Fatal(err)
	}
	// Client 7 replies STOP after the broadcast started.
	exec(`INSERT INTO contact_consents (company_id, contact_id, channel, status, source, created_at)
		SELECT $1, id, 'whatsapp', 'withdrawn', 'inbound_message', NOW() + INTERVAL '2 seconds' FROM contacts WHERE company_id = $1 AND phone_wa = $2`, cid, granted[7])

	wa := &recordingWA{sent: map[string][]string{}}
	job := BroadcastSender(br, wa)
	for i := 0; i < 2; i++ {
		if _, err := job.PerCompany(tctx, domain.CompanyLocale{CompanyID: cid}); err != nil {
			t.Fatal(err)
		}
	}
	if len(wa.sent) != 399 {
		t.Fatalf("sent %d messages, want 399", len(wa.sent))
	}
	for _, p := range append(optedOut, granted[7]) {
		if _, ok := wa.sent[p]; ok {
			t.Fatalf("messaged %s, who opted out", p)
		}
	}
	if params := wa.sent[granted[0]]; len(params) != 2 || params[0] != "Client" || params[1] != "Marina Vista" {
		t.Fatalf("params %v", params)
	}
	got, _ := br.Get(tctx, b.ID)
	if got.Status != "done" || got.Counts["sent"] != 399 || got.Counts["skipped"] != 101 {
		t.Fatalf("broadcast %+v", got)
	}
	// A delivery receipt moves the recipient forward; a stale one does not move it back.
	if ok, _ := br.ApplyStatus(tctx, "wamid.B"+granted[0], "read", ""); !ok {
		t.Fatal("read receipt not applied")
	}
	if ok, _ := br.ApplyStatus(tctx, "wamid.B"+granted[0], "delivered", ""); ok {
		t.Fatal("a stale receipt moved the status back")
	}
	if list, _ := br.Recipients(tctx, b.ID, "skipped", 500, 0); len(list) != 101 || !strings.Contains("opted_out no_consent", list[0].SkipReason) {
		t.Fatalf("skipped recipients: %d %+v", len(list), list[0])
	}
}
