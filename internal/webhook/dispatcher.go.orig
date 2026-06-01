package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
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
)

// Payload is the envelope sent to registered webhook URLs.
type Payload struct {
	Event     string      `json:"event"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Dispatcher fires outbound webhooks for CRM events.
type Dispatcher struct {
	repo   *repo.WebhookRepo
	client *http.Client
}

func NewDispatcher(webhookRepo *repo.WebhookRepo) *Dispatcher {
	return &Dispatcher{
		repo: webhookRepo,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Dispatch fires webhooks for the given event asynchronously.
// companyID scopes which subscriptions to notify.
// data is JSON-serialisable and becomes the "data" field in the payload.
func (d *Dispatcher) Dispatch(companyID uuid.UUID, event string, data interface{}) {
	go d.dispatch(companyID, event, data)
}

func (d *Dispatcher) dispatch(companyID uuid.UUID, event string, data interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subs, err := d.repo.ListActiveForEvent(ctx, companyID, event)
	if err != nil {
		log.Printf("webhook dispatcher: list subscriptions: %v", err)
		return
	}

	if len(subs) == 0 {
		return
	}

	payload := Payload{
		Event:     event,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook dispatcher: marshal payload: %v", err)
		return
	}

	for _, sub := range subs {
		d.send(ctx, sub, event, body)
	}
}

// send delivers a signed webhook to a single subscription with up to 3 retries.
func (d *Dispatcher) send(ctx context.Context, sub repo.WebhookSubscription, event string, body []byte) {
	sig := sign(body, sub.Secret)

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

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Masaar-Signature", "sha256="+sig)
		req.Header.Set("X-Masaar-Event", event)
		req.Header.Set("X-Masaar-Delivery", uuid.New().String())
		req.Header.Set("User-Agent", "Masaar-CRM-Webhooks/1.0")

		resp, err := d.client.Do(req)
		if err != nil {
			log.Printf("webhook [%s] %s attempt %d: %v", sub.Name, event, attempt, err)
			if attempt < 3 {
				time.Sleep(time.Duration(attempt*attempt) * time.Second) // 1s, 4s
			}
			continue
		}
		resp.Body.Close()

		statusCode = resp.StatusCode
		success = resp.StatusCode >= 200 && resp.StatusCode < 300
		if success {
			break
		}

		log.Printf("webhook [%s] %s attempt %d: got HTTP %d", sub.Name, event, attempt, statusCode)
		if attempt < 3 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}
	}

	d.repo.RecordDelivery(context.Background(), sub.ID, event, body, statusCode, success)
}

// sign computes HMAC-SHA256 of body using secret.
func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateSecret creates a 32-byte random hex string for a new subscription.
func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}
