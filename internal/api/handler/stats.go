package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type StatsHandler struct {
	stats *repo.StatsRepo
}

func NewStatsHandler(stats *repo.StatsRepo) *StatsHandler {
	return &StatsHandler{stats: stats}
}

// Overview godoc
// @Summary      Dashboard stats
// @Description  Returns aggregated CRM metrics for the dashboard overview.
// @Tags         Stats
// @Produce      json
// @Success      200  {object}  domain.Stats
// @Security     BearerAuth
// @Router       /stats [get]
func (h *StatsHandler) Overview(c *fiber.Ctx) error {
	s, err := h.stats.Overview(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(s)
}
