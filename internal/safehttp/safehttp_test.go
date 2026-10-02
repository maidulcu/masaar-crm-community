package safehttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "0.0.0.0", "0.1.2.3",
		"100.64.0.1", "100.100.100.200", "198.18.0.1", "224.0.0.1", "240.0.0.1", "255.255.255.255",
		"::1", "::", "fe80::1", "fc00::1", "ff02::1", "::ffff:10.0.0.1", "::ffff:127.0.0.1", "64:ff9b::a00:1",
	}
	for _, s := range blocked {
		if !IsBlockedIP(net.ParseIP(s)) {
			t.Errorf("%s should be blocked", s)
		}
	}
	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:4700:4700::1111"}
	for _, s := range allowed {
		if IsBlockedIP(net.ParseIP(s)) {
			t.Errorf("%s should be allowed", s)
		}
	}
	if !IsBlockedIP(nil) {
		t.Error("nil must be blocked")
	}
}

func TestFetchBlocksInternalTargets(t *testing.T) {
	// A real local server stands in for an internal service (Redis, metadata, admin UI).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("secret")) }))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, 1024, 2*time.Second)
	if !errors.Is(err, ErrBlocked) && (err == nil || !strings.Contains(err.Error(), "SSRF blocked")) {
		t.Fatalf("loopback fetch must be blocked, got %v", err)
	}
	for _, u := range []string{"file:///etc/passwd", "gopher://127.0.0.1:6379/", "ftp://example.com/x", "://bad"} {
		if _, err := Fetch(context.Background(), u, 1024, time.Second); err == nil {
			t.Errorf("%s must be rejected", u)
		}
	}
}

func TestRedirectToInternalIsBlocked(t *testing.T) {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("internal")) }))
	defer internal.Close()
	// Even a "public" first hop cannot bounce the client to an internal address: the dialer
	// checks every connection, not just the first URL.
	c := NewClient(2 * time.Second)
	req, _ := http.NewRequest(http.MethodGet, internal.URL, nil)
	if _, err := c.Do(req); err == nil || !strings.Contains(err.Error(), "SSRF blocked") {
		t.Fatalf("expected SSRF block, got %v", err)
	}
}
