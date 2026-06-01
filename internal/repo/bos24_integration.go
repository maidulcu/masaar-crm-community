package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

// BOS24IntegrationRepo handles storage for per-company BOS24 integration settings
// and provides idempotent upsert helpers for synced listings and leads.
type BOS24IntegrationRepo struct {
	db *pgxpool.Pool
}

func NewBOS24IntegrationRepo(db *pgxpool.Pool) *BOS24IntegrationRepo {
	return &BOS24IntegrationRepo{db: db}
}

// GetSettings returns the BOS24 integration settings for a company.
// Returns a zeroed struct (no error) if no row exists yet.
func (r *BOS24IntegrationRepo) GetSettings(ctx context.Context, companyID uuid.UUID) (*domain.BOS24Settings, error) {
	const q = `
		SELECT id, company_id, api_key, webhook_secret, webhook_id, last_sync_at, updated_at
		FROM bos24_integration_settings
		WHERE company_id = $1
	`
	s := &domain.BOS24Settings{}
	err := r.db.QueryRow(ctx, q, companyID).Scan(
		&s.ID, &s.CompanyID,
		&s.APIKey, &s.WebhookSecret, &s.WebhookID,
		&s.LastSyncAt, &s.UpdatedAt,
	)
	if err != nil {
		// No row is not an error — return an empty settings object
		return &domain.BOS24Settings{CompanyID: companyID}, nil
	}
	return s, nil
}

// SaveSettings upserts BOS24 integration settings for a company.
func (r *BOS24IntegrationRepo) SaveSettings(ctx context.Context, s *domain.BOS24Settings) error {
	const q = `
		INSERT INTO bos24_integration_settings
			(id, company_id, api_key, webhook_secret, webhook_id, last_sync_at, updated_at)
		VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, NOW())
		ON CONFLICT (company_id) DO UPDATE SET
			api_key        = EXCLUDED.api_key,
			webhook_secret = EXCLUDED.webhook_secret,
			webhook_id     = EXCLUDED.webhook_id,
			last_sync_at   = EXCLUDED.last_sync_at,
			updated_at     = NOW()
	`
	_, err := r.db.Exec(ctx, q,
		s.CompanyID, s.APIKey, s.WebhookSecret, s.WebhookID, s.LastSyncAt,
	)
	return err
}

// UpdateLastSyncAt sets the last_sync_at timestamp for a company's BOS24 settings.
func (r *BOS24IntegrationRepo) UpdateLastSyncAt(ctx context.Context, companyID uuid.UUID, t time.Time) error {
	const q = `
		UPDATE bos24_integration_settings SET last_sync_at = $1, updated_at = NOW()
		WHERE company_id = $2
	`
	_, err := r.db.Exec(ctx, q, t, companyID)
	return err
}

