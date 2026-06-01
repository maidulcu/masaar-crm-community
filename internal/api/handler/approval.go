package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type ApprovalHandler struct {
	repo *repo.ApprovalRepo
}

func NewApprovalHandler(repo *repo.ApprovalRepo) *ApprovalHandler {
	return &ApprovalHandler{repo: repo}
}

// GetConfig godoc
// @Summary      Get approval config
// @Tags         Approvals
// @Produce      json
// @Success      200  {object}  domain.ApprovalConfig
// @Security     BearerAuth
// @Router       /approval-config [get]
func (h *ApprovalHandler) GetConfig(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	cfg, err := h.repo.GetConfig(c.Context(), companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cfg)
}

// SaveConfig godoc
// @Summary      Save approval config
// @Tags         Approvals
// @Accept       json
// @Produce      json
// @Param        body  body  domain.ApprovalConfig  true  "Config"
// @Success      200  {object}  domain.ApprovalConfig
// @Security     BearerAuth
// @Router       /approval-config [patch]
func (h *ApprovalHandler) SaveConfig(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	var body struct {
		ListingApproval    *bool    `json:"listing_approval"`
		DealApprovalAbove  *float64 `json:"deal_approval_above"`
		OfferApprovalAbove *float64 `json:"offer_approval_above"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	cfg, err := h.repo.GetConfig(c.Context(), companyID)
	if err != nil || cfg == nil {
		cfg = &domain.ApprovalConfig{CompanyID: companyID}
	}
	if body.ListingApproval != nil {
		cfg.ListingApproval = *body.ListingApproval
	}
	if body.DealApprovalAbove != nil {
		cfg.DealApprovalAbove = *body.DealApprovalAbove
	}
	if body.OfferApprovalAbove != nil {
		cfg.OfferApprovalAbove = *body.OfferApprovalAbove
	}

	if err := h.repo.SaveConfig(c.Context(), cfg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	cfg, _ = h.repo.GetConfig(c.Context(), companyID)
	return c.JSON(cfg)
}

// ListRequests godoc
// @Summary      List approval requests
// @Tags         Approvals
// @Produce      json
// @Param        status       query  string  false  "Filter by status"
// @Param        entity_type  query  string  false  "Filter by entity (listing, deal, offer)"
// @Param        page         query  int     false  "Page number"
// @Param        limit        query  int     false  "Items per page"
// @Success      200  {object}  repo.PaginatedResult{data=[]domain.ApprovalRequest}
// @Security     BearerAuth
// @Router       /approval-requests [get]
func (h *ApprovalHandler) ListRequests(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	status := c.Query("status")
	entityType := c.Query("entity_type")

	list, total, err := h.repo.List(c.Context(), companyID, status, entityType, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": list, "total": total, "page": page, "limit": limit})
}

// ReviewRequest godoc
// @Summary      Approve or reject a request
// @Tags         Approvals
// @Accept       json
// @Produce      json
// @Param        id    path  string  true  "Request UUID"
// @Param        body  body  object{status=string,note=string}  true  "approved or rejected"
// @Success      200  {object}  domain.ApprovalRequest
// @Security     BearerAuth
// @Router       /approval-requests/{id}/review [post]
func (h *ApprovalHandler) ReviewRequest(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	userID, err := uuid.Parse(c.Locals("user_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
	}

	var body struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if body.Status != string(domain.ApprovalApproved) && body.Status != string(domain.ApprovalRejected) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be 'approved' or 'rejected'"})
	}

	if err := h.repo.Review(c.Context(), id, userID, domain.ApprovalStatus(body.Status), body.Note); err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	req, _ := h.repo.GetByID(c.Context(), id)
	if req == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "request not found"})
	}

	return c.JSON(req)
}
