package domain

import (
	"time"

	"github.com/google/uuid"
)

// ─── Audit Log ───────────────────────────────────────────────────────────────

const (
	AuditEntityContact = "contact"
	AuditEntityLead    = "lead"
	AuditEntityDeal    = "deal"
	AuditEntityInvoice = "invoice"
)

// ─── Company ──────────────────────────────────────────────────────────────────

type Company struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Subdomain        string     `json:"subdomain,omitempty"`
	Plan             string     `json:"plan"`
	TrialStartedAt   *time.Time `json:"trial_started_at,omitempty"`
	TrialEndsAt      *time.Time `json:"trial_ends_at,omitempty"`
	OnTrial          bool       `json:"on_trial"`
	IsActive         bool       `json:"is_active"`
	IsDemo           bool       `json:"is_demo"`
	StripeCustomerID string     `json:"stripe_customer_id,omitempty"`
	StripeSubID      string     `json:"stripe_sub_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ─── User ────────────────────────────────────────────────────────────────────

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleAgent  Role = "agent"
	RoleViewer Role = "viewer"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	CompanyID    uuid.UUID `json:"company_id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	LangPref     string    `json:"lang_pref"` // "ar" | "en"
	WANumber     string    `json:"wa_number"`
	Phone        string    `json:"phone"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// ─── Contact ─────────────────────────────────────────────────────────────────

type Contact struct {
	ID         uuid.UUID  `json:"id"`
	PhoneWA    string     `json:"phone_wa"`
	FullName   string     `json:"full_name"`
	Email      string     `json:"email"`
	Language   string     `json:"language"` // "ar" | "en"
	LeadScore  int        `json:"lead_score"`
	AssignedTo *uuid.UUID `json:"assigned_to"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ─── Pipeline Stage ─────────────────────────────────────────────────────────────

