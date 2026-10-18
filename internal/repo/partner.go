package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// Partner API storage. A partner sees only the companies whose external_source is its client
// id: every method here filters by that instead of tenant.From, because the caller is a partner
// system, not a member of one company.

var (
	ErrPartnerNotFound = errors.New("partner: not found")
	ErrSeatLimit       = errors.New("company has no free seat")
	ErrCompanyExists   = errors.New("partner: company already exists")
)

// IsSeatLimit reports the error the users_enforce_company_seats trigger raises.
func IsSeatLimit(err error) bool {
	if errors.Is(err, ErrSeatLimit) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "P0001" && pgErr.Message == "seat_limit"
}

func mapPartnerErr(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case err == nil:
		return nil
	case IsSeatLimit(err):
		return ErrSeatLimit
	case errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "idx_companies_external_ref":
		return ErrCompanyExists
	}
	return mapRegistrationConflict(err)
}

// PartnerRepo provisions companies and users for partner clients.
type PartnerRepo struct{ db *pgxpool.Pool }

func NewPartnerRepo(db *pgxpool.Pool) *PartnerRepo { return &PartnerRepo{db: db} }

// PartnerCompany is a company as a partner sees it.
type PartnerCompany struct {
	ID          uuid.UUID `json:"id"`
	ExternalRef string    `json:"external_ref"`
	Name        string    `json:"name"`
	Plan        string    `json:"plan"`
	Active      bool      `json:"active"`
	MaxUsers    *int      `json:"max_users"`
	ActiveUsers int       `json:"active_users"`
	CreatedAt   time.Time `json:"created_at"`
}

// PartnerUser is a user as a partner sees it.
type PartnerUser struct {
	ID          uuid.UUID   `json:"id"`
	ExternalRef string      `json:"external_ref"`
	Name        string      `json:"name"`
	Email       string      `json:"email"`
	Role        domain.Role `json:"role"`
	Active      bool        `json:"active"`
	LangPref    string      `json:"lang_pref"`
}

const partnerCompanyCols = `c.id, c.external_ref, c.name, c.plan, c.is_active, c.max_users, c.created_at,
	(SELECT COUNT(*) FROM users u WHERE u.company_id = c.id AND u.is_active)`

func scanPartnerCompany(row pgx.Row) (*PartnerCompany, error) {
	pc := &PartnerCompany{}
	err := row.Scan(&pc.ID, &pc.ExternalRef, &pc.Name, &pc.Plan, &pc.Active, &pc.MaxUsers, &pc.CreatedAt, &pc.ActiveUsers)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPartnerNotFound
	}
	return pc, err
}

// GetCompany returns the partner's company with reference ref.
func (r *PartnerRepo) GetCompany(ctx context.Context, clientID, ref string) (*PartnerCompany, error) {
	return scanPartnerCompany(r.db.QueryRow(ctx,
		`SELECT `+partnerCompanyCols+` FROM companies c WHERE c.external_source = $1 AND c.external_ref = $2`,
		clientID, ref))
}

// PartnerCompanyInput creates a company and its first admin.
type PartnerCompanyInput struct {
	ClientID, Ref, Name, Plan string
	MaxUsers                  *int
	Active                    bool
	Admin                     PartnerUserInput
	// LeadKey is an API key created with the company, in the same transaction.
	LeadKeyName, LeadKeyHash, LeadKeyPrefix, LeadKeyScopes string
}

// PartnerUserInput creates or updates a user. PasswordHash is used only on creation; partner
// users sign in through login links, so callers pass a hash of a random secret.
type PartnerUserInput struct {
	Ref, Name, Email, LangPref string
	Role                       domain.Role
	Active                     bool
	PasswordHash               string
}

var slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

// partnerSubdomain derives a free-looking subdomain from the company name plus a random suffix,
// matching the sign-up rules (3–40 characters, letters, digits and inner hyphens).
func partnerSubdomain(name string) string {
	base := strings.Trim(slugUnsafe.ReplaceAllString(strings.ToLower(name), "-"), "-")
	if len(base) > 30 {
		base = strings.Trim(base[:30], "-")
	}
	if base == "" {
		base = "company"
	}
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return base + "-" + hex.EncodeToString(b)
}

