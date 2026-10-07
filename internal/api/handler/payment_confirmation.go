package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/ai"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type PaymentConfirmationHandler struct {
	confirmations       *repo.PaymentConfirmationRepo
	payments            *repo.PaymentRepo
	confirmationService *ai.PaymentConfirmationService
}

func NewPaymentConfirmationHandler(
	confirmations *repo.PaymentConfirmationRepo,
	payments *repo.PaymentRepo,
	confirmationService *ai.PaymentConfirmationService,
) *PaymentConfirmationHandler {
	return &PaymentConfirmationHandler{
		confirmations:       confirmations,
		payments:            payments,
		confirmationService: confirmationService,
	}
}

// GetByPayment godoc
// @Summary      Get payment confirmation
// @Description  Returns the confirmation for a specific payment.
// @Tags         Payment Confirmations
// @Produce      json
// @Param        payment_id  path      string  true  "Payment UUID"
// @Success      200  {object}  domain.PaymentConfirmation
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /payments/{payment_id}/confirmation [get]
func (h *PaymentConfirmationHandler) GetByPayment(c *fiber.Ctx) error {
	paymentID, err := uuid.Parse(c.Params("payment_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payment_id"})
	}

	confirmation, err := h.confirmations.GetByPaymentID(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "confirmation not found"})
	}
	return c.JSON(confirmation)
}

// Send godoc
// @Summary      Send payment confirmation
// @Description  Generates and sends a payment confirmation email to the tenant.
// @Tags         Payment Confirmations
// @Produce      json
// @Param        payment_id  path      string  true  "Payment UUID"
// @Success      200  {object}  domain.PaymentConfirmation
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /payments/{payment_id}/send-confirmation [post]
func (h *PaymentConfirmationHandler) Send(c *fiber.Ctx) error {
	paymentID, err := uuid.Parse(c.Params("payment_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payment_id"})
	}

	// The payment must exist in this company and actually have been received.
	payment, err := h.payments.GetByID(c.Context(), paymentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "payment not found"})
	}
	if payment.Status != domain.PaymentReceived {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "a confirmation can only be sent for a received payment"})
	}

	if err := h.confirmationService.CreateAndSendConfirmation(c.Context(), paymentID); err != nil {
		return serverError(c, err)
	}

	// This used to answer 201 with `null` when the service produced nothing (the community edition's
	// service is a no-op), which the UI showed as a sent confirmation.
	confirmation, err := h.confirmations.GetByPaymentID(c.Context(), paymentID)
	if err != nil || confirmation == nil {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{"error": "payment confirmations are not available in this edition"})
	}
	return c.Status(fiber.StatusCreated).JSON(confirmation)
}
