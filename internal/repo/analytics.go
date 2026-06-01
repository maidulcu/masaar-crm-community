package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

type AnalyticsRepository struct {
	conn *pgxpool.Pool
	rdb  *redis.Client
}

func NewAnalyticsRepository(conn *pgxpool.Pool, rdb *redis.Client) *AnalyticsRepository {
	return &AnalyticsRepository{conn: conn, rdb: rdb}
}

const analyticsCacheTTL = 30 * time.Minute

func (r *AnalyticsRepository) cached(ctx context.Context, key string, dest interface{}, compute func() error) error {
	if r.rdb == nil {
		return compute()
	}
	cached, err := r.rdb.Get(ctx, key).Result()
	if err == nil {
		return json.Unmarshal([]byte(cached), dest)
	}
	if err := compute(); err != nil {
		return err
	}
	data, marshalErr := json.Marshal(dest)
	if marshalErr == nil {
		r.rdb.Set(ctx, key, string(data), analyticsCacheTTL)
	}
	return nil
}

func (r *AnalyticsRepository) GetTenantAnalytics(ctx context.Context, companyID uuid.UUID) (*domain.TenantAnalytics, error) {
	key := fmt.Sprintf("analytics:tenant:%s", companyID)
	result := &domain.TenantAnalytics{}
	if err := r.cached(ctx, key, result, func() error {
		return r.computeTenantAnalytics(ctx, companyID, result)
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AnalyticsRepository) computeTenantAnalytics(ctx context.Context, companyID uuid.UUID, out *domain.TenantAnalytics) error {
	query := `
		SELECT
			(SELECT COUNT(*) FROM tenants WHERE company_id = $1 AND status = 'active'),
			(SELECT COUNT(*) FROM tenants WHERE company_id = $1 AND status != 'active'),
			(SELECT COUNT(*) FROM tenants WHERE company_id = $1),
			(SELECT COUNT(*) FROM rental_properties rp WHERE rp.company_id = $1 AND rp.occupancy_status = 'vacant'),
			(SELECT COUNT(*) FROM rental_properties rp WHERE rp.company_id = $1 AND rp.occupancy_status = 'occupied'),
			(SELECT COUNT(*) FROM leases l WHERE l.company_id = $1 AND l.status = 'active'),
			(SELECT COUNT(*) FROM payments WHERE company_id = $1 AND status = 'overdue'),
			COALESCE((SELECT SUM(amount) FROM payments WHERE company_id = $1 AND status = 'overdue'), 0),
			COALESCE((SELECT AVG(monthly_rent) FROM leases WHERE company_id = $1 AND status = 'active'), 0),
			COALESCE((SELECT SUM(monthly_rent) FROM leases WHERE company_id = $1 AND status = 'active'), 0),
			(SELECT COUNT(*) FROM leases WHERE company_id = $1 AND end_date BETWEEN NOW() AND NOW() + INTERVAL '60 days' AND status = 'active')
	`
	var activeLeases int
	err := r.conn.QueryRow(ctx, query, companyID).Scan(
		&out.ActiveTenants, &out.InactiveTenants, &out.TotalTenants,
		&out.VacantUnits, &out.OccupiedUnits, &activeLeases,
		&out.OverduePayments, &out.OverdueDuesAmount, &out.AverageRentPerUnit,
		&out.TotalMonthlyRevenue, &out.UpcomingRenewals,
	)
	if err != nil {
		return err
	}

	totalUnits := out.VacantUnits + out.OccupiedUnits
	if totalUnits > 0 {
		out.OccupancyRate = float64(out.OccupiedUnits) / float64(totalUnits) * 100
	}

	var collected, totalDue float64
	r.conn.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN status = 'received' THEN amount ELSE 0 END), 0),
		       COALESCE(SUM(amount), 0)
		FROM payments WHERE company_id = $1 AND due_date <= NOW()
	`, companyID).Scan(&collected, &totalDue)
	if totalDue > 0 {
		out.CollectionRate = (collected / totalDue) * 100
	}

	var churned int
	r.conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM leases
		WHERE company_id = $1 AND status = 'terminated' AND termination_date >= NOW() - INTERVAL '12 months'
	`, companyID).Scan(&churned)
	if out.ActiveTenants > 0 {
		out.TenantChurnRate = (float64(churned) / float64(out.ActiveTenants)) * 100
	}

	return nil
}

