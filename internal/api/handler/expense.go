package handler

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
)

type ExpenseHandler struct {
	expenseRepo *repo.ExpenseRepository
}

func NewExpenseHandler(expenseRepo *repo.ExpenseRepository) *ExpenseHandler {
	return &ExpenseHandler{
		expenseRepo: expenseRepo,
	}
}

// Categories
// @Summary List expense categories
// @Tags Expenses
// @Security Bearer
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/expense-categories [get]
func (h *ExpenseHandler) ListCategories(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	categories, err := h.expenseRepo.ListCategories(c.Context(), companyID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch categories"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": categories})
}

// @Summary Create expense category
// @Tags Expenses
// @Security Bearer
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/expense-categories [post]
func (h *ExpenseHandler) CreateCategory(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	var req struct {
		CategoryName string `json:"category_name"`
		CategoryType string `json:"category_type"`
		Description  string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return badRequest(c, err)
	}

	cat := &domain.ExpenseCategory{
		ID:           uuid.New(),
		CompanyID:    companyID,
		CategoryName: req.CategoryName,
		CategoryType: domain.ExpenseCategoryType(req.CategoryType),
		Description:  req.Description,
	}

	if err := validateExpenseCategory(cat); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.expenseRepo.CreateCategory(c.Context(), cat); err != nil {
		if isUniqueViolation(err) {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "a category with this name already exists"})
		}
		return serverError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": cat})
}

// Expenses
// @Summary List expenses
// @Tags Expenses
// @Security Bearer
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/expenses [get]
func (h *ExpenseHandler) ListExpenses(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	expenses, total, err := h.expenseRepo.ListExpenses(c.Context(), companyID, limit, offset)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch expenses"})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"data": expenses,
		"meta": fiber.Map{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// @Summary Get expense details
// @Tags Expenses
// @Security Bearer
// @Param id path string true "Expense ID"
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/expenses/{id} [get]
func (h *ExpenseHandler) GetExpense(c *fiber.Ctx) error {
	expenseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid expense ID"})
	}

	expense, err := h.expenseRepo.GetExpense(c.Context(), expenseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Expense not found"})
		}
		return serverError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": expense})
}

// @Summary Create expense
// @Tags Expenses
// @Security Bearer
// @Accept json
// @Produce json
// @Success 201 {object} map[string]interface{}
// @Router /api/v1/expenses [post]
func (h *ExpenseHandler) CreateExpense(c *fiber.Ctx) error {
	companyID, _ := uuid.Parse(c.Locals("company_id").(string))
	userID := c.Locals("user_id").(uuid.UUID)

	// The expense form sends a date-only expense_date and "" for "no property", neither of which
	// time.Time / *uuid.UUID accept: creating an expense from the web app always answered 400.
	var req struct {
		CategoryID    uuid.UUID  `json:"category_id"`
		PropertyID    *uuid.UUID `json:"property_id"`
		TenantID      *uuid.UUID `json:"tenant_id"`
		Amount        float64    `json:"amount"`
		ExpenseDate   time.Time  `json:"expense_date"`
		Description   string     `json:"description"`
		VendorName    string     `json:"vendor_name"`
		VendorContact string     `json:"vendor_contact"`
		PaymentMethod string     `json:"payment_method"`
		ReceiptURL    string     `json:"receipt_url"`
		Notes         string     `json:"notes"`
	}
	if err := json.Unmarshal(normalizeDateFields(c.Body(), "expense_date", "property_id", "tenant_id"), &req); err != nil {
		return badRequest(c, err)
	}
	if req.CategoryID == uuid.Nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "category_id is required"})
	}

	expense := &domain.Expense{
		ID:            uuid.New(),
		CompanyID:     companyID,
		CategoryID:    req.CategoryID,
		PropertyID:    req.PropertyID,
		TenantID:      req.TenantID,
		Amount:        math.Round(req.Amount*100) / 100,
		Currency:      "AED",
		ExpenseDate:   req.ExpenseDate,
		Description:   req.Description,
		VendorName:    req.VendorName,
		VendorContact: req.VendorContact,
		PaymentMethod: domain.ExpensePaymentMethod(req.PaymentMethod),
		PaymentStatus: domain.ExpensePaymentPending,
		ReceiptURL:    req.ReceiptURL,
		Notes:         req.Notes,
		CreatedBy:     userID,
	}
	if err := validateExpense(expense); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.expenseRepo.CreateExpense(c.Context(), expense); err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // the INSERT only happens when the references are this company's
			return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"error": "category, property or tenant not found"})
		}
		return serverError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": expense})
}

