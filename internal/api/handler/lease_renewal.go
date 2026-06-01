package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type LeaseRenewalHandler struct {
	renewalRepo  *repo.LeaseRenewalRepo
	templateRepo *repo.RenewalTemplateRepo
	logRepo      *repo.RenewalCommunicationLogRepo
}

func NewLeaseRenewalHandler(renewalRepo *repo.LeaseRenewalRepo, templateRepo *repo.RenewalTemplateRepo, logRepo *repo.RenewalCommunicationLogRepo) *LeaseRenewalHandler {
	return &LeaseRenewalHandler{
		renewalRepo:  renewalRepo,
		templateRepo: templateRepo,
		logRepo:      logRepo,
	}
}

// @Summary List lease renewals
// @Tags Lease Renewals
// @Security Bearer
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Param status query string false "Filter by status"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/lease-renewals [get]
func (h *LeaseRenewalHandler) List(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	status := c.Query("status", "")
	var renewals []domain.LeaseRenewalWorkflow
	var total int
	var err error

	if status != "" {
		renewals, total, err = h.renewalRepo.ListByStatus(c.Context(), companyID, status, limit, offset)
	} else {
		renewals, total, err = h.renewalRepo.List(c.Context(), companyID, limit, offset)
	}

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch renewals"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": renewals,
		"meta": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// @Summary Get renewal details
// @Tags Lease Renewals
// @Security Bearer
// @Param id path string true "Renewal ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/lease-renewals/{id} [get]
func (h *LeaseRenewalHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid renewal ID"})
	}

	renewal, err := h.renewalRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Renewal not found"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": renewal})
}

// @Summary Initiate lease renewal
// @Tags Lease Renewals
// @Security Bearer
// @Param lease_id path string true "Lease ID"
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/lease-renewals/{lease_id}/initiate [post]
func (h *LeaseRenewalHandler) Initiate(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))
	leaseID, err := uuid.Parse(c.Params("lease_id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid lease ID"})
	}

	renewal := &domain.LeaseRenewalWorkflow{
		ID:               uuid.New(),
		CompanyID:        companyID,
		LeaseID:          leaseID,
		RenewalDate:      time.Now().AddDate(0, 0, 90),
		RenewalStatus:    domain.RenewalPending,
		DaysBeforeExpiry: 90,
		TenantResponse:   domain.ResponsePending,
	}

	if err := h.renewalRepo.Create(c.Context(), renewal); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to initiate renewal"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": renewal})
}

// @Summary Propose renewal terms
// @Tags Lease Renewals
// @Security Bearer
// @Param id path string true "Renewal ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/lease-renewals/{id}/propose [put]
func (h *LeaseRenewalHandler) Propose(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid renewal ID"})
	}

	renewal, err := h.renewalRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Renewal not found"})
	}

	var req struct {
		ProposedRentAmount *float64               `json:"proposed_rent_amount"`
		ProposedTerms      map[string]interface{} `json:"proposed_terms"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.ProposedRentAmount != nil {
		renewal.ProposedRentAmount = req.ProposedRentAmount
	}
	if req.ProposedTerms != nil {
		renewal.ProposedTerms = req.ProposedTerms
	}

	if err := h.renewalRepo.Update(c.Context(), renewal); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update renewal"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": renewal})
}

// @Summary Send renewal offer
// @Tags Lease Renewals
// @Security Bearer
// @Param id path string true "Renewal ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/lease-renewals/{id}/send-offer [post]
func (h *LeaseRenewalHandler) SendOffer(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid renewal ID"})
	}

	renewal, err := h.renewalRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Renewal not found"})
	}

	var req struct {
		TemplateID        uuid.UUID `json:"template_id"`
		CommunicationType string    `json:"communication_type"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	renewal.RenewalStatus = domain.RenewalOfferSent
	if err := h.renewalRepo.Update(c.Context(), renewal); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to send offer"})
	}

	var commType domain.CommunicationType
	switch req.CommunicationType {
	case "email":
		commType = domain.CommEmailSent
	case "whatsapp":
		commType = domain.CommWhatsAppOutbound
	default:
		commType = domain.CommEmailSent
	}

	log := &domain.RenewalCommunicationLog{
		ID:                uuid.New(),
		RenewalID:         renewal.ID,
		CommunicationType: commType,
		TemplateID:        &req.TemplateID,
		DeliveryStatus:    domain.DeliverySent,
	}
	now := time.Now()
	log.SentDate = &now

	if err := h.logRepo.Create(c.Context(), log); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to log communication"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": renewal})
}

