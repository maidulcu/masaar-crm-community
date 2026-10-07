package handler

import (
	"encoding/json"
	"strconv"
	"time"

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
		return serverError(c, err)
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
	var in domain.Tenant
	if err := json.Unmarshal(normalizeDateFields(c.Body(), tenantDateKeys...), &in); err != nil {
		return badRequest(c, err)
	}
	// A new tenant is always unverified: verification is admin-only (POST /tenants/:id/verify) and
	// used to be settable here. Ids and timestamps are server-owned.
	t := in
	t.ID, t.CreatedAt, t.UpdatedAt = uuid.Nil, time.Time{}, time.Time{}
	t.IsVerified, t.VerificationStatus, t.VerificationDate, t.VerifiedBy, t.VerificationNotes = false, domain.VerificationPending, nil, nil, ""
	if err := validateTenant(&t); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
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
		if isUniqueViolation(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "a tenant with this id_number already exists"})
		}
		return serverError(c, err)
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

	// Identity, audit and verification state are not the client's to edit: an agent could PATCH
	// is_verified/verification_status and skip the admin-only verify step, and an id in the body
	// would redirect the update to another tenant.
	keep := *t
	if err := json.Unmarshal(normalizeDateFields(c.Body(), tenantDateKeys...), t); err != nil {
		return badRequest(c, err)
	}
	t.ID, t.CompanyID, t.CreatedAt, t.CreatedBy = keep.ID, keep.CompanyID, keep.CreatedAt, keep.CreatedBy
	t.IsVerified, t.VerificationStatus, t.VerificationDate = keep.IsVerified, keep.VerificationStatus, keep.VerificationDate
	t.VerifiedBy, t.VerificationNotes = keep.VerifiedBy, keep.VerificationNotes
	if err := validateTenant(t); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	t.UpdatedBy = &userID

	if err := h.tenants.Update(c.Context(), t); err != nil {
		if isUniqueViolation(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "a tenant with this id_number already exists"})
		}
		return serverError(c, err)
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
		if isForeignKeyViolation(err) {
			// leases (RESTRICT) and expenses reference the tenant: they are financial records.
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this tenant has leases or expenses and cannot be deleted; mark them inactive instead"})
		}
		return serverError(c, err)
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
		return badRequest(c, err)
	}

	userID := c.Locals("user_id").(uuid.UUID)
	now := time.Now()
	t.IsVerified = true
	t.VerificationDate = &now // was never recorded
	t.VerificationStatus = domain.VerificationVerified
	t.VerificationNotes = body.VerificationNotes
	t.VerifiedBy = &userID
	t.UpdatedBy = &userID

	if err := h.tenants.Update(c.Context(), t); err != nil {
		return serverError(c, err)
	}
	return c.JSON(t)
}
