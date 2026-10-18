package repo

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/cheque"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

// lease creates a property, a tenant and a one-year lease at 10,000 a month.
func (e *testEnv) lease(t *testing.T, c company) uuid.UUID {
	t.Helper()
	prop, ten, id := uuid.New(), uuid.New(), uuid.New()
	must(t, e.pool, `INSERT INTO rental_properties (id, company_id, name, property_type) VALUES ($1, $2, 'Marina Tower', 'apartment')`, prop, c.id)
	must(t, e.pool, `INSERT INTO tenants (id, company_id, full_name_en, id_type) VALUES ($1, $2, 'Omar', 'emirates_id')`, ten, c.id)
	must(t, e.pool, `INSERT INTO leases (id, company_id, property_id, tenant_id, start_date, end_date, monthly_rent, payment_frequency, created_by)
		VALUES ($1, $2, $3, $4, '2026-01-01', '2026-12-31', 10000, 'monthly', $5)`, id, c.id, prop, ten, c.user)
	return id
}

func TestChequeLifecycle(t *testing.T) {
	e := setup(t)
	cheques := NewChequeRepo(e.pool)
	lease := e.lease(t, e.a)

	if _, err := cheques.CreateSchedule(e.b.ctx, lease, ScheduleInput{Count: 12}, nil); !isNotFound(err) {
		t.Fatalf("B scheduled cheques on A's lease: %v", err)
	}
	book, err := cheques.CreateSchedule(e.a.ctx, lease, ScheduleInput{Count: 12, FirstNumber: "0091", BankName: "ENBD"}, &e.a.user)
	if err != nil || len(book) != 12 {
		t.Fatalf("schedule: %d, %v", len(book), err)
	}
	for i, c := range book {
		want := time.Date(2026, time.Month(1+i), 1, 0, 0, 0, 0, time.UTC)
		if !c.ChequeDate.Equal(want) || c.Amount != 10000 || c.PaymentID == nil || c.Status != cheque.Received {
			t.Fatalf("cheque %d = %+v", i+1, c)
		}
	}
	if book[0].ChequeNumber != "0091" || book[11].ChequeNumber != "0102" || book[0].TenantName != "Omar" {
		t.Fatalf("numbers %s…%s", book[0].ChequeNumber, book[11].ChequeNumber)
	}
	var payments int
	_ = e.pool.QueryRow(e.a.ctx, `SELECT COUNT(*) FROM payments WHERE lease_id = $1 AND payment_method = 'check' AND status = 'pending'`, lease).Scan(&payments)
	if payments != 12 {
		t.Fatalf("%d payments", payments)
	}
	if _, err := cheques.CreateSchedule(e.a.ctx, lease, ScheduleInput{Count: 12}, nil); !errors.Is(err, ErrScheduleExists) {
		t.Fatalf("second schedule: %v", err)
	}
	if got, _ := cheques.List(e.b.ctx, ChequeFilter{}); len(got) != 0 {
		t.Fatal("B lists A's cheques")
	}

	day := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	first, second := book[0], book[1]
	if _, err := cheques.Transition(e.a.ctx, first.ID, cheque.Cleared, day, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("received → cleared: %v", err)
	}
	if _, err := cheques.Transition(e.b.ctx, first.ID, cheque.Deposited, day, ""); !isNotFound(err) {
		t.Fatalf("B deposited A's cheque: %v", err)
	}
	for _, s := range []string{cheque.Deposited, cheque.Cleared} {
		if _, err := cheques.Transition(e.a.ctx, first.ID, s, day, ""); err != nil {
			t.Fatalf("→ %s: %v", s, err)
		}
	}
	var status string
	var paid *time.Time
	_ = e.pool.QueryRow(e.a.ctx, `SELECT status, paid_date FROM payments WHERE id = $1`, *first.PaymentID).Scan(&status, &paid)
	if status != "received" || paid == nil || !paid.Equal(day) {
		t.Fatalf("cleared cheque's payment: %s %v", status, paid)
	}

	// Bounce, charge a fee, and replace.
	_, _ = cheques.Transition(e.a.ctx, second.ID, cheque.Deposited, day, "")
	if prev, err := cheques.Transition(e.a.ctx, second.ID, cheque.Bounced, day, "insufficient funds"); err != nil || prev != cheque.Deposited {
		t.Fatalf("bounce: %s %v", prev, err)
	}
	_ = e.pool.QueryRow(e.a.ctx, `SELECT status FROM payments WHERE id = $1`, *second.PaymentID).Scan(&status)
	if status != "failed" {
		t.Fatalf("bounced cheque's payment is %s", status)
	}
	if _, err := cheques.AddPenalty(e.a.ctx, second.ID, 500, day, &e.a.user); err != nil {
		t.Fatal(err)
	}
	if _, err := cheques.AddPenalty(e.b.ctx, second.ID, 500, day, nil); !isNotFound(err) {
		t.Fatalf("B charged a penalty on A's cheque: %v", err)
	}
	repl := &domain.Cheque{ChequeNumber: "7001", BankName: "ADCB", Amount: 10000, ChequeDate: day.AddDate(0, 0, 7)}
	if err := cheques.Replace(e.a.ctx, second.ID, repl, &e.a.user); err != nil {
		t.Fatal(err)
	}
	old, _ := cheques.Get(e.a.ctx, second.ID)
	if old.Status != cheque.Replaced || old.ReplacedBy == nil || *old.ReplacedBy != repl.ID || old.BounceReason != "insufficient funds" {
		t.Fatalf("replaced cheque = %+v", old)
	}
	if repl.PaymentID == nil || *repl.PaymentID != *second.PaymentID {
		t.Fatal("replacement does not pay the same payment")
	}
	_ = e.pool.QueryRow(e.a.ctx, `SELECT status FROM payments WHERE id = $1`, *second.PaymentID).Scan(&status)
	if status != "pending" {
		t.Fatalf("payment after replacement is %s", status)
	}
	if err := cheques.Replace(e.a.ctx, second.ID, &domain.Cheque{ChequeNumber: "x", Amount: 1, ChequeDate: day}, nil); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("replaced twice: %v", err)
	}

	if due, _ := cheques.ToDeposit(e.a.ctx, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)); len(due) != 2 {
		t.Fatalf("to deposit by 1 March: %d (want the March cheque and the replacement)", len(due))
	}
	if who, _ := cheques.LeaseAgent(e.a.ctx, lease); who == nil || *who != e.a.user {
		t.Fatalf("lease agent = %v", who)
	}
	if who, _ := cheques.LeaseAgent(e.b.ctx, lease); who != nil {
		t.Fatal("B sees A's lease agent")
	}
}

