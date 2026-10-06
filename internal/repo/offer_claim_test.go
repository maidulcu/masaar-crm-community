package repo

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestOfferClaimAcceptanceIsExclusive(t *testing.T) {
	e := setup(t)
	listing := e.listing(t, e.a)
	ct := e.contact(t, e.a, testPhone("97150"))
	offers := NewOfferRepo(e.pool)

	o := &domain.Offer{ListingID: listing, ContactID: ct.ID, OfferAmount: 1000}
	if err := offers.Create(e.a.ctx, o); err != nil {
		t.Fatal(err)
	}

	// Another company can neither claim nor see the offer.
	if _, ok, err := offers.ClaimAcceptance(e.b.ctx, o.ID); err != nil || ok {
		t.Fatalf("company B claimed A's offer: ok=%v err=%v", ok, err)
	}

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			prev, ok, err := offers.ClaimAcceptance(e.a.ctx, o.ID)
			if err != nil {
				t.Error(err)
				return
			}
			if ok {
				wins.Add(1)
				if prev != domain.OfferSubmitted {
					t.Errorf("prev status = %q, want submitted", prev)
				}
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("%d concurrent accepts won the claim, want exactly 1", wins.Load())
	}

	got, err := offers.GetByID(e.a.ctx, o.ID)
	if err != nil || got.Status != domain.OfferAccepted {
		t.Fatalf("offer after claim: %v %v", got, err)
	}

	// Restoring the previous status re-opens it (used when deal creation fails).
	if err := offers.UpdateStatus(e.a.ctx, o.ID, domain.OfferSubmitted); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := offers.ClaimAcceptance(e.a.ctx, o.ID); !ok {
		t.Fatal("offer should be claimable again after restore")
	}
	// Rejected/expired offers cannot be claimed.
	_ = offers.UpdateStatus(e.a.ctx, o.ID, domain.OfferRejected)
	if _, ok, _ := offers.ClaimAcceptance(e.a.ctx, o.ID); ok {
		t.Fatal("rejected offer must not be claimable")
	}
}
