package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/maidulcu/masaar-crm/internal/apiscope"
	"github.com/maidulcu/masaar-crm/internal/billing"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/session"
)

// Partner API (/partner/v1): a partner system that bundles the CRM with its own plans creates a
// company per customer, keeps its users and seats in step, gets an API key for pushing leads
// and opens the CRM for a user with a one-click login link. Authenticated with an OAuth 2.0
// client-credentials token; a partner only sees the companies it created.

// partnerRefRe is a partner's own id for a company or user. It sits in a URL path, so no "/".
var partnerRefRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

const loginLinkTTL = 2 * time.Minute

type PartnerHandler struct {
	partners *repo.PartnerRepo
	keys     *repo.ApiKeyRepo
	audit    *repo.AuditLogRepo
	redis    *redis.Client
	loginURL string // MAGIC_LINK_BASE_URL
}

func NewPartnerHandler(partners *repo.PartnerRepo, keys *repo.ApiKeyRepo, audit *repo.AuditLogRepo, rdb *redis.Client, magicLinkBaseURL string) *PartnerHandler {
	return &PartnerHandler{partners: partners, keys: keys, audit: audit, redis: rdb, loginURL: strings.TrimRight(magicLinkBaseURL, "/")}
}

func partnerClient(c *fiber.Ctx) string { s, _ := c.Locals("partner_client_id").(string); return s }

// company loads the partner's company named in the path.
func (h *PartnerHandler) company(c *fiber.Ctx) (*repo.PartnerCompany, error) {
	ref := c.Params("ref")
	if !partnerRefRe.MatchString(ref) {
		return nil, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company reference"})
	}
	pc, err := h.partners.GetCompany(c.Context(), partnerClient(c), ref)
	if errors.Is(err, repo.ErrPartnerNotFound) {
		return nil, c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "company not found"})
	}
	if err != nil {
		return nil, serverError(c, err)
	}
	return pc, nil
}

type partnerUserBody struct {
	Ref      string `json:"ref"` // only inside "admin"; a user's ref is in the path
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	LangPref string `json:"lang_pref"`
	Active   *bool  `json:"active"`
}

// validateUser normalises a user body. creating requires a name and an email; on an update
// both may be left out (the caller keeps the current values).
func validateUser(b *partnerUserBody, creating bool) string {
	b.Name = strings.TrimSpace(b.Name)
	b.Email = strings.ToLower(strings.TrimSpace(b.Email))
	if (creating && b.Name == "") || utf8.RuneCountInString(b.Name) > maxNameRunes {
		return "name is required (at most 100 characters)"
	}
	if creating || b.Email != "" {
		if addr, err := mail.ParseAddress(b.Email); err != nil || addr.Address != b.Email || len(b.Email) > maxEmailLen {
			return "a valid email is required"
		}
	}
	if b.LangPref == "" && creating {
		b.LangPref = "en"
	}
	if b.LangPref != "" && b.LangPref != "en" && b.LangPref != "ar" {
		return "lang_pref must be en or ar"
	}
	return ""
}

func randomPasswordHash() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Partner users sign in through login links; nobody knows this password.
	h, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(b)), bcrypt.DefaultCost)
	return string(h), err
}

// GetCompany godoc
// @Summary      Get a provisioned company
// @Tags         Partner
// @Produce      json
// @Param        ref  path  string  true  "Your reference for the company"
// @Success      200  {object}  repo.PartnerCompany
// @Failure      404  {object}  object{error=string}
// @Security     PartnerAuth
// @Router       /partner/v1/companies/{ref} [get]
func (h *PartnerHandler) GetCompany(c *fiber.Ctx) error {
	pc, err := h.company(c)
	if pc == nil {
		return err
	}
	return c.JSON(pc)
}

