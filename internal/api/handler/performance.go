package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// PerformanceHandler serves agent KPI data, leaderboards, targets and trends.
//
//	GET  /api/v1/performance/leaderboard        — ranked agents for a period
//	GET  /api/v1/performance/agent/:id          — KPIs for a single agent
//	GET  /api/v1/performance/agent/:id/trends   — MoM / YoY trend lines
//	GET  /api/v1/performance/agent/:id/targets  — targets with progress %
//	POST /api/v1/performance/targets            — upsert a target (admin)
//	DELETE /api/v1/performance/targets/:id      — delete a target (admin)
type PerformanceHandler struct {
	repo *repo.PerformanceRepo
}

func NewPerformanceHandler(r *repo.PerformanceRepo) *PerformanceHandler {
	return &PerformanceHandler{repo: r}
}

// Leaderboard handles GET /api/v1/performance/leaderboard
// Query params: period=YYYY-MM (default: current month), metric=revenue|deals_won|listings_added|leads_converted
func (h *PerformanceHandler) Leaderboard(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	from, to := parsePeriodParam(c.Query("period"))
	metric := c.Query("metric", "revenue")

	board, err := h.repo.GetLeaderboard(c.Context(), companyID, from, to, metric)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Re-sort by the caller's chosen metric
	repo.SortLeaderboardByMetric(board, metric)
	for i := range board {
		board[i].Rank = i + 1
	}

	return c.JSON(fiber.Map{
		"data":   board,
		"period": from.Format("2006-01"),
		"metric": metric,
	})
}

// AgentKPIs handles GET /api/v1/performance/agent/:id
// Query params: period=YYYY-MM
func (h *PerformanceHandler) AgentKPIs(c *fiber.Ctx) error {
	agentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid agent id"})
	}
	from, to := parsePeriodParam(c.Query("period"))

	kpis, err := h.repo.GetAgentKPIs(c.Context(), agentID, from, to)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(kpis)
}

// AgentTrends handles GET /api/v1/performance/agent/:id/trends
func (h *PerformanceHandler) AgentTrends(c *fiber.Ctx) error {
	agentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid agent id"})
	}

	now := time.Now().UTC()
	trends, err := h.repo.GetKPITrends(c.Context(), agentID, now)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": trends})
}

// AgentTargets handles GET /api/v1/performance/agent/:id/targets
// Query params: period=YYYY-MM
func (h *PerformanceHandler) AgentTargets(c *fiber.Ctx) error {
	agentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid agent id"})
	}
	period := c.Query("period", time.Now().Format("2006-01"))

	targets, err := h.repo.GetTargets(c.Context(), agentID, period)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Enrich targets with current progress
	from, to := parsePeriod(period)
	kpis, _ := h.repo.GetAgentKPIs(c.Context(), agentID, from, to)
	if kpis != nil {
		for i := range targets {
			cur := kpiValueByMetric(kpis, targets[i].Metric)
			targets[i].CurrentValue = cur
			if targets[i].TargetValue > 0 {
				targets[i].ProgressPct = (cur / targets[i].TargetValue) * 100
				if targets[i].ProgressPct > 100 {
					targets[i].ProgressPct = 100
				}
			}
		}
	}

	return c.JSON(fiber.Map{"data": targets, "period": period})
}

// UpsertTarget handles POST /api/v1/performance/targets (admin only)
func (h *PerformanceHandler) UpsertTarget(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	var body struct {
		AgentID     string  `json:"agent_id"`
		Metric      string  `json:"metric"`
		TargetValue float64 `json:"target_value"`
		Period      string  `json:"period"` // YYYY-MM, default current month
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.AgentID == "" || body.Metric == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "agent_id and metric are required"})
	}
	agentID, err := uuid.Parse(body.AgentID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid agent_id"})
	}
	if body.Period == "" {
		body.Period = time.Now().Format("2006-01")
	}

	t := &repo.AgentTarget{
		AgentID:     agentID,
		CompanyID:   companyID,
		Metric:      body.Metric,
		TargetValue: body.TargetValue,
		Period:      body.Period,
	}
	if err := h.repo.UpsertTarget(c.Context(), t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(t)
}

// DeleteTarget handles DELETE /api/v1/performance/targets/:id (admin only)
func (h *PerformanceHandler) DeleteTarget(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.repo.DeleteTarget(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// parsePeriodParam parses ?period=YYYY-MM into start/end timestamps.
// Defaults to the current calendar month if blank or invalid.
func parsePeriodParam(period string) (from, to time.Time) {
	return parsePeriod(period)
}

func parsePeriod(period string) (from, to time.Time) {
	t, err := time.Parse("2006-01", period)
	if err != nil || period == "" {
		t = time.Now().UTC()
	}
	from = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	to = from.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return
}

func kpiValueByMetric(k *repo.AgentKPIs, metric string) float64 {
	switch metric {
	case "deals_won":
		return float64(k.DealsWon)
	case "revenue":
		return k.Revenue
	case "listings_added":
		return float64(k.ListingsAdded)
	case "leads_converted":
		return float64(k.LeadsConverted)
	case "leads_assigned":
		return float64(k.LeadsAssigned)
	}
	return 0
}
