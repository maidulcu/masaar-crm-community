package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// TaskRepo stores tasks and follow-ups.
type TaskRepo struct {
	db *pgxpool.Pool
}

func NewTaskRepo(db *pgxpool.Pool) *TaskRepo { return &TaskRepo{db: db} }

// relatedTables maps a task's related_type to the table holding that record.
var relatedTables = map[string]string{
	"lead": "leads", "contact": "contacts", "deal": "deals",
	"listing": "listings", "viewing": "viewings", "offer": "offers",
}

// ValidRelatedType reports whether t can be linked to a task.
func ValidRelatedType(t string) bool {
	_, ok := relatedTables[t]
	return ok
}

const taskCols = `t.id, t.title, t.description, t.type, t.priority, t.status, t.due_at, t.assigned_to,
	t.related_type, t.related_id, t.created_by, t.completed_at, t.created_at, t.updated_at,
	COALESCE(u.name, ''),
	COALESCE(CASE t.related_type
		WHEN 'contact' THEN (SELECT full_name FROM contacts WHERE id = t.related_id)
		WHEN 'lead'    THEN (SELECT c.full_name FROM leads l JOIN contacts c ON c.id = l.contact_id WHERE l.id = t.related_id)
		WHEN 'deal'    THEN (SELECT title FROM deals WHERE id = t.related_id)
		WHEN 'listing' THEN (SELECT title FROM listings WHERE id = t.related_id)
		WHEN 'viewing' THEN (SELECT c.full_name FROM viewings v JOIN contacts c ON c.id = v.contact_id WHERE v.id = t.related_id)
		WHEN 'offer'   THEN (SELECT c.full_name FROM offers o JOIN contacts c ON c.id = o.contact_id WHERE o.id = t.related_id)
	END, ''),
	(t.status = 'open' AND t.due_at IS NOT NULL AND t.due_at < NOW())`

const taskFrom = ` FROM tasks t LEFT JOIN users u ON u.id = t.assigned_to AND u.company_id = t.company_id`

func scanTask(row pgx.Row, t *domain.Task) error {
	return row.Scan(&t.ID, &t.Title, &t.Description, &t.Type, &t.Priority, &t.Status, &t.DueAt, &t.AssignedTo,
		&t.RelatedType, &t.RelatedID, &t.CreatedBy, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
		&t.AssigneeName, &t.RelatedLabel, &t.Overdue)
}

// checkRefs verifies that the assignee and the related record belong to the company.
func (r *TaskRepo) checkRefs(ctx context.Context, cid uuid.UUID, t *domain.Task) error {
	if t.AssignedTo != nil {
		var ok bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND company_id = $2)`,
			*t.AssignedTo, cid).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return ErrForeignReference
		}
	}
	if t.RelatedType != nil {
		table, ok := relatedTables[*t.RelatedType]
		if !ok || t.RelatedID == nil {
			return ErrForeignReference
		}
		var exists bool
		// table comes from the fixed map above, never from input.
		if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+table+` WHERE id = $1 AND company_id = $2)`,
			*t.RelatedID, cid).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrForeignReference
		}
	}
	return nil
}

// Create stores a new open task.
func (r *TaskRepo) Create(ctx context.Context, t *domain.Task) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if err := r.checkRefs(ctx, cid, t); err != nil {
		return err
	}
	t.ID = uuid.New()
	t.Status = "open"
	return r.db.QueryRow(ctx, `
		INSERT INTO tasks (id, company_id, title, description, type, priority, status, due_at, assigned_to,
		                   related_type, related_id, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,'open',$7,$8,$9,$10,$11)
		RETURNING created_at, updated_at`,
		t.ID, cid, t.Title, t.Description, t.Type, t.Priority, t.DueAt, t.AssignedTo,
		t.RelatedType, t.RelatedID, t.CreatedBy,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
}

// Get returns one task.
func (r *TaskRepo) Get(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	t := &domain.Task{}
	if err := scanTask(r.db.QueryRow(ctx, `SELECT `+taskCols+taskFrom+` WHERE t.id = $1 AND t.company_id = $2`, id, cid), t); err != nil {
		return nil, err
	}
	return t, nil
}

// List returns tasks matching f: open tasks by due date first (no due date last), then
// closed tasks, newest first.
func (r *TaskRepo) List(ctx context.Context, f domain.TaskFilter) ([]domain.Task, int, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, 0, err
	}
	where := []string{"t.company_id = $1"}
	args := []any{cid}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.AssignedTo != nil {
		add("t.assigned_to = $%d", *f.AssignedTo)
	}
	if f.Status != "" {
		add("t.status = $%d", f.Status)
	}
	if f.RelatedType != "" {
		add("t.related_type = $%d", f.RelatedType)
	}
	if f.RelatedID != nil {
		add("t.related_id = $%d", *f.RelatedID)
	}
	if f.DueBefore != nil {
		add("t.due_at < $%d", *f.DueBefore)
	}
	if f.OverdueOnly {
		where = append(where, "t.status = 'open' AND t.due_at < NOW()")
	}
	cond := strings.Join(where, " AND ")
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks t WHERE `+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args = append(args, limit, f.Offset)
	rows, err := r.db.Query(ctx, `SELECT `+taskCols+taskFrom+` WHERE `+cond+`
		ORDER BY (t.status = 'open') DESC, t.due_at ASC NULLS LAST, t.created_at DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Task
	for rows.Next() {
		var t domain.Task
		if err := scanTask(rows, &t); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

// Update changes a task's editable fields. Moving the due date re-arms its reminder.
func (r *TaskRepo) Update(ctx context.Context, t *domain.Task) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if err := r.checkRefs(ctx, cid, t); err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE tasks SET title = $1, description = $2, type = $3, priority = $4, due_at = $5,
		       assigned_to = $6, related_type = $7, related_id = $8,
		       reminder_sent_at = CASE WHEN due_at IS DISTINCT FROM $5 OR assigned_to IS DISTINCT FROM $6
		                               THEN NULL ELSE reminder_sent_at END,
		       updated_at = NOW()
		 WHERE id = $9 AND company_id = $10`,
		t.Title, t.Description, t.Type, t.Priority, t.DueAt, t.AssignedTo, t.RelatedType, t.RelatedID, t.ID, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// SetStatus marks a task open, done or cancelled.
func (r *TaskRepo) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if status != "open" && status != "done" && status != "cancelled" {
		return errors.New("invalid task status")
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE tasks SET status = $1::text,
		       completed_at = CASE WHEN $1::text = 'done' THEN NOW() ELSE NULL END, updated_at = NOW()
		 WHERE id = $2 AND company_id = $3`, status, id, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Delete removes a task.
func (r *TaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1 AND company_id = $2`, id, cid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ClaimDueReminders marks open, assigned tasks due before `until` whose reminder has not
// been sent as reminded, and returns them. The UPDATE … RETURNING claims each task once.
func (r *TaskRepo) ClaimDueReminders(ctx context.Context, until time.Time) ([]domain.Task, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		UPDATE tasks t SET reminder_sent_at = NOW()
		 WHERE t.company_id = $1 AND t.status = 'open' AND t.assigned_to IS NOT NULL
		   AND t.reminder_sent_at IS NULL AND t.due_at IS NOT NULL AND t.due_at <= $2
		RETURNING t.id, t.title, t.due_at, t.assigned_to`, cid, until)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Task, error) {
		var t domain.Task
		err := row.Scan(&t.ID, &t.Title, &t.DueAt, &t.AssignedTo)
		return t, err
	})
}