// @Summary Accept renewal offer
// @Tags Lease Renewals
// @Security Bearer
// @Param id path string true "Renewal ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/lease-renewals/{id}/accept [put]
func (h *LeaseRenewalHandler) Accept(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid renewal ID"})
	}

	renewal, err := h.renewalRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Renewal not found"})
	}

	renewal.RenewalStatus = domain.RenewalAccepted
	renewal.TenantResponse = domain.ResponseAccepted

	if err := h.renewalRepo.Update(c.Context(), renewal); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to accept renewal"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": renewal})
}

// @Summary Reject renewal offer
// @Tags Lease Renewals
// @Security Bearer
// @Param id path string true "Renewal ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/lease-renewals/{id}/reject [put]
func (h *LeaseRenewalHandler) Reject(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid renewal ID"})
	}

	renewal, err := h.renewalRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Renewal not found"})
	}

	renewal.RenewalStatus = domain.RenewalRejected
	renewal.TenantResponse = domain.ResponseRejected

	if err := h.renewalRepo.Update(c.Context(), renewal); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to reject renewal"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": renewal})
}

// @Summary Submit counter offer
// @Tags Lease Renewals
// @Security Bearer
// @Param id path string true "Renewal ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/lease-renewals/{id}/counter-offer [post]
func (h *LeaseRenewalHandler) CounterOffer(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid renewal ID"})
	}

	renewal, err := h.renewalRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Renewal not found"})
	}

	var req struct {
		CounterOfferAmount float64 `json:"counter_offer_amount"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	renewal.TenantResponse = domain.ResponseCounterOffer
	renewal.TenantCounterOffer = &req.CounterOfferAmount
	now := time.Now()
	renewal.CounterOfferDate = &now

	if err := h.renewalRepo.Update(c.Context(), renewal); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to submit counter offer"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": renewal})
}

// @Summary List renewal templates
// @Tags Lease Renewals
// @Security Bearer
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/renewal-templates [get]
func (h *LeaseRenewalHandler) ListTemplates(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	templates, err := h.templateRepo.List(c.Context(), companyID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch templates"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": templates})
}

// @Summary Create renewal template
// @Tags Lease Renewals
// @Security Bearer
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/renewal-templates [post]
func (h *LeaseRenewalHandler) CreateTemplate(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	var req struct {
		TemplateName string `json:"template_name"`
		EmailSubject string `json:"email_subject"`
		EmailBody    string `json:"email_body"`
		WhatsAppMsg  string `json:"whatsapp_message"`
		Language     string `json:"language"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	template := &domain.RenewalCommunicationTemplate{
		ID:           uuid.New(),
		CompanyID:    companyID,
		TemplateName: req.TemplateName,
		EmailSubject: req.EmailSubject,
		EmailBody:    req.EmailBody,
		WhatsAppMsg:  req.WhatsAppMsg,
		Language:     domain.Language(req.Language),
	}

	if err := h.templateRepo.Create(c.Context(), template); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create template"})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": template})
}

// @Summary Update renewal template
// @Tags Lease Renewals
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path string true "Template UUID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/renewal-templates/{id} [patch]
func (h *LeaseRenewalHandler) UpdateTemplate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid template ID"})
	}

	tpl, err := h.templateRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Template not found"})
	}

	var req struct {
		TemplateName string `json:"template_name"`
		EmailSubject string `json:"email_subject"`
		EmailBody    string `json:"email_body"`
		WhatsAppMsg  string `json:"whatsapp_message"`
		Language     string `json:"language"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.TemplateName != "" {
		tpl.TemplateName = req.TemplateName
	}
	if req.EmailSubject != "" {
		tpl.EmailSubject = req.EmailSubject
	}
	if req.EmailBody != "" {
		tpl.EmailBody = req.EmailBody
	}
	if req.WhatsAppMsg != "" {
		tpl.WhatsAppMsg = req.WhatsAppMsg
	}
	if req.Language != "" {
		tpl.Language = domain.Language(req.Language)
	}

	if err := h.templateRepo.Update(c.Context(), tpl); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update template"})
	}
	return c.JSON(fiber.Map{"data": tpl})
}

// @Summary Delete renewal template
// @Tags Lease Renewals
// @Security Bearer
// @Produce json
// @Param id path string true "Template UUID"
// @Success 204
// @Router /api/v1/renewal-templates/{id} [delete]
func (h *LeaseRenewalHandler) DeleteTemplate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid template ID"})
	}
	if err := h.templateRepo.Delete(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete template"})
	}
	return c.SendStatus(http.StatusNoContent)
}
