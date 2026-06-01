package handler

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/webhook"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

// LeadRepository defines the interface for lead data access.
type LeadRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Lead, error)
	Create(ctx context.Context, lead *domain.Lead) error
	UpdateStage(ctx context.Context, id uuid.UUID, stage domain.LeadStage, reason string) error
	UpdateNotes(ctx context.Context, id uuid.UUID, notes string) error
	Assign(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter repo.LeadFilter) ([]domain.Lead, error)
	KanbanBoard(ctx context.Context) (map[domain.LeadStage][]domain.Lead, error)
}

// ContactRepository defines the interface for contact data access used by the lead handler.
type ContactRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error)
}

// CommunicationHistoryRepository defines the interface for communication history data access.
type CommunicationHistoryRepository interface {
	GetByLead(ctx context.Context, leadID uuid.UUID, limit int) ([]domain.CommunicationHistory, error)
}

// ScoringService defines the interface for AI lead scoring.
type ScoringService interface {
	UpdateScoreOnStageChange(ctx context.Context, leadID uuid.UUID, newStage domain.LeadStage) error
}

// AuditLogRepository defines the interface for audit logging.
type AuditLogRepository interface {
	Log(ctx context.Context, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, diff any)
}

// WebhookDispatcher defines the interface for dispatching webhook events.
type WebhookDispatcher interface {
	Dispatch(companyID uuid.UUID, event string, data interface{})
}

// LeadTagRepository defines the interface for lead tag data access.
type LeadTagRepository interface {
	GetByLead(ctx context.Context, leadID uuid.UUID) ([]string, error)
	AddTag(ctx context.Context, leadID uuid.UUID, tag string, category string, autoApplied bool, userID *uuid.UUID) error
	RemoveTag(ctx context.Context, leadID uuid.UUID, tag string) error
}

type LeadHandler struct {
	leads          LeadRepository
	contacts       ContactRepository
	commHistRepo   CommunicationHistoryRepository
	scoringService ScoringService
	tags           LeadTagRepository
	hub            *ws.Hub
	audit          AuditLogRepository
	dispatcher     WebhookDispatcher
	pipelineStages *repo.PipelineStageRepo
}

func NewLeadHandler(leads LeadRepository, contacts ContactRepository, commHistRepo CommunicationHistoryRepository, scoringService ScoringService, tags LeadTagRepository, hub *ws.Hub, audit AuditLogRepository, dispatcher WebhookDispatcher, pipelineStages *repo.PipelineStageRepo) *LeadHandler {
	return &LeadHandler{leads: leads, contacts: contacts, commHistRepo: commHistRepo, scoringService: scoringService, tags: tags, hub: hub, audit: audit, dispatcher: dispatcher, pipelineStages: pipelineStages}
}

// KanbanBoard godoc
// @Summary      Get Kanban board
// @Description  Returns all active leads grouped by stage for the Kanban pipeline view.
// @Tags         Leads
// @Produce      json
// @Success      200  {object}  object  "Map of stage → []Lead"
// @Security     BearerAuth
// @Router       /leads [get]
func (h *LeadHandler) KanbanBoard(c *fiber.Ctx) error {
	board, err := h.leads.KanbanBoard(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(board)
}

// List godoc
// @Summary      Search / list leads
// @Description  Returns a flat list of leads with optional filtering. Supports search, stage, assigned_to, source, pagination.
// @Tags         Leads
// @Produce      json
// @Param        q           query  string  false  "Search contact name or phone"
// @Param        stage       query  string  false  "Filter by stage: new|contacted|qualified|proposal|won|lost"
// @Param        assigned_to query  string  false  "Filter by agent UUID"
// @Param        source      query  string  false  "Filter by source: web|whatsapp|referral|event"
// @Param        limit       query  int     false  "Page size (default 50)"
// @Param        offset      query  int     false  "Page offset (default 0)"
// @Success      200  {array}   domain.Lead
// @Security     BearerAuth
// @Router       /leads/search [get]
func (h *LeadHandler) List(c *fiber.Ctx) error {
	f := repo.LeadFilter{
		Query:  c.Query("q"),
		Stage:  domain.LeadStage(c.Query("stage")),
		Source: c.Query("source"),
	}
	if s := c.Query("assigned_to"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			f.AssignedTo = &id
		}
	}
	if s := c.Query("contact_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			f.ContactID = &id
		}
	}
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		f.Limit = v
	}
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v >= 0 {
		f.Offset = v
	} else if v, err := strconv.Atoi(c.Query("page")); err == nil && v >= 1 {
		limit := f.Limit
		if limit == 0 {
			limit = 50
		}
		f.Offset = (v - 1) * limit
	}

	leads, err := h.leads.List(c.Context(), f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if leads == nil {
		leads = []domain.Lead{}
	}
	return c.JSON(leads)
}

