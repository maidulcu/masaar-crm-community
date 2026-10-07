package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

// Registration failures the caller can turn into a precise client error.
var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrSubdomainTaken     = errors.New("subdomain already taken")
	ErrRegistrationClosed = errors.New("registration is closed")
)

// registrationLock serialises sign-ups (an arbitrary application-wide advisory-lock key), so the
// "first user" / "email free" / "subdomain free" checks below cannot race each other. Sign-up
// is rare and already rate limited, so a global lock costs nothing.
const registrationLock = 7_063_002

// RegistrationInput describes a new workspace and its first admin.
type RegistrationInput struct {
	CompanyName       string
	Subdomain         string // already normalised (lower case)
	TrialDurationDays int
	AdminName         string
	AdminEmail        string
	AdminPasswordHash string

	// OpenRegistration allows sign-ups on a deployment that already has users. When false only
	// the very first account on an empty database is accepted.
	OpenRegistration bool
	// BootstrapCompanyID is the deployment's own company (APP_COMPANY_ID); the first account
	// adopts it. uuid.Nil creates a fresh company even for the first account.
	BootstrapCompanyID uuid.UUID
}

// RegisterTenant creates a company and its admin user in one transaction: if the user cannot be
// created no company is left behind, and concurrent sign-ups cannot both claim "first user".
func (r *CompanyRepo) RegisterTenant(ctx context.Context, in RegistrationInput) (*domain.Company, *domain.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("register tx begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, registrationLock); err != nil {
		return nil, nil, fmt.Errorf("register lock: %w", err)
	}

	var userCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return nil, nil, fmt.Errorf("register count users: %w", err)
	}
	if userCount > 0 && !in.OpenRegistration {
		return nil, nil, ErrRegistrationClosed
	}
	bootstrap := userCount == 0 && in.BootstrapCompanyID != uuid.Nil
	companyID := uuid.New()
	if bootstrap {
		companyID = in.BootstrapCompanyID
	}

	var taken bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = $1)`, in.AdminEmail).Scan(&taken); err != nil {
		return nil, nil, fmt.Errorf("register email check: %w", err)
	}
	if taken {
		return nil, nil, ErrEmailTaken
	}
	// The bootstrap company is being overwritten, so its own current subdomain does not conflict.
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM companies WHERE lower(subdomain) = $1 AND id <> $2)`,
		in.Subdomain, companyID).Scan(&taken); err != nil {
		return nil, nil, fmt.Errorf("register subdomain check: %w", err)
	}
	if taken {
		return nil, nil, ErrSubdomainTaken
	}

	company, err := insertCompany(ctx, tx, companyID, in.CompanyName, in.Subdomain, in.TrialDurationDays, bootstrap)
	if err != nil {
		return nil, nil, mapRegistrationConflict(err)
	}

	user := &domain.User{
		ID:           uuid.New(),
		CompanyID:    company.ID,
		Name:         in.AdminName,
		Email:        in.AdminEmail,
		PasswordHash: in.AdminPasswordHash,
		Role:         domain.RoleAdmin,
		LangPref:     "en",
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (id, company_id, name, email, password_hash, role, lang_pref)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING is_active, created_at`,
		user.ID, user.CompanyID, user.Name, user.Email, user.PasswordHash, user.Role, user.LangPref,
	).Scan(&user.IsActive, &user.CreatedAt); err != nil {
		return nil, nil, mapRegistrationConflict(fmt.Errorf("create user with company: %w", err))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("register tx commit: %w", err)
	}
	return company, user, nil
}

// mapRegistrationConflict turns a unique violation (a sign-up that slipped past the checks, e.g.
// through a different code path writing users) into the matching sentinel.
func mapRegistrationConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch {
		case strings.Contains(pgErr.ConstraintName, "email"):
			return ErrEmailTaken
		case strings.Contains(pgErr.ConstraintName, "subdomain"):
			return ErrSubdomainTaken
		}
	}
	return err
}
