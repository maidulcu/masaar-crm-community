package repo

import (
	"testing"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestListingCreatePublishAndPermitOverride(t *testing.T) {
	e := setup(t)
	listings := NewListingRepo(e.pool)
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	// No portal_sync_status: used to fail with a NOT NULL violation.
	l := &domain.Listing{Title: "Villa", PropertyType: "villa", ListingType: domain.ListingTypeSale, Price: 5_000_000,
		Currency: "AED", Emirate: "Dubai", Status: domain.ListingStatusDraft, PermitNumber: "7114", PermitExpiry: &exp}
	if err := listings.Create(e.a.ctx, l); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := listings.UpdateStatus(e.a.ctx, l.ID, domain.ListingStatusPublished); err != nil {
		t.Fatalf("publish: %v", err)
	}
	got, err := listings.GetByID(e.a.ctx, l.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.ListingStatusPublished || got.PublishedAt == nil || got.PermitNumber != "7114" || got.Country != "AE" {
		t.Fatalf("after publish: %+v", got)
	}

	if err := listings.SetPermitOverride(e.b.ctx, l.ID, &e.b.user, "x"); !isNotFound(err) {
		t.Fatalf("B overrode A's permit: %v", err)
	}
	if err := listings.SetPermitOverride(e.a.ctx, l.ID, &e.a.user, "permit renewal pending"); err != nil {
		t.Fatal(err)
	}
	got, _ = listings.GetByID(e.a.ctx, l.ID)
	if got.PermitOverrideBy == nil || got.PermitOverrideReason != "permit renewal pending" || got.PermitOverrideAt == nil {
		t.Fatalf("override not stored: %+v", got)
	}
	if err := listings.SetPermitOverride(e.a.ctx, l.ID, nil, ""); err != nil {
		t.Fatal(err)
	}
	if got, _ = listings.GetByID(e.a.ctx, l.ID); got.PermitOverrideBy != nil || got.PermitOverrideAt != nil {
		t.Fatalf("override not cleared: %+v", got)
	}
}
