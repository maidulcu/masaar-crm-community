package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type PipelineStageRepo struct {
	db *pgxpool.Pool
}

func NewPipelineStageRepo(db *pgxpool.Pool) *PipelineStageRepo {
	return &PipelineStageRepo{db: db}
}

func (r *PipelineStageRepo) ListByCompany(ctx context.Context, companyID uuid.UUID, entityType string) ([]domain.PipelineStage, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, company_id, entity_type, name, sort_order, color, is_won, is_lost, is_default, created_at
		FROM pipeline_stages
		WHERE company_id = $1 AND entity_type = $2
		ORDER BY sort_order ASC
	`, companyID, entityType)
	if err != nil {
		return nil, fmt.Errorf("list pipeline stages: %w", err)
	}
	defer rows.Close()

	var stages []domain.PipelineStage
	for rows.Next() {
		var s domain.PipelineStage
		if err := rows.Scan(&s.ID, &s.CompanyID, &s.EntityType, &s.Name, &s.SortOrder, &s.Color, &s.IsWon, &s.IsLost, &s.IsDefault, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan pipeline stage: %w", err)
		}
		stages = append(stages, s)
	}
	return stages, nil
}

func (r *PipelineStageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.PipelineStage, error) {
	var s domain.PipelineStage
	err := r.db.QueryRow(ctx, `
		SELECT id, company_id, entity_type, name, sort_order, color, is_won, is_lost, is_default, created_at
		FROM pipeline_stages WHERE id = $1
	`, id).Scan(&s.ID, &s.CompanyID, &s.EntityType, &s.Name, &s.SortOrder, &s.Color, &s.IsWon, &s.IsLost, &s.IsDefault, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get pipeline stage: %w", err)
	}
	return &s, nil
}

func (r *PipelineStageRepo) Create(ctx context.Context, s *domain.PipelineStage) error {
	s.ID = uuid.New()
	return r.db.QueryRow(ctx, `
		INSERT INTO pipeline_stages (id, company_id, entity_type, name, sort_order, color, is_won, is_lost, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`, s.ID, s.CompanyID, s.EntityType, s.Name, s.SortOrder, s.Color, s.IsWon, s.IsLost, s.IsDefault,
	).Scan(&s.CreatedAt)
}

func (r *PipelineStageRepo) Update(ctx context.Context, s *domain.PipelineStage) error {
	_, err := r.db.Exec(ctx, `
		UPDATE pipeline_stages
		SET name=$1, sort_order=$2, color=$3, is_won=$4, is_lost=$5
		WHERE id=$6
	`, s.Name, s.SortOrder, s.Color, s.IsWon, s.IsLost, s.ID)
	return err
}

func (r *PipelineStageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM pipeline_stages WHERE id=$1`, id)
	return err
}

func (r *PipelineStageRepo) Reorder(ctx context.Context, ids []uuid.UUID) error {
	for i, id := range ids {
		if _, err := r.db.Exec(ctx, `UPDATE pipeline_stages SET sort_order=$1 WHERE id=$2`, i, id); err != nil {
			return fmt.Errorf("reorder stage %s: %w", id, err)
		}
	}
	return nil
}

func (r *PipelineStageRepo) SetDefaults(ctx context.Context, companyID uuid.UUID, entityType string) error {
	defaults := []struct {
		Name      string
		SortOrder int
		Color     string
		IsWon     bool
		IsLost    bool
	}{
		{"new", 0, "#a3a3a3", false, false},
		{"contacted", 1, "#0ea5e9", false, false},
		{"qualified", 2, "#6366f1", false, false},
		{"proposal", 3, "#f59e0b", false, false},
		{"won", 4, "#10b981", true, false},
		{"lost", 5, "#ef4444", false, true},
	}

	for _, d := range defaults {
		_, err := r.db.Exec(ctx, `
			INSERT INTO pipeline_stages (company_id, entity_type, name, sort_order, color, is_won, is_lost, is_default)
			VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
			ON CONFLICT (company_id, entity_type, name) DO UPDATE
			SET sort_order = EXCLUDED.sort_order, color = EXCLUDED.color,
			    is_won = EXCLUDED.is_won, is_lost = EXCLUDED.is_lost
		`, companyID, entityType, d.Name, d.SortOrder, d.Color, d.IsWon, d.IsLost)
		if err != nil {
			return fmt.Errorf("set default stage %s: %w", d.Name, err)
		}
	}
	return nil
}
