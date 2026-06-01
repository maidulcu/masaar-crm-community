package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/maidulcu/masaar-crm/internal/api/middleware"
	"github.com/maidulcu/masaar-crm/internal/config"
	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/email"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/sms"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func validatePasswordStrength(p string) string {
	if len(p) < 8 {
		return "password must be at least 8 characters"
	}
	var hasUpper, hasDigit bool
	for _, r := range p {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasUpper {
		return "password must contain at least one uppercase letter"
	}
	if !hasDigit {
		return "password must contain at least one digit"
	}
	return ""
}

type AuthHandler struct {
	users       *repo.UserRepo
	companyRepo *repo.CompanyRepo
	redis       *redis.Client
	config      *config.Config
	audit       *repo.AuditLogRepo
	email       *email.Service
	sms         *sms.Client
}

func NewAuthHandler(users *repo.UserRepo, companyRepo *repo.CompanyRepo, rdb *redis.Client, cfg *config.Config, audit *repo.AuditLogRepo, emailSvc *email.Service, smsClient *sms.Client) *AuthHandler {
	return &AuthHandler{users: users, companyRepo: companyRepo, redis: rdb, config: cfg, audit: audit, email: emailSvc, sms: smsClient}
}

// Login godoc
// @Summary      Login
// @Description  Authenticate with email and password. Returns JWT access + refresh tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{email=string,password=string}  true  "Credentials"
// @Success      200   {object}  object{access_token=string,refresh_token=string,expires_in=int,user=object}
// @Failure      400   {object}  object{error=string}
// @Failure      401   {object}  object{error=string}
// @Failure      429   {object}  object{error=string}  "Too many login attempts"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	// Check lockout before touching the database
	lockKey := fmt.Sprintf("lockout:%s", body.Email)
	failKey := fmt.Sprintf("loginfail:%s", body.Email)
	ctx := context.Background()

	if locked, _ := h.redis.Exists(ctx, lockKey).Result(); locked > 0 {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "account temporarily locked due to too many failed attempts; try again in 15 minutes",
		})
	}

	user, err := h.users.FindByEmail(c.Context(), body.Email)
	if err != nil {
		// Increment failure counter even on unknown email to prevent enumeration timing
		h.redis.Incr(ctx, failKey)
		h.redis.Expire(ctx, failKey, 15*time.Minute)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		count, _ := h.redis.Incr(ctx, failKey).Result()
		h.redis.Expire(ctx, failKey, 15*time.Minute)
		if count >= 5 {
			h.redis.Set(ctx, lockKey, "1", 15*time.Minute)
			h.redis.Del(ctx, failKey)
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "account locked for 15 minutes after too many failed attempts",
			})
		}
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if !user.IsActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account is deactivated"})
	}

	// Load company and check active/trial
	var company *domain.Company
	var daysRemaining int
	if h.companyRepo != nil {
		comp, cerr := h.companyRepo.GetByID(c.Context(), user.CompanyID)
		if cerr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "company not found"})
		}
		company = comp
		if !company.IsActive {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account_suspended"})
		}
		if company.OnTrial && company.TrialEndsAt != nil {
			if company.TrialEndsAt.Before(time.Now()) {
				h.companyRepo.EndTrial(c.Context(), user.CompanyID)
				company.Plan = "community"
				company.OnTrial = false
			} else {
				daysRemaining = int(time.Until(*company.TrialEndsAt).Hours() / 24)
			}
		}
	}

	// Successful login — clear failure counter
	h.redis.Del(ctx, failKey, lockKey)

	access, refresh, err := h.generateTokenPair(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "token generation failed"})
	}

	// Store refresh token in Redis
	key := fmt.Sprintf("refresh:%s", refresh)
	ttl := time.Duration(h.config.JWTRefreshExpiryDays) * 24 * time.Hour
	if err := h.redis.Set(context.Background(), key, user.ID.String(), ttl).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session error"})
	}

	h.audit.Log(c.Context(), user.ID, repo.AuditLogin, repo.AuditUser, user.ID, fiber.Map{
		"ip": c.IP(),
	})

	resp := fiber.Map{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    h.config.JWTAccessExpiryMin * 60,
		"user": fiber.Map{
			"id":        user.ID,
			"name":      user.Name,
			"email":     user.Email,
			"role":      user.Role,
			"lang_pref": user.LangPref,
		},
	}
	if company != nil {
		resp["company"] = fiber.Map{
			"id":             company.ID,
			"name":           company.Name,
			"plan":           company.Plan,
			"on_trial":       company.OnTrial,
			"trial_ends_at":  company.TrialEndsAt,
			"days_remaining": daysRemaining,
			"is_demo":        company.IsDemo,
		}
	}
	return c.JSON(resp)
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange a valid refresh token for a new access token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{refresh_token=string}  true  "Refresh token"
// @Success      200   {object}  object{access_token=string,expires_in=int}
// @Failure      400   {object}  object{error=string}
// @Failure      401   {object}  object{error=string}
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err != nil || body.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "refresh_token required"})
	}

	key := fmt.Sprintf("refresh:%s", body.RefreshToken)
	ctx := context.Background()

	// Single-use: delete the old token atomically before issuing a new one.
	// If the same token is used twice (replay attack), the second call gets 401.
	userIDStr, err := h.redis.GetDel(ctx, key).Result()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired refresh token"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
	}

	// Verify user still exists and is active (admin may have deleted the account)
	user, err := h.users.FindByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "account not found or deactivated"})
	}

	// Issue a new token pair (rotated refresh token)
	access, newRefresh, err := h.generateTokenPair(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "token generation failed"})
	}

	// Store rotated refresh token
	ttl := time.Duration(h.config.JWTRefreshExpiryDays) * 24 * time.Hour
	if err := h.redis.Set(ctx, fmt.Sprintf("refresh:%s", newRefresh), user.ID.String(), ttl).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session error"})
	}

	return c.JSON(fiber.Map{
		"access_token":  access,
		"refresh_token": newRefresh,
		"expires_in":    h.config.JWTAccessExpiryMin * 60,
	})
}

