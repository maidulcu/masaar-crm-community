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
	"github.com/maidulcu/masaar-crm/internal/testdb"
)

func TestExpiryAlertsSentOnce(t *testing.T) {
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

	cid, admin := uuid.New(), uuid.New()
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO companies (id, name, subdomain) VALUES ($1, 'Expiry Co', $2)`, []any{cid, "x-" + cid.String()[:8]}},
		{`INSERT INTO company_settings (company_id, name, vat_number, business_address) VALUES ($1, 'Expiry Co', '', '')`, []any{cid}},
		{`INSERT INTO users (id, company_id, name, email, password_hash, role, brn, brn_expiry)
		  VALUES ($1, $2, 'Broker', $3, 'x', 'admin', 'BRN-9', CURRENT_DATE + 5)`, []any{admin, cid, admin.String() + "@test.local"}},
	} {
		if _, err := pool.Exec(ctx, q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, q := range []string{`DELETE FROM notifications WHERE company_id = $1`, `DELETE FROM compliance_alerts WHERE company_id = $1`,
			`DELETE FROM users WHERE company_id = $1`, `DELETE FROM company_settings WHERE company_id = $1`, `DELETE FROM companies WHERE id = $1`} {
			_, _ = pool.Exec(ctx, q, cid)
		}
	})

	n := NewNotifier(repo.NewNotificationRepo(pool), repo.NewUserRepo(pool), nil, nil)
	job := ExpiryAlerts(repo.NewComplianceRepo(pool), n)
	for i := 0; i < 2; i++ {
		r := NewRunner(&onlyCompany{jobRepo: repo.NewJobRepo(pool), id: cid}, time.Second, job)
		r.RunDue(ctx)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE company_id = $1 AND type = 'expiry_alert'`, cid).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("%d expiry notifications, want exactly 1 (the 7-day alert, to the broker who is also the admin)", count)
	}
}

// onlyCompany is a Store that always lets the job run and limits it to one company, so
// the test is not affected by other packages' data or by the shared schedule row.
type onlyCompany struct {
	jobRepo *repo.JobRepo
	id      uuid.UUID
}

func (o *onlyCompany) Claim(context.Context, string, time.Duration, time.Duration) (bool, error) {
	return true, nil
}
func (o *onlyCompany) Release(string) error { return nil }
func (o *onlyCompany) RecordRun(ctx context.Context, run *domain.JobRun) error {
	return o.jobRepo.RecordRun(ctx, run)
}
func (o *onlyCompany) ActiveCompanies(context.Context) ([]domain.CompanyLocale, error) {
	return []domain.CompanyLocale{{CompanyID: o.id, Timezone: "Asia/Dubai"}}, nil
}
