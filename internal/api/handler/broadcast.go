package handler

import (
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// BroadcastHandler manages WhatsApp broadcasts, thread assignment and the inbox settings.
type BroadcastHandler struct {
	broadcasts *repo.BroadcastRepo
	audit      *repo.AuditLogRepo
}

func NewBroadcastHandler(b *repo.BroadcastRepo, audit *repo.AuditLogRepo) *BroadcastHandler {
	return &BroadcastHandler{broadcasts: b, audit: audit}
}

// Meta template names: lowercase letters, digits and underscores.
var templateName = regexp.MustCompile(`^[a-z0-9_]{1,512}$`)

type broadcastBody struct {
	Name         string       `json:"name"`
	TemplateName string       `json:"template_name"`
	Language     string       `json:"language"`
	Params       []string     `json:"params"`
	Segment      repo.Segment `json:"segment"`
}

func (b *broadcastBody) validate() string {
	b.Name, b.TemplateName, b.Language = strings.TrimSpace(b.Name), strings.TrimSpace(b.TemplateName), strings.TrimSpace(b.Language)
	if b.Language == "" {
		b.Language = "en"
	}
	switch {
	case b.Name == "" || len(b.Name) > 200:
		return "name is required"
	case !templateName.MatchString(b.TemplateName):
		return "template_name must be the approved template's name (lowercase letters, digits, underscores)"
	case len(b.Language) > 15:
		return "invalid language code"
	case len(b.Params) > 20:
		return "too many parameters"
	}
	for _, p := range b.Params {
		if len(p) > 1024 {
			return "a parameter is too long"
		}
	}
	return ""
}

// Preview godoc
// @Summary      Count a broadcast audience
// @Description  How many contacts a segment reaches, and how many would be skipped for lack of WhatsApp consent or because they opted out.
// @Tags         Broadcasts
// @Accept       json
// @Produce      json
// @Param        body  body  repo.Segment  true  "Segment"
// @Success      200  {object}  repo.AudienceCount
// @Security     BearerAuth
// @Router       /broadcasts/preview [post]
func (h *BroadcastHandler) Preview(c *fiber.Ctx) error {
	var s repo.Segment
	if err := c.BodyParser(&s); err != nil {
		return badRequest(c, err)
	}
	a, err := h.broadcasts.Preview(c.Context(), s)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(a)
}

// Create godoc
// @Summary      Create a broadcast (draft)
// @Description  template_name must be a template approved in Meta's WhatsApp Manager; params are its body parameters, where {{name}} and {{first_name}} are replaced per contact. Only contacts who granted WhatsApp consent are messaged.
// @Tags         Broadcasts
// @Accept       json
// @Produce      json
// @Param        body  body  broadcastBody  true  "Broadcast"
// @Success      201  {object}  repo.Broadcast
// @Security     BearerAuth
// @Router       /broadcasts [post]
func (h *BroadcastHandler) Create(c *fiber.Ctx) error {
	var body broadcastBody
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if msg := body.validate(); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
	}
	b := &repo.Broadcast{Name: body.Name, TemplateName: body.TemplateName, Language: body.Language, Params: body.Params, Segment: body.Segment}
	if err := h.broadcasts.Create(c.Context(), b, localsUserPtr(c)); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, repo.AuditCreate, "whatsapp_broadcast", b.ID, fiber.Map{"template": b.TemplateName, "segment": b.Segment})
	}
	got, err := h.broadcasts.Get(c.Context(), b.ID)
	if err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(got)
}

