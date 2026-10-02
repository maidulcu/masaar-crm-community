// Package tenant carries the current company through a context.Context so that
// repositories can scope every query to it. Resolution is fail-closed: with no
// company in the context, From returns ErrMissing and the repository refuses to
// run the query.
package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// ErrMissing is returned when no company is associated with the context.
var ErrMissing = errors.New("tenant: no company in context")

type ctxKey struct{}

// localsKey is the Fiber locals key set by middleware.ExtractClaims. Fiber stores
// locals as fasthttp user values, which are visible through c.Context().Value().
const localsKey = "company_id"

// With returns a context bound to the given company. Use it for work that does not
// originate from an authenticated HTTP request (webhooks, background jobs).
func With(ctx context.Context, companyID uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxKey{}, companyID)
}

// From resolves the company for ctx: an explicit tenant.With binding wins,
// otherwise the authenticated request's company_id (Fiber locals) is used.
func From(ctx context.Context) (uuid.UUID, error) {
	if id, ok := ctx.Value(ctxKey{}).(uuid.UUID); ok && id != uuid.Nil {
		return id, nil
	}
	if s, ok := ctx.Value(localsKey).(string); ok {
		if id, err := uuid.Parse(s); err == nil && id != uuid.Nil {
			return id, nil
		}
	}
	return uuid.Nil, ErrMissing
}
