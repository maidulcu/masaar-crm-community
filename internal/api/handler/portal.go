package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/compliance"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/portal"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/tenant"
	"github.com/maidulcu/masaar-crm/internal/webhook"
)

// PortalHandler connects property portals: inbound leads, the outbound listing feed, and
// the lead-source report.
type PortalHandler struct {
	portals  *repo.PortalRepo
	contacts *repo.ContactRepo
	leads    *repo.LeadRepo
	listings *repo.ListingRepo
	users    *repo.UserRepo
	company  *repo.CompanySettingsRepo
	dispatch *webhook.Dispatcher
	assign   func(companyID, leadID uuid.UUID) // lead rotation (may be nil)
	apiURL   string                            // public API origin, for the URLs shown to admins
	appURL   string                            // public web origin, for listing links in the feed
}

func NewPortalHandler(portals *repo.PortalRepo, contacts *repo.ContactRepo, leads *repo.LeadRepo, listings *repo.ListingRepo,
	users *repo.UserRepo, company *repo.CompanySettingsRepo, dispatch *webhook.Dispatcher,
	assign func(companyID, leadID uuid.UUID), apiURL, appURL string) *PortalHandler {
	return &PortalHandler{portals: portals, contacts: contacts, leads: leads, listings: listings, users: users,
		company: company, dispatch: dispatch, assign: assign, apiURL: strings.TrimRight(apiURL, "/"), appURL: strings.TrimRight(appURL, "/")}
}

func validPortal(c *fiber.Ctx) (string, bool) {
	p := c.Params("portal")
	_, ok := portal.Portals[p]
	return p, ok
}

// ListConnections godoc
// @Summary      Portal connections
// @Tags         Portals
// @Produce      json
// @Success      200  {array}  repo.PortalConnection
// @Security     BearerAuth
// @Router       /settings/portals [get]
func (h *PortalHandler) ListConnections(c *fiber.Ctx) error {
	conns, err := h.portals.ListConnections(c.Context())
	if err != nil {
		return serverError(c, err)
	}
	if conns == nil {
		conns = []repo.PortalConnection{}
	}
	return c.JSON(fiber.Map{"connections": conns, "portals": portal.Portals})
}

// Connect godoc
// @Summary      Connect a portal (or rotate its tokens)
// @Description  Admin only. Returns the lead webhook URL and the listing feed URL, each with a secret token. They are shown once; calling again replaces them.
// @Tags         Portals
// @Produce      json
// @Param        portal  path  string  true  "property_finder | bayut | dubizzle | portal"
// @Success      201  {object}  object{lead_webhook_url=string,feed_url=string}
// @Security     BearerAuth
// @Router       /settings/portals/{portal} [post]
func (h *PortalHandler) Connect(c *fiber.Ctx) error {
	p, ok := validPortal(c)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "unknown portal"})
	}
	inbound, feed, err := h.portals.Connect(c.Context(), p)
	if err != nil {
		return serverError(c, err)
	}
	base := h.apiURL
	if base == "" {
		base = c.BaseURL()
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"portal":           p,
		"lead_webhook_url": fmt.Sprintf("%s/webhooks/portals/%s?token=%s", base, p, inbound),
		"feed_url":         fmt.Sprintf("%s/api/public/feeds/%s/listings.xml?token=%s", base, p, feed),
	})
}

