package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type WhatsAppRepo struct {
	db *pgxpool.Pool
}

func NewWhatsAppRepo(db *pgxpool.Pool) *WhatsAppRepo {
	return &WhatsAppRepo{db: db}
}

// UpsertThread finds or creates a thread for a contact + account pair.
func (r *WhatsAppRepo) UpsertThread(ctx context.Context, contactID uuid.UUID, waAccountID string) (*domain.WhatsAppThread, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	// The contact must belong to the caller's company.
	const q = `
		INSERT INTO whatsapp_threads (id, company_id, contact_id, wa_account_id)
		SELECT uuid_generate_v4(), $3, $1, $2
		WHERE EXISTS (SELECT 1 FROM contacts WHERE id = $1 AND company_id = $3)
		ON CONFLICT (contact_id, wa_account_id) DO UPDATE
			SET thread_status = CASE
				WHEN whatsapp_threads.thread_status = 'closed' THEN 'open'
				ELSE whatsapp_threads.thread_status
			END
		RETURNING id, contact_id, wa_account_id, thread_status, last_message_at, message_count, COALESCE(ai_summary,''), created_at
	`
	t := &domain.WhatsAppThread{}
	err = r.db.QueryRow(ctx, q, contactID, waAccountID, cid).Scan(
		&t.ID, &t.ContactID, &t.WAAccountID, &t.ThreadStatus,
		&t.LastMessageAt, &t.MessageCount, &t.AISummary, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert thread: %w", err)
	}
	return t, nil
}

// SaveMessage persists an inbound or outbound message and reports whether it was newly stored.
//
// Saving is idempotent on the Meta message id: Meta redelivers webhooks until it gets a 2xx, so
// seeing the same id again is normal and returns created=false (not an error).
func (r *WhatsAppRepo) SaveMessage(ctx context.Context, msg *domain.WhatsAppMessage) (created bool, err error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	// The thread must belong to the caller's company.
	const q = `
		INSERT INTO whatsapp_messages (id, thread_id, direction, body, media_url, wa_message_id, wa_media_id, media_mime, sent_at)
		SELECT uuid_generate_v4(), $1, $2, $3, $4, NULLIF($5, ''), NULLIF($7, ''), NULLIF($8, ''), NOW()
		WHERE EXISTS (SELECT 1 FROM whatsapp_threads WHERE id = $1 AND company_id = $6)
		ON CONFLICT (wa_message_id) DO NOTHING
		RETURNING id, sent_at
	`
	err = r.db.QueryRow(ctx, q,
		msg.ThreadID, msg.Direction, msg.Body, msg.MediaURL, msg.WAMessageID, cid, msg.MediaID, msg.MediaMime,
	).Scan(&msg.ID, &msg.SentAt)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	// No row: either this message id is already stored (a redelivery) or the thread is not ours.
	var exists bool
	if qerr := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM whatsapp_messages m JOIN whatsapp_threads t ON t.id = m.thread_id
			WHERE m.wa_message_id = $1 AND t.company_id = $2)`, msg.WAMessageID, cid).Scan(&exists); qerr != nil {
		return false, qerr
	}
	if exists {
		return false, nil
	}
	return false, ErrForeignReference
}

// MessageExists reports whether a message with this Meta message id is already stored.
func (r *WhatsAppRepo) MessageExists(ctx context.Context, waMessageID string) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM whatsapp_messages m JOIN whatsapp_threads t ON t.id = m.thread_id
			WHERE m.wa_message_id = $1 AND t.company_id = $2)`, waMessageID, cid).Scan(&exists)
	return exists, err
}

