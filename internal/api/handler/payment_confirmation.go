package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/ai"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type PaymentConfirmationHandler struct {
	confirmations *repo.PaymentConfirmationRepo
	confirmationService *ai.PaymentConfirmationService
}

func NewPaymentConfirmationHandler(
	confirmations *repo.PaymentConfirmationRepo,
	confirmationService *ai.PaymentConfirmationService,
) *PaymentConfirmationHandler {
	return &PaymentConfirmationHandler{
		confirmations:       confirmations,
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

	if err := h.confirmationService.CreateAndSendConfirmation(c.Context(), paymentID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	confirmation, _ := h.confirmations.GetByPaymentID(c.Context(), paymentID)
	return c.Status(fiber.StatusCreated).JSON(confirmation)
}