// SetEnabled godoc
// @Summary      Pause or resume a portal connection
// @Tags         Portals
// @Accept       json
// @Param        portal  path  string               true  "Portal"
// @Param        body    body  object{enabled=bool}  true  "State"
// @Success      204
// @Security     BearerAuth
// @Router       /settings/portals/{portal} [patch]
func (h *PortalHandler) SetEnabled(c *fiber.Ctx) error {
	p, ok := validPortal(c)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "unknown portal"})
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	if err := h.portals.SetEnabled(c.Context(), p, body.Enabled); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Disconnect godoc
// @Summary      Disconnect a portal (its URLs stop working)
// @Tags         Portals
// @Param        portal  path  string  true  "Portal"
// @Success      204
// @Security     BearerAuth
// @Router       /settings/portals/{portal} [delete]
func (h *PortalHandler) Disconnect(c *fiber.Ctx) error {
	p, ok := validPortal(c)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "unknown portal"})
	}
	if err := h.portals.Disconnect(c.Context(), p); err != nil {
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

const maxPortalLeadBytes = 64 << 10

// ReceiveLead godoc
// @Summary      Receive a lead from a property portal
// @Description  The portal posts a JSON lead to this URL (token from Settings → Portals). Common field names are recognised at any depth: name/full_name/first_name+last_name, phone/mobile/phone_number, email, message/comment, reference/listing_reference, and lead_id/id for deduplication. A redelivered lead creates nothing new. The contact is created or matched by phone, the lead gets the portal as its source and goes through lead rotation.
// @Tags         Public
// @Accept       json
// @Produce      json
// @Param        portal  path   string  true  "property_finder | bayut | dubizzle | portal"
// @Param        token   query  string  true  "Inbound token"
// @Success      201  {object}  object{lead_id=string}
// @Success      200  {object}  object{duplicate=bool}
// @Router       /webhooks/portals/{portal} [post]
func (h *PortalHandler) ReceiveLead(c *fiber.Ctx) error {
	p, ok := validPortal(c)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "unknown portal"})
	}
	cid, err := h.portals.CompanyForToken(c.Context(), p, "inbound", c.Query("token"))
	if errors.Is(err, repo.ErrBadPortalToken) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}
	if err != nil {
		return serverError(c, err)
	}
	body := c.Body()
	if len(body) > maxPortalLeadBytes {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "payload too large"})
	}
	lead, err := portal.ParseLead(bytes.Clone(body))
	if errors.Is(err, portal.ErrNoContact) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "the lead has no phone number"})
	}
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON"})
	}

	ctx := tenant.With(c.Context(), cid)
	// A redelivered lead is found by the portal's own id (leads.external_source/external_ref).
	if existing, err := h.leads.FindByExternalRef(ctx, p, lead.ExternalID); err == nil {
		return c.JSON(fiber.Map{"duplicate": true, "lead_id": existing.ID})
	}
	created, err := h.createLead(ctx, cid, p, lead)
	if errors.Is(err, repo.ErrDuplicateExternalRef) {
		// Two deliveries of the same lead raced; the other one created it.
		return c.JSON(fiber.Map{"duplicate": true})
	}
	if err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"lead_id": created.ID, "contact_id": created.ContactID})
}

func (h *PortalHandler) createLead(ctx context.Context, cid uuid.UUID, p string, in *portal.Lead) (*domain.Lead, error) {
	contact, err := h.contacts.Upsert(ctx, in.Phone, in.Name)
	if err != nil {
		return nil, err
	}
	if in.Email != "" && contact.Email == "" {
		contact.Email = in.Email
		_ = h.contacts.Update(ctx, contact)
	}
	notes := []string{"Lead from " + portal.Portals[p]}
	if in.ListingReference != "" {
		ref := "Listing reference: " + in.ListingReference
		if _, title, err := h.portals.ListingIDByReference(ctx, in.ListingReference); err == nil && title != "" {
			ref += " (" + title + ")"
		}
		notes = append(notes, ref)
	}
	if in.Message != "" {
		notes = append(notes, in.Message)
	}
	lead := &domain.Lead{ContactID: contact.ID, Stage: domain.StageNew, Source: domain.LeadSource(p),
		Currency: defaultCurrency(ctx), Notes: strings.Join(notes, "\n"),
		ExternalSource: p, ExternalRef: in.ExternalID}
	if err := h.leads.Create(ctx, lead); err != nil {
		return nil, err
	}
	if err := h.portals.TouchLastLead(ctx, p); err != nil {
		log.Printf("portal lead: last_lead_at for %s: %v", p, err)
	}
	if h.assign != nil {
		h.assign(cid, lead.ID)
	}
	if h.dispatch != nil {
		h.dispatch.Dispatch(cid, webhook.EventLeadCreated, fiber.Map{
			"lead_id": lead.ID, "contact_id": contact.ID, "source": lead.Source,
			"contact": fiber.Map{"name": contact.FullName, "phone": contact.PhoneWA},
		})
	}
	return lead, nil
}

