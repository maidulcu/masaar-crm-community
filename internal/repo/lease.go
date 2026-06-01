package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type LeaseRepo struct {
	db *pgxpool.Pool
}

func NewLeaseRepo(db *pgxpool.Pool) *LeaseRepo {
	return &LeaseRepo{db: db}
}

func (r *LeaseRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.Lease], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM leases WHERE company_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count leases: %w", err)
	}

	const q = `
		SELECT id, company_id, property_id, tenant_id, template_id, start_date, end_date,
		       renewal_start_date, renewal_end_date, monthly_rent, currency, security_deposit,
		       utility_charges, late_fee_percent, payment_frequency, payment_day_of_month,
		       auto_generate_payments, last_generated_payment_date, notice_period_days,
		       move_out_date, move_out_inspection_date, lease_document_url,
		       signed_by_landlord_date, signed_by_tenant_date, ejari_number, ejari_url,
		       status, termination_reason, termination_date, notes,
		       created_at, updated_at, created_by, updated_by
		FROM leases
		WHERE company_id = $1
		ORDER BY start_date DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list leases: %w", err)
	}
	defer rows.Close()

	var leases []domain.Lease
	for rows.Next() {
		var l domain.Lease
		if err := rows.Scan(
			&l.ID, &l.CompanyID, &l.PropertyID, &l.TenantID, &l.TemplateID, &l.StartDate, &l.EndDate,
			&l.RenewalStartDate, &l.RenewalEndDate, &l.MonthlyRent, &l.Currency, &l.SecurityDeposit,
			&l.UtilityCharges, &l.LateFeePct, &l.PaymentFrequency, &l.PaymentDayOfMonth,
			&l.AutoGeneratePayments, &l.LastGeneratedPaymentDt, &l.NoticePeriodDays,
			&l.MoveOutDate, &l.MoveOutInspectionDate, &l.LeaseDocumentURL,
			&l.SignedByLandlordDate, &l.SignedByTenantDate, &l.EjariNumber, &l.EjariURL,
			&l.Status, &l.TerminationReason, &l.TerminationDate, &l.Notes,
			&l.CreatedAt, &l.UpdatedAt, &l.CreatedBy, &l.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan lease: %w", err)
		}
		leases = append(leases, l)
	}

	return &domain.PaginatedResult[domain.Lease]{
		Data:  leases,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *LeaseRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Lease, error) {
	const q = `
		SELECT id, company_id, property_id, tenant_id, template_id, start_date, end_date,
		       renewal_start_date, renewal_end_date, monthly_rent, currency, security_deposit,
		       utility_charges, late_fee_percent, payment_frequency, payment_day_of_month,
		       auto_generate_payments, last_generated_payment_date, notice_period_days,
		       move_out_date, move_out_inspection_date, lease_document_url,
		       signed_by_landlord_date, signed_by_tenant_date, ejari_number, ejari_url,
		       status, termination_reason, termination_date, notes,
		       created_at, updated_at, created_by, updated_by
		FROM leases WHERE id = $1
	`
	l := &domain.Lease{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&l.ID, &l.CompanyID, &l.PropertyID, &l.TenantID, &l.TemplateID, &l.StartDate, &l.EndDate,
		&l.RenewalStartDate, &l.RenewalEndDate, &l.MonthlyRent, &l.Currency, &l.SecurityDeposit,
		&l.UtilityCharges, &l.LateFeePct, &l.PaymentFrequency, &l.PaymentDayOfMonth,
		&l.AutoGeneratePayments, &l.LastGeneratedPaymentDt, &l.NoticePeriodDays,
		&l.MoveOutDate, &l.MoveOutInspectionDate, &l.LeaseDocumentURL,
		&l.SignedByLandlordDate, &l.SignedByTenantDate, &l.EjariNumber, &l.EjariURL,
		&l.Status, &l.TerminationReason, &l.TerminationDate, &l.Notes,
		&l.CreatedAt, &l.UpdatedAt, &l.CreatedBy, &l.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get lease by id: %w", err)
	}
	return l, nil
}

func (r *LeaseRepo) Create(ctx context.Context, l *domain.Lease) error {
	const q = `
		INSERT INTO leases (
			id, company_id, property_id, tenant_id, template_id, start_date, end_date,
			renewal_start_date, renewal_end_date, monthly_rent, currency, security_deposit,
			utility_charges, late_fee_percent, payment_frequency, payment_day_of_month,
			auto_generate_payments, notice_period_days, lease_document_url,
			signed_by_landlord_date, signed_by_tenant_date, ejari_number, ejari_url,
			status, notes, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25, $26, $27
		)
		RETURNING created_at, updated_at
	`
	l.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		l.ID, l.CompanyID, l.PropertyID, l.TenantID, l.TemplateID, l.StartDate, l.EndDate,
		l.RenewalStartDate, l.RenewalEndDate, l.MonthlyRent, l.Currency, l.SecurityDeposit,
		l.UtilityCharges, l.LateFeePct, l.PaymentFrequency, l.PaymentDayOfMonth,
		l.AutoGeneratePayments, l.NoticePeriodDays, l.LeaseDocumentURL,
		l.SignedByLandlordDate, l.SignedByTenantDate, l.EjariNumber, l.EjariURL,
		l.Status, l.Notes, l.CreatedBy, l.UpdatedBy,
	).Scan(&l.CreatedAt, &l.UpdatedAt)
}

func (r *LeaseRepo) Update(ctx context.Context, l *domain.Lease) error {
	const q = `
		UPDATE leases
		SET monthly_rent=$1, security_deposit=$2, utility_charges=$3, late_fee_percent=$4,
		    payment_frequency=$5, payment_day_of_month=$6, auto_generate_payments=$7,
		    last_generated_payment_date=$8, notice_period_days=$9, move_out_date=$10,
		    move_out_inspection_date=$11, lease_document_url=$12,
		    signed_by_landlord_date=$13, signed_by_tenant_date=$14, ejari_number=$15, ejari_url=$16,
		    status=$17, termination_reason=$18, termination_date=$19, notes=$20,
		    renewal_start_date=$21, renewal_end_date=$22, updated_by=$23, updated_at=NOW()
		WHERE id=$24
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, q,
		l.MonthlyRent, l.SecurityDeposit, l.UtilityCharges, l.LateFeePct,
		l.PaymentFrequency, l.PaymentDayOfMonth, l.AutoGeneratePayments,
		l.LastGeneratedPaymentDt, l.NoticePeriodDays, l.MoveOutDate,
		l.MoveOutInspectionDate, l.LeaseDocumentURL,
		l.SignedByLandlordDate, l.SignedByTenantDate, l.EjariNumber, l.EjariURL,
		l.Status, l.TerminationReason, l.TerminationDate, l.Notes,
		l.RenewalStartDate, l.RenewalEndDate, l.UpdatedBy, l.ID,
	).Scan(&l.UpdatedAt)
}

