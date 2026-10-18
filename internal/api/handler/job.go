package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// JobHandler exposes the background-job run log to company admins.
type JobHandler struct {
	jobs *repo.JobRepo
}

func NewJobHandler(jobs *repo.JobRepo) *JobHandler { return &JobHandler{jobs: jobs} }

// ListRuns returns the company's recent background-job runs (only runs that handled
// something or failed are logged).
// @Summary Background job runs
// @Tags Admin
// @Produce json
// @Param limit query int false "Max rows (default 100, max 500)"
// @Success 200 {array} domain.JobRun
// @Router /api/v1/admin/jobs/runs [get]
// @Security Bearer
func (h *JobHandler) ListRuns(c *fiber.Ctx) error {
	runs, err := h.jobs.ListRuns(c.Context(), c.QueryInt("limit", 100))
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(runs)
}
