package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type TenantRepo struct {
	db *pgxpool.Pool
}

func NewTenantRepo(db *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{db: db}
}

func (r *TenantRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.Tenant], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM tenants WHERE company_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID).Scan(&total); err != nil {
		return nil, fmt.Errorf("count tenants: %w", err)
	}

	const q = `
		SELECT id, company_id, full_name_en, full_name_ar, email, phone, phone_wa, id_type, id_number,
		       id_expiry_date, id_document_url, is_verified, verification_status, verification_date,
		       verified_by, verification_notes, employment_status, employer_name, annual_income,
		       income_currency, salary_certificate_url, nationality, country_of_origin, permanent_address,
		       emergency_contact_name, emergency_contact_phone, status, notes, created_at, updated_at,
		       created_by, updated_by
		FROM tenants
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(
			&t.ID, &t.CompanyID, &t.FullNameEN, &t.FullNameAR, &t.Email, &t.Phone, &t.PhoneWA, &t.IDType, &t.IDNumber,
			&t.IDExpiryDate, &t.IDDocumentURL, &t.IsVerified, &t.VerificationStatus, &t.VerificationDate,
			&t.VerifiedBy, &t.VerificationNotes, &t.EmploymentStatus, &t.EmployerName, &t.AnnualIncome,
			&t.IncomeCurrency, &t.SalaryCertificateURL, &t.Nationality, &t.CountryOfOrigin, &t.PermanentAddress,
			&t.EmergencyContactName, &t.EmergencyContactPhone, &t.Status, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
			&t.CreatedBy, &t.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}

	return &domain.PaginatedResult[domain.Tenant]{
		Data:  tenants,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *TenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	const q = `
		SELECT id, company_id, full_name_en, full_name_ar, email, phone, phone_wa, id_type, id_number,
		       id_expiry_date, id_document_url, is_verified, verification_status, verification_date,
		       verified_by, verification_notes, employment_status, employer_name, annual_income,
		       income_currency, salary_certificate_url, nationality, country_of_origin, permanent_address,
		       emergency_contact_name, emergency_contact_phone, status, notes, created_at, updated_at,
		       created_by, updated_by
		FROM tenants WHERE id = $1
	`
	t := &domain.Tenant{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&t.ID, &t.CompanyID, &t.FullNameEN, &t.FullNameAR, &t.Email, &t.Phone, &t.PhoneWA, &t.IDType, &t.IDNumber,
		&t.IDExpiryDate, &t.IDDocumentURL, &t.IsVerified, &t.VerificationStatus, &t.VerificationDate,
		&t.VerifiedBy, &t.VerificationNotes, &t.EmploymentStatus, &t.EmployerName, &t.AnnualIncome,
		&t.IncomeCurrency, &t.SalaryCertificateURL, &t.Nationality, &t.CountryOfOrigin, &t.PermanentAddress,
		&t.EmergencyContactName, &t.EmergencyContactPhone, &t.Status, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
		&t.CreatedBy, &t.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return t, nil
}

func (r *TenantRepo) Create(ctx context.Context, t *domain.Tenant) error {
	const q = `
		INSERT INTO tenants (
			id, company_id, full_name_en, full_name_ar, email, phone, phone_wa, id_type, id_number,
			id_expiry_date, id_document_url, is_verified, verification_status, verification_date,
			verified_by, verification_notes, employment_status, employer_name, annual_income,
			income_currency, salary_certificate_url, nationality, country_of_origin, permanent_address,
			emergency_contact_name, emergency_contact_phone, status, notes, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30
		)
		RETURNING created_at, updated_at
	`
	t.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		t.ID, t.CompanyID, t.FullNameEN, t.FullNameAR, t.Email, t.Phone, t.PhoneWA, t.IDType, t.IDNumber,
		t.IDExpiryDate, t.IDDocumentURL, t.IsVerified, t.VerificationStatus, t.VerificationDate,
		t.VerifiedBy, t.VerificationNotes, t.EmploymentStatus, t.EmployerName, t.AnnualIncome,
		t.IncomeCurrency, t.SalaryCertificateURL, t.Nationality, t.CountryOfOrigin, t.PermanentAddress,
		t.EmergencyContactName, t.EmergencyContactPhone, t.Status, t.Notes, t.CreatedBy, t.UpdatedBy,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
}

func (r *TenantRepo) Update(ctx context.Context, t *domain.Tenant) error {
	const q = `
		UPDATE tenants
		SET full_name_en=$1, full_name_ar=$2, email=$3, phone=$4, phone_wa=$5, id_type=$6, id_number=$7,
		    id_expiry_date=$8, id_document_url=$9, is_verified=$10, verification_status=$11, verification_date=$12,
		    verified_by=$13, verification_notes=$14, employment_status=$15, employer_name=$16, annual_income=$17,
		    income_currency=$18, salary_certificate_url=$19, nationality=$20, country_of_origin=$21,
		    permanent_address=$22, emergency_contact_name=$23, emergency_contact_phone=$24, status=$25,
		    notes=$26, updated_by=$27, updated_at=NOW()
		WHERE id=$28
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, q,
		t.FullNameEN, t.FullNameAR, t.Email, t.Phone, t.PhoneWA, t.IDType, t.IDNumber,
		t.IDExpiryDate, t.IDDocumentURL, t.IsVerified, t.VerificationStatus, t.VerificationDate,
		t.VerifiedBy, t.VerificationNotes, t.EmploymentStatus, t.EmployerName, t.AnnualIncome,
		t.IncomeCurrency, t.SalaryCertificateURL, t.Nationality, t.CountryOfOrigin,
		t.PermanentAddress, t.EmergencyContactName, t.EmergencyContactPhone, t.Status,
		t.Notes, t.UpdatedBy, t.ID,
	).Scan(&t.UpdatedAt)
}

