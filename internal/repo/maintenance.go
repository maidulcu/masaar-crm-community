package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type MaintenanceTaskRepo struct {
	conn *pgxpool.Pool
}

func NewMaintenanceTaskRepo(conn *pgxpool.Pool) *MaintenanceTaskRepo {
	return &MaintenanceTaskRepo{conn: conn}
}

func (r *MaintenanceTaskRepo) Create(ctx context.Context, task *domain.MaintenanceTask) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	task.CompanyID = cid // never trust a company id supplied by the client
	// Property, inspection and assignee (when given) must belong to this company.
	return r.conn.QueryRow(ctx, `
		INSERT INTO maintenance_tasks (id, company_id, property_id, inspection_id, maintenance_type, description, priority, scheduled_date, due_date, contractor_name, contractor_contact, estimated_cost, status, assigned_to, notes, created_by)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		WHERE EXISTS (SELECT 1 FROM rental_properties WHERE id = $3 AND company_id = $2)
		  AND ($4::uuid IS NULL OR EXISTS (SELECT 1 FROM inspections WHERE id = $4 AND company_id = $2))
		  AND ($14::uuid IS NULL OR EXISTS (SELECT 1 FROM users WHERE id = $14 AND company_id = $2))
		RETURNING id, created_at, updated_at
	`, task.ID, task.CompanyID, task.PropertyID, task.InspectionID, task.MaintenanceType, task.Description, task.Priority, task.ScheduledDate, task.DueDate, task.ContractorName, task.ContractorContact, task.EstimatedCost, task.Status, task.AssignedTo, task.Notes, task.CreatedBy).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
}

