package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/compliance"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type ListingHandler struct {
	repo         *repo.ListingRepo
	approvalRepo *repo.ApprovalRepo
	audit        auditLogger
}

func NewListingHandler(repo *repo.ListingRepo, approvalRepo *repo.ApprovalRepo, audit auditLogger) *ListingHandler {
	return &ListingHandler{repo: repo, approvalRepo: approvalRepo, audit: audit}
}

// permitBlocked answers 422 when the listing may not be advertised under its emirate's
// permit rules, and returns true; otherwise it returns false and writes nothing.
func permitBlocked(c *fiber.Ctx, l *domain.Listing) (bool, error) {
	st := compliance.ListingPermit(l, time.Now())
	if st.Advertisable() {
		return false, nil
	}
	msg := "this listing needs a valid advertising permit (" + st.System + ") before it can be published or shared"
	if st.Problem == "expired" {
		msg = "the listing's advertising permit (" + st.System + ") has expired; renew it before publishing or sharing"
	}
	return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg, "permit": st})
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
		return serverError(c, err)
	}
	now := time.Now()
	for i := range result.Data {
		compliance.Decorate(&result.Data[i], now)
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
	compliance.Decorate(listing, time.Now())
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
		return badRequest(c, err)
	}

	if l.Title == "" || l.PropertyType == "" || l.Price == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title, property_type, and price are required"})
	}
	if l.ListingType == "" {
		l.ListingType = domain.ListingTypeRent
	}
	// New listings start as drafts; publishing goes through PATCH /listings/:id/status,
	// which applies the approval and advertising-permit rules.
	l.Status = domain.ListingStatusDraft
	if l.Currency == "" {
		l.Currency = defaultCurrency(c.Context())
	}
	l.PermitNumber = strings.TrimSpace(l.PermitNumber)
	l.PermitOverrideBy, l.PermitOverrideAt, l.PermitOverrideReason = nil, nil, ""

	userID := c.Locals("user_id").(uuid.UUID)
	l.CreatedBy = &userID
	l.UpdatedBy = &userID

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	l.CompanyID = companyID

	if err := h.repo.Create(c.Context(), &l); err != nil {
		return serverError(c, err)
	}
	l.Country = defaultCountry(c.Context()) // the permit rules are the company's country's
	compliance.Decorate(&l, time.Now())
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

	status, country := l.Status, l.Country
	if err := c.BodyParser(l); err != nil {
		return badRequest(c, err)
	}
	// Status changes go through UpdateStatus (approval and permit rules apply there).
	l.Status, l.Country = status, country
	l.PermitNumber = strings.TrimSpace(l.PermitNumber)

	userID := c.Locals("user_id").(uuid.UUID)
	l.UpdatedBy = &userID

	if err := h.repo.Update(c.Context(), l); err != nil {
		return serverError(c, err)
	}
	compliance.Decorate(l, time.Now())
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

	// The auth middleware stores user_id as a uuid.UUID (asserting a string here used to
	// panic, so every status change answered 500).
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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

	// If trying to publish: the advertising permit must be valid (or overridden), then
	// approval may be required.
	if body.Status == string(domain.ListingStatusPublished) {
		l, err := h.repo.GetByID(c.Context(), id)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
		}
		if blocked, err := permitBlocked(c, l); blocked {
			return err
		}
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
		return serverError(c, err)
	}

	return c.JSON(fiber.Map{"status": body.Status})
}

// SetPermitOverride godoc
// @Summary      Allow advertising without a valid permit
// @Description  Admin only. Lets a listing whose advertising permit is missing or expired be published and shared anyway. A reason is required and the override is audit-logged. DELETE clears it.
// @Tags         Listings
// @Accept       json
// @Param        id    path  string                 true  "Listing UUID"
// @Param        body  body  object{reason=string}  true  "Why"
// @Success      204
// @Security     BearerAuth
// @Router       /listings/{id}/permit-override [post]
func (h *ListingHandler) SetPermitOverride(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if body.Reason == "" || len(body.Reason) > 500 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "a reason (up to 500 characters) is required"})
	}
	if err := h.repo.SetPermitOverride(c.Context(), id, &userID, body.Reason); err != nil {
		return serverError(c, err)
	}
	if h.audit != nil {
		h.audit.Log(c.Context(), userID, repo.AuditUpdate, "listing", id, fiber.Map{"permit_override": true, "reason": body.Reason})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ClearPermitOverride godoc
// @Summary      Remove a permit override
// @Tags         Listings
// @Param        id  path  string  true  "Listing UUID"
// @Success      204
// @Security     BearerAuth
// @Router       /listings/{id}/permit-override [delete]
func (h *ListingHandler) ClearPermitOverride(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	if err := h.repo.SetPermitOverride(c.Context(), id, nil, ""); err != nil {
		return serverError(c, err)
	}
	if h.audit != nil {
		h.audit.Log(c.Context(), userID, repo.AuditUpdate, "listing", id, fiber.Map{"permit_override": false})
	}
	return c.SendStatus(fiber.StatusNoContent)
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
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
