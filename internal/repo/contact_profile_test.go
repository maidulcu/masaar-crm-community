package repo

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestOrganisationsAndProfile(t *testing.T) {
	e := setup(t)
	pr := NewContactProfileRepo(e.pool)
	org := &domain.Organisation{Name: "Emaar Brokers LLC", TRN: "100000000000003"}
	if err := pr.SaveOrganisation(e.a.ctx, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = e.pool.Exec(e.a.ctx, `DELETE FROM organisations WHERE company_id = ANY($1)`, []uuid.UUID{e.a.id, e.b.id})
	})
	ct := e.contact(t, e.a, testPhone("97151"))
	p := &domain.ContactProfile{SecondaryPhones: []string{"+971509998877"}, Nationality: "IN", JobTitle: "CFO", OrganisationID: &org.ID}
	if err := pr.UpdateProfile(e.a.ctx, ct.ID, p); err != nil {
		t.Fatal(err)
	}
	got, err := pr.GetProfile(e.a.ctx, ct.ID)
	if err != nil || got.OrganisationName != org.Name || got.Nationality != "IN" || len(got.SecondaryPhones) != 1 {
		t.Fatalf("profile = %+v, %v", got, err)
	}
	if o, _ := pr.GetOrganisation(e.a.ctx, org.ID); o.ContactCount != 1 {
		t.Fatalf("contact count = %d", o.ContactCount)
	}
	// B can't see A's organisation or attach A's contact to it.
	if _, err := pr.GetOrganisation(e.b.ctx, org.ID); !isNotFound(err) {
		t.Fatalf("B read A's organisation: %v", err)
	}
	ctB := e.contact(t, e.b, testPhone("97151"))
	if err := pr.UpdateProfile(e.b.ctx, ctB.ID, &domain.ContactProfile{OrganisationID: &org.ID}); !isNotFound(err) {
		t.Fatalf("B linked A's organisation: %v", err)
	}
}

func TestCustomFields(t *testing.T) {
	e := setup(t)
	pr := NewContactProfileRepo(e.pool)
	t.Cleanup(func() {
		_, _ = e.pool.Exec(e.a.ctx, `DELETE FROM custom_field_definitions WHERE company_id = ANY($1)`, []uuid.UUID{e.a.id, e.b.id})
	})
	budget := &domain.CustomField{EntityType: "contact", Name: "budget", Label: "Budget", Type: "number"}
	visa := &domain.CustomField{EntityType: "contact", Name: "visa", Label: "Visa", Type: "select", Options: []string{"resident", "golden"}}
	for _, f := range []*domain.CustomField{budget, visa} {
		if err := pr.SaveCustomField(e.a.ctx, f); err != nil {
			t.Fatal(err)
		}
	}
	ct := e.contact(t, e.a, testPhone("97151"))
	if err := pr.SetCustomValues(e.a.ctx, "contact", ct.ID, map[string]string{"budget": "1500000", "visa": "golden"}); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]map[string]string{
		"not a number":  {"budget": "a lot"},
		"bad option":    {"visa": "tourist"},
		"unknown field": {"pets": "yes"},
	} {
		if err := pr.SetCustomValues(e.a.ctx, "contact", ct.ID, bad); !errors.Is(err, ErrCustomFieldInvalid) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
	vals, _ := pr.CustomValues(e.a.ctx, "contact", ct.ID)
	if vals["budget"] != "1500000" || vals["visa"] != "golden" {
		t.Fatalf("values = %v", vals)
	}
	// Clearing a value; B can't write values on A's contact; B sees no A fields.
	if err := pr.SetCustomValues(e.a.ctx, "contact", ct.ID, map[string]string{"visa": ""}); err != nil {
		t.Fatal(err)
	}
	if vals, _ := pr.CustomValues(e.a.ctx, "contact", ct.ID); len(vals) != 1 {
		t.Fatalf("after clear = %v", vals)
	}
	if err := pr.SetCustomValues(e.b.ctx, "contact", ct.ID, map[string]string{}); !isNotFound(err) {
		t.Fatalf("B wrote A's contact values: %v", err)
	}
	if fields, _ := pr.ListCustomFields(e.b.ctx, ""); len(fields) != 0 {
		t.Fatalf("B sees A's fields: %+v", fields)
	}
	all, _ := pr.AllCustomValues(e.a.ctx, "contact")
	if all[ct.ID]["budget"] != "1500000" {
		t.Fatalf("all values = %v", all)
	}
}

func TestFindDuplicatesAndMerge(t *testing.T) {
	e := setup(t)
	pr := NewContactProfileRepo(e.pool)
	contacts := NewContactRepo(e.pool)
	a := e.contact(t, e.a, testPhone("97154"))
	b := e.contact(t, e.a, testPhone("97154"))
	a.Email, b.Email = "same-"+a.ID.String()[:6]+"@example.test", "SAME-"+a.ID.String()[:6]+"@example.test"
	_ = contacts.Update(e.a.ctx, a)
	_ = contacts.Update(e.a.ctx, b)
	ld := e.lead(t, e.a, b.ID)
	_ = NewPrivacyRepo(e.pool).RecordConsent(e.a.ctx, &domain.Consent{ContactID: b.ID, Channel: "email", Status: "granted", Source: "manual"})

	groups, err := pr.FindDuplicates(e.a.ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, g := range groups {
		if g.Reason == "email" && len(g.Contacts) == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("email duplicates not found: %+v", groups)
	}
	if groups, _ := pr.FindDuplicates(e.b.ctx, 0); len(groups) != 0 {
		t.Fatalf("B sees A's duplicates: %+v", groups)
	}

	if err := pr.Merge(e.b.ctx, a.ID, []uuid.UUID{b.ID}); !isNotFound(err) {
		t.Fatalf("B merged A's contacts: %v", err)
	}
	if err := pr.Merge(e.a.ctx, a.ID, []uuid.UUID{b.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := contacts.GetByID(e.a.ctx, b.ID); err == nil {
		t.Fatal("merged contact still exists")
	}
	var owner uuid.UUID
	_ = e.pool.QueryRow(e.a.ctx, `SELECT contact_id FROM leads WHERE id = $1`, ld.ID).Scan(&owner)
	if owner != a.ID {
		t.Fatalf("lead not moved to the survivor")
	}
	prof, _ := pr.GetProfile(e.a.ctx, a.ID)
	if len(prof.SecondaryPhones) != 1 || prof.SecondaryPhones[0] != b.PhoneWA {
		t.Fatalf("survivor secondary phones = %v, want the merged contact's number", prof.SecondaryPhones)
	}
	if cur, _ := NewPrivacyRepo(e.pool).CurrentConsent(e.a.ctx, a.ID); cur["email"].Status != "granted" {
		t.Fatal("consent history not moved")
	}
}