type PipelineStage struct {
	ID         uuid.UUID `json:"id"`
	CompanyID  uuid.UUID `json:"company_id"`
	EntityType string    `json:"entity_type"`
	Name       string    `json:"name"`
	SortOrder  int       `json:"sort_order"`
	Color      string    `json:"color"`
	IsWon      bool      `json:"is_won"`
	IsLost     bool      `json:"is_lost"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
}

// ─── Lead ─────────────────────────────────────────────────────────────────────

type LeadStage string

const (
	StageNew       LeadStage = "new"
	StageContacted LeadStage = "contacted"
	StageQualified LeadStage = "qualified"
	StageProposal  LeadStage = "proposal"
	StageWon       LeadStage = "won"
	StageLost      LeadStage = "lost"
)

type LeadSource string

const (
	SourceWhatsApp LeadSource = "whatsapp"
	SourceWeb      LeadSource = "web"
	SourceReferral LeadSource = "referral"
	SourceEvent    LeadSource = "event"
)

type Lead struct {
	ID             uuid.UUID  `json:"id"`
	ContactID      uuid.UUID  `json:"contact_id"`
	Stage          LeadStage  `json:"stage"`
	Source         LeadSource `json:"source"`
	DealValue      float64    `json:"deal_value"`
	Currency       string     `json:"currency"` // default: AED
	Notes          string     `json:"notes"`
	LeadScore      int        `json:"lead_score"`
	ScoreUpdatedAt *time.Time `json:"score_updated_at"`
	AssignedTo     *uuid.UUID `json:"assigned_to,omitempty"`
	ClosedReason   string     `json:"closed_reason,omitempty"`
	LastContactedAt *time.Time `json:"last_contacted_at,omitempty"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Joined
	Contact    *Contact `json:"contact,omitempty"`
	AssignedUser *User  `json:"assigned_user,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// ─── WhatsApp ─────────────────────────────────────────────────────────────────

type ThreadStatus string

const (
	ThreadOpen    ThreadStatus = "open"
	ThreadClosed  ThreadStatus = "closed"
	ThreadPending ThreadStatus = "pending"
)

type WhatsAppThread struct {
	ID            uuid.UUID    `json:"id"`
	ContactID     uuid.UUID    `json:"contact_id"`
	WAAccountID   string       `json:"wa_account_id"`
	ThreadStatus  ThreadStatus `json:"thread_status"`
	LastMessageAt *time.Time   `json:"last_message_at"`
	MessageCount  int          `json:"message_count"`
	AISummary     string       `json:"ai_summary"`
	CreatedAt     time.Time    `json:"created_at"`

	// Joined
	Contact  *Contact          `json:"contact,omitempty"`
	Messages []WhatsAppMessage `json:"messages,omitempty"`
}

type MessageDirection string

const (
	DirectionInbound  MessageDirection = "inbound"
	DirectionOutbound MessageDirection = "outbound"
)

type WhatsAppMessage struct {
	ID          uuid.UUID        `json:"id"`
	ThreadID    uuid.UUID        `json:"thread_id"`
	Direction   MessageDirection `json:"direction"`
	Body        string           `json:"body"`
	MediaURL    string           `json:"media_url"`
	WAMessageID string           `json:"wa_message_id"`
	SentAt      time.Time        `json:"sent_at"`
}

type OutboundStatus string

const (
	OutboundPending   OutboundStatus = "pending"
	OutboundSent      OutboundStatus = "sent"
	OutboundDelivered OutboundStatus = "delivered"
	OutboundRead      OutboundStatus = "read"
	OutboundFailed    OutboundStatus = "failed"
)

type WhatsAppOutbound struct {
	ID            int64          `json:"id"`
	ThreadID      uuid.UUID      `json:"thread_id"`
	ToNumber      string         `json:"to_number"`
	MessageBody   string         `json:"message_body"`
	MediaURL      string         `json:"media_url"`
	WAMessageID   string         `json:"wa_message_id"`
	Status        OutboundStatus `json:"status"`
	ErrorMsg      string         `json:"error_message"`
	ScheduledAt   *time.Time     `json:"scheduled_at"`
	SentAt        *time.Time     `json:"sent_at"`
	CreatedAt     time.Time      `json:"created_at"`
	CreatedBy     *uuid.UUID     `json:"created_by"`
	Metadata      map[string]any `json:"metadata"`
}

// ─── Deal ─────────────────────────────────────────────────────────────────────

type DealStage string

const (
	DealStageOpen DealStage = "open"
	DealStageWon  DealStage = "won"
	DealStageLost DealStage = "lost"
)

type Deal struct {
	ID          uuid.UUID  `json:"id"`
	LeadID      uuid.UUID  `json:"lead_id"`
	Title       string     `json:"title"`
	Stage       DealStage  `json:"stage"`
	Amount      float64    `json:"amount"`
	Currency    string     `json:"currency"`
	CloseDate   *time.Time `json:"close_date"`
	Probability int        `json:"probability"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ─── VAT Invoice ──────────────────────────────────────────────────────────────

type InvoiceStatus string

const (
	InvoiceDraft InvoiceStatus = "draft"
	InvoiceSent  InvoiceStatus = "sent"
	InvoicePaid  InvoiceStatus = "paid"
)

type VATInvoice struct {
	ID        uuid.UUID     `json:"id"`
	DealID    uuid.UUID     `json:"deal_id"`
	InvoiceNo string        `json:"invoice_no"`
	Subtotal  float64       `json:"subtotal"`
	VATRate   float64       `json:"vat_rate"`
	VATAmount float64       `json:"vat_amount"`
	Total     float64       `json:"total"`
	QRPayload string        `json:"qr_payload"`
	Status    InvoiceStatus `json:"status"`
	IssuedAt  time.Time     `json:"issued_at"`
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

type AuditLog struct {
	ID         int64      `json:"id"`
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
	Action     string     `json:"action"`
	ActorID    *uuid.UUID `json:"actor_id"`
	Diff       any        `json:"diff"`
	Timestamp  time.Time  `json:"ts"`
}

// ─── Pagination ───────────────────────────────────────────────────────────────

type PaginatedResult[T any] struct {
	Data  []T `json:"data"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// ─── Notification ────────────────────────────────────────────────────────────

type Notification struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Type      string    `json:"type"` // "lead_assigned", "lead_stage_change", "new_message"
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Read      bool      `json:"read"`
	Data      string    `json:"data,omitempty"` // JSON payload for navigation
	CreatedAt time.Time `json:"created_at"`
}

// ─── Stats ────────────────────────────────────────────────────────────────────

type Stats struct {
	TotalContacts  int     `json:"total_contacts"`
	ActiveLeads    int     `json:"active_leads"`
	NewLeadsWeek   int     `json:"new_leads_week"`
	OpenThreads    int     `json:"open_threads"`
	OpenDeals      int     `json:"open_deals"`
	OpenDealsValue float64 `json:"open_deals_value"`
	WonDeals       int     `json:"won_deals"`
	WonDealsValue  float64 `json:"won_deals_value"`
}

// ─── API Settings ─────────────────────────────────────────────────────────────

type APISetting struct {
	ID           uuid.UUID  `json:"id"`
	SettingKey   string     `json:"setting_key"`
	SettingValue string     `json:"setting_value"`
	Description  string     `json:"description"`
	UpdatedAt    time.Time  `json:"updated_at"`
	UpdatedBy    *uuid.UUID `json:"updated_by"`
}

// ─── Offers ──────────────────────────────────────────────────────────────────

type OfferStatus string

const (
	OfferSubmitted   OfferStatus = "submitted"
	OfferUnderReview OfferStatus = "under_review"
	OfferCountered   OfferStatus = "countered"
	OfferAccepted    OfferStatus = "accepted"
	OfferRejected    OfferStatus = "rejected"
	OfferExpired     OfferStatus = "expired"
)

type Offer struct {
	ID            uuid.UUID   `json:"id"`
	ListingID     uuid.UUID   `json:"listing_id"`
	ContactID     uuid.UUID   `json:"contact_id"`
	AgentID       *uuid.UUID  `json:"agent_id"`
	ParentOfferID *uuid.UUID  `json:"parent_offer_id"`
	OfferAmount   float64     `json:"offer_amount"`
	Currency      string      `json:"currency"`
	Status        OfferStatus `json:"status"`
	Terms         string      `json:"terms"`
	Notes         string      `json:"notes"`
	ValidUntil    *time.Time  `json:"valid_until"`
	DealID        *uuid.UUID  `json:"deal_id"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`

	// Joined fields
	Contact        *Contact  `json:"contact,omitempty"`
	ListingTitle   string    `json:"listing_title,omitempty"`
	CounterOffers  []Offer   `json:"counter_offers,omitempty"`
}

// ─── Approval Workflows ───────────────────────────────────────────────────────

type ApprovalStatus string
const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type ApprovalEntity string
const (
	ApprovalListing ApprovalEntity = "listing"
	ApprovalDeal    ApprovalEntity = "deal"
	ApprovalOffer   ApprovalEntity = "offer"
)

type ApprovalConfig struct {
	ID                 uuid.UUID `json:"id"`
	CompanyID          uuid.UUID `json:"company_id"`
	ListingApproval    bool      `json:"listing_approval"`
	DealApprovalAbove  float64   `json:"deal_approval_above"`
	OfferApprovalAbove float64   `json:"offer_approval_above"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ApprovalRequest struct {
	ID           uuid.UUID      `json:"id"`
	CompanyID    uuid.UUID      `json:"company_id"`
	EntityType   ApprovalEntity `json:"entity_type"`
	EntityID     uuid.UUID      `json:"entity_id"`
	RequestedBy  uuid.UUID      `json:"requested_by"`
	ReviewedBy   *uuid.UUID     `json:"reviewed_by"`
	Status       ApprovalStatus `json:"status"`
	Notes        string         `json:"notes"`
	ReviewerNote string         `json:"reviewer_note"`
	CreatedAt    time.Time      `json:"created_at"`
	ReviewedAt   *time.Time     `json:"reviewed_at"`
	// Joined
	RequesterName string `json:"requester_name,omitempty"`
	ReviewerName  string `json:"reviewer_name,omitempty"`
}

// ─── Viewings / Calendar ─────────────────────────────────────────────────────

type ViewingStatus string

const (
	ViewingScheduled  ViewingStatus = "scheduled"
	ViewingConfirmed  ViewingStatus = "confirmed"
	ViewingCheckedIn  ViewingStatus = "checked_in"
	ViewingCompleted  ViewingStatus = "completed"
	ViewingCancelled  ViewingStatus = "cancelled"
	ViewingNoShow     ViewingStatus = "no_show"
)

type Viewing struct {
	ID            uuid.UUID     `json:"id"`
	ListingID     *uuid.UUID    `json:"listing_id"`
	ContactID     uuid.UUID     `json:"contact_id"`
	AgentID       *uuid.UUID    `json:"agent_id"`
	LeadID        *uuid.UUID    `json:"lead_id"`
	ScheduledAt   time.Time     `json:"scheduled_at"`
	DurationMin   int           `json:"duration_min"`
	Status        ViewingStatus `json:"status"`
	Address       string        `json:"address"`
	Notes         string        `json:"notes"`
	CheckedInAt   *time.Time    `json:"checked_in_at"`
	CheckedOutAt  *time.Time    `json:"checked_out_at"`
	ReminderSent  bool          `json:"reminder_sent"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`

	// Joined
	Contact      *Contact  `json:"contact,omitempty"`
	ListingTitle string    `json:"listing_title,omitempty"`
	AgentName    string    `json:"agent_name,omitempty"`
}

// ─── Lead Rotation ───────────────────────────────────────────────────────────

// LeadRotationSettings controls how new leads are distributed across agents.
type LeadRotationSettings struct {
	ID            uuid.UUID `json:"id"`
	CompanyID     uuid.UUID `json:"company_id"`
	Mode          string    `json:"mode"`           // manual | round_robin | capacity
	Enabled       bool      `json:"enabled"`
	RotationIndex int       `json:"rotation_index"` // internal counter
	MaxPerAgent   int       `json:"max_per_agent"`  // 0 = unlimited
	UpdatedAt     time.Time `json:"updated_at"`
}

// ─── BOS24 Integration ───────────────────────────────────────────────────────

// BOS24Settings holds per-company BOS24 integration configuration.
type BOS24Settings struct {
	ID            uuid.UUID  `json:"id"`
	CompanyID     uuid.UUID  `json:"company_id"`
	APIKey        string     `json:"api_key"`         // bos24_live_xxx — integration API key
	WebhookSecret string     `json:"webhook_secret"`  // HMAC secret + URL token
	WebhookID     string     `json:"webhook_id"`      // ID returned by BOS24 on registration
	LastSyncAt    *time.Time `json:"last_sync_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// IsConfigured returns true when an API key has been set.
func (s *BOS24Settings) IsConfigured() bool {
	return s != nil && s.APIKey != ""
}

// IsWebhookRegistered returns true when BOS24 has acknowledged our webhook.
func (s *BOS24Settings) IsWebhookRegistered() bool {
	return s != nil && s.WebhookID != ""
}

// ─── Email ────────────────────────────────────────────────────────────────────

type EmailStatus string

const (
	EmailPending  EmailStatus = "pending"
	EmailSent     EmailStatus = "sent"
	EmailFailed   EmailStatus = "failed"
	EmailBounced  EmailStatus = "bounced"
)

type EmailHistory struct {
	ID          int64          `json:"id"`
	FromEmail   string         `json:"from_email"`
	ToEmail     string         `json:"to_email"`
	Subject     string         `json:"subject"`
	Body        string         `json:"body"`
	HTMLBody    string         `json:"html_body"`
	Status      EmailStatus    `json:"status"`
	ErrorMsg    string         `json:"error_message"`
	RelatedTo   string         `json:"related_to"`   // invoice, proposal, followup
	RelatedID   *int64         `json:"related_id"`
	SentAt      *time.Time     `json:"sent_at"`
	CreatedAt   time.Time      `json:"created_at"`
	CreatedBy   *uuid.UUID     `json:"created_by"`
	Metadata    map[string]any `json:"metadata"`
}

// ─── Lead Tags ────────────────────────────────────────────────────────────────

type LeadTag struct {
	ID          int64      `json:"id"`
	LeadID      uuid.UUID  `json:"lead_id"`
	Tag         string     `json:"tag"`
	Category    string     `json:"category"` // segment, quality, interest, timeline, status
	AutoApplied bool       `json:"auto_applied"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   *uuid.UUID `json:"created_by"`
}

// ─── Communication History ─────────────────────────────────────────────────────

type CommunicationType string

const (
	CommWhatsAppInbound  CommunicationType = "whatsapp_inbound"
	CommWhatsAppOutbound CommunicationType = "whatsapp_outbound"
	CommEmailSent        CommunicationType = "email_sent"
	CommEmailReceived    CommunicationType = "email_received"
	CommCall             CommunicationType = "call"
)

type CommunicationHistory struct {
	ID                int64              `json:"id"`
	LeadID            uuid.UUID          `json:"lead_id"`
	ContactID         uuid.UUID          `json:"contact_id"`
	CommunicationType CommunicationType  `json:"communication_type"`
	Direction         string             `json:"direction"` // inbound, outbound
	Body              string             `json:"body"`
	FromIdentifier    string             `json:"from_identifier"`
	ToIdentifier      string             `json:"to_identifier"`
	ExternalID        string             `json:"external_id"`
	Status            string             `json:"status"`
	Metadata          map[string]any     `json:"metadata"`
	CreatedAt         time.Time          `json:"created_at"`
	CreatedBy         *uuid.UUID         `json:"created_by"`
}

// ─── Company Settings ─────────────────────────────────────────────────────────

type CompanySettings struct {
	ID              int        `json:"id"`
	Name            string     `json:"name"`
	VATNumber       string     `json:"vat_number"`
	BusinessAddress string     `json:"business_address"`
	BusinessPhone   string     `json:"business_phone"`
	BusinessEmail   string     `json:"business_email"`
	BankName        string     `json:"bank_name"`
	BankAccount     string     `json:"bank_account"`
	BankIBAN        string     `json:"bank_iban"`
	LogoURL         string     `json:"logo_url"`
	Disclaimer      string     `json:"disclaimer"`
	PrimaryColor    string     `json:"primary_color"`
	UpdatedAt       time.Time  `json:"updated_at"`
	UpdatedBy       *uuid.UUID `json:"updated_by"`
}

// ─── Rental Properties ────────────────────────────────────────────────────────

type PropertyType string

const (
	PropertyTypeVilla       PropertyType = "villa"
	PropertyTypeApartment   PropertyType = "apartment"
	PropertyTypeTownhouse   PropertyType = "townhouse"
	PropertyTypeCommercial  PropertyType = "commercial"
)

type PropertyStatus string

const (
	PropertyStatusActive      PropertyStatus = "active"
	PropertyStatusInactive    PropertyStatus = "inactive"
	PropertyStatusSold        PropertyStatus = "sold"
	PropertyStatusMaintenance PropertyStatus = "maintenance"
)

type OccupancyStatus string

const (
	OccupancyVacant      OccupancyStatus = "vacant"
	OccupancyOccupied    OccupancyStatus = "occupied"
	OccupancyMaintenance OccupancyStatus = "maintenance"
)

type RentalProperty struct {
	ID                  uuid.UUID       `json:"id"`
	CompanyID           uuid.UUID       `json:"company_id"`
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	PropertyType        PropertyType    `json:"property_type"`
	UnitsCount          int             `json:"units_count"`
	Area                string          `json:"area"`
	StreetAddress       string          `json:"street_address"`
	BuildingNumber      string          `json:"building_number"`
	UnitNumber          string          `json:"unit_number"`
	City                string          `json:"city"`
	Emirate             string          `json:"emirate"`
	PostalCode          string          `json:"postal_code"`
	TotalSqft           float64         `json:"total_sqft"`
	Bedrooms            int             `json:"bedrooms"`
	Bathrooms           int             `json:"bathrooms"`
	ParkingSpaces       int             `json:"parking_spaces"`
	Amenities           []string        `json:"amenities"`
	PurchasePrice       float64         `json:"purchase_price"`
	PurchaseDate        *time.Time      `json:"purchase_date"`
	MarketValue         float64         `json:"market_value"`
	Currency            string          `json:"currency"`
	Status              PropertyStatus  `json:"status"`
	OccupancyStatus     OccupancyStatus `json:"occupancy_status"`
	TotalOccupiedUnits  int             `json:"total_occupied_units"`
	PropertyDeedURL     string          `json:"property_deed_url"`
	TitleDeedNumber     string          `json:"title_deed_number"`
	MunicipalityRegNum  string          `json:"municipality_registration"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
	CreatedBy           *uuid.UUID      `json:"created_by"`
	UpdatedBy           *uuid.UUID      `json:"updated_by"`
}

// ─── Listings ───────────────────────────────────────────────────────────────────

type ListingType string

const (
	ListingTypeSale ListingType = "sale"
	ListingTypeRent ListingType = "rent"
)

type ListingStatus string

const (
	ListingStatusDraft     ListingStatus = "draft"
	ListingStatusPublished ListingStatus = "published"
	ListingStatusSold      ListingStatus = "sold"
	ListingStatusRented    ListingStatus = "rented"
	ListingStatusExpired   ListingStatus = "expired"
	ListingStatusWithdrawn ListingStatus = "withdrawn"
)

type Listing struct {
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`

	Title           string       `json:"title"`
	Description     string       `json:"description"`
	PropertyType    PropertyType `json:"property_type"`
	ListingType     ListingType  `json:"listing_type"`

	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	RentPeriod  *string `json:"rent_period"`

	Area         string  `json:"area"`
	Community    string  `json:"community"`
	Subcommunity string  `json:"subcommunity"`
	City         string  `json:"city"`
	Emirate      string  `json:"emirate"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`

	Bedrooms      int      `json:"bedrooms"`
	Bathrooms     int      `json:"bathrooms"`
	TotalSqft     float64  `json:"total_sqft"`
	PlotSqft      float64  `json:"plot_sqft"`
	ParkingSpaces int      `json:"parking_spaces"`
	Furnishing    *string  `json:"furnishing"`
	Amenities     []string `json:"amenities"`
	YearBuilt     *int     `json:"year_built"`

	CoverImageURL  string   `json:"cover_image_url"`
	ImageURLs      []string `json:"image_urls"`
	VirtualTourURL string   `json:"virtual_tour_url"`
	VideoURL       string   `json:"video_url"`

	Status          ListingStatus `json:"status"`
	Featured        bool          `json:"featured"`
	ReferenceNumber string        `json:"reference_number"`
	AvailableFrom   *time.Time    `json:"available_from"`

	AssignedTo *uuid.UUID `json:"assigned_to"`

	OwnerName  string `json:"owner_name"`
	OwnerPhone string `json:"owner_phone"`
	OwnerEmail string `json:"owner_email"`

	PortalSyncStatus map[string]interface{} `json:"portal_sync_status"`

	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   *uuid.UUID `json:"created_by"`
	UpdatedBy   *uuid.UUID `json:"updated_by"`
}

// ─── Tenants ──────────────────────────────────────────────────────────────────

type IDType string

const (
	IDTypeEmiratiID     IDType = "emirati_id"
	IDTypePassport      IDType = "passport"
	IDTypeDrivingLicense IDType = "driving_license"
	IDTypeTradeLicense  IDType = "trade_license"
)

type EmploymentStatus string

const (
	EmploymentEmployed      EmploymentStatus = "employed"
	EmploymentSelfEmployed  EmploymentStatus = "self_employed"
	EmploymentRetired       EmploymentStatus = "retired"
	EmploymentStudent       EmploymentStatus = "student"
)

type TenantStatus string

const (
	TenantStatusActive      TenantStatus = "active"
	TenantStatusInactive    TenantStatus = "inactive"
	TenantStatusBlacklisted TenantStatus = "blacklisted"
)

type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationVerified VerificationStatus = "verified"
	VerificationRejected VerificationStatus = "rejected"
)

type Tenant struct {
	ID                      uuid.UUID          `json:"id"`
	CompanyID               uuid.UUID          `json:"company_id"`
	FullNameEN              string             `json:"full_name_en"`
	FullNameAR              string             `json:"full_name_ar"`
	Email                   string             `json:"email"`
	Phone                   string             `json:"phone"`
	PhoneWA                 string             `json:"phone_wa"`
	IDType                  IDType             `json:"id_type"`
	IDNumber                string             `json:"id_number"`
	IDExpiryDate            *time.Time         `json:"id_expiry_date"`
	IDDocumentURL           string             `json:"id_document_url"`
	IsVerified              bool               `json:"is_verified"`
	VerificationStatus      VerificationStatus `json:"verification_status"`
	VerificationDate        *time.Time         `json:"verification_date"`
	VerifiedBy              *uuid.UUID         `json:"verified_by"`
	VerificationNotes       string             `json:"verification_notes"`
	EmploymentStatus        EmploymentStatus   `json:"employment_status"`
	EmployerName            string             `json:"employer_name"`
	AnnualIncome            float64            `json:"annual_income"`
	IncomeCurrency          string             `json:"income_currency"`
	SalaryCertificateURL    string             `json:"salary_certificate_url"`
	Nationality             string             `json:"nationality"`
	CountryOfOrigin         string             `json:"country_of_origin"`
	PermanentAddress        string             `json:"permanent_address"`
	EmergencyContactName    string             `json:"emergency_contact_name"`
	EmergencyContactPhone   string             `json:"emergency_contact_phone"`
	Status                  TenantStatus       `json:"status"`
	Notes                   string             `json:"notes"`
	CreatedAt               time.Time          `json:"created_at"`
	UpdatedAt               time.Time          `json:"updated_at"`
	CreatedBy               *uuid.UUID         `json:"created_by"`
	UpdatedBy               *uuid.UUID         `json:"updated_by"`
}

// ─── Lease Management ──────────────────────────────────────────────────────

type PaymentFrequency string

const (
	FrequencyMonthly      PaymentFrequency = "monthly"
	FrequencyQuarterly    PaymentFrequency = "quarterly"
	FrequencySemiAnnual   PaymentFrequency = "semi_annual"
	FrequencyAnnual       PaymentFrequency = "annual"
)

type LeaseStatus string

const (
	LeaseStatusActive      LeaseStatus = "active"
	LeaseStatusRenewed     LeaseStatus = "renewed"
	LeaseStatusTerminated  LeaseStatus = "terminated"
	LeaseStatusExpired     LeaseStatus = "expired"
)

type LeaseTemplate struct {
	ID                          uuid.UUID         `json:"id"`
	CompanyID                   uuid.UUID         `json:"company_id"`
	Name                        string            `json:"name"`
	Description                 string            `json:"description"`
	IsDefault                   bool              `json:"is_default"`
	PaymentFrequency            PaymentFrequency  `json:"payment_frequency"`
	PaymentDayOfMonth           int               `json:"payment_day_of_month"`
	AutoGeneratePayments        bool              `json:"auto_generate_payments"`
	DefaultSecurityDepositPct   float64           `json:"default_security_deposit_percent"`
	DefaultUtilityCharges       float64           `json:"default_utility_charges"`
	DefaultLateFeePercent       float64           `json:"default_late_fee_percent"`
	DefaultLeaseDurationMonths  int               `json:"default_lease_duration_months"`
	DefaultNoticePeriodDays     int               `json:"default_notice_period_days"`
	DefaultRenewalDurationMonths int              `json:"default_renewal_duration_months"`
	TemplateDocumentURL         string            `json:"template_document_url"`
	TermsConditions             string            `json:"terms_conditions"`
	Status                      string            `json:"status"`
	CreatedAt                   time.Time         `json:"created_at"`
	UpdatedAt                   time.Time         `json:"updated_at"`
	CreatedBy                   *uuid.UUID        `json:"created_by"`
	UpdatedBy                   *uuid.UUID        `json:"updated_by"`
}

type Lease struct {
	ID                      uuid.UUID         `json:"id"`
	CompanyID               uuid.UUID         `json:"company_id"`
	PropertyID              uuid.UUID         `json:"property_id"`
	TenantID                uuid.UUID         `json:"tenant_id"`
	TemplateID              *uuid.UUID        `json:"template_id"`
	StartDate               time.Time         `json:"start_date"`
	EndDate                 time.Time         `json:"end_date"`
	RenewalStartDate        *time.Time        `json:"renewal_start_date"`
	RenewalEndDate          *time.Time        `json:"renewal_end_date"`
	MonthlyRent             float64           `json:"monthly_rent"`
	Currency                string            `json:"currency"`
	SecurityDeposit         float64           `json:"security_deposit"`
	UtilityCharges          float64           `json:"utility_charges"`
	LateFeePct              float64           `json:"late_fee_percent"`
	PaymentFrequency        PaymentFrequency  `json:"payment_frequency"`
	PaymentDayOfMonth       int               `json:"payment_day_of_month"`
	AutoGeneratePayments    bool              `json:"auto_generate_payments"`
	LastGeneratedPaymentDt  *time.Time        `json:"last_generated_payment_date"`
	NoticePeriodDays        int               `json:"notice_period_days"`
	MoveOutDate             *time.Time        `json:"move_out_date"`
	MoveOutInspectionDate   *time.Time        `json:"move_out_inspection_date"`
	LeaseDocumentURL        string            `json:"lease_document_url"`
	SignedByLandlordDate    *time.Time        `json:"signed_by_landlord_date"`
	SignedByTenantDate      *time.Time        `json:"signed_by_tenant_date"`
	EjariNumber             string            `json:"ejari_number"`
	EjariURL                string            `json:"ejari_url"`
	Status                  LeaseStatus       `json:"status"`
	TerminationReason       string            `json:"termination_reason"`
	TerminationDate         *time.Time        `json:"termination_date"`
	Notes                   string            `json:"notes"`
	CreatedAt               time.Time         `json:"created_at"`
	UpdatedAt               time.Time         `json:"updated_at"`
	CreatedBy               *uuid.UUID        `json:"created_by"`
	UpdatedBy               *uuid.UUID        `json:"updated_by"`

	// Joined
	Property *RentalProperty `json:"property,omitempty"`
	Tenant   *Tenant         `json:"tenant,omitempty"`
	Template *LeaseTemplate  `json:"template,omitempty"`
}

// ─── Payment Management ────────────────────────────────────────────────────────

type PaymentMethod string

const (
	MethodTransfer PaymentMethod = "transfer"
	MethodCheck    PaymentMethod = "check"
	MethodCash     PaymentMethod = "cash"
	MethodCard     PaymentMethod = "card"
	MethodOther    PaymentMethod = "other"
)

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentReceived  PaymentStatus = "received"
	PaymentOverdue   PaymentStatus = "overdue"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
)

type Payment struct {
	ID                  uuid.UUID      `json:"id"`
	CompanyID           uuid.UUID      `json:"company_id"`
	LeaseID             uuid.UUID      `json:"lease_id"`
	Amount              float64        `json:"amount"`
	Currency            string         `json:"currency"`
	DueDate             time.Time      `json:"due_date"`
	PaidDate            *time.Time     `json:"paid_date"`
	PaymentMethod       PaymentMethod  `json:"payment_method"`
	PaymentReference    string         `json:"payment_reference"`
	Status              PaymentStatus  `json:"status"`
	BankTransactionID   *uuid.UUID     `json:"bank_transaction_id"`
	ReconciledAt        *time.Time     `json:"reconciled_at"`
	ReconciledBy        *uuid.UUID     `json:"reconciled_by"`
	Notes               string         `json:"notes"`
	ReceiptURL          string         `json:"receipt_url"`
	LateFeesApplied     bool           `json:"late_fee_applied"`
	LateFeeAmount       float64        `json:"late_fee_amount"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	CreatedBy           *uuid.UUID     `json:"created_by"`
	UpdatedBy           *uuid.UUID     `json:"updated_by"`

	// Joined
	Lease *Lease `json:"lease,omitempty"`
}

type TransactionType string

const (
	TypeCredit    TransactionType = "credit"
	TypeDebit     TransactionType = "debit"
	TypeTransfer  TransactionType = "transfer"
	TypeCheck     TransactionType = "check"
)

type BankTransaction struct {
	ID                  uuid.UUID        `json:"id"`
	CompanyID           uuid.UUID        `json:"company_id"`
	BankIntegrationID   *uuid.UUID       `json:"bank_integration_id"`
	ExternalID          string           `json:"external_id"`
	TransactionDate     time.Time        `json:"transaction_date"`
	Amount              float64          `json:"amount"`
	Currency            string           `json:"currency"`
	FromAccount         string           `json:"from_account"`
	ToAccount           string           `json:"to_account"`
	FromName            string           `json:"from_name"`
	ToName              string           `json:"to_name"`
	Reference           string           `json:"reference"`
	TransactionType     TransactionType  `json:"transaction_type"`
	Status              string           `json:"status"`
	MatchedPaymentID    *uuid.UUID       `json:"matched_payment_id"`
	MatchConfidence     float64          `json:"match_confidence"`
	MatchedAt           *time.Time       `json:"matched_at"`
	ImportedAt          time.Time        `json:"imported_at"`
	LastChecked         *time.Time       `json:"last_checked"`
	SyncError           string           `json:"sync_error"`
}

type BankIntegration struct {
	ID                   uuid.UUID `json:"id"`
	CompanyID            uuid.UUID `json:"company_id"`
	BankName             string    `json:"bank_name"`
	BankCode             string    `json:"bank_code"`
	AccountNumber        string    `json:"account_number"`
	AccountName          string    `json:"account_name"`
	IBAN                 string    `json:"iban"`
	IntegrationType      string    `json:"integration_type"`
	Status               string    `json:"status"`
	APIKeyEncrypted      string    `json:"-"`
	APISecretEncrypted   string    `json:"-"`
	APIEndpoint          string    `json:"api_endpoint"`
	AutoSync             bool      `json:"auto_sync"`
	LastSyncDate         *time.Time `json:"last_sync_date"`
	SyncIntervalHours    int       `json:"sync_interval_hours"`
	LastSyncError        string    `json:"last_sync_error"`
	SyncErrorCount       int       `json:"sync_error_count"`
	IsConnected          bool      `json:"is_connected"`
	ConnectionTestDate   *time.Time `json:"connection_test_date"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedBy            *uuid.UUID `json:"created_by"`
	UpdatedBy            *uuid.UUID `json:"updated_by"`
}

// ─── Payment Reminders ────────────────────────────────────────────────────────

type ReminderType string

const (
	Reminder30Days  ReminderType = "30_days"
	Reminder15Days  ReminderType = "15_days"
	Reminder7Days   ReminderType = "7_days"
	Reminder1DayLate ReminderType = "1_day_late"
	Reminder5DaysLate ReminderType = "5_days_late"
	Reminder10DaysLate ReminderType = "10_days_late"
)

type DeliveryMethod string

const (
	DeliveryWhatsApp DeliveryMethod = "whatsapp"
	DeliveryEmail    DeliveryMethod = "email"
	DeliverySMS      DeliveryMethod = "sms"
)

type DeliveryStatus string

const (
	DeliveryPending DeliveryStatus = "pending"
	DeliverySent    DeliveryStatus = "sent"
	DeliveryFailed  DeliveryStatus = "failed"
)

type PaymentReminder struct {
	ID             uuid.UUID      `json:"id"`
	CompanyID      uuid.UUID      `json:"company_id"`
	PaymentID      uuid.UUID      `json:"payment_id"`
	ReminderType   ReminderType   `json:"reminder_type"`
	ReminderDate   time.Time      `json:"reminder_date"`
	SentAt         *time.Time     `json:"sent_at"`
	DeliveryMethod DeliveryMethod `json:"delivery_method"`
	DeliveryStatus DeliveryStatus `json:"delivery_status"`
	DeliveryError  string         `json:"delivery_error"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// ─── Bank Statements & Payment Confirmations ───────────────────────────────

type FileFormat string

const (
	FormatCSV  FileFormat = "csv"
	FormatPDF  FileFormat = "pdf"
	FormatXLSX FileFormat = "xlsx"
)

type ProcessingStatus string

const (
	StatusPending    ProcessingStatus = "pending"
	StatusProcessing ProcessingStatus = "processing"
	StatusCompleted  ProcessingStatus = "completed"
	StatusFailed     ProcessingStatus = "failed"
)

type DataClassification string

const (
	ClassPublic      DataClassification = "public"
	ClassInternal    DataClassification = "internal"
	ClassConfidential DataClassification = "confidential"
)

type BankStatement struct {
	ID                  uuid.UUID          `json:"id"`
	CompanyID           uuid.UUID          `json:"company_id"`
	BankIntegrationID   uuid.UUID          `json:"bank_integration_id"`
	FileName            string             `json:"file_name"`
	FileSizeBytes       int                `json:"file_size_bytes"`
	FileURL             string             `json:"file_url"`
	FileFormat          FileFormat         `json:"file_format"`
	UploadedBy          uuid.UUID          `json:"uploaded_by"`
	UploadDate          time.Time          `json:"upload_date"`
	ProcessingStatus    ProcessingStatus   `json:"processing_status"`
	TransactionsImported int               `json:"transactions_imported"`
	ImportError         string             `json:"import_error"`
	DataClassification  DataClassification `json:"data_classification"`
	RetentionUntil      *time.Time         `json:"retention_until"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
	DeletedAt           *time.Time         `json:"deleted_at"`
}

type ConfirmationDeliveryStatus string

const (
	ConfPending  ConfirmationDeliveryStatus = "pending"
	ConfSent     ConfirmationDeliveryStatus = "sent"
	ConfFailed   ConfirmationDeliveryStatus = "failed"
	ConfBounced  ConfirmationDeliveryStatus = "bounced"
)

type ConfirmationDeliveryMethod string

const (
	ConfDeliveryEmail    ConfirmationDeliveryMethod = "email"
	ConfDeliveryWhatsApp ConfirmationDeliveryMethod = "whatsapp"
	ConfDeliverySMS      ConfirmationDeliveryMethod = "sms"
)

type PaymentConfirmation struct {
	ID                 uuid.UUID                      `json:"id"`
	CompanyID          uuid.UUID                      `json:"company_id"`
	PaymentID          uuid.UUID                      `json:"payment_id"`
	ConfirmationNumber string                         `json:"confirmation_number"`
	TenantEmail        string                         `json:"tenant_email"`
	TenantPhone        *string                        `json:"tenant_phone"`
	SentAt             *time.Time                     `json:"sent_at"`
	DeliveryStatus     ConfirmationDeliveryStatus     `json:"delivery_status"`
	DeliveryMethod     ConfirmationDeliveryMethod     `json:"delivery_method"`
	PDFURL             *string                        `json:"pdf_url"`
	DataClassification DataClassification            `json:"data_classification"`
	RetentionUntil     *time.Time                     `json:"retention_until"`
	CreatedAt          time.Time                      `json:"created_at"`
	UpdatedAt          time.Time                      `json:"updated_at"`
	DeletedAt          *time.Time                     `json:"deleted_at"`
}

// ─── Analytics ────────────────────────────────────────────────────────────────

type TenantAnalytics struct {
	TotalTenants        int     `json:"total_tenants"`
	ActiveTenants       int     `json:"active_tenants"`
	InactiveTenants     int     `json:"inactive_tenants"`
	VacantUnits         int     `json:"vacant_units"`
	OccupiedUnits       int     `json:"occupied_units"`
	OccupancyRate       float64 `json:"occupancy_rate"`
	AverageRentPerUnit  float64 `json:"average_rent_per_unit"`
	TotalMonthlyRevenue float64 `json:"total_monthly_revenue"`
	CollectionRate      float64 `json:"collection_rate"`
	OverduePayments     int     `json:"overdue_payments"`
	OverdueDuesAmount   float64 `json:"overdue_dues_amount"`
	UpcomingRenewals    int     `json:"upcoming_renewals"`
	TenantChurnRate     float64 `json:"tenant_churn_rate"`
}

type PropertyAnalytics struct {
	PropertyID          uuid.UUID `json:"property_id"`
	PropertyName        string    `json:"property_name"`
	PropertyType        string    `json:"property_type"`
	Area                string    `json:"area"`
	TotalUnits          int       `json:"total_units"`
	OccupiedUnits       int       `json:"occupied_units"`
	VacantUnits         int       `json:"vacant_units"`
	OccupancyRate       float64   `json:"occupancy_rate"`
	MonthlyRevenue      float64   `json:"monthly_revenue"`
	OperatingExpenses   float64   `json:"operating_expenses"`
	NetOperatingIncome  float64   `json:"net_operating_income"`
	MaintenanceNeeded   int       `json:"maintenance_needed"`
	ActiveLeases        int       `json:"active_leases"`
	ExpiringLeases      int       `json:"expiring_leases"`
}

type TenantPerformanceMetrics struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	TenantName      string    `json:"tenant_name"`
	RentalHistory   int       `json:"rental_history"`
	AverageStay     float64   `json:"average_stay"`
	PaymentOnTimeRate float64 `json:"payment_on_time_rate"`
	DisputeCount    int       `json:"dispute_count"`
	RiskScore       int       `json:"risk_score"`
	Status          string    `json:"status"`
}

