package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type WhatsAppRepo struct {
	db *pgxpool.Pool
}

func NewWhatsAppRepo(db *pgxpool.Pool) *WhatsAppRepo {
	return &WhatsAppRepo{db: db}
}

// UpsertThread finds or creates a thread for a contact + account pair.
func (r *WhatsAppRepo) UpsertThread(ctx context.Context, contactID uuid.UUID, waAccountID string) (*domain.WhatsAppThread, error) {
	const q = `
		INSERT INTO whatsapp_threads (id, contact_id, wa_account_id)
		VALUES (uuid_generate_v4(), $1, $2)
		ON CONFLICT (contact_id, wa_account_id) DO UPDATE
			SET thread_status = CASE
				WHEN whatsapp_threads.thread_status = 'closed' THEN 'open'
				ELSE whatsapp_threads.thread_status
			END
		RETURNING id, contact_id, wa_account_id, thread_status, last_message_at, message_count, ai_summary, created_at
	`
	t := &domain.WhatsAppThread{}
	err := r.db.QueryRow(ctx, q, contactID, waAccountID).Scan(
		&t.ID, &t.ContactID, &t.WAAccountID, &t.ThreadStatus,
		&t.LastMessageAt, &t.MessageCount, &t.AISummary, &t.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert thread: %w", err)
	}
	return t, nil
}

// SaveMessage persists an inbound or outbound message.
func (r *WhatsAppRepo) SaveMessage(ctx context.Context, msg *domain.WhatsAppMessage) error {
	const q = `
		INSERT INTO whatsapp_messages (id, thread_id, direction, body, media_url, wa_message_id, sent_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, NOW())
		ON CONFLICT (wa_message_id) DO NOTHING
		RETURNING id, sent_at
	`
	return r.db.QueryRow(ctx, q,
		msg.ThreadID, msg.Direction, msg.Body, msg.MediaURL, msg.WAMessageID,
	).Scan(&msg.ID, &msg.SentAt)
}

// UpdateThreadMeta bumps last_message_at and message_count after saving a message.
func (r *WhatsAppRepo) UpdateThreadMeta(ctx context.Context, threadID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE whatsapp_threads
		SET last_message_at = NOW(), message_count = message_count + 1
		WHERE id = $1
	`, threadID)
	return err
}

func (r *WhatsAppRepo) ListThreads(ctx context.Context, status string, contactID *uuid.UUID, page, limit int) ([]domain.WhatsAppThread, error) {
	offset := (page - 1) * limit
	args := []any{status, limit, offset}
	contactFilter := ""
	if contactID != nil {
		contactFilter = "AND t.contact_id = $4"
		args = append(args, *contactID)
	}
	q := fmt.Sprintf(`
		SELECT t.id, t.contact_id, t.wa_account_id, t.thread_status,
		       t.last_message_at, t.message_count, t.ai_summary, t.created_at,
		       c.id, c.phone_wa, c.full_name, c.language
		FROM whatsapp_threads t
		JOIN contacts c ON c.id = t.contact_id
		WHERE ($1 = '' OR t.thread_status = $1) %s
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
	const q = `
		SELECT id, thread_id, direction, body, media_url, wa_message_id, sent_at
		FROM whatsapp_messages
		WHERE thread_id = $1
		ORDER BY sent_at ASC
		LIMIT $2
	`
	rows, err := r.db.Query(ctx, q, threadID, limit)
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
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (r *WhatsAppRepo) CloseThread(ctx context.Context, threadID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE whatsapp_threads SET thread_status='closed' WHERE id=$1`,
		threadID,
	)
	return err
}

func (r *WhatsAppRepo) UpdateAISummary(ctx context.Context, threadID uuid.UUID, summary string) error {
	_, err := r.db.Exec(ctx, `UPDATE whatsapp_threads SET ai_summary = $2 WHERE id = $1`, threadID, summary)
	return err
}

func (r *WhatsAppRepo) ReopenThread(ctx context.Context, threadID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE whatsapp_threads SET thread_status = 'open' WHERE id = $1`, threadID)
	return err
}

func (r *WhatsAppRepo) GetThread(ctx context.Context, threadID uuid.UUID) (*domain.WhatsAppThread, error) {
	const q = `
		SELECT t.id, t.contact_id, t.wa_account_id, t.thread_status,
		       t.last_message_at, t.message_count, t.ai_summary, t.created_at,
		       c.id, c.phone_wa, c.full_name, c.language
		FROM whatsapp_threads t
		JOIN contacts c ON c.id = t.contact_id
		WHERE t.id = $1
	`
	t := &domain.WhatsAppThread{}
	var c domain.Contact
	err := r.db.QueryRow(ctx, q, threadID).Scan(
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
