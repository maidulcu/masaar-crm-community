package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type StatsRepo struct {
	db *pgxpool.Pool
}

func NewStatsRepo(db *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{db: db}
}

// Overview returns a single-row aggregate of key CRM metrics.
// Bolt performance optimization: Uses CTEs and SQL conditional aggregation FILTER (WHERE ...)
// to scan leads and deals tables once each, reducing total subquery executions from 8 to 4
// and speeding up main dashboard overview query performance.
func (r *StatsRepo) Overview(ctx context.Context) (*domain.Stats, error) {
	const q = `
		WITH lead_stats AS (
			SELECT
				COUNT(*) FILTER (WHERE stage NOT IN ('won','lost')) AS active_leads,
				COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '7 days') AS new_leads_week
			FROM leads
			WHERE company_id = $1 AND deleted_at IS NULL
		),
		deal_stats AS (
			SELECT
				COUNT(*) FILTER (WHERE stage = 'open') AS open_deals,
				COALESCE(SUM(amount) FILTER (WHERE stage = 'open'), 0) AS open_deals_value,
				COUNT(*) FILTER (WHERE stage = 'won') AS won_deals,
				COALESCE(SUM(amount) FILTER (WHERE stage = 'won'), 0) AS won_deals_value
			FROM deals
			WHERE company_id = $1
		)
		SELECT
			(SELECT COUNT(*) FROM contacts WHERE company_id = $1) AS total_contacts,
			COALESCE((SELECT active_leads FROM lead_stats), 0) AS active_leads,
			COALESCE((SELECT new_leads_week FROM lead_stats), 0) AS new_leads_week,
			(SELECT COUNT(*) FROM whatsapp_threads WHERE company_id = $1 AND thread_status = 'open') AS open_threads,
			COALESCE((SELECT open_deals FROM deal_stats), 0) AS open_deals,
			COALESCE((SELECT open_deals_value FROM deal_stats), 0) AS open_deals_value,
			COALESCE((SELECT won_deals FROM deal_stats), 0) AS won_deals,
			COALESCE((SELECT won_deals_value FROM deal_stats), 0) AS won_deals_value
	`
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	s := &domain.Stats{}
	err = r.db.QueryRow(ctx, q, cid).Scan(
		&s.TotalContacts,
		&s.ActiveLeads,
		&s.NewLeadsWeek,
		&s.OpenThreads,
		&s.OpenDeals,
		&s.OpenDealsValue,
		&s.WonDeals,
		&s.WonDealsValue,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}
