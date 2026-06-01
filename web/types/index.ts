// ─── Auth ────────────────────────────────────────────────────────────────────

export type UserRole = 'admin' | 'agent' | 'viewer'

export interface User {
  id: string
  name: string
  email: string
  role: UserRole
  lang_pref: 'ar' | 'en'
  wa_number: string
  is_active: boolean
  created_at: string
}

export interface AuthUser {
  id: string
  name: string
  email: string
  role: 'admin' | 'agent' | 'viewer'
  lang_pref: 'ar' | 'en'
}

export interface Company {
  id: string
  name: string
  subdomain?: string
  plan: string
  on_trial: boolean
  trial_ends_at: string | null
  days_remaining: number
  is_demo: boolean
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: AuthUser
  company?: Company
}

// ─── Contact ─────────────────────────────────────────────────────────────────

export interface Contact {
  id: string
  phone_wa: string
  full_name: string
  email: string
  language: 'ar' | 'en'
  lead_score: number
  assigned_to: string | null
  created_at: string
  updated_at: string
}

export interface PaginatedResult<T> {
  data: T[]
  total: number
  page: number
  limit: number
}

// ─── Pipeline Stage ─────────────────────────────────────────────────────────────

export interface PipelineStage {
  id: string
  company_id: string
  entity_type: string
  name: string
  sort_order: number
  color: string
  is_won: boolean
  is_lost: boolean
  is_default: boolean
  created_at: string
}

// ─── Lead ─────────────────────────────────────────────────────────────────────

export type LeadStage = 'new' | 'contacted' | 'qualified' | 'proposal' | 'won' | 'lost'
export type LeadSource = 'whatsapp' | 'web' | 'referral' | 'event'

export interface Lead {
  id: string
  contact_id: string
  stage: LeadStage
  source: LeadSource
  deal_value: number
  currency: string
  notes: string
  lead_score?: number
  score_updated_at?: string
  assigned_to?: string | null
  closed_reason?: string
  last_contacted_at?: string
  tags?: string[]
  created_at: string
  updated_at: string
  contact?: Contact
  assigned_user?: { id: string; name: string } | null
}

export type KanbanBoard = Record<string, Lead[]>

// ─── Approval Workflow ────────────────────────────────────────────────────────

export type ApprovalStatus = 'pending' | 'approved' | 'rejected'
export type ApprovalEntity = 'listing' | 'deal' | 'offer'

export interface ApprovalConfig {
  id: string
  company_id: string
  listing_approval: boolean
  deal_approval_above: number
  offer_approval_above: number
  updated_at: string
}

export interface ApprovalRequest {
  id: string
  company_id: string
  entity_type: ApprovalEntity
  entity_id: string
  requested_by: string
  reviewed_by: string | null
  status: ApprovalStatus
  notes: string
  reviewer_note: string
  created_at: string
  reviewed_at: string | null
  requester_name: string
  reviewer_name: string
}

// ─── Communication History ────────────────────────────────────────────────────

export type CommunicationType = 'whatsapp_inbound' | 'whatsapp_outbound' | 'email_sent' | 'email_received' | 'call'
export type CommunicationStatus = 'pending' | 'sent' | 'delivered' | 'read' | 'failed' | 'bounced'

export interface CommunicationHistory {
  id: number
  lead_id: string
  contact_id: string
  communication_type: CommunicationType
  direction: 'inbound' | 'outbound'
  body: string
  from_identifier: string
  to_identifier: string
  external_id: string
  status: CommunicationStatus
  metadata?: Record<string, unknown>
  created_at: string
  created_by?: string
}

// ─── WhatsApp ─────────────────────────────────────────────────────────────────

export type ThreadStatus = 'open' | 'closed' | 'pending'
export type MessageDirection = 'inbound' | 'outbound'

export interface WhatsAppThread {
  id: string
  contact_id: string
  wa_account_id: string
  thread_status: ThreadStatus
  last_message_at: string | null
  message_count: number
  ai_summary: string
  created_at: string
  contact?: Contact
}

export interface WhatsAppMessage {
  id: string
  thread_id: string
  direction: MessageDirection
  body: string
  media_url: string
  wa_message_id: string
  sent_at: string
}

// ─── Deal ─────────────────────────────────────────────────────────────────────

export type DealStage = 'open' | 'won' | 'lost'
export type InvoiceStatus = 'draft' | 'sent' | 'paid'