func (r *LeaseRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM leases WHERE id=$1`, id)
	return err
}

func (r *LeaseRepo) GetActiveByProperty(ctx context.Context, propertyID uuid.UUID) ([]domain.Lease, error) {
	const q = `
		SELECT id, company_id, property_id, tenant_id, template_id, start_date, end_date,
		       renewal_start_date, renewal_end_date, monthly_rent, currency, security_deposit,
		       utility_charges, late_fee_percent, payment_frequency, payment_day_of_month,
		       auto_generate_payments, last_generated_payment_date, notice_period_days,
		       move_out_date, move_out_inspection_date, lease_document_url,
		       signed_by_landlord_date, signed_by_tenant_date, ejari_number, ejari_url,
		       status, termination_reason, termination_date, notes,
		       created_at, updated_at, created_by, updated_by
		FROM leases
		WHERE property_id = $1 AND status = 'active'
		ORDER BY start_date DESC
	`
	rows, err := r.db.Query(ctx, q, propertyID)
	if err != nil {
		return nil, fmt.Errorf("get active leases by property: %w", err)
	}
	defer rows.Close()

	var leases []domain.Lease
	for rows.Next() {
		var l domain.Lease
		if err := rows.Scan(
			&l.ID, &l.CompanyID, &l.PropertyID, &l.TenantID, &l.TemplateID, &l.StartDate, &l.EndDate,
			&l.RenewalStartDate, &l.RenewalEndDate, &l.MonthlyRent, &l.Currency, &l.SecurityDeposit,
			&l.UtilityCharges, &l.LateFeePct, &l.PaymentFrequency, &l.PaymentDayOfMonth,
			&l.AutoGeneratePayments, &l.LastGeneratedPaymentDt, &l.NoticePeriodDays,
			&l.MoveOutDate, &l.MoveOutInspectionDate, &l.LeaseDocumentURL,
			&l.SignedByLandlordDate, &l.SignedByTenantDate, &l.EjariNumber, &l.EjariURL,
			&l.Status, &l.TerminationReason, &l.TerminationDate, &l.Notes,
			&l.CreatedAt, &l.UpdatedAt, &l.CreatedBy, &l.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan lease: %w", err)
		}
		leases = append(leases, l)
	}
	return leases, nil
}

func (r *LeaseRepo) GetByTenant(ctx context.Context, tenantID uuid.UUID) ([]domain.Lease, error) {
	const q = `
		SELECT id, company_id, property_id, tenant_id, template_id, start_date, end_date,
		       renewal_start_date, renewal_end_date, monthly_rent, currency, security_deposit,
		       utility_charges, late_fee_percent, payment_frequency, payment_day_of_month,
		       auto_generate_payments, last_generated_payment_date, notice_period_days,
		       move_out_date, move_out_inspection_date, lease_document_url,
		       signed_by_landlord_date, signed_by_tenant_date, ejari_number, ejari_url,
		       status, termination_reason, termination_date, notes,
		       created_at, updated_at, created_by, updated_by
		FROM leases
		WHERE tenant_id = $1
		ORDER BY start_date DESC
	`
	rows, err := r.db.Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get leases by tenant: %w", err)
	}
	defer rows.Close()

	var leases []domain.Lease
	for rows.Next() {
		var l domain.Lease
		if err := rows.Scan(
			&l.ID, &l.CompanyID, &l.PropertyID, &l.TenantID, &l.TemplateID, &l.StartDate, &l.EndDate,
			&l.RenewalStartDate, &l.RenewalEndDate, &l.MonthlyRent, &l.Currency, &l.SecurityDeposit,
			&l.UtilityCharges, &l.LateFeePct, &l.PaymentFrequency, &l.PaymentDayOfMonth,
			&l.AutoGeneratePayments, &l.LastGeneratedPaymentDt, &l.NoticePeriodDays,
			&l.MoveOutDate, &l.MoveOutInspectionDate, &l.LeaseDocumentURL,
			&l.SignedByLandlordDate, &l.SignedByTenantDate, &l.EjariNumber, &l.EjariURL,
			&l.Status, &l.TerminationReason, &l.TerminationDate, &l.Notes,
			&l.CreatedAt, &l.UpdatedAt, &l.CreatedBy, &l.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan lease: %w", err)
		}
		leases = append(leases, l)
	}
	return leases, nil
}

func (r *LeaseRepo) UpdateLastGeneratedPaymentDate(ctx context.Context, leaseID uuid.UUID, date *time.Time) error {
	_, err := r.db.Exec(ctx,
		`UPDATE leases SET last_generated_payment_date=$1, updated_at=NOW() WHERE id=$2`,
		date, leaseID,
	)
	return err
}