// Logout godoc
// @Summary      Logout
// @Description  Invalidate the current session. Blacklists the access token and deletes the refresh token from Redis.
// @Tags         Auth
// @Accept       json
// @Param        body  body  object{refresh_token=string}  false  "Refresh token to invalidate"
// @Success      204
// @Security     BearerAuth
// @Router       /auth/logout [delete]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.BodyParser(&body)

	// Fail-closed on session invalidation
	if body.RefreshToken != "" {
		if err := h.redis.Del(context.Background(), fmt.Sprintf("refresh:%s", body.RefreshToken)).Err(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to revoke refresh token"})
		}
	}

	// Also invalidate current access token by bearer
	if token := middleware.BearerToken(c); token != "" {
		if err := h.redis.Set(context.Background(), "blacklist:"+token, "1", time.Duration(h.config.JWTAccessExpiryMin)*time.Minute).Err(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to revoke access token"})
		}
	}

	// Log logout — best effort, user ID from JWT claims
	if sub, ok := middleware.ClaimsFromCtx(c)["sub"].(string); ok {
		if userID, err := uuid.Parse(sub); err == nil {
			h.audit.Log(c.Context(), userID, repo.AuditLogout, repo.AuditUser, userID, nil)
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ForgotPassword godoc
// @Summary      Request password reset
// @Description  Sends a password-reset link to the registered email. Always returns 200 to prevent email enumeration.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  object{email=string}  true  "Registered email"
// @Success      200   {object}  object{message=string}
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := c.BodyParser(&body); err != nil || body.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
	}

	// Always return success — never reveal whether email exists
	const successMsg = "If that email is registered, a reset link has been sent"

	user, err := h.users.FindByEmail(c.Context(), body.Email)
	if err != nil {
		return c.JSON(fiber.Map{"message": successMsg})
	}

	token, err := h.users.CreatePasswordResetToken(c.Context(), user.ID)
	if err != nil {
		return c.JSON(fiber.Map{"message": successMsg})
	}

	// Send email if configured — log token to server output as fallback for dev
	if h.email != nil {
		resetURL := fmt.Sprintf("%s/reset-password?token=%s", h.config.AppURL, token)
		_ = h.email.Send(&domain.EmailHistory{
			ToEmail: user.Email,
			Subject: "Reset your Masaar CRM password",
			Body: fmt.Sprintf(
				"Click the link to reset your password (valid 1 hour):\n\n%s\n\nIf you did not request this, ignore this email.",
				resetURL,
			),
			RelatedTo: "password_reset",
		})
	} else {
		// Development fallback: log the token so it can be tested without SMTP
		fmt.Printf("[DEV] password reset token for %s: %s\n", user.Email, token)
	}

	return c.JSON(fiber.Map{"message": successMsg})
}

