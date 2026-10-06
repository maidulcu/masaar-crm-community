package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/repo"
)

type fakeStore struct {
	url string

	mu         sync.Mutex
	deliveries int
	lastStatus int
}

func (f *fakeStore) ListActiveForEvent(_ context.Context, companyID uuid.UUID, _ string) ([]repo.WebhookSubscription, error) {
	return []repo.WebhookSubscription{{ID: uuid.New(), CompanyID: companyID, Name: "t", URL: f.url, Secret: "s3cret"}}, nil
}

func (f *fakeStore) RecordDelivery(_ context.Context, _ uuid.UUID, _ string, _ []byte, status int, _ bool) {
	f.mu.Lock()
	f.deliveries++
	f.lastStatus = status
	f.mu.Unlock()
}

func (f *fakeStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.deliveries
}

// The SSRF-safe client refuses loopback, so tests talk to httptest servers with a plain client.
func newTestDispatcher(store SubscriptionStore) *Dispatcher {
	d := NewDispatcher(store)
	d.client = &http.Client{Timeout: 5 * time.Second}
	return d
}

func TestDispatchDeliversSignedPayload(t *testing.T) {
	got := make(chan *http.Request, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r
		w.WriteHeader(204)
	}))
	defer srv.Close()

	store := &fakeStore{url: srv.URL}
	d := newTestDispatcher(store)
	d.Dispatch(uuid.New(), EventLeadCreated, map[string]string{"id": "1"})

	select {
	case r := <-got:
		if r.Header.Get("X-Masaar-Event") != EventLeadCreated || !strings.HasPrefix(r.Header.Get("X-Masaar-Signature"), "sha256=") {
			t.Fatalf("headers = %v", r.Header)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("webhook was not delivered")
	}
	d.Shutdown(3 * time.Second)
	if store.count() != 1 || store.lastStatus != 204 {
		t.Fatalf("recorded %d deliveries, status %d; want 1 / 204", store.count(), store.lastStatus)
	}
}

// A burst of events must neither start a goroutine per event nor block the caller: at most
// dispatchWorkers deliveries run at once and the overflow beyond the queue is dropped.
func TestDispatchIsBounded(t *testing.T) {
	release := make(chan struct{})
	var inFlight, peak atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		<-release
		inFlight.Add(-1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	d := newTestDispatcher(&fakeStore{url: srv.URL})
	company := uuid.New()

	start := time.Now()
	const burst = dispatchQueueSize + dispatchWorkers + 500
	for i := 0; i < burst; i++ {
		d.Dispatch(company, EventLeadCreated, i)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Dispatch blocked the caller for %v", elapsed)
	}

	// let the workers saturate, then check the ceiling
	deadline := time.Now().Add(3 * time.Second)
	for inFlight.Load() < dispatchWorkers && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	if p := peak.Load(); p > dispatchWorkers {
		t.Fatalf("%d concurrent deliveries, limit is %d", p, dispatchWorkers)
	}
	close(release)
	d.Shutdown(30 * time.Second)
}

func TestDispatchAfterShutdownIsIgnored(t *testing.T) {
	d := newTestDispatcher(&fakeStore{url: "http://127.0.0.1:1"})
	d.Shutdown(time.Second)
	d.Dispatch(uuid.New(), EventLeadCreated, nil) // must not panic on the closed queue
	d.Shutdown(time.Second)                       // idempotent
}
