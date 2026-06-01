package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type ViewingRepo struct {
	db *pgxpool.Pool
}

func NewViewingRepo(db *pgxpool.Pool) *ViewingRepo {
	return &ViewingRepo{db: db}
}

const viewingCols = `
	v.id, v.listing_id, v.contact_id, v.agent_id, v.lead_id,
	v.scheduled_at, v.duration_min, v.status, v.address, v.notes,
	v.checked_in_at, v.checked_out_at, v.reminder_sent,
	v.created_at, v.updated_at`

func scanViewing(row interface{ Scan(...any) error }, v *domain.Viewing) error {
	return row.Scan(
		&v.ID, &v.ListingID, &v.ContactID, &v.AgentID, &v.LeadID,
		&v.ScheduledAt, &v.DurationMin, &v.Status, &v.Address, &v.Notes,
		&v.CheckedInAt, &v.CheckedOutAt, &v.ReminderSent,
		&v.CreatedAt, &v.UpdatedAt,
	)
}

// Create inserts a new viewing.
func (r *ViewingRepo) Create(ctx context.Context, v *domain.Viewing) error {
	v.ID = uuid.New()
	if v.DurationMin <= 0 {
		v.DurationMin = 30
	}
	if v.Status == "" {
		v.Status = domain.ViewingScheduled
	}
	const q = `
		INSERT INTO viewings
			(id, listing_id, contact_id, agent_id, lead_id,
			 scheduled_at, duration_min, status, address, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRow(ctx, q,
		v.ID, v.ListingID, v.ContactID, v.AgentID, v.LeadID,
		v.ScheduledAt, v.DurationMin, v.Status, v.Address, v.Notes,
	).Scan(&v.CreatedAt, &v.UpdatedAt)
}

// GetByID returns a single viewing with joined contact, listing title, agent name.
func (r *ViewingRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Viewing, error) {
	const q = `
		SELECT ` + viewingCols + `,
		       c.id, c.phone_wa, c.full_name, c.email, c.language, c.lead_score,
		       COALESCE(l.title,''), COALESCE(u.name,'')
		FROM viewings v
		JOIN contacts c ON c.id = v.contact_id
		LEFT JOIN listings l ON l.id = v.listing_id
		LEFT JOIN users    u ON u.id = v.agent_id
		WHERE v.id = $1
	`
	var vw domain.Viewing
	var c domain.Contact
	err := r.db.QueryRow(ctx, q, id).Scan(
		&vw.ID, &vw.ListingID, &vw.ContactID, &vw.AgentID, &vw.LeadID,
		&vw.ScheduledAt, &vw.DurationMin, &vw.Status, &vw.Address, &vw.Notes,
		&vw.CheckedInAt, &vw.CheckedOutAt, &vw.ReminderSent,
		&vw.CreatedAt, &vw.UpdatedAt,
		&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
		&vw.ListingTitle, &vw.AgentName,
	)
	if err != nil {
		return nil, fmt.Errorf("viewing not found: %w", err)
	}
	vw.Contact = &c
	return &vw, nil
}

// ViewingFilter for list queries.
type ViewingFilter struct {
	AgentID   *uuid.UUID
	ContactID *uuid.UUID
	ListingID *uuid.UUID
	Status    string
	From      *time.Time // scheduled_at >= From
	To        *time.Time // scheduled_at <= To
	Page      int
	Limit     int
}

// List returns viewings matching the filter, ordered by scheduled_at ASC.
func (r *ViewingRepo) List(ctx context.Context, f ViewingFilter) ([]domain.Viewing, int, error) {
	if f.Limit == 0 {
		f.Limit = 100
	}
	if f.Page < 1 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.Limit

	args := []any{}
	conds := []string{"1=1"}
	n := 1

	if f.AgentID != nil {
		conds = append(conds, fmt.Sprintf("v.agent_id = $%d", n))
		args = append(args, *f.AgentID)
		n++
	}
	if f.ContactID != nil {
		conds = append(conds, fmt.Sprintf("v.contact_id = $%d", n))
		args = append(args, *f.ContactID)
		n++
	}
	if f.ListingID != nil {
		conds = append(conds, fmt.Sprintf("v.listing_id = $%d", n))
		args = append(args, *f.ListingID)
		n++
	}
	if f.Status != "" {
		conds = append(conds, fmt.Sprintf("v.status = $%d", n))
		args = append(args, f.Status)
		n++
	}
	if f.From != nil {
		conds = append(conds, fmt.Sprintf("v.scheduled_at >= $%d", n))
		args = append(args, *f.From)
		n++
	}
	if f.To != nil {
		conds = append(conds, fmt.Sprintf("v.scheduled_at <= $%d", n))
		args = append(args, *f.To)
		n++
	}

	where := "WHERE " + joinAnd(conds)

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM viewings v "+where, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, offset)
	q := fmt.Sprintf(`
		SELECT `+viewingCols+`,
		       c.id, c.phone_wa, c.full_name, c.email, c.language, c.lead_score,
		       COALESCE(l.title,''), COALESCE(u.name,'')
		FROM viewings v
		JOIN contacts  c ON c.id = v.contact_id
		LEFT JOIN listings l ON l.id = v.listing_id
		LEFT JOIN users    u ON u.id = v.agent_id
		%s
		ORDER BY v.scheduled_at ASC
		LIMIT $%d OFFSET $%d
	`, where, n, n+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var viewings []domain.Viewing
	for rows.Next() {
		var vw domain.Viewing
		var c domain.Contact
		if err := rows.Scan(
			&vw.ID, &vw.ListingID, &vw.ContactID, &vw.AgentID, &vw.LeadID,
			&vw.ScheduledAt, &vw.DurationMin, &vw.Status, &vw.Address, &vw.Notes,
			&vw.CheckedInAt, &vw.CheckedOutAt, &vw.ReminderSent,
			&vw.CreatedAt, &vw.UpdatedAt,
			&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
			&vw.ListingTitle, &vw.AgentName,
		); err != nil {
			return nil, 0, err
		}
		vw.Contact = &c
		viewings = append(viewings, vw)
	}
	return viewings, total, nil
}

// Update replaces mutable fields on a viewing.
func (r *ViewingRepo) Update(ctx context.Context, v *domain.Viewing) error {
	const q = `
		UPDATE viewings
		SET listing_id=$1, agent_id=$2, lead_id=$3,
		    scheduled_at=$4, duration_min=$5, address=$6, notes=$7,
		    updated_at=NOW()
		WHERE id=$8
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, q,
		v.ListingID, v.AgentID, v.LeadID,
		v.ScheduledAt, v.DurationMin, v.Address, v.Notes,
		v.ID,
	).Scan(&v.UpdatedAt)
}

