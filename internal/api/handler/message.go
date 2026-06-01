package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/ai"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

type MessageHandler struct {
	aiClient      *ai.Client
	waRepo        *repo.WhatsAppRepo
	contactRepo   *repo.ContactRepo
	leadRepo      *repo.LeadRepo
	commHistRepo  *repo.CommunicationHistoryRepo
	tagRepo       *repo.LeadTagRepo
	scoringService *ai.ScoringService
	hub           *ws.Hub
}

func NewMessageHandler(aiClient *ai.Client, waRepo *repo.WhatsAppRepo, contactRepo *repo.ContactRepo, leadRepo *repo.LeadRepo, commHistRepo *repo.CommunicationHistoryRepo, tagRepo *repo.LeadTagRepo, scoringService *ai.ScoringService, hub *ws.Hub) *MessageHandler {
	return &MessageHandler{
		aiClient:      aiClient,
		waRepo:        waRepo,
		contactRepo:   contactRepo,
		leadRepo:      leadRepo,
		commHistRepo:  commHistRepo,
		tagRepo:       tagRepo,
		scoringService: scoringService,
		hub:           hub,
	}
}

// AnalyzeMessage extracts intent and enrichment from a WhatsApp message
// @Summary Analyze message for intent and lead data
// @Description Parse message to extract customer intent, property interests, budget, timeline
// @Tags Message Analysis
// @Accept json
// @Produce json
// @Param request body AnalyzeMessageRequest true "Message to analyze"
// @Success 200 {object} MessageAnalysisResponse
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/messages/analyze [post]
// @Security Bearer
func (h *MessageHandler) AnalyzeMessage(c *fiber.Ctx) error {
	if h.aiClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "AI service not available",
		})
	}

	var req AnalyzeMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message is required"})
	}

	// Parse intent
	intentJSON, err := h.aiClient.ParseIntent(c.Context(), req.Message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to parse intent: " + err.Error(),
		})
	}

	var intent map[string]interface{}
	if err := json.Unmarshal([]byte(intentJSON), &intent); err != nil {
		intent = map[string]interface{}{"error": "failed to parse intent JSON"}
	}

	// Parse enrichment if conversation provided
	var enrichment map[string]interface{}
	if req.Conversation != "" {
		enrichJSON, err := h.aiClient.EnrichLead(c.Context(), req.ContactName, req.Conversation)
		if err == nil {
			json.Unmarshal([]byte(enrichJSON), &enrichment)
		}
	}

	response := MessageAnalysisResponse{
		Message:    req.Message,
		Intent:     intent,
		Enrichment: enrichment,
	}

	return c.JSON(response)
}

// SuggestNextAction recommends next step based on message
// @Summary Get suggested next action
// @Description AI recommends next action for agent based on message and conversation
// @Tags Message Analysis
// @Accept json
// @Produce json
// @Param request body SuggestActionRequest true "Message and context"
// @Success 200 {object} SuggestActionResponse
// @Failure 400 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/messages/suggest-action [post]
// @Security Bearer
func (h *MessageHandler) SuggestNextAction(c *fiber.Ctx) error {
	if h.aiClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "AI service not available",
		})
	}

	var req SuggestActionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message is required"})
	}

	actionJSON, err := h.aiClient.SuggestAction(c.Context(), req.Message, req.ThreadSummary)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to suggest action: " + err.Error(),
		})
	}

	var action map[string]interface{}
	if err := json.Unmarshal([]byte(actionJSON), &action); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to parse action response",
		})
	}

	return c.JSON(action)
}

// AutoCreateLead creates a lead from WhatsApp message analysis
// @Summary Auto-create lead from message
// @Description Analyze message and automatically create lead if it contains sufficient intent
// @Tags Message Analysis
// @Accept json
// @Produce json
// @Param request body AutoCreateLeadRequest true "Thread and contact info"
// @Success 201 {object} domain.Lead
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/v1/messages/auto-create-lead [post]
// @Security Bearer
func (h *MessageHandler) AutoCreateLead(c *fiber.Ctx) error {
	if h.aiClient == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "AI service not available",
		})
	}

	var req AutoCreateLeadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.ThreadID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "thread_id required"})
	}

	// Get thread and messages
	threadID, err := uuid.Parse(req.ThreadID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid thread_id"})
	}

	thread, err := h.waRepo.GetThread(c.Context(), threadID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "thread not found"})
	}

	messages, err := h.waRepo.GetMessages(c.Context(), threadID, 100)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch messages",
		})
	}

	// Build conversation
	conversation := ""
	for _, msg := range messages {
		conversation += msg.Body + "\n"
	}

	// Get contact
	contact, err := h.contactRepo.GetByID(c.Context(), thread.ContactID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "contact not found"})
	}

	// Enrich lead data
	enrichJSON, err := h.aiClient.EnrichLead(c.Context(), contact.FullName, conversation)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to enrich lead data",
		})
	}

	var enrichment map[string]interface{}
	json.Unmarshal([]byte(enrichJSON), &enrichment)

	// Create lead
	lead := &domain.Lead{
		ID:        uuid.New(),
		ContactID: thread.ContactID,
		Stage:     domain.StageNew,
		Source:    domain.SourceWhatsApp,
		Currency:  "AED",
	}

	// Extract deal value if available
	if budget, ok := enrichment["budget"].(map[string]interface{}); ok {
		if maxBudget, ok := budget["max_aed"].(float64); ok && maxBudget > 0 {
			lead.DealValue = maxBudget
		}
	}

	// Build notes from enrichment
	notesMap := map[string]interface{}{
		"enriched_from": "whatsapp_message",
		"enrichment": enrichment,
	}
	notesJSON, _ := json.Marshal(notesMap)
	lead.Notes = string(notesJSON)

	if err := h.leadRepo.Create(c.Context(), lead); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create lead",
		})
	}

	// Calculate initial lead score
	if h.scoringService != nil {
		if score, err := h.scoringService.CalculateScore(c.Context(), lead.ID); err == nil {
			h.leadRepo.UpdateScore(c.Context(), lead.ID, score)
			lead.LeadScore = score
		}
	}

	// Broadcast to WebSocket
	userID := c.Locals("user_id").(uuid.UUID)
	h.hub.SendToUser(userID.String(), ws.Event{
		Type: "lead_created",
		Payload: map[string]interface{}{
			"lead_id": lead.ID,
			"contact": contact.FullName,
			"source":  "auto_enrichment",
		},
	})

	return c.Status(fiber.StatusCreated).JSON(lead)
}

// Request/Response types

type AnalyzeMessageRequest struct {
	Message      string `json:"message"`
	ContactName  string `json:"contact_name"`
	Conversation string `json:"conversation"`
}

type MessageAnalysisResponse struct {
	Message    string                 `json:"message"`
	Intent     map[string]interface{} `json:"intent"`
	Enrichment map[string]interface{} `json:"enrichment,omitempty"`
}

type SuggestActionRequest struct {
	Message       string `json:"message"`
	ThreadSummary string `json:"thread_summary"`
}

type AutoCreateLeadRequest struct {
	ThreadID string `json:"thread_id"`
}
