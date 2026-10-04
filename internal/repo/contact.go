package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/phone"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type ContactRepo struct {
	db *pgxpool.Pool
}

func NewContactRepo(db *pgxpool.Pool) *ContactRepo {
	return &ContactRepo{db: db}
}

func (r *ContactRepo) List(ctx context.Context, search string, page, limit int) (*domain.PaginatedResult[domain.Contact], error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	offset := (page - 1) * limit
	pattern := "%" + search + "%"

	const countQ = `
		SELECT COUNT(*) FROM contacts
		WHERE company_id = $2 AND ($1 = '' OR full_name ILIKE $1 OR phone_wa ILIKE $1 OR email ILIKE $1)
	`
	var total int
	if err := r.db.QueryRow(ctx, countQ, pattern, cid).Scan(&total); err != nil {
		return nil, fmt.Errorf("count contacts: %w", err)
	}

	const q = `
		SELECT id, phone_wa, full_name, COALESCE(email,''), language, lead_score, assigned_to, created_at, updated_at
		FROM contacts
		WHERE company_id = $4 AND ($1 = '' OR full_name ILIKE $1 OR phone_wa ILIKE $1 OR email ILIKE $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, pattern, limit, offset, cid)
	if err != nil {
		return nil, fmt.Errorf("list contacts: %w", err)
	}
	defer rows.Close()

	var contacts []domain.Contact
	for rows.Next() {
		var c domain.Contact
		if err := rows.Scan(
			&c.ID, &c.PhoneWA, &c.FullName, &c.Email,
			&c.Language, &c.LeadScore, &c.AssignedTo,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan contact: %w", err)
		}
		contacts = append(contacts, c)
	}

	return &domain.PaginatedResult[domain.Contact]{
		Data:  contacts,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (r *ContactRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	const q = `
		SELECT id, phone_wa, full_name, COALESCE(email,''), language, lead_score, assigned_to, created_at, updated_at
		FROM contacts WHERE id = $1 AND company_id = $2
	`
	c := &domain.Contact{}
	err = r.db.QueryRow(ctx, q, id, cid).Scan(
		&c.ID, &c.PhoneWA, &c.FullName, &c.Email,
		&c.Language, &c.LeadScore, &c.AssignedTo,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get contact by id: %w", err)
	}
	return c, nil
}

func (r *ContactRepo) GetByPhone(ctx context.Context, rawPhone string) (*domain.Contact, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	normalized, ok := phone.Normalize(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	const q = `
		SELECT id, phone_wa, full_name, COALESCE(email,''), language, lead_score, assigned_to, created_at, updated_at
		FROM contacts WHERE phone_wa = $1 AND company_id = $2
	`
	c := &domain.Contact{}
	err = r.db.QueryRow(ctx, q, normalized, cid).Scan(
		&c.ID, &c.PhoneWA, &c.FullName, &c.Email,
		&c.Language, &c.LeadScore, &c.AssignedTo,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get contact by phone: %w", err)
	}
	return c, nil
}

func (r *ContactRepo) Create(ctx context.Context, c *domain.Contact) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	normalized, ok := phone.Normalize(c.PhoneWA)
	if !ok {
		return ErrInvalidPhone
	}
	c.PhoneWA = normalized
	const q = `
		INSERT INTO contacts (id, company_id, phone_wa, full_name, email, language, lead_score, assigned_to)
		SELECT $1,$2,$3,$4,$5,$6,$7,$8
		WHERE $8::uuid IS NULL OR EXISTS (SELECT 1 FROM users WHERE id = $8 AND company_id = $2)
		RETURNING created_at, updated_at
	`
	c.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		c.ID, cid, c.PhoneWA, c.FullName, c.Email,
		c.Language, c.LeadScore, c.AssignedTo,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *ContactRepo) Update(ctx context.Context, c *domain.Contact) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	const q = `
		UPDATE contacts
		SET full_name=$1, email=$2, language=$3, lead_score=$4, assigned_to=$5, updated_at=NOW()
		WHERE id=$6 AND company_id=$7
		  AND ($5::uuid IS NULL OR EXISTS (SELECT 1 FROM users WHERE id = $5 AND company_id = $7))
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, q,
		c.FullName, c.Email, c.Language, c.LeadScore, c.AssignedTo, c.ID, cid,
	).Scan(&c.UpdatedAt)
}

func (r *ContactRepo) Delete(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `DELETE FROM contacts WHERE id=$1 AND company_id=$2`, id, cid)
	return err
}

func (r *ContactRepo) UpdateScore(ctx context.Context, id uuid.UUID, score int) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE contacts SET lead_score=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3`,
		score, id, cid,
	)
	return err
}

// Upsert finds or creates a contact by WhatsApp phone number.
// Upsert finds the contact for a phone number or creates it. The number is normalised to E.164
// first, so "971501234567" (WhatsApp) and "+971 50 123 4567" (typed by an agent) are one contact.
//
// It never overwrites information a person entered: an existing contact keeps its name unless
// that name is empty or just the phone number (a placeholder from an earlier import/message).
func (r *ContactRepo) Upsert(ctx context.Context, rawPhone, name string) (*domain.Contact, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	normalized, ok := phone.Normalize(rawPhone)
	if !ok {
		return nil, ErrInvalidPhone
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = normalized
	}
	const q = `
		INSERT INTO contacts (id, company_id, phone_wa, full_name, email)
		VALUES (uuid_generate_v4(), $1, $2, $3, '')
		ON CONFLICT (company_id, phone_wa) DO UPDATE SET full_name = CASE
			WHEN btrim(contacts.full_name) = ''
			  OR regexp_replace(contacts.full_name, '[^0-9]', '', 'g') = regexp_replace(contacts.phone_wa, '[^0-9]', '', 'g')
			THEN EXCLUDED.full_name
			ELSE contacts.full_name END
		RETURNING id, phone_wa, full_name, COALESCE(email,''), language, lead_score, assigned_to, created_at, updated_at
	`
	c := &domain.Contact{}
	err = r.db.QueryRow(ctx, q, cid, normalized, name).Scan(
		&c.ID, &c.PhoneWA, &c.FullName, &c.Email,
		&c.Language, &c.LeadScore, &c.AssignedTo,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert contact: %w", err)
	}
	return c, nil
}
