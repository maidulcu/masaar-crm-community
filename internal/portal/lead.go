// Package portal reads leads pushed by property portals and writes the listing feed they
// pull.
//
// The portals' own lead and feed formats are not published (Bayut sends its XML
// documentation to agencies on request; Property Finder integrates through its Enterprise
// API). The inbound parser therefore accepts the common field names those payloads use,
// in any nesting, and the feed is Masaar's own documented XML. A portal-specific feed
// format can be added as another Format once its documentation is in hand.
package portal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// Portals that can push leads and pull the feed. "portal" is the generic catch-all.
var Portals = map[string]string{
	"property_finder": "Property Finder",
	"bayut":           "Bayut",
	"dubizzle":        "dubizzle",
	"portal":          "Other portal",
}

// Lead is the part of a portal lead Masaar uses.
type Lead struct {
	ExternalID       string
	Name             string
	Phone            string
	Email            string
	Message          string
	ListingReference string
}

// Candidate keys for each field, in priority order, compared case-insensitively.
var (
	idKeys      = []string{"lead_id", "leadid", "enquiry_id", "inquiry_id", "id"}
	nameKeys    = []string{"name", "full_name", "fullname", "client_name", "contact_name", "customer_name", "first_name"}
	phoneKeys   = []string{"phone", "mobile", "phone_number", "mobile_number", "client_phone", "contact_phone", "telephone", "whatsapp"}
	emailKeys   = []string{"email", "email_address", "client_email", "contact_email"}
	messageKeys = []string{"message", "comment", "comments", "enquiry", "inquiry", "notes", "body"}
	refKeys     = []string{"reference", "reference_number", "listing_reference", "property_reference", "property_ref",
		"listing_ref", "ref", "ref_no", "property_ref_no"}
)

// ErrNoContact is returned when a payload carries no phone number.
var ErrNoContact = errors.New("portal lead has no phone number")

// ParseLead extracts a lead from a JSON payload. Keys are searched at any depth (portals
// nest the contact under "lead", "contact", "data", …); the first match in priority
// order wins. Without an id in the payload, a hash of the body stands in, so a byte-for-
// byte redelivery is still recognised.
func ParseLead(body []byte) (*Lead, error) {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("portal lead: %w", err)
	}
	flat := map[string]string{}
	flatten(root, flat, 0)
	pick := func(keys []string) string {
		for _, k := range keys {
			if v := strings.TrimSpace(flat[k]); v != "" {
				return v
			}
		}
		return ""
	}
	l := &Lead{
		ExternalID:       pick(idKeys),
		Name:             pick(nameKeys),
		Phone:            pick(phoneKeys),
		Email:            pick(emailKeys),
		Message:          pick(messageKeys),
		ListingReference: pick(refKeys),
	}
	if last := strings.TrimSpace(flat["last_name"]); last != "" && flat["first_name"] != "" && l.Name == strings.TrimSpace(flat["first_name"]) {
		l.Name += " " + last
	}
	if l.ExternalID == "" {
		sum := sha256.Sum256(body)
		l.ExternalID = "sha256:" + hex.EncodeToString(sum[:])
	} else if len(l.ExternalID) > maxExternalIDLen {
		// Too long to index; its hash still identifies a redelivery of the same lead.
		sum := sha256.Sum256([]byte(l.ExternalID))
		l.ExternalID = "sha256:" + hex.EncodeToString(sum[:])
	}
	if l.Phone == "" {
		return l, ErrNoContact
	}
	if l.Name == "" {
		l.Name = l.Phone
	}
	// Cut by characters: cutting by bytes can split an Arabic letter, and Postgres then refuses
	// the whole lead as invalid UTF-8 (so the portal redelivers it forever).
	l.Message = truncateRunes(l.Message, 2000)
	l.Name = truncateRunes(l.Name, 200)
	l.ListingReference = truncateRunes(l.ListingReference, 128)
	// The email is optional: one that isn't a plain address is dropped rather than stored.
	if addr, err := mail.ParseAddress(l.Email); err != nil || addr.Address != l.Email || len(l.Email) > 254 {
		l.Email = ""
	}
	return l, nil
}

// maxExternalIDLen keeps a portal's lead id small enough for the external_ref index.
const maxExternalIDLen = 128

// truncateRunes returns s cut to at most n characters, never inside one.
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// flatten records scalar values by lower-cased key; the shallowest occurrence of a key
// wins. Depth is bounded.
func flatten(v any, out map[string]string, depth int) {
	if depth > 6 {
		return
	}
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	// Scalars at this level first, so they win over deeper ones.
	for k, val := range m {
		key := strings.ToLower(k)
		if _, seen := out[key]; seen {
			continue
		}
		switch x := val.(type) {
		case string:
			out[key] = x
		case float64:
			out[key] = strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", x), "0"), ".")
		}
	}
	for _, val := range m {
		switch x := val.(type) {
		case map[string]any:
			flatten(x, out, depth+1)
		case []any:
			for _, item := range x {
				flatten(item, out, depth+1)
			}
		}
	}
}
