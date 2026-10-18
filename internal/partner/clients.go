// Package partner holds the partner clients allowed to use the Partner API: systems that sell
// the CRM as part of their own plans and create companies, users and logins for their customers.
// Clients are configured per deployment in PARTNER_CLIENTS; only a SHA-256 of each secret is kept.
package partner

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/maidulcu/masaar-crm/internal/apiscope"
)

// Client is one configured partner.
type Client struct {
	ID         string
	secretHash []byte
	Scopes     []string
}

// Clients is the set of configured partners, keyed by client id.
type Clients map[string]Client

var clientIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,62}$`)

// Parse reads "client_id:sha256hex(secret)[:scope scope],…". A client without scopes gets every
// partner scope.
func Parse(cfg string) (Clients, error) {
	out := Clients{}
	for _, entry := range strings.Split(cfg, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("PARTNER_CLIENTS: %q is not client_id:sha256hex", entry)
		}
		id := parts[0]
		if !clientIDRe.MatchString(id) {
			return nil, fmt.Errorf("PARTNER_CLIENTS: invalid client id %q (lowercase letters, digits, - and _)", id)
		}
		hash, err := hex.DecodeString(parts[1])
		if err != nil || len(hash) != sha256.Size {
			return nil, fmt.Errorf("PARTNER_CLIENTS: secret of %q must be a hex SHA-256", id)
		}
		var scopes []string
		if len(parts) == 3 && strings.TrimSpace(parts[2]) != "" {
			for _, s := range strings.Fields(parts[2]) {
				if _, ok := apiscope.PartnerScopes[s]; !ok {
					return nil, fmt.Errorf("PARTNER_CLIENTS: unknown scope %q for %q", s, id)
				}
				scopes = append(scopes, s)
			}
		} else {
			for s := range apiscope.PartnerScopes {
				scopes = append(scopes, s)
			}
		}
		sort.Strings(scopes)
		if _, dup := out[id]; dup {
			return nil, fmt.Errorf("PARTNER_CLIENTS: %q listed twice", id)
		}
		out[id] = Client{ID: id, secretHash: hash, Scopes: scopes}
	}
	return out, nil
}

// Authenticate returns the client when secret matches, comparing in constant time.
func (cs Clients) Authenticate(id, secret string) (Client, bool) {
	c, ok := cs[id]
	sum := sha256.Sum256([]byte(secret))
	if !ok {
		subtle.ConstantTimeCompare(sum[:], sum[:]) // same work for unknown ids
		return Client{}, false
	}
	return c, subtle.ConstantTimeCompare(sum[:], c.secretHash) == 1
}

// Grant returns the scopes to put in a token: the requested ones (space-separated), which must
// all be held by the client, or all of the client's scopes when none are requested.
func (c Client) Grant(requested string) ([]string, bool) {
	req := strings.Fields(requested)
	if len(req) == 0 {
		return c.Scopes, true
	}
	held := map[string]bool{}
	for _, s := range c.Scopes {
		held[s] = true
	}
	seen := map[string]bool{}
	var out []string
	for _, s := range req {
		if !held[s] {
			return nil, false
		}
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out, true
}