// PutCompany godoc
// @Summary      Create or update a company
// @Description  Creates the company with your reference, its first admin and an API key for POST /webhooks/leads (returned once, on creation only), or updates name, plan, seats and active status. Setting active to false suspends the company: its users and keys are refused until it is reactivated.
// @Tags         Partner
// @Accept       json
// @Produce      json
// @Param        ref   path  string  true  "Your reference for the company"
// @Param        body  body  object{name=string,plan=string,max_users=int,active=bool,admin=object{ref=string,name=string,email=string,lang_pref=string}}  true  "admin is required on creation and ignored on updates"
// @Success      200  {object}  object{company=repo.PartnerCompany,warnings=[]string}
// @Success      201  {object}  object{company=repo.PartnerCompany,admin=repo.PartnerUser,lead_api_key=string}
// @Failure      409  {object}  object{error=string}
// @Failure      422  {object}  object{error=string}
// @Security     PartnerAuth
// @Router       /partner/v1/companies/{ref} [put]
func (h *PartnerHandler) PutCompany(c *fiber.Ctx) error {
	client, ref := partnerClient(c), c.Params("ref")
	if !partnerRefRe.MatchString(ref) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company reference"})
	}
	var body struct {
		Name     *string          `json:"name"`
		Plan     *string          `json:"plan"`
		MaxUsers *int             `json:"max_users"`
		Active   *bool            `json:"active"`
		Admin    *partnerUserBody `json:"admin"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	var keys map[string]json.RawMessage
	_ = json.Unmarshal(c.Body(), &keys)
	clearMax := false
	if raw, ok := keys["max_users"]; ok && string(raw) == "null" {
		clearMax = true
	}
	if body.Name != nil {
		n := strings.TrimSpace(*body.Name)
		if n == "" || utf8.RuneCountInString(n) > maxCompanyRunes {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "name must be 1-150 characters"})
		}
		body.Name = &n
	}
	if body.Plan != nil && !knownPlan(*body.Plan) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "unknown plan"})
	}
	if body.MaxUsers != nil && (*body.MaxUsers < 1 || *body.MaxUsers > 10000) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "max_users must be 1-10000 or null"})
	}

	existing, err := h.partners.GetCompany(c.Context(), client, ref)
	switch {
	case err == nil:
		pc, err := h.partners.UpdateCompany(c.Context(), client, ref, repo.PartnerCompanyUpdate{
			Name: body.Name, Plan: body.Plan, MaxUsers: body.MaxUsers, ClearMaxUsers: clearMax, Active: body.Active,
		})
		if err != nil {
			return serverError(c, err)
		}
		h.audit.LogPartner(c.Context(), client, existing.ID, repo.AuditUpdate, repo.AuditCompany, existing.ID, json.RawMessage(c.Body()))
		resp := fiber.Map{"company": pc}
		// Accepted, but say so instead of letting the partner's records drift silently.
		var warnings []string
		if body.Admin != nil {
			warnings = append(warnings, "admin is only used when the company is created; manage users with PUT .../users/{user_ref}")
		}
		if pc.MaxUsers != nil && pc.ActiveUsers > *pc.MaxUsers {
			warnings = append(warnings, fmt.Sprintf("max_users (%d) is below the %d active users: no user can be added or reactivated until enough are deactivated", *pc.MaxUsers, pc.ActiveUsers))
		}
		if len(warnings) > 0 {
			resp["warnings"] = warnings
		}
		return c.JSON(resp)
	case !errors.Is(err, repo.ErrPartnerNotFound):
		return serverError(c, err)
	}

	// Creation.
	if body.Name == nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "name is required to create a company"})
	}
	if body.Admin == nil || !partnerRefRe.MatchString(body.Admin.Ref) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "admin with a valid ref is required to create a company"})
	}
	if msg := validateUser(body.Admin, true); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "admin: " + msg})
	}
	plan := billing.PlanPro
	if body.Plan != nil {
		plan = *body.Plan
	}
	active := body.Active == nil || *body.Active
	hash, err := randomPasswordHash()
	if err != nil {
		return serverError(c, err)
	}
	key, keyHash, keyPrefix, err := h.keys.GenerateKey()
	if err != nil {
		return serverError(c, err)
	}
	pc, admin, err := h.partners.CreateCompany(c.Context(), repo.PartnerCompanyInput{
		ClientID: client, Ref: ref, Name: *body.Name, Plan: plan, MaxUsers: body.MaxUsers, Active: active,
		Admin: repo.PartnerUserInput{Ref: body.Admin.Ref, Name: body.Admin.Name, Email: body.Admin.Email,
			LangPref: body.Admin.LangPref, PasswordHash: hash},
		LeadKeyName: leadKeyName(client), LeadKeyHash: keyHash, LeadKeyPrefix: keyPrefix, LeadKeyScopes: apiscope.LeadCreate,
	})
	switch {
	case errors.Is(err, repo.ErrEmailTaken):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email_taken", "detail": "the admin's email already has an account"})
	case errors.Is(err, repo.ErrCompanyExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "conflict", "detail": "the company was created by a concurrent request; retry to update it"})
	case errors.Is(err, repo.ErrSeatLimit):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "seat_limit"})
	case err != nil:
		return serverError(c, err)
	}
	c.Set(fiber.HeaderCacheControl, "no-store") // carries a secret: never replayed from storage
	h.audit.LogPartner(c.Context(), client, pc.ID, repo.AuditCreate, repo.AuditCompany, pc.ID,
		fiber.Map{"ref": ref, "name": pc.Name, "plan": pc.Plan, "max_users": pc.MaxUsers, "admin": admin.Email})
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"company": pc, "admin": admin, "lead_api_key": key})
}

func knownPlan(id string) bool {
	for _, p := range billing.Plans {
		if p.ID == id {
			return true
		}
	}
	return false
}

func leadKeyName(client string) string {
	return fmt.Sprintf("%s lead intake %s", client, time.Now().UTC().Format("2006-01-02 15:04:05.000"))
}

// newLeadKey creates an API key with lead:create for the company and returns its plaintext.
func (h *PartnerHandler) newLeadKey(ctx context.Context, companyID uuid.UUID, client string) (string, error) {
	plain, hash, prefix, err := h.keys.GenerateKey()
	if err != nil {
		return "", err
	}
	if _, err := h.keys.Create(ctx, companyID, leadKeyName(client), hash, prefix, apiscope.LeadCreate); err != nil {
		return "", err
	}
	return plain, nil
}

// PutUser godoc
// @Summary      Create or update a user
// @Description  Creates the user with your reference in the company, or updates name, role, language and active status. An update changes only the fields it sends; email is set on creation only (a different one is reported in warnings). Creating or reactivating a user fails with 409 seat_limit when the company has max_users active users.
// @Tags         Partner
// @Accept       json
// @Produce      json
// @Param        ref       path  string  true  "Your reference for the company"
// @Param        user_ref  path  string  true  "Your reference for the user"
// @Param        body      body  object{name=string,email=string,role=string,lang_pref=string,active=bool}  true  "name and email are required on creation; role admin, agent (default) or viewer"
// @Success      200  {object}  object{user=repo.PartnerUser,warnings=[]string}
// @Success      201  {object}  object{user=repo.PartnerUser}
// @Failure      409  {object}  object{error=string}
// @Failure      422  {object}  object{error=string}
// @Security     PartnerAuth
// @Router       /partner/v1/companies/{ref}/users/{user_ref} [put]
func (h *PartnerHandler) PutUser(c *fiber.Ctx) error {
	pc, err := h.company(c)
	if pc == nil {
		return err
	}
	userRef := c.Params("user_ref")
	if !partnerRefRe.MatchString(userRef) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user reference"})
	}
	var body partnerUserBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	prev, getErr := h.partners.GetUser(c.Context(), pc.ID, userRef)
	creating := errors.Is(getErr, repo.ErrPartnerNotFound)
	if getErr != nil && !creating {
		return serverError(c, getErr)
	}
	if msg := validateUser(&body, creating); msg != "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
	}
	role := domain.Role(body.Role)
	switch role {
	case "":
		role = domain.RoleAgent
	case domain.RoleAdmin, domain.RoleAgent, domain.RoleViewer:
	default:
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "role must be admin, agent or viewer"})
	}
	in := repo.PartnerUserInput{Ref: userRef, Name: body.Name, Email: body.Email, LangPref: body.LangPref,
		Role: role, Active: body.Active == nil || *body.Active}
	var warnings []string
	if !creating {
		// An update changes only what was sent: a body with just a new name must not demote an
		// admin, reset the language or reactivate a deactivated user.
		if in.Name == "" {
			in.Name = prev.Name
		}
		if body.Role == "" {
			in.Role = prev.Role
		}
		if in.LangPref == "" {
			in.LangPref = prev.LangPref
		}
		if body.Active == nil {
			in.Active = prev.Active
		}
		if body.Email != "" && body.Email != prev.Email {
			warnings = append(warnings, "email is not changed by an update; the user keeps "+prev.Email)
		}
	}
	if creating {
		if in.PasswordHash, err = randomPasswordHash(); err != nil {
			return serverError(c, err)
		}
	}
	user, created, err := h.partners.UpsertUser(c.Context(), pc.ID, in)
	switch {
	case errors.Is(err, repo.ErrSeatLimit):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "seat_limit", "detail": "all of the company's seats (max_users) are in use"})
	case errors.Is(err, repo.ErrEmailTaken):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email_taken", "detail": "this email already has an account"})
	case err != nil:
		return serverError(c, err)
	}
	action, status := repo.AuditUpdate, fiber.StatusOK
	if created {
		action, status = repo.AuditCreate, fiber.StatusCreated
	} else if prev != nil && (prev.Role != user.Role || (prev.Active && !user.Active)) && h.redis != nil {
		// The role is embedded in session tokens: end them so the change applies now.
		if err := session.RevokeAll(c.Context(), h.redis, user.ID); err != nil {
			log.Printf("partner: revoke sessions for %s: %v", user.ID, err)
		}
	}
	h.audit.LogPartner(c.Context(), partnerClient(c), pc.ID, action, repo.AuditUser, user.ID,
		fiber.Map{"ref": userRef, "role": user.Role, "active": user.Active})
	resp := fiber.Map{"user": user}
	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}
	return c.Status(status).JSON(resp)
}

// CreateAPIKey godoc
// @Summary      Create an API key for a company
// @Description  A new key with lead:create for POST /webhooks/leads, shown once. Older keys keep working until the company's admin revokes them.
// @Tags         Partner
// @Produce      json
// @Param        ref  path  string  true  "Your reference for the company"
// @Success      201  {object}  object{lead_api_key=string}
// @Security     PartnerAuth
// @Router       /partner/v1/companies/{ref}/api-keys [post]
func (h *PartnerHandler) CreateAPIKey(c *fiber.Ctx) error {
	pc, err := h.company(c)
	if pc == nil {
		return err
	}
	key, err := h.newLeadKey(c.Context(), pc.ID, partnerClient(c))
	if err != nil {
		return serverError(c, err)
	}
	h.audit.LogPartner(c.Context(), partnerClient(c), pc.ID, repo.AuditCreate, repo.AuditAPIKey, pc.ID, fiber.Map{"scopes": apiscope.LeadCreate})
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"lead_api_key": key})
}

// LoginLink godoc
// @Summary      Create a one-click login link
// @Description  A single-use link that signs the user in to the CRM, valid two minutes. Open it in the user's browser right away (e.g. redirect to it); do not email it. Two-factor authentication still applies if the user turned it on.
// @Tags         Partner
// @Produce      json
// @Param        ref       path  string  true  "Your reference for the company"
// @Param        user_ref  path  string  true  "Your reference for the user"
// @Success      201  {object}  object{url=string,expires_in=int}
// @Failure      403  {object}  object{error=string}
// @Failure      404  {object}  object{error=string}
// @Security     PartnerAuth
// @Router       /partner/v1/companies/{ref}/users/{user_ref}/login-link [post]
func (h *PartnerHandler) LoginLink(c *fiber.Ctx) error {
	pc, err := h.company(c)
	if pc == nil {
		return err
	}
	if !pc.Active {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account_suspended"})
	}
	userRef := c.Params("user_ref")
	if !partnerRefRe.MatchString(userRef) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user reference"})
	}
	user, err := h.partners.GetUser(c.Context(), pc.ID, userRef)
	if errors.Is(err, repo.ErrPartnerNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	} else if err != nil {
		return serverError(c, err)
	}
	if !user.Active {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "user is deactivated"})
	}
	if h.redis == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "login links need Redis"})
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return serverError(c, err)
	}
	token := hex.EncodeToString(b)
	// Same single-use token as an emailed magic link: POST /auth/magic-link/verify redeems it.
	if err := h.redis.Set(c.Context(), "magic:"+token, magicLinkSubject(user.ID)+"|"+user.LangPref, loginLinkTTL).Err(); err != nil {
		return serverError(c, err)
	}
	h.audit.LogPartner(c.Context(), partnerClient(c), pc.ID, "login_link", repo.AuditUser, user.ID, nil)
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"url":        h.loginURL + "/login/magic-link/verify?token=" + token,
		"expires_in": int(loginLinkTTL.Seconds()),
	})
}
