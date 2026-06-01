package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type LeaseTemplateHandler struct {
	templates *repo.LeaseTemplateRepo
}

func NewLeaseTemplateHandler(templates *repo.LeaseTemplateRepo) *LeaseTemplateHandler {
	return &LeaseTemplateHandler{templates: templates}
}

// List godoc
// @Summary      List lease templates
// @Description  Returns a paginated list of lease templates for the company.
// @Tags         Lease Templates
// @Produce      json
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Page size 1-100 (default 20)"
// @Success      200    {object}  domain.PaginatedResult[domain.LeaseTemplate]
// @Security     BearerAuth
// @Router       /lease-templates [get]
func (h *LeaseTemplateHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	result, err := h.templates.List(c.Context(), companyID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get lease template
// @Description  Returns a single lease template by UUID.
// @Tags         Lease Templates
// @Produce      json
// @Param        id  path      string  true  "Template UUID"
// @Success      200  {object}  domain.LeaseTemplate
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /lease-templates/{id} [get]
func (h *LeaseTemplateHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	template, err := h.templates.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
	}
	return c.JSON(template)
}

// Create godoc
// @Summary      Create lease template
// @Description  Creates a new lease template. Name and payment_frequency are required.
// @Tags         Lease Templates
// @Accept       json
// @Produce      json
// @Param        body  body      domain.LeaseTemplate  true  "Template payload"
// @Success      201   {object}  domain.LeaseTemplate
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /lease-templates [post]
func (h *LeaseTemplateHandler) Create(c *fiber.Ctx) error {
	var t domain.LeaseTemplate
	if err := c.BodyParser(&t); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if t.Name == "" || t.PaymentFrequency == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and payment_frequency are required"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	t.CreatedBy = &userID
	t.UpdatedBy = &userID

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	t.CompanyID = companyID

	if err := h.templates.Create(c.Context(), &t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(t)
}

// Update godoc
// @Summary      Update lease template
// @Description  Updates an existing lease template.
// @Tags         Lease Templates
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Template UUID"
// @Param        body  body      domain.LeaseTemplate  true  "Template payload"
// @Success      200   {object}  domain.LeaseTemplate
// @Failure      400   {object}  object{error=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /lease-templates/{id} [patch]
func (h *LeaseTemplateHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	t, err := h.templates.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
	}

	if err := c.BodyParser(t); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	t.UpdatedBy = &userID

	if err := h.templates.Update(c.Context(), t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(t)
}

// Delete godoc
// @Summary      Delete lease template
// @Description  Deletes a lease template by UUID.
// @Tags         Lease Templates
// @Param        id  path  string  true  "Template UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /lease-templates/{id} [delete]
func (h *LeaseTemplateHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.templates.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