// CreateCompany creates the company (no trial: billing stays with the partner) and its admin in
// one transaction. ErrEmailTaken means the admin's email already has an account.
func (r *PartnerRepo) CreateCompany(ctx context.Context, in PartnerCompanyInput) (*PartnerCompany, *PartnerUser, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Same lock as sign-up: email and subdomain checks cannot race other registrations.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, registrationLock); err != nil {
		return nil, nil, err
	}
	var taken bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = $1)`, in.Admin.Email).Scan(&taken); err != nil {
		return nil, nil, err
	}
	if taken {
		return nil, nil, ErrEmailTaken
	}
	subdomain := ""
	for i := 0; i < 5 && subdomain == ""; i++ {
		cand := partnerSubdomain(in.Name)
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM companies WHERE lower(subdomain) = $1)`, cand).Scan(&taken); err != nil {
			return nil, nil, err
		}
		if !taken {
			subdomain = cand
		}
	}
	if subdomain == "" {
		return nil, nil, ErrSubdomainTaken
	}

	company, err := insertCompany(ctx, tx, uuid.New(), in.Name, subdomain, 0, false)
	if err != nil {
		return nil, nil, mapRegistrationConflict(err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE companies SET plan = $2, on_trial = false, trial_started_at = NULL, trial_ends_at = NULL,
		       is_active = $3, max_users = $4, external_source = $5, external_ref = $6
		WHERE id = $1`,
		company.ID, in.Plan, in.Active, in.MaxUsers, in.ClientID, in.Ref); err != nil {
		return nil, nil, mapPartnerErr(err)
	}

	admin := in.Admin
	admin.Role, admin.Active = domain.RoleAdmin, true
	user, err := insertPartnerUser(ctx, tx, company.ID, admin)
	if err != nil {
		return nil, nil, err
	}
	if in.LeadKeyHash != "" {
		if _, err := tx.Exec(ctx, `INSERT INTO api_keys (id, company_id, name, key_hash, key_prefix, scopes)
			VALUES ($1, $2, $3, $4, $5, $6)`, uuid.New(), company.ID, in.LeadKeyName, in.LeadKeyHash, in.LeadKeyPrefix, in.LeadKeyScopes); err != nil {
			return nil, nil, fmt.Errorf("create lead key: %w", err)
		}
	}
	pc, err := scanPartnerCompany(tx.QueryRow(ctx, `SELECT `+partnerCompanyCols+` FROM companies c WHERE c.id = $1`, company.ID))
	if err != nil {
		return nil, nil, err
	}
	return pc, user, tx.Commit(ctx)
}

func insertPartnerUser(ctx context.Context, tx pgx.Tx, companyID uuid.UUID, in PartnerUserInput) (*PartnerUser, error) {
	u := &PartnerUser{ID: uuid.New(), ExternalRef: in.Ref, Name: in.Name, Email: in.Email, Role: in.Role, Active: in.Active, LangPref: in.LangPref}
	_, err := tx.Exec(ctx, `
		INSERT INTO users (id, company_id, name, email, password_hash, role, lang_pref, is_active, external_ref)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		u.ID, companyID, u.Name, u.Email, in.PasswordHash, u.Role, u.LangPref, u.Active, u.ExternalRef)
	if err != nil {
		return nil, mapPartnerErr(fmt.Errorf("insert partner user: %w", err))
	}
	return u, nil
}

// PartnerCompanyUpdate changes a company; nil fields stay as they are. ClearMaxUsers removes
// the seat limit.
type PartnerCompanyUpdate struct {
	Name, Plan    *string
	MaxUsers      *int
	ClearMaxUsers bool
	Active        *bool
}

