package sms

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/maidulcu/masaar-crm/internal/safehttp"
)

func TestIsEnabled(t *testing.T) {
	if !IsEnabled("key", "token") {
		t.Errorf("expected IsEnabled to be true")
	}
	if IsEnabled("", "token") {
		t.Errorf("expected IsEnabled to be false with empty authKey")
	}
	if IsEnabled("key", "") {
		t.Errorf("expected IsEnabled to be false with empty authToken")
	}
}

func TestSendOTP_SSRFBlocked(t *testing.T) {
	client := NewClient("authKey", "authToken", "SenderID")
	// Pointing to loopback address must be blocked by safehttp dialer
	client.baseURL = "http://127.0.0.1:8080"

	err := client.SendOTP(context.Background(), "+971501234567", "123456")
	if err == nil {
		t.Fatalf("expected SSRF error when targeting loopback, got nil")
	}
	if !errors.Is(err, safehttp.ErrBlocked) && !strings.Contains(err.Error(), "SSRF blocked") && !strings.Contains(err.Error(), "restricted IP") {
		t.Fatalf("expected safehttp.ErrBlocked or SSRF restriction, got: %v", err)
	}
}

func TestSendOTP_Success(t *testing.T) {
	var capturedUser, capturedPass, capturedContentType string
	var capturedBody map[string]string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUser, capturedPass, _ = r.BasicAuth()
		capturedContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer ts.Close()

	client := NewClient("myAuthKey", "myAuthToken", "MASAAR")
	client.baseURL = ts.URL
	// Override transport to allow loopback in mock server unit test
	client.httpClient.Transport = http.DefaultTransport

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.SendOTP(ctx, "+971501234567", "654321"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedUser != "myAuthKey" || capturedPass != "myAuthToken" {
		t.Errorf("unexpected basic auth: %s:%s", capturedUser, capturedPass)
	}
	if capturedContentType != "application/json" {
		t.Errorf("unexpected Content-Type: %s", capturedContentType)
	}
	if capturedBody["Number"] != "+971501234567" {
		t.Errorf("unexpected phone number: %s", capturedBody["Number"])
	}
	if capturedBody["SenderId"] != "MASAAR" {
		t.Errorf("unexpected sender id: %s", capturedBody["SenderId"])
	}
	if !strings.Contains(capturedBody["Text"], "654321") {
		t.Errorf("text does not contain otp: %s", capturedBody["Text"])
	}
}

func TestSendOTP_ErrorResponseBounded(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		// Return 10KB of error data
		_, _ = io.WriteString(w, strings.Repeat("A", 10*1024))
	}))
	defer ts.Close()

	client := NewClient("myAuthKey", "myAuthToken", "MASAAR")
	client.baseURL = ts.URL
	client.httpClient.Transport = http.DefaultTransport

	err := client.SendOTP(context.Background(), "+971501234567", "654321")
	if err == nil {
		t.Fatalf("expected error on 400 response, got nil")
	}
	// The error message contains "smscountry: error 400: " + at most 4KB of payload
	if len(err.Error()) > 4096+100 {
		t.Fatalf("error message too large, response body was not bounded: len=%d", len(err.Error()))
	}
}
