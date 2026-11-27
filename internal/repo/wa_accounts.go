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

// WAAccountRepo stores per-company WhatsApp accounts. A company may have more than one
// (a Cloud API number and a BSP number, for example), and each thread belongs to one account.
type WAAccountRepo struct {
	db *pgxpool.Pool
}

func NewWAAccountRepo(db *pgxpool.Pool) *WAAccountRepo {
	return &WAAccountRepo{db: db}
}

// waAccountColumns is the read order shared by scanWAAccount callers.
const waAccountColumns = `
	id, company_id, provider, status,
	COALESCE(bsp_slug, ''), COALESCE(bsp_client_ref, ''),
	COALESCE(phone_e164, ''), COALESCE(display_name, ''),
	COALESCE(waba_id, ''), COALESCE(messaging_account_id, ''), COALESCE(waac_id, ''),
	COALESCE(phone_number_id, ''), COALESCE(external_id, ''), COALESCE(session_ref, ''),
	COALESCE(billing_mode, ''), linked_by, linked_at, last_seen_at,
	last_error, risk_flags, created_at, updated_at
`

type waAccountScanner interface {
	Scan(dest ...any) error
}

func scanWAAccount(s waAccountScanner) (*domain.WAAccount, error) {
	a := &domain.WAAccount{}
	if err := s.Scan(
		&a.ID, &a.CompanyID, &a.Provider, &a.Status,
		&a.BSPSlug, &a.BSPClientRef,
		&a.PhoneE164, &a.DisplayName,
		&a.WABAID, &a.MessagingAccountID, &a.WAACID,
		&a.PhoneNumberID, &a.ExternalID, &a.SessionRef,
		&a.BillingMode, &a.LinkedBy, &a.LinkedAt, &a.LastSeenAt,
		&a.LastError, &a.RiskFlags, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return a, nil
}

// Create stores a new account for the caller's company. Provider and status must already be set;
// an empty status defaults to pending so a row can be created before the provider is contacted.
func (r *WAAccountRepo) Create(ctx context.Context, a *domain.WAAccount) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if a.Provider == "" {
		return fmt.Errorf("create whatsapp account: provider is required")
	}
	if a.Status == "" {
		a.Status = domain.WAAccountPending
	}
	riskFlags := string(a.RiskFlags)
	if riskFlags == "" {
		riskFlags = "{}"
	}
	const q = `
		INSERT INTO wa_accounts (
			company_id, provider, bsp_slug, bsp_client_ref, status, phone_e164, display_name,
			waba_id, messaging_account_id, waac_id, phone_number_id, external_id, session_ref,
			billing_mode, linked_by, linked_at, risk_flags
		) VALUES (
			$1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, NULLIF($6, ''), NULLIF($7, ''),
			NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''),
			NULLIF($14, ''), $15, $16, $17::jsonb
		)
		RETURNING id, company_id, created_at, updated_at
	`
	err = r.db.QueryRow(ctx, q,
		cid, a.Provider, a.BSPSlug, a.BSPClientRef, a.Status, a.PhoneE164, a.DisplayName,
		a.WABAID, a.MessagingAccountID, a.WAACID, a.PhoneNumberID, a.ExternalID, a.SessionRef,
		a.BillingMode, a.LinkedBy, a.LinkedAt, riskFlags,
	).Scan(&a.ID, &a.CompanyID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create whatsapp account: %w", err)
	}
	return nil
}

