package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyTurnstile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
		}
		if r.FormValue("secret") != "test-secret" || r.FormValue("response") != "test-token" {
			t.Errorf("unexpected form values: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer srv.Close()

	origEndpoint := turnstileEndpoint
	origClient := turnstileHTTPClient
	turnstileEndpoint = srv.URL
	turnstileHTTPClient = srv.Client()
	defer func() {
		turnstileEndpoint = origEndpoint
		turnstileHTTPClient = origClient
	}()

	err := verifyTurnstile(context.Background(), "test-secret", "test-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestVerifyTurnstile_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": false}`))
	}))
	defer srv.Close()

	origEndpoint := turnstileEndpoint
	origClient := turnstileHTTPClient
	turnstileEndpoint = srv.URL
	turnstileHTTPClient = srv.Client()
	defer func() {
		turnstileEndpoint = origEndpoint
		turnstileHTTPClient = origClient
	}()

	err := verifyTurnstile(context.Background(), "test-secret", "test-token")
	if err == nil {
		t.Fatalf("expected error for failed captcha, got nil")
	}
}