type FinancialAnalytics struct {
	Period              string  `json:"period"`
	TotalRevenue        float64 `json:"total_revenue"`
	TotalExpenses       float64 `json:"total_expenses"`
	NetProfit           float64 `json:"net_profit"`
	ProfitMargin        float64 `json:"profit_margin"`
	RentCollected       float64 `json:"rent_collected"`
	RentPending         float64 `json:"rent_pending"`
	UtilitiesExpense    float64 `json:"utilities_expense"`
	MaintenanceExpense  float64 `json:"maintenance_expense"`
	OtherExpenses       float64 `json:"other_expenses"`
}

type MaintenanceAnalytics struct {
	TotalTasks          int     `json:"total_tasks"`
	CompletedTasks      int     `json:"completed_tasks"`
	PendingTasks        int     `json:"pending_tasks"`
	AvgCompletionDays   float64 `json:"avg_completion_days"`
	HighPriorityTasks   int     `json:"high_priority_tasks"`
	CompletionRate      float64 `json:"completion_rate"`
}

// ─── Expenses ──────────────────────────────────────────────────────────────

type ExpenseCategoryType string

const (
	ExpenseCategoryMaintenance ExpenseCategoryType = "property_maintenance"
	ExpenseCategoryUtilities   ExpenseCategoryType = "utilities"
	ExpenseCategoryInsurance   ExpenseCategoryType = "insurance"
	ExpenseCategoryCleaning    ExpenseCategoryType = "cleaning"
	ExpenseCategoryRepairs     ExpenseCategoryType = "repairs"
	ExpenseCategoryStaff       ExpenseCategoryType = "staff"
	ExpenseCategoryOther       ExpenseCategoryType = "other"
)

