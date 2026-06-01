package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type MessageTemplateRepo struct {
	db *pgxpool.Pool
}

func NewMessageTemplateRepo(db *pgxpool.Pool) *MessageTemplateRepo {
	return &MessageTemplateRepo{db: db}
}

func (r *MessageTemplateRepo) List(ctx context.Context, companyID uuid.UUID, page, limit int) (*domain.PaginatedResult[domain.MessageTemplate], error) {
	offset := (page - 1) * limit

	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM message_templates WHERE company_id = $1`, companyID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count message templates: %w", err)
	}

	q := `SELECT id, company_id, name, body, category, variables, is_active, created_by, updated_by, created_at, updated_at
		FROM message_templates WHERE company_id = $1 ORDER BY category, name LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, companyID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list message templates: %w", err)
	}
	defer rows.Close()

	var templates []domain.MessageTemplate
	for rows.Next() {
		var t domain.MessageTemplate
		if err := rows.Scan(&t.ID, &t.CompanyID, &t.Name, &t.Body, &t.Category, &t.Variables,
			&t.IsActive, &t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan message template: %w", err)
		}
		templates = append(templates, t)
	}
	if templates == nil {
		templates = []domain.MessageTemplate{}
	}

	return &domain.PaginatedResult[domain.MessageTemplate]{
		Data:  templates,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *MessageTemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.MessageTemplate, error) {
	q := `SELECT id, company_id, name, body, category, variables, is_active, created_by, updated_by, created_at, updated_at
		FROM message_templates WHERE id = $1`
	t := &domain.MessageTemplate{}
	err := r.db.QueryRow(ctx, q, id).Scan(&t.ID, &t.CompanyID, &t.Name, &t.Body, &t.Category, &t.Variables,
		&t.IsActive, &t.CreatedBy, &t.UpdatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get message template: %w", err)
	}
	return t, nil
}

func (r *MessageTemplateRepo) Create(ctx context.Context, t *domain.MessageTemplate) error {
	q := `INSERT INTO message_templates (id, company_id, name, body, category, variables, is_active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING created_at, updated_at`
	t.ID = uuid.New()
	return r.db.QueryRow(ctx, q, t.ID, t.CompanyID, t.Name, t.Body, t.Category, t.Variables, t.IsActive, t.CreatedBy, t.UpdatedBy).
		Scan(&t.CreatedAt, &t.UpdatedAt)
}

func (r *MessageTemplateRepo) Update(ctx context.Context, t *domain.MessageTemplate) error {
	q := `UPDATE message_templates SET name=$1, body=$2, category=$3, variables=$4, is_active=$5, updated_by=$6, updated_at=NOW()
		WHERE id=$7 RETURNING updated_at`
	return r.db.QueryRow(ctx, q, t.Name, t.Body, t.Category, t.Variables, t.IsActive, t.UpdatedBy, t.ID).
		Scan(&t.UpdatedAt)
}

func (r *MessageTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM message_templates WHERE id = $1`, id)
	return err
}
