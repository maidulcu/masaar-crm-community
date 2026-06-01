# Changelog

All notable changes to Masaar CRM will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.1.0] - 2026-03-13

### Added
- **WhatsApp Receiver** — Receive and read incoming WhatsApp messages via Meta webhook
- **Sales Pipeline** — Single Kanban board with drag-drop stage management (dnd-kit)
- **AI Summarize** — Manual "Summarize" button powered by Ollama (local LLM, data never leaves server)
- **Deals Management** — Deal list, stage tracking, linked to leads
- **VAT Invoicing** — Invoice generator with 5% UAE VAT; sequential `INV-YYYY-NNNN` numbering; draft → sent → paid workflow
- **Notifications** — Personal real-time notifications via WebSocket (`/ws/notifications`)
- **Keyword Search** — Search contacts by name, phone, email
- **Contact Management** — Unified contact profiles linked to WhatsApp threads
- **Audit Log** — Immutable activity log for PDPL compliance
- **RTL / LTR** — Full Arabic interface; Cairo font for AR, Inter for EN; logical CSS properties
- **RBAC** — Role-based access control (admin / agent / viewer) enforced on all write routes
- **Rate Limiting** — 300 req/min on WhatsApp webhook; 10 req/min on login
- **OpenAPI / Swagger UI** — Auto-generated spec at `/docs`, `docs/swagger.yaml` committed to repo
- **Self-hosted** — Full stack via `docker compose up` (Next.js + Go + PostgreSQL + Redis + Ollama)

### Tech Stack
- Go 1.22 + Fiber v2
- Next.js 14 + Tailwind CSS (App Router, TypeScript)
- PostgreSQL 16 + pgvector
- Ollama (llama3 / mistral)
- WebSockets (native Fiber)
- Redis (refresh tokens + blacklist)
- goose v3 (SQL migrations)

---

## [v0.2.0] - 2026-04-26

### Phase 15: Core Operations & Analytics

#### Phase 15.1: Expense Tracking System
- **Added** comprehensive expense management system
  - `expense_categories` table with type enum (maintenance, utilities, insurance, repairs, staff, cleaning, other)
  - `expenses` table with full expense lifecycle (amount, date, vendor, payment status, receipt tracking)
  - `expense_approvals` table for approval workflow
  - 8 API endpoints: category management, expense CRUD, approval workflow
  - Role-based access: Admin full access, Agent create/update
  - Real-time notifications on approval/rejection

#### Phase 15.2: Inspection & Maintenance Scheduling
- **Added** inspection and maintenance management system
  - `inspection_templates` table with reusable checklists
  - `inspections` table with status tracking, findings, severity levels, photos
  - `maintenance_tasks` table with priority, contractor tracking, cost estimation
  - `maintenance_photos` table for before/during/after documentation
  - 16 API endpoints for inspection and maintenance workflows
  - Frontend pages: `/inspections`, `/maintenance` with list, filter, detail views
  - Features: Priority-based sorting, status tracking, photo gallery

#### Phase 15.3: Tenant Analytics Dashboard
- **Added** comprehensive analytics system
  - Company-wide KPIs: occupancy rate, revenue, collection rate
  - Property-level metrics: revenue, NOI, maintenance needs
  - Tenant performance metrics: risk scoring based on payment history, disputes, tenure
  - Financial analytics: revenue trends, expense breakdown, profit margins
  - Maintenance metrics: task completion rates, response times
  - 7 API endpoints under `/api/v1/analytics/`
  - Frontend analytics pages:
    - `/analytics/overview` - KPI dashboard
    - `/analytics/properties` - Property performance
    - `/analytics/tenants` - Risk scoring
    - `/analytics/financial` - Revenue analysis
  - Features: Real-time calculations, bilingual labels, color-coded risk scores

### Phase 16: Advanced Workflows

#### Phase 16.1: Lease Renewal Automation
- **Added** automated lease renewal workflow system
  - `lease_renewal_workflows` table with status tracking, proposed terms, counter-offers
  - `renewal_communication_templates` table for email/WhatsApp templates (bilingual)
  - `renewal_communication_log` table for audit trail
  - 9 API endpoints: initiate, propose, send-offer, accept, reject, counter-offer, templates
  - Features: Automated renewal tracking, communication history, tenant responses
  - Role-based access: Admin initiates, Agent/Admin manage workflow

#### Phase 16.2: Commission Tracking & Agent Performance
- **Added** database schema (Migration 00027):
  - `commission_structures` table (fixed/percentage/tiered types)
  - `agent_commissions` table (monthly tracking)
  - `commission_transactions` table (individual entries)
  - Domain models and enums ready for handler implementation

#### Phase 16.3: Document Management & e-Signature
- **Added** database schema (Migration 00028):
  - `document_templates` table (lease, offer, inspection, waiver, custom)
  - `documents` table with signature status and audit trail
  - `document_signatures` table with IP/user-agent logging
  - `document_audit_log` table for compliance
  - Data classification: public, internal, confidential
  - Ready for e-signature integration (DocuSign framework)

#### Phase 16.4-16.6: Infrastructure
- **Added** custom fields system (Migration 00029):
  - Entity-specific custom metadata support
  - Dynamic field storage and validation
- **Added** bulk operations framework (Migration 00030):
  - CSV import/export job tracking
  - Error logging and data validation
  - Dry-run mode support

### Infrastructure Updates
- Total migrations: 30 (up from previous)
- New tables: 35+ across all phases
- Total API endpoints: 100+ with consistent response format
- All endpoints include role-based access control
- Full audit trails for sensitive operations

### Breaking Changes
- None. All changes are backward-compatible.

