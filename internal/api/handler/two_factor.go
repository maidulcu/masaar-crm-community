package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/maidulcu/masaar-crm/internal/domain"
	"github.com/maidulcu/masaar-crm/internal/repo"
	"github.com/maidulcu/masaar-crm/internal/totp"
)

// Two-factor authentication with an authenticator app (#87).
//
// Login (password, magic link or SMS code) proves the first factor. When the user has
// two-factor on, or their company requires it, the answer is a short-lived pending token
// instead of a session; POST /auth/verify-totp exchanges it, with a code or a backup
// code, for the session. A user whose company requires two-factor but who has not set it
// up enrols with the same pending token (POST /auth/totp/enroll, then …/confirm).

const (
	pendingTTL     = 5 * time.Minute
	setupTTL       = 10 * time.Minute
	maxCodeFailure = 5
	totpIssuer     = "Masaar CRM"
)

// UseTwoFactor turns on two-factor authentication.
func (h *AuthHandler) UseTwoFactor(r *repo.TwoFactorRepo, box *totp.Box) *AuthHandler {
	h.twoFactor, h.totpBox = r, box
	return h
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// twoFactorGate is called once a user has passed the first factor. When a second factor
// is needed, it answers with a pending token and returns true.
func (h *AuthHandler) twoFactorGate(c *fiber.Ctx, user *domain.User, method string) (bool, error) {
	if h.twoFactor == nil {
		return false, nil
	}
	st, err := h.twoFactor.State(c.Context(), user.ID)
	if err != nil {
		return true, serverError(c, err)
	}
	if !st.Enabled && !st.RequiredByAdmin {
		return false, nil
	}
	// A suspended company is refused now, not after the user has typed a code.
	if _, _, cerr := h.loadCompany(c.Context(), user); cerr != nil {
		return true, companyError(c, cerr)
	}
	tok, err := randomToken()
	if err != nil {
		return true, serverError(c, err)
	}
	if err := h.redis.Set(c.Context(), "2fa:pending:"+tok, user.ID.String()+"|"+method, pendingTTL).Err(); err != nil {
		return true, c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session error"})
	}
	return true, c.JSON(fiber.Map{
		"two_factor_required": true,
		"setup_required":      !st.Enabled,
		"pending_token":       tok,
		"expires_in":          int(pendingTTL.Seconds()),
	})
}

// pendingUser resolves a pending token to its (active) user.
func (h *AuthHandler) pendingUser(c *fiber.Ctx, tok string) (*domain.User, string, error) {
	if len(tok) != 64 {
		return nil, "", errors.New("invalid")
	}
	v, err := h.redis.Get(c.Context(), "2fa:pending:"+tok).Result()
	if err != nil {
		return nil, "", err
	}
	idStr, method, _ := strings.Cut(v, "|")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, "", err
	}
	u, err := h.users.FindByID(c.Context(), id)
	if err != nil {
		return nil, "", err
	}
	if !u.IsActive {
		return nil, "", errors.New("inactive")
	}
	return u, method, nil
}

func expiredPending(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "the sign-in has expired; please sign in again"})
}

// codeFailure counts a wrong code. After maxCodeFailure failures the account is locked
// for 15 minutes with the same lock as password failures, and the pending token dies.
func (h *AuthHandler) codeFailure(c *fiber.Ctx, user *domain.User, pendingKey string) error {
	failKey := "2fa:fail:" + user.ID.String()
	n, _ := h.redis.Incr(c.Context(), failKey).Result()
	h.redis.Expire(c.Context(), failKey, 15*time.Minute)
	if n >= maxCodeFailure {
		h.redis.Set(c.Context(), "lockout:"+strings.ToLower(strings.TrimSpace(user.Email)), "1", 15*time.Minute)
		h.redis.Del(c.Context(), failKey)
		if pendingKey != "" {
			h.redis.Del(c.Context(), pendingKey)
		}
		h.audit.Log(c.Context(), user.ID, "2fa_locked", repo.AuditUser, user.ID, fiber.Map{"ip": c.IP()})
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "account locked for 15 minutes after too many wrong codes"})
	}
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid code"})
}

func (h *AuthHandler) locked(c *fiber.Ctx, user *domain.User) bool {
	n, _ := h.redis.Exists(c.Context(), "lockout:"+strings.ToLower(strings.TrimSpace(user.Email))).Result()
	return n > 0
}

