package docusign

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	"github.com/maidulcu/masaar-crm/internal/safehttp"
)

func generateTestRSAPrivateKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	}
	return string(pem.EncodeToMemory(block))
}

func TestClient_Enabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{
			name: "all fields present",
			cfg: Config{
				IntegrationKey: "ikey",
				PrivateKeyPEM:  "pem",
				UserID:         "uid",
				AccountID:      "acct",
			},
			want: true,
		},
		{
			name: "missing integration key",
			cfg: Config{
				PrivateKeyPEM: "pem",
				UserID:        "uid",
				AccountID:     "acct",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.cfg)
			if got := c.Enabled(); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_SSRFProtection(t *testing.T) {
	keyPEM := generateTestRSAPrivateKey(t)
	cfg := Config{
		IntegrationKey: "test-ikey",
		PrivateKeyPEM:  keyPEM,
		UserID:         "test-user-id",
		AccountID:      "test-account-id",
		BaseURL:        "http://127.0.0.1:8080/restapi",
	}

	c := NewClient(cfg)
	ctx := context.Background()

	// ensureToken attempts an OAuth request to account-d.docusign.com which is public or safehttp-guarded.
	// But GetEnvelopeStatus uses c.acctBase built from BaseURL (http://127.0.0.1:8080/restapi).
	// Since 127.0.0.1 is loopback, safehttp dialer must block it with safehttp.ErrBlocked.
	c.token = "mock-token"
	c.tokenExp = time.Now().Add(1 * time.Hour) // fake non-expired token to hit c.acctBase directly

	_, err := c.GetEnvelopeStatus(ctx, "env-123")
	if err == nil {
		t.Fatal("expected error due to loopback URL, got nil")
	}
	if !errors.Is(err, safehttp.ErrBlocked) {
		t.Errorf("expected safehttp.ErrBlocked, got: %v", err)
	}
}
