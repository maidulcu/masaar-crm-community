package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type LeaseHandler struct {
	leases *repo.LeaseRepo
}

func NewLeaseHandler(leases *repo.LeaseRepo) *LeaseHandler {
	return &LeaseHandler{leases: leases}
}

// List godoc
// @Summary      List leases
// @Description  Returns a paginated list of leases for the company.
// @Tags         Leases
// @Produce      json
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Page size 1-100 (default 20)"
// @Success      200    {object}  domain.PaginatedResult[domain.Lease]
// @Security     BearerAuth
// @Router       /leases [get]
func (h *LeaseHandler) List(c *fiber.Ctx) error {
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

	result, err := h.leases.List(c.Context(), companyID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get lease
// @Description  Returns a single lease by UUID.
// @Tags         Leases
// @Produce      json
// @Param        id  path      string  true  "Lease UUID"
// @Success      200  {object}  domain.Lease
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leases/{id} [get]
func (h *LeaseHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	lease, err := h.leases.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "lease not found"})
	}
	return c.JSON(lease)
}

// Create godoc
// @Summary      Create lease
// @Description  Creates a new lease. property_id, tenant_id, monthly_rent, start_date, end_date, and payment_frequency are required.
// @Tags         Leases
// @Accept       json
// @Produce      json
// @Param        body  body      domain.Lease  true  "Lease payload"
// @Success      201   {object}  domain.Lease
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leases [post]
func (h *LeaseHandler) Create(c *fiber.Ctx) error {
	var l domain.Lease
	if err := c.BodyParser(&l); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if l.PropertyID == uuid.Nil || l.TenantID == uuid.Nil || l.MonthlyRent <= 0 || l.PaymentFrequency == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "property_id, tenant_id, monthly_rent > 0, and payment_frequency are required"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	l.CreatedBy = &userID
	l.UpdatedBy = &userID

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	l.CompanyID = companyID

	if err := h.leases.Create(c.Context(), &l); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(l)
}

// Update godoc
// @Summary      Update lease
// @Description  Updates an existing lease.
// @Tags         Leases
// @Accept       json
// @Produce      json
// @Param        id    path      string       true  "Lease UUID"
// @Param        body  body      domain.Lease  true  "Lease payload"
// @Success      200   {object}  domain.Lease
// @Failure      400   {object}  object{error=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leases/{id} [patch]
func (h *LeaseHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	l, err := h.leases.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "lease not found"})
	}

	if err := c.BodyParser(l); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	l.UpdatedBy = &userID

	if err := h.leases.Update(c.Context(), l); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(l)
}

// Delete godoc
// @Summary      Delete lease
// @Description  Deletes a lease by UUID.
// @Tags         Leases
// @Param        id  path  string  true  "Lease UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leases/{id} [delete]
func (h *LeaseHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.leases.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