// List godoc
// @Summary      Broadcasts
// @Tags         Broadcasts
// @Produce      json
// @Success      200  {array}  repo.Broadcast
// @Security     BearerAuth
// @Router       /broadcasts [get]
func (h *BroadcastHandler) List(c *fiber.Ctx) error {
	list, err := h.broadcasts.List(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(list)
}

// Get godoc
// @Summary      A broadcast and its delivery counts
// @Tags         Broadcasts
// @Produce      json
// @Param        id  path  string  true  "Broadcast ID"
// @Success      200  {object}  repo.Broadcast
// @Security     BearerAuth
// @Router       /broadcasts/{id} [get]
func (h *BroadcastHandler) Get(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	b, err := h.broadcasts.Get(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(b)
}

// Recipients godoc
// @Summary      A broadcast's recipients
// @Tags         Broadcasts
// @Produce      json
// @Param        id      path   string  true   "Broadcast ID"
// @Param        status  query  string  false  "pending, sent, delivered, read, failed, skipped"
// @Success      200  {array}  repo.Recipient
// @Security     BearerAuth
// @Router       /broadcasts/{id}/recipients [get]
func (h *BroadcastHandler) Recipients(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	page, limit := pageParams(c, 100, 500)
	list, err := h.broadcasts.Recipients(c.Context(), id, c.Query("status"), limit, (page-1)*limit)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(list)
}

// Start godoc
// @Summary      Start sending a broadcast
// @Description  Snapshots the audience; messages go out through the background job at the company's rate. Contacts without WhatsApp consent are skipped.
// @Tags         Broadcasts
// @Param        id  path  string  true  "Broadcast ID"
// @Success      200  {object}  repo.Broadcast
// @Failure      409  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /broadcasts/{id}/start [post]
func (h *BroadcastHandler) Start(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	if err := h.broadcasts.Start(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, "start", "whatsapp_broadcast", id, nil)
	}
	b, err := h.broadcasts.Get(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(b)
}

// Cancel godoc
// @Summary      Cancel a broadcast
// @Tags         Broadcasts
// @Param        id  path  string  true  "Broadcast ID"
// @Success      204
// @Security     BearerAuth
// @Router       /broadcasts/{id}/cancel [post]
func (h *BroadcastHandler) Cancel(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	if err := h.broadcasts.Cancel(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, "cancel", "whatsapp_broadcast", id, nil)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// AssignThread godoc
// @Summary      Assign a WhatsApp conversation to an agent
// @Tags         WhatsApp
// @Accept       json
// @Param        id    path  string  true  "Thread ID"
// @Param        body  body  object{user_id=string}  true  "User (null to unassign)"
// @Success      204
// @Security     BearerAuth
// @Router       /threads/{id}/assign [patch]
func (h *BroadcastHandler) AssignThread(c *fiber.Ctx) error {
	id, ok := idParam(c, "id")
	if !ok {
		return nil
	}
	var body struct {
		UserID *uuid.UUID `json:"user_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if err := h.broadcasts.AssignThread(c.Context(), id, body.UserID); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// GetInboxSettings godoc
// @Summary      Inbox settings (broadcast rate, business-hours auto-reply)
// @Tags         Settings
// @Produce      json
// @Success      200  {object}  repo.InboxSettings
// @Security     BearerAuth
// @Router       /settings/inbox [get]
func (h *BroadcastHandler) GetInboxSettings(c *fiber.Ctx) error {
	s, err := h.broadcasts.InboxSettings(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(s)
}

// UpdateInboxSettings godoc
// @Summary      Update the inbox settings
// @Description  broadcast_per_minute: 1–1000 (stay within your WhatsApp messaging tier); hours: weekly opening hours and special periods (e.g. Ramadan), in the company timezone; outside them, a customer's message gets the auto-reply at most once every 12 hours.
// @Tags         Settings
// @Accept       json
// @Param        body  body  repo.InboxSettings  true  "Settings"
// @Success      200  {object}  repo.InboxSettings
// @Security     BearerAuth
// @Router       /settings/inbox [put]
func (h *BroadcastHandler) UpdateInboxSettings(c *fiber.Ctx) error {
	var s repo.InboxSettings
	if err := c.BodyParser(&s); err != nil {
		return badRequest(c, err)
	}
	if s.BroadcastPerMinute < 1 || s.BroadcastPerMinute > 1000 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "broadcast_per_minute must be between 1 and 1000"})
	}
	s.AutoReplyEN, s.AutoReplyAR = strings.TrimSpace(s.AutoReplyEN), strings.TrimSpace(s.AutoReplyAR)
	if len(s.AutoReplyEN) > 1000 || len(s.AutoReplyAR) > 2000 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "the auto-reply is too long"})
	}
	if s.AutoReplyEnabled && s.AutoReplyEN == "" && s.AutoReplyAR == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "write the auto-reply text"})
	}
	if s.Hours != nil {
		if err := s.Hours.Validate(); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "opening hours: " + err.Error()})
		}
	}
	if err := h.broadcasts.UpdateInboxSettings(c.Context(), &s); err != nil {
		return serverError(c, err)
	}
	return c.JSON(s)
}
