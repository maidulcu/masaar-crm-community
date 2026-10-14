package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// ComplianceRepo finds licences and permits that are about to expire and records which
// expiry alerts were sent.
type ComplianceRepo struct {
	db *pgxpool.Pool
}

func NewComplianceRepo(db *pgxpool.Pool) *ComplianceRepo { return &ComplianceRepo{db: db} }

// Expiring is something with an expiry date that people should be told about.
type Expiring struct {
	EntityType string // "listing_permit" | "brn" | "trade_license"
	EntityID   uuid.UUID
	Label      string // listing title, broker name, company name
	Expiry     time.Time
	// Notify lists who to tell (besides the company's admins).
	Notify []uuid.UUID
}

// ListExpiring returns the company's licences and permits that expire on or before
// `until`: permits of published listings, BRNs of active users, and the trade licence.
func (r *ComplianceRepo) ListExpiring(ctx context.Context, until time.Time) ([]Expiring, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT 'listing_permit', l.id, l.title, l.permit_expiry, COALESCE(l.assigned_to, l.created_by)
		  FROM listings l
		 WHERE l.company_id = $1 AND l.status = 'published'
		   AND l.permit_number <> '' AND l.permit_expiry IS NOT NULL AND l.permit_expiry <= $2
		UNION ALL
		SELECT 'brn', u.id, u.name, u.brn_expiry, u.id
		  FROM users u
		 WHERE u.company_id = $1 AND u.is_active AND u.brn_expiry IS NOT NULL AND u.brn_expiry <= $2
		UNION ALL
		SELECT 'trade_license', s.company_id, s.name, s.trade_license_expiry, NULL::uuid
		  FROM company_settings s
		 WHERE s.company_id = $1 AND s.trade_license_expiry IS NOT NULL AND s.trade_license_expiry <= $2`,
		cid, until)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Expiring, error) {
		var e Expiring
		var who *uuid.UUID
		err := row.Scan(&e.EntityType, &e.EntityID, &e.Label, &e.Expiry, &who)
		if who != nil {
			e.Notify = []uuid.UUID{*who}
		}
		return e, err
	})
}

// ClaimAlert records that the alert for (entity, expiry, daysBefore) is being sent and
// reports whether this call recorded it; only then should the alert go out.
func (r *ComplianceRepo) ClaimAlert(ctx context.Context, entityType string, entityID uuid.UUID, expiry time.Time, daysBefore int) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	tag, err := r.db.Exec(ctx, `
		INSERT INTO compliance_alerts (company_id, entity_type, entity_id, expiry_date, days_before)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`, cid, entityType, entityID, expiry, daysBefore)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// AdminIDs returns the company's active admins.
func (r *ComplianceRepo) AdminIDs(ctx context.Context) ([]uuid.UUID, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT id FROM users WHERE company_id = $1 AND role = 'admin' AND is_active`, cid)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}
