// Package safehttp provides an HTTP client for fetching URLs that a user controls
// (webhook targets, document links). It refuses to connect to internal infrastructure, which
// prevents server-side request forgery (SSRF) against cloud metadata endpoints, the database,
// Redis and other private services.
//
// The check runs inside the dialer, on the address that was actually resolved, so DNS
// rebinding and redirects to internal hosts are blocked as well.
package safehttp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// ErrBlocked is wrapped by errors returned when a destination is not allowed.
var ErrBlocked = errors.New("SSRF blocked")

// blockedNets lists ranges that are not covered by the net.IP helper methods.
var blockedNets = mustCIDRs(
	"0.0.0.0/8",       // "this network"
	"100.64.0.0/10",   // carrier-grade NAT (also used by some cloud metadata services)
	"192.0.0.0/24",    // IETF protocol assignments
	"192.0.2.0/24",    // TEST-NET-1
	"198.18.0.0/15",   // benchmarking
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"240.0.0.0/4",     // reserved
	"64:ff9b::/96",    // NAT64: could embed an internal IPv4 address
	"fec0::/10",       // deprecated site-local
)

func mustCIDRs(list ...string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(list))
	for _, c := range list {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		out = append(out, n)
	}
	return out
}

// IsBlockedIP reports whether ip is not a routable public unicast address.
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4 // normalise IPv4-mapped IPv6 (::ffff:10.0.0.1)
	}
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func controlFunc(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("%w: invalid IP %q", ErrBlocked, host)
	}
	if IsBlockedIP(ip) {
		return fmt.Errorf("%w: restricted IP %s", ErrBlocked, ip)
	}
	return nil
}

// Transport returns an http.Transport whose dialer rejects internal addresses.
func Transport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   controlFunc,
	}).DialContext
	t.Proxy = nil // an environment proxy would bypass the dial-time address check
	return t
}

// NewClient returns a client that cannot reach internal addresses and follows at most 3 redirects.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: Transport(),
		Timeout:   timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

// Fetch downloads rawURL (http or https only) and returns at most maxBytes of the body. A
// larger body is an error rather than being silently truncated.
func Fetch(ctx context.Context, rawURL string, maxBytes int64, timeout time.Duration) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := NewClient(timeout).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("response exceeds %d bytes", maxBytes)
	}
	return body, nil
}
