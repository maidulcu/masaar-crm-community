package handler

import (
	"encoding/json"
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

// publishGate decides what happens when a listing is asked to go live. With approval off (or
// when the listing is already published) it publishes. With approval on it files one approval
// request (re-using an existing pending one instead of piling up duplicates) and leaves the
// listing as it is. approvalID is non-nil when the publish is waiting for an admin.
func (h *ListingHandler) publishGate(c *fiber.Ctx, companyID, userID, listingID uuid.UUID) (approvalID *uuid.UUID, err error) {
	cfg, _ := h.approvalRepo.GetConfig(c.Context(), companyID)
	if cfg == nil || !cfg.ListingApproval {
		return nil, h.repo.UpdateStatus(c.Context(), listingID, domain.ListingStatusPublished)
	}
	if pending, perr := h.approvalRepo.GetByEntity(c.Context(), domain.ApprovalListing, listingID); perr == nil && pending != nil && pending.Status == domain.ApprovalPending {
		return &pending.ID, nil
	}
	req := &domain.ApprovalRequest{
		CompanyID: companyID, EntityType: domain.ApprovalListing, EntityID: listingID,
		RequestedBy: userID, Notes: "Request to publish listing",
	}
	if err := h.approvalRepo.Create(c.Context(), req); err != nil {
		return nil, err
	}
	return &req.ID, nil
}

// Create godoc
// @Summary      Create listing
// @Description  Creates a new listing. Title, property_type and price are required. A listing is created as a draft; asking for `status: published` publishes it only when no approval is required, otherwise it stays a draft and an approval request is filed.
// @Tags         Listings
// @Accept       json
// @Produce      json
// @Param        body  body  domain.Listing  true  "Listing payload"
// @Success      201  {object}  domain.Listing
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /listings [post]
func (h *ListingHandler) Create(c *fiber.Ctx) error {
	var in domain.Listing
	if err := json.Unmarshal(normalizeDateFields(c.Body(), "available_from"), &in); err != nil {
		return badRequest(c, err)
	}

	// Server-owned fields are never taken from the client (id, company, timestamps, sync state).
	l := in
	// portal_sync_status is NOT NULL jsonb; a nil map is sent as SQL NULL, which made every
	// create that did not carry the field fail as "invalid data".
	l.ID, l.PublishedAt, l.PortalSyncStatus = uuid.Nil, nil, map[string]interface{}{}
	l.Status = domain.ListingStatusDraft
	if l.ListingType == "" {
		l.ListingType = domain.ListingTypeRent
	}
	if l.Currency == "" {
		l.Currency = defaultCurrency(c.Context())
	}
	l.PermitNumber = strings.TrimSpace(l.PermitNumber)
	l.PermitOverrideBy, l.PermitOverrideAt, l.PermitOverrideReason = nil, nil, ""
	if err := validateListing(&l); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	wantPublished := in.Status == domain.ListingStatusPublished
	if in.Status != "" && !validListingStatuses[in.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid status"})
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
		return serverError(c, err)
	}
	l.Country = defaultCountry(c.Context()) // the permit rules are the company's country's
	// Publishing straight from create goes through the same gates as PATCH /status: a valid
	// advertising permit (otherwise it stays a draft), then the approval workflow.
	if wantPublished && compliance.ListingPermit(&l, time.Now()).Advertisable() {
		if _, err := h.publishGate(c, companyID, userID, l.ID); err != nil {
			return serverError(c, err)
		}
		if fresh, err := h.repo.GetByID(c.Context(), l.ID); err == nil {
			l = *fresh
			l.Country = defaultCountry(c.Context())
		}
	}
	compliance.Decorate(&l, time.Now())
	return c.Status(fiber.StatusCreated).JSON(l)
}

// Update godoc
// @Summary      Update listing
// @Description  Updates an existing listing. The status is not changed here (use PATCH /listings/{id}/status, which applies the approval workflow).
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

	// Remember what the client may not change, apply the patch, then put it back. Status used
	// to be writable here, which let anyone publish a listing and skip the approval workflow;
	// the id in the body could also redirect the update to a different listing.
	keep := *l
	if err := json.Unmarshal(normalizeDateFields(c.Body(), "available_from"), l); err != nil {
		return badRequest(c, err)
	}
	l.ID, l.CompanyID, l.Status, l.PublishedAt = keep.ID, keep.CompanyID, keep.Status, keep.PublishedAt
	l.PortalSyncStatus, l.CreatedBy, l.CreatedAt = keep.PortalSyncStatus, keep.CreatedBy, keep.CreatedAt
	// The country and the permit override are not client-writable either (the override has
	// its own admin-only, audited endpoint).
	l.Country = keep.Country
	l.PermitOverrideBy, l.PermitOverrideAt, l.PermitOverrideReason = keep.PermitOverrideBy, keep.PermitOverrideAt, keep.PermitOverrideReason
	l.PermitNumber = strings.TrimSpace(l.PermitNumber)
	if err := validateListing(l); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

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
// @Description  Updates the status of a listing (draft, published, sold, rented, expired, withdrawn). Publishing may require admin approval, in which case the response is `{"status":"pending_approval","approval_id":...}` and the listing is unchanged.
// @Tags         Listings
// @Produce      json
// @Param        id    path  string  true  "Listing UUID"
// @Param        body  body  object{status=string}  true  "New status"
// @Success      200  {object}  object{status=string}
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
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
		Status domain.ListingStatus `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil || body.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status is required"})
	}
	if !validListingStatuses[body.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be one of: draft, published, sold, rented, expired, withdrawn"})
	}

	current, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
	}
	if current.Status == body.Status {
		return c.JSON(fiber.Map{"status": body.Status})
	}

	if body.Status == domain.ListingStatusPublished {
		// The advertising permit must be valid (or overridden), then approval may be required.
		if blocked, err := permitBlocked(c, current); blocked {
			return err
		}
		approvalID, err := h.publishGate(c, companyID, userID, id)
		if err != nil {
			return serverError(c, err)
		}
		if approvalID != nil {
			return c.JSON(fiber.Map{"status": "pending_approval", "approval_id": approvalID})
		}
		return c.JSON(fiber.Map{"status": body.Status})
	}

	if err := h.repo.UpdateStatus(c.Context(), id, body.Status); err != nil {
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
// @Description  Deletes a listing by UUID. Offers on the listing are deleted with it, so when any exist the request is refused with 409 and the count unless `?force=true`.
// @Tags         Listings
// @Param        id     path   string  true   "Listing UUID"
// @Param        force  query  bool    false  "Also delete the listing's offers"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Failure      409  {object}  object{error=string,offers=int}
// @Security     BearerAuth
// @Router       /listings/{id} [delete]
func (h *ListingHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if _, err := h.repo.GetByID(c.Context(), id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
	}
	offers, err := h.repo.OfferCount(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if offers > 0 && !c.QueryBool("force", false) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":  "this listing has offers that would be deleted too; confirm with force=true",
			"offers": offers,
		})
	}
	if err := h.repo.Delete(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
