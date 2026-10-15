package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/bizhours"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// BroadcastRepo stores WhatsApp broadcasts and their recipients, and the inbox settings
// (sending rate, business-hours auto-reply).
type BroadcastRepo struct {
	db *pgxpool.Pool
}

func NewBroadcastRepo(db *pgxpool.Pool) *BroadcastRepo { return &BroadcastRepo{db: db} }

// Segment selects a broadcast's audience. Empty fields do not filter; a segment with no
// filter at all is every contact.
type Segment struct {
	Tag       string     `json:"tag,omitempty"`        // a lead tag
	LeadStage string     `json:"lead_stage,omitempty"` // contacts with a lead at this stage
	Source    string     `json:"source,omitempty"`     // contacts with a lead from this source
	ListingID *uuid.UUID `json:"listing_id,omitempty"` // contacts with an offer or viewing on this listing
}

// audience returns the WHERE clause (on contacts c) for a segment, with its arguments
// numbered from $2 ($1 is the company).
func (s Segment) audience() (string, []any) {
	where := []string{"c.company_id = $1", "c.erased_at IS NULL", "c.phone_wa <> ''"}
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)+1))
	}
	if s.Tag != "" {
		add(`EXISTS (SELECT 1 FROM lead_tags lt JOIN leads l ON l.id = lt.lead_id
			WHERE l.contact_id = c.id AND l.deleted_at IS NULL AND lt.tag = $%d)`, s.Tag)
	}
	if s.LeadStage != "" {
		add(`EXISTS (SELECT 1 FROM leads l WHERE l.contact_id = c.id AND l.deleted_at IS NULL AND l.stage = $%d)`, s.LeadStage)
	}
	if s.Source != "" {
		add(`EXISTS (SELECT 1 FROM leads l WHERE l.contact_id = c.id AND l.deleted_at IS NULL AND l.source = $%d)`, s.Source)
	}
	if s.ListingID != nil {
		add(`(EXISTS (SELECT 1 FROM offers o WHERE o.contact_id = c.id AND o.listing_id = $%[1]d)
			OR EXISTS (SELECT 1 FROM viewings v WHERE v.contact_id = c.id AND v.listing_id = $%[1]d))`, *s.ListingID)
	}
	return strings.Join(where, " AND "), args
}

// latestWhatsAppConsent is the contact's most recent WhatsApp choice (NULL: never asked).
const latestWhatsAppConsent = `(SELECT cc.status FROM contact_consents cc WHERE cc.contact_id = c.id AND cc.channel = 'whatsapp'
	ORDER BY cc.created_at DESC, cc.id DESC LIMIT 1)`

// AudienceCount is who a segment reaches.
type AudienceCount struct {
	Total     int `json:"total"`
	Eligible  int `json:"eligible"`   // granted WhatsApp consent
	NoConsent int `json:"no_consent"` // never agreed: skipped
	OptedOut  int `json:"opted_out"`  // withdrew: skipped
}

// Preview counts a segment's audience. Only contacts who granted WhatsApp consent are
// messaged.
func (r *BroadcastRepo) Preview(ctx context.Context, s Segment) (AudienceCount, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return AudienceCount{}, err
	}
	where, args := s.audience()
	var a AudienceCount
	err = r.db.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE consent = 'granted'),
		COUNT(*) FILTER (WHERE consent IS NULL), COUNT(*) FILTER (WHERE consent = 'withdrawn')
		FROM (SELECT `+latestWhatsAppConsent+` AS consent FROM contacts c WHERE `+where+`) x`,
		append([]any{cid}, args...)...).Scan(&a.Total, &a.Eligible, &a.NoConsent, &a.OptedOut)
	return a, err
}

// Broadcast is a WhatsApp broadcast with its progress.
type Broadcast struct {
	ID           uuid.UUID      `json:"id"`
	Name         string         `json:"name"`
	TemplateName string         `json:"template_name"`
	Language     string         `json:"language"`
	Params       []string       `json:"params"`
	Segment      Segment        `json:"segment"`
	Status       string         `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	FinishedAt   *time.Time     `json:"finished_at,omitempty"`
	Counts       map[string]int `json:"counts"`
}

const broadcastCols = `b.id, b.name, b.template_name, b.language, b.params, b.segment, b.status, b.created_at, b.started_at, b.finished_at,
	COALESCE((SELECT jsonb_object_agg(status, n) FROM (SELECT status, COUNT(*) n FROM whatsapp_broadcast_recipients
	           WHERE broadcast_id = b.id GROUP BY status) s), '{}')`

