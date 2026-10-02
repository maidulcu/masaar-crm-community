package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type SettingsRepo struct {
	db *pgxpool.Pool
}

func NewSettingsRepo(db *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{db: db}
}

// Settings are per company: one company's integration tokens are never visible to another.
func (r *SettingsRepo) Get(ctx context.Context, key string) (*domain.APISetting, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT id, setting_key, setting_value, COALESCE(description,''), updated_at, updated_by
		FROM api_settings
		WHERE setting_key = $1 AND company_id = $2
	`
	var setting domain.APISetting
	err = r.db.QueryRow(ctx, query, key, cid).Scan(
		&setting.ID,
		&setting.SettingKey,
		&setting.SettingValue,
		&setting.Description,
		&setting.UpdatedAt,
		&setting.UpdatedBy,
	)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *SettingsRepo) Update(ctx context.Context, key, value string, updatedBy *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// Upsert: new companies have no row until they first save a value.
	query := `
		INSERT INTO api_settings (company_id, setting_key, setting_value, updated_at, updated_by)
		VALUES ($1, $2, $3, NOW(), $4)
		ON CONFLICT (company_id, setting_key)
		DO UPDATE SET setting_value = EXCLUDED.setting_value, updated_at = NOW(), updated_by = EXCLUDED.updated_by
	`
	_, err = r.db.Exec(ctx, query, cid, key, value, updatedBy)
	return err
}

func (r *SettingsRepo) GetBOS24Token(ctx context.Context) (string, error) {
	setting, err := r.Get(ctx, "bos24_api_token")
	if err != nil {
		return "", err
	}
	return setting.SettingValue, nil
}

func (r *SettingsRepo) UpdateBOS24Token(ctx context.Context, token string, updatedBy *uuid.UUID) error {
	return r.Update(ctx, "bos24_api_token", token, updatedBy)
}