// checkCode verifies a six-digit code (against the user's stored secret) or a backup
// code, consuming it. It returns how the user proved it ("totp" or "backup_code").
func (h *AuthHandler) checkCode(c *fiber.Ctx, user *domain.User, st *repo.TwoFactorState, code string) (string, bool, error) {
	code = strings.TrimSpace(code)
	if totp.LooksLikeBackupCode(code) {
		ok, err := h.twoFactor.UseBackupCode(c.Context(), user.ID, code)
		return "backup_code", ok, err
	}
	secret, err := h.totpBox.Open(st.SealedSecret)
	if err != nil {
		return "", false, err
	}
	step, ok := totp.Verify(secret, code, time.Now(), st.LastStep)
	if !ok {
		return "totp", false, nil
	}
	if err := h.twoFactor.AdvanceStep(c.Context(), user.ID, step); err != nil {
		if errors.Is(err, repo.ErrCodeReused) {
			return "totp", false, nil
		}
		return "", false, err
	}
	return "totp", true, nil
}

// VerifyTOTP godoc
// @Summary      Complete sign-in with a two-factor code
// @Description  Exchanges the pending token from /auth/login (or the magic-link / SMS sign-in) and a six-digit code from the authenticator app, or a backup code, for a session. Five wrong codes lock the account for 15 minutes.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  object{pending_token=string,code=string}  true  "Pending token and code"
// @Success      200  {object}  object{access_token=string,expires_in=int,user=object}
// @Failure      401  {object}  object{error=string}
// @Failure      429  {object}  object{error=string}
// @Router       /auth/verify-totp [post]
func (h *AuthHandler) VerifyTOTP(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	var body struct {
		PendingToken string `json:"pending_token"`
		Code         string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	user, method, err := h.pendingUser(c, body.PendingToken)
	if err != nil {
		return expiredPending(c)
	}
	if h.locked(c, user) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "account temporarily locked; try again in 15 minutes"})
	}
	st, err := h.twoFactor.State(c.Context(), user.ID)
	if err != nil {
		return serverError(c, err)
	}
	if !st.Enabled {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "two-factor authentication is not set up yet", "setup_required": true})
	}
	how, ok, err := h.checkCode(c, user, st, body.Code)
	if err != nil {
		return serverError(c, err)
	}
	pendingKey := "2fa:pending:" + body.PendingToken
	if !ok {
		return h.codeFailure(c, user, pendingKey)
	}
	// One use: a second request with the same pending token finds nothing.
	if n, _ := h.redis.Del(c.Context(), pendingKey).Result(); n == 0 {
		return expiredPending(c)
	}
	h.redis.Del(c.Context(), "2fa:fail:"+user.ID.String())
	h.audit.Log(c.Context(), user.ID, "2fa_verified", repo.AuditUser, user.ID, fiber.Map{"ip": c.IP(), "first_factor": method, "second_factor": how})
	extra := fiber.Map{}
	if how == "backup_code" {
		extra["backup_codes_left"] = st.BackupCodesLeft - 1
	}
	return h.completeLogin(c, user, extra)
}

func (h *AuthHandler) newSetup(c *fiber.Ctx, user *domain.User) error {
	secret, err := totp.NewSecret()
	if err != nil {
		return serverError(c, err)
	}
	sealed, err := h.totpBox.Seal(secret)
	if err != nil {
		return serverError(c, err)
	}
	if err := h.redis.Set(c.Context(), "2fa:setup:"+user.ID.String(), sealed, setupTTL).Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session error"})
	}
	return c.JSON(fiber.Map{"secret": secret, "otpauth_url": totp.URI(totpIssuer, user.Email, secret), "expires_in": int(setupTTL.Seconds())})
}

// confirmSetup checks a code against the secret being set up and turns two-factor on.
// It returns the new backup codes, or nil after writing an error response.
func (h *AuthHandler) confirmSetup(c *fiber.Ctx, user *domain.User, code string) ([]string, error) {
	sealed, err := h.redis.Get(c.Context(), "2fa:setup:"+user.ID.String()).Result()
	if err != nil {
		return nil, c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "start the set-up again: it has expired"})
	}
	secret, err := h.totpBox.Open(sealed)
	if err != nil {
		return nil, serverError(c, err)
	}
	step, ok := totp.Verify(secret, code, time.Now(), 0)
	if !ok {
		return nil, h.codeFailure(c, user, "")
	}
	codes, err := totp.NewBackupCodes(totp.BackupCodeCount)
	if err != nil {
		return nil, serverError(c, err)
	}
	if err := h.twoFactor.Enable(c.Context(), user.ID, sealed, step, codes); err != nil {
		return nil, serverError(c, err)
	}
	h.redis.Del(c.Context(), "2fa:setup:"+user.ID.String(), "2fa:fail:"+user.ID.String())
	h.audit.Log(c.Context(), user.ID, "2fa_enabled", repo.AuditUser, user.ID, fiber.Map{"ip": c.IP()})
	return codes, nil
}

