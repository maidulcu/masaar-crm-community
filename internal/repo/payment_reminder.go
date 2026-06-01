package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type PaymentReminderRepo struct {
	db *pgxpool.Pool
}

func NewPaymentReminderRepo(db *pgxpool.Pool) *PaymentReminderRepo {
	return &PaymentReminderRepo{db: db}
}

func (r *PaymentReminderRepo) Create(ctx context.Context, reminder *domain.PaymentReminder) error {
	const q = `
		INSERT INTO payment_reminders (
			id, company_id, payment_id, reminder_type, reminder_date, delivery_method, delivery_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	reminder.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		reminder.ID, reminder.CompanyID, reminder.PaymentID, reminder.ReminderType, reminder.ReminderDate,
		reminder.DeliveryMethod, reminder.DeliveryStatus,
	).Scan(&reminder.CreatedAt, &reminder.UpdatedAt)
}

func (r *PaymentReminderRepo) GetPending(ctx context.Context, companyID uuid.UUID) ([]domain.PaymentReminder, error) {
	const q = `
		SELECT id, company_id, payment_id, reminder_type, reminder_date, sent_at,
		       delivery_method, delivery_status, delivery_error, created_at, updated_at
		FROM payment_reminders
		WHERE company_id = $1 AND delivery_status = 'pending' AND reminder_date <= CURRENT_DATE
		ORDER BY reminder_date ASC
	`
	rows, err := r.db.Query(ctx, q, companyID)
	if err != nil {
		return nil, fmt.Errorf("get pending reminders: %w", err)
	}
	defer rows.Close()

	var reminders []domain.PaymentReminder
	for rows.Next() {
		var pr domain.PaymentReminder
		if err := rows.Scan(
			&pr.ID, &pr.CompanyID, &pr.PaymentID, &pr.ReminderType, &pr.ReminderDate, &pr.SentAt,
			&pr.DeliveryMethod, &pr.DeliveryStatus, &pr.DeliveryError, &pr.CreatedAt, &pr.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan reminder: %w", err)
		}
		reminders = append(reminders, pr)
	}
	return reminders, nil
}

func (r *PaymentReminderRepo) MarkSent(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE payment_reminders
		SET delivery_status = 'sent', sent_at = NOW(), updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`
	var updated time.Time
	return r.db.QueryRow(ctx, q, id).Scan(&updated)
}

func (r *PaymentReminderRepo) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	const q = `
		UPDATE payment_reminders
		SET delivery_status = 'failed', delivery_error = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING updated_at
	`
	var updated time.Time
	return r.db.QueryRow(ctx, q, errMsg, id).Scan(&updated)
}

func (r *PaymentReminderRepo) GetByPaymentAndType(ctx context.Context, paymentID uuid.UUID, reminderType domain.ReminderType, deliveryMethod domain.DeliveryMethod) (*domain.PaymentReminder, error) {
	const q = `
		SELECT id, company_id, payment_id, reminder_type, reminder_date, sent_at,
		       delivery_method, delivery_status, delivery_error, created_at, updated_at
		FROM payment_reminders
		WHERE payment_id = $1 AND reminder_type = $2 AND delivery_method = $3
	`
	pr := &domain.PaymentReminder{}
	err := r.db.QueryRow(ctx, q, paymentID, reminderType, deliveryMethod).Scan(
		&pr.ID, &pr.CompanyID, &pr.PaymentID, &pr.ReminderType, &pr.ReminderDate, &pr.SentAt,
		&pr.DeliveryMethod, &pr.DeliveryStatus, &pr.DeliveryError, &pr.CreatedAt, &pr.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get reminder: %w", err)
	}
	return pr, nil
}
