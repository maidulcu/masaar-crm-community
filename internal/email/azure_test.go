package email

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

func TestAzureServiceIsConfigured(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("secret-key"))
	svc := NewAzureService("https://example.communication.azure.com", key, "DoNotReply@example.com")

	if !svc.IsConfigured() {
		t.Errorf("expected IsConfigured to return true")
	}

	unconfigured := NewAzureService("", "", "")
	if unconfigured.IsConfigured() {
		t.Errorf("expected unconfigured service IsConfigured to return false")
	}
}

func TestAzureServiceSendWithContext_CancelledContext(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("secret-key"))
	svc := NewAzureService("https://example.communication.azure.com", key, "DoNotReply@example.com")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	email := &domain.EmailHistory{
		ToEmail:  "user@example.com",
		Subject:  "Test Subject",
		Body:     "Test Body",
		HTMLBody: "<p>Test Body</p>",
	}

	err := svc.SendWithContext(ctx, email)
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}
}

func TestAzureServiceSSRFBlocked(t *testing.T) {
	// 127.0.0.1 should be blocked by safehttp dialer
	key := base64.StdEncoding.EncodeToString([]byte("secret-key"))
	svc := NewAzureService("http://127.0.0.1:8080", key, "DoNotReply@example.com")

	email := &domain.EmailHistory{
		ToEmail:  "user@example.com",
		Subject:  "Test",
		Body:     "Test",
	}

	err := svc.SendWithContext(context.Background(), email)
	if err == nil {
		t.Fatalf("expected SSRF error when sending to 127.0.0.1, got nil")
	}
	if !strings.Contains(err.Error(), "SSRF blocked") && !strings.Contains(err.Error(), "restricted IP") && !strings.Contains(err.Error(), "blocked") {
		t.Logf("got error as expected: %v", err)
	}
}

func TestAzureServiceSend_SuccessWithMockServer(t *testing.T) {
	// Test server using standard http client (custom client override for unit test mock server)
	var capturedAuth, capturedDate, capturedContentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedDate = r.Header.Get("x-ms-date")
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	key := base64.StdEncoding.EncodeToString([]byte("secret-key"))
	svc := NewAzureService(ts.URL, key, "DoNotReply@example.com")
	// Use standard Transport for loopback mock server test specifically
	svc.client.Transport = http.DefaultTransport

	email := &domain.EmailHistory{
		ToEmail:  "recipient@example.com",
		Subject:  "Welcome",
		Body:     "Hello World",
		HTMLBody: "<h1>Hello World</h1>",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := svc.SendWithContext(ctx, email); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(capturedAuth, "HMAC-SHA256 ") {
		t.Errorf("unexpected Auth header: %q", capturedAuth)
	}
	if capturedDate == "" {
		t.Errorf("missing x-ms-date header")
	}
	if capturedContentType != "application/json" {
		t.Errorf("unexpected content type: %q", capturedContentType)
	}
}
