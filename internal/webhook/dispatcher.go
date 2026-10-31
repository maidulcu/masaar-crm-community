package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/safehttp"
)

// Supported event names — use these constants in handlers.
const (
	EventLeadCreated      = "lead.created"
	EventLeadStageChanged = "lead.stage_changed"
	EventLeadWon          = "lead.won"
	EventLeadLost         = "lead.lost"
	EventPaymentReceived  = "payment.received"
	EventLeaseSigned      = "lease.signed"
	EventContactCreated   = "contact.created"
	EventTest             = "webhook.test"
)

// KnownEvent reports whether a subscription may name event: a supported event (including the
// reserved ones) or "*" for all.
func KnownEvent(event string) bool {
	switch event {
	case "*", EventLeadCreated, EventLeadStageChanged, EventLeadWon, EventLeadLost,
		EventPaymentReceived, EventLeaseSigned, EventContactCreated, EventTest:
		return true
	}
	return false
}

// EventInfo describes an event Masaar sends, for the docs and the OpenAPI webhooks section.
type EventInfo struct {
	Name, Description string
	Data              map[string]any // JSON Schema of the "data" field
}

var (
	uuidProp   = map[string]any{"type": "string", "format": "uuid"}
	stringProp = map[string]any{"type": "string"}
)

// SentEvents are the events Masaar sends today. payment.received, lease.signed and
// contact.created are reserved names that nothing sends yet.
var SentEvents = []EventInfo{
	{EventLeadCreated, "A lead was created: in the app, through the API, the lead intake or a portal.", map[string]any{
		"type": "object", "required": []any{"lead_id", "contact_id", "source"},
		"properties": map[string]any{
			"lead_id": uuidProp, "contact_id": uuidProp, "stage": stringProp, "source": stringProp,
			"deal_value":      map[string]any{"type": "number"},
			"contact":         map[string]any{"type": "object", "properties": map[string]any{"name": stringProp, "phone": stringProp}},
			"external_source": stringProp, "external_ref": stringProp,
		},
	}},
	{EventLeadStageChanged, "A lead moved to another pipeline stage.", stageData},
	{EventLeadWon, "A lead moved to a won stage (sent after lead.stage_changed).", stageData},
	{EventLeadLost, "A lead moved to a lost stage (sent after lead.stage_changed).", stageData},
	{EventTest, "A test ping from POST /api/v1/settings/webhooks/{id}/test.", map[string]any{"type": "object"}},
}

var stageData = map[string]any{
	"type": "object", "required": []any{"lead_id", "stage", "previous_stage"},
	"properties": map[string]any{"lead_id": uuidProp, "stage": stringProp, "previous_stage": stringProp},
}

