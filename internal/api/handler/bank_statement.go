package handler

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type BankStatementHandler struct {
	statements *repo.BankStatementRepo
}

func NewBankStatementHandler(statements *repo.BankStatementRepo) *BankStatementHandler {
	return &BankStatementHandler{statements: statements}
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
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

	fileURL := fmt.Sprintf("/uploads/bank-statements/%s-%s", companyID.String()[:8], sanitizedFilename)

	statement := &domain.BankStatement{
		CompanyID:          companyID,
		BankIntegrationID:  integrationID,
		FileName:           sanitizedFilename,
		FileSizeBytes:      int(file.Size),
		FileURL:            fileURL,
		FileFormat:         format,
		UploadedBy:         userID,
		ProcessingStatus:   domain.StatusPending,
		DataClassification: domain.ClassConfidential,
	}

	if err := h.statements.Create(c.Context(), statement); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(statement)
}

// Delete godoc
// @Summary      Delete bank statement
// @Description  Deletes a bank statement by UUID (soft delete for audit trail).
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

	if err := h.statements.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
