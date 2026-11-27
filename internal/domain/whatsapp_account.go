package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WAProvider is where a WhatsApp account's messages go through.
type WAProvider string

const (
	// WAProviderCloudAPI talks to Meta directly with the deployment's own app credentials.
	WAProviderCloudAPI WAProvider = "cloud_api"
	// WAProviderBSP talks to a Meta Business Solution Provider on the company's behalf.
	WAProviderBSP WAProvider = "bsp"
	// WAProviderWAHA is the optional, unsupported community gateway.
	WAProviderWAHA WAProvider = "waha"
)

// WAAccountStatus is the lifecycle state of a WhatsApp account.
type WAAccountStatus string

const (
	WAAccountPending   WAAccountStatus = "pending"
	WAAccountLinking   WAAccountStatus = "linking"
	WAAccountConnected WAAccountStatus = "connected"
	WAAccountDegraded  WAAccountStatus = "degraded"
	WAAccountLoggedOut WAAccountStatus = "logged_out"
	WAAccountBanned    WAAccountStatus = "banned"
)

// WABillingMode records who pays Meta for the account's message fees. It is informational:
// the authoritative billing relationship lives in the partner contract.
type WABillingMode string

const (
	WABillingClientCard       WABillingMode = "client_card"
	WABillingMasaarCreditLine WABillingMode = "masaar_credit_line"
)

// WAAccount is one WhatsApp connection owned by a company. The account, not the legacy
// phone_number_id, is the identity threads and messages hang off.
type WAAccount struct {
	ID        uuid.UUID       `json:"id"`
	CompanyID uuid.UUID       `json:"company_id"`
	Provider  WAProvider      `json:"provider"`
	Status    WAAccountStatus `json:"status"`

	// Partner routing (provider='bsp').
	BSPSlug      string `json:"bsp_slug,omitempty"`
	BSPClientRef string `json:"bsp_client_ref,omitempty"`

	PhoneE164   string `json:"phone_e164,omitempty"`
	DisplayName string `json:"display_name,omitempty"`

	// Meta identifiers. The account model is mid-migration, so all of them are kept.
	WABAID             string `json:"waba_id,omitempty"`              // Messaging account ID under the 2026 model
	MessagingAccountID string `json:"messaging_account_id,omitempty"` // required for multi-account sends from H1 2027
	WAACID             string `json:"waac_id,omitempty"`              // replaces phone_number_id in paths by H1 2028
	PhoneNumberID      string `json:"phone_number_id,omitempty"`      // legacy; not the account key

	// Provider-side identifiers.
	ExternalID string `json:"external_id,omitempty"`
	SessionRef string `json:"session_ref,omitempty"`

	BillingMode WABillingMode   `json:"billing_mode,omitempty"`
	LinkedBy    *uuid.UUID      `json:"linked_by,omitempty"`
	LinkedAt    *time.Time      `json:"linked_at,omitempty"`
	LastSeenAt  *time.Time      `json:"last_seen_at,omitempty"`
	LastError   string          `json:"last_error,omitempty"`
	RiskFlags   json.RawMessage `json:"risk_flags,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsOfficial reports whether the account goes through Meta's official platform rather than an
// unofficial linked-device gateway.
func (a *WAAccount) IsOfficial() bool {
	return a != nil && (a.Provider == WAProviderCloudAPI || a.Provider == WAProviderBSP)
}