// ResetPassword godoc
// @Summary      Reset password using token
// @Description  Consumes a one-time reset token and sets a new password.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  object{token=string,new_password=string}  true  "Reset token and new password"
// @Success      204
// @Failure      400   {object}  object{error=string}
// @Failure      401   {object}  object{error=string}
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var body struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&body); err != nil || body.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token and new_password are required"})
	}
	if msg := validatePasswordStrength(body.NewPassword); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
	}

	userID, err := h.users.ConsumePasswordResetToken(c.Context(), body.Token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid, expired, or already-used reset token"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}
	if err := h.users.UpdatePassword(c.Context(), userID, string(hash)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update password"})
	}

	h.audit.Log(c.Context(), userID, repo.AuditPasswordChange, repo.AuditUser, userID, fiber.Map{
		"method": "password_reset",
	})
	return c.SendStatus(fiber.StatusNoContent)
}

// Register godoc
// @Summary      Register new company
// @Description  Creates a new company with a 90-day trial and an admin user. Rate-limited: 3 req/min/IP.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{name=string,email=string,password=string,company_name=string,subdomain=string,turnstile_token=string}  true  "Registration details"
// @Success      201   {object}  object{access_token=string,refresh_token=string,expires_in=int,user=object,company=object}
// @Failure      400   {object}  object{error=string}
// @Failure      409   {object}  object{error=string}
// @Failure      422   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	if !h.config.AllowRegistration {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "public registration is disabled on this instance",
		})
	}

	var body struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		Password      string `json:"password"`
		CompanyName   string `json:"company_name"`
		Subdomain     string `json:"subdomain"`
		TurnstileToken string `json:"turnstile_token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	// Validate Turnstile (if configured)
	if h.config.TurnstileSecretKey != "" && body.TurnstileToken == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "turnstile token required"})
	}
	if h.config.TurnstileSecretKey != "" {
		if err := verifyTurnstile(h.config.TurnstileSecretKey, body.TurnstileToken); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "captcha verification failed"})
		}
	}

	// Validate inputs
	if body.Name == "" || body.Email == "" || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name, email, and password are required"})
	}
	if body.CompanyName == "" || body.Subdomain == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "company_name and subdomain are required"})
	}
	if msg := validatePasswordStrength(body.Password); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
	}

	// Check email uniqueness
	exists, err := h.users.EmailExists(c.Context(), body.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email_taken"})
	}

	// Check subdomain uniqueness
	if h.companyRepo != nil {
		existing, _ := h.companyRepo.GetBySubdomain(c.Context(), body.Subdomain)
		if existing != nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "subdomain_taken"})
		}
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to hash password"})
	}

	// Create company with trial
	company, err := h.companyRepo.Create(c.Context(), body.CompanyName, body.Subdomain, h.config.TrialDurationDays)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create company"})
	}

	// Create admin user
	user, err := h.users.CreateWithCompany(c.Context(), body.Name, body.Email, string(hash), company.ID, domain.RoleAdmin)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}

	// Audit log
	h.audit.Log(c.Context(), user.ID, repo.AuditCreate, repo.AuditUser, user.ID, fiber.Map{
		"action":      "registration",
		"company_id":  company.ID.String(),
		"company_name": company.Name,
	})

	// Generate JWT
	access, refresh, err := h.generateTokenPair(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "token generation failed"})
	}

	// Store refresh token
	ttl := time.Duration(h.config.JWTRefreshExpiryDays) * 24 * time.Hour
	h.redis.Set(context.Background(), fmt.Sprintf("refresh:%s", refresh), user.ID.String(), ttl)

	daysRemaining := 0
	if company.TrialEndsAt != nil {
		daysRemaining = int(time.Until(*company.TrialEndsAt).Hours() / 24)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    h.config.JWTAccessExpiryMin * 60,
		"user": fiber.Map{
			"id":        user.ID,
			"name":      user.Name,
			"email":     user.Email,
			"role":      user.Role,
			"lang_pref": user.LangPref,
		},
		"company": fiber.Map{
			"id":             company.ID,
			"name":           company.Name,
			"subdomain":      company.Subdomain,
			"plan":           company.Plan,
			"on_trial":       company.OnTrial,
			"trial_ends_at":  company.TrialEndsAt,
			"days_remaining": daysRemaining,
			"is_demo":        company.IsDemo,
		},
	})
}

func (h *AuthHandler) generateTokenPair(user *domain.User) (access, refresh string, err error) {
	now := time.Now()

	// Load company details for JWT claims
	plan := "community"
	onTrial := false
	isDemo := false
	if h.companyRepo != nil {
		company, cerr := h.companyRepo.GetByID(context.Background(), user.CompanyID)
		if cerr == nil {
			plan = company.Plan
			onTrial = company.OnTrial
			isDemo = company.IsDemo
		}
	}

	accessClaims := jwt.MapClaims{
		"sub":        user.ID.String(),
		"name":       user.Name,
		"role":       string(user.Role),
		"company_id": user.CompanyID.String(),
		"plan":       plan,
		"on_trial":   onTrial,
		"is_demo":    isDemo,
		"exp":        now.Add(time.Duration(h.config.JWTAccessExpiryMin) * time.Minute).Unix(),
		"iat":        now.Unix(),
	}
	access, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(h.config.JWTSecret))
	if err != nil {
		return
	}

	refreshClaims := jwt.MapClaims{
		"sub": user.ID.String(),
		"exp": now.Add(time.Duration(h.config.JWTRefreshExpiryDays) * 24 * time.Hour).Unix(),
		"iat": now.Unix(),
		"jti": uuid.New().String(),
	}
	refresh, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(h.config.JWTSecret))
	return
}

// ─── Magic Link Login ─────────────────────────────────────────────────────

// RequestMagicLink godoc
// @Summary      Request Magic Link
// @Description  Send a magic link to the user's email for passwordless login. Rate limited to 3 requests per email per hour.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{email=string,lang_pref=string}  true  "Email and preferred language (ar/en)"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  object{error=string}
// @Failure      429   {object}  object{error=string}  "Rate limit exceeded"
// @Router       /auth/magic-link/request [post]
func (h *AuthHandler) RequestMagicLink(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		LangPref string `json:"lang_pref"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if body.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email required"})
	}

	if body.LangPref == "" {
		body.LangPref = "en"
	}

	ctx := context.Background()

	// Rate limiting: 3 requests per email per hour
	rateKey := fmt.Sprintf("magic_rate:%s", body.Email)
	count, err := h.redis.Get(ctx, rateKey).Int()
	if err != nil && err != redis.Nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	if count >= h.config.MagicLinkRateLimit {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "too many requests, please try again later",
		})
	}

	// Increment rate limit counter
	h.redis.Incr(ctx, rateKey)
	if count == 0 {
		h.redis.Expire(ctx, rateKey, 1*time.Hour)
	}

	// Generate magic token
	token := uuid.New().String()
	tokenKey := fmt.Sprintf("magic:%s", token)
	tokenData := fmt.Sprintf("%s|%s", body.Email, body.LangPref)

	// Store in Redis with expiry
	expiry := time.Duration(h.config.MagicLinkExpiryMin) * time.Minute
	err = h.redis.Set(ctx, tokenKey, tokenData, expiry).Err()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	// Build magic link URL
	magicURL := fmt.Sprintf("%s/login/magic-link/verify?token=%s", h.config.MagicLinkBaseURL, token)

	// Send email (don't fail the request if email fails - graceful degradation)
	if h.email != nil && h.email.IsConfigured() {
		htmlBody, renderErr := h.email.RenderMagicLinkTemplate(email.MagicLinkData{
			LoginURL:  magicURL,
			ExpiryMin: h.config.MagicLinkExpiryMin,
			Lang:      body.LangPref,
		})
		if renderErr != nil {
			log.Printf("[magic-link] template render error: %v", renderErr)
		} else if sendErr := h.email.Send(&domain.EmailHistory{
			ToEmail:  body.Email,
			Subject:  "Your Masaar CRM Login Link",
			HTMLBody: htmlBody,
		}); sendErr != nil {
			log.Printf("[magic-link] email send error: %v", sendErr)
		}
	} else {
		log.Printf("[magic-link] email service not configured, skipping send")
	}

	// Always return success to prevent email enumeration
	return c.JSON(fiber.Map{
		"message": "if the email exists, a magic link has been sent",
	})
}

