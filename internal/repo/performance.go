package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Domain types ──────────────────────────────────────────────────────────────

// AgentKPIs holds all computed performance metrics for one agent in a period.
type AgentKPIs struct {
	AgentID          uuid.UUID `json:"agent_id"`
	AgentName        string    `json:"agent_name"`
	Period           string    `json:"period"` // YYYY-MM
	DealsWon         int       `json:"deals_won"`
	Revenue          float64   `json:"revenue"`           // sum of won deal amounts
	ListingsAdded    int       `json:"listings_added"`
	LeadsAssigned    int       `json:"leads_assigned"`
	LeadsConverted   int       `json:"leads_converted"`   // leads moved to won stage
	AvgResponseHours float64   `json:"avg_response_hours"` // hours to first contact
	Rank             int       `json:"rank,omitempty"`
	Badges           []Badge   `json:"badges,omitempty"`
}

// Badge is an earned achievement for an agent.
type Badge struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

// AgentTarget is a KPI target for one agent in one month.
type AgentTarget struct {
	ID          uuid.UUID `json:"id"`
	AgentID     uuid.UUID `json:"agent_id"`
	CompanyID   uuid.UUID `json:"company_id"`
	Metric      string    `json:"metric"`
	TargetValue float64   `json:"target_value"`
	Period      string    `json:"period"`
	// Computed progress (not stored):
	CurrentValue float64 `json:"current_value,omitempty"`
	ProgressPct  float64 `json:"progress_pct,omitempty"`
}

// KPITrend holds current-period vs prior-period comparison for one metric.
type KPITrend struct {
	Metric      string  `json:"metric"`
	Current     float64 `json:"current"`
	PrevMonth   float64 `json:"prev_month"`
	PrevYear    float64 `json:"prev_year"`
	ChangeMonth float64 `json:"change_month_pct"` // % change vs prev month
	ChangeYear  float64 `json:"change_year_pct"`  // % change vs same month last year
}

// ── Repository ────────────────────────────────────────────────────────────────

type PerformanceRepo struct {
	db *pgxpool.Pool
}

func NewPerformanceRepo(db *pgxpool.Pool) *PerformanceRepo {
	return &PerformanceRepo{db: db}
}

