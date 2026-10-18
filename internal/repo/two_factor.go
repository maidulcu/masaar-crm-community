package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/totp"
)

// TwoFactorRepo stores users' authenticator secrets and backup codes. Login runs before a
// company is bound to the context, so these methods take the user id directly; callers
// have authenticated the user (password, magic link or SMS code) or hold their session.
type TwoFactorRepo struct {
	db *pgxpool.Pool
}

func NewTwoFactorRepo(db *pgxpool.Pool) *TwoFactorRepo { return &TwoFactorRepo{db: db} }

// TwoFactorState is a user's two-factor status.
type TwoFactorState struct {
	Enabled         bool   `json:"enabled"`
	RequiredByAdmin bool   `json:"required_by_company"`
	BackupCodesLeft int    `json:"backup_codes_left"`
	SealedSecret    string `json:"-"`
	LastStep        int64  `json:"-"`
}

// State returns a user's two-factor status and whether their company requires it.
func (r *TwoFactorRepo) State(ctx context.Context, userID uuid.UUID) (*TwoFactorState, error) {
	s := &TwoFactorState{}
	err := r.db.QueryRow(ctx, `
		SELECT u.totp_enabled, COALESCE(cs.require_2fa, FALSE), u.totp_secret, u.totp_last_step,
		       (SELECT COUNT(*) FROM user_backup_codes b WHERE b.user_id = u.id)
		  FROM users u LEFT JOIN company_settings cs ON cs.company_id = u.company_id
		 WHERE u.id = $1`, userID).Scan(&s.Enabled, &s.RequiredByAdmin, &s.SealedSecret, &s.LastStep, &s.BackupCodesLeft)
	return s, err
}

// Enable stores the (sealed) secret, turns two-factor on, records the step of the code
// that confirmed it, and replaces the backup codes.
func (r *TwoFactorRepo) Enable(ctx context.Context, userID uuid.UUID, sealedSecret string, step int64, backupCodes []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	tag, err := tx.Exec(ctx, `UPDATE users SET totp_secret = $1, totp_enabled = TRUE, totp_enabled_at = NOW(), totp_last_step = $2
		WHERE id = $3`, sealedSecret, step, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err := replaceBackupCodes(ctx, tx, userID, backupCodes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func replaceBackupCodes(ctx context.Context, tx pgx.Tx, userID uuid.UUID, codes []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM user_backup_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, c := range codes {
		if _, err := tx.Exec(ctx, `INSERT INTO user_backup_codes (user_id, code_hash) VALUES ($1, $2)`, userID, totp.HashBackupCode(c)); err != nil {
			return err
		}
	}
	return nil
}

// ReplaceBackupCodes issues a new set of backup codes.
func (r *TwoFactorRepo) ReplaceBackupCodes(ctx context.Context, userID uuid.UUID, codes []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := replaceBackupCodes(ctx, tx, userID, codes); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Disable turns two-factor off and deletes the secret and backup codes.
func (r *TwoFactorRepo) Disable(ctx context.Context, userID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `UPDATE users SET totp_secret = '', totp_enabled = FALSE, totp_enabled_at = NULL, totp_last_step = 0
		WHERE id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_backup_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ErrCodeReused is returned when a code's time step was already used.
var ErrCodeReused = errors.New("code already used")

// AdvanceStep records a code's time step as used. It fails when that step (or a later
// one) was already used, so two concurrent logins cannot both use one code.
func (r *TwoFactorRepo) AdvanceStep(ctx context.Context, userID uuid.UUID, step int64) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET totp_last_step = $1 WHERE id = $2 AND totp_last_step < $1`, step, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCodeReused
	}
	return nil
}

// UseBackupCode consumes a backup code; it reports false when the code is unknown or used.
func (r *TwoFactorRepo) UseBackupCode(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_backup_codes WHERE user_id = $1 AND code_hash = $2`, userID, totp.HashBackupCode(code))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// SetCompanyRequirement turns the company-wide requirement on or off.
func (r *TwoFactorRepo) SetCompanyRequirement(ctx context.Context, companyID uuid.UUID, required bool) error {
	tag, err := r.db.Exec(ctx, `UPDATE company_settings SET require_2fa = $1 WHERE company_id = $2`, required, companyID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CompanyRequirement reports whether the company requires two-factor authentication, and
// how many of its active users have not enabled it.
func (r *TwoFactorRepo) CompanyRequirement(ctx context.Context, companyID uuid.UUID) (required bool, withoutTwoFactor int, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE((SELECT require_2fa FROM company_settings WHERE company_id = $1), FALSE),
		       (SELECT COUNT(*) FROM users WHERE company_id = $1 AND is_active AND NOT totp_enabled)`, companyID).Scan(&required, &withoutTwoFactor)
	return required, withoutTwoFactor, err
}
