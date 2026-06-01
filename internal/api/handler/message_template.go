package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type MessageTemplateHandler struct {
	templates *repo.MessageTemplateRepo
}

func NewMessageTemplateHandler(templates *repo.MessageTemplateRepo) *MessageTemplateHandler {
	return &MessageTemplateHandler{templates: templates}
}

// List returns paginated message templates for the company
// @Summary      List message templates
// @Tags         Message Templates
// @Produce      json
// @Param        page  query  int  false  "Page (default 1)"
// @Param        limit query  int  false  "Limit (default 20)"
// @Success      200  {object}  domain.PaginatedResult[domain.MessageTemplate]
// @Security     BearerAuth
// @Router       /message-templates [get]
func (h *MessageTemplateHandler) List(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	result, err := h.templates.List(c.Context(), companyID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Get returns a single message template
// @Summary      Get message template
// @Tags         Message Templates
// @Produce      json
// @Param        id   path  string  true  "Template UUID"
// @Success      200  {object}  domain.MessageTemplate
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /message-templates/{id} [get]
func (h *MessageTemplateHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	t, err := h.templates.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
	}
	return c.JSON(t)
}

type CreateMessageTemplateRequest struct {
	Name      string   `json:"name"`
	Body      string   `json:"body"`
	Category  string   `json:"category"`
	Variables []string `json:"variables"`
	IsActive  bool     `json:"is_active"`
}

// Create creates a new message template
// @Summary      Create message template
// @Tags         Message Templates
// @Accept       json
// @Produce      json
// @Param        request  body  CreateMessageTemplateRequest  true  "Template data"
// @Success      201  {object}  domain.MessageTemplate
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /message-templates [post]
func (h *MessageTemplateHandler) Create(c *fiber.Ctx) error {
	var req CreateMessageTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Name == "" || req.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and body are required"})
	}
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))
	userID := c.Locals("user_id").(uuid.UUID)

	tpl := &domain.MessageTemplate{
		CompanyID: companyID,
		Name:      req.Name,
		Body:      req.Body,
		Category:  req.Category,
		Variables: req.Variables,
		IsActive:  req.IsActive,
		CreatedBy: &userID,
		UpdatedBy: &userID,
	}
	if err := h.templates.Create(c.Context(), tpl); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(tpl)
}

// Update updates a message template
// @Summary      Update message template
// @Tags         Message Templates
// @Accept       json
// @Produce      json
// @Param        id       path  string                     true  "Template UUID"
// @Param        request  body  CreateMessageTemplateRequest  true  "Template data"
// @Success      200  {object}  domain.MessageTemplate
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /message-templates/{id} [patch]
func (h *MessageTemplateHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	tpl, err := h.templates.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
	}
	var req CreateMessageTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Name != "" {
		tpl.Name = req.Name
	}
	if req.Body != "" {
		tpl.Body = req.Body
	}
	if req.Category != "" {
		tpl.Category = req.Category
	}
	if req.Variables != nil {
		tpl.Variables = req.Variables
	}
	tpl.IsActive = req.IsActive
	userID := c.Locals("user_id").(uuid.UUID)
	tpl.UpdatedBy = &userID

	if err := h.templates.Update(c.Context(), tpl); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(tpl)
}

// Delete removes a message template
// @Summary      Delete message template
// @Tags         Message Templates
// @Produce      json
// @Param        id  path  string  true  "Template UUID"
// @Success      204
// @Security     BearerAuth
// @Router       /message-templates/{id} [delete]
func (h *MessageTemplateHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.templates.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
