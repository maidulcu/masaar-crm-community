package jobs

// Integration test: two runners (two replicas) against one database send a due viewing's
// reminder exactly once. Skipped unless TEST_DATABASE_URL is set.

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/testdb"
)

func TestViewingReminderSentOnceAcrossReplicas(t *testing.T) {
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

	cid, uid, contactID, viewingID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`INSERT INTO companies (id, name, subdomain) VALUES ($1, 'Jobs Co', $2)`, cid, "j-"+cid.String()[:8])
	exec(`INSERT INTO company_settings (company_id, name, vat_number, business_address) VALUES ($1, 'Jobs Co', '', '')`, cid)
	exec(`INSERT INTO users (id, company_id, name, email, password_hash, role, lang_pref) VALUES ($1, $2, 'Agent', $3, 'x', 'agent', 'ar')`,
		uid, cid, uid.String()+"@test.local")
	exec(`INSERT INTO contacts (id, company_id, phone_wa, full_name, language) VALUES ($1, $2, $3, 'Client', 'en')`,
		contactID, cid, "+97158"+uid.String()[:7])
	exec(`INSERT INTO viewings (id, company_id, contact_id, agent_id, scheduled_at, address)
	      VALUES ($1, $2, $3, $4, NOW() + INTERVAL '20 minutes', 'Marina Tower 1')`, viewingID, cid, contactID, uid)
	t.Cleanup(func() {
		for _, q := range []string{
			`DELETE FROM notifications WHERE company_id = $1`, `DELETE FROM viewings WHERE company_id = $1`,
			`DELETE FROM contacts WHERE company_id = $1`, `DELETE FROM users WHERE company_id = $1`,
			`DELETE FROM company_settings WHERE company_id = $1`, `DELETE FROM companies WHERE id = $1`,
		} {
			_, _ = pool.Exec(ctx, q, cid)
		}
	})

	// The job name is shared with any real deployment's schedule row, so reset it.
	exec(`DELETE FROM job_schedule WHERE job_name = 'viewing_reminders'`)

	jobRepo := repo.NewJobRepo(pool)
	newReplica := func() *Runner {
		n := NewNotifier(repo.NewNotificationRepo(pool), repo.NewUserRepo(pool), nil, nil)
		return NewRunner(jobRepo, time.Second, ViewingReminders(repo.NewViewingRepo(pool), n))
	}
	var wg sync.WaitGroup
	for _, r := range []*Runner{newReplica(), newReplica()} {
		wg.Add(1)
		go func(r *Runner) { defer wg.Done(); r.RunDue(ctx) }(r)
	}
	wg.Wait()
	// Even if the schedule allowed a second run, the per-viewing claim stops a resend.
	exec(`UPDATE job_schedule SET last_started_at = 'epoch', locked_until = 'epoch' WHERE job_name = 'viewing_reminders'`)
	newReplica().RunDue(ctx)

	var n int
	var title string
	if err := pool.QueryRow(ctx, `SELECT COUNT(*), COALESCE(MAX(title),'') FROM notifications WHERE company_id = $1 AND type = 'viewing_reminder'`, cid).Scan(&n, &title); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("%d reminders sent, want exactly 1", n)
	}
	if !strings.HasPrefix(title, "معاينة") {
		t.Errorf("agent prefers Arabic, got title %q", title)
	}
	v, err := repo.NewViewingRepo(pool).GetByID(tenant.With(ctx, cid), viewingID)
	if err != nil || !v.ReminderSent {
		t.Fatalf("viewing reminder_sent = %v (%v)", v != nil && v.ReminderSent, err)
	}
}