func (r *TenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tenants WHERE id=$1`, id)
	return err
}

func (r *TenantRepo) GetByIDNumber(ctx context.Context, idNumber string) (*domain.Tenant, error) {
	const q = `
		SELECT id, company_id, full_name_en, full_name_ar, email, phone, phone_wa, id_type, id_number,
		       id_expiry_date, id_document_url, is_verified, verification_status, verification_date,
		       verified_by, verification_notes, employment_status, employer_name, annual_income,
		       income_currency, salary_certificate_url, nationality, country_of_origin, permanent_address,
		       emergency_contact_name, emergency_contact_phone, status, notes, created_at, updated_at,
		       created_by, updated_by
		FROM tenants WHERE id_number = $1
	`
	t := &domain.Tenant{}
	err := r.db.QueryRow(ctx, q, idNumber).Scan(
		&t.ID, &t.CompanyID, &t.FullNameEN, &t.FullNameAR, &t.Email, &t.Phone, &t.PhoneWA, &t.IDType, &t.IDNumber,
		&t.IDExpiryDate, &t.IDDocumentURL, &t.IsVerified, &t.VerificationStatus, &t.VerificationDate,
		&t.VerifiedBy, &t.VerificationNotes, &t.EmploymentStatus, &t.EmployerName, &t.AnnualIncome,
		&t.IncomeCurrency, &t.SalaryCertificateURL, &t.Nationality, &t.CountryOfOrigin, &t.PermanentAddress,
		&t.EmergencyContactName, &t.EmergencyContactPhone, &t.Status, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
		&t.CreatedBy, &t.UpdatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("get tenant by id number: %w", err)
	}
	return t, nil
}

func (r *TenantRepo) ListByStatus(ctx context.Context, companyID uuid.UUID, status domain.TenantStatus, page, limit int) (*domain.PaginatedResult[domain.Tenant], error) {
	offset := (page - 1) * limit

	const countQ = `SELECT COUNT(*) FROM tenants WHERE company_id = $1 AND status = $2`
	var total int
	if err := r.db.QueryRow(ctx, countQ, companyID, status).Scan(&total); err != nil {
		return nil, fmt.Errorf("count tenants by status: %w", err)
	}

	const q = `
		SELECT id, company_id, full_name_en, full_name_ar, email, phone, phone_wa, id_type, id_number,
		       id_expiry_date, id_document_url, is_verified, verification_status, verification_date,
		       verified_by, verification_notes, employment_status, employer_name, annual_income,
		       income_currency, salary_certificate_url, nationality, country_of_origin, permanent_address,
		       emergency_contact_name, emergency_contact_phone, status, notes, created_at, updated_at,
		       created_by, updated_by
		FROM tenants
		WHERE company_id = $1 AND status = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, q, companyID, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list tenants by status: %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(
			&t.ID, &t.CompanyID, &t.FullNameEN, &t.FullNameAR, &t.Email, &t.Phone, &t.PhoneWA, &t.IDType, &t.IDNumber,
			&t.IDExpiryDate, &t.IDDocumentURL, &t.IsVerified, &t.VerificationStatus, &t.VerificationDate,
			&t.VerifiedBy, &t.VerificationNotes, &t.EmploymentStatus, &t.EmployerName, &t.AnnualIncome,
			&t.IncomeCurrency, &t.SalaryCertificateURL, &t.Nationality, &t.CountryOfOrigin, &t.PermanentAddress,
			&t.EmergencyContactName, &t.EmergencyContactPhone, &t.Status, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
			&t.CreatedBy, &t.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}

	return &domain.PaginatedResult[domain.Tenant]{
		Data:  tenants,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