func scanBroadcast(row pgx.Row, b *Broadcast) error {
	var seg, counts []byte
	if err := row.Scan(&b.ID, &b.Name, &b.TemplateName, &b.Language, &b.Params, &seg, &b.Status, &b.CreatedAt,
		&b.StartedAt, &b.FinishedAt, &counts); err != nil {
		return err
	}
	_ = json.Unmarshal(seg, &b.Segment)
	b.Counts = map[string]int{}
	_ = json.Unmarshal(counts, &b.Counts)
	return nil
}

// Create stores a draft broadcast.
func (r *BroadcastRepo) Create(ctx context.Context, b *Broadcast, by *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if b.Segment.ListingID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM listings WHERE id = $1 AND company_id = $2)`, *b.Segment.ListingID, cid).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return ErrForeignReference
		}
	}
	seg, _ := json.Marshal(b.Segment)
	if b.Params == nil {
		b.Params = []string{}
	}
	b.ID, b.Status = uuid.New(), "draft"
	return r.db.QueryRow(ctx, `INSERT INTO whatsapp_broadcasts (id, company_id, name, template_name, language, params, segment, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING created_at`,
		b.ID, cid, b.Name, b.TemplateName, b.Language, b.Params, seg, by).Scan(&b.CreatedAt)
}

// Get returns one broadcast.
func (r *BroadcastRepo) Get(ctx context.Context, id uuid.UUID) (*Broadcast, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	b := &Broadcast{}
	if err := scanBroadcast(r.db.QueryRow(ctx, `SELECT `+broadcastCols+` FROM whatsapp_broadcasts b WHERE b.id = $1 AND b.company_id = $2`, id, cid), b); err != nil {
		return nil, err
	}
	return b, nil
}

// List returns the company's broadcasts, newest first.
func (r *BroadcastRepo) List(ctx context.Context) ([]Broadcast, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT `+broadcastCols+` FROM whatsapp_broadcasts b WHERE b.company_id = $1 ORDER BY b.created_at DESC LIMIT 200`, cid)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Broadcast, error) {
		var b Broadcast
		err := scanBroadcast(row, &b)
		return b, err
	})
	if out == nil {
		out = []Broadcast{}
	}
	return out, err
}

// Start snapshots the audience: contacts with granted WhatsApp consent are queued, the
// others are recorded as skipped with the reason. Only a draft can start.
func (r *BroadcastRepo) Start(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var status string
	var segRaw []byte
	if err := tx.QueryRow(ctx, `SELECT status, segment FROM whatsapp_broadcasts WHERE id = $1 AND company_id = $2 FOR UPDATE`, id, cid).Scan(&status, &segRaw); err != nil {
		return err
	}
	if status != "draft" {
		return ErrInvalidTransition
	}
	var seg Segment
	_ = json.Unmarshal(segRaw, &seg)
	where, args := seg.audience()
	n := len(args) + 2
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO whatsapp_broadcast_recipients (broadcast_id, contact_id, company_id, phone, status, skip_reason)
		SELECT $%d, c.id, c.company_id, c.phone_wa,
		       CASE WHEN consent = 'granted' THEN 'pending' ELSE 'skipped' END,
		       CASE WHEN consent = 'granted' THEN '' WHEN consent = 'withdrawn' THEN 'opted_out' ELSE 'no_consent' END
		  FROM (SELECT c.*, `+latestWhatsAppConsent+` AS consent FROM contacts c WHERE `+where+`) c`, n),
		append(append([]any{cid}, args...), id)...); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE whatsapp_broadcasts SET status = 'sending', started_at = NOW() WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Cancel stops a broadcast; recipients not yet messaged are skipped.
func (r *BroadcastRepo) Cancel(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	tag, err := tx.Exec(ctx, `UPDATE whatsapp_broadcasts SET status = 'cancelled', finished_at = NOW()
		WHERE id = $1 AND company_id = $2 AND status IN ('draft','sending')`, id, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}
	if _, err := tx.Exec(ctx, `UPDATE whatsapp_broadcast_recipients SET status = 'skipped', skip_reason = 'cancelled', updated_at = NOW()
		WHERE broadcast_id = $1 AND status = 'pending'`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// BroadcastSend is one message to send.
type BroadcastSend struct {
	BroadcastID  uuid.UUID
	ContactID    uuid.UUID
	Phone        string
	Name         string
	TemplateName string
	Language     string
	Params       []string
}

// ClaimBatch marks up to n queued recipients of the company's sending broadcasts as being
// sent and returns them; concurrent runners never get the same recipient. Recipients left
// "sending" for 10 minutes (a crash mid-send) are marked failed rather than retried, so
// nobody is messaged twice.
func (r *BroadcastRepo) ClaimBatch(ctx context.Context, n int) ([]BroadcastSend, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := r.db.Exec(ctx, `UPDATE whatsapp_broadcast_recipients SET status = 'failed', error = 'interrupted', updated_at = NOW()
		WHERE company_id = $1 AND status = 'sending' AND updated_at < NOW() - INTERVAL '10 minutes'`, cid); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		WITH picked AS (
			SELECT r.broadcast_id, r.contact_id FROM whatsapp_broadcast_recipients r
			  JOIN whatsapp_broadcasts b ON b.id = r.broadcast_id AND b.status = 'sending'
			 WHERE r.company_id = $1 AND r.status = 'pending'
			 ORDER BY b.started_at, r.contact_id LIMIT $2 FOR UPDATE OF r SKIP LOCKED)
		UPDATE whatsapp_broadcast_recipients r SET status = 'sending', updated_at = NOW()
		  FROM picked, whatsapp_broadcasts b, contacts c
		 WHERE r.broadcast_id = picked.broadcast_id AND r.contact_id = picked.contact_id
		   AND b.id = r.broadcast_id AND c.id = r.contact_id
		RETURNING r.broadcast_id, r.contact_id, r.phone, c.full_name, b.template_name, b.language, b.params`, cid, n)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (BroadcastSend, error) {
		var s BroadcastSend
		err := row.Scan(&s.BroadcastID, &s.ContactID, &s.Phone, &s.Name, &s.TemplateName, &s.Language, &s.Params)
		return s, err
	})
}