export interface Deal {
  id: string
  lead_id: string
  title: string
  stage: DealStage
  amount: number
  currency: string
  close_date: string | null
  probability: number
  owner_id: string
  created_at: string
  updated_at: string
}

export interface VATInvoice {
  id: string
  deal_id: string
  invoice_no: string
  subtotal: number
  vat_rate: number
  vat_amount: number
  total: number
  qr_payload?: string
  status: InvoiceStatus
  issued_at: string
}

// ─── Notification ─────────────────────────────────────────────────────────────

export interface Notification {
  id: string
  user_id: string
  type: string
  title: string
  body: string
  read: boolean
  data: string
  created_at: string
}

// ─── Rental Property ──────────────────────────────────────────────────────────

export type PropertyType = 'villa' | 'apartment' | 'townhouse' | 'commercial'
export type PropertyStatus = 'active' | 'inactive' | 'sold' | 'maintenance'
export type OccupancyStatus = 'vacant' | 'occupied' | 'maintenance'

export interface RentalProperty {
  id: string
  company_id: string
  name: string
  description: string
  property_type: PropertyType
  units_count: number
  area: string
  street_address: string
  building_number: string
  unit_number: string
  city: string
  emirate: string
  postal_code: string
  total_sqft: number
  bedrooms: number
  bathrooms: number
  parking_spaces: number
  amenities: string[]
  purchase_price: number
  purchase_date: string | null
  market_value: number
  currency: string
  status: PropertyStatus
  occupancy_status: OccupancyStatus
  total_occupied_units: number
  property_deed_url: string
  title_deed_number: string
  municipality_registration: string
  created_at: string
  updated_at: string
  created_by: string | null
  updated_by: string | null
}

// ─── Tenant ───────────────────────────────────────────────────────────────────

export type IDType = 'emirati_id' | 'passport' | 'driving_license' | 'trade_license'
export type EmploymentStatus = 'employed' | 'self_employed' | 'retired' | 'student'
export type TenantStatus = 'active' | 'inactive' | 'blacklisted'
export type VerificationStatus = 'pending' | 'verified' | 'rejected'

export interface Tenant {
  id: string
  company_id: string
  full_name_en: string
  full_name_ar: string
  email: string
  phone: string
  phone_wa: string
  id_type: IDType
  id_number: string
  id_expiry_date: string | null
  id_document_url: string
  is_verified: boolean
  verification_status: VerificationStatus
  verification_date: string | null
  verified_by: string | null
  verification_notes: string
  employment_status: EmploymentStatus
  employer_name: string
  annual_income: number
  income_currency: string
  salary_certificate_url: string
  nationality: string
  country_of_origin: string
  permanent_address: string
  emergency_contact_name: string
  emergency_contact_phone: string
  status: TenantStatus
  notes: string
  created_at: string
  updated_at: string
  created_by: string | null
  updated_by: string | null
}

// ─── Lease Management ────────────────────────────────────────────────────────

export type PaymentFrequency = 'monthly' | 'quarterly' | 'semi_annual' | 'annual'
export type LeaseStatus = 'active' | 'renewed' | 'terminated' | 'expired'

export interface LeaseTemplate {
  id: string
  company_id: string
  name: string
  description: string
  is_default: boolean
  payment_frequency: PaymentFrequency
  payment_day_of_month: number
  auto_generate_payments: boolean
  default_security_deposit_percent: number
  default_utility_charges: number
  default_late_fee_percent: number
  default_lease_duration_months: number
  default_notice_period_days: number
  default_renewal_duration_months: number
  template_document_url: string
  terms_conditions: string
  status: string
  created_at: string
  updated_at: string
  created_by: string | null
  updated_by: string | null
}

export interface Lease {
  id: string
  company_id: string
  property_id: string
  tenant_id: string
  template_id: string | null
  start_date: string
  end_date: string
  renewal_start_date: string | null
  renewal_end_date: string | null
  monthly_rent: number
  currency: string
  security_deposit: number
  utility_charges: number
  late_fee_percent: number
  payment_frequency: PaymentFrequency
  payment_day_of_month: number
  auto_generate_payments: boolean
  last_generated_payment_date: string | null
  notice_period_days: number
  move_out_date: string | null
  move_out_inspection_date: string | null
  lease_document_url: string
  signed_by_landlord_date: string | null
  signed_by_tenant_date: string | null
  ejari_number: string
  ejari_url: string
  status: LeaseStatus
  termination_reason: string
  termination_date: string | null
  notes: string
  created_at: string
  updated_at: string
  created_by: string | null
  updated_by: string | null
  property?: RentalProperty
  tenant?: Tenant
  template?: LeaseTemplate
}