type ExpensePaymentMethod string

const (
	ExpensePaymentCash         ExpensePaymentMethod = "cash"
	ExpensePaymentTransfer     ExpensePaymentMethod = "bank_transfer"
	ExpensePaymentCard         ExpensePaymentMethod = "credit_card"
	ExpensePaymentCheck        ExpensePaymentMethod = "check"
	ExpensePaymentOther        ExpensePaymentMethod = "other"
)

type ExpensePaymentStatus string

const (
	ExpensePaymentPending  ExpensePaymentStatus = "pending"
	ExpensePaymentPaid     ExpensePaymentStatus = "paid"
	ExpensePaymentRefunded ExpensePaymentStatus = "refunded"
)

type ExpenseApprovalStatus string

const (
	ExpenseApprovalPending  ExpenseApprovalStatus = "pending"
	ExpenseApprovalApproved ExpenseApprovalStatus = "approved"
	ExpenseApprovalRejected ExpenseApprovalStatus = "rejected"
)

type ExpenseCategory struct {
	ID           uuid.UUID                `json:"id"`
	CompanyID    uuid.UUID                `json:"company_id"`
	CategoryName string                   `json:"category_name"`
	CategoryType ExpenseCategoryType      `json:"category_type"`
	Description  string                   `json:"description"`
	CreatedAt    time.Time                `json:"created_at"`
}

