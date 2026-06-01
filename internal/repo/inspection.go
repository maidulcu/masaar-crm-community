package repo

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

type InspectionTemplateRepo struct {
	conn *pgxpool.Pool
}

func NewInspectionTemplateRepo(conn *pgxpool.Pool) *InspectionTemplateRepo {
	return &InspectionTemplateRepo{conn: conn}
}

func (r *InspectionTemplateRepo) Create(ctx context.Context, template *domain.InspectionTemplate) error {
	checklistJSON, _ := json.Marshal(template.ChecklistItems)
	return r.conn.QueryRow(ctx, `
		INSERT INTO inspection_templates (id, company_id, template_name, inspection_type, checklist_items, estimated_duration_minutes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, template.ID, template.CompanyID, template.TemplateName, template.InspectionType, checklistJSON, template.EstimatedDurationMinutes).Scan(&template.ID, &template.CreatedAt)
}

func (r *InspectionTemplateRepo) List(ctx context.Context, companyID uuid.UUID) ([]domain.InspectionTemplate, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, template_name, inspection_type, checklist_items, estimated_duration_minutes, created_at
		FROM inspection_templates
		WHERE company_id = $1
		ORDER BY template_name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []domain.InspectionTemplate
	for rows.Next() {
		var t domain.InspectionTemplate
		var checklistJSON []byte
		if err := rows.Scan(&t.ID, &t.CompanyID, &t.TemplateName, &t.InspectionType, &checklistJSON, &t.EstimatedDurationMinutes, &t.CreatedAt); err != nil {
			return nil, err
		}
		if checklistJSON != nil {
			json.Unmarshal(checklistJSON, &t.ChecklistItems)
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *InspectionTemplateRepo) Get(ctx context.Context, id uuid.UUID) (*domain.InspectionTemplate, error) {
	var t domain.InspectionTemplate
	var checklistJSON []byte
	err := r.conn.QueryRow(ctx, `
		SELECT id, company_id, template_name, inspection_type, checklist_items, estimated_duration_minutes, created_at
		FROM inspection_templates
		WHERE id = $1
	`, id).Scan(&t.ID, &t.CompanyID, &t.TemplateName, &t.InspectionType, &checklistJSON, &t.EstimatedDurationMinutes, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	if checklistJSON != nil {
		json.Unmarshal(checklistJSON, &t.ChecklistItems)
	}
	return &t, nil
}

func (r *InspectionTemplateRepo) Update(ctx context.Context, template *domain.InspectionTemplate) error {
	checklistJSON, _ := json.Marshal(template.ChecklistItems)
	_, err := r.conn.Exec(ctx, `
		UPDATE inspection_templates
		SET template_name = $2, inspection_type = $3, checklist_items = $4, estimated_duration_minutes = $5
		WHERE id = $1
	`, template.ID, template.TemplateName, template.InspectionType, checklistJSON, template.EstimatedDurationMinutes)
	return err
}

type InspectionRepo struct {
	conn *pgxpool.Pool
}

func NewInspectionRepo(conn *pgxpool.Pool) *InspectionRepo {
	return &InspectionRepo{conn: conn}
}

func (r *InspectionRepo) Create(ctx context.Context, insp *domain.Inspection) error {
	checklistJSON, _ := json.Marshal(insp.ChecklistResults)
	return r.conn.QueryRow(ctx, `
		INSERT INTO inspections (id, company_id, property_id, template_id, inspection_type, scheduled_date, inspector_id, tenant_id, status, findings, severity_level, photos_urls, checklist_results, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`, insp.ID, insp.CompanyID, insp.PropertyID, insp.TemplateID, insp.InspectionType, insp.ScheduledDate, insp.InspectorID, insp.TenantID, insp.Status, insp.Findings, insp.SeverityLevel, insp.PhotosURLs, checklistJSON, insp.CreatedBy).Scan(&insp.ID, &insp.CreatedAt, &insp.UpdatedAt)
}

func (r *InspectionRepo) Get(ctx context.Context, id uuid.UUID) (*domain.Inspection, error) {
	var insp domain.Inspection
	var checklistJSON []byte
	err := r.conn.QueryRow(ctx, `
		SELECT id, company_id, property_id, template_id, inspection_type, scheduled_date, completed_date, inspector_id, tenant_id, status, findings, severity_level, photos_urls, checklist_results, created_by, created_at, updated_at
		FROM inspections
		WHERE id = $1
	`, id).Scan(
		&insp.ID, &insp.CompanyID, &insp.PropertyID, &insp.TemplateID, &insp.InspectionType, &insp.ScheduledDate, &insp.CompletedDate, &insp.InspectorID, &insp.TenantID, &insp.Status, &insp.Findings, &insp.SeverityLevel, &insp.PhotosURLs, &checklistJSON, &insp.CreatedBy, &insp.CreatedAt, &insp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if checklistJSON != nil {
		json.Unmarshal(checklistJSON, &insp.ChecklistResults)
	}
	return &insp, nil
}

func (r *InspectionRepo) List(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.Inspection, int, error) {
	var total int
	err := r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM inspections WHERE company_id = $1`, companyID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, property_id, template_id, inspection_type, scheduled_date, completed_date, inspector_id, tenant_id, status, findings, severity_level, photos_urls, checklist_results, created_by, created_at, updated_at
		FROM inspections
		WHERE company_id = $1
		ORDER BY scheduled_date DESC
		LIMIT $2 OFFSET $3
	`, companyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var inspections []domain.Inspection
	for rows.Next() {
		var insp domain.Inspection
		var checklistJSON []byte
		if err := rows.Scan(&insp.ID, &insp.CompanyID, &insp.PropertyID, &insp.TemplateID, &insp.InspectionType, &insp.ScheduledDate, &insp.CompletedDate, &insp.InspectorID, &insp.TenantID, &insp.Status, &insp.Findings, &insp.SeverityLevel, &insp.PhotosURLs, &checklistJSON, &insp.CreatedBy, &insp.CreatedAt, &insp.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if checklistJSON != nil {
			json.Unmarshal(checklistJSON, &insp.ChecklistResults)
		}
		inspections = append(inspections, insp)
	}
	return inspections, total, rows.Err()
}

func (r *InspectionRepo) ListByProperty(ctx context.Context, propertyID uuid.UUID, limit, offset int) ([]domain.Inspection, int, error) {
	var total int
	err := r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM inspections WHERE property_id = $1`, propertyID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, property_id, template_id, inspection_type, scheduled_date, completed_date, inspector_id, tenant_id, status, findings, severity_level, photos_urls, checklist_results, created_by, created_at, updated_at
		FROM inspections
		WHERE property_id = $1
		ORDER BY scheduled_date DESC
		LIMIT $2 OFFSET $3
	`, propertyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var inspections []domain.Inspection
	for rows.Next() {
		var insp domain.Inspection
		var checklistJSON []byte
		if err := rows.Scan(&insp.ID, &insp.CompanyID, &insp.PropertyID, &insp.TemplateID, &insp.InspectionType, &insp.ScheduledDate, &insp.CompletedDate, &insp.InspectorID, &insp.TenantID, &insp.Status, &insp.Findings, &insp.SeverityLevel, &insp.PhotosURLs, &checklistJSON, &insp.CreatedBy, &insp.CreatedAt, &insp.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if checklistJSON != nil {
			json.Unmarshal(checklistJSON, &insp.ChecklistResults)
		}
		inspections = append(inspections, insp)
	}
	return inspections, total, rows.Err()
}

func (r *InspectionRepo) Update(ctx context.Context, insp *domain.Inspection) error {
	checklistJSON, _ := json.Marshal(insp.ChecklistResults)
	return r.conn.QueryRow(ctx, `
		UPDATE inspections
		SET status = $2, findings = $3, severity_level = $4, photos_urls = $5, checklist_results = $6, completed_date = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`, insp.ID, insp.Status, insp.Findings, insp.SeverityLevel, insp.PhotosURLs, checklistJSON, insp.CompletedDate).Scan(&insp.UpdatedAt)
}
