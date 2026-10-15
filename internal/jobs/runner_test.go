package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// memStore is an in-memory Store: a job may be claimed once per interval.
type memStore struct {
	mu        sync.Mutex
	last      map[string]time.Time
	companies []domain.CompanyLocale
	runs      []domain.JobRun
}

func newMemStore(companies ...domain.CompanyLocale) *memStore {
	return &memStore{last: map[string]time.Time{}, companies: companies}
}

func (m *memStore) Claim(_ context.Context, name string, interval, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.last[name]; ok && time.Since(t) < interval {
		return false, nil
	}
	m.last[name] = time.Now()
	return true, nil
}
func (m *memStore) Release(string) error { return nil }
func (m *memStore) RecordRun(_ context.Context, r *domain.JobRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs = append(m.runs, *r)
	return nil
}
func (m *memStore) ActiveCompanies(context.Context) ([]domain.CompanyLocale, error) {
	return m.companies, nil
}

func TestRunnerRunsPerCompanyWithTenantAndLogsOnlyUsefulRuns(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	store := newMemStore(domain.CompanyLocale{CompanyID: a}, domain.CompanyLocale{CompanyID: b})
	var seen []uuid.UUID
	job := Job{Name: "j", Interval: time.Hour, PerCompany: func(ctx context.Context, co domain.CompanyLocale) (int, error) {
		cid, err := tenant.From(ctx)
		if err != nil || cid != co.CompanyID {
			t.Errorf("tenant = %v, %v; want %v", cid, err, co.CompanyID)
		}
		seen = append(seen, cid)
		if cid == a {
			return 2, nil
		}
		return 0, nil // idle: not logged
	}}
	r := NewRunner(store, time.Second, job)
	r.RunDue(context.Background())
	r.RunDue(context.Background()) // interval not elapsed: no second run

	if len(seen) != 2 {
		t.Fatalf("ran for %d companies, want 2 (once each)", len(seen))
	}
	if len(store.runs) != 1 || *store.runs[0].CompanyID != a || store.runs[0].Items != 2 {
		t.Fatalf("runs = %+v, want one logged run for A", store.runs)
	}
}

func TestRunnerSurvivesErrorsAndPanics(t *testing.T) {
	store := newMemStore(domain.CompanyLocale{CompanyID: uuid.New()})
	ran := false
	r := NewRunner(store, time.Second,
		Job{Name: "fails", Interval: time.Hour, System: func(context.Context) (int, error) { return 0, errors.New("boom") }},
		Job{Name: "panics", Interval: time.Hour, PerCompany: func(context.Context, domain.CompanyLocale) (int, error) { panic("oops") }},
		Job{Name: "ok", Interval: time.Hour, System: func(context.Context) (int, error) { ran = true; return 1, nil }},
	)
	r.RunDue(context.Background())
	if !ran {
		t.Fatal("a failing job stopped the jobs after it")
	}
	status := map[string]string{}
	for _, run := range store.runs {
		status[run.JobName] = run.Status
	}
	if status["fails"] != "error" || status["panics"] != "error" || status["ok"] != "ok" {
		t.Fatalf("statuses = %v", status)
	}
}

func TestRunnerStopsOnCancel(t *testing.T) {
	store := newMemStore()
	r := NewRunner(store, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	r.Start(ctx)
	cancel()
	done := make(chan struct{})
	go func() { r.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop after cancel")
	}
}
