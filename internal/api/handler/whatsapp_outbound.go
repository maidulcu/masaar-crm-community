package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/whatsapp"
)

type WhatsAppOutboundHandler struct {
	sender       *whatsapp.Sender
	outboundRepo *repo.WhatsAppOutboundRepo
	threadRepo   *repo.WhatsAppRepo
}

func NewWhatsAppOutboundHandler(sender *whatsapp.Sender, outboundRepo *repo.WhatsAppOutboundRepo, threadRepo *repo.WhatsAppRepo) *WhatsAppOutboundHandler {
	return &WhatsAppOutboundHandler{
		sender:       sender,
		outboundRepo: outboundRepo,
		threadRepo:   threadRepo,
	}
}

// SendMessage sends a WhatsApp text message to a contact
// @Summary Send WhatsApp message
// @Description Send a text message via WhatsApp to a contact
// @Tags WhatsApp
// @Accept json
// @Produce json
// @Param request body SendMessageRequest true "Message details"
// @Success 201 {object} domain.WhatsAppOutbound
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/threads/{id}/send-message [post]
// @Security Bearer
func (h *WhatsAppOutboundHandler) SendMessage(c *fiber.Ctx) error {
	if h.sender == nil || !h.sender.IsConfigured() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WhatsApp integration not configured",
		})
	}

	threadIDStr := c.Params("id")
	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
	}

	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message is required"})
	}

	// Get thread to verify it exists
	thread, err := h.threadRepo.GetThread(c.Context(), threadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
	}

	// Create outbound record
	userID := c.Locals("user_id").(uuid.UUID)
	outbound := &domain.WhatsAppOutbound{
		ThreadID:      threadID,
		ToNumber:      thread.Contact.PhoneWA,
		MessageBody:   req.Message,
		Status:        domain.OutboundPending,
		CreatedBy:     &userID,
		Metadata:      req.Metadata,
	}

	if err := h.outboundRepo.Create(c.Context(), outbound); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save message",
		})
	}

	// Send via WhatsApp API
	waMessageID, err := h.sender.SendMessage(c.Context(), outbound.ToNumber, outbound.MessageBody)
	if err != nil {
		// Update status to failed
		h.outboundRepo.UpdateStatus(c.Context(), outbound.ID, domain.OutboundFailed, "", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to send message: " + err.Error(),
		})
	}

	// Update status to sent
	h.outboundRepo.UpdateStatus(c.Context(), outbound.ID, domain.OutboundSent, waMessageID, "")
	outbound.WAMessageID = waMessageID
	outbound.Status = domain.OutboundSent

	return c.Status(fiber.StatusCreated).JSON(outbound)
}

// SendTemplate sends a WhatsApp template message
// @Summary Send WhatsApp template
// @Description Send a pre-approved template message with parameters
// @Tags WhatsApp
// @Accept json
// @Produce json
// @Param id path string true "Thread ID"
// @Param request body SendTemplateRequest true "Template details"
// @Success 201 {object} domain.WhatsAppOutbound
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/threads/{id}/send-template [post]
// @Security Bearer
func (h *WhatsAppOutboundHandler) SendTemplate(c *fiber.Ctx) error {
	if h.sender == nil || !h.sender.IsConfigured() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WhatsApp integration not configured",
		})
	}

	threadIDStr := c.Params("id")
	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
	}

	var req SendTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.TemplateName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "template_name is required"})
	}

	// Get thread
	thread, err := h.threadRepo.GetThread(c.Context(), threadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
	}

	// Create outbound record
	userID := c.Locals("user_id").(uuid.UUID)
	outbound := &domain.WhatsAppOutbound{
		ThreadID:    threadID,
		ToNumber:    thread.Contact.PhoneWA,
		MessageBody: req.TemplateName, // Template name in body for reference
		Status:      domain.OutboundPending,
		CreatedBy:   &userID,
		Metadata: map[string]any{
			"template_name": req.TemplateName,
			"parameters":    req.Parameters,
		},
	}

	if err := h.outboundRepo.Create(c.Context(), outbound); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save message",
		})
	}

	// Send template
	language := "en"
	if req.LanguageCode != "" {
		language = req.LanguageCode
	}

	waMessageID, err := h.sender.SendTemplate(c.Context(), outbound.ToNumber, req.TemplateName, language, req.Parameters)
	if err != nil {
		h.outboundRepo.UpdateStatus(c.Context(), outbound.ID, domain.OutboundFailed, "", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to send template: " + err.Error(),
		})
	}

	// Update status
	h.outboundRepo.UpdateStatus(c.Context(), outbound.ID, domain.OutboundSent, waMessageID, "")
	outbound.WAMessageID = waMessageID
	outbound.Status = domain.OutboundSent

	return c.Status(fiber.StatusCreated).JSON(outbound)
}

