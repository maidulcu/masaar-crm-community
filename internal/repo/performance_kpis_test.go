package repo

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

// GetAgentKPIs answers with one query; this checks it against the six separate queries it
// replaced, on data that exercises every filter (period, stage, deleted, company, other agent).
func TestGetAgentKPIsMatchesSeparateQueries(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	repo := NewPerformanceRepo(e.pool)
	agent := e.a.user
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	in := from.Add(5 * 24 * time.Hour)
	out := from.Add(-40 * 24 * time.Hour)

	otherAgent := uuid.New()
	must(t, e.pool, `INSERT INTO users (id, company_id, name, email, password_hash, role) VALUES ($1,$2,'Other agent',$3,'x','agent')`, otherAgent, e.a.id, otherAgent.String()+"@test.local")

	ct := e.contact(t, e.a, testPhone("97150"))
	mkLead := func(stage string, assigned uuid.UUID, created time.Time, contacted *time.Time, deleted *time.Time) uuid.UUID {
		id := uuid.New()
		must(t, e.pool, `INSERT INTO leads (id, company_id, contact_id, stage, source, assigned_to, created_at, updated_at, last_contacted_at, deleted_at)
			VALUES ($1,$2,$3,$4,'web',$5,$6,$6,$7,$8)`, id, e.a.id, ct.ID, stage, assigned, created, contacted, deleted)
		return id
	}
	hours := func(h int) *time.Time { v := in.Add(time.Duration(h) * time.Hour); return &v }
	del := in

	// Leads assigned to the agent in the period: 4 live (2 answered after 2h and 6h), 1 deleted.
	l1 := mkLead("new", agent, in, hours(2), nil)
	mkLead("won", agent, in, hours(6), nil) // also a conversion
	mkLead("new", agent, in, nil, nil)
	mkLead("won", agent, in, nil, nil)       // second conversion, never contacted
	mkLead("new", agent, in, nil, &del)      // deleted: ignored everywhere
	mkLead("new", agent, out, hours(1), nil) // outside the period
	mkLead("won", otherAgent, in, hours(1), nil)

	mkDeal := func(owner uuid.UUID, stage string, amount int, when time.Time) {
		must(t, e.pool, `INSERT INTO deals (lead_id, company_id, title, stage, amount, owner_id, created_at, updated_at) VALUES ($1,$2,'D',$3,$4,$5,$6,$6)`,
			l1, e.a.id, stage, amount, owner, when)
	}
	mkDeal(agent, "won", 1000, in)
	mkDeal(agent, "won", 2500, in)
	mkDeal(agent, "open", 9999, in)     // not won
	mkDeal(agent, "won", 7777, out)     // outside the period
	mkDeal(otherAgent, "won", 5555, in) // someone else's

	mkListing := func(by uuid.UUID, when time.Time) {
		must(t, e.pool, `INSERT INTO listings (company_id, title, property_type, price, created_by, created_at) VALUES ($1,'L','apartment',1,$2,$3)`, e.a.id, by, when)
	}
	mkListing(agent, in)
	mkListing(agent, in)
	mkListing(agent, out)
	mkListing(otherAgent, in)

	got, err := repo.GetAgentKPIs(e.a.ctx, agent, from, to)
	if err != nil {
		t.Fatal(err)
	}

	// Reference: the original six queries.
	var wantDeals, wantListings, wantAssigned, wantConverted int
	var wantRevenue, wantAvg float64
	q := func(sql string, dest ...any) {
		t.Helper()
		if err := e.pool.QueryRow(ctx, sql, agent, from, to, e.a.id).Scan(dest...); err != nil {
			t.Fatal(err)
		}
	}
	q(`SELECT COUNT(*), COALESCE(SUM(amount),0) FROM deals WHERE owner_id=$1 AND company_id=$4 AND stage='won' AND updated_at BETWEEN $2 AND $3`, &wantDeals, &wantRevenue)
	q(`SELECT COUNT(*) FROM listings WHERE created_by=$1 AND company_id=$4 AND created_at BETWEEN $2 AND $3`, &wantListings)
	q(`SELECT COUNT(*) FROM leads WHERE assigned_to=$1 AND company_id=$4 AND created_at BETWEEN $2 AND $3 AND deleted_at IS NULL`, &wantAssigned)
	q(`SELECT COUNT(*) FROM leads WHERE assigned_to=$1 AND company_id=$4 AND stage='won' AND updated_at BETWEEN $2 AND $3 AND deleted_at IS NULL`, &wantConverted)
	q(`SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (last_contacted_at - created_at)) / 3600), 0) FROM leads WHERE assigned_to=$1 AND company_id=$4 AND last_contacted_at IS NOT NULL AND created_at BETWEEN $2 AND $3 AND deleted_at IS NULL`, &wantAvg)

	// Sanity: the fixture really produces non-trivial numbers.
	if wantDeals != 2 || wantListings != 2 || wantAssigned != 4 || wantConverted != 2 || math.Abs(wantAvg-4) > 1e-9 {
		t.Fatalf("fixture produced deals=%d listings=%d assigned=%d converted=%d avg=%v", wantDeals, wantListings, wantAssigned, wantConverted, wantAvg)
	}
	if got.DealsWon != wantDeals || got.ListingsAdded != wantListings || got.LeadsAssigned != wantAssigned || got.LeadsConverted != wantConverted {
		t.Errorf("counts: got deals=%d listings=%d assigned=%d converted=%d; want %d %d %d %d",
			got.DealsWon, got.ListingsAdded, got.LeadsAssigned, got.LeadsConverted, wantDeals, wantListings, wantAssigned, wantConverted)
	}
	if math.Abs(float64(got.Revenue)-wantRevenue) > 1e-6 || math.Abs(got.AvgResponseHours-wantAvg) > 1e-9 {
		t.Errorf("revenue=%v avg=%v; want %v %v", got.Revenue, got.AvgResponseHours, wantRevenue, wantAvg)
	}
	if got.AgentName == "" {
		t.Error("agent name missing")
	}

	// An agent with no activity gets zeros, not an error.
	empty, err := repo.GetAgentKPIs(e.a.ctx, otherAgent, from.AddDate(-5, 0, 0), from.AddDate(-4, 0, 0))
	if err != nil || empty.DealsWon != 0 || empty.LeadsAssigned != 0 || empty.AvgResponseHours != 0 {
		t.Errorf("idle agent: %+v, %v", empty, err)
	}

	// Another company cannot read this agent's numbers, and a missing agent is "not found".
	if _, err := repo.GetAgentKPIs(e.b.ctx, agent, from, to); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("other company: err = %v, want ErrUserNotFound", err)
	}
	if _, err := repo.GetAgentKPIs(e.a.ctx, uuid.New(), from, to); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("unknown agent: err = %v, want ErrUserNotFound", err)
	}
}