// Create godoc
// @Summary      Create lead
// @Description  Creates a new lead and broadcasts a lead.created WebSocket event.
// @Tags         Leads
// @Accept       json
// @Produce      json
// @Param        body  body      domain.Lead  true  "Lead payload (contact_id required)"
// @Success      201   {object}  domain.Lead
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leads [post]
func (h *LeadHandler) Create(c *fiber.Ctx) error {
	var lead domain.Lead
	if err := c.BodyParser(&lead); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if lead.ContactID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "contact_id is required"})
	}
	if lead.Stage == "" {
		lead.Stage = domain.StageNew
	}
	if lead.Currency == "" {
		lead.Currency = "AED"
	}

	if err := h.leads.Create(c.Context(), &lead); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	h.hub.Broadcast(ws.Event{
		Type:    "lead.created",
		Payload: lead,
	})

	if h.dispatcher != nil {
		companyID, _ := uuid.Parse(c.Locals("company_id").(string))
		h.dispatcher.Dispatch(companyID, webhook.EventLeadCreated, fiber.Map{
			"lead_id":    lead.ID,
			"contact_id": lead.ContactID,
			"stage":      lead.Stage,
			"source":     lead.Source,
			"deal_value": lead.DealValue,
		})
	}

	actorID := c.Locals("user_id").(uuid.UUID)
	h.audit.Log(c.Context(), actorID, repo.AuditCreate, repo.AuditLead, lead.ID, lead)
	return c.Status(fiber.StatusCreated).JSON(lead)
}

// UpdateStage godoc
// @Summary      Move lead to stage
// @Description  Updates the pipeline stage of a lead (drag-drop). Accepts optional closed_reason for won/lost.
// @Tags         Leads
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Lead UUID"
// @Param        body  body      object{stage=string,closed_reason=string}  true  "New stage"
// @Success      200   {object}  object{lead_id=string,stage=string}
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leads/{id}/stage [patch]
func (h *LeadHandler) UpdateStage(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	var body struct {
		Stage        domain.LeadStage `json:"stage"`
		ClosedReason string           `json:"closed_reason"`
	}
	if err := c.BodyParser(&body); err != nil || body.Stage == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "stage is required"})
	}

	stages, err := h.pipelineStages.ListByCompany(c.Context(), companyID, "lead")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to validate stage"})
	}
	valid := false
	for _, s := range stages {
		if s.Name == string(body.Stage) {
			valid = true
			break
		}
	}
	if !valid {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid stage"})
	}

	if err := h.leads.UpdateStage(c.Context(), id, body.Stage, body.ClosedReason); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if h.scoringService != nil {
		h.scoringService.UpdateScoreOnStageChange(c.Context(), id, body.Stage)
	}

	h.hub.Broadcast(ws.Event{
		Type: "lead.stage_changed",
		Payload: fiber.Map{
			"lead_id": id,
			"stage":   body.Stage,
		},
	})

	if h.dispatcher != nil {
		h.dispatcher.Dispatch(companyID, webhook.EventLeadStageChanged, fiber.Map{
			"lead_id": id,
			"stage":   body.Stage,
		})
		if body.Stage == domain.StageWon {
			h.dispatcher.Dispatch(companyID, webhook.EventLeadWon, fiber.Map{
				"lead_id": id,
				"stage":   body.Stage,
			})
		} else if body.Stage == domain.StageLost {
			h.dispatcher.Dispatch(companyID, webhook.EventLeadLost, fiber.Map{
				"lead_id": id,
				"stage":   body.Stage,
			})
		}
	}

	return c.JSON(fiber.Map{"lead_id": id, "stage": body.Stage})
}

// UpdateNotes godoc
// @Summary      Update lead notes
// @Description  Replaces the notes text on a lead.
// @Tags         Leads
// @Accept       json
// @Produce      json
// @Param        id    path      string               true  "Lead UUID"
// @Param        body  body      object{notes=string} true  "Notes"
// @Success      200   {object}  object{lead_id=string,notes=string}
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leads/{id}/notes [patch]
func (h *LeadHandler) UpdateNotes(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		Notes string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.leads.UpdateNotes(c.Context(), id, body.Notes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"lead_id": id, "notes": body.Notes})
}

