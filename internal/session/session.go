// Package session revokes every outstanding login of a user at once.
//
// Access tokens are stateless JWTs and refresh tokens are individual Redis keys, so there is no
// list of a user's sessions to delete. Instead we store a per-user cutoff time: any token issued
// before it is rejected. The cutoff is set when something happens that must end existing logins —
// the account is deactivated or deleted, its role changes, or its password changes or is reset.
package session

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// cutoffTTL must outlive the longest-lived token (the refresh token, 7 days by default).
const cutoffTTL = 31 * 24 * time.Hour

func key(userID uuid.UUID) string { return "session_cutoff:" + userID.String() }

// RevokeAll invalidates every token issued to userID before now.
func RevokeAll(ctx context.Context, rdb *redis.Client, userID uuid.UUID) error {
	return rdb.Set(ctx, key(userID), strconv.FormatInt(time.Now().Unix(), 10), cutoffTTL).Err()
}

// IssuedBeforeCutoff reports whether a token issued at iat predates the user's revocation cutoff.
// A Redis failure is returned as an error so callers can fail closed.
func IssuedBeforeCutoff(ctx context.Context, rdb *redis.Client, userID uuid.UUID, iat time.Time) (bool, error) {
	v, err := rdb.Get(ctx, key(userID)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	cutoff, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return true, fmt.Errorf("corrupt session cutoff for %s: %w", userID, err)
	}
	// Strict "<": a token issued in the same second as the revocation (e.g. the login right
	// after a password change) stays valid.
	return iat.Unix() < cutoff, nil
}
