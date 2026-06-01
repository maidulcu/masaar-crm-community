package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

// ViewingHandler manages property viewings and the agent calendar.
//
//	GET    /api/v1/viewings                 — list (filter: agent_id, listing_id, contact_id, status, from, to)
//	GET    /api/v1/viewings/:id             — single viewing
//	POST   /api/v1/viewings                 — schedule a viewing
//	PATCH  /api/v1/viewings/:id             — update details
//	PATCH  /api/v1/viewings/:id/status      — confirm / check-in / complete / cancel
//	DELETE /api/v1/viewings/:id             — delete
type ViewingHandler struct {
	viewingRepo      *repo.ViewingRepo
	notificationRepo *repo.NotificationRepo
	hub              *ws.Hub
}

func NewViewingHandler(
	viewingRepo *repo.ViewingRepo,
	notificationRepo *repo.NotificationRepo,
	hub *ws.Hub,
) *ViewingHandler {
	return &ViewingHandler{
		viewingRepo:      viewingRepo,
		notificationRepo: notificationRepo,
		hub:              hub,
	}
}

// List handles GET /api/v1/viewings
func (h *ViewingHandler) List(c *fiber.Ctx) error {
	f := repo.ViewingFilter{
		Status: c.Query("status"),
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 200),
	}
	if v := c.Query("agent_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid agent_id"})
		}
		f.AgentID = &id
	}
	if v := c.Query("contact_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid contact_id"})
		}
		f.ContactID = &id
	}
	if v := c.Query("listing_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid listing_id"})
		}
		f.ListingID = &id
	}
	if v := c.Query("from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
		}
		if err == nil {
			f.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
		}
		if err == nil {
			f.To = &t
		}
	}

	viewings, total, err := h.viewingRepo.List(c.Context(), f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"data": viewings,
		"meta": fiber.Map{"total": total, "page": f.Page, "limit": f.Limit},
	})
}

// Get handles GET /api/v1/viewings/:id
func (h *ViewingHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	v, err := h.viewingRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "viewing not found"})
	}
	return c.JSON(v)
}

// Create handles POST /api/v1/viewings
func (h *ViewingHandler) Create(c *fiber.Ctx) error {
	var body struct {
		ListingID   string `json:"listing_id"`
		ContactID   string `json:"contact_id"`
		AgentID     string `json:"agent_id"`
		LeadID      string `json:"lead_id"`
		ScheduledAt string `json:"scheduled_at"` // RFC3339
		DurationMin int    `json:"duration_min"`
		Address     string `json:"address"`
		Notes       string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.ContactID == "" || body.ScheduledAt == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "contact_id and scheduled_at are required"})
	}

	contactID, err := uuid.Parse(body.ContactID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid contact_id"})
	}
	scheduledAt, err := time.Parse(time.RFC3339, body.ScheduledAt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid scheduled_at — use RFC3339"})
	}

	v := &domain.Viewing{
		ContactID:   contactID,
		ScheduledAt: scheduledAt,
		DurationMin: body.DurationMin,
		Address:     body.Address,
		Notes:       body.Notes,
	}

	if body.ListingID != "" {
		id, err := uuid.Parse(body.ListingID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid listing_id"})
		}
		v.ListingID = &id
	}
	if body.AgentID != "" {
		id, err := uuid.Parse(body.AgentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid agent_id"})
		}
		v.AgentID = &id
	} else {
		// Default to the requesting user
		if uid, ok := c.Locals("user_id").(uuid.UUID); ok {
			v.AgentID = &uid
		}
	}
	if body.LeadID != "" {
		id, err := uuid.Parse(body.LeadID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lead_id"})
		}
		v.LeadID = &id
	}

	// Conflict detection
	if v.AgentID != nil {
		conflict, err := h.viewingRepo.CheckConflict(c.Context(), *v.AgentID, scheduledAt, v.DurationMin, nil)
		if err == nil && conflict {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": fmt.Sprintf("agent already has a viewing at %s", scheduledAt.Format("Jan 2 15:04")),
			})
		}
	}

	if err := h.viewingRepo.Create(c.Context(), v); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Notify the assigned agent via WebSocket
	if v.AgentID != nil {
		h.hub.SendToUser(v.AgentID.String(), ws.Event{
			Type: "viewing.scheduled",
			Payload: fiber.Map{
				"viewing_id":   v.ID,
				"scheduled_at": v.ScheduledAt,
				"contact_id":   v.ContactID,
			},
		})
		// Create in-app notification
		_ = h.notificationRepo.Create(c.Context(), &domain.Notification{
			UserID: *v.AgentID,
			Type:   "viewing_scheduled",
			Title:  "New viewing scheduled",
			Body:   fmt.Sprintf("Viewing on %s", scheduledAt.Format("Jan 2 at 15:04")),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(v)
}

// Update handles PATCH /api/v1/viewings/:id
func (h *ViewingHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	existing, err := h.viewingRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "viewing not found"})
	}

	var body struct {
		ListingID   string `json:"listing_id"`
		AgentID     string `json:"agent_id"`
		LeadID      string `json:"lead_id"`
		ScheduledAt string `json:"scheduled_at"`
		DurationMin int    `json:"duration_min"`
		Address     string `json:"address"`
		Notes       string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	if body.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, body.ScheduledAt)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid scheduled_at"})
		}
		existing.ScheduledAt = t
	}
	if body.DurationMin > 0 {
		existing.DurationMin = body.DurationMin
	}
	if body.Address != "" {
		existing.Address = body.Address
	}
	if body.Notes != "" {
		existing.Notes = body.Notes
	}
	if body.ListingID != "" {
		lid, _ := uuid.Parse(body.ListingID)
		existing.ListingID = &lid
	}
	if body.AgentID != "" {
		aid, _ := uuid.Parse(body.AgentID)
		existing.AgentID = &aid
	}
	if body.LeadID != "" {
		lid, _ := uuid.Parse(body.LeadID)
		existing.LeadID = &lid
	}

	// Conflict check on reschedule
	if existing.AgentID != nil {
		conflict, err := h.viewingRepo.CheckConflict(c.Context(), *existing.AgentID, existing.ScheduledAt, existing.DurationMin, &id)
		if err == nil && conflict {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": fmt.Sprintf("agent already has a viewing at %s", existing.ScheduledAt.Format("Jan 2 15:04")),
			})
		}
	}

	if err := h.viewingRepo.Update(c.Context(), existing); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(existing)
}

// UpdateStatus handles PATCH /api/v1/viewings/:id/status
func (h *ViewingHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	allowed := map[string]bool{
		"confirmed": true, "checked_in": true, "completed": true,
		"cancelled": true, "no_show": true,
	}
	if !allowed[body.Status] {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "status must be: confirmed, checked_in, completed, cancelled, or no_show",
		})
	}

	if err := h.viewingRepo.UpdateStatus(c.Context(), id, domain.ViewingStatus(body.Status)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	h.hub.Broadcast(ws.Event{
		Type:    "viewing.status_changed",
		Payload: fiber.Map{"viewing_id": id, "status": body.Status},
	})

	return c.JSON(fiber.Map{"ok": true, "status": body.Status})
}

// Delete handles DELETE /api/v1/viewings/:id
func (h *ViewingHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.viewingRepo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
