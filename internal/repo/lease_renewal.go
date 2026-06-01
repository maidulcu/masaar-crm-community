package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

type LeaseRenewalRepo struct {
	conn *pgxpool.Pool
}

func NewLeaseRenewalRepo(conn *pgxpool.Pool) *LeaseRenewalRepo {
	return &LeaseRenewalRepo{conn: conn}
}

func (r *LeaseRenewalRepo) Create(ctx context.Context, workflow *domain.LeaseRenewalWorkflow) error {
	termsJSON, _ := json.Marshal(workflow.ProposedTerms)
	return r.conn.QueryRow(ctx, `
		INSERT INTO lease_renewal_workflows (id, company_id, lease_id, renewal_date, renewal_status, days_before_expiry, proposed_rent_amount, proposed_terms, tenant_response)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, workflow.ID, workflow.CompanyID, workflow.LeaseID, workflow.RenewalDate, workflow.RenewalStatus, workflow.DaysBeforeExpiry, workflow.ProposedRentAmount, termsJSON, workflow.TenantResponse).Scan(&workflow.ID, &workflow.CreatedAt, &workflow.UpdatedAt)
}

func (r *LeaseRenewalRepo) Get(ctx context.Context, id uuid.UUID) (*domain.LeaseRenewalWorkflow, error) {
	var workflow domain.LeaseRenewalWorkflow
	var termsJSON []byte
	err := r.conn.QueryRow(ctx, `
		SELECT id, company_id, lease_id, renewal_date, renewal_status, days_before_expiry, proposed_rent_amount, proposed_terms, tenant_response, tenant_counter_offer, counter_offer_date, created_at, updated_at
		FROM lease_renewal_workflows
		WHERE id = $1
	`, id).Scan(
		&workflow.ID, &workflow.CompanyID, &workflow.LeaseID, &workflow.RenewalDate, &workflow.RenewalStatus, &workflow.DaysBeforeExpiry, &workflow.ProposedRentAmount, &termsJSON, &workflow.TenantResponse, &workflow.TenantCounterOffer, &workflow.CounterOfferDate, &workflow.CreatedAt, &workflow.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if termsJSON != nil {
		json.Unmarshal(termsJSON, &workflow.ProposedTerms)
	}
	return &workflow, nil
}

func (r *LeaseRenewalRepo) GetByLeaseID(ctx context.Context, leaseID uuid.UUID) (*domain.LeaseRenewalWorkflow, error) {
	var workflow domain.LeaseRenewalWorkflow
	var termsJSON []byte
	err := r.conn.QueryRow(ctx, `
		SELECT id, company_id, lease_id, renewal_date, renewal_status, days_before_expiry, proposed_rent_amount, proposed_terms, tenant_response, tenant_counter_offer, counter_offer_date, created_at, updated_at
		FROM lease_renewal_workflows
		WHERE lease_id = $1
	`, leaseID).Scan(
		&workflow.ID, &workflow.CompanyID, &workflow.LeaseID, &workflow.RenewalDate, &workflow.RenewalStatus, &workflow.DaysBeforeExpiry, &workflow.ProposedRentAmount, &termsJSON, &workflow.TenantResponse, &workflow.TenantCounterOffer, &workflow.CounterOfferDate, &workflow.CreatedAt, &workflow.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if termsJSON != nil {
		json.Unmarshal(termsJSON, &workflow.ProposedTerms)
	}
	return &workflow, nil
}

func (r *LeaseRenewalRepo) List(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]domain.LeaseRenewalWorkflow, int, error) {
	var total int
	err := r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM lease_renewal_workflows WHERE company_id = $1`, companyID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, lease_id, renewal_date, renewal_status, days_before_expiry, proposed_rent_amount, proposed_terms, tenant_response, tenant_counter_offer, counter_offer_date, created_at, updated_at
		FROM lease_renewal_workflows
		WHERE company_id = $1
		ORDER BY renewal_date DESC
		LIMIT $2 OFFSET $3
	`, companyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workflows []domain.LeaseRenewalWorkflow
	for rows.Next() {
		var workflow domain.LeaseRenewalWorkflow
		var termsJSON []byte
		if err := rows.Scan(&workflow.ID, &workflow.CompanyID, &workflow.LeaseID, &workflow.RenewalDate, &workflow.RenewalStatus, &workflow.DaysBeforeExpiry, &workflow.ProposedRentAmount, &termsJSON, &workflow.TenantResponse, &workflow.TenantCounterOffer, &workflow.CounterOfferDate, &workflow.CreatedAt, &workflow.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if termsJSON != nil {
			json.Unmarshal(termsJSON, &workflow.ProposedTerms)
		}
		workflows = append(workflows, workflow)
	}
	return workflows, total, rows.Err()
}

func (r *LeaseRenewalRepo) ListByStatus(ctx context.Context, companyID uuid.UUID, status string, limit, offset int) ([]domain.LeaseRenewalWorkflow, int, error) {
	var total int
	err := r.conn.QueryRow(ctx, `SELECT COUNT(*) FROM lease_renewal_workflows WHERE company_id = $1 AND renewal_status = $2`, companyID, status).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, lease_id, renewal_date, renewal_status, days_before_expiry, proposed_rent_amount, proposed_terms, tenant_response, tenant_counter_offer, counter_offer_date, created_at, updated_at
		FROM lease_renewal_workflows
		WHERE company_id = $1 AND renewal_status = $2
		ORDER BY renewal_date DESC
		LIMIT $3 OFFSET $4
	`, companyID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workflows []domain.LeaseRenewalWorkflow
	for rows.Next() {
		var workflow domain.LeaseRenewalWorkflow
		var termsJSON []byte
		if err := rows.Scan(&workflow.ID, &workflow.CompanyID, &workflow.LeaseID, &workflow.RenewalDate, &workflow.RenewalStatus, &workflow.DaysBeforeExpiry, &workflow.ProposedRentAmount, &termsJSON, &workflow.TenantResponse, &workflow.TenantCounterOffer, &workflow.CounterOfferDate, &workflow.CreatedAt, &workflow.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if termsJSON != nil {
			json.Unmarshal(termsJSON, &workflow.ProposedTerms)
		}
		workflows = append(workflows, workflow)
	}
	return workflows, total, rows.Err()
}

func (r *LeaseRenewalRepo) Update(ctx context.Context, workflow *domain.LeaseRenewalWorkflow) error {
	workflow.UpdatedAt = time.Now()
	termsJSON, _ := json.Marshal(workflow.ProposedTerms)
	return r.conn.QueryRow(ctx, `
		UPDATE lease_renewal_workflows
		SET renewal_status = $2, proposed_rent_amount = $3, proposed_terms = $4, tenant_response = $5, tenant_counter_offer = $6, counter_offer_date = $7, updated_at = $8
		WHERE id = $1
		RETURNING updated_at
	`, workflow.ID, workflow.RenewalStatus, workflow.ProposedRentAmount, termsJSON, workflow.TenantResponse, workflow.TenantCounterOffer, workflow.CounterOfferDate, workflow.UpdatedAt).Scan(&workflow.UpdatedAt)
}

type RenewalTemplateRepo struct {
	conn *pgxpool.Pool
}

func NewRenewalTemplateRepo(conn *pgxpool.Pool) *RenewalTemplateRepo {
	return &RenewalTemplateRepo{conn: conn}
}

func (r *RenewalTemplateRepo) Create(ctx context.Context, template *domain.RenewalCommunicationTemplate) error {
	return r.conn.QueryRow(ctx, `
		INSERT INTO renewal_communication_templates (id, company_id, template_name, email_subject, email_body, whatsapp_message, language)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, template.ID, template.CompanyID, template.TemplateName, template.EmailSubject, template.EmailBody, template.WhatsAppMsg, template.Language).Scan(&template.ID, &template.CreatedAt)
}

func (r *RenewalTemplateRepo) List(ctx context.Context, companyID uuid.UUID) ([]domain.RenewalCommunicationTemplate, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT id, company_id, template_name, email_subject, email_body, whatsapp_message, language, created_at
		FROM renewal_communication_templates
		WHERE company_id = $1
		ORDER BY template_name
	`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []domain.RenewalCommunicationTemplate
	for rows.Next() {
		var t domain.RenewalCommunicationTemplate
		if err := rows.Scan(&t.ID, &t.CompanyID, &t.TemplateName, &t.EmailSubject, &t.EmailBody, &t.WhatsAppMsg, &t.Language, &t.CreatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (r *RenewalTemplateRepo) Get(ctx context.Context, id uuid.UUID) (*domain.RenewalCommunicationTemplate, error) {
	var t domain.RenewalCommunicationTemplate
	err := r.conn.QueryRow(ctx, `
		SELECT id, company_id, template_name, email_subject, email_body, whatsapp_message, language, created_at
		FROM renewal_communication_templates
		WHERE id = $1
	`, id).Scan(&t.ID, &t.CompanyID, &t.TemplateName, &t.EmailSubject, &t.EmailBody, &t.WhatsAppMsg, &t.Language, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RenewalTemplateRepo) Update(ctx context.Context, t *domain.RenewalCommunicationTemplate) error {
	_, err := r.conn.Exec(ctx, `
		UPDATE renewal_communication_templates
		SET template_name=$1, email_subject=$2, email_body=$3, whatsapp_message=$4, language=$5
		WHERE id=$6
	`, t.TemplateName, t.EmailSubject, t.EmailBody, t.WhatsAppMsg, t.Language, t.ID)
	return err
}

func (r *RenewalTemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.conn.Exec(ctx, `DELETE FROM renewal_communication_templates WHERE id=$1`, id)
	return err
}

type RenewalCommunicationLogRepo struct {
	conn *pgxpool.Pool
}

func NewRenewalCommunicationLogRepo(conn *pgxpool.Pool) *RenewalCommunicationLogRepo {
	return &RenewalCommunicationLogRepo{conn: conn}
}

func (r *RenewalCommunicationLogRepo) Create(ctx context.Context, log *domain.RenewalCommunicationLog) error {
	return r.conn.QueryRow(ctx, `
		INSERT INTO renewal_communication_log (id, renewal_id, communication_type, template_id, sent_date, delivery_status, response_text)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, log.ID, log.RenewalID, log.CommunicationType, log.TemplateID, log.SentDate, log.DeliveryStatus, log.ResponseText).Scan(&log.ID, &log.CreatedAt)
}

func (r *RenewalCommunicationLogRepo) GetByRenewalID(ctx context.Context, renewalID uuid.UUID) ([]domain.RenewalCommunicationLog, error) {
	rows, err := r.conn.Query(ctx, `
		SELECT id, renewal_id, communication_type, template_id, sent_date, delivery_status, tenant_response_date, response_text, created_at
		FROM renewal_communication_log
		WHERE renewal_id = $1
		ORDER BY created_at DESC
	`, renewalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.RenewalCommunicationLog
	for rows.Next() {
		var log domain.RenewalCommunicationLog
		if err := rows.Scan(&log.ID, &log.RenewalID, &log.CommunicationType, &log.TemplateID, &log.SentDate, &log.DeliveryStatus, &log.TenantResponseDate, &log.ResponseText, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}
