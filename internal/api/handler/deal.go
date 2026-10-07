package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		id, err := uuid.Parse(s)
		if err != nil {
			// Ignoring a bad id silently returned every deal, as if no filter had been asked for.
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid owner_id"})
		}
		ownerID = &id
	}
	if stage != "" && stage != string(domain.DealStageOpen) && stage != string(domain.DealStageWon) && stage != string(domain.DealStageLost) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "stage must be open, won or lost"})
	}

	result, err := h.deals.List(c.Context(), ownerID, stage, page, limit)
	if err != nil {
		return serverError(c, err)
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
		return serverError(c, err)
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
	var in struct {
		LeadID      uuid.UUID        `json:"lead_id"`
		Title       string           `json:"title"`
		Stage       domain.DealStage `json:"stage"`
		Amount      float64          `json:"amount"`
		Currency    string           `json:"currency"`
		CloseDate   string           `json:"close_date"`
		Probability *int             `json:"probability"` // pointer: 0% is a legitimate value
	}
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if in.LeadID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "lead_id is required"})
	}
	title, err := validateDealTitle(in.Title)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	amount, err := validateDealAmount(in.Amount)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	currency, ok := normalizeCurrency(in.Currency)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "currency must be a 3-letter code such as AED"})
	}
	closeDate, err := parseCloseDate(in.CloseDate)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	stage := in.Stage
	if stage == "" {
		stage = domain.DealStageOpen
	}
	probability := 50
	if in.Probability != nil {
		if *in.Probability < 0 || *in.Probability > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "probability must be 0-100"})
		}
		probability = *in.Probability
	}
	switch stage {
	case domain.DealStageOpen:
	case domain.DealStageWon:
		probability = 100
	case domain.DealStageLost:
		probability = 0
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "stage must be open, won or lost"})
	}

	deal := domain.Deal{
		LeadID: in.LeadID, Title: title, Stage: stage, Amount: amount, Currency: currency,
		CloseDate: closeDate, Probability: probability,
		OwnerID: c.Locals("user_id").(uuid.UUID),
	}
	if err := h.deals.Create(c.Context(), &deal); err != nil {
		return serverError(c, err)
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
		return serverError(c, err)
	}

	if updates.Title != nil {
		title, err := validateDealTitle(*updates.Title)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		deal.Title = title
	}
	if updates.Amount != nil {
		amount, err := validateDealAmount(*updates.Amount)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		deal.Amount = amount
	}
	if updates.Currency != nil {
		currency, ok := normalizeCurrency(*updates.Currency)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "currency must be a 3-letter code such as AED"})
		}
		deal.Currency = currency
	}
	if updates.Probability != nil {
		if *updates.Probability < 0 || *updates.Probability > 100 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "probability must be 0-100"})
		}
		deal.Probability = *updates.Probability
	}
	if updates.CloseDate != nil {
		// "" clears the date; a full timestamp (what GET returns) is accepted as well as YYYY-MM-DD.
		t, err := parseCloseDate(*updates.CloseDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		deal.CloseDate = t
	}

	if err := h.deals.Update(c.Context(), deal); err != nil {
		return serverError(c, err)
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
		return serverError(c, err)
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
	if _, err := h.deals.GetByID(c.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "deal not found"})
		}
		return serverError(c, err)
	}
	invoices, err := h.invoices.ListByDeal(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if invoices == nil {
		invoices = []domain.VATInvoice{}
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			// vat_invoices.deal_id is ON DELETE RESTRICT: invoices are accounting records.
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this deal has invoices and cannot be deleted"})
		}
		return serverError(c, err)
	}
	h.audit.Log(c.Context(), c.Locals("user_id").(uuid.UUID), repo.AuditDelete, repo.AuditDeal, id, nil)
	return c.SendStatus(fiber.StatusNoContent)
}
