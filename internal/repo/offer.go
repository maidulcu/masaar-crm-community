package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type OfferRepo struct {
	db *pgxpool.Pool
}

func NewOfferRepo(db *pgxpool.Pool) *OfferRepo {
	return &OfferRepo{db: db}
}

const offerCols = `
	o.id, o.listing_id, o.contact_id, o.agent_id, o.parent_offer_id,
	o.offer_amount, o.currency, o.status, o.terms, o.notes,
	o.valid_until, o.deal_id, o.created_at, o.updated_at`

func scanOffer(row interface{ Scan(...any) error }, o *domain.Offer) error {
	return row.Scan(
		&o.ID, &o.ListingID, &o.ContactID, &o.AgentID, &o.ParentOfferID,
		&o.OfferAmount, &o.Currency, &o.Status, &o.Terms, &o.Notes,
		&o.ValidUntil, &o.DealID, &o.CreatedAt, &o.UpdatedAt,
	)
}

// Create inserts a new offer.
func (r *OfferRepo) Create(ctx context.Context, o *domain.Offer) error {
	const q = `
		INSERT INTO offers
			(id, listing_id, contact_id, agent_id, parent_offer_id,
			 offer_amount, currency, status, terms, notes, valid_until)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING created_at, updated_at
	`
	o.ID = uuid.New()
	if o.Currency == "" {
		o.Currency = "AED"
	}
	if o.Status == "" {
		o.Status = domain.OfferSubmitted
	}
	return r.db.QueryRow(ctx, q,
		o.ID, o.ListingID, o.ContactID, o.AgentID, o.ParentOfferID,
		o.OfferAmount, o.Currency, o.Status, o.Terms, o.Notes, o.ValidUntil,
	).Scan(&o.CreatedAt, &o.UpdatedAt)
}

// GetByID returns a single offer with contact info joined.
func (r *OfferRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Offer, error) {
	const q = `
		SELECT ` + offerCols + `,
		       c.id, c.phone_wa, c.full_name, c.email, c.language, c.lead_score,
		       l.title
		FROM offers o
		JOIN contacts c ON c.id = o.contact_id
		JOIN listings l ON l.id = o.listing_id
		WHERE o.id = $1
	`
	var o domain.Offer
	var c domain.Contact
	err := r.db.QueryRow(ctx, q, id).Scan(
		&o.ID, &o.ListingID, &o.ContactID, &o.AgentID, &o.ParentOfferID,
		&o.OfferAmount, &o.Currency, &o.Status, &o.Terms, &o.Notes,
		&o.ValidUntil, &o.DealID, &o.CreatedAt, &o.UpdatedAt,
		&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
		&o.ListingTitle,
	)
	if err != nil {
		return nil, fmt.Errorf("offer not found: %w", err)
	}
	o.Contact = &c
	return &o, nil
}

