package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type TenantHandler struct {
	tenants *repo.TenantRepo
}

func NewTenantHandler(tenants *repo.TenantRepo) *TenantHandler {
	return &TenantHandler{tenants: tenants}
}

// List godoc
// @Summary      List tenants
// @Description  Returns a paginated list of tenants for the company.
// @Tags         Tenants
// @Produce      json
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Page size 1-100 (default 20)"
// @Success      200    {object}  domain.PaginatedResult[domain.Tenant]
// @Security     BearerAuth
// @Router       /tenants [get]
func (h *TenantHandler) List(c *fiber.Ctx) error {
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

	result, err := h.tenants.List(c.Context(), companyID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get tenant
// @Description  Returns a single tenant by UUID.
// @Tags         Tenants
// @Produce      json
// @Param        id  path      string  true  "Tenant UUID"
// @Success      200  {object}  domain.Tenant
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /tenants/{id} [get]
func (h *TenantHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	tenant, err := h.tenants.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tenant not found"})
	}
	return c.JSON(tenant)
}

// Create godoc
// @Summary      Create tenant
// @Description  Creates a new tenant. full_name_en and id_type are required.
// @Tags         Tenants
// @Accept       json
// @Produce      json
// @Param        body  body      domain.Tenant  true  "Tenant payload"
// @Success      201   {object}  domain.Tenant
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /tenants [post]
func (h *TenantHandler) Create(c *fiber.Ctx) error {
	var t domain.Tenant
	if err := c.BodyParser(&t); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if t.FullNameEN == "" || t.IDType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "full_name_en and id_type are required"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	t.CreatedBy = &userID
	t.UpdatedBy = &userID

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	t.CompanyID = companyID

	if err := h.tenants.Create(c.Context(), &t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(t)
}

// Update godoc
// @Summary      Update tenant
// @Description  Updates an existing tenant.
// @Tags         Tenants
// @Accept       json
// @Produce      json
// @Param        id    path      string        true  "Tenant UUID"
// @Param        body  body      domain.Tenant  true  "Tenant payload"
// @Success      200   {object}  domain.Tenant
// @Failure      400   {object}  object{error=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /tenants/{id} [patch]
func (h *TenantHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	t, err := h.tenants.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tenant not found"})
	}

	if err := c.BodyParser(t); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	t.UpdatedBy = &userID

	if err := h.tenants.Update(c.Context(), t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(t)
}

// Delete godoc
// @Summary      Delete tenant
// @Description  Deletes a tenant by UUID.
// @Tags         Tenants
// @Param        id  path  string  true  "Tenant UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /tenants/{id} [delete]
func (h *TenantHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.tenants.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Verify godoc
// @Summary      Mark tenant as verified
// @Description  Marks a tenant as verified after ID document validation.
// @Tags         Tenants
// @Accept       json
// @Produce      json
// @Param        id    path      string                          true  "Tenant UUID"
// @Param        body  body      object{verification_notes=string}  false "Verification notes"
// @Success      200   {object}  domain.Tenant
// @Failure      400   {object}  object{error=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /tenants/{id}/verify [post]
func (h *TenantHandler) Verify(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	t, err := h.tenants.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "tenant not found"})
	}

	var body struct {
		VerificationNotes string `json:"verification_notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	t.IsVerified = true
	t.VerificationStatus = domain.VerificationVerified
	t.VerificationNotes = body.VerificationNotes
	t.VerifiedBy = &userID
	t.UpdatedBy = &userID

	if err := h.tenants.Update(c.Context(), t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(t)
}
