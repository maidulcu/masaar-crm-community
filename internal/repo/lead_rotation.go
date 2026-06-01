package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

// LeadRotationRepo manages round-robin and capacity-based lead assignment settings.
type LeadRotationRepo struct {
	db *pgxpool.Pool
}

func NewLeadRotationRepo(db *pgxpool.Pool) *LeadRotationRepo {
	return &LeadRotationRepo{db: db}
}

// GetSettings returns the rotation settings for a company.
// Returns a default (manual, disabled) struct if no row exists.
func (r *LeadRotationRepo) GetSettings(ctx context.Context, companyID uuid.UUID) (*domain.LeadRotationSettings, error) {
	const q = `
		SELECT id, company_id, mode, enabled, rotation_index, max_per_agent, updated_at
		FROM lead_rotation_settings
		WHERE company_id = $1
	`
	s := &domain.LeadRotationSettings{CompanyID: companyID, Mode: "manual"}
	err := r.db.QueryRow(ctx, q, companyID).Scan(
		&s.ID, &s.CompanyID, &s.Mode, &s.Enabled, &s.RotationIndex, &s.MaxPerAgent, &s.UpdatedAt,
	)
	if err != nil {
		return s, nil // return default, not an error
	}
	return s, nil
}

// SaveSettings upserts the rotation configuration for a company.
func (r *LeadRotationRepo) SaveSettings(ctx context.Context, s *domain.LeadRotationSettings) error {
	const q = `
		INSERT INTO lead_rotation_settings (id, company_id, mode, enabled, rotation_index, max_per_agent, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, NOW())
		ON CONFLICT (company_id) DO UPDATE SET
			mode           = EXCLUDED.mode,
			enabled        = EXCLUDED.enabled,
			rotation_index = EXCLUDED.rotation_index,
			max_per_agent  = EXCLUDED.max_per_agent,
			updated_at     = NOW()
	`
	_, err := r.db.Exec(ctx, q,
		s.CompanyID, s.Mode, s.Enabled, s.RotationIndex, s.MaxPerAgent,
	)
	return err
}

// AdvanceRotationIndex atomically increments the rotation index by 1 for a company
// and returns the NEW index. Uses an UPDATE … RETURNING in a single round trip.
func (r *LeadRotationRepo) AdvanceRotationIndex(ctx context.Context, companyID uuid.UUID) (int, error) {
	var newIndex int
	err := r.db.QueryRow(ctx, `
		UPDATE lead_rotation_settings
		SET rotation_index = rotation_index + 1, updated_at = NOW()
		WHERE company_id = $1
		RETURNING rotation_index
	`, companyID).Scan(&newIndex)
	return newIndex, err
}

// ListActiveAgents returns all active users with role 'agent' or 'admin' for a company,
// sorted deterministically by ID for stable round-robin ordering.
func (r *LeadRotationRepo) ListActiveAgents(ctx context.Context, companyID uuid.UUID) ([]domain.User, error) {
	const q = `
		SELECT id, company_id, name, email, password_hash, role, lang_pref,
		       COALESCE(wa_number, ''), is_active, created_at
		FROM users
		WHERE company_id = $1
		  AND is_active = true
		  AND role IN ('admin', 'agent')
		ORDER BY id ASC
	`
	rows, err := r.db.Query(ctx, q, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.CompanyID, &u.Name, &u.Email, &u.PasswordHash,
			&u.Role, &u.LangPref, &u.WANumber, &u.IsActive, &u.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// LeadCountByAgent returns the number of open (non-deleted, non-won/lost) leads
// assigned to each agent — used for capacity-based routing.
// pgx accepts a []uuid.UUID slice natively for ANY($1::uuid[]).
func (r *LeadRotationRepo) LeadCountByAgent(ctx context.Context, agentIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	result := make(map[uuid.UUID]int, len(agentIDs))
	if len(agentIDs) == 0 {
		return result, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT assigned_to, COUNT(*)::int
		FROM leads
		WHERE assigned_to = ANY($1::uuid[])
		  AND deleted_at IS NULL
		  AND stage NOT IN ('won','lost')
		GROUP BY assigned_to
	`, agentIDs)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var cnt int
		if err := rows.Scan(&id, &cnt); err != nil {
			return result, err
		}
		result[id] = cnt
	}
	return result, nil
}