func (r *MaintenanceTaskRepo) Get(ctx context.Context, id uuid.UUID) (*domain.MaintenanceTask, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	var task domain.MaintenanceTask
	err = r.conn.QueryRow(ctx, `
		SELECT id, company_id, property_id, inspection_id, maintenance_type, description, priority, scheduled_date, due_date, completion_date, contractor_name, contractor_contact, estimated_cost, actual_cost, status, assigned_to, notes, created_by, created_at, updated_at, deleted_at
		FROM maintenance_tasks
		WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL
	`, id, cid).Scan(
		&task.ID, &task.CompanyID, &task.PropertyID, &task.InspectionID, &task.MaintenanceType, &task.Description, &task.Priority, &task.ScheduledDate, &task.DueDate, &task.CompletionDate, &task.ContractorName, &task.ContractorContact, &task.EstimatedCost, &task.ActualCost, &task.Status, &task.AssignedTo, &task.Notes, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *MaintenanceTaskRepo) List(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.MaintenanceTask, int, error) {
	var total int
	err := r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM maintenance_tasks WHERE company_id = $1 AND deleted_at IS NULL`, companyID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, property_id, inspection_id, maintenance_type, description, priority, scheduled_date, due_date, completion_date, contractor_name, contractor_contact, estimated_cost, actual_cost, status, assigned_to, notes, created_by, created_at, updated_at, deleted_at
		FROM maintenance_tasks
		WHERE company_id = $1 AND deleted_at IS NULL
		ORDER BY due_date ASC, priority DESC
		LIMIT $2 OFFSET $3
	`, companyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []domain.MaintenanceTask
	for rows.Next() {
		var task domain.MaintenanceTask
		if err := rows.Scan(&task.ID, &task.CompanyID, &task.PropertyID, &task.InspectionID, &task.MaintenanceType, &task.Description, &task.Priority, &task.ScheduledDate, &task.DueDate, &task.CompletionDate, &task.ContractorName, &task.ContractorContact, &task.EstimatedCost, &task.ActualCost, &task.Status, &task.AssignedTo, &task.Notes, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}
	return tasks, total, rows.Err()
}

func (r *MaintenanceTaskRepo) ListByProperty(ctx context.Context, propertyID uuid.UUID, limit, offset int) ([]domain.MaintenanceTask, int, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, 0, err
	}
	var total int
	err = r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM maintenance_tasks WHERE property_id = $1 AND company_id = $2 AND deleted_at IS NULL`, propertyID, cid).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, property_id, inspection_id, maintenance_type, description, priority, scheduled_date, due_date, completion_date, contractor_name, contractor_contact, estimated_cost, actual_cost, status, assigned_to, notes, created_by, created_at, updated_at, deleted_at
		FROM maintenance_tasks
		WHERE property_id = $1 AND company_id = $4 AND deleted_at IS NULL
		ORDER BY due_date ASC, priority DESC
		LIMIT $2 OFFSET $3
	`, propertyID, limit, offset, cid)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []domain.MaintenanceTask
	for rows.Next() {
		var task domain.MaintenanceTask
		if err := rows.Scan(&task.ID, &task.CompanyID, &task.PropertyID, &task.InspectionID, &task.MaintenanceType, &task.Description, &task.Priority, &task.ScheduledDate, &task.DueDate, &task.CompletionDate, &task.ContractorName, &task.ContractorContact, &task.EstimatedCost, &task.ActualCost, &task.Status, &task.AssignedTo, &task.Notes, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}
	return tasks, total, rows.Err()
}

func (r *MaintenanceTaskRepo) ListByStatus(ctx context.Context, companyID uuid.UUID, status string, limit, offset int) ([]domain.MaintenanceTask, int, error) {
	var total int
	err := r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM maintenance_tasks WHERE company_id = $1 AND status = $2 AND deleted_at IS NULL`, companyID, status).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, property_id, inspection_id, maintenance_type, description, priority, scheduled_date, due_date, completion_date, contractor_name, contractor_contact, estimated_cost, actual_cost, status, assigned_to, notes, created_by, created_at, updated_at, deleted_at
		FROM maintenance_tasks
		WHERE company_id = $1 AND status = $2 AND deleted_at IS NULL
		ORDER BY due_date ASC, priority DESC
		LIMIT $3 OFFSET $4
	`, companyID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []domain.MaintenanceTask
	for rows.Next() {
		var task domain.MaintenanceTask
		if err := rows.Scan(&task.ID, &task.CompanyID, &task.PropertyID, &task.InspectionID, &task.MaintenanceType, &task.Description, &task.Priority, &task.ScheduledDate, &task.DueDate, &task.CompletionDate, &task.ContractorName, &task.ContractorContact, &task.EstimatedCost, &task.ActualCost, &task.Status, &task.AssignedTo, &task.Notes, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}
	return tasks, total, rows.Err()
}

func (r *MaintenanceTaskRepo) Update(ctx context.Context, task *domain.MaintenanceTask) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	task.UpdatedAt = time.Now()
	return r.conn.QueryRow(ctx, `
		UPDATE maintenance_tasks
		SET maintenance_type = $2, description = $3, priority = $4, scheduled_date = $5, due_date = $6, contractor_name = $7, contractor_contact = $8, estimated_cost = $9, actual_cost = $10, status = $11, assigned_to = $12, notes = $13, completion_date = $14, updated_at = $15
		WHERE id = $1 AND company_id = $16 AND deleted_at IS NULL
		  AND ($12::uuid IS NULL OR EXISTS (SELECT 1 FROM users WHERE id = $12 AND company_id = $16))
		RETURNING updated_at
	`, task.ID, task.MaintenanceType, task.Description, task.Priority, task.ScheduledDate, task.DueDate, task.ContractorName, task.ContractorContact, task.EstimatedCost, task.ActualCost, task.Status, task.AssignedTo, task.Notes, task.CompletionDate, task.UpdatedAt, cid).Scan(&task.UpdatedAt)
}

func (r *MaintenanceTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.conn.Exec(ctx, `
		UPDATE maintenance_tasks
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL
	`, id, cid)
	return err
}

func (r *MaintenanceTaskRepo) AddPhoto(ctx context.Context, photo *domain.MaintenancePhoto) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// The task must belong to the caller's company.
	return r.conn.QueryRow(ctx, `
		INSERT INTO maintenance_photos (id, task_id, photo_url, photo_stage)
		SELECT $1, $2, $3, $4
		WHERE EXISTS (SELECT 1 FROM maintenance_tasks WHERE id = $2 AND company_id = $5)
		RETURNING id, uploaded_at
	`, photo.ID, photo.TaskID, photo.PhotoURL, photo.PhotoStage, cid).Scan(&photo.ID, &photo.UploadedAt)
}

func (r *MaintenanceTaskRepo) GetPhotos(ctx context.Context, taskID uuid.UUID) ([]domain.MaintenancePhoto, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.conn.Query(ctx, `
		SELECT p.id, p.task_id, p.photo_url, p.uploaded_at, p.photo_stage
		FROM maintenance_photos p JOIN maintenance_tasks t ON t.id = p.task_id
		WHERE p.task_id = $1 AND t.company_id = $2
		ORDER BY p.uploaded_at ASC
	`, taskID, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []domain.MaintenancePhoto
	for rows.Next() {
		var p domain.MaintenancePhoto
		if err := rows.Scan(&p.ID, &p.TaskID, &p.PhotoURL, &p.UploadedAt, &p.PhotoStage); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, rows.Err()
}