type Expense struct {
	ID             uuid.UUID              `json:"id"`
	CompanyID      uuid.UUID              `json:"company_id"`
	CategoryID     uuid.UUID              `json:"category_id"`
	PropertyID     *uuid.UUID             `json:"property_id,omitempty"`
	TenantID       *uuid.UUID             `json:"tenant_id,omitempty"`
	Amount         float64                `json:"amount"`
	Currency       string                 `json:"currency"`
	ExpenseDate    time.Time              `json:"expense_date"`
	Description    string                 `json:"description"`
	VendorName     string                 `json:"vendor_name"`
	VendorContact  string                 `json:"vendor_contact"`
	PaymentMethod  ExpensePaymentMethod   `json:"payment_method"`
	PaymentStatus  ExpensePaymentStatus   `json:"payment_status"`
	ReceiptURL     string                 `json:"receipt_url"`
	Notes          string                 `json:"notes"`
	CreatedBy      uuid.UUID              `json:"created_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	DeletedAt      *time.Time             `json:"deleted_at,omitempty"`

	// Joined
	Category *ExpenseCategory `json:"category,omitempty"`
}

type ExpenseApproval struct {
	ID                 uuid.UUID             `json:"id"`
	ExpenseID          uuid.UUID             `json:"expense_id"`
	ApprovalStatus     ExpenseApprovalStatus `json:"approval_status"`
	ApprovedBy         *uuid.UUID            `json:"approved_by,omitempty"`
	ApprovalComments   string                `json:"approval_comments"`
	ApprovalDate       *time.Time            `json:"approval_date,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
}

