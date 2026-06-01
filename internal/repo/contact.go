package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
)

type ContactRepo struct {
	db *pgxpool.Pool
}

func NewContactRepo(db *pgxpool.Pool) *ContactRepo {
	return &ContactRepo{db: db}
}

func (r *ContactRepo) List(ctx context.Context, search string, page, limit int) (*domain.PaginatedResult[domain.Contact], error) {
	offset := (page - 1) * limit
	pattern := "%" + search + "%"

	const countQ = `
		SELECT COUNT(*) FROM contacts
		WHERE ($1 = '' OR full_name ILIKE $1 OR phone_wa ILIKE $1 OR email ILIKE $1)
	`
	var total int
	if err := r.db.QueryRow(ctx, countQ, pattern).Scan(&total); err != nil {
		return nil, fmt.Errorf("count contacts: %w", err)
	}

	const q = `
		SELECT id, phone_wa, full_name, email, language, lead_score, assigned_to, created_at, updated_at
		FROM contacts
		WHERE ($1 = '' OR full_name ILIKE $1 OR phone_wa ILIKE $1 OR email ILIKE $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, pattern, limit, offset)
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
	const q = `
		SELECT id, phone_wa, full_name, email, language, lead_score, assigned_to, created_at, updated_at
		FROM contacts WHERE id = $1
	`
	c := &domain.Contact{}
	err := r.db.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.PhoneWA, &c.FullName, &c.Email,
		&c.Language, &c.LeadScore, &c.AssignedTo,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get contact by id: %w", err)
	}
	return c, nil
}

func (r *ContactRepo) GetByPhone(ctx context.Context, phone string) (*domain.Contact, error) {
	const q = `
		SELECT id, phone_wa, full_name, email, language, lead_score, assigned_to, created_at, updated_at
		FROM contacts WHERE phone_wa = $1
	`
	c := &domain.Contact{}
	err := r.db.QueryRow(ctx, q, phone).Scan(
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
	const q = `
		INSERT INTO contacts (id, phone_wa, full_name, email, language, lead_score, assigned_to)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING created_at, updated_at
	`
	c.ID = uuid.New()
	return r.db.QueryRow(ctx, q,
		c.ID, c.PhoneWA, c.FullName, c.Email,
		c.Language, c.LeadScore, c.AssignedTo,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *ContactRepo) Update(ctx context.Context, c *domain.Contact) error {
	const q = `
		UPDATE contacts
		SET full_name=$1, email=$2, language=$3, lead_score=$4, assigned_to=$5, updated_at=NOW()
		WHERE id=$6
		RETURNING updated_at
	`
	return r.db.QueryRow(ctx, q,
		c.FullName, c.Email, c.Language, c.LeadScore, c.AssignedTo, c.ID,
	).Scan(&c.UpdatedAt)
}

func (r *ContactRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM contacts WHERE id=$1`, id)
	return err
}

func (r *ContactRepo) UpdateScore(ctx context.Context, id uuid.UUID, score int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE contacts SET lead_score=$1, updated_at=NOW() WHERE id=$2`,
		score, id,
	)
	return err
}

// Upsert finds or creates a contact by WhatsApp phone number.
func (r *ContactRepo) Upsert(ctx context.Context, phone, name string) (*domain.Contact, error) {
	const q = `
		INSERT INTO contacts (id, phone_wa, full_name)
		VALUES (uuid_generate_v4(), $1, $2)
		ON CONFLICT (phone_wa) DO UPDATE SET full_name = EXCLUDED.full_name
		RETURNING id, phone_wa, full_name, email, language, lead_score, assigned_to, created_at, updated_at
	`
	c := &domain.Contact{}
	err := r.db.QueryRow(ctx, q, phone, name).Scan(
		&c.ID, &c.PhoneWA, &c.FullName, &c.Email,
		&c.Language, &c.LeadScore, &c.AssignedTo,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert contact: %w", err)
	}
	return c, nil
}