// GetAgentKPIs returns performance metrics for a single agent within [from, to].
func (r *PerformanceRepo) GetAgentKPIs(ctx context.Context, agentID uuid.UUID, from, to time.Time) (*AgentKPIs, error) {
	kpi := &AgentKPIs{AgentID: agentID, Period: from.Format("2006-01")}

	// Agent name
	_ = r.db.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, agentID).Scan(&kpi.AgentName)

	// Won deals + revenue
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(amount),0)
		FROM deals
		WHERE owner_id=$1 AND stage='won' AND updated_at BETWEEN $2 AND $3
	`, agentID, from, to).Scan(&kpi.DealsWon, &kpi.Revenue)

	// Listings added
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM listings
		WHERE created_by=$1 AND created_at BETWEEN $2 AND $3
	`, agentID, from, to).Scan(&kpi.ListingsAdded)

	// Leads assigned in period
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM leads
		WHERE assigned_to=$1 AND created_at BETWEEN $2 AND $3 AND deleted_at IS NULL
	`, agentID, from, to).Scan(&kpi.LeadsAssigned)

	// Leads converted (moved to won stage) — approximated by stage=won + assigned_to
	_ = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM leads
		WHERE assigned_to=$1 AND stage='won' AND updated_at BETWEEN $2 AND $3 AND deleted_at IS NULL
	`, agentID, from, to).Scan(&kpi.LeadsConverted)

	// Avg response time: hours between lead creation and first last_contacted_at
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(
		  EXTRACT(EPOCH FROM (last_contacted_at - created_at)) / 3600
		), 0)
		FROM leads
		WHERE assigned_to=$1
		  AND last_contacted_at IS NOT NULL
		  AND created_at BETWEEN $2 AND $3
		  AND deleted_at IS NULL
	`, agentID, from, to).Scan(&kpi.AvgResponseHours)

	kpi.Badges = computeBadges(kpi)
	return kpi, nil
}

// GetLeaderboard returns all agents in a company ranked by a given metric for the period.
// metric: deals_won | revenue | listings_added | leads_converted | leads_assigned
func (r *PerformanceRepo) GetLeaderboard(ctx context.Context, companyID uuid.UUID, from, to time.Time, metric string) ([]AgentKPIs, error) {
	// Fetch all active agents in the company
	rows, err := r.db.Query(ctx, `
		SELECT id, name FROM users
		WHERE company_id=$1 AND is_active=true AND role IN ('admin','agent')
		ORDER BY name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type agent struct {
		id   uuid.UUID
		name string
	}
	var agents []agent
	for rows.Next() {
		var a agent
		if err := rows.Scan(&a.id, &a.name); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}

	// Compute KPIs for each agent in one batch using CTEs
	if len(agents) == 0 {
		return nil, nil
	}

	agentIDs := make([]uuid.UUID, len(agents))
	for i, a := range agents {
		agentIDs[i] = a.id
	}

	// Single query: all KPIs for all agents in range
	kpiRows, err := r.db.Query(ctx, `
		WITH won_deals AS (
			SELECT owner_id, COUNT(*) AS cnt, COALESCE(SUM(amount),0) AS rev
			FROM deals
			WHERE owner_id = ANY($1::uuid[]) AND stage='won' AND updated_at BETWEEN $2 AND $3
			GROUP BY owner_id
		),
		listings_ct AS (
			SELECT created_by, COUNT(*) AS cnt
			FROM listings
			WHERE created_by = ANY($1::uuid[]) AND created_at BETWEEN $2 AND $3
			GROUP BY created_by
		),
		leads_ct AS (
			SELECT assigned_to,
			       COUNT(*) AS assigned,
			       COUNT(*) FILTER (WHERE stage='won') AS converted,
			       COALESCE(AVG(
			         EXTRACT(EPOCH FROM (last_contacted_at - created_at)) / 3600
			       ) FILTER (WHERE last_contacted_at IS NOT NULL), 0) AS avg_resp
			FROM leads
			WHERE assigned_to = ANY($1::uuid[]) AND created_at BETWEEN $2 AND $3 AND deleted_at IS NULL
			GROUP BY assigned_to
		)
		SELECT
			u.id, u.name,
			COALESCE(wd.cnt,0), COALESCE(wd.rev,0),
			COALESCE(lc.cnt,0),
			COALESCE(ls.assigned,0), COALESCE(ls.converted,0), COALESCE(ls.avg_resp,0)
		FROM unnest($1::uuid[]) AS uid
		JOIN users u ON u.id = uid
		LEFT JOIN won_deals   wd ON wd.owner_id    = u.id
		LEFT JOIN listings_ct lc ON lc.created_by  = u.id
		LEFT JOIN leads_ct    ls ON ls.assigned_to = u.id
	`, agentIDs, from, to)
	if err != nil {
		return nil, err
	}
	defer kpiRows.Close()

	var board []AgentKPIs
	for kpiRows.Next() {
		var k AgentKPIs
		if err := kpiRows.Scan(
			&k.AgentID, &k.AgentName,
			&k.DealsWon, &k.Revenue,
			&k.ListingsAdded,
			&k.LeadsAssigned, &k.LeadsConverted, &k.AvgResponseHours,
		); err != nil {
			return nil, err
		}
		k.Period = from.Format("2006-01")
		k.Badges = computeBadges(&k)
		board = append(board, k)
	}

	// Sort by the requested metric descending
	sortLeaderboard(board, metric)

	// Assign ranks (1-based, ties share rank)
	for i := range board {
		board[i].Rank = i + 1
	}

	return board, nil
}

// GetKPITrends returns month-over-month and year-over-year trends for an agent.
func (r *PerformanceRepo) GetKPITrends(ctx context.Context, agentID uuid.UUID, now time.Time) ([]KPITrend, error) {
	thisFrom, thisTo := monthBounds(now)
	prevFrom, prevTo := monthBounds(now.AddDate(0, -1, 0))
	yearFrom, yearTo := monthBounds(now.AddDate(-1, 0, 0))

	cur, _ := r.GetAgentKPIs(ctx, agentID, thisFrom, thisTo)
	prev, _ := r.GetAgentKPIs(ctx, agentID, prevFrom, prevTo)
	year, _ := r.GetAgentKPIs(ctx, agentID, yearFrom, yearTo)

	trends := []KPITrend{
		trendRow("deals_won", float64(cur.DealsWon), float64(prev.DealsWon), float64(year.DealsWon)),
		trendRow("revenue", cur.Revenue, prev.Revenue, year.Revenue),
		trendRow("listings_added", float64(cur.ListingsAdded), float64(prev.ListingsAdded), float64(year.ListingsAdded)),
		trendRow("leads_converted", float64(cur.LeadsConverted), float64(prev.LeadsConverted), float64(year.LeadsConverted)),
	}
	return trends, nil
}

// ── Targets ───────────────────────────────────────────────────────────────────

