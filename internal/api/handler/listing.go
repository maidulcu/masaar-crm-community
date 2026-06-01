package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type ListingHandler struct {
	repo         *repo.ListingRepo
	approvalRepo *repo.ApprovalRepo
}

func NewListingHandler(repo *repo.ListingRepo, approvalRepo *repo.ApprovalRepo) *ListingHandler {
	return &ListingHandler{repo: repo, approvalRepo: approvalRepo}
}

// List godoc
// @Summary      List listings
// @Description  Returns a paginated list of listings for the company.
// @Tags         Listings
// @Produce      json
// @Param        page   query  int  false  "Page number (default 1)"
// @Param        limit  query  int  false  "Page size 1-100 (default 20)"
// @Success      200  {object}  domain.PaginatedResult[domain.Listing]
// @Security     BearerAuth
// @Router       /listings [get]
func (h *ListingHandler) List(c *fiber.Ctx) error {
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

	result, err := h.repo.List(c.Context(), companyID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get listing
// @Description  Returns a single listing by UUID.
// @Tags         Listings
// @Produce      json
// @Param        id  path  string  true  "Listing UUID"
// @Success      200  {object}  domain.Listing
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /listings/{id} [get]
func (h *ListingHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	listing, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
	}
	return c.JSON(listing)
}

// Create godoc
// @Summary      Create listing
// @Description  Creates a new listing. Title, property_type, listing_type, and price are required.
// @Tags         Listings
// @Accept       json
// @Produce      json
// @Param        body  body  domain.Listing  true  "Listing payload"
// @Success      201  {object}  domain.Listing
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /listings [post]
func (h *ListingHandler) Create(c *fiber.Ctx) error {
	var l domain.Listing
	if err := c.BodyParser(&l); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if l.Title == "" || l.PropertyType == "" || l.Price == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title, property_type, and price are required"})
	}
	if l.ListingType == "" {
		l.ListingType = domain.ListingTypeRent
	}
	if l.Status == "" {
		l.Status = domain.ListingStatusDraft
	}
	if l.Currency == "" {
		l.Currency = "AED"
	}

	userID := c.Locals("user_id").(uuid.UUID)
	l.CreatedBy = &userID
	l.UpdatedBy = &userID

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	l.CompanyID = companyID

	if err := h.repo.Create(c.Context(), &l); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(l)
}

// Update godoc
// @Summary      Update listing
// @Description  Updates an existing listing.
// @Tags         Listings
// @Accept       json
// @Produce      json
// @Param        id    path  string         true  "Listing UUID"
// @Param        body  body  domain.Listing  true  "Listing payload"
// @Success      200  {object}  domain.Listing
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /listings/{id} [patch]
func (h *ListingHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	l, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
	}

	if err := c.BodyParser(l); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	l.UpdatedBy = &userID

	if err := h.repo.Update(c.Context(), l); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(l)
}

// UpdateStatus godoc
// @Summary      Update listing status
// @Description  Updates the status of a listing (draft, published, sold, rented, expired, withdrawn).
// @Tags         Listings
// @Produce      json
// @Param        id    path  string  true  "Listing UUID"
// @Param        body  body  object{status=string}  true  "New status"
// @Success      200  {object}  object{status=string}
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /listings/{id}/status [patch]
func (h *ListingHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	userID, err := uuid.Parse(c.Locals("user_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status is required"})
	}

	// If trying to publish, check if approval is required
	if body.Status == string(domain.ListingStatusPublished) {
		cfg, _ := h.approvalRepo.GetConfig(c.Context(), companyID)
		if cfg != nil && cfg.ListingApproval {
			req := &domain.ApprovalRequest{
				CompanyID:   companyID,
				EntityType:  domain.ApprovalListing,
				EntityID:    id,
				RequestedBy: userID,
				Notes:       "Request to publish listing",
			}
			if err := h.approvalRepo.Create(c.Context(), req); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create approval request"})
			}
			return c.JSON(fiber.Map{"status": "pending_approval", "approval_id": req.ID})
		}
	}

	if err := h.repo.UpdateStatus(c.Context(), id, domain.ListingStatus(body.Status)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": body.Status})
}

// Delete godoc
// @Summary      Delete listing
// @Description  Deletes a listing by UUID.
// @Tags         Listings
// @Param        id  path  string  true  "Listing UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /listings/{id} [delete]
func (h *ListingHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.repo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
