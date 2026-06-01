package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type LeaseTemplateRepo struct {
	db *pgxpool.Pool
}

func NewLeaseTemplateRepo(db *pgxpool.Pool) *LeaseTemplateRepo {
	return &LeaseTemplateRepo{db: db}
}

func (r *LeaseTemplateRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.LeaseTemplate], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM lease_templates WHERE company_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count templates: %w", err)
	}

	const q = `
		SELECT id, company_id, name, description, is_default, payment_frequency, payment_day_of_month,
		       auto_generate_payments, default_security_deposit_percent, default_utility_charges,
		       default_late_fee_percent, default_lease_duration_months, default_notice_period_days,
		       default_renewal_duration_months, template_document_url, terms_conditions, status,
		       created_at, updated_at, created_by, updated_by
		FROM lease_templates
		WHERE company_id = $1
		ORDER BY is_default DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []domain.LeaseTemplate
	for rows.Next() {
		var t domain.LeaseTemplate
		if err := rows.Scan(
			&t.ID, &t.CompanyID, &t.Name, &t.Description, &t.IsDefault, &t.PaymentFrequency, &t.PaymentDayOfMonth,
			&t.AutoGeneratePayments, &t.DefaultSecurityDepositPct, &t.DefaultUtilityCharges,
			&t.DefaultLateFeePercent, &t.DefaultLeaseDurationMonths, &t.DefaultNoticePeriodDays,
			&t.DefaultRenewalDurationMonths, &t.TemplateDocumentURL, &t.TermsConditions, &t.Status,
			&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}

	return &domain.PaginatedResult[domain.LeaseTemplate]{
		Data:  templates,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *LeaseTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.LeaseTemplate, error) {
	const q = `
		SELECT id, company_id, name, description, is_default, payment_frequency, payment_day_of_month,
		       auto_generate_payments, default_security_deposit_percent, default_utility_charges,
		       default_late_fee_percent, default_lease_duration_months, default_notice_period_days,
		       default_renewal_duration_months, template_document_url, terms_conditions, status,
		       created_at, updated_at, created_by, updated_by
		FROM lease_templates WHERE id = $1
	`
	t := &domain.LeaseTemplate{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.CompanyID, &t.Name, &t.Description, &t.IsDefault, &t.PaymentFrequency, &t.PaymentDayOfMonth,
		&t.AutoGeneratePayments, &t.DefaultSecurityDepositPct, &t.DefaultUtilityCharges,
		&t.DefaultLateFeePercent, &t.DefaultLeaseDurationMonths, &t.DefaultNoticePeriodDays,
		&t.DefaultRenewalDurationMonths, &t.TemplateDocumentURL, &t.TermsConditions, &t.Status,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get template by id: %w", err)
	}
	return t, nil
}

func (r *LeaseTemplateRepo) Create(ctx context.Context, t *domain.LeaseTemplate) error {
	const q = `
		INSERT INTO lease_templates (
			id, company_id, name, description, is_default, payment_frequency, payment_day_of_month,
			auto_generate_payments, default_security_deposit_percent, default_utility_charges,
			default_late_fee_percent, default_lease_duration_months, default_notice_period_days,
			default_renewal_duration_months, template_document_url, terms_conditions, status,
			created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)
		RETURNING created_at, updated_at
	`
	t.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		t.ID, t.CompanyID, t.Name, t.Description, t.IsDefault, t.PaymentFrequency, t.PaymentDayOfMonth,
		t.AutoGeneratePayments, t.DefaultSecurityDepositPct, t.DefaultUtilityCharges,
		t.DefaultLateFeePercent, t.DefaultLeaseDurationMonths, t.DefaultNoticePeriodDays,
		t.DefaultRenewalDurationMonths, t.TemplateDocumentURL, t.TermsConditions, t.Status,
		t.CreatedBy, t.UpdatedBy,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
}

func (r *LeaseTemplateRepo) Update(ctx context.Context, t *domain.LeaseTemplate) error {
	const q = `
		UPDATE lease_templates
		SET name=$1, description=$2, is_default=$3, payment_frequency=$4, payment_day_of_month=$5,
		    auto_generate_payments=$6, default_security_deposit_percent=$7, default_utility_charges=$8,
		    default_late_fee_percent=$9, default_lease_duration_months=$10, default_notice_period_days=$11,
		    default_renewal_duration_months=$12, template_document_url=$13, terms_conditions=$14,
		    status=$15, updated_by=$16, updated_at=NOW()
		WHERE id=$17
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, q,
		t.Name, t.Description, t.IsDefault, t.PaymentFrequency, t.PaymentDayOfMonth,
		t.AutoGeneratePayments, t.DefaultSecurityDepositPct, t.DefaultUtilityCharges,
		t.DefaultLateFeePercent, t.DefaultLeaseDurationMonths, t.DefaultNoticePeriodDays,
		t.DefaultRenewalDurationMonths, t.TemplateDocumentURL, t.TermsConditions,
		t.Status, t.UpdatedBy, t.ID,
	).Scan(&t.UpdatedAt)
}

func (r *LeaseTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM lease_templates WHERE id=$1`, id)
	return err
}

func (r *LeaseTemplateRepo) GetDefault(ctx context.Context, companyID uuid.UUID) (*domain.LeaseTemplate, error) {
	const q = `
		SELECT id, company_id, name, description, is_default, payment_frequency, payment_day_of_month,
		       auto_generate_payments, default_security_deposit_percent, default_utility_charges,
		       default_late_fee_percent, default_lease_duration_months, default_notice_period_days,
		       default_renewal_duration_months, template_document_url, terms_conditions, status,
		       created_at, updated_at, created_by, updated_by
		FROM lease_templates
		WHERE company_id = $1 AND is_default = TRUE
		LIMIT 1
	`
	t := &domain.LeaseTemplate{}
	err := r.db.QueryRow(ctx, q, companyID).Scan(
		&t.ID, &t.CompanyID, &t.Name, &t.Description, &t.IsDefault, &t.PaymentFrequency, &t.PaymentDayOfMonth,
		&t.AutoGeneratePayments, &t.DefaultSecurityDepositPct, &t.DefaultUtilityCharges,
		&t.DefaultLateFeePercent, &t.DefaultLeaseDurationMonths, &t.DefaultNoticePeriodDays,
		&t.DefaultRenewalDurationMonths, &t.TemplateDocumentURL, &t.TermsConditions, &t.Status,
		&t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get default template: %w", err)
	}
	return t, nil
}
