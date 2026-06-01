package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// CommissionHandler manages commission structures and agent commission records.
//
//	GET  /api/v1/commissions/structures                 — list structures
//	POST /api/v1/commissions/structures                 — create structure
//	PATCH /api/v1/commissions/structures/:id            — update structure
//	DELETE /api/v1/commissions/structures/:id           — delete structure
//
//	GET  /api/v1/commissions                            — list agent commissions
//	POST /api/v1/commissions                            — create commission record
//	PATCH /api/v1/commissions/:id/status                — approve / mark paid
//	PATCH /api/v1/commissions/:id/amount                — update total amount
//	POST /api/v1/commissions/calculate                  — calculate for a period
type CommissionHandler struct {
	repo *repo.CommissionRepo
}

func NewCommissionHandler(r *repo.CommissionRepo) *CommissionHandler {
	return &CommissionHandler{repo: r}
}

// ── Structures ────────────────────────────────────────────────────────────────

func (h *CommissionHandler) ListStructures(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	structs, err := h.repo.ListStructures(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": structs})
}

func (h *CommissionHandler) CreateStructure(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	var s repo.CommissionStructure
	if err := c.BodyParser(&s); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if s.Name == "" || s.Type == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "structure_name and commission_type are required"})
	}
	s.CompanyID = companyID
	if err := h.repo.CreateStructure(c.Context(), &s); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(s)
}

func (h *CommissionHandler) UpdateStructure(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	var s repo.CommissionStructure
	if err := c.BodyParser(&s); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	s.ID = id
	s.CompanyID = companyID
	if err := h.repo.UpdateStructure(c.Context(), &s); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *CommissionHandler) DeleteStructure(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	if err := h.repo.DeleteStructure(c.Context(), id, companyID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Agent Commissions ─────────────────────────────────────────────────────────

func (h *CommissionHandler) List(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	var agentID *uuid.UUID
	if v := c.Query("agent_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid agent_id"})
		}
		agentID = &id
	}
	status := c.Query("status")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	list, total, err := h.repo.ListAgentCommissions(c.Context(), companyID, agentID, status, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"data": list,
		"meta": fiber.Map{"total": total, "page": page, "limit": limit},
	})
}

func (h *CommissionHandler) Create(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	var body repo.AgentCommission
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	body.CompanyID = companyID
	if body.AgentID == uuid.Nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "agent_id is required"})
	}
	if body.PeriodStart.IsZero() || body.PeriodEnd.IsZero() {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "commission_period_start and commission_period_end are required"})
	}
	if err := h.repo.CreateAgentCommission(c.Context(), &body); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(body)
}

func (h *CommissionHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Status           string `json:"status"`
		PaymentReference string `json:"payment_reference"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	valid := map[string]bool{"approved": true, "paid": true, "disputed": true, "pending": true}
	if !valid[body.Status] {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "status must be one of: pending, approved, paid, disputed",
		})
	}
	if err := h.repo.UpdateCommissionStatus(c.Context(), id, repo.CommissionStatus(body.Status), body.PaymentReference); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true, "status": body.Status})
}

func (h *CommissionHandler) UpdateAmount(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		TotalCommission float64 `json:"total_commission"`
		Notes           string  `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.repo.UpdateCommissionAmount(c.Context(), id, body.TotalCommission, body.Notes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// Calculate handles POST /api/v1/commissions/calculate
// Returns commission metrics for an agent over a given date range without saving.
func (h *CommissionHandler) Calculate(c *fiber.Ctx) error {
	var body struct {
		AgentID   string `json:"agent_id"`
		PeriodFrom string `json:"period_from"` // YYYY-MM-DD
		PeriodTo   string `json:"period_to"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	agentID, err := uuid.Parse(body.AgentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid agent_id"})
	}
	from, err := time.Parse("2006-01-02", body.PeriodFrom)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid period_from (use YYYY-MM-DD)"})
	}
	to, err := time.Parse("2006-01-02", body.PeriodTo)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid period_to (use YYYY-MM-DD)"})
	}

	dc, dr, lc, lr, err := h.repo.CalculateForPeriod(c.Context(), agentID, from, to)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"agent_id":       agentID,
		"period_from":    body.PeriodFrom,
		"period_to":      body.PeriodTo,
		"deals_count":    dc,
		"deals_revenue":  dr,
		"leases_count":   lc,
		"leases_revenue": lr,
		"total_revenue":  dr + lr,
	})
}
