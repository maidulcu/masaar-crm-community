package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Domain types (kept inline since commission schema is self-contained) ─────

type CommissionType string

const (
	CommissionFixed      CommissionType = "fixed"
	CommissionPercentage CommissionType = "percentage"
	CommissionTiered     CommissionType = "tiered"
)

type CommissionStructure struct {
	ID            uuid.UUID       `json:"id"`
	CompanyID     uuid.UUID       `json:"company_id"`
	Name          string          `json:"structure_name"`
	Type          CommissionType  `json:"commission_type"`
	ApplicableTo  string          `json:"applicable_to"` // deals, leases, all
	EffectiveFrom *time.Time      `json:"effective_from"`
	EffectiveTo   *time.Time      `json:"effective_to"`
	Rules         json.RawMessage `json:"rules"` // flexible JSONB
	CreatedAt     time.Time       `json:"created_at"`
}

type CommissionStatus string

const (
	CommissionPending  CommissionStatus = "pending"
	CommissionApproved CommissionStatus = "approved"
	CommissionPaid     CommissionStatus = "paid"
	CommissionDisputed CommissionStatus = "disputed"
)

type AgentCommission struct {
	ID                  uuid.UUID        `json:"id"`
	CompanyID           uuid.UUID        `json:"company_id"`
	AgentID             uuid.UUID        `json:"agent_id"`
	AgentName           string           `json:"agent_name,omitempty"`
	PeriodStart         time.Time        `json:"commission_period_start"`
	PeriodEnd           time.Time        `json:"commission_period_end"`
	StructureID         *uuid.UUID       `json:"commission_structure_id"`
	DealsCount          int              `json:"deals_count"`
	DealsRevenue        float64          `json:"deals_revenue"`
	LeasesCount         int              `json:"leases_count"`
	LeasesRevenue       float64          `json:"leases_revenue"`
	TotalCommission     *float64         `json:"total_commission"`
	Status              CommissionStatus `json:"status"`
	ApprovalDate        *time.Time       `json:"approval_date"`
	PaymentDate         *time.Time       `json:"payment_date"`
	PaymentReference    string           `json:"payment_reference"`
	Notes               string           `json:"notes"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

// CommissionRepo handles commission structures and agent commission records.
type CommissionRepo struct {
	db *pgxpool.Pool
}

func NewCommissionRepo(db *pgxpool.Pool) *CommissionRepo {
	return &CommissionRepo{db: db}
}

// ── Commission Structures ─────────────────────────────────────────────────────

func (r *CommissionRepo) ListStructures(ctx context.Context, companyID uuid.UUID) ([]CommissionStructure, error) {
	const q = `
		SELECT id, company_id, structure_name, commission_type, applicable_to,
		       effective_from, effective_to, rules, created_at
		FROM commission_structures
		WHERE company_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, q, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var structs []CommissionStructure
	for rows.Next() {
		var s CommissionStructure
		if err := rows.Scan(
			&s.ID, &s.CompanyID, &s.Name, &s.Type, &s.ApplicableTo,
			&s.EffectiveFrom, &s.EffectiveTo, &s.Rules, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		structs = append(structs, s)
	}
	return structs, nil
}

func (r *CommissionRepo) CreateStructure(ctx context.Context, s *CommissionStructure) error {
	s.ID = uuid.New()
	if s.Rules == nil {
		s.Rules = json.RawMessage(`{}`)
	}
	const q = `
		INSERT INTO commission_structures
			(id, company_id, structure_name, commission_type, applicable_to,
			 effective_from, effective_to, rules)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING created_at
	`
	return r.db.QueryRow(ctx, q,
		s.ID, s.CompanyID, s.Name, s.Type, s.ApplicableTo,
		s.EffectiveFrom, s.EffectiveTo, s.Rules,
	).Scan(&s.CreatedAt)
}

func (r *CommissionRepo) UpdateStructure(ctx context.Context, s *CommissionStructure) error {
	const q = `
		UPDATE commission_structures
		SET structure_name=$1, commission_type=$2, applicable_to=$3,
		    effective_from=$4, effective_to=$5, rules=$6
		WHERE id=$7 AND company_id=$8
	`
	_, err := r.db.Exec(ctx, q,
		s.Name, s.Type, s.ApplicableTo,
		s.EffectiveFrom, s.EffectiveTo, s.Rules,
		s.ID, s.CompanyID,
	)
	return err
}

func (r *CommissionRepo) DeleteStructure(ctx context.Context, id, companyID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM commission_structures WHERE id=$1 AND company_id=$2`, id, companyID,
	)
	return err
}

// ── Agent Commissions ─────────────────────────────────────────────────────────

func (r *CommissionRepo) ListAgentCommissions(ctx context.Context, companyID uuid.UUID, agentID *uuid.UUID, status string, page, limit int) ([]AgentCommission, int, error) {
	if limit == 0 {
		limit = 50
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	args := []any{companyID}
	conds := []string{"ac.company_id = $1"}
	n := 2

	if agentID != nil {
		conds = append(conds, fmt.Sprintf("ac.agent_id = $%d", n))
		args = append(args, *agentID)
		n++
	}
	if status != "" {
		conds = append(conds, fmt.Sprintf("ac.status = $%d", n))
		args = append(args, status)
		n++
	}

	where := "WHERE " + joinAnd(conds)
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM agent_commissions ac "+where, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	q := fmt.Sprintf(`
		SELECT ac.id, ac.company_id, ac.agent_id, COALESCE(u.name,''),
		       ac.commission_period_start, ac.commission_period_end,
		       ac.commission_structure_id,
		       ac.deals_count, ac.deals_revenue, ac.leases_count, ac.leases_revenue,
		       ac.total_commission, ac.status,
		       ac.approval_date, ac.payment_date, ac.payment_reference,
		       COALESCE(ac.notes,''), ac.created_at, ac.updated_at
		FROM agent_commissions ac
		LEFT JOIN users u ON u.id = ac.agent_id
		%s
		ORDER BY ac.commission_period_start DESC, ac.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, n, n+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []AgentCommission
	for rows.Next() {
		var c AgentCommission
		if err := rows.Scan(
			&c.ID, &c.CompanyID, &c.AgentID, &c.AgentName,
			&c.PeriodStart, &c.PeriodEnd, &c.StructureID,
			&c.DealsCount, &c.DealsRevenue, &c.LeasesCount, &c.LeasesRevenue,
			&c.TotalCommission, &c.Status,
			&c.ApprovalDate, &c.PaymentDate, &c.PaymentReference,
			&c.Notes, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	return list, total, nil
}

func (r *CommissionRepo) CreateAgentCommission(ctx context.Context, c *AgentCommission) error {
	c.ID = uuid.New()
	if c.Status == "" {
		c.Status = CommissionPending
	}
	const q = `
		INSERT INTO agent_commissions
			(id, company_id, agent_id, commission_period_start, commission_period_end,
			 commission_structure_id, deals_count, deals_revenue, leases_count, leases_revenue,
			 total_commission, status, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRow(ctx, q,
		c.ID, c.CompanyID, c.AgentID, c.PeriodStart, c.PeriodEnd,
		c.StructureID, c.DealsCount, c.DealsRevenue, c.LeasesCount, c.LeasesRevenue,
		c.TotalCommission, c.Status, c.Notes,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *CommissionRepo) UpdateCommissionStatus(ctx context.Context, id uuid.UUID, status CommissionStatus, paymentRef string) error {
	var paymentDate *time.Time
	if status == CommissionPaid {
		now := time.Now()
		paymentDate = &now
	}
	var approvalDate *time.Time
	if status == CommissionApproved {
		now := time.Now()
		approvalDate = &now
	}
	_, err := r.db.Exec(ctx, `
		UPDATE agent_commissions
		SET status=$1, payment_reference=$2, payment_date=$3, approval_date=$4, updated_at=NOW()
		WHERE id=$5
	`, status, paymentRef, paymentDate, approvalDate, id)
	return err
}

func (r *CommissionRepo) UpdateCommissionAmount(ctx context.Context, id uuid.UUID, total float64, notes string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE agent_commissions SET total_commission=$1, notes=$2, updated_at=NOW() WHERE id=$3`,
		total, notes, id,
	)
	return err
}

// CalculateForPeriod computes commission totals for an agent over a date range
// by querying their won deals and signed leases.
func (r *CommissionRepo) CalculateForPeriod(ctx context.Context, agentID uuid.UUID, from, to time.Time) (dealsCount int, dealsRevenue float64, leasesCount int, leasesRevenue float64, err error) {
	// Won deals in period
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount),0)
		FROM deals
		WHERE owner_id = $1 AND stage = 'won'
		  AND updated_at >= $2 AND updated_at <= $3
	`, agentID, from, to).Scan(&dealsCount, &dealsRevenue)
	if err != nil {
		return
	}

	// Active leases signed in period
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(rent_amount),0)
		FROM leases
		WHERE agent_id = $1
		  AND start_date >= $2 AND start_date <= $3
	`, agentID, from, to).Scan(&leasesCount, &leasesRevenue)
	return
}