// Feed godoc
// @Summary      Listing feed for a portal
// @Description  Published listings as XML (format "masaar-listings/1"). Listings whose advertising permit is missing or expired are left out, unless an admin overrode the check. Portals that need their own XML schema can map this feed; the portals' schemas are not public.
// @Tags         Public
// @Produce      xml
// @Param        portal  path   string  true  "Portal"
// @Param        token   query  string  true  "Feed token"
// @Success      200
// @Router       /api/public/feeds/{portal}/listings.xml [get]
func (h *PortalHandler) Feed(c *fiber.Ctx) error {
	p, ok := validPortal(c)
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "unknown portal"})
	}
	cid, err := h.portals.CompanyForToken(c.Context(), p, "feed", c.Query("token"))
	if errors.Is(err, repo.ErrBadPortalToken) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}
	if err != nil {
		return serverError(c, err)
	}
	ctx := tenant.With(c.Context(), cid)
	listings, err := h.listings.ListPublished(ctx)
	if err != nil {
		return serverError(c, err)
	}
	settings, _ := h.company.Get(ctx)
	if settings == nil {
		settings = &domain.CompanySettings{}
	}
	agents := map[uuid.UUID]domain.User{}
	if users, _, err := h.users.ListByCompany(ctx, cid, 1, 1000); err == nil {
		for _, u := range users {
			agents[u.ID] = u
		}
	}
	now := time.Now()
	items := make([]portal.FeedListing, 0, len(listings))
	for _, l := range listings {
		if !compliance.ListingPermit(&l, now).Advertisable() {
			continue
		}
		it := portal.FeedListing{
			ID: l.ID.String(), Reference: l.ReferenceNumber, PermitNumber: l.PermitNumber, PermitExpiry: l.PermitExpiry,
			Title: l.Title, Description: l.Description, PropertyType: string(l.PropertyType), ListingType: string(l.ListingType),
			Price: l.Price, Currency: l.Currency, Emirate: l.Emirate, City: l.City, Community: l.Community,
			SubCommunity: l.Subcommunity, Area: l.Area, Bedrooms: l.Bedrooms, Bathrooms: l.Bathrooms, SizeSqft: l.TotalSqft,
			Amenities: l.Amenities, Latitude: l.Latitude, Longitude: l.Longitude, UpdatedAt: l.UpdatedAt, AvailableFrom: l.AvailableFrom,
			PublicURL:   h.appURL + "/l/" + l.ID.String(),
			CompanyName: settings.Name, CompanyORN: settings.ORN, CompanyLicence: settings.TradeLicenseNo,
			CompanyPhone: settings.BusinessPhone, CompanyEmail: settings.BusinessEmail, CompanyAddress: settings.BusinessAddress,
			CompanyLogoURL: settings.LogoURL,
		}
		if l.RentPeriod != nil {
			it.RentPeriod = *l.RentPeriod
		}
		if l.Furnishing != nil {
			it.Furnishing = *l.Furnishing
		}
		if l.CoverImageURL != "" {
			it.Photos = append(it.Photos, l.CoverImageURL)
		}
		it.Photos = append(it.Photos, l.ImageURLs...)
		if l.AssignedTo != nil {
			if a, ok := agents[*l.AssignedTo]; ok {
				it.AgentName, it.AgentEmail, it.AgentPhone, it.AgentBRN = a.Name, a.Email, a.WANumber, a.BRN
			}
		}
		if it.AgentName == "" {
			it.AgentName, it.AgentEmail, it.AgentPhone = settings.Name, settings.BusinessEmail, settings.BusinessPhone
		}
		items = append(items, it)
	}
	var buf bytes.Buffer
	if err := portal.WriteXML(&buf, items, now); err != nil {
		return serverError(c, err)
	}
	c.Set("Content-Type", "application/xml; charset=utf-8")
	c.Set("Cache-Control", "private, max-age=300")
	return c.Send(buf.Bytes())
}

// LeadSources godoc
// @Summary      Leads, wins and revenue by source
// @Tags         Reports
// @Produce      json
// @Param        from  query  string  false  "YYYY-MM-DD (default: 90 days ago)"
// @Param        to    query  string  false  "YYYY-MM-DD, exclusive (default: tomorrow)"
// @Success      200  {array}  repo.LeadSourceRow
// @Security     BearerAuth
// @Router       /reports/lead-sources [get]
func (h *PortalHandler) LeadSources(c *fiber.Ctx) error {
	to := time.Now().AddDate(0, 0, 1).Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -91)
	for _, q := range []struct {
		key string
		dst *time.Time
	}{{"from", &from}, {"to", &to}} {
		if s := c.Query(q.key); s != "" {
			d, err := time.Parse("2006-01-02", s)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": q.key + " must be YYYY-MM-DD"})
			}
			*q.dst = d
		}
	}
	rows, err := h.portals.LeadSourceReport(c.Context(), from, to)
	if err != nil {
		return serverError(c, err)
	}
	if rows == nil {
		rows = []repo.LeadSourceRow{}
	}
	return c.JSON(fiber.Map{"from": from.Format("2006-01-02"), "to": to.Format("2006-01-02"), "rows": rows})
}