// EnrollPending godoc
// @Summary      Start setting up two-factor during sign-in
// @Description  For users whose company requires two-factor and who have not set it up. Returns the secret and an otpauth:// URL to show as a QR code.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  object{pending_token=string}  true  "Pending token"
// @Success      200  {object}  object{secret=string,otpauth_url=string}
// @Router       /auth/totp/enroll [post]
func (h *AuthHandler) EnrollPending(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	var body struct {
		PendingToken string `json:"pending_token"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	user, _, err := h.pendingUser(c, body.PendingToken)
	if err != nil {
		return expiredPending(c)
	}
	if st, err := h.twoFactor.State(c.Context(), user.ID); err != nil {
		return serverError(c, err)
	} else if st.Enabled {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "two-factor is already set up; enter a code"})
	}
	return h.newSetup(c, user)
}

// ConfirmEnrollPending godoc
// @Summary      Finish setting up two-factor during sign-in
// @Description  Confirms the set-up with a code from the app, turns two-factor on and signs the user in. The response includes the 8 backup codes, shown once.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  object{pending_token=string,code=string}  true  "Pending token and code"
// @Success      200  {object}  object{access_token=string,backup_codes=[]string}
// @Router       /auth/totp/enroll/confirm [post]
func (h *AuthHandler) ConfirmEnrollPending(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	var body struct {
		PendingToken string `json:"pending_token"`
		Code         string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	user, _, err := h.pendingUser(c, body.PendingToken)
	if err != nil {
		return expiredPending(c)
	}
	if h.locked(c, user) {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "account temporarily locked; try again in 15 minutes"})
	}
	codes, err := h.confirmSetup(c, user, body.Code)
	if codes == nil {
		return err
	}
	if n, _ := h.redis.Del(c.Context(), "2fa:pending:"+body.PendingToken).Result(); n == 0 {
		return expiredPending(c)
	}
	return h.completeLogin(c, user, fiber.Map{"backup_codes": codes})
}

// ── Signed-in user ────────────────────────────────────────────────────────────

func (h *AuthHandler) me(c *fiber.Ctx) (*domain.User, error) {
	id, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return nil, errors.New("no user")
	}
	return h.users.FindByID(c.Context(), id)
}

// TwoFactorStatus godoc
// @Summary      My two-factor status
// @Tags         Users
// @Produce      json
// @Success      200  {object}  repo.TwoFactorState
// @Security     BearerAuth
// @Router       /users/me/2fa [get]
func (h *AuthHandler) TwoFactorStatus(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	user, err := h.me(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	st, err := h.twoFactor.State(c.Context(), user.ID)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(st)
}

// TwoFactorSetup godoc
// @Summary      Start setting up two-factor
// @Description  Returns a new secret and an otpauth:// URL to show as a QR code. Confirm with POST /users/me/2fa/enable within 10 minutes.
// @Tags         Users
// @Produce      json
// @Success      200  {object}  object{secret=string,otpauth_url=string}
// @Security     BearerAuth
// @Router       /users/me/2fa/setup [post]
func (h *AuthHandler) TwoFactorSetup(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	user, err := h.me(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	if st, err := h.twoFactor.State(c.Context(), user.ID); err != nil {
		return serverError(c, err)
	} else if st.Enabled {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "two-factor is already on; turn it off first to change the app"})
	}
	return h.newSetup(c, user)
}

// TwoFactorEnable godoc
// @Summary      Turn two-factor on
// @Description  Confirms the set-up with a code from the app. Returns 8 backup codes, shown once.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body  body  object{code=string}  true  "Code from the app"
// @Success      200  {object}  object{backup_codes=[]string}
// @Security     BearerAuth
// @Router       /users/me/2fa/enable [post]
func (h *AuthHandler) TwoFactorEnable(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	user, err := h.me(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	codes, err := h.confirmSetup(c, user, body.Code)
	if codes == nil {
		return err
	}
	return c.JSON(fiber.Map{"backup_codes": codes})
}

// verifyPasswordAndCode re-authenticates a signed-in user for a sensitive change.
func (h *AuthHandler) verifyPasswordAndCode(c *fiber.Ctx, user *domain.User, password, code string, needPassword bool) (bool, error) {
	if h.locked(c, user) {
		return false, c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "account temporarily locked; try again in 15 minutes"})
	}
	if needPassword && bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return false, h.codeFailure(c, user, "")
	}
	st, err := h.twoFactor.State(c.Context(), user.ID)
	if err != nil {
		return false, serverError(c, err)
	}
	if !st.Enabled {
		return false, c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "two-factor is not on"})
	}
	_, ok, err := h.checkCode(c, user, st, code)
	if err != nil {
		return false, serverError(c, err)
	}
	if !ok {
		return false, h.codeFailure(c, user, "")
	}
	return true, nil
}

// TwoFactorDisable godoc
// @Summary      Turn two-factor off
// @Description  Needs the password and a code (or a backup code). Refused while the company requires two-factor.
// @Tags         Users
// @Accept       json
// @Param        body  body  object{password=string,code=string}  true  "Password and code"
// @Success      204
// @Security     BearerAuth
// @Router       /users/me/2fa/disable [post]
func (h *AuthHandler) TwoFactorDisable(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	user, err := h.me(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	var body struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if st, err := h.twoFactor.State(c.Context(), user.ID); err != nil {
		return serverError(c, err)
	} else if st.RequiredByAdmin {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "your company requires two-factor authentication"})
	}
	if ok, err := h.verifyPasswordAndCode(c, user, body.Password, body.Code, true); !ok {
		return err
	}
	if err := h.twoFactor.Disable(c.Context(), user.ID); err != nil {
		return serverError(c, err)
	}
	h.audit.Log(c.Context(), user.ID, "2fa_disabled", repo.AuditUser, user.ID, fiber.Map{"ip": c.IP()})
	return c.SendStatus(fiber.StatusNoContent)
}

// TwoFactorBackupCodes godoc
// @Summary      Replace my backup codes
// @Description  Needs a current code. The old codes stop working.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        body  body  object{code=string}  true  "Code from the app"
// @Success      200  {object}  object{backup_codes=[]string}
// @Security     BearerAuth
// @Router       /users/me/2fa/backup-codes [post]
func (h *AuthHandler) TwoFactorBackupCodes(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	user, err := h.me(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if ok, err := h.verifyPasswordAndCode(c, user, "", body.Code, false); !ok {
		return err
	}
	codes, err := totp.NewBackupCodes(totp.BackupCodeCount)
	if err != nil {
		return serverError(c, err)
	}
	if err := h.twoFactor.ReplaceBackupCodes(c.Context(), user.ID, codes); err != nil {
		return serverError(c, err)
	}
	h.audit.Log(c.Context(), user.ID, "2fa_backup_codes_replaced", repo.AuditUser, user.ID, nil)
	return c.JSON(fiber.Map{"backup_codes": codes})
}

// ── Admin ─────────────────────────────────────────────────────────────────────

func adminCompany(c *fiber.Ctx) (uuid.UUID, error) {
	s, _ := c.Locals("company_id").(string)
	return uuid.Parse(s)
}

// GetSecuritySettings godoc
// @Summary      Company security settings
// @Tags         Settings
// @Produce      json
// @Success      200  {object}  object{require_2fa=bool,users_without_2fa=int}
// @Security     BearerAuth
// @Router       /settings/security [get]
func (h *AuthHandler) GetSecuritySettings(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	cid, err := adminCompany(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	req, without, err := h.twoFactor.CompanyRequirement(c.Context(), cid)
	if err != nil {
		return serverError(c, err)
	}
	return c.JSON(fiber.Map{"require_2fa": req, "users_without_2fa": without})
}

// UpdateSecuritySettings godoc
// @Summary      Require two-factor for every user
// @Description  When on, users without two-factor are asked to set it up at their next sign-in. Existing sessions are not ended.
// @Tags         Settings
// @Accept       json
// @Param        body  body  object{require_2fa=bool}  true  "Setting"
// @Success      204
// @Security     BearerAuth
// @Router       /settings/security [put]
func (h *AuthHandler) UpdateSecuritySettings(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	cid, err := adminCompany(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	var body struct {
		Require2FA bool `json:"require_2fa"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}
	if err := h.twoFactor.SetCompanyRequirement(c.Context(), cid, body.Require2FA); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, repo.AuditUpdate, "company_settings", cid, fiber.Map{"require_2fa": body.Require2FA})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ResetUserTwoFactor godoc
// @Summary      Reset a user's two-factor (lost phone)
// @Description  Admin only, for a user of the same company. The user sets two-factor up again at the next sign-in if the company requires it.
// @Tags         Users
// @Param        id  path  string  true  "User ID"
// @Success      204
// @Security     BearerAuth
// @Router       /users/{id}/2fa/reset [post]
func (h *AuthHandler) ResetUserTwoFactor(c *fiber.Ctx) error {
	if h.twoFactor == nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	cid, err := adminCompany(c)
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}
	target, err := h.users.FindByID(c.Context(), id)
	if err != nil || target.CompanyID != cid {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	if err := h.twoFactor.Disable(c.Context(), id); err != nil {
		return serverError(c, err)
	}
	if actor := localsUserPtr(c); actor != nil {
		h.audit.Log(c.Context(), *actor, "2fa_reset", repo.AuditUser, id, fiber.Map{"user": fmt.Sprint(target.Email)})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
