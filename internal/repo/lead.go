package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type LeadRepo struct {
	db *pgxpool.Pool
}

func NewLeadRepo(db *pgxpool.Pool) *LeadRepo {
	return &LeadRepo{db: db}
}

// leadCols is the base SELECT column list for a lead row (no joins).
const leadCols = `
	l.id, l.contact_id, l.stage, l.source, l.deal_value, l.currency, l.notes,
	l.lead_score, l.score_updated_at,
	l.assigned_to, l.closed_reason, l.last_contacted_at,
	l.created_at, l.updated_at`

func scanLead(row interface {
	Scan(...any) error
}, l *domain.Lead) error {
	return row.Scan(
		&l.ID, &l.ContactID, &l.Stage, &l.Source,
		&l.DealValue, &l.Currency, &l.Notes,
		&l.LeadScore, &l.ScoreUpdatedAt,
		&l.AssignedTo, &l.ClosedReason, &l.LastContactedAt,
		&l.CreatedAt, &l.UpdatedAt,
	)
}

// KanbanBoard returns active (non-deleted) leads grouped by stage with joined contact.
func (r *LeadRepo) KanbanBoard(ctx context.Context) (map[domain.LeadStage][]domain.Lead, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT` + leadCols + `,
		       c.id, c.phone_wa, c.full_name, COALESCE(c.email,''), c.language, c.lead_score
		FROM leads l
		JOIN contacts c ON c.id = l.contact_id AND c.company_id = l.company_id
		WHERE l.deleted_at IS NULL AND l.company_id = $1
		ORDER BY l.stage, l.created_at DESC
	`
	rows, err := r.db.Query(ctx, q, cid)
	if err != nil {
		return nil, fmt.Errorf("kanban query: %w", err)
	}
	defer rows.Close()

	board := map[domain.LeadStage][]domain.Lead{}
	for rows.Next() {
		var l domain.Lead
		var c domain.Contact
		if err := rows.Scan(
			&l.ID, &l.ContactID, &l.Stage, &l.Source,
			&l.DealValue, &l.Currency, &l.Notes,
			&l.LeadScore, &l.ScoreUpdatedAt,
			&l.AssignedTo, &l.ClosedReason, &l.LastContactedAt,
			&l.CreatedAt, &l.UpdatedAt,
			&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
		); err != nil {
			return nil, fmt.Errorf("scan kanban row: %w", err)
		}
		l.Contact = &c
		board[l.Stage] = append(board[l.Stage], l)
	}
	return board, nil
}

// LeadFilter holds optional search/filter parameters for List.
type LeadFilter struct {
	Query      string           // full-text search on contact name or phone
	Stage      domain.LeadStage // filter by stage
	AssignedTo *uuid.UUID       // filter by agent
	Source     string           // filter by source
	ContactID  *uuid.UUID       // filter by contact
	Limit      int              // default 50
	Offset     int
}

// List returns leads matching the filter, ordered by created_at DESC.
func (r *LeadRepo) List(ctx context.Context, f LeadFilter) ([]domain.Lead, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	if f.Limit == 0 {
		f.Limit = 50
	}

	args := []any{cid}
	conds := []string{"l.deleted_at IS NULL", "l.company_id = $1"}
	n := 2

	if f.Query != "" {
		conds = append(conds, fmt.Sprintf(
			"(c.full_name ILIKE $%d OR c.phone_wa ILIKE $%d)", n, n+1,
		))
		like := "%" + f.Query + "%"
		args = append(args, like, like)
		n += 2
	}
	if f.Stage != "" {
		conds = append(conds, fmt.Sprintf("l.stage = $%d", n))
		args = append(args, string(f.Stage))
		n++
	}
	if f.AssignedTo != nil {
		conds = append(conds, fmt.Sprintf("l.assigned_to = $%d", n))
		args = append(args, *f.AssignedTo)
		n++
	}
	if f.Source != "" {
		conds = append(conds, fmt.Sprintf("l.source = $%d", n))
		args = append(args, f.Source)
		n++
	}
	if f.ContactID != nil {
		conds = append(conds, fmt.Sprintf("l.contact_id = $%d", n))
		args = append(args, *f.ContactID)
		n++
	}

	where := "WHERE " + strings.Join(conds, " AND ")
	args = append(args, f.Limit, f.Offset)

	q := fmt.Sprintf(`
		SELECT`+leadCols+`,
		       c.id, c.phone_wa, c.full_name, COALESCE(c.email,''), c.language, c.lead_score
		FROM leads l
		JOIN contacts c ON c.id = l.contact_id AND c.company_id = l.company_id
		%s
		ORDER BY l.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, n, n+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list leads: %w", err)
	}
	defer rows.Close()

	var leads []domain.Lead
	for rows.Next() {
		var l domain.Lead
		var c domain.Contact
		if err := rows.Scan(
			&l.ID, &l.ContactID, &l.Stage, &l.Source,
			&l.DealValue, &l.Currency, &l.Notes,
			&l.LeadScore, &l.ScoreUpdatedAt,
			&l.AssignedTo, &l.ClosedReason, &l.LastContactedAt,
			&l.CreatedAt, &l.UpdatedAt,
			&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
		); err != nil {
			return nil, fmt.Errorf("scan lead row: %w", err)
		}
		l.Contact = &c
		leads = append(leads, l)
	}
	return leads, nil
}