// ─── Inspection & Maintenance ────────────────────────────────────────────────

type InspectionType string

const (
	InspectionGeneral         InspectionType = "general"
	InspectionPreLease        InspectionType = "pre_lease"
	InspectionEndLease        InspectionType = "end_lease"
	InspectionDamageAssessment InspectionType = "damage_assessment"
	InspectionSafety         InspectionType = "safety"
)

type InspectionStatus string

const (
	InspectionScheduled InspectionStatus = "scheduled"
	InspectionInProgress InspectionStatus = "in_progress"
	InspectionCompleted InspectionStatus = "completed"
	InspectionCancelled InspectionStatus = "cancelled"
)

type SeverityLevel string

const (
	SeverityGreen  SeverityLevel = "green"
	SeverityYellow SeverityLevel = "yellow"
	SeverityRed    SeverityLevel = "red"
)

type ChecklistItem struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Critical    bool   `json:"critical"`
}

type ChecklistResult struct {
	Status string `json:"status"` // pass/fail
	Notes  string `json:"notes,omitempty"`
}

type InspectionTemplate struct {
	ID                       uuid.UUID        `json:"id"`
	CompanyID                uuid.UUID        `json:"company_id"`
	TemplateName             string           `json:"template_name"`
	InspectionType           InspectionType   `json:"inspection_type"`
	ChecklistItems           []ChecklistItem  `json:"checklist_items,omitempty"`
	EstimatedDurationMinutes int              `json:"estimated_duration_minutes"`
	CreatedAt                time.Time        `json:"created_at"`
}

