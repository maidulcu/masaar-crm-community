package repo

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestKanbanBoardCapsPerStageButKeepsTotals(t *testing.T) {
	e := setup(t)
	leads := NewLeadRepo(e.pool)

	// 5 "new" leads (values 10..50, created 5..1 minutes ago) and 2 "won" leads.
	base := time.Now().Add(-time.Hour).Truncate(time.Microsecond)
	var newIDs []uuid.UUID // newest first
	insert := func(stage string, value float64, age time.Duration) uuid.UUID {
		ct := e.contact(t, e.a, testPhone("97152"))
		id := uuid.New()
		must(t, e.pool, `INSERT INTO leads (id, company_id, contact_id, stage, source, deal_value, created_at)
			VALUES ($1, $2, $3, $4, 'web', $5, $6)`, id, e.a.id, ct.ID, stage, value, base.Add(-age))
		return id
	}
	for i := 1; i <= 5; i++ {
		newIDs = append(newIDs, insert("new", float64(i*10), time.Duration(i)*time.Minute))
	}
	insert("won", 1000, time.Minute)
	insert("won", 500, 2*time.Minute)
	// a soft-deleted lead is neither shown nor counted
	gone := insert("new", 99999, 30*time.Second)
	must(t, e.pool, `UPDATE leads SET deleted_at = NOW() WHERE id = $1`, gone)

	board, totals, err := leads.KanbanBoard(e.a.ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	got := board[domain.StageNew]
	if len(got) != 3 {
		t.Fatalf("new column has %d cards, want the 3 newest", len(got))
	}
	for i, l := range got {
		if l.ID != newIDs[i] {
			t.Errorf("card %d = %s, want %s (newest first)", i, l.ID, newIDs[i])
		}
	}
	if tt := totals[domain.StageNew]; tt.Count != 5 || tt.Value != 150 {
		t.Errorf("new totals = %+v, want count 5 value 150 (all non-deleted leads, not just the loaded ones)", tt)
	}
	if len(board[domain.StageWon]) != 2 || totals[domain.StageWon].Count != 2 || totals[domain.StageWon].Value != 1500 {
		t.Errorf("won column = %d cards, totals %+v", len(board[domain.StageWon]), totals[domain.StageWon])
	}

	// Company B sees none of it.
	bBoard, bTotals, err := leads.KanbanBoard(e.b.ctx, 3)
	if err != nil || len(bBoard) != 0 || len(bTotals) != 0 {
		t.Fatalf("company B board = %v %v err %v", bBoard, bTotals, err)
	}

	// Keyset paging continues exactly where the board stopped, even though a card was
	// moved into the stage in the meantime (which would shift an OFFSET).
	last := got[len(got)-1]
	insert("new", 1, 0) // newest of all: offset-based paging would now repeat last
	rest, err := leads.List(e.a.ctx, LeadFilter{
		Stage: domain.StageNew, Limit: 10, BeforeCreatedAt: &last.CreatedAt, BeforeID: &last.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rest) != 2 || rest[0].ID != newIDs[3] || rest[1].ID != newIDs[4] {
		t.Fatalf("next page = %v, want the 4th and 5th lead", ids(rest))
	}
}

func ids(ls []domain.Lead) []uuid.UUID {
	out := make([]uuid.UUID, len(ls))
	for i, l := range ls {
		out[i] = l.ID
	}
	return out
}

func TestLeadAndContactSearchTreatWildcardsLiterally(t *testing.T) {
	e := setup(t)
	plain := e.contact(t, e.a, testPhone("97153"))
	pct := &domain.Contact{PhoneWA: testPhone("97154"), FullName: "100% Real_Estate", Language: "en"}
	if err := NewContactRepo(e.pool).Create(e.a.ctx, pct); err != nil {
		t.Fatal(err)
	}
	e.lead(t, e.a, plain.ID)
	e.lead(t, e.a, pct.ID)

	leads, err := NewLeadRepo(e.pool).List(e.a.ctx, LeadFilter{Query: "%", Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(leads) != 1 || leads[0].ContactID != pct.ID {
		t.Fatalf("searching '%%' matched %d leads, want only the one whose name contains a literal %%", len(leads))
	}
	if leads, _ = NewLeadRepo(e.pool).List(e.a.ctx, LeadFilter{Query: "Real_E", Limit: 50}); len(leads) != 1 {
		t.Errorf("'Real_E' matched %d leads, want 1", len(leads))
	}
	if leads, _ = NewLeadRepo(e.pool).List(e.a.ctx, LeadFilter{Query: "Real?E", Limit: 50}); len(leads) != 0 {
		t.Errorf("'Real?E' matched %d leads, want 0", len(leads))
	}
	res, err := NewContactRepo(e.pool).List(e.a.ctx, "_", 1, 50)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 {
		t.Errorf("contact search for '_' matched %d, want only the name containing an underscore", res.Total)
	}
}
