package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// actorAuditLogger is the audit trail as handlers reachable by both users and API keys use it.
type actorAuditLogger interface {
	Log(ctx context.Context, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, diff any)
	LogAPIKey(ctx context.Context, apiKeyID uuid.UUID, action, entityType string, entityID uuid.UUID, diff any)
}

// auditWrite records a write under whoever made the request: the signed-in user, or the API key.
func auditWrite(c *fiber.Ctx, audit actorAuditLogger, action, entityType string, entityID uuid.UUID, diff any) {
	if audit == nil {
		return
	}
	if id, ok := c.Locals("user_id").(uuid.UUID); ok {
		audit.Log(c.Context(), id, action, entityType, entityID, diff)
	} else if id, ok := c.Locals("api_key_id").(uuid.UUID); ok {
		audit.LogAPIKey(c.Context(), id, action, entityType, entityID, diff)
	}
}

// actorUserID returns the signed-in user, or nil for an API-key request.
func actorUserID(c *fiber.Ctx) *uuid.UUID {
	if id, ok := c.Locals("user_id").(uuid.UUID); ok {
		return &id
	}
	return nil
}

// actorAPIKeyID returns the API key of an API-key request, or nil.
func actorAPIKeyID(c *fiber.Ctx) *uuid.UUID {
	if id, ok := c.Locals("api_key_id").(uuid.UUID); ok {
		return &id
	}
	return nil
}
