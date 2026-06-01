package repo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApiKey struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	Name      string
	KeyHash   string    // Never expose
	KeyPrefix string    // Safe to show in UI (first 8 chars)
	Scopes    string    // Comma-separated: "lead:create,lead:read"
	LastUsedAt *time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

type ApiKeyRepo struct {
	db *pgxpool.Pool
}

func NewApiKeyRepo(db *pgxpool.Pool) *ApiKeyRepo {
	return &ApiKeyRepo{db: db}
}

// GenerateKey creates a new API key in format sk_live_[random32chars] and returns both plaintext and hash.
func (r *ApiKeyRepo) GenerateKey() (plaintext, hash, prefix string, err error) {
	// Generate 24 random bytes → 48 hex chars
	randBytes := make([]byte, 24)
	if _, err := rand.Read(randBytes); err != nil {
		return "", "", "", fmt.Errorf("random generation failed: %w", err)
	}

	plaintext = "sk_live_" + hex.EncodeToString(randBytes)
	hashSum := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(hashSum[:])
	prefix = plaintext[:16] // "sk_live_abcd1234"

	return plaintext, hash, prefix, nil
}

// Create inserts a new API key (hash is pre-computed by caller).
func (r *ApiKeyRepo) Create(ctx context.Context, companyID uuid.UUID, name, keyHash, keyPrefix, scopes string) (uuid.UUID, error) {
	id := uuid.New()
	err := r.db.QueryRow(
		ctx,
		`INSERT INTO api_keys (id, company_id, name, key_hash, key_prefix, scopes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		id, companyID, name, keyHash, keyPrefix, scopes,
	).Scan(&id)
	return id, err
}

// ValidateKey checks if key_hash matches a non-revoked key, returns the ApiKey and updates last_used_at.
func (r *ApiKeyRepo) ValidateKey(ctx context.Context, keyHash string) (*ApiKey, error) {
	key := &ApiKey{}
	err := r.db.QueryRow(
		ctx,
		`SELECT id, company_id, name, key_hash, key_prefix, scopes, last_used_at, created_at, revoked_at
		 FROM api_keys
		 WHERE key_hash = $1 AND revoked_at IS NULL`,
		keyHash,
	).Scan(&key.ID, &key.CompanyID, &key.Name, &key.KeyHash, &key.KeyPrefix, &key.Scopes, &key.LastUsedAt, &key.CreatedAt, &key.RevokedAt)

	if err != nil {
		return nil, fmt.Errorf("api key validation failed: %w", err)
	}

	// Update last_used_at (best-effort, don't fail if this errors)
	_, _ = r.db.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, key.ID)

	return key, nil
}

// List returns all non-revoked API keys for a company.
func (r *ApiKeyRepo) List(ctx context.Context, companyID uuid.UUID) ([]ApiKey, error) {
	rows, err := r.db.Query(
		ctx,
		`SELECT id, company_id, name, key_hash, key_prefix, scopes, last_used_at, created_at, revoked_at
		 FROM api_keys
		 WHERE company_id = $1 AND revoked_at IS NULL
		 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []ApiKey
	for rows.Next() {
		key := ApiKey{}
		if err := rows.Scan(&key.ID, &key.CompanyID, &key.Name, &key.KeyHash, &key.KeyPrefix, &key.Scopes, &key.LastUsedAt, &key.CreatedAt, &key.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// Revoke marks a key as revoked.
func (r *ApiKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE api_keys SET revoked_at = NOW() WHERE id = $1`, id)
	return err
}

// HasScope checks if an API key has a required scope.
func (k *ApiKey) HasScope(requiredScope string) bool {
	if k.Scopes == "" {
		return false
	}
	for _, scope := range strings.Split(k.Scopes, ",") {
		if strings.TrimSpace(scope) == requiredScope {
			return true
		}
	}
	return false
}
