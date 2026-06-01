package repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maidulcu/masaar-crm/internal/domain"
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
	if f.Limit == 0 {
		f.Limit = 50
	}
	args := []any{}
	conds := []string{}
	n := 1

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

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + joinStrings(conds, " AND ")
	}

	var total int
	countQ := "SELECT COUNT(*) FROM audit_logs a " + where
	if err := r.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count audit logs: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	q := fmt.Sprintf(`
		SELECT a.id, a.entity_type, a.entity_id, a.action, a.actor_id, a.diff, a.ts
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
		if err := rows.Scan(&l.ID, &l.EntityType, &l.EntityID, &l.Action, &l.ActorID, &rawDiff, &l.Timestamp); err != nil {
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
	const q = `
		INSERT INTO audit_logs (entity_type, entity_id, action, actor_id, diff)
		VALUES ($1, $2, $3, $4, $5)
	`
	r.db.Exec(ctx, q, entityType, entityID, action, actorID, diffJSON)
}

// AuditLog constants used across handlers.
const (
	AuditCreate         = "create"
	AuditUpdate         = "update"
	AuditDelete         = "delete"
	AuditLogin          = "login"
	AuditLogout         = "logout"
	AuditPasswordChange = "password_change"

	AuditContact = domain.AuditEntityContact
	AuditLead    = domain.AuditEntityLead
	AuditDeal    = domain.AuditEntityDeal
	AuditInvoice = domain.AuditEntityInvoice
	AuditUser     = "user"
	AuditDocument = "document"
)
