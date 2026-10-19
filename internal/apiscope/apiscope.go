// Package apiscope defines the permissions an API key can carry. Scope names follow the OAuth 2.0
// convention (RFC 6749 §3.3): space-free tokens, here "<resource>:<action>". A key stores them
// comma-separated.
package apiscope

import (
	"fmt"
	"sort"
	"strings"
)

const (
	LeadCreate     = "lead:create" // POST /webhooks/leads only (the original intake scope)
	LeadsRead      = "leads:read"
	LeadsWrite     = "leads:write" // implies lead:create
	ContactsRead   = "contacts:read"
	ContactsWrite  = "contacts:write"
	ListingsRead   = "listings:read"
	WebhooksManage = "webhooks:manage"
)

// Partner scopes, held by partner clients (PARTNER_CLIENTS) rather than API keys.
const (
	PartnerProvision = "partner:provision" // create and update companies, users and their API keys
	PartnerLogin     = "partner:login"     // create one-click login links for provisioned users
)

// PartnerScopes lists the partner scopes with what each allows.
var PartnerScopes = map[string]string{
	PartnerProvision: "Create and update companies, their users and API keys",
	PartnerLogin:     "Create one-click login links for users of provisioned companies",
}

// Known lists every scope a new key may be given, with what it allows.
var Known = map[string]string{
	LeadCreate:     "Submit leads to POST /webhooks/leads",
	LeadsRead:      "List and read leads",
	LeadsWrite:     "Create leads, change their stage and notes (includes lead:create)",
	ContactsRead:   "List and read contacts",
	ContactsWrite:  "Create and update contacts",
	ListingsRead:   "List and read listings",
	WebhooksManage: "List, register and delete webhook subscriptions",
}

// implied maps a scope to the narrower scopes it includes.
var implied = map[string][]string{
	LeadsWrite: {LeadCreate},
}

// Normalize validates a comma- or space-separated scope list for a new key and returns it
// de-duplicated, sorted and comma-separated. Unknown scopes and wildcards are refused.
func Normalize(raw string) (string, error) {
	seen := map[string]bool{}
	for _, s := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' }) {
		if _, ok := Known[s]; !ok {
			return "", fmt.Errorf("unknown scope %q", s)
		}
		seen[s] = true
	}
	if len(seen) == 0 {
		return "", fmt.Errorf("at least one scope is required")
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return strings.Join(out, ","), nil
}

// Grants reports whether a key's stored scope list allows required. Keys created before scopes
// were validated may hold "*" or "<resource>:*"; those still match.
func Grants(granted, required string) bool {
	for _, s := range strings.Split(granted, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if s == required || s == "*" {
			return true
		}
		if strings.HasSuffix(s, ":*") && strings.HasPrefix(required, strings.TrimSuffix(s, "*")) {
			return true
		}
		for _, sub := range implied[s] {
			if sub == required {
				return true
			}
		}
	}
	return false
}
