package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/email"
	"github.com/maidulcu/masaar-crm/internal/pdf"
	"github.com/maidulcu/masaar-crm/internal/repo"
	qrcode "github.com/skip2/go-qrcode"
)

// MarketingHandler serves listing marketing tools.
//
//	GET  /api/public/listings/:id           — public listing data (no auth)
//	GET  /api/v1/listings/:id/brochure      — download PDF brochure
//	POST /api/v1/listings/:id/email-campaign — send listing to contacts
type MarketingHandler struct {
	listingRepo     *repo.ListingRepo
	contactRepo     *repo.ContactRepo
	companySettings *repo.CompanySettingsRepo
	userRepo        *repo.UserRepo
	emailService    *email.Service
}

func NewMarketingHandler(
	listingRepo *repo.ListingRepo,
	contactRepo *repo.ContactRepo,
	companySettings *repo.CompanySettingsRepo,
	userRepo *repo.UserRepo,
	emailService *email.Service,
) *MarketingHandler {
	return &MarketingHandler{
		listingRepo:     listingRepo,
		contactRepo:     contactRepo,
		companySettings: companySettings,
		userRepo:        userRepo,
		emailService:    emailService,
	}
}

// PublicListing handles GET /api/public/listings/:id
// Returns listing data with no authentication — for shareable pages and OG previews.
func (h *MarketingHandler) PublicListing(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	listing, err := h.listingRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
	}
	// Only expose published listings publicly
	if listing.Status != domain.ListingStatusPublished {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not available"})
	}
	// Strip sensitive owner fields before returning publicly
	listing.OwnerEmail = ""
	listing.OwnerPhone = ""

	// Fetch company branding for the public page
	settings, _ := h.companySettings.Get(c.Context())
	resp := fiber.Map{"listing": listing}
	if settings != nil {
		resp["company"] = fiber.Map{
			"name":          settings.Name,
			"phone":         settings.BusinessPhone,
			"email":         settings.BusinessEmail,
			"logo_url":      settings.LogoURL,
			"primary_color": settings.PrimaryColor,
		}
	}
	return c.JSON(resp)
}

// DownloadBrochure handles GET /api/v1/listings/:id/brochure
// Generates a PDF brochure for the listing and streams it for download.
func (h *MarketingHandler) DownloadBrochure(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	listing, err := h.listingRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
	}

	settings, _ := h.companySettings.Get(c.Context())
	cs := &domain.CompanySettings{}
	if settings != nil {
		cs = settings
	}

	// Try to fetch agent details
	var agentName, agentPhone, agentEmail string
	if listing.AssignedTo != nil {
		if agent, err := h.userRepo.FindByID(c.Context(), *listing.AssignedTo); err == nil {
			agentName = agent.Name
			agentEmail = agent.Email
			agentPhone = agent.WANumber
		}
	}
	if agentName == "" {
		agentName = cs.Name
		agentPhone = cs.BusinessPhone
		agentEmail = cs.BusinessEmail
	}

	var yearBuilt int
	if listing.YearBuilt != nil {
		yearBuilt = *listing.YearBuilt
	}
	var furnishing string
	if listing.Furnishing != nil {
		furnishing = *listing.Furnishing
	}
	var rentPeriod string
	if listing.RentPeriod != nil {
		rentPeriod = *listing.RentPeriod
	}

	data := pdf.BrochureData{
		CompanyName:     cs.Name,
		CompanyAddress:  cs.BusinessAddress,
		CompanyPhone:    cs.BusinessPhone,
		CompanyEmail:    cs.BusinessEmail,
		LogoURL:         cs.LogoURL,
		PrimaryColor:    cs.PrimaryColor,
		Disclaimer:      cs.Disclaimer,
		AgentName:       agentName,
		AgentPhone:      agentPhone,
		AgentEmail:      agentEmail,
		ListingID:       listing.ID.String(),
		Title:           listing.Title,
		ReferenceNumber: listing.ReferenceNumber,
		PropertyType:    string(listing.PropertyType),
		ListingType:     string(listing.ListingType),
		Price:           listing.Price,
		Currency:        listing.Currency,
		RentPeriod:      rentPeriod,
		Bedrooms:        listing.Bedrooms,
		Bathrooms:       listing.Bathrooms,
		TotalSqft:       listing.TotalSqft,
		ParkingSpaces:   listing.ParkingSpaces,
		Furnishing:      furnishing,
		YearBuilt:       yearBuilt,
		Area:            listing.Area,
		Community:       listing.Community,
		City:            listing.City,
		Emirate:         listing.Emirate,
		Description:     listing.Description,
		Amenities:       listing.Amenities,
		CoverImageURL:   listing.CoverImageURL,
		AvailableFrom:   listing.AvailableFrom,
	}

	pdfBytes, err := pdf.GenerateBrochure(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate brochure"})
	}

	filename := fmt.Sprintf("brochure-%s.pdf", listing.ReferenceNumber)
	if listing.ReferenceNumber == "" {
		filename = fmt.Sprintf("brochure-%s.pdf", listing.ID.String()[:8])
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Send(pdfBytes)
}

