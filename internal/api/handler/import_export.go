package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

// ImportExportHandler handles CSV bulk import and export for contacts, leads, listings.
//
//	POST /api/v1/import/contacts           — upload CSV, upsert contacts
//	POST /api/v1/import/leads              — upload CSV, create leads
//	GET  /api/v1/export/contacts           — download contacts CSV
//	GET  /api/v1/export/leads              — download leads CSV
//	GET  /api/v1/export/listings           — download listings CSV
//	GET  /api/v1/import/template/:entity   — download empty CSV template
type ImportExportHandler struct {
	contactRepo *repo.ContactRepo
	leadRepo    *repo.LeadRepo
	listingRepo *repo.ListingRepo
}

func NewImportExportHandler(
	contactRepo *repo.ContactRepo,
	leadRepo *repo.LeadRepo,
	listingRepo *repo.ListingRepo,
) *ImportExportHandler {
	return &ImportExportHandler{
		contactRepo: contactRepo,
		leadRepo:    leadRepo,
		listingRepo: listingRepo,
	}
}

// ── Templates ─────────────────────────────────────────────────────────────────

// Template handles GET /api/v1/import/template/:entity
// Returns a blank CSV with the correct headers for the given entity type.
func (h *ImportExportHandler) Template(c *fiber.Ctx) error {
	entity := c.Params("entity")
	var headers []string
	switch entity {
	case "contacts":
		headers = []string{"full_name", "phone_wa", "email", "language", "notes"}
	case "leads":
		headers = []string{"contact_phone", "stage", "source", "deal_value", "currency", "notes"}
	case "listings":
		headers = []string{"title", "property_type", "listing_type", "price", "currency", "area", "city", "emirate", "bedrooms", "bathrooms", "total_sqft", "description", "cover_image_url", "reference_number"}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "entity must be contacts, leads, or listings"})
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(headers)
	w.Flush()

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="template-%s.csv"`, entity))
	return c.Send(buf.Bytes())
}

// ── Import ────────────────────────────────────────────────────────────────────

// ImportContacts handles POST /api/v1/import/contacts (multipart file upload).
// Max 500 rows per request.
func (h *ImportExportHandler) ImportContacts(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required (multipart field: file)"})
	}
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot open file"})
	}
	defer f.Close()

	records, err := parseCSV(f)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid CSV: " + err.Error()})
	}
	if len(records) < 2 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "CSV must have a header row and at least one data row"})
	}
	if len(records) > 501 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "maximum 500 rows per import"})
	}

	header := normaliseHeader(records[0])
	var imported, skipped int
	var errors []map[string]interface{}

	for i, row := range records[1:] {
		rowNum := i + 2
		m := rowToMap(header, row)

		phone := strings.TrimSpace(m["phone_wa"])
		name := strings.TrimSpace(m["full_name"])
		if phone == "" || name == "" {
			errors = append(errors, fiber.Map{"row": rowNum, "error": "phone_wa and full_name are required"})
			skipped++
			continue
		}

		contact, err := h.contactRepo.Upsert(c.Context(), phone, name)
		if err != nil {
			errors = append(errors, fiber.Map{"row": rowNum, "error": err.Error()})
			skipped++
			continue
		}

		// Update optional fields
		if email := strings.TrimSpace(m["email"]); email != "" && contact.Email == "" {
			contact.Email = email
			_ = h.contactRepo.Update(c.Context(), contact)
		}
		imported++
	}

	return c.JSON(fiber.Map{
		"ok":       true,
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
	})
}

// ImportLeads handles POST /api/v1/import/leads.
// Looks up contact by phone_wa, creates lead for each row.
func (h *ImportExportHandler) ImportLeads(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot open file"})
	}
	defer f.Close()

	records, err := parseCSV(f)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid CSV: " + err.Error()})
	}
	if len(records) < 2 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "CSV must have header + data rows"})
	}
	if len(records) > 501 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "maximum 500 rows per import"})
	}

	header := normaliseHeader(records[0])
	validStages := map[string]bool{"new": true, "contacted": true, "qualified": true, "proposal": true, "won": true, "lost": true}
	validSources := map[string]bool{"whatsapp": true, "web": true, "referral": true, "event": true, "bos24": true, "": true}

	var imported, skipped int
	var errors []map[string]interface{}

	for i, row := range records[1:] {
		rowNum := i + 2
		m := rowToMap(header, row)

		phone := strings.TrimSpace(m["contact_phone"])
		if phone == "" {
			errors = append(errors, fiber.Map{"row": rowNum, "error": "contact_phone is required"})
			skipped++
			continue
		}

		// Upsert contact
		contact, err := h.contactRepo.Upsert(c.Context(), phone, phone) // name = phone as fallback
		if err != nil {
			errors = append(errors, fiber.Map{"row": rowNum, "error": "contact lookup failed: " + err.Error()})
			skipped++
			continue
		}

		stage := strings.ToLower(strings.TrimSpace(m["stage"]))
		if stage == "" {
			stage = "new"
		}
		if !validStages[stage] {
			errors = append(errors, fiber.Map{"row": rowNum, "error": "invalid stage: " + stage})
			skipped++
			continue
		}

		source := strings.ToLower(strings.TrimSpace(m["source"]))
		if !validSources[source] {
			source = "web"
		}

		var dealValue float64
		if v := strings.TrimSpace(m["deal_value"]); v != "" {
			dealValue, _ = strconv.ParseFloat(v, 64)
		}
		currency := strings.TrimSpace(m["currency"])
		if currency == "" {
			currency = "AED"
		}

		lead := &domain.Lead{
			ContactID: contact.ID,
			Stage:     domain.LeadStage(stage),
			Source:    domain.LeadSource(source),
			DealValue: dealValue,
			Currency:  currency,
			Notes:     strings.TrimSpace(m["notes"]),
		}
		if err := h.leadRepo.Create(c.Context(), lead); err != nil {
			errors = append(errors, fiber.Map{"row": rowNum, "error": err.Error()})
			skipped++
			continue
		}
		imported++
	}

	return c.JSON(fiber.Map{
		"ok":       true,
		"imported": imported,
		"skipped":  skipped,
		"errors":   errors,
	})
}

// ── Export ────────────────────────────────────────────────────────────────────

// ExportContacts handles GET /api/v1/export/contacts — streams contacts as CSV.
func (h *ImportExportHandler) ExportContacts(c *fiber.Ctx) error {
	search := c.Query("search", "")
	result, err := h.contactRepo.List(c.Context(), search, 1, 10000)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "full_name", "phone_wa", "email", "language", "lead_score", "created_at"})
	for _, ct := range result.Data {
		_ = w.Write([]string{
			ct.ID.String(), ct.FullName, ct.PhoneWA, ct.Email,
			ct.Language, strconv.Itoa(ct.LeadScore), ct.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="contacts-%s.csv"`, time.Now().Format("2006-01-02")))
	return c.Send(buf.Bytes())
}