// List returns the caller's accounts, newest first.
func (r *WAAccountRepo) List(ctx context.Context) ([]domain.WAAccount, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT `+waAccountColumns+`
		FROM wa_accounts WHERE company_id = $1 ORDER BY created_at DESC`, cid)
	if err != nil {
		return nil, fmt.Errorf("list whatsapp accounts: %w", err)
	}
	defer rows.Close()

	var accounts []domain.WAAccount
	for rows.Next() {
		a, err := scanWAAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan whatsapp account: %w", err)
		}
		accounts = append(accounts, *a)
	}
	return accounts, rows.Err()
}

// Get returns one account belonging to the caller's company.
func (r *WAAccountRepo) Get(ctx context.Context, id uuid.UUID) (*domain.WAAccount, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	a, err := scanWAAccount(r.db.QueryRow(ctx, `SELECT `+waAccountColumns+`
		FROM wa_accounts WHERE id = $1 AND company_id = $2`, id, cid))
	if err != nil {
		return nil, fmt.Errorf("get whatsapp account: %w", err)
	}
	return a, nil
}

// ResolveByPhoneNumberID looks an account up for one company, for signed inbound webhooks.
// Callers must verify the provider's signature first; the account row is the only source of
// the company, never the payload. The company is passed in because the legacy deployment-wide
// number can legitimately appear under more than one company.
func (r *WAAccountRepo) ResolveByPhoneNumberID(ctx context.Context, companyID uuid.UUID, phoneNumberID string) (*domain.WAAccount, error) {
	if phoneNumberID == "" {
		return nil, fmt.Errorf("resolve whatsapp account: phone number id is empty")
	}
	a, err := scanWAAccount(r.db.QueryRow(ctx, `SELECT `+waAccountColumns+`
		FROM wa_accounts WHERE company_id = $1 AND phone_number_id = $2`, companyID, phoneNumberID))
	if err != nil {
		return nil, fmt.Errorf("resolve whatsapp account by phone number id: %w", err)
	}
	return a, nil
}

// EnsureDeploymentAccount makes the environment-configured Cloud API number visible as an
// account row for its company. It is idempotent: startup can call it on every boot.
func (r *WAAccountRepo) EnsureDeploymentAccount(ctx context.Context, companyID uuid.UUID, phoneNumberID, displayName string) (*domain.WAAccount, error) {
	if phoneNumberID == "" {
		return nil, fmt.Errorf("ensure whatsapp account: phone number id is required")
	}
	const q = `
		INSERT INTO wa_accounts (company_id, provider, status, phone_number_id, display_name, linked_at)
		VALUES ($1, 'cloud_api', 'connected', $2, NULLIF($3, ''), NOW())
		ON CONFLICT (company_id, phone_number_id) WHERE phone_number_id IS NOT NULL DO NOTHING
		RETURNING ` + waAccountColumns
	a, err := scanWAAccount(r.db.QueryRow(ctx, q, companyID, phoneNumberID, displayName))
	if err == nil {
		return a, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("ensure whatsapp account: %w", err)
	}

	// The number exists already: this is a later startup.
	a, err = r.ResolveByPhoneNumberID(ctx, companyID, phoneNumberID)
	if err != nil {
		return nil, err
	}
	if displayName != "" && a.DisplayName == "" {
		if _, err := r.db.Exec(ctx,
			`UPDATE wa_accounts SET display_name = $2, updated_at = NOW() WHERE id = $1`,
			a.ID, displayName); err != nil {
			return nil, fmt.Errorf("ensure whatsapp account: set display name: %w", err)
		}
		a.DisplayName = displayName
	}
	return a, nil
}

// UpdateStatus records a lifecycle transition and the last error, touching last_seen_at when
// the account becomes connected. Empty lastError clears the previous error.
func (r *WAAccountRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.WAAccountStatus, lastError string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE wa_accounts
		SET status = $3,
		    last_error = $4,
		    last_seen_at = CASE WHEN $3 = 'connected' THEN NOW() ELSE last_seen_at END,
		    updated_at = NOW()
		WHERE id = $1 AND company_id = $2`, id, cid, status, lastError)
	if err != nil {
		return fmt.Errorf("update whatsapp account status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// LinkThreads attaches the company's still-unlinked threads to an account. The migration does
// this for existing data; onboarding uses it when an account is created after threads exist.
func (r *WAAccountRepo) LinkThreads(ctx context.Context, accountID uuid.UUID) (int64, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return 0, err
	}
	a, err := r.Get(ctx, accountID)
	if err != nil {
		return 0, err
	}
	if a.PhoneNumberID == "" {
		return 0, nil
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE whatsapp_threads
		SET account_id = $1
		WHERE company_id = $2 AND account_id IS NULL AND wa_account_id = $3`,
		accountID, cid, a.PhoneNumberID)
	if err != nil {
		return 0, fmt.Errorf("link whatsapp threads: %w", err)
	}
	return tag.RowsAffected(), nil
}

// UpdateLastSeen marks an account as seen now, for health reporting.
func (r *WAAccountRepo) UpdateLastSeen(ctx context.Context, id uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE wa_accounts SET last_seen_at = $3, updated_at = NOW() WHERE id = $1 AND company_id = $2`,
		id, cid, time.Now())
	return err
}
