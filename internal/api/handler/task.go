package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// TaskHandler serves tasks and follow-ups.
type TaskHandler struct {
	tasks *repo.TaskRepo
}

func NewTaskHandler(tasks *repo.TaskRepo) *TaskHandler { return &TaskHandler{tasks: tasks} }

type taskInput struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Type        string     `json:"type"`
	Priority    string     `json:"priority"`
	DueAt       *time.Time `json:"due_at"`
	AssignedTo  *uuid.UUID `json:"assigned_to"`
	RelatedType *string    `json:"related_type"`
	RelatedID   *uuid.UUID `json:"related_id"`
}

var (
	taskTypes      = map[string]bool{"call": true, "meeting": true, "follow_up": true, "todo": true}
	taskPriorities = map[string]bool{"low": true, "normal": true, "high": true}
)

// apply validates in and copies it onto t; it returns a client-facing message or "".
func (in *taskInput) apply(t *domain.Task) string {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || len(in.Title) > 300 {
		return "title is required (up to 300 characters)"
	}
	if len(in.Description) > 5000 {
		return "description is too long"
	}
	if in.Type == "" {
		in.Type = "follow_up"
	}
	if !taskTypes[in.Type] {
		return "type must be call, meeting, follow_up or todo"
	}
	if in.Priority == "" {
		in.Priority = "normal"
	}
	if !taskPriorities[in.Priority] {
		return "priority must be low, normal or high"
	}
	if in.RelatedType != nil && *in.RelatedType == "" {
		in.RelatedType = nil
	}
	if (in.RelatedType == nil) != (in.RelatedID == nil) {
		return "related_type and related_id go together"
	}
	if in.RelatedType != nil && !repo.ValidRelatedType(*in.RelatedType) {
		return "related_type must be lead, contact, deal, listing, viewing or offer"
	}
	t.Title, t.Description, t.Type, t.Priority = in.Title, in.Description, in.Type, in.Priority
	t.DueAt, t.AssignedTo, t.RelatedType, t.RelatedID = in.DueAt, in.AssignedTo, in.RelatedType, in.RelatedID
	return ""
}

// List godoc
// @Summary      List tasks
// @Description  Filters: mine=true (assigned to me), status=open|done|cancelled, overdue=true, related_type + related_id, assigned_to.
// @Tags         Tasks
// @Produce      json
// @Success      200  {object}  object{data=[]domain.Task,total=int}
// @Security     BearerAuth
// @Router       /tasks [get]
func (h *TaskHandler) List(c *fiber.Ctx) error {
	page, limit := pageParams(c, 50, 200)
	f := domain.TaskFilter{Status: c.Query("status"), RelatedType: c.Query("related_type"),
		OverdueOnly: c.QueryBool("overdue"), Limit: limit, Offset: (page - 1) * limit}
	if c.QueryBool("mine") {
		f.AssignedTo = localsUserPtr(c)
	} else if s := c.Query("assigned_to"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid assigned_to"})
		}
		f.AssignedTo = &id
	}
	if s := c.Query("related_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid related_id"})
		}
		f.RelatedID = &id
	}
	tasks, total, err := h.tasks.List(c.Context(), f)
	if err != nil {
		return serverError(c, err)
	}
	if tasks == nil {
		tasks = []domain.Task{}
	}
	return c.JSON(fiber.Map{"data": tasks, "total": total})
}

// Create godoc
// @Summary      Create a task
// @Description  Unassigned tasks are assigned to the caller.
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        body  body  taskInput  true  "Task"
// @Success      201  {object}  domain.Task
// @Security     BearerAuth
// @Router       /tasks [post]
func (h *TaskHandler) Create(c *fiber.Ctx) error {
	var in taskInput
	if err := c.BodyParser(&in); err != nil {
		return badRequest(c, err)
	}
	t := &domain.Task{CreatedBy: localsUserPtr(c)}
	if msg := in.apply(t); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
	}
	if t.AssignedTo == nil {
		t.AssignedTo = t.CreatedBy
	}
	if err := h.tasks.Create(c.Context(), t); err != nil {
		return serverError(c, err)
	}
	created, err := h.tasks.Get(c.Context(), t.ID)
	if err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

// Update godoc
// @Summary      Update a task
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        id    path  string     true  "Task UUID"
// @Param        body  body  taskInput  true  "Task"
// @Success      200  {object}  domain.Task
// @Security     BearerAuth
// @Router       /tasks/{id} [patch]
func (h *TaskHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	t, err := h.tasks.Get(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	in := taskInput{Title: t.Title, Description: t.Description, Type: t.Type, Priority: t.Priority,
		DueAt: t.DueAt, AssignedTo: t.AssignedTo, RelatedType: t.RelatedType, RelatedID: t.RelatedID}
	if err := c.BodyParser(&in); err != nil {
		return badRequest(c, err)
	}
	if msg := in.apply(t); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
	}
	if err := h.tasks.Update(c.Context(), t); err != nil {
		return serverError(c, err)
	}
	updated, err := h.tasks.Get(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(updated)
}

// SetStatus godoc
// @Summary      Complete, cancel or reopen a task
// @Tags         Tasks
// @Accept       json
// @Param        id    path  string                 true  "Task UUID"
// @Param        body  body  object{status=string}  true  "open | done | cancelled"
// @Success      204
// @Security     BearerAuth
// @Router       /tasks/{id}/status [patch]
func (h *TaskHandler) SetStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if body.Status != "open" && body.Status != "done" && body.Status != "cancelled" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "status must be open, done or cancelled"})
	}
	if err := h.tasks.SetStatus(c.Context(), id, body.Status); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Delete godoc
// @Summary      Delete a task
// @Tags         Tasks
// @Param        id  path  string  true  "Task UUID"
// @Success      204
// @Security     BearerAuth
// @Router       /tasks/{id} [delete]
func (h *TaskHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.tasks.Delete(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
