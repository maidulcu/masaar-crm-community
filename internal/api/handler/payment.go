package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type PaymentHandler struct {
	payments *repo.PaymentRepo
}

func NewPaymentHandler(payments *repo.PaymentRepo) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

// List godoc
// @Summary      List payments
// @Description  Returns a paginated list of payments for the company.
// @Tags         Payments
// @Produce      json
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Page size 1-100 (default 20)"
// @Success      200    {object}  domain.PaginatedResult[domain.Payment]
// @Security     BearerAuth
// @Router       /payments [get]
func (h *PaymentHandler) List(c *fiber.Ctx) error {
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

	result, err := h.payments.List(c.Context(), companyID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get payment
// @Description  Returns a single payment by UUID.
// @Tags         Payments
// @Produce      json
// @Param        id  path      string  true  "Payment UUID"
// @Success      200  {object}  domain.Payment
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /payments/{id} [get]
func (h *PaymentHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	payment, err := h.payments.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "payment not found"})
	}
	return c.JSON(payment)
}

// Create godoc
// @Summary      Create payment
// @Description  Creates a new payment record. lease_id, amount, due_date, and payment_method are required.
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        body  body      domain.Payment  true  "Payment payload"
// @Success      201   {object}  domain.Payment
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /payments [post]
func (h *PaymentHandler) Create(c *fiber.Ctx) error {
	var p domain.Payment
	if err := c.BodyParser(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if p.LeaseID == uuid.Nil || p.Amount <= 0 || p.PaymentMethod == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "lease_id, amount > 0, and payment_method are required"})
	}
	const maxPaymentAED = 10_000_000
	if p.Amount > maxPaymentAED {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount exceeds maximum allowed (10,000,000 AED)"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	p.CreatedBy = &userID
	p.UpdatedBy = &userID

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	p.CompanyID = companyID

	if err := h.payments.Create(c.Context(), &p); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

// Update godoc
// @Summary      Update payment
// @Description  Updates an existing payment.
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        id    path      string         true  "Payment UUID"
// @Param        body  body      domain.Payment  true  "Payment payload"
// @Success      200   {object}  domain.Payment
// @Failure      400   {object}  object{error=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /payments/{id} [patch]
func (h *PaymentHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	p, err := h.payments.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "payment not found"})
	}

	if err := c.BodyParser(p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	p.UpdatedBy = &userID

	if err := h.payments.Update(c.Context(), p); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(p)
}

// Delete godoc
// @Summary      Delete payment
// @Description  Deletes a payment by UUID.
// @Tags         Payments
// @Param        id  path  string  true  "Payment UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /payments/{id} [delete]
func (h *PaymentHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.payments.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
