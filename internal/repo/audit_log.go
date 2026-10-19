package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/tenant"
)

type AuditLogFilter struct {
	EntityType string
	EntityID   *uuid.UUID
	ActorID    *uuid.UUID
	Action     string
	Limit      int
	Offset     int
}

type AuditLogRepo struct {
	db *pgxpool.Pool
}

func NewAuditLogRepo(db *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) List(ctx context.Context, f AuditLogFilter) (*domain.PaginatedResult[domain.AuditLog], error) {
	cid, err := tenant.From(ctx)
	if err != nil {
		return nil, err
	}
	if f.Limit == 0 {
		f.Limit = 50
	}
	args := []any{cid}
	conds := []string{"a.company_id = $1"}
	n := 2

	if f.EntityType != "" {
		conds = append(conds, fmt.Sprintf("a.entity_type = $%d", n))
		args = append(args, f.EntityType)
		n++
	}
	if f.EntityID != nil {
		conds = append(conds, fmt.Sprintf("a.entity_id = $%d", n))
		args = append(args, *f.EntityID)
		n++
	}
	if f.ActorID != nil {
		conds = append(conds, fmt.Sprintf("a.actor_id = $%d", n))
		args = append(args, *f.ActorID)
		n++
	}
	if f.Action != "" {
		conds = append(conds, fmt.Sprintf("a.action = $%d", n))
		args = append(args, f.Action)
		n++
	}

	where := "WHERE " + joinStrings(conds, " AND ")

	var total int
	countQ := "SELECT COUNT(*) FROM audit_logs a " + where
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count audit logs: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	q := fmt.Sprintf(`
		SELECT a.id, a.entity_type, a.entity_id, a.action, a.actor_id, a.api_key_id, a.diff, a.ts
		FROM audit_logs a
		%s
		ORDER BY a.ts DESC
		LIMIT $%d OFFSET $%d
	`, where, n, n+1)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var rawDiff []byte
		if err := rows.Scan(&l.ID, &l.EntityType, &l.EntityID, &l.Action, &l.ActorID, &l.APIKeyID, &rawDiff, &l.Timestamp); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		if rawDiff != nil {
			l.Diff = json.RawMessage(rawDiff)
		}
		logs = append(logs, l)
	}
	return &domain.PaginatedResult[domain.AuditLog]{
		Data: logs, Total: total, Page: f.Offset/f.Limit + 1, Limit: f.Limit,
	}, nil
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}

// Log inserts an audit entry. Errors are intentionally non-fatal — a failed
// audit write must not block the primary operation.
func (r *AuditLogRepo) Log(ctx context.Context, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, diff any) {
	var diffJSON []byte
	if diff != nil {
		diffJSON, _ = json.Marshal(diff)
	}
	// The actor's own company is authoritative (login events are logged before a
	// request is authenticated, so the request context carries no company yet).
	const q = `
		INSERT INTO audit_logs (entity_type, entity_id, action, actor_id, diff, company_id)
		SELECT $1, $2, $3, $4, $5, u.company_id FROM users u WHERE u.id = $4
	`
	r.db.Exec(ctx, q, entityType, entityID, action, actorID, diffJSON)
}

// LogAPIKey inserts an audit entry for an action taken with an API key, which has no
// user actor. The company comes from the key itself. Errors are non-fatal, as in Log.
func (r *AuditLogRepo) LogAPIKey(ctx context.Context, apiKeyID uuid.UUID, action, entityType string, entityID uuid.UUID, diff any) {
	var diffJSON []byte
	if diff != nil {
		diffJSON, _ = json.Marshal(diff)
	}
	const q = `
		INSERT INTO audit_logs (entity_type, entity_id, action, api_key_id, diff, company_id)
		SELECT $1, $2, $3, $4, $5, k.company_id FROM api_keys k WHERE k.id = $4
	`
	r.db.Exec(ctx, q, entityType, entityID, action, apiKeyID, diffJSON)
}

// LogPartner records an action a partner client (Partner API) took in companyID.
func (r *AuditLogRepo) LogPartner(ctx context.Context, clientID string, companyID uuid.UUID, action, entityType string, entityID uuid.UUID, diff any) {
	var diffJSON []byte
	if diff != nil {
		diffJSON, _ = json.Marshal(diff)
	}
	r.db.Exec(ctx, `
		INSERT INTO audit_logs (entity_type, entity_id, action, partner_client_id, diff, company_id)
		VALUES ($1, $2, $3, $4, $5, $6)`, entityType, entityID, action, clientID, diffJSON, companyID)
}

// AuditLog constants used across handlers.
const (
	AuditCreate         = "create"
	AuditUpdate         = "update"
	AuditDelete         = "delete"
	AuditLogin          = "login"
	AuditLogout         = "logout"
	AuditPasswordChange = "password_change"

	AuditContact  = domain.AuditEntityContact
	AuditLead     = domain.AuditEntityLead
	AuditDeal     = domain.AuditEntityDeal
	AuditInvoice  = domain.AuditEntityInvoice
	AuditUser     = "user"
	AuditDocument = "document"
	AuditWebhook  = "webhook_subscription"
	AuditCompany  = "company"
	AuditAPIKey   = "api_key"
)
