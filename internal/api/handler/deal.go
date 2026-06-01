package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type DealHandler struct {
	deals    *repo.DealRepo
	invoices *repo.InvoiceRepo
	audit    *repo.AuditLogRepo
}

func NewDealHandler(deals *repo.DealRepo, invoices *repo.InvoiceRepo, audit *repo.AuditLogRepo) *DealHandler {
	return &DealHandler{deals: deals, invoices: invoices, audit: audit}
}

// List godoc
// @Summary      List deals
// @Description  Returns a paginated list of deals. Optional filters: stage, owner_id.
// @Tags         Deals
// @Produce      json
// @Param        page      query     int     false  "Page (default 1)"
// @Param        limit     query     int     false  "Page size (default 20)"
// @Param        stage     query     string  false  "Filter by stage: open|won|lost"
// @Param        owner_id  query     string  false  "Filter by owner UUID"
// @Success      200       {object}  domain.PaginatedResult[domain.Deal]
// @Security     BearerAuth
// @Router       /deals [get]
func (h *DealHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	stage := c.Query("stage", "")

	var ownerID *uuid.UUID
	if s := c.Query("owner_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			ownerID = &id
		}
	}

	result, err := h.deals.List(c.Context(), ownerID, stage, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get deal
// @Description  Returns a single deal by UUID.
// @Tags         Deals
// @Produce      json
// @Param        id  path      string  true  "Deal UUID"
// @Success      200  {object}  domain.Deal
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /deals/{id} [get]
func (h *DealHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	deal, err := h.deals.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "deal not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(deal)
}

// Create godoc
// @Summary      Create deal
// @Description  Creates a deal linked to a lead. Owner is set automatically from the JWT subject.
// @Tags         Deals
// @Accept       json
// @Produce      json
// @Param        body  body      domain.Deal  true  "Deal payload (lead_id and title required)"
// @Success      201   {object}  domain.Deal
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /deals [post]
func (h *DealHandler) Create(c *fiber.Ctx) error {
	var deal domain.Deal
	if err := c.BodyParser(&deal); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if deal.LeadID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "lead_id is required"})
	}
	if deal.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "title is required"})
	}
	if deal.Amount < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount must be non-negative"})
	}
	if deal.Currency == "" {
		deal.Currency = "AED"
	}
	if deal.Stage == "" {
		deal.Stage = domain.DealStageOpen
	}
	if deal.Probability == 0 {
		deal.Probability = 50
	}

	deal.OwnerID = c.Locals("user_id").(uuid.UUID)

	if err := h.deals.Create(c.Context(), &deal); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	h.audit.Log(c.Context(), deal.OwnerID, repo.AuditCreate, repo.AuditDeal, deal.ID, deal)
	return c.Status(fiber.StatusCreated).JSON(deal)
}

// Update godoc
// @Summary      Update deal
// @Description  Updates deal fields: title, amount, currency, probability, close_date.
// @Tags         Deals
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Deal UUID"
// @Param        body  body      object{title=string,amount=number,currency=string,probability=integer,close_date=string}  true  "Update payload"
// @Success      200   {object}  domain.Deal
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /deals/{id} [patch]
func (h *DealHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var updates struct {
		Title       *string  `json:"title"`
		Amount      *float64 `json:"amount"`
		Currency    *string  `json:"currency"`
		Probability *int     `json:"probability"`
		CloseDate   *string  `json:"close_date"`
	}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	deal, err := h.deals.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "deal not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if updates.Title != nil {
		deal.Title = *updates.Title
	}
	if updates.Amount != nil {
		if *updates.Amount < 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount must be non-negative"})
		}
		deal.Amount = *updates.Amount
	}
	if updates.Currency != nil {
		deal.Currency = *updates.Currency
	}
	if updates.Probability != nil {
		if *updates.Probability < 0 || *updates.Probability > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "probability must be 0-100"})
		}
		deal.Probability = *updates.Probability
	}
	if updates.CloseDate != nil {
		t, err := time.Parse("2006-01-02", *updates.CloseDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid close_date format, expected YYYY-MM-DD"})
		}
		deal.CloseDate = &t
	}

	if err := h.deals.Update(c.Context(), deal); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	h.audit.Log(c.Context(), c.Locals("user_id").(uuid.UUID), repo.AuditUpdate, repo.AuditDeal, deal.ID, deal)
	return c.JSON(deal)
}

// UpdateStage godoc
// @Summary      Update deal stage
// @Description  Moves a deal to a new stage: open|won|lost.
// @Tags         Deals
// @Accept       json
// @Produce      json
// @Param        id    path      string                  true  "Deal UUID"
// @Param        body  body      object{stage=string}    true  "New stage"
// @Success      200   {object}  object{id=string,stage=string}
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /deals/{id}/stage [patch]
func (h *DealHandler) UpdateStage(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Stage domain.DealStage `json:"stage"`
	}
	if err := c.BodyParser(&body); err != nil || body.Stage == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "stage required"})
	}

	validStages := map[domain.DealStage]bool{
		domain.DealStageOpen: true,
		domain.DealStageWon:  true,
		domain.DealStageLost: true,
	}
	if !validStages[body.Stage] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid stage"})
	}

	if err := h.deals.UpdateStage(c.Context(), id, body.Stage); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	h.audit.Log(c.Context(), c.Locals("user_id").(uuid.UUID), repo.AuditUpdate, repo.AuditDeal, id, fiber.Map{"stage": body.Stage})
	return c.JSON(fiber.Map{"id": id, "stage": body.Stage})
}

// ListInvoices godoc
// @Summary      List deal invoices
// @Description  Returns all VAT invoices attached to a deal.
// @Tags         Deals
// @Produce      json
// @Param        id  path      string  true  "Deal UUID"
// @Success      200  {array}   domain.VATInvoice
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /deals/{id}/invoices [get]
func (h *DealHandler) ListInvoices(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	invoices, err := h.invoices.ListByDeal(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(invoices)
}

// Delete godoc
// @Summary      Delete deal
// @Description  Permanently deletes a deal. Admin only.
// @Tags         Deals
// @Param        id  path  string  true  "Deal UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /deals/{id} [delete]
func (h *DealHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.deals.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	h.audit.Log(c.Context(), c.Locals("user_id").(uuid.UUID), repo.AuditDelete, repo.AuditDeal, id, nil)
	return c.SendStatus(fiber.StatusNoContent)
}