// ExportLeads handles GET /api/v1/export/leads — streams leads as CSV.
func (h *ImportExportHandler) ExportLeads(c *fiber.Ctx) error {
	stage := c.Query("stage", "")
	source := c.Query("source", "")

	leads, err := h.leadRepo.List(c.Context(), repo.LeadFilter{
		Stage:  domain.LeadStage(stage),
		Source: source,
		Limit:  10000,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "contact_name", "contact_phone", "stage", "source", "deal_value", "currency", "notes", "created_at"})
	for _, l := range leads {
		contactName, contactPhone := "", ""
		if l.Contact != nil {
			contactName = l.Contact.FullName
			contactPhone = l.Contact.PhoneWA
		}
		_ = w.Write([]string{
			l.ID.String(), contactName, contactPhone,
			string(l.Stage), string(l.Source),
			strconv.FormatFloat(l.DealValue, 'f', 2, 64),
			l.Currency, l.Notes, l.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="leads-%s.csv"`, time.Now().Format("2006-01-02")))
	return c.Send(buf.Bytes())
}

// ExportListings handles GET /api/v1/export/listings — streams listings as CSV.
func (h *ImportExportHandler) ExportListings(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	result, err := h.listingRepo.List(c.Context(), companyID, 1, 10000)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{
		"id", "title", "property_type", "listing_type", "status",
		"price", "currency", "bedrooms", "bathrooms", "total_sqft",
		"area", "community", "city", "emirate", "reference_number",
		"cover_image_url", "created_at",
	})
	for _, l := range result.Data {
		_ = w.Write([]string{
			l.ID.String(), l.Title, string(l.PropertyType), string(l.ListingType), string(l.Status),
			strconv.FormatFloat(l.Price, 'f', 2, 64), l.Currency,
			strconv.Itoa(l.Bedrooms), strconv.Itoa(l.Bathrooms),
			strconv.FormatFloat(l.TotalSqft, 'f', 0, 64),
			l.Area, l.Community, l.City, l.Emirate, l.ReferenceNumber,
			l.CoverImageURL, l.CreatedAt.Format(time.RFC3339),
		})
	}
	w.Flush()

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="listings-%s.csv"`, time.Now().Format("2006-01-02")))
	return c.Send(buf.Bytes())
}

// ── CSV helpers ───────────────────────────────────────────────────────────────

func parseCSV(r io.Reader) ([][]string, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1 // allow variable columns
	return cr.ReadAll()
}

func normaliseHeader(row []string) []string {
	out := make([]string, len(row))
	for i, h := range row {
		out[i] = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(h, " ", "_")))
	}
	return out
}

func rowToMap(header, row []string) map[string]string {
	m := make(map[string]string, len(header))
	for i, h := range header {
		if i < len(row) {
			m[h] = strings.TrimSpace(row[i])
		}
	}
	return m
}
