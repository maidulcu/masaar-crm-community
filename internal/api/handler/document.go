package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/docusign"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type DocumentHandler struct {
	docs    *repo.DocumentRepo
	audit   *repo.AuditLogRepo
	dsClient *docusign.Client
}

func NewDocumentHandler(docs *repo.DocumentRepo, audit *repo.AuditLogRepo, dsClient *docusign.Client) *DocumentHandler {
	return &DocumentHandler{docs: docs, audit: audit, dsClient: dsClient}
}

// ListTemplates godoc
// @Summary      List document templates
// @Description  Returns templates for reusable document creation.
// @Tags         Documents
// @Produce      json
// @Param        page   query     int  false  "Page (default 1)"
// @Param        limit  query     int  false  "Page size (default 20)"
// @Success      200    {object}  domain.PaginatedResult[domain.DocumentTemplate]
// @Security     BearerAuth
// @Router       /documents/templates [get]
func (h *DocumentHandler) ListTemplates(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	companyID, _ := uuid.Parse(c.Locals("company_id").(string))
	result, err := h.docs.ListTemplates(c.Context(), companyID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(result)
}

// GetTemplate godoc
// @Summary      Get template
// @Description  Returns a single document template by ID.
// @Tags         Documents
// @Produce      json
// @Param        id  path      string  true  "Template UUID"
// @Success      200  {object}  domain.DocumentTemplate
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents/templates/{id} [get]
func (h *DocumentHandler) GetTemplate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	t, err := h.docs.GetTemplate(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "template not found"})
	}
	return c.JSON(t)
}

// CreateTemplate godoc
// @Summary      Create template
// @Description  Creates a reusable document template.
// @Tags         Documents
// @Accept       json
// @Produce      json
// @Param        body  body      domain.DocumentTemplate  true  "Template payload"
// @Success      201   {object}  domain.DocumentTemplate
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents/templates [post]
func (h *DocumentHandler) CreateTemplate(c *fiber.Ctx) error {
	var t domain.DocumentTemplate
	if err := c.BodyParser(&t); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if t.TemplateName == "" || t.DocumentType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "template_name and document_type required"})
	}

	t.CompanyID, _ = uuid.Parse(c.Locals("company_id").(string))
	t.CreatedBy = c.Locals("user_id").(uuid.UUID)

	if err := h.docs.CreateTemplate(c.Context(), &t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	h.audit.Log(c.Context(), t.CreatedBy, repo.AuditCreate, "document_template", t.ID, t)
	return c.Status(fiber.StatusCreated).JSON(t)
}

