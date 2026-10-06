package handler

import (
	"bytes"
	"errors"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/mediastore"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type BankStatementHandler struct {
	statements *repo.BankStatementRepo
	store      *mediastore.Store // nil when the storage directory could not be opened
	maxBytes   int64
}

// NewBankStatementHandler returns the handler. store may be nil (uploads then answer 503);
// maxMB caps one uploaded file.
func NewBankStatementHandler(statements *repo.BankStatementRepo, store *mediastore.Store, maxMB int) *BankStatementHandler {
	if maxMB <= 0 {
		maxMB = 10
	}
	return &BankStatementHandler{statements: statements, store: store, maxBytes: int64(maxMB) << 20}
}

// List godoc
// @Summary      List bank statements
// @Description  Returns a paginated list of bank statements uploaded by the company.
// @Tags         Bank Statements
// @Produce      json
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Page size 1-100 (default 20)"
// @Success      200    {object}  domain.PaginatedResult[domain.BankStatement]
// @Security     BearerAuth
// @Router       /bank-statements [get]
func (h *BankStatementHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	result, err := h.statements.ListByCompany(c.Context(), companyID, page, limit)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get bank statement
// @Description  Returns a single bank statement by UUID.
// @Tags         Bank Statements
// @Produce      json
// @Param        id  path      string  true  "Statement UUID"
// @Success      200  {object}  domain.BankStatement
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /bank-statements/{id} [get]
func (h *BankStatementHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	statement, err := h.statements.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "statement not found"})
	}
	return c.JSON(statement)
}

// Upload godoc
// @Summary      Upload bank statement
// @Description  Uploads a bank statement file (CSV, PDF, or XLSX).
// @Tags         Bank Statements
// @Accept       multipart/form-data
// @Produce      json
// @Param        file                  formData  file    true  "Bank statement file"
// @Param        bank_integration_id   formData  string  true  "Bank integration UUID"
// @Success      201   {object}  domain.BankStatement
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /bank-statements/upload [post]
func (h *BankStatementHandler) Upload(c *fiber.Ctx) error {
	if h.store == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "file storage is not available"})
	}
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file required"})
	}

	bankIntegrationID := c.FormValue("bank_integration_id")
	if bankIntegrationID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bank_integration_id required"})
	}

	integrationID, err := uuid.Parse(bankIntegrationID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid bank_integration_id"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}

	// Always sanitize user-provided filenames to prevent path traversal and XSS
	sanitizedFilename := filepath.Base(file.Filename)
	fileExt := strings.ToLower(filepath.Ext(sanitizedFilename))
	var format domain.FileFormat
	switch fileExt {
	case ".csv":
		format = domain.FormatCSV
	case ".pdf":
		format = domain.FormatPDF
	case ".xlsx":
		format = domain.FormatXLSX
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unsupported file format"})
	}
	if file.Size > h.maxBytes {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file is too large"})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "cannot open file"})
	}
	defer f.Close()

	// The extension is client-chosen: check the bytes really are that format before keeping them.
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	if !contentMatchesFormat(format, head) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "file content does not match its extension"})
	}

	key, size, err := h.store.Save(io.MultiReader(bytes.NewReader(head), f), h.maxBytes)
	if err != nil {
		if errors.Is(err, mediastore.ErrTooLarge) {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file is too large"})
		}
		return serverError(c, err)
	}

	statement := &domain.BankStatement{
		CompanyID:          companyID,
		BankIntegrationID:  integrationID,
		FileName:           sanitizedFilename,
		FileSizeBytes:      int(size),
		StorageKey:         key,
		FileFormat:         format,
		UploadedBy:         userID,
		UploadDate:         time.Now(),
		ProcessingStatus:   domain.StatusPending,
		DataClassification: domain.ClassConfidential,
	}

	if err := h.statements.Create(c.Context(), statement); err != nil {
		h.removeFile(key) // do not leave an unreferenced confidential file behind
		return serverError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(statement)
}

// contentMatchesFormat checks the leading bytes of an upload against the format its extension
// claims. CSV has no signature, so it must at least be text (no NUL bytes, valid UTF-8 apart
// from a possibly truncated trailing rune).
func contentMatchesFormat(format domain.FileFormat, head []byte) bool {
	switch format {
	case domain.FormatPDF:
		return bytes.HasPrefix(head, []byte("%PDF-"))
	case domain.FormatXLSX:
		return bytes.HasPrefix(head, []byte("PK\x03\x04"))
	case domain.FormatCSV:
		if len(head) == 0 || bytes.IndexByte(head, 0) >= 0 {
			return false
		}
		for i := 0; i < utf8.UTFMax && i < len(head); i++ {
			if utf8.Valid(head[:len(head)-i]) {
				return true
			}
		}
		return false
	}
	return false
}

func (h *BankStatementHandler) removeFile(key string) {
	if h.store == nil || key == "" {
		return
	}
	if err := h.store.Remove(key); err != nil {
		log.Printf("bank statements: remove stored file: %v", err)
	}
}

// Download godoc
// @Summary      Download bank statement file
// @Description  Streams the originally uploaded file as an attachment.
// @Tags         Bank Statements
// @Param        id  path  string  true  "Statement UUID"
// @Success      200
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /bank-statements/{id}/download [get]
func (h *BankStatementHandler) Download(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	statement, err := h.statements.GetByID(c.Context(), id)
	if err != nil || statement.StorageKey == "" || h.store == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
	}
	f, err := h.store.Open(statement.StorageKey)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "file not found"})
		}
		return serverError(c, err)
	}

	contentType := map[domain.FileFormat]string{
		domain.FormatCSV:  "text/csv; charset=utf-8",
		domain.FormatPDF:  "application/pdf",
		domain.FormatXLSX: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}[statement.FileFormat]
	if contentType == "" {
		contentType = fiber.MIMEOctetStream
	}
	c.Set(fiber.HeaderContentType, contentType)
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Set(fiber.HeaderCacheControl, "private, no-store")
	c.Set(fiber.HeaderContentDisposition, mime.FormatMediaType("attachment", map[string]string{"filename": safeFilename(statement.FileName)}))
	return c.SendStream(f, statement.FileSizeBytes)
}

// Delete godoc
// @Summary      Delete bank statement
// @Description  Deletes a bank statement by UUID. The record is kept for the audit trail (soft delete); the stored file is removed.
// @Tags         Bank Statements
// @Param        id  path  string  true  "Statement UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /bank-statements/{id} [delete]
func (h *BankStatementHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	// Look up the stored file first: GetByID is company-scoped, so this also refuses
	// another company's id before anything is deleted.
	statement, err := h.statements.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "statement not found"})
	}
	if err := h.statements.Delete(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	h.removeFile(statement.StorageKey) // the row stays for the audit trail; the confidential file does not
	return c.SendStatus(fiber.StatusNoContent)
}
