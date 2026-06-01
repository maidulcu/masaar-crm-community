package repo

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookSubscription struct {
	ID           uuid.UUID
	CompanyID    uuid.UUID
	Name         string
	URL          string
	Events       []string // parsed from comma-separated
	Secret       string
	Active       bool
	CreatedAt    time.Time
	LastFiredAt  *time.Time
	FailureCount int
}

type WebhookRepo struct {
	db *pgxpool.Pool
}

func NewWebhookRepo(db *pgxpool.Pool) *WebhookRepo {
	return &WebhookRepo{db: db}
}

func (r *WebhookRepo) Create(ctx context.Context, companyID uuid.UUID, name, url, events, secret string) (*WebhookSubscription, error) {
	sub := &WebhookSubscription{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO webhook_subscriptions (company_id, name, url, events, secret)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, company_id, name, url, events, secret, active, created_at`,
		companyID, name, url, events, secret,
	).Scan(&sub.ID, &sub.CompanyID, &sub.Name, &sub.URL, new(string), &sub.Secret, &sub.Active, &sub.CreatedAt)
	if err != nil {
		return nil, err
	}
	sub.Events = parseEvents(events)
	return sub, nil
}

func (r *WebhookRepo) List(ctx context.Context, companyID uuid.UUID) ([]WebhookSubscription, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, company_id, name, url, events, secret, active, created_at, last_fired_at, failure_count
		 FROM webhook_subscriptions
		 WHERE company_id = $1
		 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []WebhookSubscription
	for rows.Next() {
		var s WebhookSubscription
		var eventsStr string
		if err := rows.Scan(&s.ID, &s.CompanyID, &s.Name, &s.URL, &eventsStr, &s.Secret, &s.Active, &s.CreatedAt, &s.LastFiredAt, &s.FailureCount); err != nil {
			return nil, err
		}
		s.Events = parseEvents(eventsStr)
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// ListActiveForEvent returns all active subscriptions that listen for a given event.
func (r *WebhookRepo) ListActiveForEvent(ctx context.Context, companyID uuid.UUID, event string) ([]WebhookSubscription, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, company_id, name, url, events, secret, active, created_at
		 FROM webhook_subscriptions
		 WHERE company_id = $1
		   AND active = TRUE
		   AND (events = '' OR events ILIKE $2)`,
		companyID, "%"+event+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []WebhookSubscription
	for rows.Next() {
		var s WebhookSubscription
		var eventsStr string
		if err := rows.Scan(&s.ID, &s.CompanyID, &s.Name, &s.URL, &eventsStr, &s.Secret, &s.Active, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.Events = parseEvents(eventsStr)
		// Double-check exact match (ILIKE can produce false positives on partial names)
		for _, e := range s.Events {
			if e == event || e == "*" {
				subs = append(subs, s)
				break
			}
		}
	}
	return subs, rows.Err()
}

func (r *WebhookRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM webhook_subscriptions WHERE id = $1`, id)
	return err
}

func (r *WebhookRepo) RecordDelivery(ctx context.Context, subscriptionID uuid.UUID, event string, payload []byte, statusCode int, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	r.db.Exec(ctx,
		`INSERT INTO webhook_deliveries (subscription_id, event, payload, status, response_code, attempts, delivered_at)
		 VALUES ($1, $2, $3, $4, $5, 1, NOW())`,
		subscriptionID, event, payload, status, statusCode,
	)
	if success {
		r.db.Exec(ctx, `UPDATE webhook_subscriptions SET last_fired_at = NOW(), failure_count = 0 WHERE id = $1`, subscriptionID)
	} else {
		r.db.Exec(ctx, `UPDATE webhook_subscriptions SET failure_count = failure_count + 1 WHERE id = $1`, subscriptionID)
	}
}

func parseEvents(s string) []string {
	if s == "" {
		return []string{"*"}
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
