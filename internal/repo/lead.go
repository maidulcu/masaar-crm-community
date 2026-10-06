package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// StageTotal is the full size of a Kanban column, which can exceed the cards loaded into it.
type StageTotal struct {
	Count int     `json:"count"`
	Value float64 `json:"value"`
}

// KanbanBoard returns active (non-deleted) leads grouped by stage with joined contact. Each
// stage holds at most perStage of its newest leads, so a company with a large history does not
// ship its whole pipeline on every board load; totals carries the real count and deal value of
// every stage, so columns can still show correct headers and a "load more" control.
func (r *LeadRepo) KanbanBoard(ctx context.Context, perStage int) (map[domain.LeadStage][]domain.Lead, map[domain.LeadStage]StageTotal, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, nil, err
	}
	if perStage < 1 {
		perStage = 1
	}
	const q = `
		WITH ranked AS (
			SELECT l.id,
			       ROW_NUMBER() OVER (PARTITION BY l.stage ORDER BY l.created_at DESC, l.id DESC) AS rn,
			       COUNT(*)                OVER (PARTITION BY l.stage) AS stage_count,
			       COALESCE(SUM(l.deal_value) OVER (PARTITION BY l.stage), 0) AS stage_value
			FROM leads l
			WHERE l.deleted_at IS NULL AND l.company_id = $1
		)
		SELECT` + leadCols + `,
		       c.id, c.phone_wa, c.full_name, COALESCE(c.email,''), c.language, c.lead_score,
		       r.stage_count, r.stage_value
		FROM ranked r
		JOIN leads l ON l.id = r.id
		JOIN contacts c ON c.id = l.contact_id AND c.company_id = l.company_id
		WHERE r.rn <= $2
		ORDER BY l.stage, l.created_at DESC, l.id DESC
	`
	rows, err := r.db.Query(ctx, q, cid, perStage)
	if err != nil {
		return nil, nil, fmt.Errorf("kanban query: %w", err)
	}
	defer rows.Close()

	board := map[domain.LeadStage][]domain.Lead{}
	totals := map[domain.LeadStage]StageTotal{}
	for rows.Next() {
		var l domain.Lead
		var c domain.Contact
		var t StageTotal
		if err := rows.Scan(
			&l.ID, &l.ContactID, &l.Stage, &l.Source,
			&l.DealValue, &l.Currency, &l.Notes,
			&l.LeadScore, &l.ScoreUpdatedAt,
			&l.AssignedTo, &l.ClosedReason, &l.LastContactedAt,
			&l.CreatedAt, &l.UpdatedAt,
			&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
			&t.Count, &t.Value,
		); err != nil {
			return nil, nil, fmt.Errorf("scan kanban row: %w", err)
		}
		l.Contact = &c
		board[l.Stage] = append(board[l.Stage], l)
		totals[l.Stage] = t
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("kanban rows: %w", err)
	}
	return board, totals, nil
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
	// Keyset cursor: only leads strictly older than (BeforeCreatedAt, BeforeID) in the
	// list order. Unlike Offset it stays correct when cards are added or moved meanwhile.
	BeforeCreatedAt *time.Time
	BeforeID        *uuid.UUID
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
	if f.BeforeCreatedAt != nil {
		beforeID := uuid.Max // no id given: everything created before that instant
		if f.BeforeID != nil {
			beforeID = *f.BeforeID
		}
		conds = append(conds, fmt.Sprintf("(l.created_at, l.id) < ($%d, $%d)", n, n+1))
		args = append(args, *f.BeforeCreatedAt, beforeID)
		n += 2
	}

	where := "WHERE " + strings.Join(conds, " AND ")
	args = append(args, f.Limit, f.Offset)

	q := fmt.Sprintf(`
		SELECT`+leadCols+`,
		       c.id, c.phone_wa, c.full_name, COALESCE(c.email,''), c.language, c.lead_score
		FROM leads l
		JOIN contacts c ON c.id = l.contact_id AND c.company_id = l.company_id
		%s
		ORDER BY l.created_at DESC, l.id DESC
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
