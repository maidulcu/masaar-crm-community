package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type RentalPropertyHandler struct {
	properties *repo.RentalPropertyRepo
}

func NewRentalPropertyHandler(properties *repo.RentalPropertyRepo) *RentalPropertyHandler {
	return &RentalPropertyHandler{properties: properties}
}

// List godoc
// @Summary      List rental properties
// @Description  Returns a paginated list of rental properties for the company.
// @Tags         Rental Properties
// @Produce      json
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Page size 1-100 (default 20)"
// @Success      200    {object}  domain.PaginatedResult[domain.RentalProperty]
// @Security     BearerAuth
// @Router       /rental-properties [get]
func (h *RentalPropertyHandler) List(c *fiber.Ctx) error {
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

	result, err := h.properties.List(c.Context(), companyID, page, limit)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get rental property
// @Description  Returns a single rental property by UUID.
// @Tags         Rental Properties
// @Produce      json
// @Param        id  path      string  true  "Property UUID"
// @Success      200  {object}  domain.RentalProperty
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /rental-properties/{id} [get]
func (h *RentalPropertyHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	property, err := h.properties.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "property not found"})
	}
	return c.JSON(property)
}

// Create godoc
// @Summary      Create rental property
// @Description  Creates a new rental property. Name and property_type are required.
// @Tags         Rental Properties
// @Accept       json
// @Produce      json
// @Param        body  body      domain.RentalProperty  true  "Property payload"
// @Success      201   {object}  domain.RentalProperty
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /rental-properties [post]
func (h *RentalPropertyHandler) Create(c *fiber.Ctx) error {
	var in domain.RentalProperty
	if err := json.Unmarshal(normalizeDateFields(c.Body(), "purchase_date"), &in); err != nil {
		return badRequest(c, err)
	}
	// Server-owned fields are never taken from the client.
	p := in
	p.ID, p.CreatedAt, p.UpdatedAt = uuid.Nil, time.Time{}, time.Time{}
	if err := validateRentalProperty(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	p.CreatedBy = &userID
	p.UpdatedBy = &userID

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	p.CompanyID = companyID

	if err := h.properties.Create(c.Context(), &p); err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

// Update godoc
// @Summary      Update rental property
// @Description  Updates an existing rental property.
// @Tags         Rental Properties
// @Accept       json
// @Produce      json
// @Param        id    path      string                 true  "Property UUID"
// @Param        body  body      domain.RentalProperty  true  "Property payload"
// @Success      200   {object}  domain.RentalProperty
// @Failure      400   {object}  object{error=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /rental-properties/{id} [patch]
func (h *RentalPropertyHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	p, err := h.properties.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "property not found"})
	}

	// The id in the body must not redirect the update to another property, and the record's
	// identity and audit fields are not the client's to edit.
	keep := *p
	if err := json.Unmarshal(normalizeDateFields(c.Body(), "purchase_date"), p); err != nil {
		return badRequest(c, err)
	}
	p.ID, p.CompanyID, p.CreatedAt, p.CreatedBy = keep.ID, keep.CompanyID, keep.CreatedAt, keep.CreatedBy
	if err := validateRentalProperty(p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	p.UpdatedBy = &userID

	if err := h.properties.Update(c.Context(), p); err != nil {
		return serverError(c, err)
	}
	return c.JSON(p)
}

// Delete godoc
// @Summary      Delete rental property
// @Description  Deletes a rental property by UUID.
// @Tags         Rental Properties
// @Param        id  path  string  true  "Property UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /rental-properties/{id} [delete]
func (h *RentalPropertyHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.properties.Delete(c.Context(), id); err != nil {
		if errors.Is(err, repo.ErrPropertyHasWork) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this property has maintenance tasks or inspections and cannot be deleted; mark it inactive instead"})
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			// leases (RESTRICT) and expenses reference the property: they are financial records.
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this property has leases or expenses and cannot be deleted; mark it inactive instead"})
		}
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
