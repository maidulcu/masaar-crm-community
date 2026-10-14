package repo

import (
	"testing"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestReraFormPrefillAndStore(t *testing.T) {
	e := setup(t)
	rr := NewReraRepo(e.pool)
	listing := e.listing(t, e.a)
	must(t, e.pool, `UPDATE listings SET title = 'Marina 2BR', emirate = 'Dubai', owner_name = 'Ahmed', permit_number = '7114', price = 2000000 WHERE id = $1`, listing)
	must(t, e.pool, `UPDATE company_settings SET orn = '12345' WHERE company_id = $1`, e.a.id)
	must(t, e.pool, `UPDATE users SET brn = 'BRN-1' WHERE id = $1`, e.a.user)
	ct := e.contact(t, e.a, testPhone("97155"))

	v, err := rr.Prefill(e.a.ctx, &listing, nil, nil, e.a.user)
	if err != nil {
		t.Fatal(err)
	}
	if v["property_title"] != "Marina 2BR" || v["owner_name"] != "Ahmed" || v["permit_number"] != "7114" ||
		v["broker_orn"] != "12345" || v["agent_brn"] != "BRN-1" || v["start_date"] == "" {
		t.Fatalf("prefill = %v", v)
	}
	if _, err := rr.Prefill(e.b.ctx, &listing, nil, nil, e.b.user); !isNotFound(err) {
		t.Fatalf("B pre-filled from A's listing: %v", err)
	}
	o := &domain.Offer{ListingID: listing, ContactID: ct.ID, OfferAmount: 1900000, AgentID: &e.a.user}
	if err := NewOfferRepo(e.pool).Create(e.a.ctx, o); err != nil {
		t.Fatal(err)
	}
	v, err = rr.Prefill(e.a.ctx, nil, nil, &o.ID, e.a.user)
	if err != nil || v["sale_price"] != "1900000" || v["buyer_name"] != ct.FullName || v["owner_name"] != "Ahmed" || v["deposit_amount"] != "190000.00" {
		t.Fatalf("offer prefill = %v, %v", v, err)
	}

	f := &ReraForm{FormType: "A", ListingID: &listing, Data: map[string]string{"owner_name": "Ahmed"}, Title: "Form A"}
	if err := rr.Create(e.a.ctx, f, e.a.user, "listing", &listing); err != nil {
		t.Fatal(err)
	}
	got, err := rr.Get(e.a.ctx, f.DocumentID)
	if err != nil || got.Data["owner_name"] != "Ahmed" || got.SignatureStatus != "pending" {
		t.Fatalf("get = %+v, %v", got, err)
	}
	if list, _ := rr.List(e.a.ctx, &listing, nil); len(list) != 1 {
		t.Fatalf("list = %d", len(list))
	}
	if _, err := rr.Get(e.b.ctx, f.DocumentID); !isNotFound(err) {
		t.Fatalf("B read A's form: %v", err)
	}
	if pub, err := rr.ForDocument(e.a.ctx, f.DocumentID); err != nil || pub.FormType != "A" {
		t.Fatalf("for document = %+v, %v", pub, err)
	}
	t.Cleanup(func() { _, _ = e.pool.Exec(e.a.ctx, `DELETE FROM documents WHERE company_id = $1`, e.a.id) })
}