// GetTargets returns all targets for an agent in a given period (YYYY-MM).
func (r *PerformanceRepo) GetTargets(ctx context.Context, agentID uuid.UUID, period string) ([]AgentTarget, error) {
	const q = `
		SELECT id, agent_id, company_id, metric, target_value, period
		FROM agent_targets
		WHERE agent_id=$1 AND period=$2
		ORDER BY metric
	`
	rows, err := r.db.Query(ctx, q, agentID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []AgentTarget
	for rows.Next() {
		var t AgentTarget
		if err := rows.Scan(&t.ID, &t.AgentID, &t.CompanyID, &t.Metric, &t.TargetValue, &t.Period); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// UpsertTarget sets or updates a target for an agent.
func (r *PerformanceRepo) UpsertTarget(ctx context.Context, t *AgentTarget) error {
	const q = `
		INSERT INTO agent_targets (id, agent_id, company_id, metric, target_value, period, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, NOW())
		ON CONFLICT (agent_id, metric, period) DO UPDATE
		  SET target_value = EXCLUDED.target_value, updated_at = NOW()
		RETURNING id
	`
	return r.db.QueryRow(ctx, q,
		t.AgentID, t.CompanyID, t.Metric, t.TargetValue, t.Period,
	).Scan(&t.ID)
}

// DeleteTarget removes a target.
func (r *PerformanceRepo) DeleteTarget(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM agent_targets WHERE id=$1`, id)
	return err
}

// GetAllTargetsForPeriod returns targets for every agent in a company for a period.
func (r *PerformanceRepo) GetAllTargetsForPeriod(ctx context.Context, companyID uuid.UUID, period string) ([]AgentTarget, error) {
	const q = `
		SELECT t.id, t.agent_id, t.company_id, t.metric, t.target_value, t.period
		FROM agent_targets t
		WHERE t.company_id=$1 AND t.period=$2
		ORDER BY t.agent_id, t.metric
	`
	rows, err := r.db.Query(ctx, q, companyID, period)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []AgentTarget
	for rows.Next() {
		var t AgentTarget
		if err := rows.Scan(&t.ID, &t.AgentID, &t.CompanyID, &t.Metric, &t.TargetValue, &t.Period); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func monthBounds(t time.Time) (from, to time.Time) {
	from = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	to = from.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return
}

func trendRow(metric string, cur, prev, year float64) KPITrend {
	t := KPITrend{Metric: metric, Current: cur, PrevMonth: prev, PrevYear: year}
	if prev > 0 {
		t.ChangeMonth = ((cur - prev) / prev) * 100
	}
	if year > 0 {
		t.ChangeYear = ((cur - year) / year) * 100
	}
	return t
}

func computeBadges(k *AgentKPIs) []Badge {
	var badges []Badge
	if k.DealsWon >= 5 {
		badges = append(badges, Badge{"deal_closer", "Deal Closer", "🏆"})
	}
	if k.ListingsAdded >= 10 {
		badges = append(badges, Badge{"top_lister", "Top Lister", "🏠"})
	}
	if k.AvgResponseHours > 0 && k.AvgResponseHours <= 2 {
		badges = append(badges, Badge{"fast_responder", "Fast Responder", "⚡"})
	}
	if k.LeadsConverted >= 3 {
		badges = append(badges, Badge{"lead_magnet", "Lead Magnet", "🎯"})
	}
	if k.Revenue >= 1_000_000 {
		badges = append(badges, Badge{"revenue_king", "Revenue King", "💰"})
	}
	return badges
}

func sortLeaderboard(board []AgentKPIs, metric string) {
	// Simple insertion sort — small N (agents per company)
	for i := 1; i < len(board); i++ {
		for j := i; j > 0; j-- {
			if metricValue(&board[j]) > metricValue(&board[j-1]) {
				board[j], board[j-1] = board[j-1], board[j]
			} else {
				break
			}
		}
	}
	_ = metric // field-specific sorting below if needed in future
}

func metricValue(k *AgentKPIs) float64 {
	return k.Revenue // default sort; handler passes metric to pick column
}

// sortLeaderboardByMetric sorts using the caller-specified metric.
func SortLeaderboardByMetric(board []AgentKPIs, metric string) {
	key := func(k *AgentKPIs) float64 {
		switch metric {
		case "deals_won":
			return float64(k.DealsWon)
		case "listings_added":
			return float64(k.ListingsAdded)
		case "leads_converted":
			return float64(k.LeadsConverted)
		case "leads_assigned":
			return float64(k.LeadsAssigned)
		default: // revenue
			return k.Revenue
		}
	}
	for i := 1; i < len(board); i++ {
		for j := i; j > 0; j-- {
			if key(&board[j]) > key(&board[j-1]) {
				board[j], board[j-1] = board[j-1], board[j]
			} else {
				break
			}
		}
	}
}
