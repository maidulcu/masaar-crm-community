package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/ws"
)

// OfferHandler manages buyer offers on listings.
//
//	GET    /api/v1/offers                — list (filter: listing_id, contact_id, status)
//	GET    /api/v1/offers/:id            — single offer
//	POST   /api/v1/offers                — create offer
//	PATCH  /api/v1/offers/:id/status     — update status (under_review, rejected, expired)
//	POST   /api/v1/offers/:id/counter    — create counter-offer
//	POST   /api/v1/offers/:id/accept     — accept offer → auto-create deal
//	DELETE /api/v1/offers/:id            — delete (admin only)
type OfferHandler struct {
	offerRepo   *repo.OfferRepo
	contactRepo *repo.ContactRepo
	leadRepo    *repo.LeadRepo
	dealRepo    *repo.DealRepo
	hub         *ws.Hub
}

func NewOfferHandler(
	offerRepo *repo.OfferRepo,
	contactRepo *repo.ContactRepo,
	leadRepo *repo.LeadRepo,
	dealRepo *repo.DealRepo,
	hub *ws.Hub,
) *OfferHandler {
	return &OfferHandler{
		offerRepo:   offerRepo,
		contactRepo: contactRepo,
		leadRepo:    leadRepo,
		dealRepo:    dealRepo,
		hub:         hub,
	}
}

// List handles GET /api/v1/offers
func (h *OfferHandler) List(c *fiber.Ctx) error {
	var listingID, contactID *uuid.UUID
	if v := c.Query("listing_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid listing_id"})
		}
		listingID = &id
	}
	if v := c.Query("contact_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid contact_id"})
		}
		contactID = &id
	}
	status := c.Query("status")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	offers, total, err := h.offerRepo.List(c.Context(), listingID, contactID, status, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"data": offers,
		"meta": fiber.Map{"total": total, "page": page, "limit": limit},
	})
}

// Get handles GET /api/v1/offers/:id
func (h *OfferHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	offer, err := h.offerRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "offer not found"})
	}
	return c.JSON(offer)
}

// Create handles POST /api/v1/offers
func (h *OfferHandler) Create(c *fiber.Ctx) error {
	var body struct {
		ListingID   string  `json:"listing_id"`
		ContactID   string  `json:"contact_id"`
		OfferAmount float64 `json:"offer_amount"`
		Currency    string  `json:"currency"`
		Terms       string  `json:"terms"`
		Notes       string  `json:"notes"`
		ValidUntil  string  `json:"valid_until"` // RFC3339 or date
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.ListingID == "" || body.ContactID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "listing_id and contact_id are required"})
	}
	if body.OfferAmount <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "offer_amount must be positive"})
	}

	listingID, err := uuid.Parse(body.ListingID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid listing_id"})
	}
	contactID, err := uuid.Parse(body.ContactID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid contact_id"})
	}

	agentID, _ := c.Locals("user_id").(uuid.UUID)

	offer := &domain.Offer{
		ListingID:   listingID,
		ContactID:   contactID,
		AgentID:     &agentID,
		OfferAmount: body.OfferAmount,
		Currency:    body.Currency,
		Terms:       body.Terms,
		Notes:       body.Notes,
	}
	if body.Currency == "" {
		offer.Currency = "AED"
	}
	if body.ValidUntil != "" {
		t, err := time.Parse(time.RFC3339, body.ValidUntil)
		if err != nil {
			t, err = time.Parse("2006-01-02", body.ValidUntil)
		}
		if err == nil {
			offer.ValidUntil = &t
		}
	}

	if err := h.offerRepo.Create(c.Context(), offer); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Notify via WebSocket
	h.hub.Broadcast(ws.Event{
		Type: "offer.created",
		Payload: fiber.Map{
			"offer_id":   offer.ID,
			"listing_id": offer.ListingID,
			"amount":     offer.OfferAmount,
			"currency":   offer.Currency,
		},
	})

	return c.Status(fiber.StatusCreated).JSON(offer)
}

