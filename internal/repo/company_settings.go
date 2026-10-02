package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type CompanySettingsRepo struct {
	pool *pgxpool.Pool
}

func NewCompanySettingsRepo(pool *pgxpool.Pool) *CompanySettingsRepo {
	return &CompanySettingsRepo{pool: pool}
}

// Get returns the caller's company settings. (It used to return the first row of the table,
// i.e. whichever company happened to come first, including its bank details.)
func (r *CompanySettingsRepo) Get(ctx context.Context) (*domain.CompanySettings, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT id, name, vat_number, business_address, COALESCE(business_phone,''), COALESCE(business_email,''),
		COALESCE(bank_name,''), COALESCE(bank_account,''), COALESCE(bank_iban,''),
		COALESCE(logo_url,''), COALESCE(disclaimer,''), COALESCE(primary_color,'#1a3a5c'),
		updated_at, updated_by
	FROM company_settings WHERE company_id = $1`

	var s domain.CompanySettings
	err = r.pool.QueryRow(ctx, query, cid).Scan(
		&s.ID, &s.Name, &s.VATNumber, &s.BusinessAddress, &s.BusinessPhone, &s.BusinessEmail,
		&s.BankName, &s.BankAccount, &s.BankIBAN,
		&s.LogoURL, &s.Disclaimer, &s.PrimaryColor,
		&s.UpdatedAt, &s.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CompanySettingsRepo) Update(ctx context.Context, settings *domain.CompanySettings, userID *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE company_settings SET
		name=$1, vat_number=$2, business_address=$3, business_phone=$4, business_email=$5,
		bank_name=$6, bank_account=$7, bank_iban=$8,
		logo_url=$9, disclaimer=$10, primary_color=$11,
		updated_at=$12, updated_by=$13
		WHERE id=$14 AND company_id=$15`

	_, err = r.pool.Exec(ctx, query,
		settings.Name, settings.VATNumber, settings.BusinessAddress,
		settings.BusinessPhone, settings.BusinessEmail,
		settings.BankName, settings.BankAccount, settings.BankIBAN,
		settings.LogoURL, settings.Disclaimer, settings.PrimaryColor,
		time.Now(), userID, settings.ID, cid,
	)
	return err
}
