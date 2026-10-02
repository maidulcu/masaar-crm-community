package config

import (
	"strings"
	"testing"
)

func validProd() *Config {
	return &Config{
		AppEnv:         "production",
		JWTSecret:      strings.Repeat("a", 40),
		AllowedOrigins: "https://crm.example.com",
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string // substring; empty means no error expected
	}{
		{"valid production", func(*Config) {}, ""},
		{"development ignores weak secret", func(c *Config) { c.AppEnv = "development"; c.JWTSecret = "x" }, ""},
		{"default secret", func(c *Config) { c.JWTSecret = defaultJWTSecret }, "JWT_SECRET is a placeholder"},
		{"env.example placeholder", func(c *Config) { c.JWTSecret = "change-me-to-random-string-at-least-32-chars" }, "JWT_SECRET is a placeholder"},
		{"short secret", func(c *Config) { c.JWTSecret = "short" }, "at least 32"},
		{"wildcard cors", func(c *Config) { c.AllowedOrigins = "*" }, "ALLOWED_ORIGINS"},
		{"empty cors", func(c *Config) { c.AllowedOrigins = "" }, "ALLOWED_ORIGINS"},
		{"whatsapp without app secret", func(c *Config) {
			c.WAAccessToken = "tok"
			c.WAVerifyToken = "custom-verify"
		}, "WA_APP_SECRET"},
		{"whatsapp default verify token", func(c *Config) {
			c.WAAccessToken = "tok"
			c.WAAppSecret = "s"
			c.WAVerifyToken = defaultWAVerifyToken
		}, "WA_VERIFY_TOKEN"},
		{"whatsapp fully configured", func(c *Config) {
			c.WAAccessToken = "tok"
			c.WAAppSecret = "s"
			c.WAVerifyToken = "custom-verify"
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validProd()
			tt.mutate(c)
			err := c.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