// EmailCampaign handles POST /api/v1/listings/:id/email-campaign
// Sends a listing summary email to a list of contact IDs.
func (h *MarketingHandler) EmailCampaign(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var body struct {
		ContactIDs []string `json:"contact_ids"`
		Subject    string   `json:"subject"`
		Message    string   `json:"message"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if len(body.ContactIDs) == 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "at least one contact_id is required"})
	}
	if len(body.ContactIDs) > 100 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "maximum 100 contacts per campaign"})
	}

	listing, err := h.listingRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
	}

	settings, _ := h.companySettings.Get(c.Context())
	companyName := "Masaar CRM"
	if settings != nil {
		companyName = settings.Name
	}

	subject := body.Subject
	if subject == "" {
		subject = fmt.Sprintf("Property Listing: %s", listing.Title)
	}

	// Build a simple HTML email
	priceStr := fmt.Sprintf("%s %.0f", listing.Currency, listing.Price)
	if listing.RentPeriod != nil {
		priceStr += " / " + *listing.RentPeriod
	}
	location := listing.Area
	if listing.City != "" {
		location += ", " + listing.City
	}

	htmlBody := fmt.Sprintf(`
<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;color:#333">
  <div style="background:#1a3a5c;padding:20px;color:white">
    <h1 style="margin:0;font-size:20px">%s</h1>
    <p style="margin:4px 0 0;opacity:0.8">%s</p>
  </div>
  %s
  <div style="padding:20px">
    <h2 style="color:#1a3a5c;margin-top:0">%s</h2>
    <div style="background:#f5f7fa;border-radius:8px;padding:16px;margin-bottom:16px">
      <table style="width:100%%">
        <tr><td style="color:#666;padding:4px 0">Price</td><td style="font-weight:bold;color:#b48c3c">%s</td></tr>
        <tr><td style="color:#666;padding:4px 0">Type</td><td>%s · %s</td></tr>
        <tr><td style="color:#666;padding:4px 0">Size</td><td>%d Beds · %d Baths · %.0f sqft</td></tr>
        <tr><td style="color:#666;padding:4px 0">Location</td><td>%s</td></tr>
      </table>
    </div>
    <p style="color:#555;line-height:1.6">%s</p>
    %s
    <p style="color:#999;font-size:12px;border-top:1px solid #eee;padding-top:12px;margin-top:20px">
      Sent by %s | %s
    </p>
  </div>
</div>`,
		listing.Title, location,
		coverImg(listing.CoverImageURL),
		listing.Title,
		priceStr,
		listing.PropertyType, listing.ListingType,
		listing.Bedrooms, listing.Bathrooms, listing.TotalSqft,
		location,
		body.Message,
		refLink(listing.ReferenceNumber),
		companyName, settings.BusinessPhone,
	)

	sent, failed := 0, 0
	for _, cidStr := range body.ContactIDs {
		cid, err := uuid.Parse(cidStr)
		if err != nil {
			failed++
			continue
		}
		contact, err := h.contactRepo.GetByID(c.Context(), cid)
		if err != nil || contact.Email == "" {
			failed++
			continue
		}
		emailRecord := &domain.EmailHistory{
			ToEmail:  contact.Email,
			Subject:  subject,
			HTMLBody: htmlBody,
			Body:     fmt.Sprintf("Listing: %s\nPrice: %s\nLocation: %s", listing.Title, priceStr, location),
			RelatedTo: "listing",
		}
		if err := h.emailService.Send(emailRecord); err != nil {
			failed++
		} else {
			sent++
		}
	}

	return c.JSON(fiber.Map{
		"ok":     true,
		"sent":   sent,
		"failed": failed,
	})
}

// GenerateQR handles GET /api/v1/listings/:id/qr
// Returns a QR code PNG that points to the public listing page.
func (h *MarketingHandler) GenerateQR(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	listing, err := h.listingRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "listing not found"})
	}

	publicURL := fmt.Sprintf("%s/l/%s", c.BaseURL(), listing.ID.String())
	if listing.ReferenceNumber != "" {
		publicURL = fmt.Sprintf("%s/l/%s", c.BaseURL(), listing.ID.String())
	}

	png, err := qrcode.Encode(publicURL, qrcode.Medium, 512)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate QR code"})
	}

	filename := fmt.Sprintf("qr-%s.png", listing.ReferenceNumber)
	if listing.ReferenceNumber == "" {
		filename = fmt.Sprintf("qr-%s.png", listing.ID.String()[:8])
	}
	c.Set("Content-Type", "image/png")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Send(png)
}

func coverImg(url string) string {
	if url == "" {
		return ""
	}
	return fmt.Sprintf(`<img src="%s" style="width:100%%;max-height:300px;object-fit:cover;display:block" alt="Property" />`, url)
}

func refLink(ref string) string {
	if ref == "" {
		return ""
	}
	return fmt.Sprintf(`<p style="color:#666;font-size:13px">Reference: <strong>%s</strong></p>`, ref)
}
