package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type LeadTagRepo struct {
	pool *pgxpool.Pool
}

func NewLeadTagRepo(pool *pgxpool.Pool) *LeadTagRepo {
	return &LeadTagRepo{pool: pool}
}

func (r *LeadTagRepo) Create(ctx context.Context, tag *domain.LeadTag) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// Tags belong to a lead; the lead must be in the caller's company.
	query := `INSERT INTO lead_tags (lead_id, tag, category, auto_applied, created_by)
	SELECT $1, $2, $3, $4, $5
	WHERE EXISTS (SELECT 1 FROM leads WHERE id = $1 AND company_id = $6)
	RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		tag.LeadID,
		tag.Tag,
		tag.Category,
		tag.AutoApplied,
		tag.CreatedBy,
		cid,
	).Scan(&tag.ID, &tag.CreatedAt)
}

func (r *LeadTagRepo) GetByLead(ctx context.Context, leadID uuid.UUID) ([]string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT t.tag FROM lead_tags t JOIN leads l ON l.id = t.lead_id
	WHERE t.lead_id = $1 AND l.company_id = $2 ORDER BY t.tag`

	rows, err := r.pool.Query(ctx, query, leadID, cid)
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
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	query := `INSERT INTO lead_tags (lead_id, tag, category, auto_applied, created_by)
	SELECT $1, $2, $3, $4, $5
	WHERE EXISTS (SELECT 1 FROM leads WHERE id = $1 AND company_id = $6)
	ON CONFLICT(lead_id, tag) DO NOTHING`

	_, err = r.pool.Exec(ctx, query, leadID, tag, category, autoApplied, userID, cid)
	return err
}

func (r *LeadTagRepo) RemoveTag(ctx context.Context, leadID uuid.UUID, tag string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	query := `DELETE FROM lead_tags WHERE lead_id = $1 AND tag = $2
	AND EXISTS (SELECT 1 FROM leads WHERE id = $1 AND company_id = $3)`
	_, err = r.pool.Exec(ctx, query, leadID, tag, cid)
	return err
}

func (r *LeadTagRepo) ListByCategory(ctx context.Context, leadID uuid.UUID, category string) ([]string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	query := `SELECT t.tag FROM lead_tags t JOIN leads l ON l.id = t.lead_id
	WHERE t.lead_id = $1 AND t.category = $2 AND l.company_id = $3`

	rows, err := r.pool.Query(ctx, query, leadID, category, cid)
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
