package repo

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestPortalConnectionsAndLeads(t *testing.T) {
	e := setup(t)
	portals := NewPortalRepo(e.pool)

	inbound, feed, err := portals.Connect(e.a.ctx, "bayut")
	if err != nil || inbound == "" || feed == "" || inbound == feed {
		t.Fatalf("connect: %q %q %v", inbound, feed, err)
	}
	if cid, err := portals.CompanyForToken(e.a.ctx, "bayut", "inbound", inbound); err != nil || cid != e.a.id {
		t.Fatalf("inbound token → %v, %v", cid, err)
	}
	if cid, err := portals.CompanyForToken(e.a.ctx, "bayut", "feed", feed); err != nil || cid != e.a.id {
		t.Fatalf("feed token → %v, %v", cid, err)
	}
	// A token only opens its own portal and its own kind.
	for _, c := range []struct{ portal, kind, token string }{
		{"property_finder", "inbound", inbound}, {"bayut", "feed", inbound}, {"bayut", "inbound", feed}, {"bayut", "inbound", ""},
	} {
		if _, err := portals.CompanyForToken(e.a.ctx, c.portal, c.kind, c.token); !errors.Is(err, ErrBadPortalToken) {
			t.Errorf("%+v accepted: %v", c, err)
		}
	}

	// Rotating replaces the old tokens.
	inbound2, _, err := portals.Connect(e.a.ctx, "bayut")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := portals.CompanyForToken(e.a.ctx, "bayut", "inbound", inbound); !errors.Is(err, ErrBadPortalToken) {
		t.Fatal("old token still works after rotation")
	}
	if err := portals.SetEnabled(e.a.ctx, "bayut", false); err != nil {
		t.Fatal(err)
	}
	if _, err := portals.CompanyForToken(e.a.ctx, "bayut", "inbound", inbound2); !errors.Is(err, ErrBadPortalToken) {
		t.Fatal("disabled connection accepted a lead")
	}
	_ = portals.SetEnabled(e.a.ctx, "bayut", true)
	if conns, _ := portals.ListConnections(e.b.ctx); len(conns) != 0 {
		t.Fatal("B sees A's portal connections")
	}
	if conns, _ := portals.ListConnections(e.a.ctx); len(conns) != 1 || !conns[0].Enabled {
		t.Fatalf("A's connections = %+v", conns)
	}

	// A portal lead id is held once per company and portal, on the lead itself.
	leads := NewLeadRepo(e.pool)
	portalLead := func(c company, ref string) error {
		ct := e.contact(t, c, testPhone("97155"))
		return leads.Create(c.ctx, &domain.Lead{ContactID: ct.ID, Stage: domain.StageNew, Source: domain.SourceBayut,
			Currency: "AED", ExternalSource: "bayut", ExternalRef: ref})
	}
	if err := portalLead(e.a, "ext-1"); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if err := portalLead(e.a, "ext-1"); !errors.Is(err, ErrDuplicateExternalRef) {
		t.Fatalf("redelivery: want ErrDuplicateExternalRef, got %v", err)
	}
	if err := portalLead(e.b, "ext-1"); err != nil {
		t.Fatalf("another company's lead with the same id was blocked: %v", err)
	}
	if err := portals.TouchLastLead(e.a.ctx, "bayut"); err != nil {
		t.Fatal(err)
	}
	if conns, _ := portals.ListConnections(e.a.ctx); len(conns) != 1 || conns[0].LastLeadAt == nil {
		t.Fatalf("last_lead_at not set: %+v", conns)
	}

	if err := portals.Disconnect(e.a.ctx, "bayut"); err != nil {
		t.Fatal(err)
	}
	if _, err := portals.CompanyForToken(e.a.ctx, "bayut", "inbound", inbound2); !errors.Is(err, ErrBadPortalToken) {
		t.Fatal("disconnected portal accepted a lead")
	}
}

func TestListingByReferenceAndLeadSourceReport(t *testing.T) {
	e := setup(t)
	portals := NewPortalRepo(e.pool)
	lid := e.listing(t, e.a)
	must(t, e.pool, `UPDATE listings SET reference_number = 'MS-101', title = 'Marina 2BR' WHERE id = $1`, lid)
	if id, title, err := portals.ListingIDByReference(e.a.ctx, "MS-101"); err != nil || id == nil || *id != lid || title != "Marina 2BR" {
		t.Fatalf("by reference: %v %q %v", id, title, err)
	}
	if id, _, err := portals.ListingIDByReference(e.b.ctx, "MS-101"); err != nil || id != nil {
		t.Fatal("B found A's listing by reference")
	}

	leads := NewLeadRepo(e.pool)
	for _, src := range []domain.LeadSource{domain.SourceBayut, domain.SourceBayut, domain.SourcePropertyFinder} {
		ct := e.contact(t, e.a, testPhone("97156"))
		l := &domain.Lead{ContactID: ct.ID, Stage: domain.StageNew, Source: src, Currency: "AED"}
		if err := leads.Create(e.a.ctx, l); err != nil {
			t.Fatal(err)
		}
		if src == domain.SourcePropertyFinder {
			must(t, e.pool, `UPDATE leads SET stage = 'won' WHERE id = $1`, l.ID)
			must(t, e.pool, `INSERT INTO deals (id, company_id, lead_id, title, stage, amount, owner_id) VALUES ($1, $2, $3, 'Sale', 'won', 2500000, $4)`,
				uuid.New(), e.a.id, l.ID, e.a.user)
		}
	}
	from, to := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	rows, err := portals.LeadSourceReport(e.a.ctx, from, to)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]LeadSourceRow{}
	for _, r := range rows {
		got[r.Source] = r
	}
	if got["bayut"].Leads != 2 || got["bayut"].WonLeads != 0 {
		t.Errorf("bayut = %+v", got["bayut"])
	}
	if pf := got["property_finder"]; pf.Leads != 1 || pf.WonLeads != 1 || pf.Deals != 1 || pf.WonRevenue != 2500000 {
		t.Errorf("property_finder = %+v", pf)
	}
	if rows, _ := portals.LeadSourceReport(e.b.ctx, from, to); len(rows) != 0 {
		t.Fatalf("B's report includes A's leads: %+v", rows)
	}
}