// @Summary Update expense
// @Tags Expenses
// @Security Bearer
// @Param id path string true "Expense ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/expenses/{id} [patch]
func (h *ExpenseHandler) UpdateExpense(c *fiber.Ctx) error {
	expenseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid expense ID"})
	}

	expense, err := h.expenseRepo.GetExpense(c.Context(), expenseID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Expense not found"})
		}
		return serverError(c, err)
	}

	var req struct {
		CategoryID    *uuid.UUID `json:"category_id"`
		Amount        *float64   `json:"amount"`
		ExpenseDate   *time.Time `json:"expense_date"`
		Description   *string    `json:"description"`
		VendorName    *string    `json:"vendor_name"`
		VendorContact *string    `json:"vendor_contact"`
		PaymentMethod *string    `json:"payment_method"`
		PaymentStatus *string    `json:"payment_status"`
		ReceiptURL    *string    `json:"receipt_url"`
		Notes         *string    `json:"notes"`
	}
	if err := json.Unmarshal(normalizeDateFields(c.Body(), "expense_date"), &req); err != nil {
		return badRequest(c, err)
	}

	before := expense.Amount
	if req.CategoryID != nil && *req.CategoryID != uuid.Nil {
		expense.CategoryID = *req.CategoryID
	}
	if req.Amount != nil {
		expense.Amount = math.Round(*req.Amount*100) / 100
	}
	if req.ExpenseDate != nil {
		expense.ExpenseDate = *req.ExpenseDate
	}
	if req.Description != nil {
		expense.Description = *req.Description
	}
	if req.VendorName != nil {
		expense.VendorName = *req.VendorName
	}
	if req.VendorContact != nil {
		expense.VendorContact = *req.VendorContact
	}
	if req.PaymentMethod != nil {
		expense.PaymentMethod = domain.ExpensePaymentMethod(*req.PaymentMethod)
	}
	if req.PaymentStatus != nil {
		expense.PaymentStatus = domain.ExpensePaymentStatus(*req.PaymentStatus)
	}
	if req.ReceiptURL != nil {
		expense.ReceiptURL = *req.ReceiptURL
	}
	if req.Notes != nil {
		expense.Notes = *req.Notes
	}
	if err := validateExpense(expense); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	// An approved expense is what the approver saw: its amount can no longer change.
	if expense.Amount != before {
		if ap, err := h.expenseRepo.GetApprovalByExpense(c.Context(), expenseID); err == nil && ap.ApprovalStatus == domain.ExpenseApprovalApproved {
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "this expense has been approved; its amount can no longer be changed"})
		}
	}

	if err := h.expenseRepo.UpdateExpense(c.Context(), expense); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(http.StatusUnprocessableEntity).JSON(fiber.Map{"error": "category not found"})
		}
		return serverError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": expense})
}

// @Summary Delete expense
// @Tags Expenses
// @Security Bearer
// @Param id path string true "Expense ID"
// @Success 204
// @Router /api/v1/expenses/{id} [delete]
func (h *ExpenseHandler) DeleteExpense(c *fiber.Ctx) error {
	expenseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid expense ID"})
	}

	if err := h.expenseRepo.DeleteExpense(c.Context(), expenseID); err != nil {
		return serverError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}

// Approvals
// @Summary Approve expense
// @Tags Expenses
// @Security Bearer
// @Param id path string true "Expense ID"
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/expenses/{id}/approve [post]
func (h *ExpenseHandler) ApproveExpense(c *fiber.Ctx) error {
	expenseID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid expense ID"})
	}

	userID := c.Locals("user_id").(uuid.UUID)

	// The web client posts no body at all; an empty body used to be "invalid request body".
	var req struct {
		Comments string `json:"comments"`
	}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return badRequest(c, err)
		}
	}
	if tooLong(req.Comments, 5000) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "comments are too long"})
	}
	if _, err := h.expenseRepo.GetExpense(c.Context(), expenseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Expense not found"})
		}
		return serverError(c, err)
	}

	approval := &domain.ExpenseApproval{
		ID:               uuid.New(),
		ExpenseID:        expenseID,
		ApprovalStatus:   domain.ExpenseApprovalApproved,
		ApprovedBy:       &userID,
		ApprovalComments: req.Comments,
		ApprovalDate:     timePtr(time.Now()),
	}

	if err := h.expenseRepo.CreateApproval(c.Context(), approval); err != nil {
		if isUniqueViolation(err) { // expense_approvals.expense_id is unique
			return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "this expense has already been reviewed"})
		}
		return serverError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"data": approval})
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