// VerifyMagicLink godoc
// @Summary      Verify Magic Link
// @Description  Validate magic link token and issue JWT tokens. Requires an existing account — new users must register first.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      object{token=string}  true  "Magic link token"
// @Success      200   {object}  object{access_token=string,refresh_token=string,expires_in=int,user=object}
// @Failure      400   {object}  object{error=string}
// @Failure      401   {object}  object{error=string}
// @Router       /auth/magic-link/verify [post]
func (h *AuthHandler) VerifyMagicLink(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	if body.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token required"})
	}

	ctx := context.Background()
	tokenKey := fmt.Sprintf("magic:%s", body.Token)

	// Get and delete token (single-use)
	tokenData, err := h.redis.Get(ctx, tokenKey).Result()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
	}

	// Delete token to prevent reuse
	h.redis.Del(ctx, tokenKey)

	// Parse token data: email|lang_pref
	parts := split(tokenData, "|")
	if len(parts) != 2 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token data"})
	}
	email := parts[0]
	// lang_pref is stored in token but not needed here — user record already has it.
	_ = parts[1]

	// Find user — magic link does NOT auto-create accounts in multi-tenant mode.
	// New users must register via POST /auth/register to create a company workspace first.
	user, err := h.users.FindByEmail(ctx, email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "no account found for this email — please register at /signup first",
		})
	}

	if !user.IsActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account is deactivated"})
	}

	// Generate JWT tokens
	access, refresh, err := h.generateTokenPair(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "token generation failed"})
	}

	// Store refresh token in Redis
	key := fmt.Sprintf("refresh:%s", refresh)
	ttl := time.Duration(h.config.JWTRefreshExpiryDays) * 24 * time.Hour
	if err := h.redis.Set(ctx, key, user.ID.String(), ttl).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session error"})
	}

	resp := fiber.Map{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    h.config.JWTAccessExpiryMin * 60,
		"user": fiber.Map{
			"id":        user.ID,
			"name":      user.Name,
			"email":     user.Email,
			"role":      user.Role,
			"lang_pref": user.LangPref,
		},
	}
	if h.companyRepo != nil {
		if company, cerr := h.companyRepo.GetByID(ctx, user.CompanyID); cerr == nil {
			resp["company"] = fiber.Map{
				"id":             company.ID,
				"name":           company.Name,
				"plan":           company.Plan,
				"on_trial":       company.OnTrial,
				"trial_ends_at":  company.TrialEndsAt,
				"days_remaining": 0,
				"is_demo":        company.IsDemo,
			}
		}
	}
	return c.JSON(resp)
}

