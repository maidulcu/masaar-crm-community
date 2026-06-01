package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type AuditHandler struct {
	repo *repo.AuditLogRepo
}

func NewAuditHandler(repo *repo.AuditLogRepo) *AuditHandler {
	return &AuditHandler{repo: repo}
}

// List godoc
// @Summary      List audit logs
// @Description  Returns paginated audit log entries. Requires admin role.
// @Tags         Audit
// @Produce      json
// @Param        entity_type  query  string  false  "Filter by entity type (contact, lead, deal, invoice, user, document)"
// @Param        entity_id    query  string  false  "Filter by entity UUID"
// @Param        actor_id     query  string  false  "Filter by actor UUID"
// @Param        action       query  string  false  "Filter by action (create, update, delete, login, logout)"
// @Param        page         query  int     false  "Page (default 1)"
// @Param        limit        query  int     false  "Page size (default 50)"
// @Success      200          {object} object{data=[]domain.AuditLog,total=int,page=int,limit=int}
// @Security     BearerAuth
// @Router       /audit-logs [get]
func (h *AuditHandler) List(c *fiber.Ctx) error {
	f := repo.AuditLogFilter{
		EntityType: c.Query("entity_type"),
		Action:     c.Query("action"),
		Limit:      c.QueryInt("limit", 50),
		Offset:     (c.QueryInt("page", 1) - 1) * c.QueryInt("limit", 50),
	}

	if eid := c.Query("entity_id"); eid != "" {
		id, err := uuid.Parse(eid)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid entity_id"})
		}
		f.EntityID = &id
	}
	if aid := c.Query("actor_id"); aid != "" {
		id, err := uuid.Parse(aid)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid actor_id"})
		}
		f.ActorID = &id
	}

	result, err := h.repo.List(c.Context(), f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}
