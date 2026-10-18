package repo

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestJobClaimIsExclusive(t *testing.T) {
	e := setup(t)
	jobs := NewJobRepo(e.pool)
	name := "test-" + uuid.NewString()
	t.Cleanup(func() { _, _ = e.pool.Exec(e.a.ctx, `DELETE FROM job_schedule WHERE job_name = $1`, name) })

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := jobs.Claim(e.a.ctx, name, time.Hour, time.Minute)
			if err != nil {
				t.Error(err)
			}
			if ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("%d replicas claimed the job, want 1", wins.Load())
	}

	// Released, but the interval has not elapsed: nobody may run it again yet.
	if err := jobs.Release(name); err != nil {
		t.Fatal(err)
	}
	if ok, _ := jobs.Claim(e.a.ctx, name, time.Hour, time.Minute); ok {
		t.Fatal("claimed again before the interval elapsed")
	}
	// A zero interval allows an immediate rerun once released...
	if ok, _ := jobs.Claim(e.a.ctx, name, 0, time.Minute); !ok {
		t.Fatal("could not claim a released job with a zero interval")
	}
	// ...but not while another replica holds the lease.
	if ok, _ := jobs.Claim(e.a.ctx, name, 0, time.Minute); ok {
		t.Fatal("claimed a job whose lease is still held")
	}
}

func TestJobRunsScopedToCompany(t *testing.T) {
	e := setup(t)
	jobs := NewJobRepo(e.pool)
	now := time.Now()
	for _, c := range []company{e.a, e.b} {
		id := c.id
		if err := jobs.RecordRun(e.a.ctx, &domain.JobRun{
			JobName: "test", CompanyID: &id, StartedAt: now, FinishedAt: now, Status: "ok", Items: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	runs, err := jobs.ListRuns(e.a.ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || *runs[0].CompanyID != e.a.id {
		t.Fatalf("A sees runs %+v, want only its own", runs)
	}
}

func TestViewingReminderClaimedOnceAndResetOnReschedule(t *testing.T) {
	e := setup(t)
	ct := e.contact(t, e.a, testPhone("97156"))
	listing := e.listing(t, e.a)
	viewings := NewViewingRepo(e.pool)

	v := &domain.Viewing{ContactID: ct.ID, ListingID: &listing, AgentID: &e.a.user, ScheduledAt: time.Now().Add(30 * time.Minute)}
	if err := viewings.Create(e.a.ctx, v); err != nil {
		t.Fatal(err)
	}
	due, err := viewings.DueForReminder(e.a.ctx, 60)
	if err != nil || len(due) != 1 {
		t.Fatalf("due = %v, %v; want the viewing", due, err)
	}
	if ok, _ := viewings.ClaimReminder(e.b.ctx, v.ID); ok {
		t.Fatal("company B claimed A's reminder")
	}
	if ok, err := viewings.ClaimReminder(e.a.ctx, v.ID); err != nil || !ok {
		t.Fatalf("first claim = %v, %v", ok, err)
	}
	if ok, _ := viewings.ClaimReminder(e.a.ctx, v.ID); ok {
		t.Fatal("reminder claimed twice")
	}

	// Saving without moving it keeps the flag; rescheduling clears it.
	if err := viewings.Update(e.a.ctx, v); err != nil {
		t.Fatal(err)
	}
	if got, _ := viewings.GetByID(e.a.ctx, v.ID); !got.ReminderSent {
		t.Fatal("unchanged update reset the reminder")
	}
	v.ScheduledAt = v.ScheduledAt.Add(time.Hour)
	if err := viewings.Update(e.a.ctx, v); err != nil {
		t.Fatal(err)
	}
	if got, _ := viewings.GetByID(e.a.ctx, v.ID); got.ReminderSent {
		t.Fatal("rescheduled viewing kept reminder_sent")
	}
}

func TestOfferExpireDueIsScopedAndOnce(t *testing.T) {
	e := setup(t)
	offers := NewOfferRepo(e.pool)
	mk := func(c company, validUntil time.Time) *domain.Offer {
		ct := e.contact(t, c, testPhone("97157"))
		o := &domain.Offer{ListingID: e.listing(t, c), ContactID: ct.ID, OfferAmount: 10, AgentID: &c.user, ValidUntil: &validUntil}
		if err := offers.Create(c.ctx, o); err != nil {
			t.Fatal(err)
		}
		return o
	}
	pastA := mk(e.a, time.Now().Add(-time.Hour))
	futureA := mk(e.a, time.Now().Add(time.Hour))
	pastB := mk(e.b, time.Now().Add(-time.Hour))

	expired, err := offers.ExpireDue(e.a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 1 || expired[0].ID != pastA.ID || expired[0].AgentID == nil || *expired[0].AgentID != e.a.user {
		t.Fatalf("expired = %+v, want only A's past offer", expired)
	}
	if again, _ := offers.ExpireDue(e.a.ctx); len(again) != 0 {
		t.Fatalf("offer reported twice: %+v", again)
	}
	for _, c := range []struct {
		ctx  company
		o    *domain.Offer
		want domain.OfferStatus
	}{{e.a, pastA, domain.OfferExpired}, {e.a, futureA, domain.OfferSubmitted}, {e.b, pastB, domain.OfferSubmitted}} {
		got, err := offers.GetByID(c.ctx.ctx, c.o.ID)
		if err != nil || got.Status != c.want {
			t.Errorf("offer %s status = %v (%v), want %s", c.o.ID, got.Status, err, c.want)
		}
	}
}

func TestMoneyKeepsThreeDecimals(t *testing.T) {
	e := setup(t)
	id := e.listing(t, e.a)
	must(t, e.pool, `UPDATE listings SET price = 12.345, currency = 'BHD' WHERE id = $1`, id)
	var price float64
	if err := e.pool.QueryRow(e.a.ctx, `SELECT price FROM listings WHERE id = $1`, id).Scan(&price); err != nil {
		t.Fatal(err)
	}
	if price != 12.345 {
		t.Fatalf("price = %v, want 12.345", price)
	}
}

func TestCompanyLocaleDefaultsAndUpdate(t *testing.T) {
	e := setup(t)
	settings := NewCompanySettingsRepo(e.pool)
	loc, err := settings.Locale(e.a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if loc.Country != "AE" || loc.Currency != "AED" || loc.Timezone != "Asia/Dubai" {
		t.Fatalf("defaults = %+v", loc)
	}
	s, err := settings.Get(e.a.ctx)
	if err != nil {
		t.Fatal(err)
	}
	s.Region, s.Currency = "AE-DU", "USD"
	if err := settings.Update(e.a.ctx, s, nil); err != nil {
		t.Fatal(err)
	}
	if loc, _ := settings.Locale(e.a.ctx); loc.Region != "AE-DU" || loc.Currency != "USD" {
		t.Fatalf("after update = %+v", loc)
	}
	if loc, _ := settings.Locale(e.b.ctx); loc.Currency != "AED" {
		t.Fatalf("B's locale changed: %+v", loc)
	}
}