// ─── Payment Management ───────────────────────────────────────────────────────

export type PaymentStatus = 'pending' | 'received' | 'overdue' | 'failed' | 'refunded'
export type PaymentMethod = 'transfer' | 'check' | 'cash' | 'card' | 'other'

export interface Payment {
  id: string
  company_id: string
  lease_id: string
  amount: number
  currency: string
  due_date: string
  paid_date: string | null
  payment_method: PaymentMethod
  payment_reference: string
  status: PaymentStatus
  bank_transaction_id: string | null
  reconciled_at: string | null
  reconciled_by: string | null
  notes: string
  receipt_url: string
  late_fee_applied: boolean
  late_fee_amount: number
  created_at: string
  updated_at: string
  created_by: string | null
  updated_by: string | null
  lease?: Lease
}

// ─── Bank Integration ─────────────────────────────────────────────────────────

export type TransactionType = 'credit' | 'debit' | 'transfer' | 'check'

export interface BankTransaction {
  id: string
  company_id: string
  bank_integration_id: string
  external_id: string
  amount: number
  currency: string
  transaction_date: string
  description: string
  transaction_type: TransactionType
  matched_payment_id: string | null
  match_confidence: number
  created_at: string
  updated_at: string
}

export interface BankIntegration {
  id: string
  company_id: string
  bank_name: string
  bank_code: string
  account_number: string
  account_name: string
  iban: string
  integration_type: string
  status: string
  api_key_encrypted?: string
  api_secret_encrypted?: string
  api_endpoint: string
  auto_sync: boolean
  last_sync_date: string | null
  sync_interval_hours: number
  last_sync_error: string | null
  sync_error_count: number
  is_connected: boolean
  connection_test_date: string | null
  created_at: string
  updated_at: string
  created_by: string | null
  updated_by: string | null
}

// ─── Bank Statements & Payment Confirmations ───────────────────────────────

export type FileFormat = 'csv' | 'pdf' | 'xlsx'
export type ProcessingStatus = 'pending' | 'processing' | 'completed' | 'failed'
export type DataClassification = 'public' | 'internal' | 'confidential'

export interface BankStatement {
  id: string
  company_id: string
  bank_integration_id: string
  file_name: string
  file_size_bytes: number
  file_url: string
  file_format: FileFormat
  uploaded_by: string
  upload_date: string
  processing_status: ProcessingStatus
  transactions_imported: number
  import_error: string | null
  data_classification: DataClassification
  retention_until: string | null
  created_at: string
  updated_at: string
  deleted_at: string | null
}

export type ConfirmationDeliveryStatus = 'pending' | 'sent' | 'failed' | 'bounced'
export type ConfirmationDeliveryMethod = 'email' | 'whatsapp' | 'sms'

export interface PaymentConfirmation {
  id: string
  company_id: string
  payment_id: string
  confirmation_number: string
  tenant_email: string
  tenant_phone: string | null
  sent_at: string | null
  delivery_status: ConfirmationDeliveryStatus
  delivery_method: ConfirmationDeliveryMethod
  pdf_url: string | null
  data_classification: DataClassification
  retention_until: string | null
  created_at: string
  updated_at: string
  deleted_at: string | null
}

// ─── WhatsApp Outbound ────────────────────────────────────────────────────────

export type OutboundStatus = 'pending' | 'sent' | 'delivered' | 'read' | 'failed'

export interface WhatsAppOutbound {
  id: number
  thread_id: string
  to_number: string
  message_body: string
  media_url: string
  wa_message_id: string
  status: OutboundStatus
  error_message: string
  scheduled_at: string | null
  sent_at: string | null
  created_at: string
  created_by: string | null
  metadata: Record<string, unknown> | null
}

// ─── Lease Renewals ────────────────────────────────────────────────────────────

export type RenewalStatus = 'pending' | 'in_progress' | 'offer_sent' | 'accepted' | 'rejected' | 'expired'
export type TenantResponse = 'pending' | 'accepted' | 'rejected' | 'counter_offer'

