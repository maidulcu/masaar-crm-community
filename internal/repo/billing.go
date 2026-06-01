package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BillingRepo struct {
	db *pgxpool.Pool
}

func NewBillingRepo(pool *pgxpool.Pool) *BillingRepo {
	return &BillingRepo{db: pool}
}

// CompanyPlan holds the billing state for a company.
type CompanyPlan struct {
	Plan           string
	StripeCustomer string
	StripeSubID    string
	PlanStartedAt  *time.Time
	PlanExpiresAt  *time.Time
}

// GetPlan reads the current plan and Stripe IDs from the companies table.
func (r *BillingRepo) GetPlan(ctx context.Context, companyID uuid.UUID) (*CompanyPlan, error) {
	const q = `
		SELECT COALESCE(plan, 'community'),
		       COALESCE(stripe_customer_id, ''),
		       COALESCE(stripe_sub_id, ''),
		       NULL, NULL
		FROM companies WHERE id = $1`

	p := &CompanyPlan{}
	err := r.db.QueryRow(ctx, q, companyID).Scan(
		&p.Plan,
		&p.StripeCustomer,
		&p.StripeSubID,
		&p.PlanStartedAt,
		&p.PlanExpiresAt,
	)
	return p, err
}

// SetPlan updates plan and Stripe identifiers on the companies table.
func (r *BillingRepo) SetPlan(ctx context.Context, companyID uuid.UUID, plan, customerID, subID string) error {
	const q = `
		UPDATE companies SET
			plan               = $2,
			stripe_customer_id = $3,
			stripe_sub_id      = $4
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, companyID, plan, customerID, subID)
	return err
}

// IncrUsage atomically increments the usage counter for a resource this month.
// Returns the new count after increment.
func (r *BillingRepo) IncrUsage(ctx context.Context, companyID uuid.UUID, resource string) (int, error) {
	period := time.Now().UTC().Format("2006-01")
	const q = `
		INSERT INTO usage_tracking (company_id, resource, period, count, updated_at)
		VALUES ($1, $2, $3, 1, NOW())
		ON CONFLICT (company_id, resource, period)
		DO UPDATE SET count = usage_tracking.count + 1, updated_at = NOW()
		RETURNING count`

	var count int
	err := r.db.QueryRow(ctx, q, companyID, resource, period).Scan(&count)
	return count, err
}

// GetUsage returns current month usage for all tracked resources.
func (r *BillingRepo) GetUsage(ctx context.Context, companyID uuid.UUID) (map[string]int, error) {
	period := time.Now().UTC().Format("2006-01")
	const q = `
		SELECT resource, count
		FROM usage_tracking
		WHERE company_id = $1 AND period = $2`

	rows, err := r.db.Query(ctx, q, companyID, period)
	if err != nil {
		return nil, fmt.Errorf("get usage: %w", err)
	}
	defer rows.Close()

	result := map[string]int{"bos24": 0, "ai": 0, "pdf": 0}
	for rows.Next() {
		var resource string
		var count int
		if err := rows.Scan(&resource, &count); err != nil {
			return nil, err
		}
		result[resource] = count
	}
	return result, rows.Err()
}
