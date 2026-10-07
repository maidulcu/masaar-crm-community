package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type BankIntegrationHandler struct {
	integrations *repo.BankIntegrationRepo
}

func NewBankIntegrationHandler(integrations *repo.BankIntegrationRepo) *BankIntegrationHandler {
	return &BankIntegrationHandler{integrations: integrations}
}

// List godoc
// @Summary      List bank integrations
// @Description  Returns a paginated list of bank integrations for the company.
// @Tags         Bank Integrations
// @Produce      json
// @Param        page   query     int  false  "Page number (default 1)"
// @Param        limit  query     int  false  "Page size 1-100 (default 20)"
// @Success      200    {object}  domain.PaginatedResult[domain.BankIntegration]
// @Security     BearerAuth
// @Router       /bank-integrations [get]
func (h *BankIntegrationHandler) List(c *fiber.Ctx) error {
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

	result, err := h.integrations.List(c.Context(), companyID, page, limit)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(result)
}

// Get godoc
// @Summary      Get bank integration
// @Description  Returns a single bank integration by UUID.
// @Tags         Bank Integrations
// @Produce      json
// @Param        id  path      string  true  "Integration UUID"
// @Success      200  {object}  domain.BankIntegration
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /bank-integrations/{id} [get]
func (h *BankIntegrationHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	integration, err := h.integrations.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "integration not found"})
	}
	return c.JSON(integration)
}

// Create godoc
// @Summary      Create bank integration
// @Description  Creates a new bank integration. bank_name, account_number, and integration_type are required.
// @Tags         Bank Integrations
// @Accept       json
// @Produce      json
// @Param        body  body      domain.BankIntegration  true  "Integration payload"
// @Success      201   {object}  domain.BankIntegration
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /bank-integrations [post]
func (h *BankIntegrationHandler) Create(c *fiber.Ctx) error {
	var in domain.BankIntegration
	if err := c.BodyParser(&in); err != nil {
		return badRequest(c, err)
	}
	// Connection and sync state is maintained by the sync job, not the client.
	bi := domain.BankIntegration{
		BankName: in.BankName, BankCode: in.BankCode, AccountNumber: in.AccountNumber, AccountName: in.AccountName,
		IBAN: in.IBAN, IntegrationType: in.IntegrationType, Status: in.Status, APIEndpoint: in.APIEndpoint,
		AutoSync: in.AutoSync, SyncIntervalHours: in.SyncIntervalHours,
	}
	if err := validateBankIntegration(&bi); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	bi.CreatedBy = &userID
	bi.UpdatedBy = &userID

	companyID, err := uuid.Parse(c.Locals("company_id").(string))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company_id"})
	}
	bi.CompanyID = companyID

	if err := h.integrations.Create(c.Context(), &bi); err != nil {
		return serverError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(bi)
}

// Update godoc
// @Summary      Update bank integration
// @Description  Updates an existing bank integration.
// @Tags         Bank Integrations
// @Accept       json
// @Produce      json
// @Param        id    path      string                  true  "Integration UUID"
// @Param        body  body      domain.BankIntegration  true  "Integration payload"
// @Success      200   {object}  domain.BankIntegration
// @Failure      400   {object}  object{error=string}
// @Failure      404   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /bank-integrations/{id} [patch]
func (h *BankIntegrationHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	bi, err := h.integrations.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "integration not found"})
	}

	// An id in the body would redirect the update to another integration, and the connection and
	// sync state belongs to the sync job.
	keep := *bi
	if err := c.BodyParser(bi); err != nil {
		return badRequest(c, err)
	}
	bi.ID, bi.CompanyID, bi.CreatedAt, bi.CreatedBy = keep.ID, keep.CompanyID, keep.CreatedAt, keep.CreatedBy
	bi.APIKeyEncrypted, bi.APISecretEncrypted = keep.APIKeyEncrypted, keep.APISecretEncrypted
	bi.IsConnected, bi.ConnectionTestDate, bi.LastSyncDate = keep.IsConnected, keep.ConnectionTestDate, keep.LastSyncDate
	bi.LastSyncError, bi.SyncErrorCount = keep.LastSyncError, keep.SyncErrorCount
	if err := validateBankIntegration(bi); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	bi.UpdatedBy = &userID

	if err := h.integrations.Update(c.Context(), bi); err != nil {
		return serverError(c, err)
	}
	bi.APIKeyEncrypted, bi.APISecretEncrypted = "", "" // never leave the process (they are json:"-" anyway)
	return c.JSON(bi)
}

// Delete godoc
// @Summary      Delete bank integration
// @Description  Deletes a bank integration by UUID.
// @Tags         Bank Integrations
// @Param        id  path  string  true  "Integration UUID"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /bank-integrations/{id} [delete]
func (h *BankIntegrationHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.integrations.Delete(c.Context(), id); err != nil {
		if isForeignKeyViolation(err) || errors.Is(err, repo.ErrBankIntegrationInUse) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "this integration has statements or transactions and cannot be deleted; mark it inactive instead"})
		}
		return serverError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
