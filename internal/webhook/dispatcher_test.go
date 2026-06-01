package webhook

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maidulcu/masaar-crm/internal/repo"
)

func TestDispatcher_SSRF(t *testing.T) {
	// Create a dummy repository for testing to satisfy NewDispatcher
	dummyRepo := &repo.WebhookRepo{}
	dispatcher := NewDispatcher(dummyRepo)

	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{
			name:        "Loopback IP",
			url:         "http://127.0.0.1:8080/webhook",
			expectError: true,
		},
		{
			name:        "Private IP",
			url:         "http://10.0.0.1:8080/webhook",
			expectError: true,
		},
		{
			name:        "Link Local IP",
			url:         "http://169.254.169.254/metadata",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, tt.url, strings.NewReader(`{}`))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			_, err = dispatcher.client.Do(req)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URL %s, but got none", tt.url)
				} else if !strings.Contains(err.Error(), "SSRF blocked") {
					t.Errorf("Expected SSRF block error, got: %v", err)
				}
			} else {
				if err != nil && strings.Contains(err.Error(), "SSRF blocked") {
					t.Errorf("Did not expect SSRF block error, got: %v", err)
				}
			}
		})
	}
}

func TestDispatcher_AllowedIP(t *testing.T) {
	dummyRepo := &repo.WebhookRepo{}
	dispatcher := NewDispatcher(dummyRepo)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://8.8.8.8:80", strings.NewReader(`{}`))
	// Use a short timeout context to fail fast
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	_, err := dispatcher.client.Do(req)
	if err != nil && strings.Contains(err.Error(), "SSRF blocked") {
		t.Errorf("Expected connection to public IP to NOT be blocked by SSRF filter, but it was: %v", err)
	}
}
