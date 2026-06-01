package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/email"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	users  *repo.UserRepo
	audit  *repo.AuditLogRepo
	email  *email.Service
	config *config.Config
}

func NewUserHandler(users *repo.UserRepo, audit *repo.AuditLogRepo, emailSvc *email.Service, cfg *config.Config) *UserHandler {
	return &UserHandler{users: users, audit: audit, email: emailSvc, config: cfg}
}

// GetMe godoc
// @Summary      Get current user
// @Description  Returns the authenticated user's profile.
// @Tags         Users
// @Produce      json
// @Success      200  {object}  object{id=string,name=string,email=string,role=string,lang_pref=string}
// @Security     BearerAuth
// @Router       /users/me [get]
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	claims := middleware.ClaimsFromCtx(c)
	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	user, err := h.users.FindByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(fiber.Map{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"role":      user.Role,
		"lang_pref": user.LangPref,
		"is_active": user.IsActive,
	})
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Change the authenticated user's password. Requires current password for verification.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body  body  object{current_password=string,new_password=string}  true  "Passwords"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      401  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /users/me/password [patch]
func (h *UserHandler) ChangePassword(c *fiber.Ctx) error {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if body.CurrentPassword == "" || body.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "current_password and new_password are required"})
	}
	if msg := validatePasswordStrength(body.NewPassword); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
	}

	claims := middleware.ClaimsFromCtx(c)
	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	user, err := h.users.FindByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "current password is incorrect"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	if err := h.users.UpdatePassword(c.Context(), userID, string(hash)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update password"})
	}

	h.audit.Log(c.Context(), userID, repo.AuditPasswordChange, repo.AuditUser, userID, nil)
	return c.SendStatus(fiber.StatusNoContent)
}

// UpdateLang godoc
// @Summary      Update language preference
// @Description  Update the authenticated user's language preference (ar or en).
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body  body  object{lang=string}  true  "Language"
// @Success      200   {object}  object{lang_pref=string}
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /users/me/lang [patch]
func (h *UserHandler) UpdateLang(c *fiber.Ctx) error {
	var body struct {
		Lang string `json:"lang"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if body.Lang != "ar" && body.Lang != "en" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "lang must be 'ar' or 'en'"})
	}

	claims := middleware.ClaimsFromCtx(c)
	userID, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	if err := h.users.UpdateLangPref(c.Context(), userID, body.Lang); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update language"})
	}

	return c.JSON(fiber.Map{"lang_pref": body.Lang})
}

// ListUsers godoc
// @Summary      List users
// @Description  Returns all users in the company. Admin only.
// @Tags         Users
// @Produce      json
// @Success      200  {array}   object{id=string,name=string,email=string,role=string,is_active=bool}
// @Security     BearerAuth
// @Router       /users [get]
func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	claims := middleware.ClaimsFromCtx(c)
	companyIDStr, _ := claims["company_id"].(string)
	companyID, _ := uuid.Parse(companyIDStr)

	users, _, err := h.users.ListByCompany(c.Context(), companyID, 1, 1000)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch users"})
	}

	out := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		out = append(out, fiber.Map{
			"id":         u.ID,
			"name":       u.Name,
			"email":      u.Email,
			"role":       u.Role,
			"lang_pref":  u.LangPref,
			"wa_number":  u.WANumber,
			"is_active":  u.IsActive,
			"created_at": u.CreatedAt,
		})
	}
	return c.JSON(out)
}

// CreateUser godoc
// @Summary      Create user
// @Description  Create a new user account. Admin only. Password must be 8+ chars with uppercase and digit.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body  body  object{name=string,email=string,password=string,role=string,lang_pref=string}  true  "User data"
// @Success      201   {object}  object{id=string,name=string,email=string,role=string}
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /users [post]
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var body struct {
		Name     string      `json:"name"`
		Email    string      `json:"email"`
		Password string      `json:"password"`
		Role     domain.Role `json:"role"`
		LangPref string      `json:"lang_pref"`
		WANumber string      `json:"wa_number"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if strings.TrimSpace(body.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	if strings.TrimSpace(body.Email) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
	}
	if msg := validatePasswordStrength(body.Password); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
	}

	validRoles := map[domain.Role]bool{
		domain.RoleAdmin:  true,
		domain.RoleAgent:  true,
		domain.RoleViewer: true,
	}
	if !validRoles[body.Role] {
		body.Role = domain.RoleAgent
	}
	if body.LangPref != "ar" && body.LangPref != "en" {
		body.LangPref = "ar"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	claims := middleware.ClaimsFromCtx(c)
	companyIDStr, _ := claims["company_id"].(string)
	companyID, _ := uuid.Parse(companyIDStr)

	user := &domain.User{
		Name:         body.Name,
		CompanyID:    companyID,
		Email:        body.Email,
		PasswordHash: string(hash),
		Role:         body.Role,
		LangPref:     body.LangPref,
		WANumber:     body.WANumber,
	}
	if err := h.users.Create(c.Context(), user); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already in use"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"role":      user.Role,
		"lang_pref": user.LangPref,
	})
}

