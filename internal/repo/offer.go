package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
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
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	// Contact, listing and agent must all belong to the caller's company.
	const q = `
		INSERT INTO offers
			(id, company_id, listing_id, contact_id, agent_id, parent_offer_id,
			 offer_amount, currency, status, terms, notes, valid_until)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12
		WHERE EXISTS (SELECT 1 FROM contacts WHERE id = $4 AND company_id = $2)
		  AND EXISTS (SELECT 1 FROM listings WHERE id = $3 AND company_id = $2)
		  AND ($5::uuid IS NULL OR EXISTS (SELECT 1 FROM users WHERE id = $5 AND company_id = $2))
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
		o.ID, cid, o.ListingID, o.ContactID, o.AgentID, o.ParentOfferID,
		o.OfferAmount, o.Currency, o.Status, o.Terms, o.Notes, o.ValidUntil,
	).Scan(&o.CreatedAt, &o.UpdatedAt)
}

// GetByID returns a single offer with contact info joined.
func (r *OfferRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Offer, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT ` + offerCols + `,
		       c.id, c.phone_wa, c.full_name, COALESCE(c.email,''), c.language, c.lead_score,
		       l.title
		FROM offers o
		JOIN contacts c ON c.id = o.contact_id AND c.company_id = o.company_id
		JOIN listings l ON l.id = o.listing_id AND l.company_id = o.company_id
		WHERE o.id = $1 AND o.company_id = $2
	`
	var o domain.Offer
	var c domain.Contact
	err = r.db.QueryRow(ctx, q, id, cid).Scan(
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
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, 0, err
	}
	if limit == 0 {
		limit = 50
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM offers WHERE listing_id = $1 AND company_id = $2`, listingID, cid,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `
		SELECT ` + offerCols + `,
		       c.id, c.phone_wa, c.full_name, COALESCE(c.email,''), c.language, c.lead_score,
		       l.title
		FROM offers o
		JOIN contacts c ON c.id = o.contact_id AND c.company_id = o.company_id
		JOIN listings l ON l.id = o.listing_id AND l.company_id = o.company_id
		WHERE o.listing_id = $1 AND o.company_id = $4
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, listingID, limit, offset, cid)
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
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, 0, err
	}
	if limit == 0 {
		limit = 50
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	args := []any{cid}
	conds := []string{"o.company_id = $1"}
	n := 2

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
		       c.id, c.phone_wa, c.full_name, COALESCE(c.email,''), c.language, c.lead_score,
		       l.title
		FROM offers o
		JOIN contacts c ON c.id = o.contact_id AND c.company_id = o.company_id
		JOIN listings l ON l.id = o.listing_id AND l.company_id = o.company_id
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
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE offers SET status=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3`,
		status, id, cid,
	)
	return err
}

// ClaimAcceptance atomically moves an open offer to 'accepted' and returns the status it
// had, so concurrent accept requests cannot both proceed to create a deal. ok is false when
// the offer does not exist in the caller's company or was already accepted/rejected/expired.
// If the caller's follow-up work fails it should restore prev with UpdateStatus.
func (r *OfferRepo) ClaimAcceptance(ctx context.Context, id uuid.UUID) (prev domain.OfferStatus, ok bool, err error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", false, err
	}
	err = r.db.QueryRow(ctx, `
		UPDATE offers o SET status = 'accepted', updated_at = NOW()
		FROM (SELECT id, status FROM offers
		      WHERE id = $1 AND company_id = $2 AND status NOT IN ('accepted','rejected','expired')
		      FOR UPDATE) old
		WHERE o.id = old.id
		RETURNING old.status`, id, cid).Scan(&prev)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return prev, true, nil
}

// SetDeal links an accepted offer to its created deal.
func (r *OfferRepo) SetDeal(ctx context.Context, offerID, dealID uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE offers SET deal_id=$1, status='accepted', updated_at=NOW()
		 WHERE id=$2 AND company_id=$3
		   AND EXISTS (SELECT 1 FROM deals WHERE id=$1 AND company_id=$3)`,
		dealID, offerID, cid,
	)
	return err
}

// Counter creates a counter-offer linked to the parent.
// The parent offer's status is set to 'countered'.
func (r *OfferRepo) Counter(ctx context.Context, parentID uuid.UUID, o *domain.Offer) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Mark parent as countered
	tag, err := tx.Exec(ctx,
		`UPDATE offers SET status='countered', updated_at=NOW() WHERE id=$1 AND company_id=$2`, parentID, cid,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("offer not found")
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
			(id, company_id, listing_id, contact_id, agent_id, parent_offer_id,
			 offer_amount, currency, status, terms, notes, valid_until)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12
		WHERE EXISTS (SELECT 1 FROM contacts WHERE id = $4 AND company_id = $2)
		  AND EXISTS (SELECT 1 FROM listings WHERE id = $3 AND company_id = $2)
		  AND ($5::uuid IS NULL OR EXISTS (SELECT 1 FROM users WHERE id = $5 AND company_id = $2))
		RETURNING created_at, updated_at
	`,
		o.ID, cid, o.ListingID, o.ContactID, o.AgentID, o.ParentOfferID,
		o.OfferAmount, o.Currency, o.Status, o.Terms, o.Notes, o.ValidUntil,
	).Scan(&o.CreatedAt, &o.UpdatedAt); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Delete hard-deletes an offer.
func (r *OfferRepo) Delete(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `DELETE FROM offers WHERE id=$1 AND company_id=$2`, id, cid)
	return err
}

// ExpireOlderThan marks submitted/under_review offers whose valid_until has passed.
// Called by a background job. Intentionally NOT company-scoped: it is a system-wide
// sweep that only flips overdue offers to 'expired' and returns no tenant data.
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