// UpdateStatus changes the viewing status (confirm, check-in, complete, cancel, no-show).
func (r *ViewingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ViewingStatus) error {
	var checkinClause string
	if status == domain.ViewingCheckedIn {
		checkinClause = ", checked_in_at = NOW()"
	} else if status == domain.ViewingCompleted {
		checkinClause = ", checked_out_at = NOW()"
	}
	_, err := r.db.Exec(ctx, fmt.Sprintf(
		`UPDATE viewings SET status=$1%s, updated_at=NOW() WHERE id=$2`,
		checkinClause,
	), status, id)
	return err
}

// Delete hard-deletes a viewing.
func (r *ViewingRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM viewings WHERE id=$1`, id)
	return err
}

// CheckConflict returns true if the agent already has a viewing that overlaps
// the proposed time window [start, start+durationMin). Excludes cancelledviewing.
func (r *ViewingRepo) CheckConflict(ctx context.Context, agentID uuid.UUID, start time.Time, durationMin int, excludeID *uuid.UUID) (bool, error) {
	end := start.Add(time.Duration(durationMin) * time.Minute)
	q := `
		SELECT EXISTS (
			SELECT 1 FROM viewings
			WHERE agent_id = $1
			  AND status NOT IN ('cancelled','no_show','completed')
			  AND scheduled_at < $3
			  AND (scheduled_at + duration_min * interval '1 minute') > $2
	`
	args := []any{agentID, start, end}
	if excludeID != nil {
		q += " AND id <> $4"
		args = append(args, *excludeID)
	}
	q += ")"

	var conflict bool
	err := r.db.QueryRow(ctx, q, args...).Scan(&conflict)
	return conflict, err
}

// DueForReminder returns viewings scheduled within the next `withinMinutes` minutes
// that haven't had a reminder sent yet.
func (r *ViewingRepo) DueForReminder(ctx context.Context, withinMinutes int) ([]domain.Viewing, error) {
	cutoff := time.Now().UTC().Add(time.Duration(withinMinutes) * time.Minute)
	const q = `
		SELECT ` + viewingCols + `,
		       c.id, c.phone_wa, c.full_name, c.email, c.language, c.lead_score,
		       COALESCE(l.title,''), COALESCE(u.name,'')
		FROM viewings v
		JOIN contacts  c ON c.id = v.contact_id
		LEFT JOIN listings l ON l.id = v.listing_id
		LEFT JOIN users    u ON u.id = v.agent_id
		WHERE v.reminder_sent = false
		  AND v.status IN ('scheduled','confirmed')
		  AND v.scheduled_at BETWEEN NOW() AND $1
		ORDER BY v.scheduled_at ASC
	`
	rows, err := r.db.Query(ctx, q, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var viewings []domain.Viewing
	for rows.Next() {
		var vw domain.Viewing
		var c domain.Contact
		if err := rows.Scan(
			&vw.ID, &vw.ListingID, &vw.ContactID, &vw.AgentID, &vw.LeadID,
			&vw.ScheduledAt, &vw.DurationMin, &vw.Status, &vw.Address, &vw.Notes,
			&vw.CheckedInAt, &vw.CheckedOutAt, &vw.ReminderSent,
			&vw.CreatedAt, &vw.UpdatedAt,
			&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
			&vw.ListingTitle, &vw.AgentName,
		); err != nil {
			return nil, err
		}
		vw.Contact = &c
		viewings = append(viewings, vw)
	}
	return viewings, nil
}

// MarkReminderSent sets reminder_sent = true for a viewing.
func (r *ViewingRepo) MarkReminderSent(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE viewings SET reminder_sent=true WHERE id=$1`, id)
	return err
}