// InviteUser godoc
// @Summary      Invite user
// @Description  Create a new user and send them a setup link via email. Admin only.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body  body  object{name=string,email=string,role=string}  true  "Invite data"
// @Success      201   {object}  object{id=string,name=string,email=string,role=string}
// @Failure      400   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /users/invite [post]
func (h *UserHandler) InviteUser(c *fiber.Ctx) error {
	var body struct {
		Name  string      `json:"name"`
		Email string      `json:"email"`
		Role  domain.Role `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if strings.TrimSpace(body.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}
	if strings.TrimSpace(body.Email) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
	}

	validRoles := map[domain.Role]bool{
		domain.RoleAdmin: true, domain.RoleAgent: true, domain.RoleViewer: true,
	}
	if !validRoles[body.Role] {
		body.Role = domain.RoleAgent
	}

	// Get company_id from JWT claims
	claims := middleware.ClaimsFromCtx(c)
	companyIDStr, _ := claims["company_id"].(string)
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
	}

	// Create user with empty password — they must set it via the invite link
	user := &domain.User{
		Name:         strings.TrimSpace(body.Name),
		CompanyID:    companyID,
		Email:        strings.ToLower(strings.TrimSpace(body.Email)),
		PasswordHash: "",
		Role:         body.Role,
		LangPref:     "ar",
		IsActive:     true,
	}
	if err := h.users.Create(c.Context(), user); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already in use"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}

	// Generate a setup token (reuses the password reset flow)
	token, err := h.users.CreatePasswordResetToken(c.Context(), user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate invite token"})
	}

	setupURL := fmt.Sprintf("%s/reset-password?token=%s", h.config.AppURL, token)
	if h.email != nil && h.email.IsConfigured() {
		_ = h.email.Send(&domain.EmailHistory{
			ToEmail:   user.Email,
			Subject:   "You've been invited to Masaar CRM",
			Body:      fmt.Sprintf("Hello %s,\n\nYou have been invited to Masaar CRM. Set your password via the link below (valid 1 hour):\n\n%s\n\nIf you weren't expecting this, you can ignore this email.", user.Name, setupURL),
			RelatedTo: "user_invite",
		})
	} else {
		fmt.Printf("[DEV] invite link for %s: %s\n", user.Email, setupURL)
	}

	callerID, _ := uuid.Parse(claims["sub"].(string))
	h.audit.Log(c.Context(), callerID, repo.AuditCreate, repo.AuditUser, user.ID,
		fiber.Map{"email": user.Email, "role": body.Role, "method": "invite"})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	})
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Update a user's name and role. Admin only. Cannot edit own account.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id    path  string                              true  "User ID"
// @Param        body  body  object{name=string,role=string}     true  "Update data"
// @Success      204
// @Failure      400  {object}  object{error=string}
// @Failure      403  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /users/{id} [patch]
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	callerID, _ := uuid.Parse(middleware.ClaimsFromCtx(c)["sub"].(string))
	if targetID == callerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot edit your own account via this endpoint"})
	}

	var body struct {
		Name string      `json:"name"`
		Role domain.Role `json:"role"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if strings.TrimSpace(body.Name) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name is required"})
	}

	validRoles := map[domain.Role]bool{
		domain.RoleAdmin: true, domain.RoleAgent: true, domain.RoleViewer: true,
	}
	if !validRoles[body.Role] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid role"})
	}

	if err := h.users.UpdateUser(c.Context(), targetID, body.Name, body.Role); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update user"})
	}

	h.audit.Log(c.Context(), callerID, repo.AuditUpdate, repo.AuditUser, targetID,
		fiber.Map{"name": body.Name, "role": body.Role})
	return c.SendStatus(fiber.StatusNoContent)
}

// SetActive godoc
// @Summary      Set user active status
// @Description  Activate or deactivate a user. Admin only. Cannot deactivate own account.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id    path  string                  true  "User ID"
// @Param        body  body  object{active=bool}     true  "Active status"
// @Success      200   {object}  object{is_active=bool}
// @Failure      403   {object}  object{error=string}
// @Security     BearerAuth
// @Router       /users/{id}/active [patch]
func (h *UserHandler) SetActive(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	callerID, _ := uuid.Parse(middleware.ClaimsFromCtx(c)["sub"].(string))
	if targetID == callerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot deactivate yourself"})
	}

	var body struct {
		Active bool `json:"active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if err := h.users.SetActive(c.Context(), targetID, body.Active); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update status"})
	}

	h.audit.Log(c.Context(), callerID, repo.AuditUpdate, repo.AuditUser, targetID,
		fiber.Map{"is_active": body.Active})
	return c.JSON(fiber.Map{"is_active": body.Active})
}

// DeleteUser godoc
// @Summary      Delete user
// @Description  Permanently delete a user account. Admin only. Cannot delete own account.
// @Tags         Users
// @Produce      json
// @Param        id  path  string  true  "User ID"
// @Success      204
// @Failure      403  {object}  object{error=string}
// @Security     BearerAuth
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	callerID, _ := uuid.Parse(middleware.ClaimsFromCtx(c)["sub"].(string))
	if targetID == callerID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "cannot delete yourself"})
	}

	if err := h.users.Delete(c.Context(), targetID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete user"})
	}

	h.audit.Log(c.Context(), callerID, repo.AuditDelete, repo.AuditUser, targetID, nil)
	return c.SendStatus(fiber.StatusNoContent)
}
