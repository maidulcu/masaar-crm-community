package handler

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/notice"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// LeaseOpsHandler serves property management around leases: the lease's agent and Ejari
// expiry, its payments, notices to the tenant, rent reminders and their settings.
type LeaseOpsHandler struct {
	ops      *repo.LeaseOpsRepo
	payments *repo.PaymentRepo
	audit    *repo.AuditLogRepo
}

func NewLeaseOpsHandler(ops *repo.LeaseOpsRepo, payments *repo.PaymentRepo, audit *repo.AuditLogRepo) *LeaseOpsHandler {
	return &LeaseOpsHandler{ops: ops, payments: payments, audit: audit}
}

func fmtLocal(c *fiber.Ctx, key string) string {
	s, _ := c.Locals(key).(string)
	return s
}

func leaseID(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	return id, nil
}

// Extras godoc
// @Summary      A lease's agent and Ejari expiry
// @Tags         Leases
// @Produce      json
// @Param        id  path  string  true  "Lease ID"
// @Success      200  {object}  repo.LeaseExtras
// @Security     BearerAuth
// @Router       /leases/{id}/extras [get]
func (h *LeaseOpsHandler) Extras(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	e, err := h.ops.Extras(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(e)
}

// SetExtras godoc
// @Summary      Set a lease's agent and Ejari expiry
// @Description  The agent receives the lease's alerts and bounced-cheque notices.
// @Tags         Leases
// @Accept       json
// @Param        id    path  string  true  "Lease ID"
// @Param        body  body  object{agent_id=string,ejari_expiry=string,rent_reminders=bool}  true  "Values (ejari_expiry YYYY-MM-DD or empty; rent_reminders: whether the tenant receives rent reminders)"
// @Success      204
// @Security     BearerAuth
// @Router       /leases/{id}/extras [patch]
func (h *LeaseOpsHandler) SetExtras(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	var body struct {
		AgentID       *uuid.UUID `json:"agent_id"`
		EjariExpiry   string     `json:"ejari_expiry"`
		RentReminders *bool      `json:"rent_reminders"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	expiry, err := parseDay(body.EjariExpiry)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ejari_expiry must be YYYY-MM-DD"})
	}
	if err := h.ops.SetLeaseAgent(c.Context(), id, body.AgentID, expiry); err != nil {
		return serverError(c, err)
	}
	if body.RentReminders != nil {
		if err := h.ops.SetRentReminders(c.Context(), id, *body.RentReminders); err != nil {
			return serverError(c, err)
		}
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, repo.AuditUpdate, "lease", id, fiber.Map{"agent_id": body.AgentID, "ejari_expiry": body.EjariExpiry, "rent_reminders": body.RentReminders})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Payments godoc
// @Summary      A lease's payments
// @Tags         Leases
// @Produce      json
// @Param        id  path  string  true  "Lease ID"
// @Success      200  {array}  domain.Payment
// @Security     BearerAuth
// @Router       /leases/{id}/payments [get]
func (h *LeaseOpsHandler) Payments(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	list, err := h.payments.GetByLeaseID(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if list == nil {
		list = []domain.Payment{}
	}
	return c.JSON(list)
}

// Reminders godoc
// @Summary      Rent reminders sent for a lease
// @Tags         Leases
// @Produce      json
// @Param        id  path  string  true  "Lease ID"
// @Success      200  {array}  domain.PaymentReminder
// @Security     BearerAuth
// @Router       /leases/{id}/reminders [get]
func (h *LeaseOpsHandler) Reminders(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	list, err := h.ops.PaymentReminders(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if list == nil {
		list = []domain.PaymentReminder{}
	}
	return c.JSON(list)
}

// ── Notices ───────────────────────────────────────────────────────────────────

type noticeBody struct {
	Kind          string   `json:"kind"`
	NewAnnualRent *float64 `json:"new_annual_rent"`
	EffectiveDate string   `json:"effective_date"`
	BodyEN        string   `json:"body_en"`
	BodyAR        string   `json:"body_ar"`
}

// DraftNotice godoc
// @Summary      Draft a notice to the tenant
// @Description  Returns English and Arabic text to review and edit; nothing is stored. Kinds: rent_change, renewal_terms, non_renewal, other. Drafts are not legal advice; check the current rules of the emirate.
// @Tags         Leases
// @Accept       json
// @Produce      json
// @Param        id    path  string      true  "Lease ID"
// @Param        body  body  noticeBody  true  "Notice"
// @Success      200  {object}  object{body_en=string,body_ar=string,current_annual_rent=number}
// @Security     BearerAuth
// @Router       /leases/{id}/notices/draft [post]
func (h *LeaseOpsHandler) DraftNotice(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	var body noticeBody
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if !notice.ValidKind(body.Kind) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "unknown notice kind"})
	}
	nc, err := h.ops.NoticeContext(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	settings, err := h.ops.Settings(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	in := notice.Input{Kind: body.Kind, Today: time.Now(), TenantName: nc.TenantName, PropertyName: nc.PropertyName,
		UnitNumber: nc.UnitNumber, EjariNumber: nc.EjariNumber, CompanyName: nc.CompanyName, EndDate: nc.EndDate,
		Currency: nc.Currency, CurrentAnnual: nc.MonthlyRent * 12, NewAnnual: nc.MonthlyRent * 12, NoticeDays: settings.RentNoticeDays}
	if body.NewAnnualRent != nil {
		in.NewAnnual = *body.NewAnnualRent
	}
	if d, err := parseDay(body.EffectiveDate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "effective_date must be YYYY-MM-DD"})
	} else if d != nil {
		in.EffectiveDate = *d
	}
	en, ar := notice.Draft(in)
	deadline := nc.EndDate.AddDate(0, 0, -settings.RentNoticeDays)
	return c.JSON(fiber.Map{"body_en": en, "body_ar": ar, "current_annual_rent": in.CurrentAnnual, "currency": nc.Currency,
		"lease_end": nc.EndDate.Format("2006-01-02"), "notice_deadline": deadline.Format("2006-01-02"),
		"notice_days": settings.RentNoticeDays})
}

// CreateNotice godoc
// @Summary      Save a notice to the tenant
// @Tags         Leases
// @Accept       json
// @Produce      json
// @Param        id    path  string      true  "Lease ID"
// @Param        body  body  noticeBody  true  "Notice (edited text)"
// @Success      201  {object}  domain.LeaseNotice
// @Security     BearerAuth
// @Router       /leases/{id}/notices [post]
func (h *LeaseOpsHandler) CreateNotice(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	var body noticeBody
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if !notice.ValidKind(body.Kind) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "unknown notice kind"})
	}
	if strings.TrimSpace(body.BodyEN) == "" && strings.TrimSpace(body.BodyAR) == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "the notice has no text"})
	}
	n := &domain.LeaseNotice{LeaseID: id, Kind: body.Kind, NewRent: body.NewAnnualRent, BodyEN: body.BodyEN, BodyAR: body.BodyAR,
		CreatedBy: localsUserPtr(c)}
	if d, err := parseDay(body.EffectiveDate); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "effective_date must be YYYY-MM-DD"})
	} else {
		n.EffectiveDate = d
	}
	if nc, err := h.ops.NoticeContext(c.Context(), id); err == nil {
		cur := nc.MonthlyRent * 12
		n.CurrentRent = &cur
	}
	if err := h.ops.CreateNotice(c.Context(), n); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, repo.AuditCreate, "lease_notice", n.ID, fiber.Map{"lease_id": id, "kind": n.Kind})
	}
	return c.Status(fiber.StatusCreated).JSON(n)
}

// ListNotices godoc
// @Summary      A lease's notices
// @Tags         Leases
// @Produce      json
// @Param        id  path  string  true  "Lease ID"
// @Success      200  {array}  domain.LeaseNotice
// @Security     BearerAuth
// @Router       /leases/{id}/notices [get]
func (h *LeaseOpsHandler) ListNotices(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	list, err := h.ops.ListNotices(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	if list == nil {
		list = []domain.LeaseNotice{}
	}
	return c.JSON(list)
}

// GetNotice godoc
// @Summary      One notice (for printing)
// @Tags         Leases
// @Produce      json
// @Param        id  path  string  true  "Notice ID"
// @Success      200  {object}  domain.LeaseNotice
// @Security     BearerAuth
// @Router       /lease-notices/{id} [get]
func (h *LeaseOpsHandler) GetNotice(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	n, err := h.ops.GetNotice(c.Context(), id)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(n)
}

var noticeChannels = map[string]bool{"email": true, "whatsapp": true, "hand": true, "registered_mail": true, "notary": true, "other": true}

// MarkNoticeSent godoc
// @Summary      Record that a notice was served
// @Tags         Leases
// @Accept       json
// @Param        id    path  string  true  "Notice ID"
// @Param        body  body  object{via=string,sent_at=string}  true  "via: email, whatsapp, hand, registered_mail, notary, other; sent_at YYYY-MM-DD (default today)"
// @Success      204
// @Failure      409  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /lease-notices/{id}/sent [post]
func (h *LeaseOpsHandler) MarkNoticeSent(c *fiber.Ctx) error {
	id, err := leaseID(c)
	if id == uuid.Nil {
		return err
	}
	var body struct {
		Via    string `json:"via"`
		SentAt string `json:"sent_at"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if !noticeChannels[body.Via] {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "unknown channel"})
	}
	at := time.Now()
	if d, err := parseDay(body.SentAt); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "sent_at must be YYYY-MM-DD"})
	} else if d != nil {
		if d.After(time.Now()) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "sent_at is in the future"})
		}
		at = *d
	}
	if err := h.ops.MarkNoticeSent(c.Context(), id, body.Via, at); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, repo.AuditUpdate, "lease_notice", id, fiber.Map{"sent_via": body.Via, "sent_at": at.Format("2006-01-02")})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ── Settings ──────────────────────────────────────────────────────────────────