type Inspection struct {
	ID               uuid.UUID                      `json:"id"`
	CompanyID        uuid.UUID                      `json:"company_id"`
	PropertyID       uuid.UUID                      `json:"property_id"`
	TemplateID       *uuid.UUID                     `json:"template_id,omitempty"`
	InspectionType   string                         `json:"inspection_type"`
	ScheduledDate    time.Time                      `json:"scheduled_date"`
	CompletedDate    *time.Time                     `json:"completed_date,omitempty"`
	InspectorID      *uuid.UUID                     `json:"inspector_id,omitempty"`
	TenantID         *uuid.UUID                     `json:"tenant_id,omitempty"`
	Status           InspectionStatus               `json:"status"`
	Findings         string                         `json:"findings"`
	SeverityLevel    SeverityLevel                  `json:"severity_level"`
	PhotosURLs       []string                       `json:"photos_urls,omitempty"`
	ChecklistResults map[string]ChecklistResult     `json:"checklist_results,omitempty"`
	CreatedBy        uuid.UUID                      `json:"created_by"`
	CreatedAt        time.Time                      `json:"created_at"`
	UpdatedAt        time.Time                      `json:"updated_at"`
}

type MaintenanceType string

const (
	MaintenancePlumbing    MaintenanceType = "plumbing"
	MaintenanceElectrical  MaintenanceType = "electrical"
	MaintenanceHVAC        MaintenanceType = "hvac"
	MaintenanceFlooring    MaintenanceType = "flooring"
	MaintenancePainting    MaintenanceType = "painting"
	MaintenanceStructural  MaintenanceType = "structural"
	MaintenanceOther       MaintenanceType = "other"
)

type TaskPriority string

const (
	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
	PriorityUrgent TaskPriority = "urgent"
)

type MaintenanceStatus string

const (
	MaintenancePending     MaintenanceStatus = "pending"
	MaintenanceScheduled   MaintenanceStatus = "scheduled"
	MaintenanceInProgress  MaintenanceStatus = "in_progress"
	MaintenanceCompleted   MaintenanceStatus = "completed"
	MaintenanceCancelled   MaintenanceStatus = "cancelled"
)

