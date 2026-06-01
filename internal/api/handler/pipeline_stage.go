package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type PipelineStageHandler struct {
	repo *repo.PipelineStageRepo
}

func NewPipelineStageHandler(repo *repo.PipelineStageRepo) *PipelineStageHandler {
	return &PipelineStageHandler{repo: repo}
}

// List godoc
// @Summary      List pipeline stages
// @Description  Returns pipeline stages for a company. Filters by entity_type (lead, deal).
// @Tags         Pipeline Stages
// @Produce      json
// @Param        entity_type  query  string  false  "Entity type (lead, deal). Defaults to lead."
// @Success      200  {array}  domain.PipelineStage
// @Security     BearerAuth
// @Router       /pipeline-stages [get]
func (h *PipelineStageHandler) List(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	entityType := c.Query("entity_type", "lead")
	stages, err := h.repo.ListByCompany(c.Context(), companyID, entityType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stages)
}

// Create godoc
// @Summary      Create pipeline stage
// @Description  Creates a new pipeline stage for the company.
// @Tags         Pipeline Stages
// @Accept       json
// @Produce      json
// @Param        body  body  domain.PipelineStage  true  "Stage payload"
// @Success      201  {object}  domain.PipelineStage
// @Security     BearerAuth
// @Router       /pipeline-stages [post]
func (h *PipelineStageHandler) Create(c *fiber.Ctx) error {
	var s domain.PipelineStage
	if err := c.BodyParser(&s); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if s.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	if s.EntityType == "" {
		s.EntityType = "lead"
	}
	if s.Color == "" {
		s.Color = "#6366f1"
	}

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	s.CompanyID = companyID

	if err := h.repo.Create(c.Context(), &s); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(s)
}

// Update godoc
// @Summary      Update pipeline stage
// @Description  Updates a pipeline stage name, color, sort order, or won/lost flags.
// @Tags         Pipeline Stages
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "Stage UUID"
// @Param        body  body  domain.PipelineStage  true  "Stage payload"
// @Success      200  {object}  domain.PipelineStage
// @Security     BearerAuth
// @Router       /pipeline-stages/{id} [patch]
func (h *PipelineStageHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	s, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "stage not found"})
	}

	var body struct {
		Name      *string `json:"name"`
		SortOrder *int    `json:"sort_order"`
		Color     *string `json:"color"`
		IsWon     *bool   `json:"is_won"`
		IsLost    *bool   `json:"is_lost"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if body.Name != nil {
		s.Name = *body.Name
	}
	if body.SortOrder != nil {
		s.SortOrder = *body.SortOrder
	}
	if body.Color != nil {
		s.Color = *body.Color
	}
	if body.IsWon != nil {
		s.IsWon = *body.IsWon
	}
	if body.IsLost != nil {
		s.IsLost = *body.IsLost
	}

	if err := h.repo.Update(c.Context(), s); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(s)
}

// Delete godoc
// @Summary      Delete pipeline stage
// @Description  Deletes a pipeline stage. Leads in this stage will need reassignment.
// @Tags         Pipeline Stages
// @Param        id  path  string  true  "Stage UUID"
// @Success      204
// @Security     BearerAuth
// @Router       /pipeline-stages/{id} [delete]
func (h *PipelineStageHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.repo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Reorder godoc
// @Summary      Reorder pipeline stages
// @Description  Sets the sort order of all stages. Send an ordered array of stage UUIDs.
// @Tags         Pipeline Stages
// @Accept       json
// @Produce      json
// @Param        body  body  object{ids=array}  true  "Ordered stage IDs"
// @Success      200  {object}  object{ok=boolean}
// @Security     BearerAuth
// @Router       /pipeline-stages/reorder [post]
func (h *PipelineStageHandler) Reorder(c *fiber.Ctx) error {
	var body struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if err := c.BodyParser(&body); err != nil || len(body.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ids array required"})
	}

	if err := h.repo.Reorder(c.Context(), body.IDs); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

// ResetDefault godoc
// @Summary      Reset to default stages
// @Description  Replaces all custom stages with the default 6-stage pipeline.
// @Tags         Pipeline Stages
// @Produce      json
// @Param        entity_type  query  string  false  "Entity type (lead, deal). Defaults to lead."
// @Success      200  {object}  object{ok=boolean}
// @Security     BearerAuth
// @Router       /pipeline-stages/reset-default [post]
func (h *PipelineStageHandler) ResetDefault(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	entityType := c.Query("entity_type", "lead")
	if err := h.repo.SetDefaults(c.Context(), companyID, entityType); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}
