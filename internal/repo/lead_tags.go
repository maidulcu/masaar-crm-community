package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type LeadTagRepo struct {
	pool *pgxpool.Pool
}

func NewLeadTagRepo(pool *pgxpool.Pool) *LeadTagRepo {
	return &LeadTagRepo{pool: pool}
}

func (r *LeadTagRepo) Create(ctx context.Context, tag *domain.LeadTag) error {
	query := `INSERT INTO lead_tags (lead_id, tag, category, auto_applied, created_by)
	VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		tag.LeadID,
		tag.Tag,
		tag.Category,
		tag.AutoApplied,
		tag.CreatedBy,
	).Scan(&tag.ID, &tag.CreatedAt)
}

func (r *LeadTagRepo) GetByLead(ctx context.Context, leadID uuid.UUID) ([]string, error) {
	query := `SELECT tag FROM lead_tags WHERE lead_id = $1 ORDER BY tag`

	rows, err := r.pool.Query(ctx, query, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

func (r *LeadTagRepo) AddTag(ctx context.Context, leadID uuid.UUID, tag string, category string, autoApplied bool, userID *uuid.UUID) error {
	query := `INSERT INTO lead_tags (lead_id, tag, category, auto_applied, created_by)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT(lead_id, tag) DO NOTHING`

	_, err := r.pool.Exec(ctx, query, leadID, tag, category, autoApplied, userID)
	return err
}

func (r *LeadTagRepo) RemoveTag(ctx context.Context, leadID uuid.UUID, tag string) error {
	query := `DELETE FROM lead_tags WHERE lead_id = $1 AND tag = $2`
	_, err := r.pool.Exec(ctx, query, leadID, tag)
	return err
}

func (r *LeadTagRepo) ListByCategory(ctx context.Context, leadID uuid.UUID, category string) ([]string, error) {
	query := `SELECT tag FROM lead_tags WHERE lead_id = $1 AND category = $2`

	rows, err := r.pool.Query(ctx, query, leadID, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// AutoTag applies tags based on enrichment data
func (r *LeadTagRepo) AutoTag(ctx context.Context, leadID uuid.UUID, tags map[string][]string, userID *uuid.UUID) error {
	for category, tagList := range tags {
		for _, tag := range tagList {
			if err := r.AddTag(ctx, leadID, tag, category, true, userID); err != nil {
				return err
			}
		}
	}
	return nil
}
