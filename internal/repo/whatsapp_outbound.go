package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type WhatsAppOutboundRepo struct {
	pool *pgxpool.Pool
}

func NewWhatsAppOutboundRepo(pool *pgxpool.Pool) *WhatsAppOutboundRepo {
	return &WhatsAppOutboundRepo{pool: pool}
}

func (r *WhatsAppOutboundRepo) Create(ctx context.Context, msg *domain.WhatsAppOutbound) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	var metadataJSON []byte
	if msg.Metadata != nil {
		b, err := json.Marshal(msg.Metadata)
		if err != nil {
			return err
		}
		metadataJSON = b
	}

	query := `INSERT INTO whatsapp_outbound
	(thread_id, to_number, message_body, media_url, status, scheduled_at, created_by, metadata, company_id)
	SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
	WHERE EXISTS (SELECT 1 FROM whatsapp_threads WHERE id = $1 AND company_id = $9)
	RETURNING id, created_at`

	err = r.pool.QueryRow(ctx, query,
		msg.ThreadID,
		msg.ToNumber,
		msg.MessageBody,
		msg.MediaURL,
		msg.Status,
		msg.ScheduledAt,
		msg.CreatedBy,
		metadataJSON,
		cid,
	).Scan(&msg.ID, &msg.CreatedAt)

	return err
}

func (r *WhatsAppOutboundRepo) UpdateStatus(ctx context.Context, id int64, status domain.OutboundStatus, waMessageID, errorMsg string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	var query string
	var args []any

	if status == domain.OutboundSent {
		query = `UPDATE whatsapp_outbound SET status = $1, wa_message_id = $2, sent_at = $3, error_message = $4 WHERE id = $5 AND company_id = $6`
		args = []any{status, waMessageID, time.Now(), errorMsg, id, cid}
	} else {
		query = `UPDATE whatsapp_outbound SET status = $1, wa_message_id = $2, error_message = $3 WHERE id = $4 AND company_id = $5`
		args = []any{status, waMessageID, errorMsg, id, cid}
	}

	_, err = r.pool.Exec(ctx, query, args...)
	return err
}

func (r *WhatsAppOutboundRepo) GetByID(ctx context.Context, id int64) (*domain.WhatsAppOutbound, error) {
	query := `SELECT id, thread_id, to_number, message_body, media_url, wa_message_id, status, error_message,
	scheduled_at, sent_at, created_at, created_by, metadata
	FROM whatsapp_outbound WHERE id = $1 AND company_id = $2`

	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	var msg domain.WhatsAppOutbound
	var metadataJSON []byte

	err = r.pool.QueryRow(ctx, query, id, cid).Scan(
		&msg.ID,
		&msg.ThreadID,
		&msg.ToNumber,
		&msg.MessageBody,
		&msg.MediaURL,
		&msg.WAMessageID,
		&msg.Status,
		&msg.ErrorMsg,
		&msg.ScheduledAt,
		&msg.SentAt,
		&msg.CreatedAt,
		&msg.CreatedBy,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &msg.Metadata)
	}

	return &msg, nil
}

func (r *WhatsAppOutboundRepo) ListByThread(ctx context.Context, threadID uuid.UUID) ([]domain.WhatsAppOutbound, error) {
	query := `SELECT id, thread_id, to_number, message_body, media_url, wa_message_id, status, error_message,
	scheduled_at, sent_at, created_at, created_by, metadata
	FROM whatsapp_outbound WHERE thread_id = $1 AND company_id = $2 ORDER BY created_at DESC`

	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, threadID, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.WhatsAppOutbound
	for rows.Next() {
		var msg domain.WhatsAppOutbound
		var metadataJSON []byte

		err := rows.Scan(
			&msg.ID,
			&msg.ThreadID,
			&msg.ToNumber,
			&msg.MessageBody,
			&msg.MediaURL,
			&msg.WAMessageID,
			&msg.Status,
			&msg.ErrorMsg,
			&msg.ScheduledAt,
			&msg.SentAt,
			&msg.CreatedAt,
			&msg.CreatedBy,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &msg.Metadata)
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

func (r *WhatsAppOutboundRepo) ListPending(ctx context.Context, limit int) ([]domain.WhatsAppOutbound, error) {
	query := `SELECT id, thread_id, to_number, message_body, media_url, wa_message_id, status, error_message,
	scheduled_at, sent_at, created_at, created_by, metadata
	FROM whatsapp_outbound WHERE company_id = $2 AND status = 'pending' AND (scheduled_at IS NULL OR scheduled_at <= NOW())
	ORDER BY created_at ASC LIMIT $1`

	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, limit, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.WhatsAppOutbound
	for rows.Next() {
		var msg domain.WhatsAppOutbound
		var metadataJSON []byte

		err := rows.Scan(
			&msg.ID,
			&msg.ThreadID,
			&msg.ToNumber,
			&msg.MessageBody,
			&msg.MediaURL,
			&msg.WAMessageID,
			&msg.Status,
			&msg.ErrorMsg,
			&msg.ScheduledAt,
			&msg.SentAt,
			&msg.CreatedAt,
			&msg.CreatedBy,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &msg.Metadata)
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}
