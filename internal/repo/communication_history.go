package repo

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type CommunicationHistoryRepo struct {
	pool *pgxpool.Pool
}

func NewCommunicationHistoryRepo(pool *pgxpool.Pool) *CommunicationHistoryRepo {
	return &CommunicationHistoryRepo{pool: pool}
}

func (r *CommunicationHistoryRepo) Create(ctx context.Context, comm *domain.CommunicationHistory) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	var metadataJSON []byte
	if comm.Metadata != nil {
		b, err := json.Marshal(comm.Metadata)
		if err != nil {
			return err
		}
		metadataJSON = b
	}

	query := `INSERT INTO communication_history
	(lead_id, contact_id, communication_type, direction, body, from_identifier, to_identifier, external_id, status, metadata, created_by, company_id)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		comm.LeadID,
		comm.ContactID,
		comm.CommunicationType,
		comm.Direction,
		comm.Body,
		comm.FromIdentifier,
		comm.ToIdentifier,
		comm.ExternalID,
		comm.Status,
		metadataJSON,
		comm.CreatedBy,
		cid,
	).Scan(&comm.ID, &comm.CreatedAt)
}

func (r *CommunicationHistoryRepo) GetByLead(ctx context.Context, leadID uuid.UUID, limit int) ([]domain.CommunicationHistory, error) {
	query := `SELECT id, lead_id, contact_id, communication_type, direction, body,
	from_identifier, to_identifier, external_id, status, metadata, created_at, created_by
	FROM communication_history WHERE lead_id = $1 AND company_id = $3 ORDER BY created_at DESC LIMIT $2`

	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, leadID, limit, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var communications []domain.CommunicationHistory
	for rows.Next() {
		var comm domain.CommunicationHistory
		var metadataJSON []byte

		err := rows.Scan(
			&comm.ID,
			&comm.LeadID,
			&comm.ContactID,
			&comm.CommunicationType,
			&comm.Direction,
			&comm.Body,
			&comm.FromIdentifier,
			&comm.ToIdentifier,
			&comm.ExternalID,
			&comm.Status,
			&metadataJSON,
			&comm.CreatedAt,
			&comm.CreatedBy,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &comm.Metadata)
		}

		communications = append(communications, comm)
	}

	return communications, rows.Err()
}

func (r *CommunicationHistoryRepo) GetByContact(ctx context.Context, contactID uuid.UUID, limit int) ([]domain.CommunicationHistory, error) {
	query := `SELECT id, lead_id, contact_id, communication_type, direction, body,
	from_identifier, to_identifier, external_id, status, metadata, created_at, created_by
	FROM communication_history WHERE contact_id = $1 AND company_id = $3 ORDER BY created_at DESC LIMIT $2`

	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, contactID, limit, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var communications []domain.CommunicationHistory
	for rows.Next() {
		var comm domain.CommunicationHistory
		var metadataJSON []byte

		err := rows.Scan(
			&comm.ID,
			&comm.LeadID,
			&comm.ContactID,
			&comm.CommunicationType,
			&comm.Direction,
			&comm.Body,
			&comm.FromIdentifier,
			&comm.ToIdentifier,
			&comm.ExternalID,
			&comm.Status,
			&metadataJSON,
			&comm.CreatedAt,
			&comm.CreatedBy,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &comm.Metadata)
		}

		communications = append(communications, comm)
	}

	return communications, rows.Err()
}

func (r *CommunicationHistoryRepo) GetByExternalID(ctx context.Context, externalID string) (*domain.CommunicationHistory, error) {
	query := `SELECT id, lead_id, contact_id, communication_type, direction, body,
	from_identifier, to_identifier, external_id, status, metadata, created_at, created_by
	FROM communication_history WHERE external_id = $1 AND company_id = $2 LIMIT 1`

	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	var comm domain.CommunicationHistory
	var metadataJSON []byte

	err = r.pool.QueryRow(ctx, query, externalID, cid).Scan(
		&comm.ID,
		&comm.LeadID,
		&comm.ContactID,
		&comm.CommunicationType,
		&comm.Direction,
		&comm.Body,
		&comm.FromIdentifier,
		&comm.ToIdentifier,
		&comm.ExternalID,
		&comm.Status,
		&metadataJSON,
		&comm.CreatedAt,
		&comm.CreatedBy,
	)

	if err != nil {
		return nil, err
	}

	if metadataJSON != nil {
		json.Unmarshal(metadataJSON, &comm.Metadata)
	}

	return &comm, nil
}

func (r *CommunicationHistoryRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE communication_history SET status = $1 WHERE id = $2 AND company_id = $3`
	_, err = r.pool.Exec(ctx, query, status, id, cid)
	return err
}

// LogWhatsApp adds a WhatsApp message to the timeline of the contact's most recent lead.
// Timeline entries belong to a lead, so a contact with no lead yet gets none (returns false);
// the message itself is always kept in the WhatsApp thread.
func (r *CommunicationHistoryRepo) LogWhatsApp(ctx context.Context, contactID uuid.UUID, comm *domain.CommunicationHistory) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO communication_history
			(lead_id, contact_id, communication_type, direction, body, from_identifier, to_identifier, external_id, status, created_by, company_id)
		SELECT l.id, $1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, $9, $10
		FROM leads l
		WHERE l.contact_id = $1 AND l.company_id = $10
		ORDER BY l.created_at DESC
		LIMIT 1`,
		contactID, comm.CommunicationType, comm.Direction, comm.Body, comm.FromIdentifier, comm.ToIdentifier,
		comm.ExternalID, comm.Status, comm.CreatedBy, cid)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// UpdateStatusByExternalID mirrors a WhatsApp delivery receipt onto the timeline entry.
func (r *CommunicationHistoryRepo) UpdateStatusByExternalID(ctx context.Context, externalID, status string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE communication_history SET status = $1 WHERE external_id = $2 AND company_id = $3`, status, externalID, cid)
	return err
}
