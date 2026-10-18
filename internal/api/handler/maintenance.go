package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type MaintenanceTaskHandler struct {
	maintenanceRepo *repo.MaintenanceTaskRepo
}

func NewMaintenanceTaskHandler(maintenanceRepo *repo.MaintenanceTaskRepo) *MaintenanceTaskHandler {
	return &MaintenanceTaskHandler{
		maintenanceRepo: maintenanceRepo,
	}
}

// @Summary List maintenance tasks
// @Tags Maintenance
// @Security Bearer
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Param status query string false "Filter by status"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/maintenance-tasks [get]
func (h *MaintenanceTaskHandler) List(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	status := c.Query("status", "")
	if status != "" && !validMaintenanceStatuses[domain.MaintenanceStatus(status)] {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
	}
	var tasks []domain.MaintenanceTask
	var total int
	var err error

	if status != "" {
		tasks, total, err = h.maintenanceRepo.ListByStatus(c.Context(), companyID, status, limit, offset)
	} else {
		tasks, total, err = h.maintenanceRepo.List(c.Context(), companyID, limit, offset)
	}

	if err != nil {
		return serverError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": tasks,
		"meta": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// @Summary Get maintenance task details
// @Tags Maintenance
// @Security Bearer
// @Param id path string true "Task ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/maintenance-tasks/{id} [get]
func (h *MaintenanceTaskHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	task, err := h.maintenanceRepo.Get(c.Context(), id)
	if err != nil {
		return taskLookupError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": task})
}

// @Summary Create maintenance task
// @Tags Maintenance
// @Security Bearer
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/maintenance-tasks [post]
func (h *MaintenanceTaskHandler) Create(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))
	userID := c.Locals("user_id").(uuid.UUID)

	var req struct {
		PropertyID        uuid.UUID  `json:"property_id"`
		InspectionID      *uuid.UUID `json:"inspection_id"`
		MaintenanceType   string     `json:"maintenance_type"`
		Description       string     `json:"description"`
		Priority          string     `json:"priority"`
		ScheduledDate     *time.Time `json:"scheduled_date"`
		DueDate           *time.Time `json:"due_date"`
		ContractorName    string     `json:"contractor_name"`
		ContractorContact string     `json:"contractor_contact"`
		EstimatedCost     *float64   `json:"estimated_cost"`
		AssignedTo        *uuid.UUID `json:"assigned_to"`
		Notes             string     `json:"notes"`
	}
	// The columns are DATEs and the browser sends "2026-05-01" (or ""), which *time.Time rejects.
	if err := json.Unmarshal(normalizeDateFields(c.Body(), "scheduled_date", "due_date", "inspection_id", "assigned_to"), &req); err != nil {
		return badRequest(c, err)
	}
	if req.PropertyID == uuid.Nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "property_id is required"})
	}

	task := &domain.MaintenanceTask{
		ID:                uuid.New(),
		CompanyID:         companyID,
		PropertyID:        req.PropertyID,
		InspectionID:      req.InspectionID,
		MaintenanceType:   domain.MaintenanceType(req.MaintenanceType),
		Description:       req.Description,
		Priority:          domain.TaskPriority(req.Priority),
		ScheduledDate:     req.ScheduledDate,
		DueDate:           req.DueDate,
		ContractorName:    req.ContractorName,
		ContractorContact: req.ContractorContact,
		EstimatedCost:     req.EstimatedCost,
		Status:            domain.MaintenancePending,
		AssignedTo:        req.AssignedTo,
		Notes:             req.Notes,
		CreatedBy:         userID,
	}

	if err := validateMaintenanceTask(task); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.maintenanceRepo.Create(c.Context(), task); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // property, inspection or assignee is not this company's
			return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"error": "property, inspection or assignee not found"})
		}
		return serverError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": task})
}