func TestLeaseNotices(t *testing.T) {
	e := setup(t)
	ops := NewLeaseOpsRepo(e.pool)
	lease := e.lease(t, e.a)
	nc, err := ops.NoticeContext(e.a.ctx, lease)
	if err != nil || nc.TenantName != "Omar" || nc.MonthlyRent != 10000 {
		t.Fatalf("context %+v %v", nc, err)
	}
	if _, err := ops.NoticeContext(e.b.ctx, lease); !isNotFound(err) {
		t.Fatalf("B read A's lease: %v", err)
	}
	n := &domain.LeaseNotice{LeaseID: lease, Kind: "rent_change", BodyEN: "x", BodyAR: "y", CreatedBy: &e.a.user}
	if err := ops.CreateNotice(e.b.ctx, n); !isNotFound(err) {
		t.Fatalf("B created a notice on A's lease: %v", err)
	}
	if err := ops.CreateNotice(e.a.ctx, n); err != nil {
		t.Fatal(err)
	}
	if err := ops.MarkNoticeSent(e.a.ctx, n.ID, "email", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := ops.MarkNoticeSent(e.a.ctx, n.ID, "hand", time.Now()); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("marked sent twice: %v", err)
	}
	if list, _ := ops.ListNotices(e.a.ctx, lease); len(list) != 1 || list[0].SentVia != "email" {
		t.Fatalf("notices %+v", list)
	}
	if err := ops.SetLeaseAgent(e.a.ctx, lease, &e.b.user, nil); !isNotFound(err) {
		t.Fatalf("A assigned B's user: %v", err)
	}
	s, err := ops.Settings(e.a.ctx)
	if err != nil || len(s.LeaseAlertDays) != 4 || s.RentNoticeDays != 90 {
		t.Fatalf("default settings %+v %v", s, err)
	}
}
