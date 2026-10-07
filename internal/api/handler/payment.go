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
// @Param        lease_id query   string  false  "Only payments of this lease"
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

	// ?lease_id lets the lease page load exactly its own payments (it used to fetch the company's
	// first 100 payments and filter in the browser, so older leases showed none of theirs).
	var leaseID *uuid.UUID
	if s := c.Query("lease_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid lease_id"})
		}
		leaseID = &id
	}

	result, err := h.payments.List(c.Context(), companyID, leaseID, page, limit)
	if err != nil {
		return serverError(c, err)
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
	var in domain.Payment
	if err := json.Unmarshal(normalizeDateFields(c.Body(), paymentDateKeys...), &in); err != nil {
		return badRequest(c, err)
	}
	// Reconciliation links are set by the bank-reconciliation flow, never by the client (a
	// bank_transaction_id from another company would otherwise be accepted as long as it exists).
	p := in
	p.ID, p.CreatedAt, p.UpdatedAt = uuid.Nil, time.Time{}, time.Time{}
	p.BankTransactionID, p.ReconciledAt, p.ReconciledBy = nil, nil, nil

	if p.LeaseID == uuid.Nil || p.PaymentMethod == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "lease_id, amount > 0, and payment_method are required"})
	}
	p.DueDate = dateOnly(p.DueDate)
	if err := validatePayment(&p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
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
		return serverError(c, err)
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

	// Identity, the owning lease/currency and the reconciliation links are not editable (the
	// UPDATE ignored some of them, but the response echoed them back as if saved).
	keep := *p
	if err := json.Unmarshal(normalizeDateFields(c.Body(), paymentDateKeys...), p); err != nil {
		return badRequest(c, err)
	}
	p.ID, p.CompanyID, p.LeaseID, p.Currency, p.DueDate = keep.ID, keep.CompanyID, keep.LeaseID, keep.Currency, keep.DueDate
	p.CreatedAt, p.CreatedBy = keep.CreatedAt, keep.CreatedBy
	p.BankTransactionID, p.ReconciledAt, p.ReconciledBy = keep.BankTransactionID, keep.ReconciledAt, keep.ReconciledBy
	if err := validatePayment(p); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	p.UpdatedBy = &userID

	if err := h.payments.Update(c.Context(), p); err != nil {
		return serverError(c, err)
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
		if isForeignKeyViolation(err) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this payment is referenced by a confirmation and cannot be deleted"})
		}
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