// UpdateStatus handles PATCH /api/v1/offers/:id/status
// Allowed transitions: submitted→under_review, any→rejected, any→expired
func (h *OfferHandler) UpdateStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	allowed := map[string]bool{
		"under_review": true,
		"rejected":     true,
		"expired":      true,
	}
	if !allowed[body.Status] {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "status must be one of: under_review, rejected, expired",
		})
	}

	if err := h.offerRepo.UpdateStatus(c.Context(), id, domain.OfferStatus(body.Status)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true, "status": body.Status})
}

// Counter handles POST /api/v1/offers/:id/counter
// Creates a counter-offer and marks the parent as 'countered'.
func (h *OfferHandler) Counter(c *fiber.Ctx) error {
	parentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	parent, err := h.offerRepo.GetByID(c.Context(), parentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "offer not found"})
	}
	if parent.Status == domain.OfferAccepted || parent.Status == domain.OfferRejected {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "cannot counter an accepted or rejected offer",
		})
	}

	var body struct {
		OfferAmount float64 `json:"offer_amount"`
		Terms       string  `json:"terms"`
		Notes       string  `json:"notes"`
		ValidUntil  string  `json:"valid_until"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.OfferAmount <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "offer_amount must be positive"})
	}

	agentID, _ := c.Locals("user_id").(uuid.UUID)
	counter := &domain.Offer{
		ListingID:   parent.ListingID,
		ContactID:   parent.ContactID,
		AgentID:     &agentID,
		OfferAmount: body.OfferAmount,
		Currency:    parent.Currency,
		Terms:       body.Terms,
		Notes:       body.Notes,
	}
	if body.ValidUntil != "" {
		t, err := time.Parse(time.RFC3339, body.ValidUntil)
		if err != nil {
			t, err = time.Parse("2006-01-02", body.ValidUntil)
		}
		if err == nil {
			counter.ValidUntil = &t
		}
	}

	if err := h.offerRepo.Counter(c.Context(), parentID, counter); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(counter)
}

// Accept handles POST /api/v1/offers/:id/accept
// Marks the offer as accepted and auto-creates a Deal.
func (h *OfferHandler) Accept(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	offer, err := h.offerRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "offer not found"})
	}
	if offer.Status == domain.OfferAccepted {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "offer already accepted"})
	}
	if offer.Status == domain.OfferRejected || offer.Status == domain.OfferExpired {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "cannot accept a rejected or expired offer"})
	}

	// Auto-create a Deal for the accepted offer.
	// Find or create a lead for the buyer contact so the deal has a lead_id.
	leads, _ := h.leadRepo.List(c.Context(), repo.LeadFilter{
		ContactID: &offer.ContactID,
		Limit:     1,
	})

	var leadID uuid.UUID
	if len(leads) > 0 {
		leadID = leads[0].ID
	} else {
		// Create a minimal lead for this contact
		lead := &domain.Lead{
			ContactID: offer.ContactID,
			Stage:     domain.StageQualified,
			Source:    "web",
			Notes:     "Auto-created from accepted offer",
		}
		lead.Currency = "AED"
		if err := h.leadRepo.Create(c.Context(), lead); err == nil {
			leadID = lead.ID
		}
	}

	ownerID, _ := c.Locals("user_id").(uuid.UUID)
	listingTitle := offer.ListingTitle
	if listingTitle == "" {
		listingTitle = "Listing"
	}

	deal := &domain.Deal{
		LeadID:      leadID,
		Title:       "Offer: " + listingTitle,
		Stage:       domain.DealStageOpen,
		Amount:      offer.OfferAmount,
		Currency:    offer.Currency,
		Probability: 80,
		OwnerID:     ownerID,
	}
	if err := h.dealRepo.Create(c.Context(), deal); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create deal: " + err.Error()})
	}

	// Link offer → deal and mark as accepted
	if err := h.offerRepo.SetDeal(c.Context(), id, deal.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	h.hub.Broadcast(ws.Event{
		Type: "offer.accepted",
		Payload: fiber.Map{
			"offer_id": id,
			"deal_id":  deal.ID,
			"amount":   offer.OfferAmount,
		},
	})

	return c.JSON(fiber.Map{
		"ok":      true,
		"deal_id": deal.ID,
		"offer":   offer,
	})
}

// Delete handles DELETE /api/v1/offers/:id (admin only)
func (h *OfferHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.offerRepo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
