// Package jobs runs the CRM's background work: reminders, expiries and clean-up.
//
// Every API replica runs a Runner. Before running a job, a replica claims it in the
// job_schedule table (repo.JobRepo.Claim): the claim succeeds for one replica per
// interval, and its lease keeps a slow run from being started twice. Jobs must still be
// idempotent per item (claim each reminder, expire each offer with a conditional
// UPDATE): a crash after a send but before the claim is released can repeat a run.
package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// Job is one unit of scheduled work. Set exactly one of PerCompany or System.
type Job struct {
	Name     string
	Interval time.Duration
	// Lease bounds how long a run may hold the job before another replica may take it
	// over. Defaults to 10 × Interval, at least one minute.
	Lease time.Duration

	// PerCompany runs once for each active company with ctx bound to it (tenant.With),
	// and returns how many items it handled.
	PerCompany func(ctx context.Context, company domain.CompanyLocale) (int, error)
	// System runs once per interval for the whole deployment.
	System func(ctx context.Context) (int, error)
}

// Store is the scheduling state the runner needs (implemented by repo.JobRepo).
type Store interface {
	Claim(ctx context.Context, name string, interval, lease time.Duration) (bool, error)
	Release(name string) error
	RecordRun(ctx context.Context, run *domain.JobRun) error
	ActiveCompanies(ctx context.Context) ([]domain.CompanyLocale, error)
}

// Runner checks its jobs every Tick and runs the ones it can claim.
type Runner struct {
	store Store
	jobs  []Job
	tick  time.Duration
	now   func() time.Time

	wg sync.WaitGroup
}

// NewRunner returns a runner that polls every tick (30s if zero).
func NewRunner(store Store, tick time.Duration, jobs ...Job) *Runner {
	if tick <= 0 {
		tick = 30 * time.Second
	}
	for i := range jobs {
		if jobs[i].Lease <= 0 {
			jobs[i].Lease = 10 * jobs[i].Interval
			if jobs[i].Lease < time.Minute {
				jobs[i].Lease = time.Minute
			}
		}
	}
	return &Runner{store: store, jobs: jobs, tick: tick, now: time.Now}
}

// Start runs the scheduling loop until ctx is cancelled. Call Wait to block until the
// loop and any run in progress have finished.
func (r *Runner) Start(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		t := time.NewTicker(r.tick)
		defer t.Stop()
		for {
			r.RunDue(ctx)
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
		}
	}()
}

// Wait blocks until Start's loop has returned.
func (r *Runner) Wait() { r.wg.Wait() }

// RunDue runs, once, every job whose interval has elapsed and that this replica claims.
// Jobs run one after another; a failing or panicking job does not stop the others.
func (r *Runner) RunDue(ctx context.Context) {
	for _, j := range r.jobs {
		if ctx.Err() != nil {
			return
		}
		claimed, err := r.store.Claim(ctx, j.Name, j.Interval, j.Lease)
		if err != nil {
			log.Printf("jobs: claim %s: %v", j.Name, err)
			continue
		}
		if !claimed {
			continue
		}
		r.run(ctx, j)
		if err := r.store.Release(j.Name); err != nil {
			log.Printf("jobs: release %s: %v", j.Name, err)
		}
	}
}

func (r *Runner) run(ctx context.Context, j Job) {
	runCtx, cancel := context.WithTimeout(ctx, j.Lease)
	defer cancel()

	if j.System != nil {
		started := r.now()
		n, err := safeCall(func() (int, error) { return j.System(runCtx) })
		r.record(j.Name, nil, started, n, err)
		return
	}

	companies, err := r.store.ActiveCompanies(runCtx)
	if err != nil {
		log.Printf("jobs: %s: list companies: %v", j.Name, err)
		return
	}
	for _, co := range companies {
		if runCtx.Err() != nil {
			return
		}
		started := r.now()
		cctx := tenant.With(runCtx, co.CompanyID)
		n, err := safeCall(func() (int, error) { return j.PerCompany(cctx, co) })
		id := co.CompanyID
		r.record(j.Name, &id, started, n, err)
	}
}

// record logs runs that did something or failed; idle runs are not stored.
func (r *Runner) record(name string, companyID *uuid.UUID, started time.Time, items int, err error) {
	if err == nil && items == 0 {
		return
	}
	run := &domain.JobRun{
		JobName:    name,
		StartedAt:  started,
		FinishedAt: r.now(),
		Status:     "ok",
		Items:      items,
	}
	run.CompanyID = companyID
	if err != nil {
		run.Status = "error"
		run.Error = truncate(err.Error(), 500)
		log.Printf("jobs: %s: %v", name, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if rerr := r.store.RecordRun(ctx, run); rerr != nil {
		log.Printf("jobs: record %s: %v", name, rerr)
	}
}

func safeCall(f func() (int, error)) (n int, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()
	return f()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