// UpdateCompany applies upd to the partner's company ref.
func (r *PartnerRepo) UpdateCompany(ctx context.Context, clientID, ref string, upd PartnerCompanyUpdate) (*PartnerCompany, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE companies SET
		  name      = COALESCE($3, name),
		  plan      = COALESCE($4, plan),
		  max_users = CASE WHEN $6 THEN NULL ELSE COALESCE($5, max_users) END,
		  is_active = COALESCE($7, is_active)
		WHERE external_source = $1 AND external_ref = $2`,
		clientID, ref, upd.Name, upd.Plan, upd.MaxUsers, upd.ClearMaxUsers, upd.Active)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrPartnerNotFound
	}
	if upd.Name != nil {
		if _, err := r.db.Exec(ctx, `UPDATE company_settings SET name = $2 WHERE company_id = (
			SELECT id FROM companies WHERE external_source = $1 AND external_ref = $3)`, clientID, *upd.Name, ref); err != nil {
			return nil, err
		}
	}
	return r.GetCompany(ctx, clientID, ref)
}

// GetUser returns the user with reference ref in company companyID.
func (r *PartnerRepo) GetUser(ctx context.Context, companyID uuid.UUID, ref string) (*PartnerUser, error) {
	u := &PartnerUser{}
	err := r.db.QueryRow(ctx, `
		SELECT id, external_ref, name, email, role, is_active, lang_pref FROM users
		WHERE company_id = $1 AND external_ref = $2`, companyID, ref).
		Scan(&u.ID, &u.ExternalRef, &u.Name, &u.Email, &u.Role, &u.Active, &u.LangPref)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPartnerNotFound
	}
	return u, err
}

// UpsertUser creates the user ref in companyID, or updates its name, role, language and active
// flag (email is set only on creation). It returns ErrSeatLimit when the change would put more
// active users in the company than max_users, and ErrEmailTaken for an email already in use.
func (r *PartnerRepo) UpsertUser(ctx context.Context, companyID uuid.UUID, in PartnerUserInput) (*PartnerUser, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Lock the company row so two concurrent calls cannot both take the last seat.
	var maxUsers *int
	var active int
	if err := tx.QueryRow(ctx, `SELECT max_users FROM companies WHERE id = $1 FOR UPDATE`, companyID).Scan(&maxUsers); err != nil {
		return nil, false, err
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE company_id = $1 AND is_active`, companyID).Scan(&active); err != nil {
		return nil, false, err
	}

	var existing PartnerUser
	err = tx.QueryRow(ctx, `SELECT id, is_active FROM users WHERE company_id = $1 AND external_ref = $2`, companyID, in.Ref).
		Scan(&existing.ID, &existing.Active)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if in.Active && maxUsers != nil && active >= *maxUsers {
			return nil, false, ErrSeatLimit
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, registrationLock); err != nil {
			return nil, false, err
		}
		var taken bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = $1)`, in.Email).Scan(&taken); err != nil {
			return nil, false, err
		}
		if taken {
			return nil, false, ErrEmailTaken
		}
		u, err := insertPartnerUser(ctx, tx, companyID, in)
		if err != nil {
			return nil, false, err
		}
		return u, true, tx.Commit(ctx)
	case err != nil:
		return nil, false, err
	}

	if in.Active && !existing.Active && maxUsers != nil && active >= *maxUsers {
		return nil, false, ErrSeatLimit
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET name = $3, role = $4, lang_pref = $5, is_active = $6
		WHERE company_id = $1 AND id = $2`,
		companyID, existing.ID, in.Name, in.Role, in.LangPref, in.Active); err != nil {
		return nil, false, mapPartnerErr(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	u, err := r.GetUser(ctx, companyID, in.Ref)
	return u, false, err
}

// PartnerTokenRepo stores partner access tokens (hashed).
type PartnerTokenRepo struct{ db *pgxpool.Pool }

func NewPartnerTokenRepo(db *pgxpool.Pool) *PartnerTokenRepo { return &PartnerTokenRepo{db: db} }

// Issue stores a new token hash. Expired tokens are cleaned up opportunistically.
func (r *PartnerTokenRepo) Issue(ctx context.Context, tokenHash, clientID, scopes string, expiresAt time.Time) error {
	_, _ = r.db.Exec(ctx, `DELETE FROM partner_tokens WHERE expires_at < NOW() - INTERVAL '1 day'`)
	_, err := r.db.Exec(ctx, `INSERT INTO partner_tokens (token_hash, client_id, scopes, expires_at) VALUES ($1,$2,$3,$4)`,
		tokenHash, clientID, scopes, expiresAt)
	return err
}

// Validate returns the client and scopes of a live (unexpired, unrevoked) token.
func (r *PartnerTokenRepo) Validate(ctx context.Context, tokenHash string) (clientID, scopes string, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT client_id, scopes FROM partner_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`, tokenHash).Scan(&clientID, &scopes)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrPartnerNotFound
	}
	return clientID, scopes, err
}

// Revoke revokes the token if it belongs to clientID (RFC 7009: unknown tokens are not an error).
func (r *PartnerTokenRepo) Revoke(ctx context.Context, tokenHash, clientID string) error {
	_, err := r.db.Exec(ctx, `UPDATE partner_tokens SET revoked_at = NOW()
		WHERE token_hash = $1 AND client_id = $2 AND revoked_at IS NULL`, tokenHash, clientID)
	return err
}
