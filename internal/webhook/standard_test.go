package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/repo"
)

// The test vector published with the Standard Webhooks reference libraries.
func TestSignStandardReferenceVector(t *testing.T) {
	key := signingKey("whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw")
	got := SignStandard(key, "msg_p5jXN8AQM9LWM0D4loKWxJek", "1614265330", []byte(`{"test": 2432232314}`))
	if want := "v1,g0hM9SsE+OTPJTGt/tmIKtSyZlE3uFJELVlNIOLJ1OE="; got != want {
		t.Fatalf("signature = %s, want %s", got, want)
	}
}

func TestGenerateSecret(t *testing.T) {
	s, err := GenerateSecret()
	if err != nil || !strings.HasPrefix(s, "whsec_") {
		t.Fatalf("secret %q, %v", s, err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, "whsec_"))
	if err != nil || len(key) != 32 {
		t.Fatalf("key: %d bytes, %v", len(key), err)
	}
}

// verify checks a delivery the way a receiver following the spec would, written from the spec
// rather than with SignStandard.
func verify(t *testing.T, secret string, h http.Header, body []byte) {
	t.Helper()
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil {
		t.Fatalf("secret is not base64: %v", err)
	}
	id, ts := h.Get("webhook-id"), h.Get("webhook-timestamp")
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || time.Since(time.Unix(sec, 0)).Abs() > 5*time.Minute {
		t.Fatalf("timestamp %q not within tolerance", ts)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + ts + "." + string(body)))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	for _, sig := range strings.Fields(h.Get("webhook-signature")) {
		if v, b64, ok := strings.Cut(sig, ","); ok && v == "v1" && hmac.Equal([]byte(b64), []byte(want)) {
			return
		}
	}
	t.Fatalf("no valid v1 signature in %q", h.Get("webhook-signature"))
}

type secretStore struct {
	url, secret string
	mu          sync.Mutex
	done        int
}

func (s *secretStore) ListActiveForEvent(_ context.Context, co uuid.UUID, _ string) ([]repo.WebhookSubscription, error) {
	return []repo.WebhookSubscription{{ID: uuid.New(), CompanyID: co, Name: "t", URL: s.url, Secret: s.secret}}, nil
}

func (s *secretStore) RecordDelivery(context.Context, uuid.UUID, string, []byte, int, bool) {
	s.mu.Lock()
	s.done++
	s.mu.Unlock()
}

type delivery struct {
	h    http.Header
	body []byte
}

// A retried delivery keeps its webhook-id, re-signs with a fresh timestamp, and still carries the
// legacy headers; both a whsec_ secret and a pre-whsec_ hex secret verify per the spec.
func TestStandardWebhookDelivery(t *testing.T) {
	for _, secret := range []string{
		"whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw",
		"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08", // created before whsec_
	} {
		var mu sync.Mutex
		var got []delivery
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			got = append(got, delivery{r.Header.Clone(), b})
			first := len(got) == 1
			mu.Unlock()
			if first {
				w.WriteHeader(http.StatusServiceUnavailable) // force one retry
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		store := &secretStore{url: srv.URL, secret: secret}
		d := newTestDispatcher(store)
		d.Dispatch(uuid.New(), EventLeadWon, map[string]string{"lead_id": "1"})
		d.Shutdown(10 * time.Second)
		srv.Close()

		if len(got) != 2 {
			t.Fatalf("%d deliveries, want 2 (one retry)", len(got))
		}
		id := got[0].h.Get("webhook-id")
		if !strings.HasPrefix(id, "msg_") || got[1].h.Get("webhook-id") != id || got[1].h.Get("X-Masaar-Delivery") != id {
			t.Errorf("webhook-id not stable across retries: %q, %q", id, got[1].h.Get("webhook-id"))
		}
		for _, dl := range got {
			verify(t, secret, dl.h, dl.body)
			if !strings.HasPrefix(dl.h.Get("X-Masaar-Signature"), "sha256=") || dl.h.Get("X-Masaar-Event") != EventLeadWon {
				t.Errorf("legacy headers missing: %v", dl.h)
			}
		}
		var p map[string]any
		if err := json.Unmarshal(got[1].body, &p); err != nil || p["type"] != EventLeadWon || p["event"] != EventLeadWon {
			t.Errorf("payload = %s", got[1].body)
		}
	}
}