// GetCompanyByWebhookSecret finds the company whose BOS24 webhook_secret matches
// the provided token. Used to route inbound webhook requests to the correct tenant.
func (r *BOS24IntegrationRepo) GetCompanyByWebhookSecret(ctx context.Context, secret string) (*domain.BOS24Settings, error) {
	const q = `
		SELECT id, company_id, api_key, webhook_secret, webhook_id, last_sync_at, updated_at
		FROM bos24_integration_settings
		WHERE webhook_secret = $1 AND webhook_secret <> ''
	`
	s := &domain.BOS24Settings{}
	err := r.db.QueryRow(ctx, q, secret).Scan(
		&s.ID, &s.CompanyID,
		&s.APIKey, &s.WebhookSecret, &s.WebhookID,
		&s.LastSyncAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// UpsertListing inserts or updates a listing imported from BOS24.
// Idempotent: ON CONFLICT (company_id, bos24_listing_uuid) updates the key fields.
// Returns the listing UUID (Masaar's internal UUID).
func (r *BOS24IntegrationRepo) UpsertListing(ctx context.Context,
	companyID uuid.UUID,
	bos24UUID, title, description, propertyType, listingType, city, status, coverImageURL, currency string,
	price float64,
) (uuid.UUID, error) {
	const q = `
		INSERT INTO listings (
			id, company_id, bos24_listing_uuid,
			title, description, property_type, listing_type,
			city, status, cover_image_url,
			price, currency,
			portal_sync_status
		) VALUES (
			uuid_generate_v4(), $1, $2,
			$3, $4, $5, $6,
			$7, $8, $9,
			$10, $11,
			'{"source":"bos24"}'::jsonb
		)
		ON CONFLICT (company_id, bos24_listing_uuid) DO UPDATE SET
			title          = EXCLUDED.title,
			description    = EXCLUDED.description,
			property_type  = EXCLUDED.property_type,
			listing_type   = EXCLUDED.listing_type,
			city           = EXCLUDED.city,
			status         = EXCLUDED.status,
			cover_image_url = EXCLUDED.cover_image_url,
			price          = EXCLUDED.price,
			currency       = EXCLUDED.currency,
			updated_at     = NOW()
		RETURNING id
	`
	var id uuid.UUID
	err := r.db.QueryRow(ctx, q,
		companyID, bos24UUID,
		title, description, propertyType, listingType,
		city, status, coverImageURL,
		price, currency,
	).Scan(&id)
	return id, err
}

// DeactivateListing marks a BOS24 listing as inactive/deleted.
func (r *BOS24IntegrationRepo) DeactivateListing(ctx context.Context, companyID uuid.UUID, bos24UUID string) error {
	const q = `
		UPDATE listings SET status = 'inactive', updated_at = NOW()
		WHERE company_id = $1 AND bos24_listing_uuid = $2
	`
	_, err := r.db.Exec(ctx, q, companyID, bos24UUID)
	return err
}

// CreateLeadFromInquiry creates a lead from a BOS24 inquiry if it doesn't already exist.
// Idempotent: does nothing on conflict (bos24_inquiry_id already imported).
// Returns true if a new lead was created, false if it was a duplicate.
func (r *BOS24IntegrationRepo) CreateLeadFromInquiry(ctx context.Context,
	contactID uuid.UUID,
	bos24InquiryID int,
	notes string,
) (created bool, leadID uuid.UUID, err error) {
	const q = `
		INSERT INTO leads (id, contact_id, stage, source, deal_value, currency, notes, bos24_inquiry_id)
		VALUES (uuid_generate_v4(), $1, 'new', 'bos24', 0, 'AED', $2, $3)
		ON CONFLICT (bos24_inquiry_id) DO NOTHING
		RETURNING id
	`
	err = r.db.QueryRow(ctx, q, contactID, notes, bos24InquiryID).Scan(&leadID)
	if err != nil {
		// pgx returns ErrNoRows when DO NOTHING fires — that's a duplicate, not an error
		if err.Error() == "no rows in result set" {
			return false, uuid.Nil, nil
		}
		return false, uuid.Nil, err
	}
	return true, leadID, nil
}

// UpdateContactEmailIfEmpty sets email on a contact only when the contact's
// current email is NULL or empty — used when enriching from a BOS24 inquiry.
func (r *BOS24IntegrationRepo) UpdateContactEmailIfEmpty(ctx context.Context, contactID uuid.UUID, email string) error {
	const q = `
		UPDATE contacts SET email = $1, updated_at = NOW()
		WHERE id = $2 AND (email IS NULL OR email = '')
	`
	_, err := r.db.Exec(ctx, q, email, contactID)
	return err
}

// ListCompaniesWithBOS24 returns all companies that have a non-empty BOS24 API key.
// Used by the nightly sync goroutine to iterate over active integrations.
func (r *BOS24IntegrationRepo) ListCompaniesWithBOS24(ctx context.Context) ([]*domain.BOS24Settings, error) {
	const q = `
		SELECT id, company_id, api_key, webhook_secret, webhook_id, last_sync_at, updated_at
		FROM bos24_integration_settings
		WHERE api_key <> ''
		ORDER BY company_id
	`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.BOS24Settings
	for rows.Next() {
		s := &domain.BOS24Settings{}
		if err := rows.Scan(
			&s.ID, &s.CompanyID,
			&s.APIKey, &s.WebhookSecret, &s.WebhookID,
			&s.LastSyncAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, nil
}