// GetOutboundMessages retrieves sent messages for a thread
// @Summary Get sent messages
// @Description List all outbound messages sent to a contact
// @Tags WhatsApp
// @Produce json
// @Param id path string true "Thread ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /api/v1/threads/{id}/outbound-messages [get]
// @Security Bearer
func (h *WhatsAppOutboundHandler) GetOutboundMessages(c *fiber.Ctx) error {
	threadIDStr := c.Params("id")
	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
	}

	messages, err := h.outboundRepo.ListByThread(c.Context(), threadID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch messages",
		})
	}

	if messages == nil {
		messages = []domain.WhatsAppOutbound{}
	}

	return c.JSON(fiber.Map{
		"messages": messages,
	})
}

// SendMedia sends a media message (image, document, audio, video)
// @Summary Send WhatsApp media
// @Description Send a media message via WhatsApp
// @Tags WhatsApp
// @Accept json
// @Produce json
// @Param id path string true "Thread ID"
// @Param request body SendMediaRequest true "Media details"
// @Success 201 {object} domain.WhatsAppOutbound
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/threads/{id}/send-media [post]
// @Security Bearer
func (h *WhatsAppOutboundHandler) SendMedia(c *fiber.Ctx) error {
	if !h.sender.IsConfigured() {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "WhatsApp integration not configured",
		})
	}

	threadIDStr := c.Params("id")
	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread id"})
	}

	var req SendMediaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.MediaURL == "" || req.MediaType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "media_url and media_type are required"})
	}

	thread, err := h.threadRepo.GetThread(c.Context(), threadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	outbound := &domain.WhatsAppOutbound{
		ThreadID:    threadID,
		ToNumber:    thread.Contact.PhoneWA,
		MessageBody: req.Caption,
		MediaURL:    req.MediaURL,
		Status:      domain.OutboundPending,
		CreatedBy:   &userID,
		Metadata: map[string]any{
			"media_type": req.MediaType,
		},
	}

	if err := h.outboundRepo.Create(c.Context(), outbound); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save message",
		})
	}

	waMessageID, err := h.sender.SendMedia(c.Context(), outbound.ToNumber, req.MediaType, req.MediaURL)
	if err != nil {
		h.outboundRepo.UpdateStatus(c.Context(), outbound.ID, domain.OutboundFailed, "", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to send media: " + err.Error(),
		})
	}

	h.outboundRepo.UpdateStatus(c.Context(), outbound.ID, domain.OutboundSent, waMessageID, "")
	outbound.WAMessageID = waMessageID
	outbound.Status = domain.OutboundSent

	return c.Status(fiber.StatusCreated).JSON(outbound)
}

// Request types
type SendMessageRequest struct {
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata"`
}

type SendTemplateRequest struct {
	TemplateName string   `json:"template_name"`
	Parameters   []string `json:"parameters"`
	LanguageCode string   `json:"language_code"`
}

type SendMediaRequest struct {
	MediaURL  string `json:"media_url"`
	MediaType string `json:"media_type"`
	Caption   string `json:"caption"`
}
