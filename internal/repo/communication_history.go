package repo

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type CommunicationHistoryRepo struct {
	pool *pgxpool.Pool
}

func NewCommunicationHistoryRepo(pool *pgxpool.Pool) *CommunicationHistoryRepo {
	return &CommunicationHistoryRepo{pool: pool}
}

func (r *CommunicationHistoryRepo) Create(ctx context.Context, comm *domain.CommunicationHistory) error {
	var metadataJSON []byte
	if comm.Metadata != nil {
		b, err := json.Marshal(comm.Metadata)
		if err != nil {
			return err
		}
		metadataJSON = b
	}

	query := `INSERT INTO communication_history
	(lead_id, contact_id, communication_type, direction, body, from_identifier, to_identifier, external_id, status, metadata, created_by)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
	).Scan(&comm.ID, &comm.CreatedAt)
}

func (r *CommunicationHistoryRepo) GetByLead(ctx context.Context, leadID uuid.UUID, limit int) ([]domain.CommunicationHistory, error) {
	query := `SELECT id, lead_id, contact_id, communication_type, direction, body,
	from_identifier, to_identifier, external_id, status, metadata, created_at, created_by
	FROM communication_history WHERE lead_id = $1 ORDER BY created_at DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, leadID, limit)
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
	FROM communication_history WHERE contact_id = $1 ORDER BY created_at DESC LIMIT $2`

	rows, err := r.pool.Query(ctx, query, contactID, limit)
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
	FROM communication_history WHERE external_id = $1 LIMIT 1`

	var comm domain.CommunicationHistory
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, externalID).Scan(
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
	query := `UPDATE communication_history SET status = $1 WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, status, id)
	return err
}