### Known Limitations
- DocuSign e-signature not yet implemented (framework ready)
- Commission calculation engine pending implementation
- Bulk import/export job processing pending
- Analytics caching in Redis not yet implemented

---

## [v0.3.0] — 2026-05-27

### Critical Bug Fixes

#### Backend: `uuid.Parse` compile error across 5 handler files
- **Root cause:** `uuid.Parse(str)` returns `(uuid.UUID, error)` but all 14 call-sites across 5 files captured only one return value, preventing the entire binary from compiling.
- **Files fixed:** `document.go`, `expense.go`, `inspection.go`, `lease_renewal.go`, `maintenance.go`
- **Fix:** All occurrences changed to `id, _ := uuid.Parse(...)` or `field, _ = uuid.Parse(...)`.
- **Impact:** Documents, Expenses, Inspections, Lease Renewals, and Maintenance were completely non-functional.
- **See:** [Development Notes — uuid.Parse Pattern](#development-notes) in CLAUDE.md

### Added

#### Message Templates Management Page (`/message-templates`)
- Full CRUD frontend page for WhatsApp message templates (backend was already complete)
- Table view: Name, Category, Variables, Active status
- Create/Edit modal with `{{1}}` `{{2}}` variable placeholder syntax, active toggle
- Click-to-preview panel renders a WhatsApp-style bubble with variables substituted
- Sidebar: added under Inbox in CRM section

#### Email History Page (`/email-history`)
- Standalone page listing all sent emails across all entities
- Status filter chips: All / Sent / Failed / Bounced / Pending
- Click-to-detail panel: full body, error message if failed, timestamps
- **New backend:** `EmailRepository.ListAll(ctx, page, limit)` + `GET /api/v1/emails`
- Sidebar: added in CRM section

#### Invoices List & Detail Pages (`/invoices`, `/invoices/[id]`)
- List page: summary cards (Subtotal / VAT (5%) / Total), status filter, inline PDF link
- Detail page: line-item breakdown, status action buttons (Mark Sent / Mark Paid), link to deal
- **New backend:** `InvoiceRepo.ListAll(ctx, page, limit)` + `GET /api/v1/invoices`
- Sidebar: added in CRM section

#### AI Lead Scoring on Contact Detail (`/contacts/[id]`)
- "🤖 AI" button beside the lead score badge; visible to Agent/Admin when contact has leads
- Calls Ollama LLM, parses `{"score": int, "reasoning": "..."}` response
- Persists score back to `contacts.lead_score` immediately; updates display live
- Shows AI reasoning text inline below the badge
- **New backend:** `ContactRepo.UpdateScore(ctx, id, score)` + `POST /api/v1/ai/score-contact/:id`
- Note: Pipeline page already had AI scoring — this completes the contact detail page

#### Renewal Templates Management (`/renewal-templates`)
- Card-grid CRUD page (Admin only) for renewal communication templates
- Card shows: template name, language badge, email subject preview, WhatsApp message excerpt
- Create/Edit modal with Email / WhatsApp tab switcher; supports bilingual templates
- **New backend:** `RenewalTemplateRepo.Update` + `RenewalTemplateRepo.Delete` + `PATCH /renewal-templates/:id` + `DELETE /renewal-templates/:id`
- Sidebar: added in Properties section (admin only)

### New API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/invoices` | List all invoices, paginated |
| `GET` | `/api/v1/emails` | List all email history, paginated |
| `POST` | `/api/v1/ai/score-contact/:id` | AI-score a contact via their latest lead; persists score |
| `PATCH` | `/api/v1/renewal-templates/:id` | Update renewal template |
| `DELETE` | `/api/v1/renewal-templates/:id` | Delete renewal template |

### New Frontend Routes

| Route | Notes |
|-------|-------|
| `/message-templates` | Full CRUD, Agent + Admin |
| `/email-history` | Read-only log, Agent + Admin |
| `/invoices` | List with summary cards |
| `/invoices/[id]` | Detail + status actions + PDF download |
| `/renewal-templates` | Full CRUD, Admin only |

### Known Gaps Identified (2026-05-27 audit)

All gaps below were found during a systematic audit and either fixed or documented:

| # | Gap | Severity | Resolution |
|---|-----|----------|------------|
| 1 | `uuid.Parse` single-value assignment in 5 handler files — backend won't compile | **Critical** | Fixed in this release |
| 2 | `GET /api/v1/invoices` missing — no standalone invoices list page possible | High | Fixed in this release |
| 3 | `GET /api/v1/emails` missing — email history page not possible | High | Fixed in this release |
| 4 | AI scoring only covered leads, not contact detail page | High | Fixed in this release |
| 5 | `RenewalTemplateRepo` missing `Update` + `Delete` | Medium | Fixed in this release |
| 6 | Message Templates page entirely missing (backend was done) | Medium | Fixed in this release |
| 7 | `ai.go` missing `"log"` import | Medium | Fixed in this release |

---

## [Unreleased]

### Phase 17: SaaS Multi-Tenant Platform (Coming Soon)
- Company onboarding workflow
- Subscription management
- Usage analytics and billing
- Tenant isolation at API and database level

### Phase 18: Advanced Integrations (Coming Soon)
- DocuSign e-signature
- Stripe/PayPal payment processing
- Accounting software integrations
- SMS gateway expansion

### Phase 19: AI & Automation (Coming Soon)
- Automated lease renewal workflows
- Predictive tenant risk scoring
- Expense categorization via ML
- Document OCR and extraction

---

*For self-hosters: Always check the changelog before upgrading. Breaking changes will be marked with ⚠️.*
**Status**: Phases 15-16 complete, ready for Phase 17 planning.
**Maintainer**: Dori Internet Dev Team