// UpdateThreadMeta bumps last_message_at and message_count after saving a message.
func (r *WhatsAppRepo) UpdateThreadMeta(ctx context.Context, threadID uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE whatsapp_threads
		SET last_message_at = NOW(), message_count = message_count + 1
		WHERE id = $1 AND company_id = $2
	`, threadID, cid)
	return err
}

func (r *WhatsAppRepo) ListThreads(ctx context.Context, status string, contactID *uuid.UUID, page, limit int) ([]domain.WhatsAppThread, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	offset := (page - 1) * limit
	args := []any{status, limit, offset, cid}
	contactFilter := ""
	if contactID != nil {
		contactFilter = "AND t.contact_id = $5"
		args = append(args, *contactID)
	}
	q := fmt.Sprintf(`
		SELECT t.id, t.contact_id, t.wa_account_id, t.thread_status,
		       t.last_message_at, t.message_count, COALESCE(t.ai_summary,''), t.created_at,
		       c.id, c.phone_wa, c.full_name, c.language
		FROM whatsapp_threads t
		JOIN contacts c ON c.id = t.contact_id AND c.company_id = t.company_id
		WHERE t.company_id = $4 AND ($1 = '' OR t.thread_status = $1) %s
		ORDER BY t.last_message_at DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`, contactFilter)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list threads: %w", err)
	}
	defer rows.Close()

	var threads []domain.WhatsAppThread
	for rows.Next() {
		var t domain.WhatsAppThread
		var c domain.Contact
		if err := rows.Scan(
			&t.ID, &t.ContactID, &t.WAAccountID, &t.ThreadStatus,
			&t.LastMessageAt, &t.MessageCount, &t.AISummary, &t.CreatedAt,
			&c.ID, &c.PhoneWA, &c.FullName, &c.Language,
		); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		t.Contact = &c
		threads = append(threads, t)
	}
	return threads, nil
}

func (r *WhatsAppRepo) GetMessages(ctx context.Context, threadID uuid.UUID, limit int) ([]domain.WhatsAppMessage, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	// Outbound rows carry the delivery state recorded from Meta's status webhooks.
	const q = `
		SELECT m.id, m.thread_id, m.direction, m.body, COALESCE(m.media_url,''), COALESCE(m.wa_message_id,''), m.sent_at,
		       COALESCE(m.wa_media_id,''), COALESCE(m.media_mime,''),
		       COALESCE(o.status,''), COALESCE(o.error_message,'')
		FROM whatsapp_messages m
		JOIN whatsapp_threads t ON t.id = m.thread_id
		LEFT JOIN whatsapp_outbound o ON o.wa_message_id = m.wa_message_id AND o.company_id = t.company_id
		WHERE m.thread_id = $1 AND t.company_id = $3
		ORDER BY m.sent_at ASC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, q, threadID, limit, cid)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	defer rows.Close()

	var msgs []domain.WhatsAppMessage
	for rows.Next() {
		var m domain.WhatsAppMessage
		if err := rows.Scan(
			&m.ID, &m.ThreadID, &m.Direction, &m.Body,
			&m.MediaURL, &m.WAMessageID, &m.SentAt,
			&m.MediaID, &m.MediaMime, &m.Status, &m.ErrorMsg,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *WhatsAppRepo) CloseThread(ctx context.Context, threadID uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE whatsapp_threads SET thread_status='closed' WHERE id=$1 AND company_id=$2`,
		threadID, cid,
	)
	return err
}

func (r *WhatsAppRepo) UpdateAISummary(ctx context.Context, threadID uuid.UUID, summary string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `UPDATE whatsapp_threads SET ai_summary = $2 WHERE id = $1 AND company_id = $3`, threadID, summary, cid)
	return err
}

func (r *WhatsAppRepo) ReopenThread(ctx context.Context, threadID uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `UPDATE whatsapp_threads SET thread_status = 'open' WHERE id = $1 AND company_id = $2`, threadID, cid)
	return err
}

func (r *WhatsAppRepo) GetThread(ctx context.Context, threadID uuid.UUID) (*domain.WhatsAppThread, error) {
	const q = `
		SELECT t.id, t.contact_id, t.wa_account_id, t.thread_status,
		       t.last_message_at, t.message_count, COALESCE(t.ai_summary,''), t.created_at,
		       c.id, c.phone_wa, c.full_name, c.language
		FROM whatsapp_threads t
		JOIN contacts c ON c.id = t.contact_id AND c.company_id = t.company_id
		WHERE t.id = $1 AND t.company_id = $2
	`
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	t := &domain.WhatsAppThread{}
	var c domain.Contact
	err = r.db.QueryRow(ctx, q, threadID, cid).Scan(
		&t.ID, &t.ContactID, &t.WAAccountID, &t.ThreadStatus,
		&t.LastMessageAt, &t.MessageCount, &t.AISummary, &t.CreatedAt,
		&c.ID, &c.PhoneWA, &c.FullName, &c.Language,
	)
	if err != nil {
		return nil, fmt.Errorf("get thread: %w", err)
	}
	t.Contact = &c
	return t, nil
}
