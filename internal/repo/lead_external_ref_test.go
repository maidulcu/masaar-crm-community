package repo

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// A lead pushed twice with the same external ref is stored once per company, and the
// same ref in another company is a different lead.
func TestLeadExternalRefIdempotentPerCompany(t *testing.T) {
	e := setup(t)
	leads := NewLeadRepo(e.pool)

	push := func(c company, contactID uuid.UUID, ref string) (*domain.Lead, error) {
		l := &domain.Lead{ContactID: contactID, Stage: domain.StageNew, Source: domain.SourcePortal, Currency: "AED",
			ExternalSource: "waqov", ExternalRef: ref}
		return l, leads.Create(c.ctx, l)
	}

	ca := e.contact(t, e.a, testPhone("97153"))
	first, err := push(e.a, ca.ID, "enquiry:42")
	if err != nil {
		t.Fatalf("first push: %v", err)
	}
	if _, err := push(e.a, ca.ID, "enquiry:42"); !errors.Is(err, ErrDuplicateExternalRef) {
		t.Fatalf("second push in A: want ErrDuplicateExternalRef, got %v", err)
	}
	got, err := leads.FindByExternalRef(e.a.ctx, "waqov", "enquiry:42")
	if err != nil || got.ID != first.ID {
		t.Fatalf("A finds its lead by ref: %+v %v", got, err)
	}

	// B cannot see A's ref, and can hold its own lead with the same ref.
	if _, err := leads.FindByExternalRef(e.b.ctx, "waqov", "enquiry:42"); !isNotFound(err) {
		t.Fatalf("B found A's lead by ref: %v", err)
	}
	cb := e.contact(t, e.b, testPhone("97154"))
	if _, err := push(e.b, cb.ID, "enquiry:42"); err != nil {
		t.Fatalf("same ref in B: %v", err)
	}

	// A lead on a foreign contact still fails as not found, not as a duplicate.
	if _, err := push(e.b, ca.ID, "enquiry:43"); errors.Is(err, ErrDuplicateExternalRef) || !isNotFound(err) {
		t.Fatalf("B lead on A's contact: want not found, got %v", err)
	}
}

// An API-key action is written to the audit log under the key's company.
func TestAuditLogAPIKeyActor(t *testing.T) {
	e := setup(t)
	keyID, err := NewApiKeyRepo(e.pool).Create(e.a.ctx, e.a.id, "intake", "hash-"+uuid.NewString(), "sk_live_test", "lead:create")
	if err != nil {
		t.Fatal(err)
	}
	audit := NewAuditLogRepo(e.pool)
	entity := uuid.New()
	audit.LogAPIKey(e.a.ctx, keyID, AuditCreate, AuditLead, entity, nil)

	res, err := audit.List(e.a.ctx, AuditLogFilter{EntityID: &entity})
	if err != nil || res.Total != 1 || res.Data[0].APIKeyID == nil || *res.Data[0].APIKeyID != keyID {
		t.Fatalf("A should see the API-key audit entry: %+v %v", res, err)
	}
	if res, _ := audit.List(e.b.ctx, AuditLogFilter{EntityID: &entity}); res == nil || res.Total != 0 {
		t.Fatal("B must not see A's API-key audit entry")
	}
}
