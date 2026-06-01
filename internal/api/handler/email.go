package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/email"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type EmailHandler struct {
	emailService *email.Service
	emailRepo    *repo.EmailRepository
}

func NewEmailHandler(emailService *email.Service, emailRepo *repo.EmailRepository) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
		emailRepo:    emailRepo,
	}
}

// SendEmail sends an email and logs it
// @Summary Send email
// @Description Send an email to a recipient
// @Tags Email
// @Accept json
// @Produce json
// @Param request body SendEmailRequest true "Email details"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/emails/send [post]
// @Security Bearer
func (h *EmailHandler) SendEmail(c *fiber.Ctx) error {
	if !h.emailService.IsConfigured() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "email service not configured",
		})
	}

	var req SendEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	// Validate
	if req.ToEmail == "" || req.Subject == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "to_email and subject required"})
	}

	// Create email history record
	userID := c.Locals("user_id").(uuid.UUID)
	emailHist := &domain.EmailHistory{
		FromEmail: req.FromEmail,
		ToEmail:   req.ToEmail,
		Subject:   req.Subject,
		Body:      req.Body,
		HTMLBody:  req.HTMLBody,
		Status:    domain.EmailPending,
		RelatedTo: req.RelatedTo,
		CreatedBy: &userID,
		Metadata:  req.Metadata,
	}

	// Parse related ID if provided
	if req.RelatedID > 0 {
		emailHist.RelatedID = &req.RelatedID
	}

	// Save to database first
	err := h.emailRepo.Create(c.Context(), emailHist)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save email: " + err.Error(),
		})
	}

	// Attempt to send
	err = h.emailService.Send(emailHist)
	if err != nil {
		// Update status to failed
		h.emailRepo.UpdateStatus(c.Context(), emailHist.ID, domain.EmailFailed, err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to send email: " + err.Error(),
		})
	}

	// Update status to sent
	h.emailRepo.UpdateStatus(c.Context(), emailHist.ID, domain.EmailSent, "")

	return c.JSON(fiber.Map{
		"success": true,
		"email_id": emailHist.ID,
	})
}

// GetEmailHistory retrieves email history for a related entity
// @Summary Get email history
// @Description Get all emails sent for an invoice, proposal, etc
// @Tags Email
// @Produce json
// @Param related_to query string true "Entity type (invoice, proposal, followup)"
// @Param related_id query integer true "Entity ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/v1/emails/history [get]
// @Security Bearer
func (h *EmailHandler) GetEmailHistory(c *fiber.Ctx) error {
	relatedTo := c.Query("related_to")
	relatedIDStr := c.Query("related_id")

	if relatedTo == "" || relatedIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "related_to and related_id required",
		})
	}

	relatedID, err := strconv.Atoi(relatedIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid related_id",
		})
	}

	emails, err := h.emailRepo.ListByRelated(c.Context(), relatedTo, int64(relatedID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch email history: " + err.Error(),
		})
	}

	if emails == nil {
		emails = []domain.EmailHistory{}
	}

	return c.JSON(fiber.Map{
		"emails": emails,
	})
}

// ListAllEmailHistory returns paginated email history across all entities
// @Summary      List all email history
// @Tags         Email
// @Produce      json
// @Param        page  query  int  false  "Page (default 1)"
// @Param        limit query  int  false  "Limit (default 50)"
// @Success      200  {object}  object{emails=[]domain.EmailHistory,total=int}
// @Security     BearerAuth
// @Router       /emails [get]
func (h *EmailHandler) ListAllEmailHistory(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	emails, total, err := h.emailRepo.ListAll(c.Context(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if emails == nil {
		emails = []domain.EmailHistory{}
	}
	return c.JSON(fiber.Map{"emails": emails, "total": total})
}

// Request types
type SendEmailRequest struct {
	FromEmail   string                 `json:"from_email"`
	ToEmail     string                 `json:"to_email"`
	Subject     string                 `json:"subject"`
	Body        string                 `json:"body"`
	HTMLBody    string                 `json:"html_body"`
	RelatedTo   string                 `json:"related_to"`
	RelatedID   int64                  `json:"related_id"`
	Metadata    map[string]interface{} `json:"metadata"`
}
