package handler

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

// LeadRotationHandler manages round-robin / capacity-based lead assignment.
//
//	GET  /api/v1/settings/lead-rotation  — get current config
//	PATCH /api/v1/settings/lead-rotation — update config (admin only)
//	POST /api/v1/leads/:id/auto-assign   — manually trigger auto-assign for a lead
type LeadRotationHandler struct {
	rotationRepo *repo.LeadRotationRepo
	leadRepo     *repo.LeadRepo
	userRepo     *repo.UserRepo
	hub          *ws.Hub
}

func NewLeadRotationHandler(
	rotationRepo *repo.LeadRotationRepo,
	leadRepo *repo.LeadRepo,
	userRepo *repo.UserRepo,
	hub *ws.Hub,
) *LeadRotationHandler {
	return &LeadRotationHandler{
		rotationRepo: rotationRepo,
		leadRepo:     leadRepo,
		userRepo:     userRepo,
		hub:          hub,
	}
}

// GetSettings handles GET /api/v1/settings/lead-rotation
func (h *LeadRotationHandler) GetSettings(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	settings, err := h.rotationRepo.GetSettings(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Return available agents count for UI
	agents, _ := h.rotationRepo.ListActiveAgents(c.Context(), companyID)

	return c.JSON(fiber.Map{
		"mode":          settings.Mode,
		"enabled":       settings.Enabled,
		"max_per_agent": settings.MaxPerAgent,
		"agent_count":   len(agents),
		"updated_at":    settings.UpdatedAt,
	})
}

// UpdateSettings handles PATCH /api/v1/settings/lead-rotation
func (h *LeadRotationHandler) UpdateSettings(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	var body struct {
		Mode        string `json:"mode"`
		Enabled     bool   `json:"enabled"`
		MaxPerAgent int    `json:"max_per_agent"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	valid := map[string]bool{"manual": true, "round_robin": true, "capacity": true}
	if body.Mode != "" && !valid[body.Mode] {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "mode must be one of: manual, round_robin, capacity",
		})
	}

	existing, _ := h.rotationRepo.GetSettings(c.Context(), companyID)
	if existing == nil {
		existing = &domain.LeadRotationSettings{CompanyID: companyID}
	}
	if body.Mode != "" {
		existing.Mode = body.Mode
	}
	existing.Enabled = body.Enabled
	existing.MaxPerAgent = body.MaxPerAgent

	if err := h.rotationRepo.SaveSettings(c.Context(), existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true, "mode": existing.Mode, "enabled": existing.Enabled})
}

// AutoAssign handles POST /api/v1/leads/:id/auto-assign
// Picks the next agent using the configured rotation mode and assigns the lead.
func (h *LeadRotationHandler) AutoAssign(c *fiber.Ctx) error {
	leadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lead id"})
	}
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	agentID, err := h.pickAgent(c, companyID)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	if agentID == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "no available agents"})
	}

	if err := h.leadRepo.Assign(c.Context(), leadID, agentID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	h.hub.Broadcast(ws.Event{
		Type: "lead.assigned",
		Payload: fiber.Map{"lead_id": leadID, "agent_id": agentID},
	})

	return c.JSON(fiber.Map{"ok": true, "lead_id": leadID, "assigned_to": agentID})
}

// AssignNewLead is called internally (e.g. from webhook/public lead intake) to auto-assign
// a freshly created lead when rotation is enabled. Safe to call from goroutines.
func (h *LeadRotationHandler) AssignNewLead(companyID, leadID uuid.UUID) {
	ctx := context.Background()
	settings, err := h.rotationRepo.GetSettings(ctx, companyID)
	if err != nil || !settings.Enabled || settings.Mode == "manual" {
		return
	}

	agents, err := h.rotationRepo.ListActiveAgents(ctx, companyID)
	if err != nil || len(agents) == 0 {
		return
	}

	var agentID *uuid.UUID
	switch settings.Mode {
	case "round_robin":
		newIdx, err := h.rotationRepo.AdvanceRotationIndex(ctx, companyID)
		if err != nil {
			return
		}
		idx := (newIdx - 1) % len(agents)
		if idx < 0 {
			idx = 0
		}
		id := agents[idx].ID
		agentID = &id

	case "capacity":
		ids := make([]uuid.UUID, len(agents))
		for i, a := range agents {
			ids[i] = a.ID
		}
		counts, _ := h.rotationRepo.LeadCountByAgent(ctx, ids)
		minCount := -1
		for _, a := range agents {
			cnt := counts[a.ID]
			if settings.MaxPerAgent > 0 && cnt >= settings.MaxPerAgent {
				continue
			}
			if minCount < 0 || cnt < minCount {
				minCount = cnt
				id := a.ID
				agentID = &id
			}
		}
	}

	if agentID != nil {
		_ = h.leadRepo.Assign(ctx, leadID, agentID)
		log.Printf("[LeadRotation] company %s: lead %s → agent %s", companyID, leadID, *agentID)
	}
}

// pickAgent selects the next agent for the given company using the configured mode.
func (h *LeadRotationHandler) pickAgent(c *fiber.Ctx, companyID uuid.UUID) (*uuid.UUID, error) {
	settings, err := h.rotationRepo.GetSettings(c.Context(), companyID)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled || settings.Mode == "manual" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "lead rotation is not enabled")
	}

	agents, err := h.rotationRepo.ListActiveAgents(c.Context(), companyID)
	if err != nil || len(agents) == 0 {
		return nil, fiber.NewError(fiber.StatusServiceUnavailable, "no active agents found")
	}

	var selected *uuid.UUID

	switch settings.Mode {
	case "round_robin":
		newIdx, err := h.rotationRepo.AdvanceRotationIndex(c.Context(), companyID)
		if err != nil {
			return nil, err
		}
		idx := (newIdx - 1) % len(agents)
		if idx < 0 {
			idx = 0
		}
		id := agents[idx].ID
		selected = &id

	case "capacity":
		ids := make([]uuid.UUID, len(agents))
		for i, a := range agents {
			ids[i] = a.ID
		}
		counts, err := h.rotationRepo.LeadCountByAgent(c.Context(), ids)
		if err != nil {
			return nil, err
		}
		minCount := -1
		for _, a := range agents {
			cnt := counts[a.ID]
			if settings.MaxPerAgent > 0 && cnt >= settings.MaxPerAgent {
				continue
			}
			if minCount < 0 || cnt < minCount {
				minCount = cnt
				id := a.ID
				selected = &id
			}
		}
	}

	return selected, nil
}