// StillConsents reports whether the contact's latest WhatsApp choice is still "granted"
// and the contact still exists unerased (an opt-out may arrive while a broadcast runs).
func (r *BroadcastRepo) StillConsents(ctx context.Context, contactID uuid.UUID) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	err = r.db.QueryRow(ctx, `SELECT COALESCE((SELECT `+latestWhatsAppConsent+` = 'granted' FROM contacts c
		WHERE c.id = $1 AND c.company_id = $2 AND c.erased_at IS NULL), FALSE)`, contactID, cid).Scan(&ok)
	return ok, err
}

// FinishSend records the outcome for one recipient.
func (r *BroadcastRepo) FinishSend(ctx context.Context, s BroadcastSend, status, waMessageID, errMsg string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	var waid *string
	if waMessageID != "" {
		waid = &waMessageID
	}
	reason := ""
	if status == "skipped" {
		reason, errMsg = errMsg, ""
	}
	_, err = r.db.Exec(ctx, `UPDATE whatsapp_broadcast_recipients SET status = $1, wa_message_id = $2, error = $3, skip_reason = $4,
		sent_at = CASE WHEN $1 = 'sent' THEN NOW() ELSE sent_at END, updated_at = NOW()
		WHERE broadcast_id = $5 AND contact_id = $6 AND company_id = $7`, status, waid, errMsg, reason, s.BroadcastID, s.ContactID, cid)
	return err
}

// CompleteFinished marks sending broadcasts with nothing left to send as done.
func (r *BroadcastRepo) CompleteFinished(ctx context.Context) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `UPDATE whatsapp_broadcasts b SET status = 'done', finished_at = NOW()
		WHERE b.company_id = $1 AND b.status = 'sending'
		  AND NOT EXISTS (SELECT 1 FROM whatsapp_broadcast_recipients r WHERE r.broadcast_id = b.id AND r.status IN ('pending','sending'))`, cid)
	return err
}

// ApplyStatus records a delivery receipt for a broadcast message; the status only moves
// forward (sent → delivered → read; failed only from sent). It returns false when the
// message is not a broadcast's.
func (r *BroadcastRepo) ApplyStatus(ctx context.Context, waMessageID, status, errMsg string) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE whatsapp_broadcast_recipients SET status = $1, error = CASE WHEN $1 = 'failed' THEN $2 ELSE error END, updated_at = NOW()
		 WHERE wa_message_id = $3 AND company_id = $4
		   AND CASE WHEN $1 = 'failed' THEN status = 'sent'
		            ELSE COALESCE(array_position(ARRAY['sent','delivered','read'], $1::text), 0)
		               > COALESCE(array_position(ARRAY['sent','delivered','read'], status), 99) END`,
		status, errMsg, waMessageID, cid)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// Recipient is one row of a broadcast's delivery report.
type Recipient struct {
	ContactID  uuid.UUID  `json:"contact_id"`
	Name       string     `json:"name"`
	Phone      string     `json:"phone"`
	Status     string     `json:"status"`
	SkipReason string     `json:"skip_reason,omitempty"`
	Error      string     `json:"error,omitempty"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
}

