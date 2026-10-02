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
func (r *StatsRepo) Overview(ctx context.Context) (*domain.Stats, error) {
	const q = `
		SELECT
		  (SELECT COUNT(*)                          FROM contacts WHERE company_id = $1)                                           AS total_contacts,
		  (SELECT COUNT(*)                          FROM leads WHERE company_id = $1 AND deleted_at IS NULL AND stage NOT IN ('won','lost'))             AS active_leads,
		  (SELECT COUNT(*)                          FROM leads WHERE company_id = $1 AND deleted_at IS NULL AND created_at >= NOW() - INTERVAL '7 days') AS new_leads_week,
		  (SELECT COUNT(*)                          FROM whatsapp_threads WHERE company_id = $1 AND thread_status = 'open')       AS open_threads,
		  (SELECT COUNT(*)                          FROM deals WHERE company_id = $1 AND stage = 'open')                          AS open_deals,
		  (SELECT COALESCE(SUM(amount), 0)          FROM deals WHERE company_id = $1 AND stage = 'open')                          AS open_deals_value,
		  (SELECT COUNT(*)                          FROM deals WHERE company_id = $1 AND stage = 'won')                           AS won_deals,
		  (SELECT COALESCE(SUM(amount), 0)          FROM deals WHERE company_id = $1 AND stage = 'won')                           AS won_deals_value
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