export interface LeaseRenewalWorkflow {
  id: string
  company_id: string
  lease_id: string
  renewal_date: string
  renewal_status: RenewalStatus
  days_before_expiry: number
  proposed_rent_amount: number | null
  proposed_terms: Record<string, unknown> | null
  tenant_response: TenantResponse
  tenant_counter_offer: number | null
  counter_offer_date: string | null
  created_at: string
  updated_at: string
  lease?: Lease
}

export interface RenewalCommunicationTemplate {
  id: string
  company_id: string
  template_name: string
  email_subject: string
  email_body: string
  whatsapp_message: string
  language: 'ar' | 'en'
  created_at: string
}

// ─── Inspections ──────────────────────────────────────────────────────────────

export type InspectionType = 'general' | 'pre_lease' | 'end_lease' | 'damage_assessment' | 'safety'
export type InspectionStatus = 'scheduled' | 'in_progress' | 'completed' | 'cancelled'
export type SeverityLevel = 'green' | 'yellow' | 'red'

export interface ChecklistItem {
  id: string
  description: string
  critical: boolean
}

export interface ChecklistResult {
  status: string
  notes: string
}

export interface InspectionTemplate {
  id: string
  company_id: string
  template_name: string
  inspection_type: InspectionType
  checklist_items: ChecklistItem[]
  estimated_duration_minutes: number
  created_at: string
}

export interface Inspection {
  id: string
  company_id: string
  property_id: string
  template_id: string | null
  inspection_type: string
  scheduled_date: string
  completed_date: string | null
  inspector_id: string | null
  tenant_id: string | null
  status: InspectionStatus
  findings: string
  severity_level: SeverityLevel
  photos_urls: string[]
  checklist_results: Record<string, ChecklistResult> | null
  created_by: string
  created_at: string
  updated_at: string
}

// ─── Viewings ─────────────────────────────────────────────────────────────────

export type ViewingStatus = 'scheduled' | 'confirmed' | 'checked_in' | 'completed' | 'cancelled' | 'no_show'

export interface Viewing {
  id: string
  listing_id?: string
  contact_id: string
  agent_id?: string
  lead_id?: string
  scheduled_at: string
  duration_min: number
  status: ViewingStatus
  address: string
  notes: string
  checked_in_at?: string
  checked_out_at?: string
  reminder_sent: boolean
  created_at: string
  updated_at: string
  contact?: { id: string; full_name: string; phone_wa: string; email?: string }
  listing_title?: string
  agent_name?: string
}

// ─── Maintenance ──────────────────────────────────────────────────────────────

export type MaintenanceType = 'plumbing' | 'electrical' | 'hvac' | 'flooring' | 'painting' | 'structural' | 'other'
export type TaskPriority = 'low' | 'medium' | 'high' | 'urgent'
export type MaintenanceStatus = 'pending' | 'scheduled' | 'in_progress' | 'completed' | 'cancelled'

export interface MaintenancePhoto {
  id: string
  task_id: string
  photo_url: string
  uploaded_at: string
  photo_stage: string
}

export interface MaintenanceTask {
  id: string
  company_id: string
  property_id: string
  inspection_id: string | null
  maintenance_type: MaintenanceType
  description: string
  priority: TaskPriority
  scheduled_date: string | null
  due_date: string | null
  completion_date: string | null
  contractor_name: string
  contractor_contact: string
  estimated_cost: number | null
  actual_cost: number | null
  status: MaintenanceStatus
  assigned_to: string | null
  notes: string
  created_by: string
  created_at: string
  updated_at: string
}

// ─── Expenses ─────────────────────────────────────────────────────────────────

export type ExpenseCategoryType = 'property_maintenance' | 'utilities' | 'insurance' | 'cleaning' | 'repairs' | 'staff' | 'other'
export type ExpensePaymentMethod = 'cash' | 'bank_transfer' | 'credit_card' | 'check' | 'other'
export type ExpensePaymentStatus = 'pending' | 'paid' | 'refunded'
export type ExpenseApprovalStatus = 'pending' | 'approved' | 'rejected'

export interface ExpenseCategory {
  id: string
  company_id: string
  category_name: string
  category_type: ExpenseCategoryType
  description: string
  created_at: string
}