// Recipients lists a broadcast's recipients, optionally by status.
func (r *BroadcastRepo) Recipients(ctx context.Context, id uuid.UUID, status string, limit, offset int) ([]Recipient, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT r.contact_id, c.full_name, r.phone, r.status, r.skip_reason, r.error, r.sent_at
		  FROM whatsapp_broadcast_recipients r JOIN contacts c ON c.id = r.contact_id
		 WHERE r.broadcast_id = $1 AND r.company_id = $2 AND ($3 = '' OR r.status = $3)
		 ORDER BY c.full_name LIMIT $4 OFFSET $5`, id, cid, status, limit, offset)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Recipient, error) {
		var x Recipient
		err := row.Scan(&x.ContactID, &x.Name, &x.Phone, &x.Status, &x.SkipReason, &x.Error, &x.SentAt)
		return x, err
	})
	if out == nil {
		out = []Recipient{}
	}
	return out, err
}

// ── Inbox settings ────────────────────────────────────────────────────────────

// InboxSettings are the broadcast rate and the business-hours auto-reply.
type InboxSettings struct {
	BroadcastPerMinute int                `json:"broadcast_per_minute"`
	AutoReplyEnabled   bool               `json:"auto_reply_enabled"`
	AutoReplyEN        string             `json:"auto_reply_en"`
	AutoReplyAR        string             `json:"auto_reply_ar"`
	Hours              *bizhours.Schedule `json:"hours"` // nil: the default for the country
}

// InboxSettings returns the company's inbox settings.
func (r *BroadcastRepo) InboxSettings(ctx context.Context) (*InboxSettings, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	s := &InboxSettings{BroadcastPerMinute: 30}
	var hours []byte
	err = r.db.QueryRow(ctx, `SELECT wa_broadcast_per_minute, wa_auto_reply_enabled, wa_auto_reply_en, wa_auto_reply_ar, wa_business_hours
		FROM company_settings WHERE company_id = $1`, cid).Scan(&s.BroadcastPerMinute, &s.AutoReplyEnabled, &s.AutoReplyEN, &s.AutoReplyAR, &hours)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(hours) > 0 {
		var sch bizhours.Schedule
		if json.Unmarshal(hours, &sch) == nil {
			s.Hours = &sch
		}
	}
	return s, nil
}

// UpdateInboxSettings saves the company's inbox settings.
func (r *BroadcastRepo) UpdateInboxSettings(ctx context.Context, s *InboxSettings) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	var hours []byte
	if s.Hours != nil {
		hours, _ = json.Marshal(s.Hours)
	}
	tag, err := r.db.Exec(ctx, `UPDATE company_settings SET wa_broadcast_per_minute = $1, wa_auto_reply_enabled = $2, wa_auto_reply_en = $3,
		wa_auto_reply_ar = $4, wa_business_hours = $5 WHERE company_id = $6`,
		s.BroadcastPerMinute, s.AutoReplyEnabled, s.AutoReplyEN, s.AutoReplyAR, hours, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ── Threads ───────────────────────────────────────────────────────────────────

// ClaimAutoReply reports whether the auto-reply may be sent on a thread now: at most once
// per `every`.
func (r *BroadcastRepo) ClaimAutoReply(ctx context.Context, threadID uuid.UUID, every time.Duration) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	tag, err := r.db.Exec(ctx, `UPDATE whatsapp_threads SET auto_reply_at = NOW()
		WHERE id = $1 AND company_id = $2 AND (auto_reply_at IS NULL OR auto_reply_at < NOW() - make_interval(secs => $3))`,
		threadID, cid, every.Seconds())
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// AssignThread gives a thread to one of the company's users (nil unassigns).
func (r *BroadcastRepo) AssignThread(ctx context.Context, threadID uuid.UUID, userID *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if userID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND company_id = $2 AND is_active)`, *userID, cid).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return ErrForeignReference
		}
	}
	tag, err := r.db.Exec(ctx, `UPDATE whatsapp_threads SET assigned_to = $1 WHERE id = $2 AND company_id = $3`, userID, threadID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
