package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type InspectionHandler struct {
	templateRepo  *repo.InspectionTemplateRepo
	inspectionRepo *repo.InspectionRepo
}

func NewInspectionHandler(templateRepo *repo.InspectionTemplateRepo, inspectionRepo *repo.InspectionRepo) *InspectionHandler {
	return &InspectionHandler{
		templateRepo:   templateRepo,
		inspectionRepo: inspectionRepo,
	}
}

// @Summary List inspection templates
// @Tags Inspections
// @Security Bearer
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/inspection-templates [get]
func (h *InspectionHandler) ListTemplates(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	templates, err := h.templateRepo.List(c.Context(), companyID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch templates"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": templates})
}

// @Summary Create inspection template
// @Tags Inspections
// @Security Bearer
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/inspection-templates [post]
func (h *InspectionHandler) CreateTemplate(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	var req struct {
		TemplateName             string                 `json:"template_name"`
		InspectionType           string                 `json:"inspection_type"`
		ChecklistItems           []domain.ChecklistItem `json:"checklist_items"`
		EstimatedDurationMinutes int                    `json:"estimated_duration_minutes"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	template := &domain.InspectionTemplate{
		ID:                       uuid.New(),
		CompanyID:                companyID,
		TemplateName:             req.TemplateName,
		InspectionType:           domain.InspectionType(req.InspectionType),
		ChecklistItems:           req.ChecklistItems,
		EstimatedDurationMinutes: req.EstimatedDurationMinutes,
	}

	if err := h.templateRepo.Create(c.Context(), template); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create template"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": template})
}

// @Summary List inspections
// @Tags Inspections
// @Security Bearer
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/inspections [get]
func (h *InspectionHandler) ListInspections(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	inspections, total, err := h.inspectionRepo.List(c.Context(), companyID, limit, offset)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch inspections"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": inspections,
		"meta": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// @Summary Get inspection details
// @Tags Inspections
// @Security Bearer
// @Param id path string true "Inspection ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/inspections/{id} [get]
func (h *InspectionHandler) GetInspection(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid inspection ID"})
	}

	inspection, err := h.inspectionRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Inspection not found"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": inspection})
}

// @Summary Create inspection
// @Tags Inspections
// @Security Bearer
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/inspections [post]
func (h *InspectionHandler) CreateInspection(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))
	userID := c.Locals("user_id").(uuid.UUID)

	var req struct {
		PropertyID     uuid.UUID             `json:"property_id"`
		TemplateID     *uuid.UUID            `json:"template_id"`
		InspectionType string                `json:"inspection_type"`
		ScheduledDate  time.Time             `json:"scheduled_date"`
		InspectorID    *uuid.UUID            `json:"inspector_id"`
		TenantID       *uuid.UUID            `json:"tenant_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	inspection := &domain.Inspection{
		ID:             uuid.New(),
		CompanyID:      companyID,
		PropertyID:     req.PropertyID,
		TemplateID:     req.TemplateID,
		InspectionType: req.InspectionType,
		ScheduledDate:  req.ScheduledDate,
		InspectorID:    req.InspectorID,
		TenantID:       req.TenantID,
		Status:         domain.InspectionScheduled,
		CreatedBy:      userID,
	}

	if err := h.inspectionRepo.Create(c.Context(), inspection); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create inspection"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": inspection})
}

// @Summary Update inspection
// @Tags Inspections
// @Security Bearer
// @Param id path string true "Inspection ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/inspections/{id} [patch]
func (h *InspectionHandler) UpdateInspection(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid inspection ID"})
	}

	inspection, err := h.inspectionRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Inspection not found"})
	}

	var req struct {
		Status           *string                          `json:"status"`
		Findings         *string                          `json:"findings"`
		SeverityLevel    *string                          `json:"severity_level"`
		PhotosURLs       []string                         `json:"photos_urls"`
		ChecklistResults *map[string]domain.ChecklistResult `json:"checklist_results"`
		CompletedDate    *time.Time                       `json:"completed_date"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Status != nil {
		inspection.Status = domain.InspectionStatus(*req.Status)
	}
	if req.Findings != nil {
		inspection.Findings = *req.Findings
	}
	if req.SeverityLevel != nil {
		inspection.SeverityLevel = domain.SeverityLevel(*req.SeverityLevel)
	}
	if len(req.PhotosURLs) > 0 {
		inspection.PhotosURLs = req.PhotosURLs
	}
	if req.ChecklistResults != nil {
		inspection.ChecklistResults = *req.ChecklistResults
	}
	if req.CompletedDate != nil {
		inspection.CompletedDate = req.CompletedDate
	}

	if err := h.inspectionRepo.Update(c.Context(), inspection); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update inspection"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": inspection})
}

// @Summary Complete inspection
// @Tags Inspections
// @Security Bearer
// @Param id path string true "Inspection ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/inspections/{id}/complete [post]
func (h *InspectionHandler) CompleteInspection(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid inspection ID"})
	}

	inspection, err := h.inspectionRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Inspection not found"})
	}

	inspection.Status = domain.InspectionCompleted
	now := time.Now()
	inspection.CompletedDate = &now

	if err := h.inspectionRepo.Update(c.Context(), inspection); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to complete inspection"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": inspection})
}
