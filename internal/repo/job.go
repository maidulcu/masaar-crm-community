package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// JobRepo stores background-job scheduling state and the per-company run log.
type JobRepo struct {
	db *pgxpool.Pool
}

func NewJobRepo(db *pgxpool.Pool) *JobRepo { return &JobRepo{db: db} }

// Claim takes the job for this replica if its interval has elapsed since the last start
// and no other replica holds it. The claim is a single conditional UPDATE, so exactly one
// replica wins; the lease (locked_until) stops a slow run from being started twice.
func (r *JobRepo) Claim(ctx context.Context, name string, interval, lease time.Duration) (bool, error) {
	if _, err := r.db.Exec(ctx,
		`INSERT INTO job_schedule (job_name) VALUES ($1) ON CONFLICT (job_name) DO NOTHING`, name); err != nil {
		return false, err
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE job_schedule
		   SET last_started_at = NOW(), locked_until = NOW() + $3 * INTERVAL '1 second'
		 WHERE job_name = $1
		   AND locked_until < NOW()
		   AND last_started_at <= NOW() - $2 * INTERVAL '1 second'`,
		name, interval.Seconds(), lease.Seconds())
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// Release ends the lease taken by Claim. It uses its own short context so a cancelled
// run (shutdown) still frees the job for another replica.
func (r *JobRepo) Release(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.db.Exec(ctx, `UPDATE job_schedule SET locked_until = NOW() WHERE job_name = $1`, name)
	return err
}

// RecordRun appends to the run log. companyID may be nil for system-wide work.
func (r *JobRepo) RecordRun(ctx context.Context, run *domain.JobRun) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO job_runs (job_name, company_id, started_at, finished_at, status, items, error)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		run.JobName, run.CompanyID, run.StartedAt, run.FinishedAt, run.Status, run.Items, run.Error,
	).Scan(&run.ID)
}

// ListRuns returns the caller's company's most recent job runs, newest first.
func (r *JobRepo) ListRuns(ctx context.Context, limit int) ([]domain.JobRun, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, job_name, company_id, started_at, finished_at, status, items, error
		  FROM job_runs WHERE company_id = $1
		 ORDER BY started_at DESC, id DESC LIMIT $2`, cid, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.JobRun, error) {
		var j domain.JobRun
		err := row.Scan(&j.ID, &j.JobName, &j.CompanyID, &j.StartedAt, &j.FinishedAt, &j.Status, &j.Items, &j.Error)
		return j, err
	})
}

// PruneRuns deletes run-log rows older than before and reports how many went.
func (r *JobRepo) PruneRuns(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM job_runs WHERE started_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ActiveCompanies returns the id and timezone of every active company, for jobs that
// work company by company.
func (r *JobRepo) ActiveCompanies(ctx context.Context) ([]domain.CompanyLocale, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, COALESCE(s.country,'AE'), COALESCE(s.region,''), COALESCE(s.currency,'AED'),
		       COALESCE(s.timezone,'Asia/Dubai')
		  FROM companies c
		  LEFT JOIN company_settings s ON s.company_id = c.id
		 WHERE c.is_active
		 ORDER BY c.created_at`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.CompanyLocale, error) {
		var l domain.CompanyLocale
		err := row.Scan(&l.CompanyID, &l.Country, &l.Region, &l.Currency, &l.Timezone)
		return l, err
	})
}
