package repo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

// PortalRepo stores portal connections, recognises redelivered portal leads, and reads
// what the listing feed and the lead-source report need.
type PortalRepo struct {
	db *pgxpool.Pool
}

func NewPortalRepo(db *pgxpool.Pool) *PortalRepo { return &PortalRepo{db: db} }

// PortalConnection is a portal set up for a company (tokens are never returned after
// creation).
type PortalConnection struct {
	Portal     string     `json:"portal"`
	Enabled    bool       `json:"enabled"`
	LastLeadAt *time.Time `json:"last_lead_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// HashToken returns the stored form of a portal token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ptk_" + hex.EncodeToString(b), nil
}

// ListConnections returns the company's portal connections.
func (r *PortalRepo) ListConnections(ctx context.Context) ([]PortalConnection, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT portal, enabled, last_lead_at, created_at FROM portal_connections
		WHERE company_id = $1 ORDER BY portal`, cid)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (PortalConnection, error) {
		var p PortalConnection
		err := row.Scan(&p.Portal, &p.Enabled, &p.LastLeadAt, &p.CreatedAt)
		return p, err
	})
}

// Connect creates the portal connection, or replaces its tokens, and returns the new
// plaintext tokens (shown once).
func (r *PortalRepo) Connect(ctx context.Context, portal string) (inbound, feed string, err error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return "", "", err
	}
	if inbound, err = newToken(); err != nil {
		return "", "", err
	}
	if feed, err = newToken(); err != nil {
		return "", "", err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO portal_connections (company_id, portal, inbound_token_hash, feed_token_hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (company_id, portal) DO UPDATE SET inbound_token_hash = EXCLUDED.inbound_token_hash,
		       feed_token_hash = EXCLUDED.feed_token_hash, enabled = TRUE`,
		cid, portal, HashToken(inbound), HashToken(feed))
	return inbound, feed, err
}

// SetEnabled pauses or resumes a connection.
func (r *PortalRepo) SetEnabled(ctx context.Context, portal string, enabled bool) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE portal_connections SET enabled = $1 WHERE company_id = $2 AND portal = $3`, enabled, cid, portal)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Disconnect removes a connection; its tokens stop working.
func (r *PortalRepo) Disconnect(ctx context.Context, portal string) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `DELETE FROM portal_connections WHERE company_id = $1 AND portal = $2`, cid, portal)
	return err
}

// ErrBadPortalToken is returned for an unknown, disabled or wrong-portal token.
var ErrBadPortalToken = errors.New("invalid portal token")

// CompanyForToken resolves an enabled connection by token. kind is "inbound" or "feed".
func (r *PortalRepo) CompanyForToken(ctx context.Context, portal, kind, token string) (uuid.UUID, error) {
	col := "inbound_token_hash"
	if kind == "feed" {
		col = "feed_token_hash"
	}
	var cid uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT company_id FROM portal_connections WHERE `+col+` = $1 AND portal = $2 AND enabled`,
		HashToken(token), portal).Scan(&cid)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrBadPortalToken
	}
	return cid, err
}

// ClaimLead records a portal lead id; it reports false when the lead was already
// received (a redelivery).
func (r *PortalRepo) ClaimLead(ctx context.Context, portal, externalID string) (bool, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return false, err
	}
	tag, err := r.db.Exec(ctx, `INSERT INTO portal_leads (company_id, portal, external_id) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, cid, portal, externalID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ReleaseLead forgets a claim whose lead could not be created, so a retry can succeed.
func (r *PortalRepo) ReleaseLead(ctx context.Context, portal, externalID string) {
	if cid, err := tenant.From(ctx); err == nil {
		_, _ = r.db.Exec(ctx, `DELETE FROM portal_leads WHERE company_id = $1 AND portal = $2 AND external_id = $3 AND lead_id IS NULL`,
			cid, portal, externalID)
	}
}

// AttachLead links a claimed portal lead to the CRM lead created for it.
func (r *PortalRepo) AttachLead(ctx context.Context, portal, externalID string, leadID uuid.UUID) error {
	cid, err := tenant.From(ctx)
	if err != nil {
		return err
	}
	if _, err := r.db.Exec(ctx, `UPDATE portal_leads SET lead_id = $1 WHERE company_id = $2 AND portal = $3 AND external_id = $4`,
		leadID, cid, portal, externalID); err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `UPDATE portal_connections SET last_lead_at = NOW() WHERE company_id = $1 AND portal = $2`, cid, portal)
	return err
}

// ListingIDByReference finds a listing by its reference number, for linking a lead.
func (r *PortalRepo) ListingIDByReference(ctx context.Context, ref string) (*uuid.UUID, string, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, "", err
	}
	var id uuid.UUID
	var title string
	err = r.db.QueryRow(ctx, `SELECT id, title FROM listings WHERE company_id = $1 AND reference_number = $2 LIMIT 1`,
		cid, ref).Scan(&id, &title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return &id, title, nil
}

// LeadSourceRow summarises leads, wins and revenue for one source.
type LeadSourceRow struct {
	Source     string  `json:"source"`
	Leads      int     `json:"leads"`
	WonLeads   int     `json:"won_leads"`
	Deals      int     `json:"deals"`
	WonRevenue float64 `json:"won_revenue"`
}

// LeadSourceReport counts the company's leads created in [from, to) by source, with the
// leads won and the value of deals won from them.
func (r *PortalRepo) LeadSourceReport(ctx context.Context, from, to time.Time) ([]LeadSourceRow, error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT l.source, COUNT(DISTINCT l.id),
		       COUNT(DISTINCT l.id) FILTER (WHERE l.stage = 'won'),
		       COUNT(d.id),
		       COALESCE(SUM(d.amount) FILTER (WHERE d.stage = 'won'), 0)::float8
		  FROM leads l LEFT JOIN deals d ON d.lead_id = l.id AND d.company_id = l.company_id
		 WHERE l.company_id = $1 AND l.deleted_at IS NULL AND l.created_at >= $2 AND l.created_at < $3
		 GROUP BY l.source ORDER BY COUNT(DISTINCT l.id) DESC`, cid, from, to)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (LeadSourceRow, error) {
		var s LeadSourceRow
		err := row.Scan(&s.Source, &s.Leads, &s.WonLeads, &s.Deals, &s.WonRevenue)
		return s, err
	})
}