type MaintenanceTask struct {
	ID               uuid.UUID           `json:"id"`
	CompanyID        uuid.UUID           `json:"company_id"`
	PropertyID       uuid.UUID           `json:"property_id"`
	InspectionID     *uuid.UUID          `json:"inspection_id,omitempty"`
	MaintenanceType  MaintenanceType     `json:"maintenance_type"`
	Description      string              `json:"description"`
	Priority         TaskPriority        `json:"priority"`
	ScheduledDate    *time.Time          `json:"scheduled_date,omitempty"`
	DueDate          *time.Time          `json:"due_date,omitempty"`
	CompletionDate   *time.Time          `json:"completion_date,omitempty"`
	ContractorName   string              `json:"contractor_name"`
	ContractorContact string             `json:"contractor_contact"`
	EstimatedCost    *float64            `json:"estimated_cost,omitempty"`
	ActualCost       *float64            `json:"actual_cost,omitempty"`
	Status           MaintenanceStatus   `json:"status"`
	AssignedTo       *uuid.UUID          `json:"assigned_to,omitempty"`
	Notes            string              `json:"notes"`
	CreatedBy        uuid.UUID           `json:"created_by"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
	DeletedAt        *time.Time          `json:"deleted_at,omitempty"`
}

type MaintenancePhoto struct {
	ID          uuid.UUID `json:"id"`
	TaskID      uuid.UUID `json:"task_id"`
	PhotoURL    string    `json:"photo_url"`
	UploadedAt  time.Time `json:"uploaded_at"`
	PhotoStage  string    `json:"photo_stage"` // before/during/after
}

// ─── Lease Renewal ──────────────────────────────────────────────────────────

type RenewalStatus string

const (
	RenewalPending    RenewalStatus = "pending"
	RenewalInProgress RenewalStatus = "in_progress"
	RenewalOfferSent  RenewalStatus = "offer_sent"
	RenewalAccepted   RenewalStatus = "accepted"
	RenewalRejected   RenewalStatus = "rejected"
	RenewalExpired    RenewalStatus = "expired"
)

type TenantRenewalResponse string

const (
	ResponsePending      TenantRenewalResponse = "pending"
	ResponseAccepted     TenantRenewalResponse = "accepted"
	ResponseRejected     TenantRenewalResponse = "rejected"
	ResponseCounterOffer TenantRenewalResponse = "counter_offer"
)

type Language string

const (
	LangArabic  Language = "ar"
	LangEnglish Language = "en"
)

// ─── Commission Tracking ────────────────────────────────────────────────────

type CommissionType string

const (
	CommissionFixed      CommissionType = "fixed_amount"
	CommissionPercentage CommissionType = "percentage"
	CommissionTiered     CommissionType = "tiered"
)

type ApplicableToType string

const (
	ApplicableDeals  ApplicableToType = "deals"
	ApplicableLeases ApplicableToType = "leases"
	ApplicableBoth   ApplicableToType = "both"
)

type CommissionStatusType string

const (
	CommissionStatusPending   CommissionStatusType = "pending"
	CommissionStatusApproved  CommissionStatusType = "approved"
	CommissionStatusPaid      CommissionStatusType = "paid"
	CommissionStatusDisputed  CommissionStatusType = "disputed"
)

type CommissionTransactionType string

const (
	TransactionDealCommission  CommissionTransactionType = "deal_commission"
	TransactionLeaseCommission CommissionTransactionType = "lease_commission"
	TransactionBonus           CommissionTransactionType = "bonus"
	TransactionDeduction       CommissionTransactionType = "deduction"
)

type CommissionStructure struct {
	ID              uuid.UUID          `json:"id"`
	CompanyID       uuid.UUID          `json:"company_id"`
	StructureName   string             `json:"structure_name"`
	CommissionType  CommissionType     `json:"commission_type"`
	ApplicableTo    ApplicableToType   `json:"applicable_to"`
	EffectiveFrom   *time.Time         `json:"effective_from,omitempty"`
	EffectiveTo     *time.Time         `json:"effective_to,omitempty"`
	Rules           map[string]interface{} `json:"rules,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

type AgentCommission struct {
	ID                    uuid.UUID                `json:"id"`
	CompanyID             uuid.UUID                `json:"company_id"`
	AgentID               uuid.UUID                `json:"agent_id"`
	CommissionPeriodStart time.Time                `json:"commission_period_start"`
	CommissionPeriodEnd   time.Time                `json:"commission_period_end"`
	CommissionStructureID *uuid.UUID               `json:"commission_structure_id,omitempty"`
	DealsCount            int                      `json:"deals_count"`
	DealsRevenue          float64                  `json:"deals_revenue"`
	LeasesCount           int                      `json:"leases_count"`
	LeasesRevenue         float64                  `json:"leases_revenue"`
	TotalCommission       float64                  `json:"total_commission"`
	Status                CommissionStatusType    `json:"status"`
	ApprovalDate          *time.Time               `json:"approval_date,omitempty"`
	PaymentDate           *time.Time               `json:"payment_date,omitempty"`
	PaymentReference      string                  `json:"payment_reference"`
	Notes                 string                  `json:"notes"`
	CreatedAt             time.Time               `json:"created_at"`
	UpdatedAt             time.Time               `json:"updated_at"`
}

type CommissionTransaction struct {
	ID                  uuid.UUID                    `json:"id"`
	CommissionID        uuid.UUID                    `json:"commission_id"`
	DealID              *uuid.UUID                   `json:"deal_id,omitempty"`
	LeaseID             *uuid.UUID                   `json:"lease_id,omitempty"`
	TransactionAmount   float64                      `json:"transaction_amount"`
	TransactionType     CommissionTransactionType    `json:"transaction_type"`
	CreatedAt           time.Time                    `json:"created_at"`
}

type LeaseRenewalWorkflow struct {
	ID                 uuid.UUID                `json:"id"`
	CompanyID          uuid.UUID                `json:"company_id"`
	LeaseID            uuid.UUID                `json:"lease_id"`
	RenewalDate        time.Time                `json:"renewal_date"`
	RenewalStatus      RenewalStatus            `json:"renewal_status"`
	DaysBeforeExpiry   int                      `json:"days_before_expiry"`
	ProposedRentAmount *float64                 `json:"proposed_rent_amount,omitempty"`
	ProposedTerms      map[string]interface{}   `json:"proposed_terms,omitempty"`
	TenantResponse     TenantRenewalResponse    `json:"tenant_response"`
	TenantCounterOffer *float64                 `json:"tenant_counter_offer,omitempty"`
	CounterOfferDate   *time.Time               `json:"counter_offer_date,omitempty"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
}

type RenewalCommunicationTemplate struct {
	ID           uuid.UUID `json:"id"`
	CompanyID    uuid.UUID `json:"company_id"`
	TemplateName string    `json:"template_name"`
	EmailSubject string    `json:"email_subject"`
	EmailBody    string    `json:"email_body"`
	WhatsAppMsg  string    `json:"whatsapp_message"`
	Language     Language  `json:"language"`
	CreatedAt    time.Time `json:"created_at"`
}

type RenewalCommunicationLog struct {
	ID                  uuid.UUID          `json:"id"`
	RenewalID           uuid.UUID          `json:"renewal_id"`
	CommunicationType   CommunicationType  `json:"communication_type"`
	TemplateID          *uuid.UUID         `json:"template_id,omitempty"`
	SentDate            *time.Time         `json:"sent_date,omitempty"`
	DeliveryStatus      DeliveryStatus     `json:"delivery_status"`
	TenantResponseDate  *time.Time         `json:"tenant_response_date,omitempty"`
	ResponseText        string             `json:"response_text"`
	CreatedAt           time.Time          `json:"created_at"`
}

// ─── Document Management ────────────────────────────────────────────────────

type DocumentType string

const (
	DocTypeLease         DocumentType = "lease"
	DocTypeOffer         DocumentType = "offer"
	DocTypeInspection    DocumentType = "inspection_report"
	DocTypeWaiver        DocumentType = "maintenance_waiver"
	DocTypeCustom        DocumentType = "custom"
)

type SignatureStatus string

const (
	SignatureNotRequired SignatureStatus = "not_required"
	SignaturePending     SignatureStatus = "pending"
	SignatureSigned      SignatureStatus = "signed"
)

type MessageTemplate struct {
	ID        uuid.UUID  `json:"id"`
	CompanyID uuid.UUID  `json:"company_id"`
	Name      string     `json:"name"`
	Body      string     `json:"body"`
	Category  string     `json:"category"`
	Variables []string   `json:"variables"`
	IsActive  bool       `json:"is_active"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type DocumentTemplate struct {
	ID                 uuid.UUID                    `json:"id"`
	CompanyID          uuid.UUID                    `json:"company_id"`
	TemplateName       string                       `json:"template_name"`
	DocumentType       DocumentType                 `json:"document_type"`
	TemplateContent    string                       `json:"template_content"`
	Language           Language                     `json:"language"`
	SignatureRequired  bool                         `json:"signature_required"`
	SignatureFields    []map[string]interface{}     `json:"signature_fields,omitempty"`
	CreatedBy          uuid.UUID                    `json:"created_by"`
	CreatedAt          time.Time                    `json:"created_at"`
}

type Document struct {
	ID                   uuid.UUID               `json:"id"`
	CompanyID            uuid.UUID               `json:"company_id"`
	DocumentType         DocumentType            `json:"document_type"`
	OriginalTemplateID   *uuid.UUID              `json:"original_template_id,omitempty"`
	RelatedEntityType    string                  `json:"related_entity_type"`
	RelatedEntityID      *uuid.UUID              `json:"related_entity_id,omitempty"`
	DocumentTitle        string                  `json:"document_title"`
	FileURL              string                  `json:"file_url"`
	FileSizeBytes        int                     `json:"file_size_bytes,omitempty"`
	ContentHash          string                  `json:"content_hash"`
	SignatureStatus      SignatureStatus         `json:"signature_status"`
	CreatedBy            uuid.UUID               `json:"created_by"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
	DataClassification   DataClassification      `json:"data_classification"`
	RetentionUntil       *time.Time              `json:"retention_until,omitempty"`
	DeletedAt            *time.Time              `json:"deleted_at,omitempty"`
}

type DocumentSignature struct {
	ID                  uuid.UUID       `json:"id"`
	DocumentID          uuid.UUID       `json:"document_id"`
	SignerName          string          `json:"signer_name"`
	SignerEmail         string          `json:"signer_email"`
	SignatureFieldName  string          `json:"signature_field_name"`
	SignatureStatus     SignatureStatus `json:"signature_status"`
	SignedAt            *time.Time      `json:"signed_at,omitempty"`
	SignatureImageURL   string          `json:"signature_image_url"`
	IPAddress           string          `json:"ip_address"`
	UserAgent           string          `json:"user_agent"`
	EnvelopeID          string          `json:"envelope_id,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
}

type DocumentAuditLog struct {
	ID         uuid.UUID              `json:"id"`
	DocumentID uuid.UUID              `json:"document_id"`
	Action     string                 `json:"action"`
	ActorID    *uuid.UUID             `json:"actor_id,omitempty"`
	ActorName  string                 `json:"actor_name"`
	OldValues  map[string]interface{} `json:"old_values,omitempty"`
	NewValues  map[string]interface{} `json:"new_values,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}