// ListByListing returns all offers for a listing, newest first.
func (r *OfferRepo) ListByListing(ctx context.Context, listingID uuid.UUID, page, limit int) ([]domain.Offer, int, error) {
	if limit == 0 {
		limit = 50
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM offers WHERE listing_id = $1`, listingID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `
		SELECT ` + offerCols + `,
		       c.id, c.phone_wa, c.full_name, c.email, c.language, c.lead_score,
		       l.title
		FROM offers o
		JOIN contacts c ON c.id = o.contact_id
		JOIN listings l ON l.id = o.listing_id
		WHERE o.listing_id = $1
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, listingID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var offers []domain.Offer
	for rows.Next() {
		var o domain.Offer
		var c domain.Contact
		if err := rows.Scan(
			&o.ID, &o.ListingID, &o.ContactID, &o.AgentID, &o.ParentOfferID,
			&o.OfferAmount, &o.Currency, &o.Status, &o.Terms, &o.Notes,
			&o.ValidUntil, &o.DealID, &o.CreatedAt, &o.UpdatedAt,
			&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
			&o.ListingTitle,
		); err != nil {
			return nil, 0, err
		}
		o.Contact = &c
		offers = append(offers, o)
	}
	return offers, total, nil
}

// List returns all offers with optional filters.
func (r *OfferRepo) List(ctx context.Context, listingID, contactID *uuid.UUID, status string, page, limit int) ([]domain.Offer, int, error) {
	if limit == 0 {
		limit = 50
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	args := []any{}
	conds := []string{"1=1"}
	n := 1

	if listingID != nil {
		conds = append(conds, fmt.Sprintf("o.listing_id = $%d", n))
		args = append(args, *listingID)
		n++
	}
	if contactID != nil {
		conds = append(conds, fmt.Sprintf("o.contact_id = $%d", n))
		args = append(args, *contactID)
		n++
	}
	if status != "" {
		conds = append(conds, fmt.Sprintf("o.status = $%d", n))
		args = append(args, status)
		n++
	}
	where := "WHERE " + joinAnd(conds)

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM offers o "+where, countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	q := fmt.Sprintf(`
		SELECT `+offerCols+`,
		       c.id, c.phone_wa, c.full_name, c.email, c.language, c.lead_score,
		       l.title
		FROM offers o
		JOIN contacts c ON c.id = o.contact_id
		JOIN listings l ON l.id = o.listing_id
		%s
		ORDER BY o.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, n, n+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var offers []domain.Offer
	for rows.Next() {
		var o domain.Offer
		var c domain.Contact
		if err := rows.Scan(
			&o.ID, &o.ListingID, &o.ContactID, &o.AgentID, &o.ParentOfferID,
			&o.OfferAmount, &o.Currency, &o.Status, &o.Terms, &o.Notes,
			&o.ValidUntil, &o.DealID, &o.CreatedAt, &o.UpdatedAt,
			&c.ID, &c.PhoneWA, &c.FullName, &c.Email, &c.Language, &c.LeadScore,
			&o.ListingTitle,
		); err != nil {
			return nil, 0, err
		}
		o.Contact = &c
		offers = append(offers, o)
	}
	return offers, total, nil
}

// UpdateStatus changes an offer's status.
func (r *OfferRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OfferStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE offers SET status=$1, updated_at=NOW() WHERE id=$2`,
		status, id,
	)
	return err
}

// SetDeal links an accepted offer to its created deal.
func (r *OfferRepo) SetDeal(ctx context.Context, offerID, dealID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE offers SET deal_id=$1, status='accepted', updated_at=NOW() WHERE id=$2`,
		dealID, offerID,
	)
	return err
}

// Counter creates a counter-offer linked to the parent.
// The parent offer's status is set to 'countered'.
func (r *OfferRepo) Counter(ctx context.Context, parentID uuid.UUID, o *domain.Offer) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Mark parent as countered
	if _, err = tx.Exec(ctx,
		`UPDATE offers SET status='countered', updated_at=NOW() WHERE id=$1`, parentID,
	); err != nil {
		return err
	}

	// Create counter offer
	o.ParentOfferID = &parentID
	o.Status = domain.OfferSubmitted
	o.ID = uuid.New()
	if o.Currency == "" {
		o.Currency = "AED"
	}
	if err = tx.QueryRow(ctx, `
		INSERT INTO offers
			(id, listing_id, contact_id, agent_id, parent_offer_id,
			 offer_amount, currency, status, terms, notes, valid_until)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING created_at, updated_at
	`,
		o.ID, o.ListingID, o.ContactID, o.AgentID, o.ParentOfferID,
		o.OfferAmount, o.Currency, o.Status, o.Terms, o.Notes, o.ValidUntil,
	).Scan(&o.CreatedAt, &o.UpdatedAt); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Delete hard-deletes an offer.
func (r *OfferRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM offers WHERE id=$1`, id)
	return err
}

// ExpireOlderThan marks submitted/under_review offers whose valid_until has passed.
// Called by a background job.
func (r *OfferRepo) ExpireOlderThan(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE offers SET status='expired', updated_at=NOW()
		WHERE status IN ('submitted','under_review')
		  AND valid_until IS NOT NULL
		  AND valid_until < $1
	`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func joinAnd(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " AND "
		}
		result += p
	}
	return result
}
