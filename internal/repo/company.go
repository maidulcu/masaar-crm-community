package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type CompanyRepo struct {
	db *pgxpool.Pool
}

func NewCompanyRepo(db *pgxpool.Pool) *CompanyRepo {
	return &CompanyRepo{db: db}
}

func (r *CompanyRepo) Create(ctx context.Context, name, subdomain string, trialDurationDays int) (*domain.Company, error) {
	now := time.Now()
	trialEnd := now.AddDate(0, 0, trialDurationDays)
	c := &domain.Company{
		ID:             uuid.New(),
		Name:           name,
		Subdomain:      subdomain,
		Plan:           "starter",
		TrialStartedAt: &now,
		TrialEndsAt:    &trialEnd,
		OnTrial:        true,
		IsActive:       true,
		IsDemo:         false,
		CreatedAt:      now,
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("create company tx begin: %w", err)
	}
	defer tx.Rollback(ctx)

	const q = `INSERT INTO companies (id, name, subdomain, plan, trial_started_at, trial_ends_at, on_trial, is_active, is_demo, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = tx.Exec(ctx, q,
		c.ID, c.Name, c.Subdomain, c.Plan, c.TrialStartedAt, c.TrialEndsAt, c.OnTrial, c.IsActive, c.IsDemo, c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert company: %w", err)
	}

	// Create default company_settings row
	const csq = `INSERT INTO company_settings (company_id, name, vat_number, business_address)
		VALUES ($1, $2, '', '')`
	_, err = tx.Exec(ctx, csq, c.ID, c.Name)
	if err != nil {
		return nil, fmt.Errorf("insert company_settings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("create company tx commit: %w", err)
	}
	return c, nil
}

// scanCompany reads a full company row including is_demo.
// Column order: id, name, subdomain, plan, trial_started_at, trial_ends_at,
//
//	on_trial, is_active, is_demo, stripe_customer_id, stripe_sub_id, created_at
func scanCompany(row interface {
	Scan(dest ...any) error
}, c *domain.Company) error {
	return row.Scan(
		&c.ID, &c.Name, &c.Subdomain, &c.Plan,
		&c.TrialStartedAt, &c.TrialEndsAt, &c.OnTrial, &c.IsActive, &c.IsDemo,
		&c.StripeCustomerID, &c.StripeSubID, &c.CreatedAt,
	)
}

const selectCompanyCols = `SELECT id, name, COALESCE(subdomain,''), COALESCE(plan,'community'),
	trial_started_at, trial_ends_at, on_trial, is_active, is_demo,
	COALESCE(stripe_customer_id,''), COALESCE(stripe_sub_id,''), created_at`

func (r *CompanyRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	q := selectCompanyCols + ` FROM companies WHERE id = $1`
	row := r.db.QueryRow(ctx, q, id)
	var c domain.Company
	if err := scanCompany(row, &c); err != nil {
		return nil, fmt.Errorf("get company by id: %w", err)
	}
	return &c, nil
}

func (r *CompanyRepo) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Company, error) {
	q := selectCompanyCols + ` FROM companies WHERE subdomain = $1`
	row := r.db.QueryRow(ctx, q, subdomain)
	var c domain.Company
	if err := scanCompany(row, &c); err != nil {
		return nil, fmt.Errorf("get company by subdomain: %w", err)
	}
	return &c, nil
}

func (r *CompanyRepo) SetPlan(ctx context.Context, id uuid.UUID, plan, stripeCustomerID, stripeSubID string) error {
	const q = `UPDATE companies SET plan = $2,
		stripe_customer_id = $3, stripe_sub_id = $4
		WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id, plan, stripeCustomerID, stripeSubID)
	if err != nil {
		return fmt.Errorf("set plan: %w", err)
	}
	return nil
}

func (r *CompanyRepo) EndTrial(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE companies SET on_trial = FALSE, plan = 'community' WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("end trial: %w", err)
	}
	return nil
}

func (r *CompanyRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE companies SET is_active = FALSE WHERE id = $1`
	_, err := r.db.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("deactivate company: %w", err)
	}
	return nil
}

func (r *CompanyRepo) ListExpiredTrials(ctx context.Context) ([]domain.Company, error) {
	q := selectCompanyCols + ` FROM companies WHERE on_trial = TRUE AND trial_ends_at < NOW()`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list expired trials: %w", err)
	}
	defer rows.Close()

	var companies []domain.Company
	for rows.Next() {
		var c domain.Company
		if err := scanCompany(rows, &c); err != nil {
			return nil, fmt.Errorf("scan expired trial: %w", err)
		}
		companies = append(companies, c)
	}
	return companies, nil
}

func (r *CompanyRepo) List(ctx context.Context) ([]domain.Company, error) {
	q := selectCompanyCols + ` FROM companies ORDER BY created_at ASC`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	var companies []domain.Company
	for rows.Next() {
		var c domain.Company
		if err := scanCompany(rows, &c); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		companies = append(companies, c)
	}
	return companies, nil
}
