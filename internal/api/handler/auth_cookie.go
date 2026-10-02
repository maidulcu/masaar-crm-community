package handler

import (
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Browser sessions keep the refresh token in an HttpOnly cookie so that an XSS bug cannot
// exfiltrate a long-lived credential. It is opt-in per request: the SPA sends
// "X-Auth-Mode: cookie"; API/mobile clients that omit it keep receiving the refresh token in
// the JSON body exactly as before.
//
// CSRF: in cookie mode the browser attaches the cookie automatically, so the endpoints that
// consume it require (a) the custom X-Auth-Mode header, which a cross-site page cannot send
// without passing a CORS preflight, (b) an Origin that is in ALLOWED_ORIGINS, and (c) the
// cookie is SameSite (Lax by default) and scoped to /api/v1/auth.
const (
	refreshCookieName = "masaar_rt"
	authModeHeader    = "X-Auth-Mode"
	authModeCookie    = "cookie"
	refreshCookiePath = "/api/v1/auth"
)

func cookieMode(c *fiber.Ctx) bool {
	return c.Get(authModeHeader) == authModeCookie
}

// attachRefresh adds the refresh token to a login-style response body, or — in cookie mode —
// sets it as the HttpOnly cookie and keeps it out of the body.
func (h *AuthHandler) attachRefresh(c *fiber.Ctx, resp fiber.Map, refresh string) fiber.Map {
	if cookieMode(c) {
		h.setRefreshCookie(c, refresh)
		return resp
	}
	resp["refresh_token"] = refresh
	return resp
}

func (h *AuthHandler) cookieSameSite() string {
	switch strings.ToLower(h.config.AuthCookieSameSite) {
	case "strict":
		return fiber.CookieSameSiteStrictMode
	case "none":
		return fiber.CookieSameSiteNoneMode
	default:
		return fiber.CookieSameSiteLaxMode
	}
}

func (h *AuthHandler) setRefreshCookie(c *fiber.Ctx, token string) {
	sameSite := h.cookieSameSite()
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     refreshCookiePath,
		Domain:   h.config.AuthCookieDomain,
		MaxAge:   h.config.JWTRefreshExpiryDays * 24 * 60 * 60,
		Expires:  time.Now().Add(time.Duration(h.config.JWTRefreshExpiryDays) * 24 * time.Hour),
		HTTPOnly: true,
		// SameSite=None is only honoured by browsers together with Secure.
		Secure:   h.config.IsProduction() || sameSite == fiber.CookieSameSiteNoneMode,
		SameSite: sameSite,
	})
}

func (h *AuthHandler) clearRefreshCookie(c *fiber.Ctx) {
	sameSite := h.cookieSameSite()
	c.Cookie(&fiber.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     refreshCookiePath,
		Domain:   h.config.AuthCookieDomain,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		Secure:   h.config.IsProduction() || sameSite == fiber.CookieSameSiteNoneMode,
		SameSite: sameSite,
	})
}

// originAllowed rejects cookie-authenticated requests that browsers mark as coming from a
// foreign origin. A missing Origin (non-browser client) is allowed; such a client cannot be
// riding a victim's cookie. With ALLOWED_ORIGINS="*" (development) every origin passes.
func (h *AuthHandler) originAllowed(c *fiber.Ctx) bool {
	origin := c.Get(fiber.HeaderOrigin)
	if origin == "" {
		return true
	}
	allowed := strings.TrimSpace(h.config.AllowedOrigins)
	if allowed == "*" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	normalized := strings.ToLower(u.Scheme + "://" + u.Host)
	for _, o := range strings.Split(allowed, ",") {
		if strings.ToLower(strings.TrimRight(strings.TrimSpace(o), "/")) == normalized {
			return true
		}
	}
	return false
}
