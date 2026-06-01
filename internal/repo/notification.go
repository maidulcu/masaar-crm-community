package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type NotificationRepo struct {
	db *pgxpool.Pool
}

func NewNotificationRepo(db *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{db: db}
}

func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, type, title, body, read, data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		n.ID, n.UserID, n.Type, n.Title, n.Body, n.Read, n.Data, n.CreatedAt,
	)
	return err
}

func (r *NotificationRepo) ListByUser(ctx context.Context, userID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.Notification], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count notifications: %w", err)
	}

	query := `
		SELECT id, user_id, type, title, body, read, data, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Read, &n.Data, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return &domain.PaginatedResult[domain.Notification]{
		Data:  notifications,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	query := `UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, id, userID)
	return err
}

func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE notifications SET read = true WHERE user_id = $1 AND read = false`, userID)
	return err
}

func (r *NotificationRepo) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = false`
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}
