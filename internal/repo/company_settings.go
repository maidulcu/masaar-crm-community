package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
		country, region, currency, timezone, show_hijri,
		orn, trade_license_no, trade_license_expiry,
		updated_at, updated_by
	FROM company_settings WHERE company_id = $1`

	var s domain.CompanySettings
	err = r.pool.QueryRow(ctx, query, cid).Scan(
		&s.ID, &s.Name, &s.VATNumber, &s.BusinessAddress, &s.BusinessPhone, &s.BusinessEmail,
		&s.BankName, &s.BankAccount, &s.BankIBAN,
		&s.LogoURL, &s.Disclaimer, &s.PrimaryColor,
		&s.Country, &s.Region, &s.Currency, &s.Timezone, &s.ShowHijri,
		&s.ORN, &s.TradeLicenseNo, &s.TradeLicenseExpiry,
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
		updated_at=$12, updated_by=$13,
		country=$16, region=$17, currency=$18, timezone=$19,
		orn=$20, trade_license_no=$21, trade_license_expiry=$22,
		show_hijri=COALESCE($23, show_hijri)
		WHERE id=$14 AND company_id=$15`

	_, err = r.pool.Exec(ctx, query,
		settings.Name, settings.VATNumber, settings.BusinessAddress,
		settings.BusinessPhone, settings.BusinessEmail,
		settings.BankName, settings.BankAccount, settings.BankIBAN,
		settings.LogoURL, settings.Disclaimer, settings.PrimaryColor,
		time.Now(), userID, settings.ID, cid,
		settings.Country, settings.Region, settings.Currency, settings.Timezone,
		settings.ORN, settings.TradeLicenseNo, settings.TradeLicenseExpiry,
		settings.ShowHijri,
	)
	return err
}

// Locale returns the caller's company's country, region, currency and timezone. It is
// readable by every role (amounts and dates are formatted with it), unlike Get, which
// includes bank details.
func (r *CompanySettingsRepo) Locale(ctx context.Context) (*domain.CompanyLocale, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	l := &domain.CompanyLocale{CompanyID: cid, Country: "AE", Currency: "AED", Timezone: "Asia/Dubai"}
	err = r.pool.QueryRow(ctx,
		`SELECT country, region, currency, timezone, show_hijri FROM company_settings WHERE company_id = $1`, cid,
	).Scan(&l.Country, &l.Region, &l.Currency, &l.Timezone, &l.ShowHijri)
	if errors.Is(err, pgx.ErrNoRows) {
		return l, nil // no settings row yet: the defaults the migration would have given it
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}
