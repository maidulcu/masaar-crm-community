package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type ApprovalRepo struct {
	db *pgxpool.Pool
}

func NewApprovalRepo(db *pgxpool.Pool) *ApprovalRepo {
	return &ApprovalRepo{db: db}
}

// GetConfig returns approval workflow config for a company.
func (r *ApprovalRepo) GetConfig(ctx context.Context, companyID uuid.UUID) (*domain.ApprovalConfig, error) {
	const q = `
		SELECT id, company_id, listing_approval, deal_approval_above, offer_approval_above, updated_at
		FROM approval_configs WHERE company_id = $1
	`
	c := &domain.ApprovalConfig{CompanyID: companyID}
	err := r.db.QueryRow(ctx, q, companyID).Scan(
		&c.ID, &c.CompanyID, &c.ListingApproval, &c.DealApprovalAbove, &c.OfferApprovalAbove, &c.UpdatedAt,
	)
	if err != nil {
		return c, nil // return default
	}
	return c, nil
}

// SaveConfig upserts approval workflow config.
func (r *ApprovalRepo) SaveConfig(ctx context.Context, c *domain.ApprovalConfig) error {
	const q = `
		INSERT INTO approval_configs (id, company_id, listing_approval, deal_approval_above, offer_approval_above, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, $4, NOW())
		ON CONFLICT (company_id) DO UPDATE SET
			listing_approval     = EXCLUDED.listing_approval,
			deal_approval_above  = EXCLUDED.deal_approval_above,
			offer_approval_above = EXCLUDED.offer_approval_above,
			updated_at           = NOW()
	`
	_, err := r.db.Exec(ctx, q, c.CompanyID, c.ListingApproval, c.DealApprovalAbove, c.OfferApprovalAbove)
	return err
}

// Create inserts a new approval request.
func (r *ApprovalRepo) Create(ctx context.Context, a *domain.ApprovalRequest) error {
	a.ID = uuid.New()
	const q = `
		INSERT INTO approval_requests
			(id, company_id, entity_type, entity_id, requested_by, status, notes)
		VALUES ($1,$2,$3,$4,$5,'pending',$6)
		RETURNING created_at
	`
	return r.db.QueryRow(ctx, q,
		a.ID, a.CompanyID, a.EntityType, a.EntityID, a.RequestedBy, a.Notes,
	).Scan(&a.CreatedAt)
}

// GetByEntity returns the latest pending approval for an entity, or nil.
func (r *ApprovalRepo) GetByEntity(ctx context.Context, entityType domain.ApprovalEntity, entityID uuid.UUID) (*domain.ApprovalRequest, error) {
	const q = `
		SELECT ar.id, ar.company_id, ar.entity_type, ar.entity_id,
		       ar.requested_by, ar.reviewed_by, ar.status,
		       ar.notes, ar.reviewer_note, ar.created_at, ar.reviewed_at,
		       COALESCE(u1.name,''), COALESCE(u2.name,'')
		FROM approval_requests ar
		LEFT JOIN users u1 ON u1.id = ar.requested_by
		LEFT JOIN users u2 ON u2.id = ar.reviewed_by
		WHERE ar.entity_type = $1 AND ar.entity_id = $2
		ORDER BY ar.created_at DESC LIMIT 1
	`
	a := &domain.ApprovalRequest{}
	err := r.db.QueryRow(ctx, q, entityType, entityID).Scan(
		&a.ID, &a.CompanyID, &a.EntityType, &a.EntityID,
		&a.RequestedBy, &a.ReviewedBy, &a.Status,
		&a.Notes, &a.ReviewerNote, &a.CreatedAt, &a.ReviewedAt,
		&a.RequesterName, &a.ReviewerName,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// List returns approval requests for a company, optionally filtered by status/entity.
func (r *ApprovalRepo) List(ctx context.Context, companyID uuid.UUID, status, entityType string, page, limit int) ([]domain.ApprovalRequest, int, error) {
	if limit == 0 {
		limit = 50
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	args := []any{companyID}
	conds := []string{"ar.company_id = $1"}
	n := 2

	if status != "" {
		conds = append(conds, fmt.Sprintf("ar.status = $%d", n))
		args = append(args, status)
		n++
	}
	if entityType != "" {
		conds = append(conds, fmt.Sprintf("ar.entity_type = $%d", n))
		args = append(args, entityType)
		n++
	}
	where := "WHERE " + joinAnd(conds)

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM approval_requests ar "+where, countArgs...).Scan(&total)

	args = append(args, limit, offset)
	q := fmt.Sprintf(`
		SELECT ar.id, ar.company_id, ar.entity_type, ar.entity_id,
		       ar.requested_by, ar.reviewed_by, ar.status,
		       ar.notes, ar.reviewer_note, ar.created_at, ar.reviewed_at,
		       COALESCE(u1.name,''), COALESCE(u2.name,'')
		FROM approval_requests ar
		LEFT JOIN users u1 ON u1.id = ar.requested_by
		LEFT JOIN users u2 ON u2.id = ar.reviewed_by
		%s
		ORDER BY ar.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, n, n+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []domain.ApprovalRequest
	for rows.Next() {
		var a domain.ApprovalRequest
		if err := rows.Scan(
			&a.ID, &a.CompanyID, &a.EntityType, &a.EntityID,
			&a.RequestedBy, &a.ReviewedBy, &a.Status,
			&a.Notes, &a.ReviewerNote, &a.CreatedAt, &a.ReviewedAt,
			&a.RequesterName, &a.ReviewerName,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, a)
	}
	return list, total, nil
}

// GetByID returns a single approval request by its ID.
func (r *ApprovalRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ApprovalRequest, error) {
	const q = `
		SELECT ar.id, ar.company_id, ar.entity_type, ar.entity_id,
		       ar.requested_by, ar.reviewed_by, ar.status,
		       ar.notes, ar.reviewer_note, ar.created_at, ar.reviewed_at,
		       COALESCE(u1.name,''), COALESCE(u2.name,'')
		FROM approval_requests ar
		LEFT JOIN users u1 ON u1.id = ar.requested_by
		LEFT JOIN users u2 ON u2.id = ar.reviewed_by
		WHERE ar.id = $1
	`
	a := &domain.ApprovalRequest{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&a.ID, &a.CompanyID, &a.EntityType, &a.EntityID,
		&a.RequestedBy, &a.ReviewedBy, &a.Status,
		&a.Notes, &a.ReviewerNote, &a.CreatedAt, &a.ReviewedAt,
		&a.RequesterName, &a.ReviewerName,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// Review approves or rejects a request, sets reviewer + timestamp.
func (r *ApprovalRepo) Review(ctx context.Context, id uuid.UUID, reviewerID uuid.UUID, status domain.ApprovalStatus, note string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE approval_requests
		SET status=$1, reviewed_by=$2, reviewer_note=$3, reviewed_at=$4
		WHERE id=$5 AND status='pending'
	`, status, reviewerID, note, now, id)
	return err
}