// GetSettings godoc
// @Summary      Property management settings
// @Tags         Settings
// @Produce      json
// @Success      200  {object}  repo.PropertySettings
// @Security     BearerAuth
// @Router       /settings/property [get]
func (h *LeaseOpsHandler) GetSettings(c *fiber.Ctx) error {
	s, err := h.ops.Settings(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(s)
}

var errSettings = errors.New("invalid settings")

func validPropertySettings(s *repo.PropertySettings) error {
	if len(s.LeaseAlertDays) > 8 {
		return errSettings
	}
	for _, d := range s.LeaseAlertDays {
		if d < 1 || d > 365 {
			return errSettings
		}
	}
	if s.RentNoticeDays < 0 || s.RentNoticeDays > 365 || s.RentReminderDaysBefore < 0 || s.RentReminderDaysBefore > 30 {
		return errSettings
	}
	if len(s.RentReminderWATemplate) > 512 || len(s.RentReminderWALanguage) > 15 {
		return errSettings
	}
	return nil
}

// UpdateSettings godoc
// @Summary      Update property management settings
// @Description  lease_alert_days: up to 8 values of 1–365 days before a lease ends; rent_notice_days: 0–365 (Dubai: 90, check the rules for your emirate); rent_reminder_days_before: 0–30; rent_reminder_wa_template: an approved WhatsApp template with four body parameters (tenant name, amount, due date, property).
// @Tags         Settings
// @Accept       json
// @Param        body  body  repo.PropertySettings  true  "Settings"
// @Success      200  {object}  repo.PropertySettings
// @Security     BearerAuth
// @Router       /settings/property [put]
func (h *LeaseOpsHandler) UpdateSettings(c *fiber.Ctx) error {
	var s repo.PropertySettings
	if err := c.BodyParser(&s); err != nil {
		return badRequest(c, err)
	}
	if s.LeaseAlertDays == nil {
		s.LeaseAlertDays = []int{}
	}
	s.RentReminderWATemplate = strings.TrimSpace(s.RentReminderWATemplate)
	if s.RentReminderWALanguage == "" {
		s.RentReminderWALanguage = "en"
	}
	if err := validPropertySettings(&s); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "invalid settings: check the ranges"})
	}
	if err := h.ops.UpdateSettings(c.Context(), &s); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		if cid, err := uuid.Parse(fmtLocal(c, "company_id")); err == nil {
			h.audit.Log(c.Context(), *actor, repo.AuditUpdate, "company_settings", cid, fiber.Map{"property_settings": s})
		}
	}
	return c.JSON(s)
}