// Payload is the envelope sent to registered webhook URLs. "type" is the Standard Webhooks
// name for the event; "event" carries the same value for receivers written before it.
type Payload struct {
	Type      string      `json:"type"`
	Event     string      `json:"event"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

const (
	// dispatchWorkers bounds how many events are delivered at once. Each delivery can retry
	// for several seconds, so an unbounded goroutine per event let a burst of lead changes
	// (an import, a flood on the public intake endpoint) exhaust memory and sockets.
	dispatchWorkers = 8
	// dispatchQueueSize is how many events may wait for a worker. Beyond it new events are
	// dropped (and logged): webhooks are best effort and must never block the request path.
	dispatchQueueSize = 1024
	// eventTimeout bounds delivery of one event to all of its subscribers, retries included.
	eventTimeout = 60 * time.Second
)

// SubscriptionStore is the part of the webhook repository the dispatcher uses.
type SubscriptionStore interface {
	ListActiveForEvent(ctx context.Context, companyID uuid.UUID, event string) ([]repo.WebhookSubscription, error)
	RecordDelivery(ctx context.Context, subscriptionID uuid.UUID, event string, payload []byte, statusCode int, success bool)
}

type job struct {
	companyID uuid.UUID
	event     string
	msgID     string // webhook-id: one per event, the same on every retry and subscriber
	body      []byte
}

// Dispatcher fires outbound webhooks for CRM events through a fixed pool of workers.
type Dispatcher struct {
	repo   SubscriptionStore
	client *http.Client

	mu     sync.RWMutex // guards closed and sends on queue
	closed bool
	queue  chan job
	wg     sync.WaitGroup
}

func NewDispatcher(store SubscriptionStore) *Dispatcher {
	d := &Dispatcher{
		repo:   store,
		client: safehttp.NewClient(10 * time.Second), // refuses internal addresses (SSRF)
		queue:  make(chan job, dispatchQueueSize),
	}
	d.wg.Add(dispatchWorkers)
	for i := 0; i < dispatchWorkers; i++ {
		go d.worker()
	}
	return d
}

// Dispatch queues webhooks for the given event and returns immediately.
// companyID scopes which subscriptions to notify.
// data is JSON-serialisable and becomes the "data" field in the payload; it is serialised
// here, so later changes to the caller's value cannot leak into (or race with) the delivery.
func (d *Dispatcher) Dispatch(companyID uuid.UUID, event string, data interface{}) {
	body, err := json.Marshal(Payload{
		Type:      event,
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	})
	if err != nil {
		log.Printf("webhook dispatcher: marshal payload: %v", err)
		return
	}

	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.closed {
		return
	}
	select {
	case d.queue <- job{companyID: companyID, event: event, msgID: "msg_" + uuid.NewString(), body: body}:
	default:
		log.Printf("webhook dispatcher: queue full, dropping %s event", event)
	}
}

// Shutdown stops accepting events, lets queued ones finish and waits up to timeout.
func (d *Dispatcher) Shutdown(timeout time.Duration) {
	d.mu.Lock()
	if !d.closed {
		d.closed = true
		close(d.queue)
	}
	d.mu.Unlock()

	done := make(chan struct{})
	go func() { d.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(timeout):
		log.Printf("webhook dispatcher: shutdown timed out with deliveries still running")
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for j := range d.queue {
		d.dispatch(j)
	}
}

func (d *Dispatcher) dispatch(j job) {
	// The timeout allows for 3 attempts with backoff (1s, 4s) per subscriber plus request overhead.
	ctx, cancel := context.WithTimeout(context.Background(), eventTimeout)
	defer cancel()

	subs, err := d.repo.ListActiveForEvent(ctx, j.companyID, j.event)
	if err != nil {
		log.Printf("webhook dispatcher: list subscriptions: %v", err)
		return
	}
	for _, sub := range subs {
		d.send(ctx, sub, j)
	}
}

// send delivers a signed webhook to a single subscription with up to 3 retries.
func (d *Dispatcher) send(ctx context.Context, sub repo.WebhookSubscription, j job) {
	event, body := j.event, j.body
	sig := sign(body, sub.Secret)
	key := signingKey(sub.Secret)

	var (
		statusCode int
		success    bool
	)

	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(body))
		if err != nil {
			log.Printf("webhook [%s] %s: create request: %v", sub.Name, event, err)
			break
		}

		// Standard Webhooks (https://www.standardwebhooks.com): the timestamp is the attempt's,
		// so a receiver's replay window also accepts retries.
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("webhook-id", j.msgID)
		req.Header.Set("webhook-timestamp", ts)
		req.Header.Set("webhook-signature", SignStandard(key, j.msgID, ts, body))
		// Masaar's original headers, kept for receivers that verify them.
		req.Header.Set("X-Masaar-Signature", "sha256="+sig)
		req.Header.Set("X-Masaar-Event", event)
		req.Header.Set("X-Masaar-Delivery", j.msgID)
		req.Header.Set("User-Agent", "Masaar-CRM-Webhooks/1.0")

		resp, err := d.client.Do(req)
		if err != nil {
			log.Printf("webhook [%s] %s attempt %d: %v", sub.Name, event, attempt, err)
		} else {
			// Drain (a bounded amount) so the connection can be reused.
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			resp.Body.Close()

			statusCode = resp.StatusCode
			success = resp.StatusCode >= 200 && resp.StatusCode < 300
			if success {
				break
			}
			log.Printf("webhook [%s] %s attempt %d: got HTTP %d", sub.Name, event, attempt, statusCode)
		}

		if attempt < 3 && !sleepCtx(ctx, time.Duration(attempt*attempt)*time.Second) { // 1s, 4s
			break
		}
	}

	d.repo.RecordDelivery(context.Background(), sub.ID, event, body, statusCode, success)
}

// sleepCtx waits for d or until ctx is done; it reports whether the full wait elapsed.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// sign computes the legacy X-Masaar-Signature value: hex HMAC-SHA256 of body, keyed with the
// secret string as is.
func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// signingKey derives the Standard Webhooks key from a subscription secret the way the standard
// libraries do: drop a "whsec_" prefix, then base64-decode. Secrets created before whsec_
// (64 hex characters) are valid base64 too, so those libraries accept them unchanged.
func signingKey(secret string) []byte {
	s := strings.TrimPrefix(secret, "whsec_")
	if key, err := base64.StdEncoding.DecodeString(s); err == nil && len(key) > 0 {
		return key
	}
	return []byte(s)
}

// SignStandard returns the webhook-signature header value: "v1," + base64 HMAC-SHA256 of
// "<msgID>.<timestamp>.<body>".
func SignStandard(key []byte, msgID, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(msgID + "." + timestamp + "."))
	mac.Write(body)
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// GenerateSecret creates a new subscription secret: "whsec_" + 32 random bytes, base64.
func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return "whsec_" + base64.StdEncoding.EncodeToString(b), nil
}