// verifyTurnstile validates a Cloudflare Turnstile token.
func verifyTurnstile(secret, token string) error {
	resp, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		url.Values{"secret": {secret}, "response": {token}},
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.Success {
		return fmt.Errorf("turnstile verification failed")
	}
	return nil
}

// split is a simple string split helper
func split(s, sep string) []string {
	idx := len(s)
	for i := 0; i < len(s)-len(sep)+1; i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
			break
		}
	}
	if idx == len(s) {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

// ─── SMS OTP Login ────────────────────────────────────────────────────────────

func generateOTP() (string, error) {
	const digits = "0123456789"
	otp := make([]byte, 6)
	for i := range otp {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[n.Int64()]
	}
	return string(otp), nil
}

// RequestSMSOTP godoc
// @Summary      Request SMS OTP
// @Description  Send a 6-digit OTP to a phone number for passwordless login. Rate limited to 3 per hour.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  object{phone=string,lang=string}  true  "Phone number (E.164 format, e.g. +971501234567)"
// @Success      200   {object}  object{message=string}
// @Failure      400   {object}  object{error=string}
// @Failure      503   {object}  object{error=string}
// @Router       /auth/sms/request [post]
func (h *AuthHandler) RequestSMSOTP(c *fiber.Ctx) error {
	if h.sms == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "SMS service not configured"})
	}

	var body struct {
		Phone string `json:"phone"`
		Lang  string `json:"lang"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	phone := strings.TrimSpace(body.Phone)
	if phone == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "phone is required"})
	}
	if len(phone) < 7 || len(phone) > 20 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid phone number"})
	}

	ctx := context.Background()

	// Rate limit: 3 OTPs per phone per hour
	rateKey := fmt.Sprintf("sms_rate:%s", phone)
	count, _ := h.redis.Get(ctx, rateKey).Int()
	if count >= 3 {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many OTP requests, please try again later"})
	}

	otp, err := generateOTP()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate OTP"})
	}

	lang := body.Lang
	if lang == "" {
		lang = "en"
	}

	// Store OTP in Redis (10 min TTL)
	otpKey := fmt.Sprintf("sms_otp:%s", phone)
	if err := h.redis.Set(ctx, otpKey, otp+"|"+lang, 10*time.Minute).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session error"})
	}

	// Increment rate counter
	h.redis.Incr(ctx, rateKey)
	h.redis.Expire(ctx, rateKey, 1*time.Hour)

	if err := h.sms.SendOTP(ctx, phone, otp); err != nil {
		h.redis.Del(ctx, otpKey)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "failed to send SMS"})
	}

	return c.JSON(fiber.Map{"message": "OTP sent"})
}

// VerifySMSOTP godoc
// @Summary      Verify SMS OTP
// @Description  Validate OTP and issue JWT tokens. Requires an existing account — new users must register first.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  object{phone=string,otp=string}  true  "Phone and OTP"
// @Success      200   {object}  object{access_token=string,refresh_token=string,expires_in=int,user=object}
// @Failure      400   {object}  object{error=string}
// @Failure      401   {object}  object{error=string}
// @Router       /auth/sms/verify [post]
func (h *AuthHandler) VerifySMSOTP(c *fiber.Ctx) error {
	var body struct {
		Phone string `json:"phone"`
		OTP   string `json:"otp"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	phone := strings.TrimSpace(body.Phone)
	otp := strings.TrimSpace(body.OTP)
	if phone == "" || otp == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "phone and otp are required"})
	}

	ctx := context.Background()
	otpKey := fmt.Sprintf("sms_otp:%s", phone)

	stored, err := h.redis.Get(ctx, otpKey).Result()
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired OTP"})
	}

	parts := split(stored, "|")
	storedOTP := parts[0]
	// lang_pref is stored in OTP payload but not needed here — user record already has it.

	if otp != storedOTP {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid OTP"})
	}

	// Consume OTP (single-use)
	h.redis.Del(ctx, otpKey)
	h.redis.Del(ctx, fmt.Sprintf("sms_rate:%s", phone))

	// Find user by phone — SMS OTP does NOT auto-create accounts in multi-tenant mode.
	// New users must register via POST /auth/register to create a company workspace first.
	user, err := h.users.FindByPhone(ctx, phone)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "no account found for this phone number — please register at /signup first",
		})
	}

	if !user.IsActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account is deactivated"})
	}

	access, refresh, err := h.generateTokenPair(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "token generation failed"})
	}

	key := fmt.Sprintf("refresh:%s", refresh)
	ttl := time.Duration(h.config.JWTRefreshExpiryDays) * 24 * time.Hour
	if err := h.redis.Set(ctx, key, user.ID.String(), ttl).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session error"})
	}

	resp := fiber.Map{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    h.config.JWTAccessExpiryMin * 60,
		"user": fiber.Map{
			"id":        user.ID,
			"name":      user.Name,
			"email":     user.Email,
			"role":      user.Role,
			"lang_pref": user.LangPref,
			"phone":     user.Phone,
		},
	}
	if h.companyRepo != nil {
		if company, cerr := h.companyRepo.GetByID(ctx, user.CompanyID); cerr == nil {
			resp["company"] = fiber.Map{
				"id":             company.ID,
				"name":           company.Name,
				"plan":           company.Plan,
				"on_trial":       company.OnTrial,
				"trial_ends_at":  company.TrialEndsAt,
				"days_remaining": 0,
				"is_demo":        company.IsDemo,
			}
		}
	}
	return c.JSON(resp)
}
