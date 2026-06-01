package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type SettingsRepo struct {
	db *pgxpool.Pool
}

func NewSettingsRepo(db *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(ctx context.Context, key string) (*domain.APISetting, error) {
	query := `
		SELECT id, setting_key, setting_value, description, updated_at, updated_by
		FROM api_settings
		WHERE setting_key = $1
	`
	var setting domain.APISetting
	err := r.db.QueryRow(ctx, query, key).Scan(
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
	query := `
		UPDATE api_settings
		SET setting_value = $1, updated_at = NOW(), updated_by = $2
		WHERE setting_key = $3
	`
	_, err := r.db.Exec(ctx, query, value, updatedBy, key)
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