// @Summary Update maintenance task
// @Tags Maintenance
// @Security Bearer
// @Param id path string true "Task ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/maintenance-tasks/{id} [patch]
func (h *MaintenanceTaskHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	task, err := h.maintenanceRepo.Get(c.Context(), id)
	if err != nil {
		return taskLookupError(c, err)
	}

	var req struct {
		MaintenanceType   *string    `json:"maintenance_type"`
		Description       *string    `json:"description"`
		Priority          *string    `json:"priority"`
		ScheduledDate     *time.Time `json:"scheduled_date"`
		DueDate           *time.Time `json:"due_date"`
		ContractorName    *string    `json:"contractor_name"`
		ContractorContact *string    `json:"contractor_contact"`
		EstimatedCost     *float64   `json:"estimated_cost"`
		ActualCost        *float64   `json:"actual_cost"`
		Status            *string    `json:"status"`
		AssignedTo        *uuid.UUID `json:"assigned_to"`
		Notes             *string    `json:"notes"`
	}
	if err := json.Unmarshal(normalizeDateFields(c.Body(), "scheduled_date", "due_date", "assigned_to"), &req); err != nil {
		return badRequest(c, err)
	}
	if task.Status == domain.MaintenanceCompleted || task.Status == domain.MaintenanceCancelled {
		// Re-opening is allowed (status), editing a closed task's details is not.
		if req.Status == nil || domain.MaintenanceStatus(*req.Status) == task.Status {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "this task is closed; change its status to re-open it"})
		}
	}

	if req.MaintenanceType != nil {
		task.MaintenanceType = domain.MaintenanceType(*req.MaintenanceType)
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Priority != nil {
		task.Priority = domain.TaskPriority(*req.Priority)
	}
	if req.ScheduledDate != nil {
		task.ScheduledDate = req.ScheduledDate
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}
	if req.ContractorName != nil {
		task.ContractorName = *req.ContractorName
	}
	if req.ContractorContact != nil {
		task.ContractorContact = *req.ContractorContact
	}
	if req.EstimatedCost != nil {
		task.EstimatedCost = req.EstimatedCost
	}
	if req.ActualCost != nil {
		task.ActualCost = req.ActualCost
	}
	if req.Status != nil {
		task.Status = domain.MaintenanceStatus(*req.Status)
	}
	if req.AssignedTo != nil {
		task.AssignedTo = req.AssignedTo
	}
	if req.Notes != nil {
		task.Notes = *req.Notes
	}

	if err := validateMaintenanceTask(task); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.maintenanceRepo.Update(c.Context(), task); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"error": "assignee not found"})
		}
		return serverError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": task})
}

// @Summary Complete maintenance task
// @Tags Maintenance
// @Security Bearer
// @Param id path string true "Task ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/maintenance-tasks/{id}/complete [post]
func (h *MaintenanceTaskHandler) Complete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	task, err := h.maintenanceRepo.Get(c.Context(), id)
	if err != nil {
		return taskLookupError(c, err)
	}

	// The web client posts no body when no cost is entered; that used to be "invalid request body".
	var req struct {
		ActualCost *float64 `json:"actual_cost"`
	}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, err)
		}
	}
	if task.Status == domain.MaintenanceCompleted || task.Status == domain.MaintenanceCancelled {
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "this task is already " + string(task.Status)})
	}

	task.Status = domain.MaintenanceCompleted
	if req.ActualCost != nil {
		task.ActualCost = req.ActualCost
	}
	if err := validateMaintenanceTask(task); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.maintenanceRepo.Update(c.Context(), task); err != nil {
		return serverError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": task})
}

// @Summary Add photo to maintenance task
// @Tags Maintenance
// @Security Bearer
// @Param id path string true "Task ID"
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/maintenance-tasks/{id}/photos [post]
func (h *MaintenanceTaskHandler) AddPhoto(c *fiber.Ctx) error {
	taskID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	var req struct {
		PhotoURL   string `json:"photo_url"`
		PhotoStage string `json:"photo_stage"` // before/during/after
	}
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, err)
	}
	// The URL is rendered as a link: only http(s) or site-relative addresses may be stored.
	req.PhotoURL = strings.TrimSpace(req.PhotoURL)
	if req.PhotoURL == "" || tooLong(req.PhotoURL, 500) || !safeLink(req.PhotoURL) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "photo_url must be an http(s) URL"})
	}
	if req.PhotoStage == "" {
		req.PhotoStage = "after"
	}
	if !validPhotoStages[req.PhotoStage] {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "photo_stage must be one of: before, during, after"})
	}

	photo := &domain.MaintenancePhoto{
		ID:         uuid.New(),
		TaskID:     taskID,
		PhotoURL:   req.PhotoURL,
		PhotoStage: req.PhotoStage,
	}

	if err := h.maintenanceRepo.AddPhoto(c.Context(), photo); err != nil {
		return taskLookupError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": photo})
}

// @Summary Get task photos
// @Tags Maintenance
// @Security Bearer
// @Param id path string true "Task ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/maintenance-tasks/{id}/photos [get]
func (h *MaintenanceTaskHandler) GetPhotos(c *fiber.Ctx) error {
	taskID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	if _, err := h.maintenanceRepo.Get(c.Context(), taskID); err != nil {
		return taskLookupError(c, err)
	}
	photos, err := h.maintenanceRepo.GetPhotos(c.Context(), taskID)
	if err != nil {
		return serverError(c, err)
	}
	if photos == nil {
		photos = []domain.MaintenancePhoto{}
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": photos})
}

// @Summary Delete maintenance task
// @Tags Maintenance
// @Security Bearer
// @Param id path string true "Task ID"
// @Success 204
// @Router /api/v1/maintenance-tasks/{id} [delete]
func (h *MaintenanceTaskHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid task ID"})
	}

	if err := h.maintenanceRepo.Delete(c.Context(), id); err != nil {
		return serverError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}

// taskLookupError answers 404 for a task that does not exist in the caller's company and a generic
// error for anything else (every failure used to be reported as "not found" or a bare 500).
func taskLookupError(c *fiber.Ctx, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Task not found"})
	}
	return serverError(c, err)
}
