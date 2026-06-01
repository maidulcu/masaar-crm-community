package repo

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type EmailRepository struct {
	pool *pgxpool.Pool
}

func NewEmailRepository(pool *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{pool: pool}
}

func (r *EmailRepository) Create(ctx context.Context, e *domain.EmailHistory) error {
	var metadataJSON []byte
	if e.Metadata != nil {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return err
		}
		metadataJSON = b
	}

	query := `INSERT INTO email_history
	(from_email, to_email, subject, body, html_body, status, error_message, related_to, related_id, created_by, metadata)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, query,
		e.FromEmail,
		e.ToEmail,
		e.Subject,
		e.Body,
		e.HTMLBody,
		e.Status,
		e.ErrorMsg,
		e.RelatedTo,
		e.RelatedID,
		e.CreatedBy,
		metadataJSON,
	).Scan(&e.ID, &e.CreatedAt)

	return err
}

func (r *EmailRepository) UpdateStatus(ctx context.Context, id int64, status domain.EmailStatus, errorMsg string) error {
	query := `UPDATE email_history SET status = $1, error_message = $2`
	args := []any{status, errorMsg}

	if status == domain.EmailSent {
		query += `, sent_at = $3`
		args = append(args, time.Now())
	}

	query += ` WHERE id = $` + strconv.Itoa(len(args)+1)
	args = append(args, id)

	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *EmailRepository) GetByID(ctx context.Context, id int64) (*domain.EmailHistory, error) {
	query := `SELECT id, from_email, to_email, subject, body, html_body, status, error_message,
	related_to, related_id, sent_at, created_at, created_by, metadata
	FROM email_history WHERE id = $1`

	var e domain.EmailHistory
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&e.ID,
		&e.FromEmail,
		&e.ToEmail,
		&e.Subject,
		&e.Body,
		&e.HTMLBody,
		&e.Status,
		&e.ErrorMsg,
		&e.RelatedTo,
		&e.RelatedID,
		&e.SentAt,
		&e.CreatedAt,
		&e.CreatedBy,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &e.Metadata)
	}

	return &e, nil
}

func (r *EmailRepository) ListByRelated(ctx context.Context, relatedTo string, relatedID int64) ([]domain.EmailHistory, error) {
	query := `SELECT id, from_email, to_email, subject, body, html_body, status, error_message,
	related_to, related_id, sent_at, created_at, created_by, metadata
	FROM email_history WHERE related_to = $1 AND related_id = $2 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, relatedTo, relatedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []domain.EmailHistory
	for rows.Next() {
		var e domain.EmailHistory
		var metadataJSON []byte

		err := rows.Scan(
			&e.ID,
			&e.FromEmail,
			&e.ToEmail,
			&e.Subject,
			&e.Body,
			&e.HTMLBody,
			&e.Status,
			&e.ErrorMsg,
			&e.RelatedTo,
			&e.RelatedID,
			&e.SentAt,
			&e.CreatedAt,
			&e.CreatedBy,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}

		emails = append(emails, e)
	}

	return emails, rows.Err()
}

func (r *EmailRepository) ListAll(ctx context.Context, page, limit int) ([]domain.EmailHistory, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM email_history`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, from_email, to_email, subject, body, html_body, status, error_message,
	related_to, related_id, sent_at, created_at, created_by, metadata
	FROM email_history ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var emails []domain.EmailHistory
	for rows.Next() {
		var e domain.EmailHistory
		var metadataJSON []byte

		err := rows.Scan(
			&e.ID,
			&e.FromEmail,
			&e.ToEmail,
			&e.Subject,
			&e.Body,
			&e.HTMLBody,
			&e.Status,
			&e.ErrorMsg,
			&e.RelatedTo,
			&e.RelatedID,
			&e.SentAt,
			&e.CreatedAt,
			&e.CreatedBy,
			&metadataJSON,
		)
		if err != nil {
			return nil, 0, err
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}

		emails = append(emails, e)
	}

	return emails, total, rows.Err()
}

func (r *EmailRepository) ListByStatus(ctx context.Context, status domain.EmailStatus, limit int) ([]domain.EmailHistory, error) {
	query := `SELECT id, from_email, to_email, subject, body, html_body, status, error_message,
	related_to, related_id, sent_at, created_at, created_by, metadata
	FROM email_history WHERE status = $1 ORDER BY created_at DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []domain.EmailHistory
	for rows.Next() {
		var e domain.EmailHistory
		var metadataJSON []byte

		err := rows.Scan(
			&e.ID,
			&e.FromEmail,
			&e.ToEmail,
			&e.Subject,
			&e.Body,
			&e.HTMLBody,
			&e.Status,
			&e.ErrorMsg,
			&e.RelatedTo,
			&e.RelatedID,
			&e.SentAt,
			&e.CreatedAt,
			&e.CreatedBy,
			&metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}

		emails = append(emails, e)
	}

	return emails, rows.Err()
}
