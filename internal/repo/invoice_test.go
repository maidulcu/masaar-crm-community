package repo

import (
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/invoice"
)

func (e *testEnv) deal(t *testing.T, c company) uuid.UUID {
	t.Helper()
	ct := e.contact(t, c, testPhone("97159"))
	ld := e.lead(t, c, ct.ID)
	d := &domain.Deal{LeadID: ld.ID, Title: "Deal", Stage: domain.DealStageOpen, Amount: 100, Currency: "AED", OwnerID: c.user}
	if err := NewDealRepo(e.pool).Create(c.ctx, d); err != nil {
		t.Fatal(err)
	}
	return d.ID
}

// testInvoice is a one-line standard-rated AED invoice of net amount `net`.
func testInvoice(dealID uuid.UUID, net float64) *domain.VATInvoice {
	vat := net * 0.05
	return &domain.VATInvoice{
		Kind: domain.KindInvoice, DealID: dealID, Currency: "AED",
		Subtotal: net, VATRate: 0.05, VATAmount: vat, Total: net + vat,
		Supplier: domain.InvoiceParty{Name: "Supplier", TRN: "100123456700003"},
		Customer: domain.InvoiceParty{Name: "Customer"},
		Lines: []domain.InvoiceLine{{Description: "Fee", Quantity: 1, UnitPrice: net,
			VATCategory: "standard", VATRate: 0.05, NetAmount: net, VATAmount: vat}},
	}
}

func creditNote(orig *domain.VATInvoice, total float64) *domain.VATInvoice {
	cn := testInvoice(orig.DealID, total/1.05)
	cn.Kind, cn.OriginalInvoiceID, cn.Reason = domain.KindCreditNote, &orig.ID, "refund"
	cn.Total = total
	return cn
}

func TestInvoiceNumbersAreSequentialUnderConcurrency(t *testing.T) {
	e := setup(t)
	invoices := NewInvoiceRepo(e.pool)
	deal := e.deal(t, e.a)

	const n = 12
	nos := make([]string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			inv := testInvoice(deal, 100)
			if err := invoices.Issue(e.a.ctx, inv); err != nil {
				t.Error(err)
				return
			}
			nos[i] = inv.InvoiceNo
		}(i)
	}
	wg.Wait()
	sort.Strings(nos)
	for i, no := range nos {
		if want := invoice.Number(invoice.PrefixInvoice, 2026, i+1); no[len(no)-4:] != want[len(want)-4:] {
			t.Fatalf("numbers %v are not 1..%d without gaps", nos, n)
		}
	}

	got, err := invoices.GetByID(e.a.ctx, mustFindByNo(t, e, nos[0]))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Lines) != 1 || got.Lines[0].VATCategory != "standard" || got.Total != 105 || !got.IsTaxInvoice || got.Outstanding != 105 {
		t.Fatalf("stored invoice = %+v", got)
	}
}

func mustFindByNo(t *testing.T, e *testEnv, no string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := e.pool.QueryRow(e.a.ctx, `SELECT id FROM vat_invoices WHERE company_id = $1 AND invoice_no = $2`, e.a.id, no).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCreditNotesCannotOverCredit(t *testing.T) {
	e := setup(t)
	invoices := NewInvoiceRepo(e.pool)
	inv := testInvoice(e.deal(t, e.a), 1000) // total 1050
	if err := invoices.Issue(e.a.ctx, inv); err != nil {
		t.Fatal(err)
	}

	// Five concurrent credit notes of 300 each: at most three fit (900 ≤ 1050 < 1200).
	var wg sync.WaitGroup
	var mu sync.Mutex
	issued := 0
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := invoices.Issue(e.a.ctx, creditNote(inv, 300))
			switch {
			case err == nil:
				mu.Lock()
				issued++
				mu.Unlock()
			case !invoice.IsValidation(err):
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if issued != 3 {
		t.Fatalf("%d credit notes of 300 issued against 1050, want 3", issued)
	}
	got, _ := invoices.GetByID(e.a.ctx, inv.ID)
	if got.CreditedTotal != 900 || got.Outstanding != 150 {
		t.Fatalf("credited %v outstanding %v, want 900 / 150", got.CreditedTotal, got.Outstanding)
	}
	notes, _ := invoices.ListCreditNotes(e.a.ctx, inv.ID)
	if len(notes) != 3 || notes[0].InvoiceNo[:3] != "CN-" {
		t.Fatalf("credit notes = %+v", notes)
	}

	// An invoice with credit notes cannot be voided; voiding the notes frees it.
	if err := invoices.UpdateStatus(e.a.ctx, inv.ID, domain.InvoiceVoid, "error"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("voided an invoice with credit notes: %v", err)
	}
	for _, cn := range notes {
		if err := invoices.UpdateStatus(e.a.ctx, cn.ID, domain.InvoiceVoid, "error"); err != nil {
			t.Fatal(err)
		}
		if err := invoices.UpdateStatus(e.a.ctx, cn.ID, domain.InvoicePaid, ""); !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("paid a void credit note: %v", err)
		}
	}
	if err := invoices.UpdateStatus(e.a.ctx, inv.ID, domain.InvoiceVoid, "issued in error"); err != nil {
		t.Fatal(err)
	}
	if err := invoices.UpdateStatus(e.a.ctx, inv.ID, domain.InvoiceSent, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("a void invoice changed status: %v", err)
	}
	if err := invoices.Issue(e.a.ctx, creditNote(inv, 10)); !invoice.IsValidation(err) {
		t.Fatalf("credited a void invoice: %v", err)
	}
}

func TestCreditNoteMustMatchCurrencyAndCompany(t *testing.T) {
	e := setup(t)
	invoices := NewInvoiceRepo(e.pool)
	inv := testInvoice(e.deal(t, e.a), 100)
	if err := invoices.Issue(e.a.ctx, inv); err != nil {
		t.Fatal(err)
	}
	cn := creditNote(inv, 10)
	cn.Currency = "USD"
	if err := invoices.Issue(e.a.ctx, cn); !invoice.IsValidation(err) {
		t.Fatalf("credit note in another currency: %v", err)
	}
	if err := invoices.Issue(e.b.ctx, creditNote(inv, 10)); !isNotFound(err) {
		t.Fatalf("B credited A's invoice: %v", err)
	}
}
