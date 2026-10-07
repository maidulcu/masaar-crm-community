package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

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
		return serverError(c, err)
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
	body := normalizeDateFields(c.Body(), leaseDateKeys...)
	var in domain.Lease
	if err := json.Unmarshal(body, &in); err != nil {
		return badRequest(c, err)
	}
	// The column default for auto_generate_payments is true, but Go's zero value for an omitted
	// field is false and was inserted explicitly, silently switching payment generation off.
	var keys map[string]json.RawMessage
	_ = json.Unmarshal(body, &keys)

	l := in
	l.ID, l.CreatedAt, l.UpdatedAt = uuid.Nil, time.Time{}, time.Time{}
	l.LastGeneratedPaymentDt, l.TerminationDate, l.TerminationReason = nil, nil, ""
	if _, given := keys["auto_generate_payments"]; !given {
		l.AutoGeneratePayments = true
	}
	if l.PropertyID == uuid.Nil || l.TenantID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "property_id and tenant_id are required"})
	}
	l.StartDate, l.EndDate = dateOnly(l.StartDate), dateOnly(l.EndDate)
	if in.Status != "" && in.Status != domain.LeaseStatusActive {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "a new lease starts as active"})
	}
	if err := validateLease(&l); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
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
		if errors.Is(err, repo.ErrLeaseOverlap) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this property already has an active lease overlapping those dates"})
		}
		return serverError(c, err)
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

	// Who, what and when a lease covers cannot be changed by editing it (the UPDATE never wrote
	// those columns, yet the response echoed whatever the client sent), and an id in the body
	// would redirect the update to a different lease. A new term is a new lease or a renewal.
	keep := *l
	if err := json.Unmarshal(normalizeDateFields(c.Body(), leaseDateKeys...), l); err != nil {
		return badRequest(c, err)
	}
	l.ID, l.CompanyID, l.PropertyID, l.TenantID, l.TemplateID = keep.ID, keep.CompanyID, keep.PropertyID, keep.TenantID, keep.TemplateID
	l.StartDate, l.EndDate, l.CreatedAt, l.CreatedBy = keep.StartDate, keep.EndDate, keep.CreatedAt, keep.CreatedBy
	l.LastGeneratedPaymentDt = keep.LastGeneratedPaymentDt // maintained by the payment generator
	l.Currency = keep.Currency                             // the UPDATE never wrote it
	if err := validateLease(l); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if l.Status == domain.LeaseStatusTerminated && keep.Status != domain.LeaseStatusTerminated && l.TerminationDate == nil {
		today := dateOnly(time.Now())
		l.TerminationDate = &today
	}
	// Re-activating a lease must not double-book the property.
	if l.Status == domain.LeaseStatusActive && keep.Status != domain.LeaseStatusActive {
		if clash, err := h.leases.HasActiveOverlap(c.Context(), l.PropertyID, l.StartDate, l.EndDate, l.ID); err != nil {
			return serverError(c, err)
		} else if clash {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this property already has an active lease overlapping those dates"})
		}
	}

	userID := c.Locals("user_id").(uuid.UUID)
	l.UpdatedBy = &userID

	if err := h.leases.Update(c.Context(), l); err != nil {
		return serverError(c, err)
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
		if isForeignKeyViolation(err) {
			// payments are ON DELETE RESTRICT: they are the rent ledger.
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this lease has payments and cannot be deleted; terminate it instead"})
		}
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