export interface ExpenseApproval {
  id: string
  expense_id: string
  approval_status: ExpenseApprovalStatus
  approved_by: string | null
  approval_comments: string
  approval_date: string | null
  created_at: string
}

export interface Expense {
  id: string
  company_id: string
  category_id: string
  property_id: string | null
  tenant_id: string | null
  amount: number
  currency: string
  expense_date: string
  description: string
  vendor_name: string
  vendor_contact: string
  payment_method: ExpensePaymentMethod
  payment_status: ExpensePaymentStatus
  receipt_url: string
  notes: string
  created_by: string
  created_at: string
  updated_at: string
  deleted_at: string | null
  category?: ExpenseCategory
}

// ─── Documents ─────────────────────────────────────────────────────────────────

export type DocumentType = 'lease' | 'offer' | 'inspection_report' | 'maintenance_waiver' | 'custom'
export type SignatureStatus = 'not_required' | 'pending' | 'signed'
export interface Document {
  id: string
  company_id: string
  document_type: DocumentType
  original_template_id: string | null
  related_entity_type: string
  related_entity_id: string | null
  document_title: string
  file_url: string
  file_size_bytes: number
  content_hash: string
  signature_status: SignatureStatus
  created_by: string
  created_at: string
  updated_at: string
  data_classification: DataClassification
  retention_until: string | null
  deleted_at: string | null
}

export interface DocumentSignature {
  id: string
  document_id: string
  signer_name: string
  signer_email: string
  signature_field_name: string
  signature_status: SignatureStatus
  signed_at: string | null
  signature_image_url: string
  ip_address: string
  user_agent: string
  created_at: string
}

export interface DocumentTemplate {
  id: string
  company_id: string
  template_name: string
  document_type: DocumentType
  template_content: string
  language: string
  signature_required: boolean
  signature_fields: string[]
  created_by: string
  created_at: string
}

// ─── Message Templates ─────────────────────────────────────────────────────────

export interface MessageTemplate {
  id: string
  company_id: string
  name: string
  body: string
  category: string
  variables: string[]
  is_active: boolean
  created_by?: string
  updated_by?: string
  created_at: string
  updated_at: string
}

// ─── Renewal Templates ────────────────────────────────────────────────────────

export interface RenewalTemplate {
  id: string
  company_id: string
  template_name: string
  email_subject: string
  email_body: string
  whatsapp_message: string
  language: string
  created_at: string
}

// ─── Email History ────────────────────────────────────────────────────────────

export type EmailStatus = 'pending' | 'sent' | 'failed' | 'bounced'

export interface EmailHistory {
  id: number
  from_email: string
  to_email: string
  subject: string
  body: string
  html_body: string
  status: EmailStatus
  error_message: string
  related_to: string
  related_id: number | null
  sent_at: string | null
  created_at: string
  created_by: string | null
  metadata: Record<string, unknown> | null
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

export interface AuditLog {
  id: number
  entity_type: string
  entity_id: string
  action: string
  actor_id: string | null
  diff: unknown
  ts: string
}

// ─── Listing ───────────────────────────────────────────────────────────────────

export type ListingType = 'sale' | 'rent'
export type ListingStatus = 'draft' | 'published' | 'sold' | 'rented' | 'expired' | 'withdrawn'

export interface Listing {
  id: string
  company_id: string
  title: string
  description: string
  property_type: string
  listing_type: ListingType
  price: number
  currency: string
  rent_period: string | null
  area: string
  community: string
  subcommunity: string
  city: string
  emirate: string
  latitude: number | null
  longitude: number | null
  bedrooms: number
  bathrooms: number
  total_sqft: number
  plot_sqft: number
  parking_spaces: number
  furnishing: string | null
  amenities: string[]
  year_built: number | null
  cover_image_url: string
  image_urls: string[]
  virtual_tour_url: string
  video_url: string
  status: ListingStatus
  featured: boolean
  reference_number: string
  available_from: string | null
  assigned_to: string | null
  owner_name: string
  owner_phone: string
  owner_email: string
  portal_sync_status: Record<string, unknown>
  published_at: string | null
  created_at: string
  updated_at: string
  created_by: string | null
  updated_by: string | null
}

// ─── WebSocket Events ─────────────────────────────────────────────────────────

export interface WSEvent {
  type: 'lead.created' | 'lead.stage_changed' | 'whatsapp.message' | 'notification'
  payload: unknown
}
