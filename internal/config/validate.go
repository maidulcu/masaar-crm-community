package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	defaultJWTSecret     = "change-me-in-production"
	defaultWAVerifyToken = "masaar-webhook-token"
	minJWTSecretLen      = 32
)

// IsProduction reports whether the app is running with APP_ENV=production.
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

// Validate checks that the configuration is safe to run with. In production it
// refuses placeholder or weak secrets and wildcard CORS; in other environments
// it is a no-op so local development keeps working with .env.example values.
// Call it once at startup and exit on error.
func (c *Config) Validate() error {
	if !c.IsProduction() {
		return nil
	}

	var problems []string

	secret := strings.TrimSpace(c.JWTSecret)
	switch {
	case secret == defaultJWTSecret || strings.HasPrefix(strings.ToLower(secret), "change-me"):
		problems = append(problems, "JWT_SECRET is a placeholder; generate one with `openssl rand -hex 32`")
	case len(secret) < minJWTSecretLen:
		problems = append(problems, fmt.Sprintf("JWT_SECRET must be at least %d characters", minJWTSecretLen))
	}

	origins := strings.TrimSpace(c.AllowedOrigins)
	if origins == "" || origins == "*" {
		problems = append(problems, "ALLOWED_ORIGINS must list your frontend origin(s), not \"*\"")
	}

	// The database password must not be the published default / placeholder.
	if u, err := url.Parse(c.DatabaseURL); err == nil && u.User != nil {
		pw, _ := u.User.Password()
		lower := strings.ToLower(pw)
		if pw == "" || lower == "masaar" || lower == "postgres" || lower == "password" || strings.HasPrefix(lower, "change-me") {
			problems = append(problems, "database password (DB_PASSWORD / DATABASE_URL) is empty or a well-known default; set a strong one")
		}
	}

	// Only enforce WhatsApp settings when the integration is actually configured.
	if c.WAPhoneNumberID != "" || c.WAAccessToken != "" {
		if c.WAAppSecret == "" {
			problems = append(problems, "WA_APP_SECRET is required when WhatsApp is configured (inbound webhooks are unauthenticated without it)")
		}
		if c.WAVerifyToken == "" || c.WAVerifyToken == defaultWAVerifyToken {
			problems = append(problems, "WA_VERIFY_TOKEN must be changed from the published default")
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return errors.New("insecure production configuration:\n  - " + strings.Join(problems, "\n  - "))
}
