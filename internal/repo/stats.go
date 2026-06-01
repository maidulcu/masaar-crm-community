package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
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
		  (SELECT COUNT(*)                          FROM contacts)                                           AS total_contacts,
		  (SELECT COUNT(*)                          FROM leads WHERE stage NOT IN ('won','lost'))             AS active_leads,
		  (SELECT COUNT(*)                          FROM leads WHERE created_at >= NOW() - INTERVAL '7 days') AS new_leads_week,
		  (SELECT COUNT(*)                          FROM whatsapp_threads WHERE thread_status = 'open')       AS open_threads,
		  (SELECT COUNT(*)                          FROM deals WHERE stage = 'open')                          AS open_deals,
		  (SELECT COALESCE(SUM(amount), 0)          FROM deals WHERE stage = 'open')                          AS open_deals_value,
		  (SELECT COUNT(*)                          FROM deals WHERE stage = 'won')                           AS won_deals,
		  (SELECT COALESCE(SUM(amount), 0)          FROM deals WHERE stage = 'won')                           AS won_deals_value
	`
	s := &domain.Stats{}
	err := r.db.QueryRow(ctx, q).Scan(
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