func (r *LeadRepo) Create(ctx context.Context, l *domain.Lead) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// The contact (and agent, if any) must belong to the same company; a foreign
	// UUID inserts nothing and surfaces as ErrNoRows.
	const q = `
		INSERT INTO leads (id, company_id, contact_id, stage, source, deal_value, currency, notes, assigned_to)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9
		WHERE EXISTS (SELECT 1 FROM contacts WHERE id = $3 AND company_id = $2)
		  AND ($9::uuid IS NULL OR EXISTS (SELECT 1 FROM users WHERE id = $9 AND company_id = $2))
		RETURNING created_at, updated_at
	`
	l.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		l.ID, cid, l.ContactID, l.Stage, l.Source, l.DealValue, l.Currency, l.Notes, l.AssignedTo,
	).Scan(&l.CreatedAt, &l.UpdatedAt)
}

func (r *LeadRepo) UpdateNotes(ctx context.Context, id uuid.UUID, notes string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE leads SET notes=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3 AND deleted_at IS NULL`,
		notes, id, cid,
	)
	return err
}

// UpdateStage moves the lead stage and optionally sets closed_reason for won/lost.
func (r *LeadRepo) UpdateStage(ctx context.Context, id uuid.UUID, stage domain.LeadStage, reason string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE leads SET stage=$1, closed_reason=$2, updated_at=NOW() WHERE id=$3 AND company_id=$4 AND deleted_at IS NULL`,
		stage, reason, id, cid,
	)
	return err
}

// Assign sets or clears the agent assigned to a lead.
func (r *LeadRepo) Assign(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE leads SET assigned_to=$1, updated_at=NOW()
		 WHERE id=$2 AND company_id=$3 AND deleted_at IS NULL
		   AND ($1::uuid IS NULL OR EXISTS (SELECT 1 FROM users WHERE id=$1 AND company_id=$3))`,
		userID, id, cid,
	)
	return err
}

// TouchLastContacted updates last_contacted_at to now.
func (r *LeadRepo) TouchLastContacted(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE leads SET last_contacted_at=NOW(), updated_at=NOW() WHERE id=$1 AND company_id=$2 AND deleted_at IS NULL`,
		id, cid,
	)
	return err
}

func (r *LeadRepo) UpdateScore(ctx context.Context, id uuid.UUID, score int) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE leads SET lead_score=$1, score_updated_at=NOW(), updated_at=NOW() WHERE id=$2 AND company_id=$3`,
		score, id, cid,
	)
	return err
}

func (r *LeadRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Lead, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	q := `SELECT` + leadCols + `
		FROM leads l WHERE l.id = $1 AND l.company_id = $2 AND l.deleted_at IS NULL`
	l := &domain.Lead{}
	if err := scanLead(r.db.QueryRow(ctx, q, id, cid), l); err != nil {
		return nil, fmt.Errorf("get lead by id: %w", err)
	}
	return l, nil
}

// Delete soft-deletes a lead by setting deleted_at.
func (r *LeadRepo) Delete(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx,
		`UPDATE leads SET deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND company_id=$2 AND deleted_at IS NULL`,
		id, cid,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("lead not found")
	}
	return nil
}
