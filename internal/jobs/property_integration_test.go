package jobs

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/testdb"
)

type fakeMail struct{ sent []*domain.EmailHistory }

func (f *fakeMail) IsConfigured() bool { return true }
func (f *fakeMail) Send(e *domain.EmailHistory) error {
	f.sent = append(f.sent, e)
	return nil
}

type fakeWA struct{ sent [][]string }

func (f *fakeWA) IsConfigured() bool { return true }
func (f *fakeWA) SendTemplate(_ context.Context, to, name, lang string, params []string) (string, error) {
	f.sent = append(f.sent, append([]string{to, name, lang}, params...))
	return "wamid.1", nil
}

func TestPropertyJobs(t *testing.T) {
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

	// 10:00 in Dubai, so the reminder and deposit windows are open.
	fixed := time.Date(2026, 10, 14, 6, 0, 0, 0, time.UTC)
	clock = func() time.Time { return fixed }
	defer func() { clock = time.Now }()
	today := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)

	cid, admin, agent := uuid.New(), uuid.New(), uuid.New()
	prop, ten, lease, optedTen, lease2 := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	phone, optedPhone := "+97150"+cid.String()[:7], "+97152"+cid.String()[:7]
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`INSERT INTO companies (id, name, subdomain) VALUES ($1, 'PM Co', $2)`, cid, "pm-"+cid.String()[:8])
	exec(`INSERT INTO company_settings (company_id, name, vat_number, business_address, timezone, rent_reminders_enabled,
	      rent_reminder_wa_template) VALUES ($1, 'PM Co', '', '', 'Asia/Dubai', true, 'rent_due')`, cid)
	exec(`INSERT INTO users (id, company_id, name, email, password_hash, role) VALUES ($1, $2, 'Admin', $3, 'x', 'admin'),
	      ($4, $2, 'Agent', $5, 'x', 'agent')`, admin, cid, admin.String()+"@test.local", agent, agent.String()+"@test.local")
	exec(`INSERT INTO rental_properties (id, company_id, name, property_type) VALUES ($1, $2, 'Marina Tower', 'apartment')`, prop, cid)
	exec(`INSERT INTO tenants (id, company_id, full_name_en, id_type, email, phone_wa) VALUES
	      ($1, $2, 'Omar', 'emirates_id', 'omar@example.test', $3), ($4, $2, 'Sara', 'emirates_id', 'sara@example.test', $5)`,
		ten, cid, phone, optedTen, optedPhone)
	// Ends in 85 days: the 90-day alert and the rent-notice deadline alert are due.
	exec(`INSERT INTO leases (id, company_id, property_id, tenant_id, start_date, end_date, monthly_rent, payment_frequency,
	      agent_id, ejari_expiry, status) VALUES ($1, $2, $3, $4, $5::date - 280, $5::date + 85, 10000, 'monthly', $6, $5::date + 5, 'active'),
	      ($7, $2, $3, $8, $5::date - 280, $5::date + 300, 9000, 'monthly', $6, NULL, 'active')`,
		lease, cid, prop, ten, today, agent, lease2, optedTen)
	exec(`INSERT INTO payments (company_id, lease_id, amount, due_date, payment_method, status) VALUES
	      ($1, $2, 10000, $3::date + 2, 'transfer', 'pending'), ($1, $4, 9000, $3::date + 1, 'transfer', 'pending')`,
		cid, lease, today, lease2)
	// Sara is also a contact who withdrew WhatsApp consent.
	contact := uuid.New()
	exec(`INSERT INTO contacts (id, company_id, phone_wa, full_name, language) VALUES ($1, $2, $3, 'Sara', 'en')`, contact, cid, optedPhone)
	exec(`INSERT INTO contact_consents (company_id, contact_id, channel, status, source) VALUES ($1, $2, 'whatsapp', 'withdrawn', 'manual')`, cid, contact)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM payment_reminders WHERE company_id = $1`, `DELETE FROM cheques WHERE company_id = $1`,
			`DELETE FROM payments WHERE company_id = $1`, `DELETE FROM leases WHERE company_id = $1`,
			`DELETE FROM tenants WHERE company_id = $1`, `DELETE FROM rental_properties WHERE company_id = $1`,
			`DELETE FROM contact_consents WHERE company_id = $1`, `DELETE FROM contacts WHERE company_id = $1`,
			`DELETE FROM notifications WHERE company_id = $1`, `DELETE FROM compliance_alerts WHERE company_id = $1`,
			`DELETE FROM users WHERE company_id = $1`, `DELETE FROM company_settings WHERE company_id = $1`,
			`DELETE FROM companies WHERE id = $1`,
		} {
			_, _ = pool.Exec(ctx, q, cid)
		}
	})

	tctx := tenant.With(ctx, cid)
	co := domain.CompanyLocale{CompanyID: cid, Country: "AE", Currency: "AED", Timezone: "Asia/Dubai"}
	ops, compliance, cheques := repo.NewLeaseOpsRepo(pool), repo.NewComplianceRepo(pool), repo.NewChequeRepo(pool)
	n := NewNotifier(repo.NewNotificationRepo(pool), repo.NewUserRepo(pool), nil, nil)
	mail, wa := &fakeMail{}, &fakeWA{}

	// A 12-cheque book; the first cheque is dated in the past, so it is due for deposit.
	book, err := cheques.CreateSchedule(tctx, lease, repo.ScheduleInput{Count: 12, FirstDate: today.AddDate(0, 0, -1), FirstNumber: "000101",
		BankName: "ENBD", DrawerName: "Omar"}, &agent)
	if err != nil || len(book) != 12 || book[11].ChequeNumber != "000112" {
		t.Fatalf("schedule: %d cheques, %v", len(book), err)
	}

	jobs := []Job{LeaseAlerts(ops, compliance, n), ChequeDeposits(cheques, compliance, n), RentReminders(ops, repo.NewPrivacyRepo(pool), mail, wa)}
	for round := 0; round < 2; round++ {
		for _, j := range jobs {
			if _, err := j.PerCompany(tctx, co); err != nil {
				t.Fatalf("%s: %v", j.Name, err)
			}
		}
	}

	count := func(user uuid.UUID, typ string) int {
		var c int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND type = $2`, user, typ).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	for _, u := range []uuid.UUID{admin, agent} {
		for _, typ := range []string{"lease_ending", "rent_notice_deadline", "ejari_expiry", "cheques_to_deposit"} {
			if got := count(u, typ); got != 1 {
				t.Errorf("%s notifications for %s: %d, want exactly 1", typ, u, got)
			}
		}
	}
	var days int
	if err := pool.QueryRow(ctx, `SELECT days_before FROM compliance_alerts WHERE company_id = $1 AND entity_type = 'lease_end'`, cid).Scan(&days); err != nil || days != 90 {
		t.Errorf("lease_end alert at %d days (%v), want the 90-day threshold", days, err)
	}

	// Omar: the payment due in 2 days is not covered by a cheque in this lease's book (the
	// book's payments are), so he is reminded by email and WhatsApp. Sara opted out of
	// WhatsApp: email only, and the WhatsApp reminder is recorded as skipped.
	// The book's own payments are covered by cheques in hand: no reminders for them.
	if len(mail.sent) != 2 {
		t.Errorf("%d reminder emails, want 2", len(mail.sent))
	}
	if len(wa.sent) != 1 || wa.sent[0][0] != phone || wa.sent[0][1] != "rent_due" || wa.sent[0][3] != "Omar" {
		t.Errorf("WhatsApp reminders %v", wa.sent)
	}
	var skipped int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM payment_reminders WHERE company_id = $1 AND delivery_status = 'skipped'
		AND delivery_method = 'whatsapp'`, cid).Scan(&skipped); err != nil || skipped != 1 {
		t.Errorf("skipped WhatsApp reminders: %d (%v)", skipped, err)
	}
}