// Assign godoc
// @Summary      Assign lead to agent
// @Description  Assigns a lead to a user (agent or admin). Pass null user_id to unassign.
// @Tags         Leads
// @Accept       json
// @Produce      json
// @Param        id    path      string                   true  "Lead UUID"
// @Param        body  body      object{user_id=string}   true  "Agent UUID or null"
// @Success      200   {object}  object{lead_id=string,assigned_to=string}
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leads/{id}/assign [patch]
func (h *LeadHandler) Assign(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		UserID *uuid.UUID `json:"assigned_to"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.leads.Assign(c.Context(), id, body.UserID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"lead_id": id, "assigned_to": body.UserID})
}

// Delete godoc
// @Summary      Delete lead
// @Description  Soft-deletes a lead (hidden from UI, data retained). Admin only.
// @Tags         Leads
// @Param        id  path  string  true  "Lead UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leads/{id} [delete]
func (h *LeadHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.leads.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "lead not found"})
	}

	actorID := c.Locals("user_id").(uuid.UUID)
	h.audit.Log(c.Context(), actorID, repo.AuditDelete, repo.AuditLead, id, nil)
	return c.SendStatus(fiber.StatusNoContent)
}

// Get godoc
// @Summary      Get lead
// @Description  Returns a single lead with its contact joined.
// @Tags         Leads
// @Produce      json
// @Param        id  path      string  true  "Lead UUID"
// @Success      200  {object}  domain.Lead
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leads/{id} [get]
func (h *LeadHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	lead, err := h.leads.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "lead not found"})
	}

	contact, _ := h.contacts.GetByID(c.Context(), lead.ContactID)
	lead.Contact = contact

	tags, _ := h.tags.GetByLead(c.Context(), lead.ID)
	lead.Tags = tags

	return c.JSON(lead)
}

// GetCommunications godoc
// @Summary      Get lead communications
// @Description  Returns all communications (WhatsApp, email, calls) for a lead, ordered by date.
// @Tags         Leads
// @Produce      json
// @Param        id    path      string  true  "Lead UUID"
// @Param        limit query     int     false  "Max communications to return (default 100)"
// @Success      200   {array}   domain.CommunicationHistory
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /leads/{id}/communications [get]
// GetTags godoc
// @Summary      Get lead tags
// @Description  Returns all tags attached to a lead.
// @Tags         Leads
// @Produce      json
// @Param        id  path  string  true  "Lead UUID"
// @Success      200  {array}  string
// @Security     BearerAuth
// @Router       /leads/{id}/tags [get]
func (h *LeadHandler) GetTags(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	tags, err := h.tags.GetByLead(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if tags == nil {
		tags = []string{}
	}
	return c.JSON(tags)
}

// AddTag godoc
// @Summary      Add tag to lead
// @Description  Attaches a tag to a lead. Duplicates are silently ignored.
// @Tags         Leads
// @Accept       json
// @Param        id    path  string  true  "Lead UUID"
// @Param        body  body  object{tag=string,category=string}  true  "Tag payload"
// @Success      201
// @Security     BearerAuth
// @Router       /leads/{id}/tags [post]
func (h *LeadHandler) AddTag(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Tag      string `json:"tag"`
		Category string `json:"category"`
	}
	if err := c.BodyParser(&body); err != nil || body.Tag == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tag is required"})
	}
	if body.Category == "" {
		body.Category = "manual"
	}
	userID := c.Locals("user_id").(uuid.UUID)
	if err := h.tags.AddTag(c.Context(), id, body.Tag, body.Category, false, &userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusCreated)
}

// RemoveTag godoc
// @Summary      Remove tag from lead
// @Description  Detaches a tag from a lead. No error if the tag doesn't exist.
// @Tags         Leads
// @Param        id    path  string  true  "Lead UUID"
// @Param        tag   path  string  true  "Tag value to remove"
// @Success      204
// @Security     BearerAuth
// @Router       /leads/{id}/tags/{tag} [delete]
func (h *LeadHandler) RemoveTag(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	tag := c.Params("tag")
	if tag == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tag is required"})
	}
	if err := h.tags.RemoveTag(c.Context(), id, tag); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *LeadHandler) GetCommunications(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	comms, err := h.commHistRepo.GetByLead(c.Context(), id, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(comms)
}