func (r *AnalyticsRepository) GetPropertyAnalytics(ctx context.Context, companyID uuid.UUID, propertyID uuid.UUID) (*domain.PropertyAnalytics, error) {
	key := fmt.Sprintf("analytics:property:%s:%s", companyID, propertyID)
	result := &domain.PropertyAnalytics{}
	if err := r.cached(ctx, key, result, func() error {
		return r.computePropertyAnalytics(ctx, companyID, propertyID, result)
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AnalyticsRepository) computePropertyAnalytics(ctx context.Context, companyID uuid.UUID, propertyID uuid.UUID, out *domain.PropertyAnalytics) error {
	query := `
		SELECT
			rp.id, rp.name, rp.property_type, rp.area, rp.units_count,
			COALESCE((SELECT COUNT(*) FROM leases WHERE property_id = rp.id AND status = 'active'), 0),
			(rp.units_count - COALESCE((SELECT COUNT(*) FROM leases WHERE property_id = rp.id AND status = 'active'), 0)),
			COALESCE((SELECT SUM(l.monthly_rent) FROM leases l WHERE l.property_id = rp.id AND l.status = 'active'), 0),
			(SELECT COUNT(*) FROM maintenance_tasks WHERE property_id = rp.id AND status IN ('pending', 'scheduled')),
			(SELECT COUNT(*) FROM leases WHERE property_id = rp.id AND status = 'active'),
			(SELECT COUNT(*) FROM leases WHERE property_id = rp.id AND end_date BETWEEN NOW() AND NOW() + INTERVAL '60 days' AND status = 'active')
		FROM rental_properties rp
		WHERE rp.company_id = $1 AND rp.id = $2
	`
	var occupied, vacant, activeLs, expiringLs, pendingMaint int
	var monthlyRev float64

	err := r.conn.QueryRow(ctx, query, companyID, propertyID).Scan(
		&out.PropertyID, &out.PropertyName, &out.PropertyType, &out.Area, &out.TotalUnits,
		&occupied, &vacant, &monthlyRev, &pendingMaint, &activeLs, &expiringLs,
	)
	if err != nil {
		return err
	}

	out.OccupiedUnits = occupied
	out.VacantUnits = vacant
	out.MonthlyRevenue = monthlyRev
	out.MaintenanceNeeded = pendingMaint
	out.ActiveLeases = activeLs
	out.ExpiringLeases = expiringLs
	if out.TotalUnits > 0 {
		out.OccupancyRate = float64(occupied) / float64(out.TotalUnits) * 100
	}

	r.conn.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM expenses
		WHERE company_id = $1 AND property_id = $2 AND expense_date >= NOW() - INTERVAL '30 days'
	`, companyID, propertyID).Scan(&out.OperatingExpenses)
	out.NetOperatingIncome = monthlyRev - out.OperatingExpenses

	return nil
}

func (r *AnalyticsRepository) ListPropertiesAnalytics(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.PropertyAnalytics, int, error) {
	key := fmt.Sprintf("analytics:properties:%s:%d:%d", companyID, limit, offset)
	type listResult struct {
		Items []domain.PropertyAnalytics `json:"items"`
		Total int                        `json:"total"`
	}
	var cached listResult
	if r.rdb != nil {
		if data, err := r.rdb.Get(ctx, key).Result(); err == nil {
			if err := json.Unmarshal([]byte(data), &cached); err == nil {
				return cached.Items, cached.Total, nil
			}
		}
	}
	items, total, err := r.computeListPropertiesAnalytics(ctx, companyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if r.rdb != nil {
		if data, marshalErr := json.Marshal(listResult{Items: items, Total: total}); marshalErr == nil {
			r.rdb.Set(ctx, key, string(data), analyticsCacheTTL)
		}
	}
	return items, total, nil
}

func (r *AnalyticsRepository) computeListPropertiesAnalytics(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.PropertyAnalytics, int, error) {
	countQuery := `SELECT COUNT(*) FROM rental_properties WHERE company_id = $1`
	var total int
	err := r.conn.QueryRow(ctx, countQuery, companyID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			rp.id,
			rp.name,
			rp.property_type,
			rp.area,
			rp.units_count,
			COALESCE((SELECT COUNT(*) FROM leases WHERE property_id = rp.id AND status = 'active'), 0) as occupied,
			(rp.units_count - COALESCE((SELECT COUNT(*) FROM leases WHERE property_id = rp.id AND status = 'active'), 0)) as vacant,
			COALESCE((SELECT SUM(l.monthly_rent) FROM leases l WHERE l.property_id = rp.id AND l.status = 'active'), 0) as monthly_revenue,
			(SELECT COUNT(*) FROM maintenance_tasks WHERE property_id = rp.id AND status IN ('pending', 'scheduled')) as pending_maintenance,
			(SELECT COUNT(*) FROM leases WHERE property_id = rp.id AND status = 'active') as active_leases,
			(SELECT COUNT(*) FROM leases WHERE property_id = rp.id AND end_date BETWEEN NOW() AND NOW() + INTERVAL '60 days' AND status = 'active') as expiring_leases
		FROM rental_properties rp
		WHERE rp.company_id = $1
		ORDER BY rp.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.conn.Query(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var properties []domain.PropertyAnalytics
	for rows.Next() {
		var p domain.PropertyAnalytics
		var occupied, vacant, activeLs, expiringLs, pendingMaint int
		var monthlyRev float64

		err := rows.Scan(
			&p.PropertyID,
			&p.PropertyName,
			&p.PropertyType,
			&p.Area,
			&p.TotalUnits,
			&occupied,
			&vacant,
			&monthlyRev,
			&pendingMaint,
			&activeLs,
			&expiringLs,
		)
		if err != nil {
			return nil, 0, err
		}

		p.OccupiedUnits = occupied
		p.VacantUnits = vacant
		p.MonthlyRevenue = monthlyRev
		p.MaintenanceNeeded = pendingMaint
		p.ActiveLeases = activeLs
		p.ExpiringLeases = expiringLs

		if p.TotalUnits > 0 {
			p.OccupancyRate = float64(occupied) / float64(p.TotalUnits) * 100
		}

		properties = append(properties, p)
	}

	return properties, total, rows.Err()
}

func (r *AnalyticsRepository) GetTenantPerformance(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (*domain.TenantPerformanceMetrics, error) {
	query := `
		SELECT
			t.id,
			COALESCE(t.full_name_en, t.full_name_ar) as name,
			t.status,
			(SELECT COUNT(*) FROM leases WHERE tenant_id = t.id) as rental_history,
			(SELECT COUNT(*) FROM leases WHERE tenant_id = t.id AND status = 'terminated') as terminated_count
		FROM tenants t
		WHERE t.company_id = $1 AND t.id = $2
	`

	var tenantID2 uuid.UUID
	var name, status string
	var rentalHistory, terminatedCount int

	err := r.conn.QueryRow(ctx, query, companyID, tenantID).Scan(
		&tenantID2,
		&name,
		&status,
		&rentalHistory,
		&terminatedCount,
	)
	if err != nil {
		return nil, err
	}

	metrics := &domain.TenantPerformanceMetrics{
		TenantID:      tenantID,
		TenantName:    name,
		RentalHistory: rentalHistory,
		Status:        status,
	}

	// Calculate average stay
	avgStayQuery := `
		SELECT COALESCE(AVG(EXTRACT(DAY FROM (end_date - start_date))), 0)
		FROM leases
		WHERE tenant_id = $1 AND status IN ('terminated', 'expired')
	`
	err = r.conn.QueryRow(ctx, avgStayQuery, tenantID).Scan(&metrics.AverageStay)
	if err != nil {
		return nil, err
	}

	// Calculate payment on-time rate
	paymentQuery := `
		SELECT
			COALESCE(COUNT(*) FILTER (WHERE status = 'received' AND paid_date <= due_date), 0) as on_time,
			COALESCE(COUNT(*), 0) as total
		FROM payments p
		JOIN leases l ON p.lease_id = l.id
		WHERE l.tenant_id = $1 AND p.status = 'received'
	`
	var onTime, totalPayments int
	err = r.conn.QueryRow(ctx, paymentQuery, tenantID).Scan(&onTime, &totalPayments)
	if err != nil {
		return nil, err
	}

	if totalPayments > 0 {
		metrics.PaymentOnTimeRate = (float64(onTime) / float64(totalPayments)) * 100
	}

	// Calculate risk score (0-100)
	// Factors: late payments (40%), disputes (30%), tenure (30%)
	disputeQuery := `SELECT COUNT(*) FROM leads WHERE contact_id = (SELECT id FROM contacts WHERE email = (SELECT email FROM tenants WHERE id = $1)) AND stage = 'lost'`
	var disputes int
	err = r.conn.QueryRow(ctx, disputeQuery, tenantID).Scan(&disputes)
	if err == nil {
		metrics.DisputeCount = disputes
	}

	latePaymentScore := 0
	if totalPayments > 0 {
		latePayments := totalPayments - onTime
		latePaymentScore = int((float64(latePayments) / float64(totalPayments)) * 40)
	}

	disputeScore := disputes * 10
	if disputeScore > 30 {
		disputeScore = 30
	}

	tenureScore := 0
	if rentalHistory > 5 {
		tenureScore = 0
	} else if rentalHistory > 2 {
		tenureScore = 15
	} else {
		tenureScore = 30
	}

	metrics.RiskScore = latePaymentScore + disputeScore + tenureScore

	return metrics, nil
}

func (r *AnalyticsRepository) ListTenantsPerformance(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.TenantPerformanceMetrics, int, error) {
	countQuery := `SELECT COUNT(*) FROM tenants WHERE company_id = $1`
	var total int
	err := r.conn.QueryRow(ctx, countQuery, companyID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			t.id,
			COALESCE(t.full_name_en, t.full_name_ar) as name,
			t.status,
			(SELECT COUNT(*) FROM leases WHERE tenant_id = t.id) as rental_history,
			(SELECT COUNT(*) FROM leases WHERE tenant_id = t.id AND status = 'terminated') as terminated_count
		FROM tenants t
		WHERE t.company_id = $1
		ORDER BY t.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.conn.Query(ctx, query, companyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var metrics []domain.TenantPerformanceMetrics
	for rows.Next() {
		var m domain.TenantPerformanceMetrics
		var status string
		var terminatedCount int

		err := rows.Scan(
			&m.TenantID,
			&m.TenantName,
			&status,
			&m.RentalHistory,
			&terminatedCount,
		)
		if err != nil {
			return nil, 0, err
		}

		m.Status = status
		metrics = append(metrics, m)
	}

	return metrics, total, rows.Err()
}

func (r *AnalyticsRepository) GetFinancialAnalytics(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.FinancialAnalytics, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'received' THEN amount ELSE 0 END), 0) as rent_collected,
			COALESCE(SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END), 0) as rent_pending,
			COALESCE(SUM(CASE WHEN status = 'overdue' THEN amount ELSE 0 END), 0) as rent_overdue,
			COALESCE(SUM(amount) FILTER (WHERE category_type = 'utilities'), 0) as utilities_expense,
			COALESCE(SUM(amount) FILTER (WHERE category_type IN ('plumbing', 'electrical', 'hvac', 'flooring', 'painting', 'structural')), 0) as maintenance_expense,
			COALESCE(SUM(amount) FILTER (WHERE category_type NOT IN ('utilities', 'plumbing', 'electrical', 'hvac', 'flooring', 'painting', 'structural')), 0) as other_expense
		FROM (
			SELECT amount, NULL::VARCHAR as category_type, status FROM payments WHERE company_id = $1 AND due_date BETWEEN $2 AND $3
			UNION ALL
			SELECT amount, category_type, NULL::VARCHAR as status FROM expenses WHERE company_id = $1 AND expense_date BETWEEN $2 AND $3
		) combined_data
	`

	var rentCollected, rentPending, rentOverdue, utilitiesExp, maintenanceExp, otherExp float64

	err := r.conn.QueryRow(ctx, query, companyID, startDate, endDate).Scan(
		&rentCollected,
		&rentPending,
		&rentOverdue,
		&utilitiesExp,
		&maintenanceExp,
		&otherExp,
	)
	if err != nil {
		return nil, err
	}

	totalExpenses := utilitiesExp + maintenanceExp + otherExp
	totalRevenue := rentCollected
	netProfit := totalRevenue - totalExpenses

	profitMargin := 0.0
	if totalRevenue > 0 {
		profitMargin = (netProfit / totalRevenue) * 100
	}

	return &domain.FinancialAnalytics{
		Period:              startDate.Format("2006-01-02") + " to " + endDate.Format("2006-01-02"),
		TotalRevenue:        totalRevenue,
		TotalExpenses:       totalExpenses,
		NetProfit:           netProfit,
		ProfitMargin:        profitMargin,
		RentCollected:       rentCollected,
		RentPending:         rentPending,
		UtilitiesExpense:    utilitiesExp,
		MaintenanceExpense:  maintenanceExp,
		OtherExpenses:       otherExp,
	}, nil
}

func (r *AnalyticsRepository) GetMaintenanceAnalytics(ctx context.Context, companyID uuid.UUID) (*domain.MaintenanceAnalytics, error) {
	query := `
		SELECT
			COUNT(*) as total_tasks,
			COUNT(*) FILTER (WHERE status = 'completed') as completed_tasks,
			COUNT(*) FILTER (WHERE status IN ('pending', 'scheduled', 'in_progress')) as pending_tasks,
			COUNT(*) FILTER (WHERE priority = 'high' OR priority = 'urgent') as high_priority,
			COALESCE(AVG(EXTRACT(DAY FROM (completion_date - created_at))) FILTER (WHERE status = 'completed'), 0) as avg_completion_days
		FROM maintenance_tasks
		WHERE company_id = $1
	`

	var analytics domain.MaintenanceAnalytics
	var completedTasks, pendingTasks, avgDays int
	var highPriority int

	err := r.conn.QueryRow(ctx, query, companyID).Scan(
		&analytics.TotalTasks,
		&completedTasks,
		&pendingTasks,
		&avgDays,
		&highPriority,
	)
	if err != nil {
		return nil, err
	}

	analytics.CompletedTasks = completedTasks
	analytics.PendingTasks = pendingTasks
	analytics.HighPriorityTasks = highPriority
	analytics.AvgCompletionDays = float64(avgDays)

	if analytics.TotalTasks > 0 {
		analytics.CompletionRate = (float64(completedTasks) / float64(analytics.TotalTasks)) * 100
	}

	return &analytics, nil
}