// UpdateTemplate godoc
// @Summary      Update template
// @Description  Updates template name and content.
// @Tags         Documents
// @Accept       json
// @Produce      json
// @Param        id    path      string                            true  "Template UUID"
// @Param        body  body      object{template_name=string,template_content=string}  true  "Update payload"
// @Success      200   {object}  object{id=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents/templates/{id} [patch]
func (h *DocumentHandler) UpdateTemplate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	var body struct {
		TemplateName    string `json:"template_name"`
		TemplateContent string `json:"template_content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.docs.UpdateTemplate(c.Context(), id, body.TemplateName, body.TemplateContent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	userID := c.Locals("user_id").(uuid.UUID)
	h.audit.Log(c.Context(), userID, repo.AuditUpdate, "document_template", id, body)
	return c.JSON(fiber.Map{"id": id})
}

// DeleteTemplate godoc
// @Summary      Delete template
// @Description  Deletes a document template.
// @Tags         Documents
// @Param        id  path      string  true  "Template UUID"
// @Success      204
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents/templates/{id} [delete]
func (h *DocumentHandler) DeleteTemplate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.docs.DeleteTemplate(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	userID := c.Locals("user_id").(uuid.UUID)
	h.audit.Log(c.Context(), userID, repo.AuditDelete, "document_template", id, nil)
	return c.SendStatus(fiber.StatusNoContent)
}

// CreateDocument godoc
// @Summary      Create document
// @Description  Creates a document, optionally from a template. File upload handled separately.
// @Tags         Documents
// @Accept       json
// @Produce      json
// @Param        body  body      domain.Document  true  "Document payload"
// @Success      201   {object}  domain.Document
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents [post]
func (h *DocumentHandler) CreateDocument(c *fiber.Ctx) error {
	var d domain.Document
	if err := c.BodyParser(&d); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if d.DocumentTitle == "" || d.DocumentType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "document_title and document_type required"})
	}

	d.CompanyID, _ = uuid.Parse(c.Locals("company_id").(string))
	d.CreatedBy = c.Locals("user_id").(uuid.UUID)
	d.SignatureStatus = domain.SignaturePending

	if err := h.docs.CreateDocument(c.Context(), &d); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	h.audit.Log(c.Context(), d.CreatedBy, repo.AuditCreate, repo.AuditDocument, d.ID, d)
	return c.Status(fiber.StatusCreated).JSON(d)
}

// GetDocument godoc
// @Summary      Get document
// @Description  Returns a document with its signatures.
// @Tags         Documents
// @Produce      json
// @Param        id  path      string  true  "Document UUID"
// @Success      200  {object}  object{document=domain.Document,signatures=[]domain.DocumentSignature}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents/{id} [get]
func (h *DocumentHandler) GetDocument(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	d, err := h.docs.GetDocument(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "document not found"})
	}
	sigs, _ := h.docs.GetSignatures(c.Context(), id)
	return c.JSON(fiber.Map{
		"document":   d,
		"signatures": sigs,
	})
}

// ListDocuments godoc
// @Summary      List documents
// @Description  Lists documents attached to an entity (deal, lease, etc).
// @Tags         Documents
// @Produce      json
// @Param        entity_type    query     string  true  "Entity type (deal, lease, etc)"
// @Param        entity_id      query     string  true  "Entity UUID"
// @Success      200   {array}   domain.Document
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents [get]
func (h *DocumentHandler) ListDocuments(c *fiber.Ctx) error {
	entityType := c.Query("entity_type")
	entityIDStr := c.Query("entity_id")
	if entityType == "" || entityIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "entity_type and entity_id required"})
	}
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid entity_id"})
	}

	docs, err := h.docs.ListDocumentsByEntity(c.Context(), entityType, entityID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(docs)
}

// SendForSignature godoc
// @Summary      Send document for signature
// @Description  Creates a signature request for a signer. Set use_docusign=true to send via DocuSign API.
// @Tags         Documents
// @Accept       json
// @Produce      json
// @Param        id    path      string                                                              true  "Document UUID"
// @Param        body  body      object{signer_name=string,signer_email=string,use_docusign=bool}  true  "Signer info"
// @Success      201   {object}  domain.DocumentSignature
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents/{id}/request-signature [post]
func (h *DocumentHandler) SendForSignature(c *fiber.Ctx) error {
	docID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	d, err := h.docs.GetDocument(c.Context(), docID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "document not found"})
	}

	var body struct {
		SignerName  string `json:"signer_name"`
		SignerEmail string `json:"signer_email"`
		UseDocusign bool   `json:"use_docusign"`
	}
	if err := c.BodyParser(&body); err != nil || body.SignerName == "" || body.SignerEmail == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "signer_name and signer_email required"})
	}

	sig := &domain.DocumentSignature{
		DocumentID:         docID,
		SignerName:         body.SignerName,
		SignerEmail:        body.SignerEmail,
		SignatureFieldName: "signature_1",
		SignatureStatus:    domain.SignaturePending,
		IPAddress:          c.IP(),
		UserAgent:          c.Get("User-Agent"),
	}

	// DocuSign flow
	if body.UseDocusign && h.dsClient != nil && h.dsClient.Enabled() {
		if d.FileURL != "" {
			docContent, fetchErr := fetchDocumentContent(d.FileURL)
			if fetchErr == nil {
				envResult, dsErr := h.dsClient.SendEnvelope(c.Context(), docContent, d.DocumentTitle,
					docusign.Signer{Name: body.SignerName, Email: body.SignerEmail})
				if dsErr == nil {
					sig.EnvelopeID = envResult.EnvelopeID
				}
			}
		}
	}

	if err := h.docs.CreateSignature(c.Context(), sig); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	h.audit.Log(c.Context(), c.Locals("user_id").(uuid.UUID), "signature_request", repo.AuditDocument, docID, sig)

	if d.SignatureStatus != domain.SignatureSigned {
		h.docs.UpdateSignatureStatus(c.Context(), docID, domain.SignaturePending)
	}

	return c.Status(fiber.StatusCreated).JSON(sig)
}

// fetchDocumentContent retrieves document bytes from a URL or file path.
func fetchDocumentContent(url string) ([]byte, error) {
	if url == "" {
		return nil, fiber.ErrBadRequest
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// MarkSigned godoc
// @Summary      Mark signature as signed
// @Description  Records signature completion.
// @Tags         Documents
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Signature UUID"
// @Success      200   {object}  object{id=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents/signatures/{id}/mark-signed [patch]
func (h *DocumentHandler) MarkSigned(c *fiber.Ctx) error {
	sigID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.docs.MarkSigned(c.Context(), sigID, time.Now()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"id": sigID})
}

// PublicGetSignature godoc
// @Summary      Get signature details (public)
// @Description  Returns the document and signature details for a given signature ID. No auth required.
// @Tags         Public
// @Produce      json
// @Param        id  path      string  true  "Signature UUID"
// @Success      200  {object}  object{signature=domain.DocumentSignature,document=domain.Document}
// @Failure      404  {object}  object{error=string}
// @Router       /public/sign/{id} [get]
func (h *DocumentHandler) PublicGetSignature(c *fiber.Ctx) error {
	sigID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	sigs, err := h.docs.GetSignatureByID(c.Context(), sigID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "signature not found"})
	}

	doc, err := h.docs.GetDocument(c.Context(), sigs.DocumentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "document not found"})
	}

	return c.JSON(fiber.Map{"signature": sigs, "document": doc})
}

// PublicSign godoc
// @Summary      Sign a document (public)
// @Description  Marks a signature as signed. No auth required — the signature UUID is the access token.
// @Tags         Public
// @Produce      json
// @Param        id  path      string  true  "Signature UUID"
// @Success      200  {object}  object{status=string}
// @Failure      400  {object}  object{error=string}
// @Router       /public/sign/{id} [post]
func (h *DocumentHandler) PublicSign(c *fiber.Ctx) error {
	sigID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.docs.MarkSigned(c.Context(), sigID, time.Now()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "signed"})
}

// DeleteDocument godoc
// @Summary      Delete document
// @Description  Soft-deletes a document (not permanently removed).
// @Tags         Documents
// @Param        id  path      string  true  "Document UUID"
// @Success      204
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /documents/{id} [delete]
func (h *DocumentHandler) DeleteDocument(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.docs.SoftDeleteDocument(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	userID := c.Locals("user_id").(uuid.UUID)
	h.audit.Log(c.Context(), userID, repo.AuditDelete, repo.AuditDocument, id, nil)
	return c.SendStatus(fiber.StatusNoContent)
}
