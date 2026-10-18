package repo

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/kyc"
)

func TestKYCAndAML(t *testing.T) {
	e := setup(t)
	k := NewKYCRepo(e.pool)
	ct := e.contact(t, e.a, testPhone("97154"))
	ld := e.lead(t, e.a, ct.ID)
	deal := uuid.New()
	must(t, e.pool, `INSERT INTO deals (id, company_id, lead_id, title, amount, owner_id) VALUES ($1, $2, $3, 'Villa', 3000000, $4)`,
		deal, e.a.id, ld.ID, e.a.user)

	// No ID yet: the deal cannot be won.
	dk, err := k.DealKYC(e.a.ctx, deal)
	if err != nil || dk.CanWin || len(dk.Missing) != 1 || dk.Missing[0] != kyc.MissingID {
		t.Fatalf("deal KYC without ID = %+v, %v", dk, err)
	}
	if _, err := k.DealKYC(e.b.ctx, deal); !isNotFound(err) {
		t.Fatalf("B read A's deal KYC: %v", err)
	}
	expired := time.Now().AddDate(0, 0, -1)
	if err := k.AddDocument(e.a.ctx, &KYCDocument{ContactID: ct.ID, DocType: kyc.EmiratesID, Number: "784-1990-1234567-1", ExpiryDate: &expired}, "", &e.a.user); err != nil {
		t.Fatal(err)
	}
	if dk, _ := k.DealKYC(e.a.ctx, deal); dk.CanWin || dk.Missing[0] != kyc.ExpiredID {
		t.Fatalf("with an expired ID = %+v", dk)
	}
	if err := k.AddDocument(e.b.ctx, &KYCDocument{ContactID: ct.ID, DocType: kyc.Passport, Number: "X"}, "", nil); !isNotFound(err) {
		t.Fatalf("B added a document to A's contact: %v", err)
	}
	// The override lets it be won; a valid passport completes the record.
	if err := k.OverrideDealKYC(e.a.ctx, deal, "Client abroad; passport copy promised", e.a.user); err != nil {
		t.Fatal(err)
	}
	if dk, _ := k.DealKYC(e.a.ctx, deal); !dk.CanWin || dk.OverrideReason == "" {
		t.Fatalf("after override = %+v", dk)
	}
	valid := time.Now().AddDate(3, 0, 0)
	if err := k.AddDocument(e.a.ctx, &KYCDocument{ContactID: ct.ID, DocType: kyc.Passport, Number: "N1234567", IssuingCountry: "GB", ExpiryDate: &valid}, "", &e.a.user); err != nil {
		t.Fatal(err)
	}
	if missing, err := k.LeadKYCMissing(e.a.ctx, ld.ID); err != nil || len(missing) != 0 {
		t.Fatalf("lead KYC = %v, %v", missing, err)
	}
	p, err := k.Profile(e.a.ctx, ct.ID)
	if err != nil || len(p.Documents) != 2 || len(p.Missing) != 0 || p.Documents[0].Number != "N1234567" {
		t.Fatalf("profile %+v %v", p, err)
	}
	p.Risk, p.RiskReason, p.IsPEP, p.Screening = "high", "PEP: senior official", true, kyc.PossibleMatch
	if err := k.UpdateProfile(e.a.ctx, p, e.a.user); err != nil {
		t.Fatal(err)
	}
	if err := k.UpdateProfile(e.b.ctx, p, e.b.user); !isNotFound(err) {
		t.Fatalf("B updated A's KYC: %v", err)
	}

	// AML: a transfer is not flagged; cash over the threshold is; a review clears the queue.
	if status, _, err := k.SetDealPayment(e.a.ctx, deal, kyc.PayTransfer, 0); err != nil || status != "" {
		t.Fatalf("transfer: %q %v", status, err)
	}
	status, reasons, err := k.SetDealPayment(e.a.ctx, deal, kyc.PayMixed, 60000)
	if err != nil || status != "flagged" || len(reasons) != 1 {
		t.Fatalf("cash 60,000: %q %v %v", status, reasons, err)
	}
	cases, _ := k.AMLCases(e.a.ctx, "flagged")
	if len(cases) != 1 || cases[0].ContactName != ct.FullName || !cases[0].ContactIsPEP {
		t.Fatalf("queue %+v", cases)
	}
	if cases, _ := k.AMLCases(e.b.ctx, ""); len(cases) != 0 {
		t.Fatal("B sees A's AML cases")
	}
	if err := k.ReviewAML(e.a.ctx, deal, "reported", "STR filed", "GOAML-2026-001", e.a.user); err != nil {
		t.Fatal(err)
	}
	// A reviewed deal keeps its review when the payment changes.
	if status, _, _ := k.SetDealPayment(e.a.ctx, deal, kyc.PayTransfer, 0); status != "reported" {
		t.Fatalf("after review, status %q", status)
	}
	rows, err := k.AMLExport(e.a.ctx, "", time.Now().AddDate(0, 0, -1), time.Now().AddDate(0, 0, 1))
	if err != nil || len(rows) != 1 || rows[0].IDNumber != "N1234567" || rows[0].GoAMLRef != "GOAML-2026-001" || !rows[0].ClientPEP {
		t.Fatalf("export %+v %v", rows, err)
	}

	// Beneficial owners.
	org := uuid.New()
	must(t, e.pool, `INSERT INTO organisations (id, company_id, name) VALUES ($1, $2, 'Holding LLC')`, org, e.a.id)
	pct := 60.0
	if err := k.SetOwner(e.a.ctx, org, ct.ID, &pct, "owner"); err != nil {
		t.Fatal(err)
	}
	if err := k.SetOwner(e.b.ctx, org, ct.ID, &pct, "owner"); !isNotFound(err) {
		t.Fatalf("B linked an owner to A's organisation: %v", err)
	}
	if owners, _ := k.Owners(e.a.ctx, org); len(owners) != 1 || len(owners[0].KYCMissing) != 0 {
		t.Fatalf("owners %+v", owners)
	}
	must(t, e.pool, `DELETE FROM beneficial_owners WHERE organisation_id = $1`, org)
	must(t, e.pool, `DELETE FROM organisations WHERE id = $1`, org)
}
