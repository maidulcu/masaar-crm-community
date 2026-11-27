package repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// A suspended company disappears from every public surface: its share pages, its portal
// feed and lead intake, and its signing links.
func TestSuspendedCompany_IsHiddenPublicly(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	co := e.a

	listings, docs := NewListingRepo(e.pool), NewDocumentRepo(e.pool)
	l := &domain.Listing{Title: "L", PropertyType: "apartment", Price: 1, Currency: "AED"}
	if err := listings.Create(co.ctx, l); err != nil {
		t.Fatal(err)
	}
	listingID := l.ID
	must(t, e.pool, `UPDATE listings SET status = 'published' WHERE id = $1`, listingID)
	docID, sigID := uuid.New(), uuid.New()
	must(t, e.pool, `INSERT INTO documents (id, company_id, document_title, signature_status, created_by) VALUES ($1, $2, 'Offer', 'pending', $3)`, docID, co.id, co.user)
	must(t, e.pool, `INSERT INTO document_signatures (id, document_id, signer_name, signer_email, signature_status) VALUES ($1, $2, 'S', 's@example.com', 'pending')`, sigID, docID)
	t.Cleanup(func() {
		_, _ = e.pool.Exec(ctx, `DELETE FROM document_signatures WHERE document_id = $1`, docID)
		_, _ = e.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, docID)
	})
	portals := NewPortalRepo(e.pool)
	inbound, feed, err := portals.Connect(co.ctx, "bayut")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := listings.GetPublicByID(ctx, listingID); err != nil {
		t.Fatalf("active company's published listing: %v", err)
	}
	if !docs.CompanyActive(ctx, co.id) {
		t.Fatal("active company reported inactive")
	}

	must(t, e.pool, `UPDATE companies SET is_active = false WHERE id = $1`, co.id)

	if _, err := listings.GetPublicByID(ctx, listingID); err == nil {
		t.Error("suspended company's listing is still public")
	}
	for kind, tok := range map[string]string{"inbound": inbound, "feed": feed} {
		if _, err := portals.CompanyForToken(ctx, "bayut", kind, tok); !errors.Is(err, ErrPortalCompanySuspended) {
			t.Errorf("%s token of a suspended company: %v, want ErrPortalCompanySuspended", kind, err)
		}
	}
	if docs.CompanyActive(ctx, co.id) {
		t.Error("suspended company reported active")
	}
	if err := docs.SignByToken(ctx, sigID, time.Now(), "203.0.113.1", "ua"); !errors.Is(err, ErrNotSignable) {
		t.Errorf("signing a suspended company's document: %v, want ErrNotSignable", err)
	}
}
